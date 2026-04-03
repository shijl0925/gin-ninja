package ninja

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
	// SecuritySchemes defines reusable OpenAPI security schemes, such as JWT
	// bearer authentication shown by Swagger UI's "Authorize" button.
	SecuritySchemes map[string]SecurityScheme
	// DisableGinDefault disables the default gin Logger and Recovery middleware
	// when set to true.  Set this to true when you provide your own middleware
	// via UseGin (e.g. the structured logger from middleware.Logger).
	DisableGinDefault bool
}

// NinjaAPI is the central API instance.  It wraps a *gin.Engine and
// manages routers, middleware, and OpenAPI documentation generation in a
// style inspired by django-ninja.
//
//	api := ninja.New(ninja.Config{Title: "My API", Version: "1.0.0"})
//	api.AddRouter(usersRouter)
//	api.Run(":8080")
type NinjaAPI struct {
	engine  *gin.Engine
	config  Config
	openAPI *openAPISpec
	routers []*Router
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

	var engine *gin.Engine
	if config.DisableGinDefault {
		engine = gin.New()
	} else {
		engine = gin.Default()
	}

	api := &NinjaAPI{
		engine:  engine,
		config:  config,
		openAPI: newOpenAPISpec(config),
	}

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
	api.registerRouter(api.engine.Group(api.config.Prefix), api.config.Prefix, router)
}

// Run starts the HTTP server on the given address (e.g. ":8080").
func (api *NinjaAPI) Run(addr string) error {
	return api.engine.Run(addr)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// registerRouter recursively registers all routes from a Router tree.
func (api *NinjaAPI) registerRouter(parent *gin.RouterGroup, parentPrefix string, router *Router) {
	prefix := parentPrefix + router.prefix
	group := parent.Group(router.prefix)

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
		api.openAPI.addOperation(&opForSpec)

		group.Handle(op.method, op.path, op.ginHandler)
	}

	for _, sub := range router.subrouters {
		api.registerRouter(group, prefix, sub)
	}
}

// setupInternalRoutes adds the OpenAPI JSON and Swagger UI routes.
func (api *NinjaAPI) setupInternalRoutes() {
	if api.config.OpenAPIURL != "" {
		api.engine.GET(api.config.OpenAPIURL, func(c *gin.Context) {
			c.JSON(http.StatusOK, api.openAPI.build())
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
}
