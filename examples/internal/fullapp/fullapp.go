package fullapp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"github.com/shijl0925/gin-ninja/bootstrap"
	_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
	"github.com/shijl0925/gin-ninja/examples/full/app"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Options struct {
	Description           string
	Versions              map[string]ninja.VersionConfig
	IncludeAuth           bool
	IncludeUsersV1        bool
	IncludeUsersV2        bool
	IncludeAdminAPI       bool
	IncludeAdminPages     bool
	IncludeFeatureDemos   bool
	IncludeVersionedDemos bool
	IncludeHealth         bool
}

func FullOptions() Options {
	return Options{
		Description: "A full-featured gin-ninja example with bootstrap, middleware, settings, caching, streaming, and API versioning demos.",
		Versions: map[string]ninja.VersionConfig{
			"v2": {
				Prefix:      "/v2",
				Description: "Cached users CRUD example with explicit invalidation.",
			},
			"v1": {
				Prefix:      "/v1",
				Description: "Current example API version.",
			},
			"v0": {
				Prefix:       "/v0",
				Description:  "Legacy example API demonstrating deprecation headers.",
				Deprecated:   true,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/docs/gin-ninja/v1-migration",
			},
		},
		IncludeAuth:           true,
		IncludeUsersV1:        true,
		IncludeUsersV2:        true,
		IncludeAdminAPI:       true,
		IncludeAdminPages:     true,
		IncludeFeatureDemos:   true,
		IncludeVersionedDemos: true,
		IncludeHealth:         true,
	}
}

func UsersOptions() Options {
	return Options{
		Description: "A focused gin-ninja example covering JWT auth plus versioned users CRUD endpoints.",
		Versions: map[string]ninja.VersionConfig{
			"v2": {
				Prefix:      "/v2",
				Description: "Cached users CRUD example with explicit invalidation.",
			},
			"v1": {
				Prefix:      "/v1",
				Description: "Current example API version.",
			},
		},
		IncludeAuth:    true,
		IncludeUsersV1: true,
		IncludeUsersV2: true,
		IncludeHealth:  true,
	}
}

func FeaturesOptions() Options {
	return Options{
		Description: "A focused gin-ninja example covering caching, versioned routing, SSE, WebSocket, upload, and download demos.",
		Versions: map[string]ninja.VersionConfig{
			"v1": {
				Prefix:      "/v1",
				Description: "Current example API version.",
			},
			"v0": {
				Prefix:       "/v0",
				Description:  "Legacy example API demonstrating deprecation headers.",
				Deprecated:   true,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/docs/gin-ninja/v1-migration",
			},
		},
		IncludeFeatureDemos:   true,
		IncludeVersionedDemos: true,
		IncludeHealth:         true,
	}
}

func AdminOptions() Options {
	return Options{
		Description: "A focused gin-ninja example covering the metadata-driven admin resource APIs and standalone admin pages.",
		Versions: map[string]ninja.VersionConfig{
			"v1": {
				Prefix:      "/v1",
				Description: "Current example API version.",
			},
		},
		IncludeAuth:       true,
		IncludeAdminAPI:   true,
		IncludeAdminPages: true,
		IncludeHealth:     true,
	}
}

func MustLoadConfig(configPath string) *settings.Config {
	cfg := settings.MustLoad(configPath)
	normalizeConfigPaths(configPath, cfg)
	return cfg
}

func normalizeConfigPaths(configPath string, cfg *settings.Config) {
	if cfg == nil || cfg.Database.Driver != "sqlite" || cfg.Database.DSN == "" {
		return
	}
	if strings.HasPrefix(cfg.Database.DSN, "file:") || filepath.IsAbs(cfg.Database.DSN) {
		return
	}
	cfg.Database.DSN = filepath.Join(filepath.Dir(configPath), cfg.Database.DSN)
}

func InitDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
	db, err := bootstrap.InitDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}
	if err := db.AutoMigrate(&app.User{}, &app.Role{}, &app.Project{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	orm.Init(db)
	return db, nil
}

func initCacheStore(cfg settings.Config) (ninja.ResponseCacheStore, func(context.Context) error) {
	cacheStore := ninja.ResponseCacheStore(ninja.NewMemoryCacheStore())
	var cacheStoreShutdown func(context.Context) error
	if cfg.Redis.Enabled {
		redisStore, err := ninja.NewRedisCacheStore(ninja.RedisCacheConfig{
			Addr:     cfg.Redis.Addr,
			Username: cfg.Redis.Username,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			Prefix:   cfg.Redis.Prefix,
		})
		if err != nil {
			log.Printf("cache: falling back to in-memory store: %v", err)
		} else if err := redisStore.Ping(context.Background()); err != nil {
			log.Printf("cache: redis unavailable, falling back to in-memory store: %v", err)
			_ = redisStore.Close()
		} else {
			cacheStore = redisStore
			cacheStoreShutdown = func(context.Context) error { return redisStore.Close() }
			log.Printf("cache: using redis store at %s", cfg.Redis.Addr)
		}
	}
	return cacheStore, cacheStoreShutdown
}

func BuildAPI(cfg settings.Config, db *gorm.DB, log_ *zap.Logger, opts Options) *ninja.NinjaAPI {
	// Ensure the global settings reflect the provided config so that helpers
	// such as middleware.JWTAuth() and middleware.GenerateToken() use the
	// correct values when called from route setup functions.
	settings.SetGlobal(cfg)

	api := ninja.New(ninja.Config{
		Title:       cfg.App.Name,
		Version:     cfg.App.Version,
		Description: opts.Description,
		Prefix:      "/api",
		Versions:    cloneVersions(opts.Versions),
		SecuritySchemes: map[string]ninja.SecurityScheme{
			"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
		},
		DisableGinDefault: true,
	})
	api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	})

	var cacheStore ninja.ResponseCacheStore
	if opts.IncludeUsersV2 || opts.IncludeFeatureDemos {
		cacheStoreShutdown := func(context.Context) error { return nil }
		cacheStore, cacheStoreShutdown = initCacheStore(cfg)
		api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
			return cacheStoreShutdown(ctx)
		})
	}

	api.UseGin(
		middleware.RequestID(),
		middleware.Recovery(log_),
		middleware.Logger(log_),
		middleware.CORS(nil),
		orm.Middleware(db),
	)

	if opts.IncludeAuth {
		addAuthRoutes(api)
	}
	if opts.IncludeUsersV1 {
		addUsersV1Routes(api)
	}
	if opts.IncludeUsersV2 {
		app.ConfigureUsersV2Cache(cacheStore)
		addUsersV2Routes(api, cacheStore)
	}
	if opts.IncludeAdminAPI {
		addAdminRoutes(api)
	}
	if opts.IncludeFeatureDemos {
		addFeatureRoutes(api, cacheStore)
	}
	if opts.IncludeVersionedDemos {
		addVersionedRoutes(api)
	}
	if opts.IncludeAdminPages {
		addAdminPages(api)
	}
	if opts.IncludeHealth {
		addHealthRoute(api)
	}

	return api
}

func Run(cfg settings.Config, log_ *zap.Logger, opts Options) error {
	db, err := InitDB(&cfg.Database)
	if err != nil {
		return err
	}

	api := BuildAPI(cfg, db, log_, opts)
	addr := cfg.Server.Addr()
	log.Printf("Starting %s v%s on http://%s", cfg.App.Name, cfg.App.Version, addr)
	log.Printf("Swagger UI: http://%s/docs", addr)
	return api.Run(addr)
}

func cloneVersions(in map[string]ninja.VersionConfig) map[string]ninja.VersionConfig {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ninja.VersionConfig, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func addAuthRoutes(api *ninja.NinjaAPI) {
	authRouter := ninja.NewRouter(
		"/auth",
		ninja.WithTags("Auth"),
		ninja.WithTagDescription("Auth", "Authentication endpoints for login and registration"),
		ninja.WithVersion("v1"),
	)
	ninja.Post(authRouter, "/register", app.Register, ninja.Summary("Register a new user"), ninja.WithTransaction())
	ninja.Post(authRouter, "/login", app.Login, ninja.Summary("Login and get JWT token"))
	api.AddRouter(authRouter)
}

func addUsersV1Routes(api *ninja.NinjaAPI) {
	usersRouter := ninja.NewRouter(
		"/users",
		ninja.WithTags("Users"),
		ninja.WithTagDescription("Users", "JWT-protected user CRUD endpoints"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v1"),
	)
	usersRouter.UseGin(middleware.JWTAuth())

	ninja.Get(usersRouter, "/", app.ListUsers,
		ninja.Summary("List users"),
		ninja.Description("Returns a paginated list of users"),
		ninja.Paginated[app.UserOut](),
		ninja.Timeout(2*time.Second),
		ninja.RateLimit(20, 40),
	)
	ninja.Get(usersRouter, "/:id", app.GetUser,
		ninja.Summary("Get user"))
	ninja.Post(usersRouter, "/", app.CreateUser,
		ninja.Summary("Create user"),
		ninja.WithTransaction())
	ninja.Put(usersRouter, "/:id", app.UpdateUser,
		ninja.Summary("Update user"),
		ninja.WithTransaction())
	ninja.Delete(usersRouter, "/:id", app.DeleteUser,
		ninja.Summary("Delete user"),
		ninja.WithTransaction())

	api.AddRouter(usersRouter)
}

func addUsersV2Routes(api *ninja.NinjaAPI, cacheStore ninja.ResponseCacheStore) {
	usersV2Router := ninja.NewRouter(
		"/users",
		ninja.WithTags("Users"),
		ninja.WithTagDescription("Users", "JWT-protected user CRUD endpoints"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v2"),
	)
	usersV2Router.UseGin(middleware.JWTAuth())

	ninja.Get(usersV2Router, "/", app.ListUsersV2,
		ninja.Summary("List users (cached CRUD demo)"),
		ninja.Description("Demonstrates cached list responses plus tag-based invalidation after create, update, and delete operations."),
		ninja.Paginated[app.UserOut](),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithTags(app.UsersV2ListCacheTags),
		),
	)
	ninja.Get(usersV2Router, "/:id", app.GetUserV2,
		ninja.Summary("Get user (cached CRUD demo)"),
		ninja.Description("Demonstrates detail response caching with a stable cache key and explicit invalidation after update or delete."),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithKey(app.UsersV2DetailCacheKey),
			ninja.CacheWithTags(app.UsersV2DetailCacheTags),
		),
	)
	ninja.Post(usersV2Router, "/", app.CreateUserV2,
		ninja.Summary("Create user (invalidates cached lists)"),
		ninja.Description("Creates a user and invalidates cached list queries so subsequent reads observe the new record."),
		ninja.WithTransaction(),
	)
	ninja.Put(usersV2Router, "/:id", app.UpdateUserV2,
		ninja.Summary("Update user (invalidates cached detail + lists)"),
		ninja.Description("Updates a user, deletes the cached detail entry, and invalidates cached list queries."),
		ninja.WithTransaction(),
	)
	ninja.Delete(usersV2Router, "/:id", app.DeleteUserV2,
		ninja.Summary("Delete user (invalidates cached detail + lists)"),
		ninja.Description("Deletes a user and invalidates cached detail and list responses."),
		ninja.WithTransaction(),
	)
	api.AddRouter(usersV2Router)
}

func addAdminRoutes(api *ninja.NinjaAPI) {
	adminRouter := ninja.NewRouter(
		"/admin",
		ninja.WithTags("Admin"),
		ninja.WithTagDescription("Admin", "JWT-protected metadata-driven admin resource APIs"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v1"),
	)
	adminRouter.UseGin(middleware.JWTAuth())
	app.NewAdminSite().Mount(adminRouter)
	api.AddRouter(adminRouter)
}

func addFeatureRoutes(api *ninja.NinjaAPI, cacheStore ninja.ResponseCacheStore) {
	exampleRouter := ninja.NewRouter(
		"/examples",
		ninja.WithTags("Examples"),
		ninja.WithTagDescription("Examples", "Framework feature demos for manual testing"),
		ninja.WithVersion("v1"),
	)
	ninja.Get(exampleRouter, "/request-meta", app.EchoRequestMeta,
		ninja.Summary("Echo request metadata"),
		ninja.Description("Demonstrates cookie parameter binding, query/header/cookie defaults, and extra response declarations."),
		ninja.Response(http.StatusUnauthorized, "Example unauthorized response", nil),
		ninja.Response(http.StatusNotFound, "Example detailed response", &app.RequestMetaOutput{}),
	)
	ninja.Get(exampleRouter, "/features", app.ListFeatureDemos,
		ninja.Summary("List framework feature demos"),
		ninja.Description("Demonstrates standardized paginated response declarations."),
		ninja.Paginated[app.FeatureItemOut](),
	)
	ninja.Get(exampleRouter, "/cache", app.CachedFeatureDemo,
		ninja.Summary("Cache + ETag endpoint"),
		ninja.Description("Demonstrates route-level response caching with pluggable memory/Redis stores, Cache-Control, and conditional requests with ETag."),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithTags(func(ctx *ninja.Context) []string { return []string{"examples", "examples:cache"} }),
		),
	)
	ninja.Get(exampleRouter, "/limited", app.LimitedOperation,
		ninja.Summary("Rate-limited endpoint"),
		ninja.Description("Call this endpoint repeatedly to trigger the per-operation rate limiter."),
		ninja.RateLimit(1, 1),
	)
	ninja.Get(exampleRouter, "/slow", app.SlowOperation,
		ninja.Summary("Timeout endpoint"),
		ninja.Description("This endpoint intentionally exceeds its timeout so you can observe a 408 response."),
		ninja.Timeout(150*time.Millisecond),
	)
	ninja.Get(exampleRouter, "/hidden", app.HiddenOperation,
		ninja.Summary("Hidden example endpoint"),
		ninja.Description("This route is reachable but excluded from OpenAPI."),
		ninja.ExcludeFromDocs(),
	)
	ninja.SSE(exampleRouter, "/events", app.StreamEventsDemo,
		ninja.Summary("SSE endpoint"),
		ninja.Description("Demonstrates server-sent events with typed input binding."),
	)
	ninja.WebSocket(exampleRouter, "/ws", app.WebSocketEchoDemo,
		ninja.Summary("WebSocket endpoint"),
		ninja.Description("Demonstrates WebSocket upgrades and bidirectional messaging."),
	)
	ninja.Post(exampleRouter, "/upload-single", app.UploadSingleDemo,
		ninja.Summary("Single file upload"),
		ninja.Description("Demonstrates multipart form-data binding with one file and extra form fields."),
	)
	ninja.Post(exampleRouter, "/upload-many", app.UploadManyDemo,
		ninja.Summary("Multiple file upload"),
		ninja.Description("Demonstrates multipart form-data binding with multiple files and extra form fields."),
	)
	ninja.Get(exampleRouter, "/download", app.DownloadDemo,
		ninja.Summary("Binary download"),
		ninja.Description("Demonstrates download responses without JSON serialization."),
	)
	ninja.Get(exampleRouter, "/download-reader", app.DownloadReaderDemo,
		ninja.Summary("Reader-backed download"),
		ninja.Description("Demonstrates streaming-style download responses backed by an io.Reader."),
	)
	api.AddRouter(exampleRouter)
}

func addVersionedRoutes(api *ninja.NinjaAPI) {
	versionedV1Router := ninja.NewRouter(
		"/examples/versioned",
		ninja.WithTags("Examples"),
		ninja.WithVersion("v1"),
	)
	ninja.Get(versionedV1Router, "/info", app.VersionedInfoV1,
		ninja.Summary("Versioned info (v1)"),
		ninja.Description("Demonstrates version-scoped routing for the current API version."),
	)
	api.AddRouter(versionedV1Router)

	versionedV0Router := ninja.NewRouter(
		"/examples/versioned",
		ninja.WithTags("Examples"),
		ninja.WithVersion("v0"),
	)
	ninja.Get(versionedV0Router, "/info", app.VersionedInfoV0,
		ninja.Summary("Versioned info (v0)"),
		ninja.Description("Demonstrates version-scoped routing and deprecation headers on a legacy version."),
	)
	api.AddRouter(versionedV0Router)
}

func addAdminPages(api *ninja.NinjaAPI) {
	admin.MountUI(api.Engine(), admin.DefaultUIConfig())
}

func addHealthRoute(api *ninja.NinjaAPI) {
	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})
}
