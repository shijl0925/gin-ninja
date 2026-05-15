# Getting Started

## Quick Start

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/middleware"
    "github.com/shijl0925/gin-ninja/settings"
)

type HelloInput struct {
    Name string `query:"name" binding:"required"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    api := ninja.New(ninja.Config{
        Title:             "Hello API",
        Version:           "1.0.0",
        DisableGinDefault: true, // use custom middleware instead
    })

    api.UseGin(
        gin.Logger(),                // keep native [GIN] access logs
        gin.Recovery(),              // keep native panic recovery
        middleware.RequestID(),
        middleware.CORSFromConfig(settings.CORSConfig{}),
    )

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

After startup you can visit:

- `http://localhost:8080/` for the default welcome homepage
- `http://localhost:8080/docs` for the Swagger UI
- `http://localhost:8080/openapi.json` for the raw OpenAPI document

If you want the homepage to include a shortcut to your admin backend, set `AdminURL` in `ninja.Config`.
If you want to keep Swagger UI enabled but hide the homepage shortcut in production, set `HideDocsShortcut: true`.

---

## API Controller

The `Controller` interface lets you group all routes for a single resource into one struct, injecting shared dependencies (database, service layer) once at construction time and reusing them across every handler — the same pattern as django-ninja's `APIController`.

### Interface

```go
type Controller interface {
    Register(r *ninja.Router)
}
```

Implement `Register` to wire all endpoints onto the provided `Router`. Then mount it with `api.AddController`.

### Example

```go
import (
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/pagination"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// Book is a GORM model — gorm.Model provides auto-increment ID, CreatedAt,
// UpdatedAt, and soft-delete (DeletedAt) fields automatically.
type Book struct {
    gorm.Model
    Title  string `gorm:"not null"`
    Author string `gorm:"not null"`
}

// --- request / response schemas ---

type BookOut struct {
    ID     uint   `json:"id"`
    Title  string `json:"title"`
    Author string `json:"author"`
}

type ListBooksInput  struct{ pagination.PageInput }
type GetBookInput    struct{ BookID uint `path:"id" binding:"required"` }
type CreateBookInput struct {
    Title  string `json:"title"  binding:"required"`
    Author string `json:"author" binding:"required"`
}
type UpdateBookInput struct {
    BookID uint   `path:"id"     binding:"required"`
    Title  string `json:"title"  binding:"omitempty"`
    Author string `json:"author" binding:"omitempty"`
}
type DeleteBookInput struct{ BookID uint `path:"id" binding:"required"` }

// --- controller ---

type BookController struct {
    db *gorm.DB   // injected once, shared by all handlers
}

// Register wires every CRUD endpoint onto the router.
func (c *BookController) Register(r *ninja.Router) {
    ninja.Get(r,    "/",    c.List,   ninja.Summary("List books"), ninja.Paginated[BookOut]())
    ninja.Get(r,    "/:id", c.Get,    ninja.Summary("Get book"))
    ninja.Post(r,   "/",    c.Create, ninja.Summary("Create book"))
    ninja.Put(r,    "/:id", c.Update, ninja.Summary("Update book"))
    ninja.Delete(r, "/:id", c.Delete, ninja.Summary("Delete book"))
}

func (c *BookController) List(_ *ninja.Context, in *ListBooksInput) (*pagination.Page[BookOut], error) {
    var books []Book
    c.db.Find(&books)
    // ... paginate and return ...
}

func (c *BookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
    var book Book
    if err := c.db.First(&book, in.BookID).Error; err != nil {
        return nil, ninja.NotFoundError()
    }
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *BookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
    book := Book{Title: in.Title, Author: in.Author}
    c.db.Create(&book)
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

// Update and Delete follow the same pattern.

// --- wiring ---

func main() {
    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    db.AutoMigrate(&Book{})

    api := ninja.New(ninja.Config{Title: "Books API", Version: "1.0.0"})

    // All router options (tags, auth, middleware) are set here.
    // BookController.Register handles every route internally.
    api.AddController("/books", &BookController{db: db},
        ninja.WithTags("Books"),
        ninja.WithTagDescription("Books", "CRUD endpoints for the book catalogue"),
    )

    api.Run(":8080")
}
```

### Why dependency injection?

Passing dependencies into the struct at construction time (rather than reading package-level globals) gives three concrete advantages:

| Concern | With DI | Without DI (global) |
|---|---|---|
| **Unit tests** | Pass an in-memory DB; no shared state between tests | Every test touches the same global; setup/teardown is fragile |
| **Multiple instances** | Same controller type can be mounted with different DBs (primary + read replica) | Only one DB reachable from the handler |
| **Explicit dependencies** | All requirements visible in the struct definition | Hidden global state; harder to reason about |

For small scripts or quick prototypes where testability is not a priority, a package-level global is perfectly fine.

### Without dependency injection (package-level variable)

If you prefer to keep a shared database handle at the package level instead of threading it through the struct, the controller still compiles and works — the `Controller` interface has no opinion on how the handler accesses its state.

```go
package main

import (
    "sync"

    ninja "github.com/shijl0925/gin-ninja"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// db is initialized once in main() and shared across all handlers.
// No pointer is stored in the controller struct.
var (
    db     *gorm.DB
    dbOnce sync.Once
)

func initDB() {
    dbOnce.Do(func() {
        var err error
        db, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
        if err != nil {
            panic(err)
        }
        db.AutoMigrate(&Book{})
    })
}

// StaticBookController has no fields — it accesses the global `db` directly.
type StaticBookController struct{}

func (c *StaticBookController) Register(r *ninja.Router) {
    ninja.Get(r,    "/",    c.List,   ninja.Summary("List books"))
    ninja.Get(r,    "/:id", c.Get,    ninja.Summary("Get book"))
    ninja.Post(r,   "/",    c.Create, ninja.Summary("Create book"))
    ninja.Delete(r, "/:id", c.Delete, ninja.Summary("Delete book"))
}

func (c *StaticBookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
    var book Book
    if err := db.First(&book, in.BookID).Error; err != nil {
        return nil, ninja.NotFoundError()
    }
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *StaticBookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
    book := Book{Title: in.Title, Author: in.Author}
    db.Create(&book)
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

// List and Delete follow the same pattern, accessing `db` directly.

func main() {
    initDB()

    api := ninja.New(ninja.Config{Title: "Books API", Version: "1.0.0"})
    api.AddController("/books", &StaticBookController{}, ninja.WithTags("Books"))
    api.Run(":8080")
}
```

> **Trade-off** — `StaticBookController` is simpler to wire but harder to test in isolation: swapping the database requires changing the global, and parallel tests that each need a fresh DB become difficult to manage. For production services, prefer the dependency-injected form.

### Inline controller with `ControllerFunc`

For small or test scenarios where a full struct is unnecessary, use the `ControllerFunc` adapter:

```go
api.AddController("/items", ninja.ControllerFunc(func(r *ninja.Router) {
    ninja.Get(r,  "/",    listItems,  ninja.Summary("List items"))
    ninja.Post(r, "/",    createItem, ninja.Summary("Create item"))
}), ninja.WithTags("Items"))
```

### When to use Controller vs. Router

| Scenario | Recommended approach |
|---|---|
| A group of routes that share one dependency (DB, service) | `Controller` — dependency injected once in the struct |
| Independent utility routes with no shared state | `NewRouter` + `api.AddRouter` |
| Inline / test routes with no struct needed | `ControllerFunc` |

> See [examples/controller](../../examples/controller/) for a fully runnable example with GORM SQLite.

---

## Lightweight path recommendation

For a small or medium CRUD/internal API, start with the shortest path and add layers only when needed:

- Follow [examples/basic](../../examples/basic/): `New + Router + Handler + orm.Middleware`
- Start scaffolding with the default `minimal` template: `gin-ninja-cli startproject mysite -module github.com/acme/mysite`
- Scaffolded repos keep the built-in repo interface layer, while `minimal` still stays the recommended starting point for small CRUD services
- Move to `-template standard|auth|admin` only when you need the extra infrastructure, auth, or admin surface

A good minimal app package usually contains only:

- `app/models.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

---
