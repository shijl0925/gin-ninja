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
//   - GET  http://localhost:8080/api/v1/users        – list users (requires JWT)
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
	if err := db.AutoMigrate(&app.User{}); err != nil {
		log.Fatal("auto migrate:", err)
	}
	orm.Init(db)

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
		ninja.Description("Returns a paginated list of users"))
	ninja.Get(usersRouter, "/:id", app.GetUser,
		ninja.Summary("Get user"))
	ninja.Post(usersRouter, "/", app.CreateUser,
		ninja.Summary("Create user"))
	ninja.Put(usersRouter, "/:id", app.UpdateUser,
		ninja.Summary("Update user"))
	ninja.Delete(usersRouter, "/:id", app.DeleteUser,
		ninja.Summary("Delete user"))

	api.AddRouter(usersRouter)

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
