package ninja

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const ninjaAPIContextKey = "gin_ninja_api"

// Config holds configuration options for a NinjaAPI instance.
type Config struct {
	// Title is the API title shown in the OpenAPI docs (default: "Gin Ninja API").
	Title string
	// Version is the API version string (default: "1.0.0").
	Version string
	// Description is an optional long description of the API.
	Description string
	// DocsURL is the path at which the Swagger UI is served (default: "/docs").
	// Set to an empty string to disable the UI.
	DocsURL string
	// OpenAPIURL is the path at which the raw OpenAPI JSON is served (default: "/openapi.json").
	OpenAPIURL string
	// Prefix is a global path prefix prepended to every route (default: "").
	Prefix string
	// Versions configures named API version namespaces and version-scoped docs.
	Versions map[string]VersionConfig
	// SecuritySchemes defines reusable OpenAPI security schemes, such as JWT
	// bearer authentication shown by Swagger UI's "Authorize" button.
	SecuritySchemes map[string]SecurityScheme
	// DisableGinDefault disables the default gin Logger and Recovery middleware
	// when set to true.  Set this to true when you provide your own middleware
	// via UseGin (e.g. the structured logger from middleware.Logger).
	DisableGinDefault bool
	// ReadTimeout limits the time allowed to read the full request, including
	// the body. Zero uses the default safe timeout.
	ReadTimeout time.Duration
	// WriteTimeout limits the time allowed to write a response. Zero uses the
	// default safe timeout.
	WriteTimeout time.Duration
	// IdleTimeout limits how long keep-alive connections stay idle. Zero uses
	// the default safe timeout.
	IdleTimeout time.Duration
	// GracefulShutdownTimeout bounds how long Run waits for shutdown hooks and
	// in-flight requests after receiving SIGINT or SIGTERM. Zero uses 10s.
	GracefulShutdownTimeout time.Duration
}

// NinjaAPI is the central API instance.  It wraps a *gin.Engine and
// manages routers, middleware, and OpenAPI documentation generation in a
// style inspired by django-ninja.
//
//	api := ninja.New(ninja.Config{Title: "My API", Version: "1.0.0"})
//	api.AddRouter(usersRouter)
//	api.Run(":8080")
type NinjaAPI struct {
	engine         *gin.Engine
	config         Config
	openAPI        *openAPISpec
	openAPICache   openAPICacheState
	versionSpecsMu sync.RWMutex
	versionSpecs   map[string]*openAPISpec
	routers        []*Router
	errorMappersMu sync.RWMutex
	errorMappers   []ErrorMapper
	hooksMu        sync.RWMutex
	startupHooks   []LifecycleHook
	shutdownHooks  []LifecycleHook
	lifecycle      lifecycleState
	serverState    serverState
}

// New creates a new NinjaAPI with the supplied configuration.
// Sensible defaults are applied for any empty fields.
func New(config Config) *NinjaAPI {
	if config.Title == "" {
		config.Title = "Gin Ninja API"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.DocsURL == "" {
		config.DocsURL = "/docs"
	}
	if config.OpenAPIURL == "" {
		config.OpenAPIURL = "/openapi.json"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 60 * time.Second
	}
	if config.GracefulShutdownTimeout == 0 {
		config.GracefulShutdownTimeout = 10 * time.Second
	}

	var engine *gin.Engine
	if config.DisableGinDefault {
		engine = gin.New()
	} else {
		engine = gin.Default()
	}

	api := &NinjaAPI{
		engine:       engine,
		config:       config,
		openAPI:      newOpenAPISpec(config),
		openAPICache: openAPICacheState{versions: map[string][]byte{}},
		versionSpecs: map[string]*openAPISpec{},
	}

	api.engine.Use(api.attachContext())
	api.setupInternalRoutes()
	return api
}

// Engine returns the underlying *gin.Engine so callers can add custom
// middleware or register routes outside of the ninja router system.
func (api *NinjaAPI) Engine() *gin.Engine {
	return api.engine
}

// Handler returns the API as an http.Handler, useful for testing with
// httptest.NewServer or embedding in existing servers.
func (api *NinjaAPI) Handler() http.Handler {
	return api.engine
}

// UseGin registers one or more raw gin.HandlerFunc middleware on the
// underlying engine.  This is the preferred way to attach infrastructure
// middleware such as CORS, request-ID injection, structured logging, and
// JWT authentication.
//
//	api.UseGin(middleware.RequestID())
//	api.UseGin(middleware.CORS(nil))
//	api.UseGin(middleware.Logger(log))
//	api.UseGin(middleware.JWTAuth())
func (api *NinjaAPI) UseGin(mw ...gin.HandlerFunc) {
	api.engine.Use(mw...)
}

// AddRouter mounts a Router under the API.
// All operations defined on the router (and any nested sub-routers) are
// registered with the gin engine and included in the OpenAPI spec.
func (api *NinjaAPI) AddRouter(router *Router) {
	api.routers = append(api.routers, router)
	api.registerRouter(api.engine.Group(api.config.Prefix), api.config.Prefix, "", nil, router)
	api.invalidateOpenAPICache()
}

// Run starts the HTTP server on the given address (e.g. ":8080").
func (api *NinjaAPI) Run(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	startupCtx, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()

	errCh := make(chan error, 1)
	go func() {
		errCh <- api.serve(listener, startupCtx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		cancelStartup()
		shutdownCtx, cancel := api.shutdownContext(context.Background())
		defer cancel()
		shutdownErr := api.Shutdown(shutdownCtx)
		serveErr := <-errCh
		if errors.Is(serveErr, http.ErrServerClosed) || errors.Is(serveErr, context.Canceled) {
			serveErr = nil
		}
		return errors.Join(serveErr, shutdownErr)
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// registerRouter recursively registers all routes from a Router tree.
func (api *NinjaAPI) registerRouter(parent *gin.RouterGroup, parentPrefix, inheritedVersion string, inheritedInfo *VersionConfig, router *Router) {
	currentVersion := inheritedVersion
	currentInfo := cloneVersionInfo(inheritedInfo)
	groupPrefix := router.prefix
	startedVersionScope := false
	if router.version != "" && router.version != inheritedVersion {
		currentVersion = router.version
		cfg := api.lookupVersion(router.version)
		currentInfo = &cfg
		groupPrefix = cfg.Prefix + groupPrefix
		startedVersionScope = true
	}

	prefix := parentPrefix + groupPrefix
	group := parent.Group(groupPrefix)

	if startedVersionScope && currentInfo != nil {
		group.Use(versionDeprecationMiddleware(*currentInfo))
	}

	// Attach raw gin middleware first.
	if len(router.ginMiddleware) > 0 {
		group.Use(router.ginMiddleware...)
	}

	// Convert typed ninja middleware into gin.HandlerFunc.
	for _, mw := range router.middleware {
		mw := mw // capture loop var
		group.Use(func(c *gin.Context) {
			ctx := newContext(c)
			if err := mw(ctx); err != nil {
				writeError(c, err)
				c.Abort()
				return
			}
			c.Next()
		})
	}

	for _, op := range router.operations {
		// Build a copy of the operation with the correct full path for the spec.
		opForSpec := *op
		opForSpec.path = prefix + op.path
		opForSpec.version = currentVersion
		opForSpec.versionInfo = cloneVersionInfo(currentInfo)
		if currentInfo != nil && currentInfo.Deprecated {
			opForSpec.deprecated = true
		}
		api.openAPI.addOperation(&opForSpec)
		if currentVersion != "" {
			api.versionSpec(currentVersion).addOperation(&opForSpec)
		}

		group.Handle(op.method, op.path, op.ginHandler)
		if op.method == http.MethodGet {
			group.Handle(http.MethodHead, op.path, op.ginHandler)
		}
	}

	for _, sub := range router.subrouters {
		api.registerRouter(group, prefix, currentVersion, currentInfo, sub)
	}
}

// setupInternalRoutes adds the OpenAPI JSON and Swagger UI routes.
func (api *NinjaAPI) setupInternalRoutes() {
	if api.config.OpenAPIURL != "" {
		api.engine.GET(api.config.OpenAPIURL, func(c *gin.Context) {
			data, err := api.openAPIBytes()
			if err != nil {
				writeError(c, err)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", data)
		})
	}

	if api.config.DocsURL != "" {
		docsURL := api.config.DocsURL
		openAPIURL := api.config.OpenAPIURL
		title := api.config.Title
		api.engine.GET(docsURL, func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8",
				[]byte(swaggerUIHTML(openAPIURL, title)))
		})
	}

	if pattern := versionedOpenAPIPattern(api.config.OpenAPIURL); pattern != "" {
		api.engine.GET(pattern, func(c *gin.Context) {
			version := requestVersion(c)
			data, ok, err := api.versionOpenAPIBytes(version)
			if err != nil {
				writeError(c, err)
				return
			}
			if !ok {
				versionNotFound(c)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", data)
		})
	}

	if pattern := versionedDocsPattern(api.config.DocsURL); pattern != "" {
		baseOpenAPIURL := api.config.OpenAPIURL
		title := api.config.Title
		api.engine.GET(pattern, func(c *gin.Context) {
			version := requestVersion(c)
			if _, ok := api.lookupVersionSpec(version); !ok {
				versionNotFound(c)
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8",
				[]byte(swaggerUIHTML(versionedOpenAPIPath(baseOpenAPIURL, version), title+" ("+version+")")))
		})
	}
}

func (api *NinjaAPI) RegisterErrorMapper(mapper ErrorMapper) {
	if mapper == nil {
		return
	}
	api.errorMappersMu.Lock()
	api.errorMappers = append(api.errorMappers, mapper)
	api.errorMappersMu.Unlock()
}

func (api *NinjaAPI) mapError(err error) error {
	if err == nil {
		return nil
	}
	mappers := errorMappersSnapshot()
	api.errorMappersMu.RLock()
	mappers = append(mappers, api.errorMappers...)
	api.errorMappersMu.RUnlock()
	return mapErrorWithMappers(err, mappers)
}

func (api *NinjaAPI) attachContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ninjaAPIContextKey, api)
		c.Next()
	}
}

func currentAPI(c *gin.Context) (*NinjaAPI, bool) {
	if c == nil {
		return nil, false
	}
	v, ok := c.Get(ninjaAPIContextKey)
	if !ok {
		return nil, false
	}
	api, ok := v.(*NinjaAPI)
	return api, ok
}

func (api *NinjaAPI) lookupVersion(name string) VersionConfig {
	if cfg, ok := api.config.Versions[name]; ok {
		return normalizeVersionConfig(name, cfg)
	}
	return normalizeVersionConfig(name, VersionConfig{})
}

func (api *NinjaAPI) versionSpec(name string) *openAPISpec {
	api.versionSpecsMu.RLock()
	if spec, ok := api.versionSpecs[name]; ok {
		api.versionSpecsMu.RUnlock()
		return spec
	}
	api.versionSpecsMu.RUnlock()

	api.versionSpecsMu.Lock()
	defer api.versionSpecsMu.Unlock()
	if spec, ok := api.versionSpecs[name]; ok {
		return spec
	}
	cfg := versionSpecConfig(api.config, name, api.lookupVersion(name))
	spec := newOpenAPISpec(cfg)
	api.versionSpecs[name] = spec
	return spec
}

func (api *NinjaAPI) lookupVersionSpec(name string) (*openAPISpec, bool) {
	api.versionSpecsMu.RLock()
	defer api.versionSpecsMu.RUnlock()
	spec, ok := api.versionSpecs[name]
	return spec, ok
}
