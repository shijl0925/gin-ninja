package ninja

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const apiContextKey = "gin_ninja_api"

var (
	errorType      = reflect.TypeOf((*error)(nil)).Elem()
	contextType    = reflect.TypeOf((*context.Context)(nil)).Elem()
	ginContextType = reflect.TypeOf(&gin.Context{})
	ninjaCtxType   = reflect.TypeOf(&Context{})
)

type providerResolver struct {
	name       string
	targetType reflect.Type
	resolve    func(*Context) (reflect.Value, error)
}

type providerRegistry struct {
	mu         sync.RWMutex
	named      map[string]providerResolver
	registered []providerResolver
}

func newProviderRegistry() *providerRegistry {
	return &providerRegistry{
		named: make(map[string]providerResolver),
	}
}

// Provide registers application-level dependencies that can be injected into
// handler input structs via `inject:""` or resolved inside handlers with ctx.Resolve.
// Values can be concrete instances, zero-argument factories, or factories that
// accept *Context, context.Context, or *gin.Context.
func (api *NinjaAPI) Provide(values ...any) error {
	for _, value := range values {
		resolver, err := newProviderResolver("", value)
		if err != nil {
			return err
		}
		api.providers.mu.Lock()
		api.providers.registered = append(api.providers.registered, resolver)
		api.providers.mu.Unlock()
	}
	return nil
}

// MustProvide is like Provide but panics on error.
func (api *NinjaAPI) MustProvide(values ...any) {
	if err := api.Provide(values...); err != nil {
		panic(err)
	}
}

// ProvideNamed registers a named dependency for explicit `inject:"name"` usage.
func (api *NinjaAPI) ProvideNamed(name string, value any) error {
	resolver, err := newProviderResolver(name, value)
	if err != nil {
		return err
	}
	api.providers.mu.Lock()
	api.providers.named[name] = resolver
	api.providers.mu.Unlock()
	return nil
}

// MustProvideNamed is like ProvideNamed but panics on error.
func (api *NinjaAPI) MustProvideNamed(name string, value any) {
	if err := api.ProvideNamed(name, value); err != nil {
		panic(err)
	}
}

func newProviderResolver(name string, value any) (providerResolver, error) {
	if value == nil {
		return providerResolver{}, fmt.Errorf("provider %q: nil is not allowed", name)
	}

	v := reflect.ValueOf(value)
	t := v.Type()
	if t.Kind() != reflect.Func {
		return providerResolver{
			name:       name,
			targetType: t,
			resolve: func(*Context) (reflect.Value, error) {
				return v, nil
			},
		}, nil
	}

	if t.NumOut() == 0 || t.NumOut() > 2 {
		return providerResolver{}, fmt.Errorf("provider %q: functions must return one value or (value, error)", name)
	}
	if t.NumOut() == 2 && !t.Out(1).Implements(errorType) {
		return providerResolver{}, fmt.Errorf("provider %q: second return value must implement error", name)
	}
	if t.NumIn() > 1 {
		return providerResolver{}, fmt.Errorf("provider %q: functions may accept at most one argument", name)
	}

	var argType reflect.Type
	if t.NumIn() == 1 {
		argType = t.In(0)
		switch {
		case argType == ninjaCtxType, argType == ginContextType, argType == contextType:
		default:
			return providerResolver{}, fmt.Errorf("provider %q: unsupported argument type %s", name, argType)
		}
	}

	return providerResolver{
		name:       name,
		targetType: t.Out(0),
		resolve: func(ctx *Context) (reflect.Value, error) {
			args := []reflect.Value{}
			if argType != nil {
				switch argType {
				case ninjaCtxType:
					args = append(args, reflect.ValueOf(ctx))
				case ginContextType:
					args = append(args, reflect.ValueOf(ctx.Context))
				case contextType:
					args = append(args, reflect.ValueOf(ctx.StdContext()))
				}
			}
			out := v.Call(args)
			if len(out) == 2 && !out[1].IsNil() {
				err, _ := out[1].Interface().(error)
				return reflect.Value{}, err
			}
			return out[0], nil
		},
	}, nil
}

func (c *Context) API() *NinjaAPI {
	v, exists := c.Get(apiContextKey)
	if !exists {
		return nil
	}
	api, _ := v.(*NinjaAPI)
	return api
}

// Resolve stores a dependency into target, which must be a non-nil pointer.
func (c *Context) Resolve(target any) error {
	return resolveIntoTarget(c, target)
}

// MustResolve is like Resolve but panics on error.
func (c *Context) MustResolve(target any) {
	if err := c.Resolve(target); err != nil {
		panic(err)
	}
}

func resolveIntoTarget(ctx *Context, target any) error {
	if target == nil {
		return fmt.Errorf("resolve target must not be nil")
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("resolve target must be a non-nil pointer")
	}
	resolved, err := resolveDependency(ctx, value.Elem().Type(), "")
	if err != nil {
		return err
	}
	value.Elem().Set(resolved)
	return nil
}

func injectDependencies(ctx *Context, t reflect.Type, v reflect.Value) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := injectDependencies(ctx, field.Type, fv); err != nil {
				return err
			}
			continue
		}

		tag, ok := injectTag(field)
		if !ok {
			continue
		}

		resolved, err := resolveDependency(ctx, field.Type, tag)
		if err != nil {
			return &Error{
				Status:  500,
				Code:    "DEPENDENCY_INJECTION_ERROR",
				Message: fmt.Sprintf("inject field '%s': %s", field.Name, err.Error()),
			}
		}
		fv.Set(resolved)
	}
	return nil
}

func resolveDependency(ctx *Context, targetType reflect.Type, name string) (reflect.Value, error) {
	if targetType == nil {
		return reflect.Value{}, fmt.Errorf("target type must not be nil")
	}
	if targetType == ninjaCtxType {
		return reflect.ValueOf(ctx), nil
	}
	if targetType == ginContextType {
		return reflect.ValueOf(ctx.Context), nil
	}

	if name != "" {
		if value, ok, err := resolveNamedRequestValue(ctx, name, targetType); ok || err != nil {
			return value, err
		}
		api := ctx.API()
		if api == nil {
			return reflect.Value{}, fmt.Errorf("named dependency %q is unavailable", name)
		}
		return api.providers.resolveNamed(ctx, name, targetType)
	}

	if value, ok := resolveRequestValueByType(ctx, targetType); ok {
		return value, nil
	}

	api := ctx.API()
	if api == nil {
		return reflect.Value{}, fmt.Errorf("no provider registered for %s", targetType)
	}
	return api.providers.resolveByType(ctx, targetType)
}

func resolveNamedRequestValue(ctx *Context, name string, targetType reflect.Type) (reflect.Value, bool, error) {
	value, exists := ctx.Get(name)
	if !exists {
		return reflect.Value{}, false, nil
	}
	resolved, ok := makeAssignableValue(reflect.ValueOf(value), targetType)
	if !ok {
		return reflect.Value{}, true, fmt.Errorf("request value %q is not assignable to %s", name, targetType)
	}
	return resolved, true, nil
}

func resolveRequestValueByType(ctx *Context, targetType reflect.Type) (reflect.Value, bool) {
	if ctx.Keys == nil {
		return reflect.Value{}, false
	}

	var fallback reflect.Value
	for key, value := range ctx.Keys {
		if key == apiContextKey {
			continue
		}
		resolved, ok := makeAssignableValue(reflect.ValueOf(value), targetType)
		if !ok {
			continue
		}
		if resolved.Type() == targetType {
			return resolved, true
		}
		if !fallback.IsValid() {
			fallback = resolved
		}
	}
	if fallback.IsValid() {
		return fallback, true
	}
	return reflect.Value{}, false
}

func (r *providerRegistry) resolveNamed(ctx *Context, name string, targetType reflect.Type) (reflect.Value, error) {
	r.mu.RLock()
	resolver, ok := r.named[name]
	r.mu.RUnlock()
	if !ok {
		return reflect.Value{}, fmt.Errorf("named dependency %q is not registered", name)
	}
	value, err := resolver.resolve(ctx)
	if err != nil {
		return reflect.Value{}, err
	}
	resolved, ok := makeAssignableValue(value, targetType)
	if !ok {
		return reflect.Value{}, fmt.Errorf("named dependency %q is not assignable to %s", name, targetType)
	}
	return resolved, nil
}

func (r *providerRegistry) resolveByType(ctx *Context, targetType reflect.Type) (reflect.Value, error) {
	r.mu.RLock()
	providers := append([]providerResolver(nil), r.registered...)
	r.mu.RUnlock()

	var candidate reflect.Value
	var count int
	for _, provider := range providers {
		if !providerMatches(provider.targetType, targetType) {
			continue
		}
		value, err := provider.resolve(ctx)
		if err != nil {
			return reflect.Value{}, err
		}
		resolved, ok := makeAssignableValue(value, targetType)
		if !ok {
			continue
		}
		if resolved.Type() == targetType {
			return resolved, nil
		}
		candidate = resolved
		count++
	}

	switch count {
	case 0:
		return reflect.Value{}, fmt.Errorf("no provider registered for %s", targetType)
	case 1:
		return candidate, nil
	default:
		return reflect.Value{}, fmt.Errorf("multiple providers match %s; use inject:\"name\"", targetType)
	}
}

func providerMatches(sourceType, targetType reflect.Type) bool {
	if sourceType == targetType {
		return true
	}
	if sourceType.AssignableTo(targetType) {
		return true
	}
	if targetType.Kind() == reflect.Interface && sourceType.Implements(targetType) {
		return true
	}
	return false
}

func makeAssignableValue(value reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	if value.Type() == targetType || value.Type().AssignableTo(targetType) {
		return value, true
	}
	if targetType.Kind() == reflect.Interface && value.Type().Implements(targetType) {
		return value, true
	}
	return reflect.Value{}, false
}

func injectTag(field reflect.StructField) (string, bool) {
	tag, ok := field.Tag.Lookup("inject")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(tag), true
}

func isInjectedField(field reflect.StructField) bool {
	_, ok := injectTag(field)
	return ok
}
