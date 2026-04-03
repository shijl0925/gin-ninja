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
//   - http://localhost:8080/docs           – Swagger UI
//   - http://localhost:8080/openapi.json   – raw OpenAPI spec
//   - POST http://localhost:8080/api/v1/auth/register – register a new user
//   - POST http://localhost:8080/api/v1/auth/login   – get a JWT token
//   - GET  http://localhost:8080/api/v1/rbac/me      – inspect current roles and permissions
//   - GET  http://localhost:8080/api/v1/users        – list users (requires System:User:List)
// Seeded demo users:
//   - admin@example.com / password123
//   - manager@example.com / password123
//   - auditor@example.com / password123
package main

import (
	"log"
	"net/http"

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
	if err := db.AutoMigrate(&app.User{}, &app.Role{}, &app.Permission{}); err != nil {
		log.Fatal("auto migrate rbac:", err)
	}
	orm.Init(db)
	if err := app.SeedRBAC(db); err != nil {
		log.Fatal("seed rbac:", err)
	}

	// ── 4. Build API ─────────────────────────────────────────────────────────
	api := ninja.New(ninja.Config{
		Title:       cfg.App.Name,
		Version:     cfg.App.Version,
		Description: "A full-featured gin-ninja example with bootstrap, middleware, and settings.",
		Prefix:      "/api/v1",
		SecuritySchemes: map[string]ninja.SecurityScheme{
			"bearerAuth": ninja.HTTPBearerSecurityScheme("JWT"),
		},
		DisableGinDefault: true,
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
	authRouter := ninja.NewRouter("/auth", ninja.WithTags("Auth"))
	ninja.Post(authRouter, "/register", app.Register, ninja.Summary("Register a new user"))
	ninja.Post(authRouter, "/login", app.Login, ninja.Summary("Login and get JWT token"))
	api.AddRouter(authRouter)

	// ── 7. Users router (protected by JWT) ───────────────────────────────────
	usersRouter := ninja.NewRouter("/users", ninja.WithTags("Users"), ninja.WithBearerAuth())
	usersRouter.UseGin(middleware.JWTAuth())

	ninja.Get(usersRouter, "/", app.ListUsers,
		ninja.Summary("List users"),
		ninja.Description("Returns a paginated list of users"),
		ninja.WithMiddleware(middleware.RequirePermissions(app.ResolvePermissions, app.PermissionUserList)))
	ninja.Get(usersRouter, "/:id", app.GetUser,
		ninja.Summary("Get user"),
		ninja.WithMiddleware(middleware.RequirePermissions(app.ResolvePermissions, app.PermissionUserDetail)))
	ninja.Post(usersRouter, "/", app.CreateUser,
		ninja.Summary("Create user"),
		ninja.WithMiddleware(middleware.RequirePermissions(app.ResolvePermissions, app.PermissionUserCreate)))
	ninja.Put(usersRouter, "/:id", app.UpdateUser,
		ninja.Summary("Update user"),
		ninja.WithMiddleware(middleware.RequirePermissions(app.ResolvePermissions, app.PermissionUserEdit)))
	ninja.Delete(usersRouter, "/:id", app.DeleteUser,
		ninja.Summary("Delete user"),
		ninja.WithMiddleware(middleware.RequirePermissions(app.ResolvePermissions, app.PermissionUserDelete)))

	api.AddRouter(usersRouter)

	rbacRouter := ninja.NewRouter("/rbac", ninja.WithTags("RBAC"), ninja.WithBearerAuth())
	rbacRouter.UseGin(middleware.JWTAuth())
	ninja.Get(rbacRouter, "/me", app.GetCurrentSubject, ninja.Summary("Get current RBAC subject"))
	api.AddRouter(rbacRouter)

	// ── 8. Health-check (no auth) ─────────────────────────────────────────────
	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	// ── 9. Start server ───────────────────────────────────────────────────────
	addr := cfg.Server.Addr()
	log.Printf("Starting %s v%s on http://%s", cfg.App.Name, cfg.App.Version, addr)
	log.Printf("Swagger UI: http://%s/docs", addr)
	if err := api.Run(addr); err != nil {
		log.Fatal(err)
	}
}
