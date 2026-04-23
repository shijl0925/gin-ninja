package ninja

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterOption is a functional option for configuring a Router.
type RouterOption func(*Router)

// WithTags adds OpenAPI tags to all operations registered on this router.
func WithTags(tags ...string) RouterOption {
	return func(r *Router) {
		r.tags = append(r.tags, tags...)
	}
}

// WithTagDescription records a top-level OpenAPI tag description.
func WithTagDescription(tag, description string) RouterOption {
	return func(r *Router) {
		if r.tagDescriptions == nil {
			r.tagDescriptions = map[string]string{}
		}
		r.tagDescriptions[tag] = description
	}
}

// WithTagDescriptions records multiple top-level OpenAPI tag descriptions.
func WithTagDescriptions(descriptions map[string]string) RouterOption {
	return func(r *Router) {
		if r.tagDescriptions == nil {
			r.tagDescriptions = map[string]string{}
		}
		for tag, description := range descriptions {
			r.tagDescriptions[tag] = description
		}
	}
}

// WithSecurity adds an OpenAPI security requirement to all operations
// registered on this router.
func WithSecurity(name string, scopes ...string) RouterOption {
	return func(r *Router) {
		r.security = append(r.security, SecurityRequirement{name: append([]string{}, scopes...)})
	}
}

// WithBearerAuth applies the default JWT bearer OpenAPI security requirement.
func WithBearerAuth() RouterOption {
	return WithSecurity("bearerAuth")
}

// WithVersion marks the router and its nested operations as belonging to a
// named API version.
func WithVersion(version string) RouterOption {
	return func(r *Router) {
		r.version = version
	}
}

// Router groups a set of API endpoints under a common URL prefix.
// Routers can be nested arbitrarily.
type Router struct {
	prefix          string
	tags            []string
	tagDescriptions map[string]string
	operations      []*operation
	subrouters      []*Router
	security        []SecurityRequirement
	middleware      []func(*Context) error
	ginMiddleware   []gin.HandlerFunc
	version         string
}

// NewRouter creates a new Router with the given URL prefix and options.
//
//	r := ninja.NewRouter("/users", ninja.WithTags("Users"))
func NewRouter(prefix string, opts ...RouterOption) *Router {
	r := &Router{prefix: prefix}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// AddRouter mounts a sub-router under this router.
// The sub-router's prefix is appended to this router's prefix.
func (r *Router) AddRouter(sub *Router) {
	r.subrouters = append(r.subrouters, sub)
}

// Use adds a typed middleware function that runs before every handler on this
// router.  Returning a non-nil error aborts the request with an appropriate
// error response.
func (r *Router) Use(mw func(*Context) error) {
	r.middleware = append(r.middleware, mw)
}

// UseGin adds one or more raw gin.HandlerFunc middleware to this router.
// Use this to attach infrastructure middleware (JWT, CORS, rate limiting, etc.)
// at the router level instead of the engine level.
//
//	r := ninja.NewRouter("/admin", ninja.WithTags("Admin"))
//	r.UseGin(middleware.JWTAuthWithSecret("secret"))
func (r *Router) UseGin(mw ...gin.HandlerFunc) {
	r.ginMiddleware = append(r.ginMiddleware, mw...)
}

// ---------------------------------------------------------------------------
// Route registration helpers (typed generics)
// ---------------------------------------------------------------------------

// Get registers a GET endpoint.
//
//	type ListUsersQuery struct {
//	    Page int `form:"page"`
//	    Size int `form:"size"`
//	}
//	ninja.Get(router, "/", listUsersHandler)
func Get[TIn any, TOut any](r *Router, path string, handler func(*Context, *TIn) (*TOut, error), opts ...OperationOption) {
	registerTypedOperation(r, newOperation[TIn, TOut](http.MethodGet, path, handler, r.tags), opts...)
}

// Post registers a POST endpoint.
func Post[TIn any, TOut any](r *Router, path string, handler func(*Context, *TIn) (*TOut, error), opts ...OperationOption) {
	op := newOperation[TIn, TOut](http.MethodPost, path, handler, r.tags)
	if op.successStatus == http.StatusOK {
		op.successStatus = http.StatusCreated
	}
	registerTypedOperation(r, op, opts...)
}

// Put registers a PUT endpoint.
func Put[TIn any, TOut any](r *Router, path string, handler func(*Context, *TIn) (*TOut, error), opts ...OperationOption) {
	registerTypedOperation(r, newOperation[TIn, TOut](http.MethodPut, path, handler, r.tags), opts...)
}

// Patch registers a PATCH endpoint.
func Patch[TIn any, TOut any](r *Router, path string, handler func(*Context, *TIn) (*TOut, error), opts ...OperationOption) {
	registerTypedOperation(r, newOperation[TIn, TOut](http.MethodPatch, path, handler, r.tags), opts...)
}

func registerTypedOperation(r *Router, op *operation, opts ...OperationOption) {
	op.security = cloneSecurityRequirements(r.security)
	op.tagDescriptions = cloneStringMap(r.tagDescriptions)
	for _, opt := range opts {
		opt(op)
	}
	op.finalize()
	r.operations = append(r.operations, op)
}

// Controller groups related routes for a single domain resource.
// Implement Register to wire all of its endpoints onto the provided Router.
//
// Use api.AddController to create the router, call Register, and mount it in
// one step:
//
//	type BookController struct{ db *gorm.DB }
//
//	func (c *BookController) Register(r *ninja.Router) {
//	    ninja.Get(r,    "/",    c.List,   ninja.Summary("List books"))
//	    ninja.Post(r,   "/",    c.Create, ninja.Summary("Create book"))
//	    ninja.Get(r,    "/:id", c.Get,    ninja.Summary("Get book"))
//	    ninja.Put(r,    "/:id", c.Update, ninja.Summary("Update book"))
//	    ninja.Delete(r, "/:id", c.Delete, ninja.Summary("Delete book"))
//	}
//
//	api.AddController("/books", &BookController{db: db},
//	    ninja.WithTags("Books"),
//	    ninja.WithBearerAuth(),
//	)
type Controller interface {
	Register(r *Router)
}

// ControllerFunc is an adapter that lets a plain function serve as a
// [Controller].  It is useful for small, inline controllers and in tests.
//
//	api.AddController("/items", ninja.ControllerFunc(func(r *ninja.Router) {
//	    ninja.Get(r, "/", listItems)
//	    ninja.Post(r, "/", createItem)
//	}), ninja.WithTags("Items"))
type ControllerFunc func(r *Router)

// Register calls f(r), satisfying the [Controller] interface.
func (f ControllerFunc) Register(r *Router) { f(r) }

// Delete registers a DELETE endpoint with a typed input but no response body.
//
//	type DeleteUserInput struct {
//	    UserID int `path:"id"`
//	}
//	ninja.Delete(router, "/:id", deleteUserHandler)
func Delete[TIn any](r *Router, path string, handler func(*Context, *TIn) error, opts ...OperationOption) {
	op := newVoidOperation[TIn](http.MethodDelete, path, handler, r.tags)
	op.security = cloneSecurityRequirements(r.security)
	op.tagDescriptions = cloneStringMap(r.tagDescriptions)
	for _, opt := range opts {
		opt(op)
	}
	op.finalize()
	r.operations = append(r.operations, op)
}
