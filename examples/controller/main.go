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
	"sync"
	"sync/atomic"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/pagination"
)

var runControllerMain = run
var fatalController = func(v ...any) { log.Fatal(v...) }

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type Book struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
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
// In-memory store (stand-in for a real service / database layer)
// ---------------------------------------------------------------------------

type bookStore struct {
	mu     sync.RWMutex
	seq    atomic.Uint64
	books  map[uint]*Book
}

func newBookStore() *bookStore {
	return &bookStore{books: map[uint]*Book{}}
}

func (s *bookStore) list() []*Book {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Book, 0, len(s.books))
	for _, b := range s.books {
		out = append(out, b)
	}
	return out
}

func (s *bookStore) get(id uint) (*Book, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.books[id]
	return b, ok
}

func (s *bookStore) create(title, author string) *Book {
	id := uint(s.seq.Add(1))
	b := &Book{ID: id, Title: title, Author: author}
	s.mu.Lock()
	s.books[id] = b
	s.mu.Unlock()
	return b
}

func (s *bookStore) update(id uint, title, author string) (*Book, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.books[id]
	if !ok {
		return nil, false
	}
	if title != "" {
		b.Title = title
	}
	if author != "" {
		b.Author = author
	}
	return b, true
}

func (s *bookStore) delete(id uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.books[id]
	if ok {
		delete(s.books, id)
	}
	return ok
}

// ---------------------------------------------------------------------------
// BookController — implements ninja.Controller
// ---------------------------------------------------------------------------

// BookController groups all book-related handlers and holds the store
// dependency.  It implements ninja.Controller so it can be mounted via
// api.AddController.
type BookController struct {
	store *bookStore
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
	all := c.store.list()
	total := int64(len(all))

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
	for _, b := range all[start:end] {
		out = append(out, BookOut{ID: b.ID, Title: b.Title, Author: b.Author})
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func (c *BookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
	b, ok := c.store.get(in.BookID)
	if !ok {
		return nil, ninja.NotFoundError()
	}
	return &BookOut{ID: b.ID, Title: b.Title, Author: b.Author}, nil
}

func (c *BookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
	b := c.store.create(in.Title, in.Author)
	return &BookOut{ID: b.ID, Title: b.Title, Author: b.Author}, nil
}

func (c *BookController) Update(_ *ninja.Context, in *UpdateBookInput) (*BookOut, error) {
	b, ok := c.store.update(in.BookID, in.Title, in.Author)
	if !ok {
		return nil, ninja.NotFoundError()
	}
	return &BookOut{ID: b.ID, Title: b.Title, Author: b.Author}, nil
}

func (c *BookController) Delete(_ *ninja.Context, in *DeleteBookInput) error {
	if !c.store.delete(in.BookID) {
		return ninja.NotFoundError()
	}
	return nil
}

// ---------------------------------------------------------------------------
// API wiring
// ---------------------------------------------------------------------------

func buildAPI() *ninja.NinjaAPI {
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
	api.AddController("/books", &BookController{store: newBookStore()},
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
