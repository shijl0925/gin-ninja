// Package main demonstrates the gin-ninja Controller pattern.
//
// A Controller groups related route registrations into a struct so that
// dependencies (e.g. database, service layer) are injected once and reused
// across all handlers.  This mirrors the API-controller pattern found in
// frameworks like django-ninja.
//
// Run:
//
//	go run ./examples/controller
//
// Then visit:
//   - http://localhost:8080/docs
//   - http://localhost:8080/openapi.json
package main

import (
	"errors"
	"log"
	"net/http"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var runControllerMain = run
var fatalController = func(v ...any) { log.Fatal(v...) }

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Book is a GORM model. gorm.Model provides ID (auto-increment primary key),
// CreatedAt, UpdatedAt, and DeletedAt (soft-delete) fields automatically —
// no manual mutex or sequence counter needed.
type Book struct {
	gorm.Model
	Title  string `gorm:"not null"`
	Author string `gorm:"not null"`
}

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

type BookOut struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

type ListBooksInput struct {
	pagination.PageInput
}

type GetBookInput struct {
	BookID uint `path:"id" binding:"required"`
}

type CreateBookInput struct {
	Title  string `json:"title"  binding:"required"`
	Author string `json:"author" binding:"required"`
}

type UpdateBookInput struct {
	BookID uint   `path:"id"     binding:"required"`
	Title  string `json:"title"  binding:"omitempty"`
	Author string `json:"author" binding:"omitempty"`
}

type DeleteBookInput struct {
	BookID uint `path:"id" binding:"required"`
}

// ---------------------------------------------------------------------------
// BookController — implements ninja.Controller
// ---------------------------------------------------------------------------

// BookController groups all book-related handlers and holds a *gorm.DB
// dependency injected at construction time.
type BookController struct {
	db *gorm.DB
}

// Register wires up the CRUD endpoints onto the provided router.
// This method is the single registration point for every book route.
func (c *BookController) Register(r *ninja.Router) {
	ninja.Get(r, "/", c.List,
		ninja.Summary("List books"),
		ninja.Paginated[BookOut](),
	)
	ninja.Get(r, "/:id", c.Get,
		ninja.Summary("Get book"),
	)
	ninja.Post(r, "/", c.Create,
		ninja.Summary("Create book"),
	)
	ninja.Put(r, "/:id", c.Update,
		ninja.Summary("Update book"),
	)
	ninja.Delete(r, "/:id", c.Delete,
		ninja.Summary("Delete book"),
	)
}

func (c *BookController) List(_ *ninja.Context, in *ListBooksInput) (*pagination.Page[BookOut], error) {
	var books []Book
	if err := c.db.Find(&books).Error; err != nil {
		return nil, err
	}
	total := int64(len(books))

	page := in.GetPage()
	size := in.GetSize()
	start := (page - 1) * size
	if start >= int(total) {
		return pagination.NewPage([]BookOut{}, total, in.PageInput), nil
	}
	end := start + size
	if end > int(total) {
		end = int(total)
	}
	out := make([]BookOut, 0, end-start)
	for _, b := range books[start:end] {
		out = append(out, BookOut{ID: b.ID, Title: b.Title, Author: b.Author})
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func (c *BookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
	var book Book
	if err := c.db.First(&book, in.BookID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *BookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
	book := Book{Title: in.Title, Author: in.Author}
	if err := c.db.Create(&book).Error; err != nil {
		return nil, err
	}
	return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *BookController) Update(_ *ninja.Context, in *UpdateBookInput) (*BookOut, error) {
	var book Book
	if err := c.db.First(&book, in.BookID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	if in.Title != "" {
		book.Title = in.Title
	}
	if in.Author != "" {
		book.Author = in.Author
	}
	if err := c.db.Save(&book).Error; err != nil {
		return nil, err
	}
	return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *BookController) Delete(_ *ninja.Context, in *DeleteBookInput) error {
	result := c.db.Delete(&Book{}, in.BookID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ninja.NotFoundError()
	}
	return nil
}

// ---------------------------------------------------------------------------
// API wiring
// ---------------------------------------------------------------------------

func buildAPI() *ninja.NinjaAPI {
	// Open an in-memory SQLite database. Each call to buildAPI gets a fresh,
	// isolated DB — ideal for tests. In production replace this with a
	// persistent driver, e.g. gorm.Open(postgres.Open(dsn), &gorm.Config{}).
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open database: " + err.Error())
	}
	// AutoMigrate creates the books table from the Book model definition.
	if err := db.AutoMigrate(&Book{}); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	api := ninja.New(ninja.Config{
		Title:             "Gin Ninja Controller Example",
		Version:           "1.0.0",
		Description:       "Demonstrates the ninja.Controller pattern for grouping related routes into a struct.",
		Prefix:            "/api/v1",
		DisableGinDefault: true,
	})

	api.UseGin(
		ginpkg.Logger(),
		ginpkg.Recovery(),
		middleware.RequestID(),
		middleware.CORS(nil),
	)

	// Mount the BookController.  Dependencies are injected here; the
	// controller's Register method handles all route wiring internally.
	api.AddController("/books", &BookController{db: db},
		ninja.WithTags("Books"),
		ninja.WithTagDescription("Books", "CRUD endpoints for the book catalogue"),
	)

	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	return api
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func run(addr string) error {
	api := buildAPI()
	log.Printf("Docs: http://localhost:8080/docs")
	return api.Run(addr)
}

func main() {
	if err := runControllerMain(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatalController(err)
	}
}
