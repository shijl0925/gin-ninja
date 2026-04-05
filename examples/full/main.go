// Package main is the entry-point for the gin-ninja full example.
//
// Project layout:
//
//	examples/full/
//	├── main.go          ← this file: reads config, bootstraps the app, starts the server
//	├── config.yaml      ← sample configuration file
//	└── app/
//	    ├── models.go    ← GORM domain models
//	    ├── schemas.go   ← request / response structs (API schema)
//	    ├── repos.go     ← gormx repository layer
//	    └── users.go     ← handler functions
//
// Run:
//
//	cd examples/full
//	go run .
//
// Then visit:
//   - http://localhost:8080/docs           – Swagger UI (all routes)
//   - http://localhost:8080/docs/v1        – versioned Swagger UI for v1
//   - http://localhost:8080/docs/v2        – versioned Swagger UI for v2
//   - http://localhost:8080/openapi.json   – raw OpenAPI spec (all routes)
//   - http://localhost:8080/openapi/v1.json – raw OpenAPI spec for v1
//   - http://localhost:8080/openapi/v2.json – raw OpenAPI spec for v2
//   - POST http://localhost:8080/api/v1/auth/register – register a new user
//   - POST http://localhost:8080/api/v1/auth/login    – get a JWT token
//   - GET  http://localhost:8080/api/v1/users         – list users (requires JWT)
//   - GET  http://localhost:8080/api/v1/examples/request-meta   – binding/defaults demo
//   - GET  http://localhost:8080/api/v1/examples/cache          – cache / ETag demo
//   - GET  http://localhost:8080/api/v1/examples/events?name=bot – SSE demo
//   - WS   ws://localhost:8080/api/v1/examples/ws?name=bot      – WebSocket demo
//   - GET  http://localhost:8080/api/v1/examples/versioned/info – deprecated version route demo
//   - GET  http://localhost:8080/api/v2/examples/versioned/info – current version route demo
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"

	"github.com/shijl0925/gin-ninja/examples/full/app"
)

func main() {
	// ── 1. Load configuration ────────────────────────────────────────────────
	cfg := settings.MustLoad("examples/full/config.yaml")

	// ── 2. Initialise logger ─────────────────────────────────────────────────
	log_ := bootstrap.InitLogger(&cfg.Log)
	defer logger.Sync()

	// ── 3. Initialise database ───────────────────────────────────────────────
	db := bootstrap.MustInitDB(&cfg.Database)
	if err := db.AutoMigrate(&app.User{}); err != nil {
		log.Fatal("auto migrate:", err)
	}
	orm.Init(db)

	// ── 4. Build API ─────────────────────────────────────────────────────────
	api := ninja.New(ninja.Config{
		Title:       cfg.App.Name,
		Version:     cfg.App.Version,
		Description: "A full-featured gin-ninja example with bootstrap, middleware, settings, caching, streaming, and API versioning demos.",
		Prefix:      "/api",
		Versions: map[string]ninja.VersionConfig{
			"v1": {
				Prefix:       "/v1",
				Description:  "Legacy example API demonstrating deprecation headers.",
				Deprecated:   false,
				Sunset:       "Wed, 31 Dec 2026 23:59:59 GMT",
				MigrationURL: "https://example.com/docs/gin-ninja/v2-migration",
			},
			"v2": {
				Prefix:      "/v2",
				Deprecated:  true,
				Description: "Current example API version.",
			},
		},
		SecuritySchemes: map[string]ninja.SecurityScheme{
			"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
		},
		DisableGinDefault: true,
	})
	api.MustProvide(app.NewUserRepo)
	api.MustProvideNamed("config", &cfg)
	api.OnShutdown(func(ctx context.Context, api *ninja.NinjaAPI) error {
		logger.Sync()
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	})

	// ── 5. Global middleware (engine level) ──────────────────────────────────
	api.UseGin(
		middleware.RequestID(),
		middleware.Recovery(log_),
		middleware.Logger(log_),
		middleware.CORS(nil),
		orm.Middleware(db),
	)

	// ── 6. Auth router (public) ──────────────────────────────────────────────
	authRouter := ninja.NewRouter(
		"/auth",
		ninja.WithTags("Auth"),
		ninja.WithTagDescription("Auth", "Authentication endpoints for login and registration"),
		ninja.WithVersion("v1"),
	)
	ninja.Post(authRouter, "/register", app.Register, ninja.Summary("Register a new user"))
	ninja.Post(authRouter, "/login", app.Login, ninja.Summary("Login and get JWT token"))
	api.AddRouter(authRouter)

	// ── 7. Users router (protected by JWT) ───────────────────────────────────
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

	// ── 8. Feature demos (public, for manual testing) ────────────────────────
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
		ninja.Description("Demonstrates route-level response caching, Cache-Control, and conditional requests with ETag."),
		ninja.Cache(time.Minute),
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

	versionedV1Router := ninja.NewRouter(
		"/examples/versioned",
		ninja.WithTags("Examples"),
		ninja.WithVersion("v1"),
	)
	ninja.Get(versionedV1Router, "/info", app.VersionedInfoV1,
		ninja.Summary("Versioned info (v1)"),
		ninja.Description("Demonstrates version-scoped routing and deprecation headers on a legacy version."),
	)
	api.AddRouter(versionedV1Router)

	versionedV2Router := ninja.NewRouter(
		"/examples/versioned",
		ninja.WithTags("Examples"),
		ninja.WithVersion("v2"),
	)
	ninja.Get(versionedV2Router, "/info", app.VersionedInfoV2,
		ninja.Summary("Versioned info (v2)"),
		ninja.Description("Demonstrates version-scoped routing for the current API version."),
	)
	api.AddRouter(versionedV2Router)

	// ── 9. Health-check (no auth) ─────────────────────────────────────────────
	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	// ── 10. Start server ──────────────────────────────────────────────────────
	addr := cfg.Server.Addr()
	log.Printf("Starting %s v%s on http://%s", cfg.App.Name, cfg.App.Version, addr)
	log.Printf("Swagger UI: http://%s/docs", addr)
	if err := api.Run(addr); err != nil {
		log.Fatal(err)
	}
}
