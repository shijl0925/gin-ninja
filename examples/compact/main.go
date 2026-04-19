// Package main runs a compact counterpart to the full gin-ninja example.
//
// Run:
//
//	go run ./examples/compact
//
// Then visit:
//   - http://localhost:8080/docs
//   - http://localhost:8080/docs/v2
//   - http://localhost:8080/openapi.json
//   - http://localhost:8080/admin
package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/examples/full/app"
	"github.com/shijl0925/gin-ninja/examples/internal/fullapp"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var runCompactMain = run
var fatalCompact = func(v ...any) { log.Fatal(v...) }

func initDB(cfg *settings.DatabaseConfig) (*gorm.DB, error) {
	return fullapp.InitDB(cfg)
}

func initCacheStore(cfg settings.Config) (ninja.ResponseCacheStore, func(context.Context) error) {
	cacheStore := ninja.ResponseCacheStore(ninja.NewMemoryCacheStore())
	cacheStoreShutdown := func(context.Context) error { return nil }
	if !cfg.Redis.Enabled {
		return cacheStore, cacheStoreShutdown
	}

	redisStore, err := ninja.NewRedisCacheStore(ninja.RedisCacheConfig{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Prefix:   cfg.Redis.Prefix,
	})
	if err != nil {
		log.Printf("cache: falling back to in-memory store: %v", err)
		return cacheStore, cacheStoreShutdown
	}
	if err := redisStore.Ping(context.Background()); err != nil {
		log.Printf("cache: redis unavailable, falling back to in-memory store: %v", err)
		_ = redisStore.Close()
		return cacheStore, cacheStoreShutdown
	}
	log.Printf("cache: using redis store at %s", cfg.Redis.Addr)
	return redisStore, func(context.Context) error { return redisStore.Close() }
}

func buildAPI(cfg settings.Config, db *gorm.DB, log_ *zap.Logger) *ninja.NinjaAPI {
	settings.SetGlobal(cfg)
	opts := fullapp.FullOptions()

	api := ninja.New(ninja.Config{
		Title:       cfg.App.Name,
		Version:     cfg.App.Version,
		Description: opts.Description,
		Prefix:      "/api",
		Versions:    opts.Versions,
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

	cacheStore, cacheShutdown := initCacheStore(cfg)
	api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
		return cacheShutdown(ctx)
	})

	api.UseGin(
		middleware.RequestID(),
		middleware.Recovery(log_),
		middleware.Logger(log_),
		middleware.CORS(nil),
		orm.Middleware(db),
	)

	registerAuthRoutes(api)
	registerUsersV1Routes(api)
	registerUsersV2Routes(api, cacheStore)
	registerAdminRoutes(api)
	registerFeatureRoutes(api, cacheStore)
	registerVersionedRoutes(api)
	admin.MountUI(api.Engine(), admin.DefaultUIConfig())
	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	return api
}

func registerAuthRoutes(api *ninja.NinjaAPI) {
	router := ninja.NewRouter(
		"/auth",
		ninja.WithTags("Auth"),
		ninja.WithTagDescription("Auth", "Authentication endpoints for login and registration"),
		ninja.WithVersion("v1"),
	)
	ninja.Post(router, "/register", app.Register, ninja.Summary("Register a new user"), ninja.WithTransaction())
	ninja.Post(router, "/login", app.Login, ninja.Summary("Login and get JWT token"))
	api.AddRouter(router)
}

func registerUsersV1Routes(api *ninja.NinjaAPI) {
	router := ninja.NewRouter(
		"/users",
		ninja.WithTags("Users"),
		ninja.WithTagDescription("Users", "JWT-protected user CRUD endpoints"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v1"),
	)
	router.UseGin(middleware.JWTAuth())

	ninja.Get(router, "/", app.ListUsers,
		ninja.Summary("List users"),
		ninja.Description("Returns a paginated list of users"),
		ninja.Paginated[app.UserOut](),
		ninja.Timeout(2*time.Second),
		ninja.RateLimit(20, 40),
	)
	ninja.Get(router, "/:id", app.GetUser, ninja.Summary("Get user"))
	ninja.Post(router, "/", app.CreateUser, ninja.Summary("Create user"), ninja.WithTransaction())
	ninja.Put(router, "/:id", app.UpdateUser, ninja.Summary("Update user"), ninja.WithTransaction())
	ninja.Delete(router, "/:id", app.DeleteUser, ninja.Summary("Delete user"), ninja.WithTransaction())
	api.AddRouter(router)
}

func registerUsersV2Routes(api *ninja.NinjaAPI, cacheStore ninja.ResponseCacheStore) {
	app.ConfigureUsersV2Cache(cacheStore)

	router := ninja.NewRouter(
		"/users",
		ninja.WithTags("Users"),
		ninja.WithTagDescription("Users", "JWT-protected user CRUD endpoints"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v2"),
	)
	router.UseGin(middleware.JWTAuth())

	ninja.Get(router, "/", app.ListUsersV2,
		ninja.Summary("List users (cached CRUD demo)"),
		ninja.Description("Demonstrates cached list responses plus tag-based invalidation after create, update, and delete operations."),
		ninja.Paginated[app.UserOut](),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithTags(app.UsersV2ListCacheTags),
		),
	)
	ninja.Get(router, "/:id", app.GetUserV2,
		ninja.Summary("Get user (cached CRUD demo)"),
		ninja.Description("Demonstrates detail response caching with a stable cache key and explicit invalidation after update or delete."),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithKey(app.UsersV2DetailCacheKey),
			ninja.CacheWithTags(app.UsersV2DetailCacheTags),
		),
	)
	ninja.Post(router, "/", app.CreateUserV2,
		ninja.Summary("Create user (invalidates cached lists)"),
		ninja.Description("Creates a user and invalidates cached list queries so subsequent reads observe the new record."),
		ninja.WithTransaction(),
	)
	ninja.Put(router, "/:id", app.UpdateUserV2,
		ninja.Summary("Update user (invalidates cached detail + lists)"),
		ninja.Description("Updates a user, deletes the cached detail entry, and invalidates cached list queries."),
		ninja.WithTransaction(),
	)
	ninja.Delete(router, "/:id", app.DeleteUserV2,
		ninja.Summary("Delete user (invalidates cached detail + lists)"),
		ninja.Description("Deletes a user and invalidates cached detail and list responses."),
		ninja.WithTransaction(),
	)
	api.AddRouter(router)
}

func registerAdminRoutes(api *ninja.NinjaAPI) {
	router := ninja.NewRouter(
		"/admin",
		ninja.WithTags("Admin"),
		ninja.WithTagDescription("Admin", "JWT-protected metadata-driven admin resource APIs"),
		ninja.WithBearerAuth(),
		ninja.WithVersion("v1"),
	)
	router.UseGin(middleware.JWTAuth())
	app.NewAdminSite().Mount(router)
	api.AddRouter(router)
}

func registerFeatureRoutes(api *ninja.NinjaAPI, cacheStore ninja.ResponseCacheStore) {
	router := ninja.NewRouter(
		"/examples",
		ninja.WithTags("Examples"),
		ninja.WithTagDescription("Examples", "Framework feature demos for manual testing"),
		ninja.WithVersion("v1"),
	)
	ninja.Get(router, "/request-meta", app.EchoRequestMeta,
		ninja.Summary("Echo request metadata"),
		ninja.Description("Demonstrates cookie parameter binding, query/header/cookie defaults, and extra response declarations."),
		ninja.Response(http.StatusUnauthorized, "Example unauthorized response", nil),
		ninja.Response(http.StatusNotFound, "Example detailed response", &app.RequestMetaOutput{}),
	)
	ninja.Get(router, "/features", app.ListFeatureDemos,
		ninja.Summary("List framework feature demos"),
		ninja.Description("Demonstrates standardized paginated response declarations."),
		ninja.Paginated[app.FeatureItemOut](),
	)
	ninja.Get(router, "/cache", app.CachedFeatureDemo,
		ninja.Summary("Cache + ETag endpoint"),
		ninja.Description("Demonstrates route-level response caching with pluggable memory/Redis stores, Cache-Control, and conditional requests with ETag."),
		ninja.Cache(time.Minute,
			ninja.CacheWithStore(cacheStore),
			ninja.CacheWithTags(func(ctx *ninja.Context) []string { return []string{"examples", "examples:cache"} }),
		),
	)
	ninja.Get(router, "/limited", app.LimitedOperation,
		ninja.Summary("Rate-limited endpoint"),
		ninja.Description("Call this endpoint repeatedly to trigger the per-operation rate limiter."),
		ninja.RateLimit(1, 1),
	)
	ninja.Get(router, "/slow", app.SlowOperation,
		ninja.Summary("Timeout endpoint"),
		ninja.Description("This endpoint intentionally exceeds its timeout so you can observe a 408 response."),
		ninja.Timeout(150*time.Millisecond),
	)
	ninja.Get(router, "/hidden", app.HiddenOperation,
		ninja.Summary("Hidden example endpoint"),
		ninja.Description("This route is reachable but excluded from OpenAPI."),
		ninja.ExcludeFromDocs(),
	)
	ninja.SSE(router, "/events", app.StreamEventsDemo,
		ninja.Summary("SSE endpoint"),
		ninja.Description("Demonstrates server-sent events with typed input binding."),
	)
	ninja.WebSocket(router, "/ws", app.WebSocketEchoDemo,
		ninja.Summary("WebSocket endpoint"),
		ninja.Description("Demonstrates WebSocket upgrades and bidirectional messaging."),
	)
	ninja.Post(router, "/upload-single", app.UploadSingleDemo,
		ninja.Summary("Single file upload"),
		ninja.Description("Demonstrates multipart form-data binding with one file and extra form fields."),
	)
	ninja.Post(router, "/upload-many", app.UploadManyDemo,
		ninja.Summary("Multiple file upload"),
		ninja.Description("Demonstrates multipart form-data binding with multiple files and extra form fields."),
	)
	ninja.Get(router, "/download", app.DownloadDemo,
		ninja.Summary("Binary download"),
		ninja.Description("Demonstrates download responses without JSON serialization."),
	)
	ninja.Get(router, "/download-reader", app.DownloadReaderDemo,
		ninja.Summary("Reader-backed download"),
		ninja.Description("Demonstrates streaming-style download responses backed by an io.Reader."),
	)
	api.AddRouter(router)
}

func registerVersionedRoutes(api *ninja.NinjaAPI) {
	v1 := ninja.NewRouter("/examples/versioned", ninja.WithTags("Examples"), ninja.WithVersion("v1"))
	ninja.Get(v1, "/info", app.VersionedInfoV1,
		ninja.Summary("Versioned info (v1)"),
		ninja.Description("Demonstrates version-scoped routing for the current API version."),
	)
	api.AddRouter(v1)

	v0 := ninja.NewRouter("/examples/versioned", ninja.WithTags("Examples"), ninja.WithVersion("v0"))
	ninja.Get(v0, "/info", app.VersionedInfoV0,
		ninja.Summary("Versioned info (v0)"),
		ninja.Description("Demonstrates version-scoped routing and deprecation headers on a legacy version."),
	)
	api.AddRouter(v0)
}

func run(cfg settings.Config, log_ *zap.Logger) error {
	db, err := initDB(&cfg.Database)
	if err != nil {
		return err
	}
	api := buildAPI(cfg, db, log_)
	addr := cfg.Server.Addr()
	log.Printf("Starting compact example %s v%s on http://%s", cfg.App.Name, cfg.App.Version, addr)
	log.Printf("Swagger UI: http://%s/docs", addr)
	return api.Run(addr)
}

func main() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fatalCompact("resolve config path")
	}

	cfg := fullapp.MustLoadConfig(filepath.Join(filepath.Dir(file), "config.yaml"))
	log_ := bootstrap.InitLogger(&cfg.Log)
	defer logger.Sync()

	if err := runCompactMain(*cfg, log_); err != nil {
		fatalCompact(err)
	}
}
