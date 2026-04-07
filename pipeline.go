package ninja

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MiddlewareFunc is a typed middleware that can abort a request by returning an error.
type MiddlewareFunc func(*Context) error

// NextHandler advances an interceptor chain.
type NextHandler func(*Context) (any, error)

// Interceptor wraps a typed handler invocation.
type Interceptor func(*Context, NextHandler) (any, error)

// RequestTransformer mutates or validates the bound request payload before the handler runs.
type RequestTransformer func(*Context, any) error

// ResponseTransformer rewrites a successful handler result before it is serialized.
type ResponseTransformer func(*Context, any) (any, error)

// UseMiddleware attaches typed middleware directly to an operation.
func UseMiddleware(middleware ...MiddlewareFunc) OperationOption {
	return func(op *operation) {
		op.middleware = append(op.middleware, middleware...)
	}
}

// MiddlewareChain attaches one or more named middleware chains to an operation.
func MiddlewareChain(names ...string) OperationOption {
	return func(op *operation) {
		op.middlewareChainNames = append(op.middlewareChainNames, names...)
	}
}

// Intercept attaches one or more interceptors to an operation.
func Intercept(interceptors ...Interceptor) OperationOption {
	return func(op *operation) {
		op.interceptors = append(op.interceptors, interceptors...)
	}
}

// TransformRequest attaches request transformers to an operation.
func TransformRequest(transformers ...RequestTransformer) OperationOption {
	return func(op *operation) {
		op.requestTransformers = append(op.requestTransformers, transformers...)
	}
}

// TransformResponse attaches response transformers to an operation.
func TransformResponse(transformers ...ResponseTransformer) OperationOption {
	return func(op *operation) {
		op.responseTransformers = append(op.responseTransformers, transformers...)
	}
}

type routePipeline struct {
	interceptors         []Interceptor
	requestTransformers  []RequestTransformer
	responseTransformers []ResponseTransformer
}

func (p routePipeline) extend(interceptors []Interceptor, request []RequestTransformer, response []ResponseTransformer) routePipeline {
	next := routePipeline{
		interceptors:         append([]Interceptor(nil), p.interceptors...),
		requestTransformers:  append([]RequestTransformer(nil), p.requestTransformers...),
		responseTransformers: append([]ResponseTransformer(nil), p.responseTransformers...),
	}
	next.interceptors = append(next.interceptors, interceptors...)
	next.requestTransformers = append(next.requestTransformers, request...)
	next.responseTransformers = append(next.responseTransformers, response...)
	return next
}

func cloneMiddlewareChains(registry map[string][]MiddlewareFunc) map[string][]MiddlewareFunc {
	if len(registry) == 0 {
		return nil
	}
	cloned := make(map[string][]MiddlewareFunc, len(registry))
	for name, chain := range registry {
		cloned[name] = append([]MiddlewareFunc(nil), chain...)
	}
	return cloned
}

func mergeMiddlewareChains(base map[string][]MiddlewareFunc, extra map[string][]MiddlewareFunc) map[string][]MiddlewareFunc {
	merged := cloneMiddlewareChains(base)
	if len(extra) == 0 {
		return merged
	}
	if merged == nil {
		merged = map[string][]MiddlewareFunc{}
	}
	for name, chain := range extra {
		merged[name] = append([]MiddlewareFunc(nil), chain...)
	}
	return merged
}

func resolveMiddlewareChains(registry map[string][]MiddlewareFunc, names []string) ([]MiddlewareFunc, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var middleware []MiddlewareFunc
	for _, name := range names {
		chain, ok := registry[name]
		if !ok {
			return nil, fmt.Errorf("ninja: middleware chain %q is not registered", name)
		}
		middleware = append(middleware, chain...)
	}
	return middleware, nil
}

func typedMiddlewareHandler(mw MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := newContext(c)
		if err := mw(ctx); err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func errorAbortHandler(err error) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeError(c, NewError(http.StatusInternalServerError, err.Error()))
		c.Abort()
	}
}

func runInterceptors(ctx *Context, interceptors []Interceptor, final NextHandler) (any, error) {
	next := final
	for i := len(interceptors) - 1; i >= 0; i-- {
		interceptor := interceptors[i]
		previous := next
		next = func(current *Context) (any, error) {
			return interceptor(current, previous)
		}
	}
	return next(ctx)
}
