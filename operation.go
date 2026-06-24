package ninja

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
)

// OperationOption is a functional option for configuring an Operation.
type OperationOption func(*operation)

// Summary sets the human-readable summary shown in the OpenAPI docs.
func Summary(s string) OperationOption {
	return func(op *operation) { op.spec.summary = s }
}

// Description sets the long description shown in the OpenAPI docs.
func Description(d string) OperationOption {
	return func(op *operation) { op.spec.description = d }
}

// OperationID sets an explicit operationId in the OpenAPI spec.
func OperationID(id string) OperationOption {
	return func(op *operation) { op.spec.operationID = id }
}

// Tags overrides the tags for this specific operation.
func Tags(tags ...string) OperationOption {
	return func(op *operation) { op.spec.tags = tags }
}

// TagDescription records a top-level OpenAPI tag description for this operation.
func TagDescription(tag, description string) OperationOption {
	return func(op *operation) {
		if op.spec.tagDescriptions == nil {
			op.spec.tagDescriptions = map[string]string{}
		}
		op.spec.tagDescriptions[tag] = description
	}
}

// Security adds an OpenAPI security requirement to this operation.
func Security(name string, scopes ...string) OperationOption {
	return func(op *operation) {
		op.spec.security = append(op.spec.security, SecurityRequirement{name: append([]string{}, scopes...)})
	}
}

// SecurityMiddleware adds an OpenAPI security requirement and attaches the
// matching Gin middleware to this operation.
func SecurityMiddleware(name string, mw gin.HandlerFunc, scopes ...string) OperationOption {
	return func(op *operation) {
		Security(name, scopes...)(op)
		appendOperationAuthMiddleware(op, mw)
	}
}

// BearerAuth marks this operation as requiring the default bearerAuth scheme
// and attaches the matching middleware. Use Security("bearerAuth") for
// documentation-only security metadata.
func BearerAuth(mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		requireAuthMiddleware("BearerAuth", mw)
		Security("bearerAuth")(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// BasicAuth marks this operation as requiring the default basicAuth scheme and
// attaches the matching middleware. Use Security("basicAuth") for
// documentation-only security metadata.
func BasicAuth(mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		requireAuthMiddleware("BasicAuth", mw)
		Security("basicAuth")(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// APIKeyAuth marks this operation as requiring the named API key scheme and
// attaches the matching middleware. Use Security(name) for documentation-only
// security metadata.
func APIKeyAuth(name string, mw ...gin.HandlerFunc) OperationOption {
	return func(op *operation) {
		requireAuthMiddleware("APIKeyAuth", mw)
		Security(name)(op)
		appendOperationAuthMiddleware(op, mw...)
	}
}

// OAuth2Auth marks this operation as requiring the default oauth2 scheme.
func OAuth2Auth(scopes ...string) OperationOption {
	return Security("oauth2", scopes...)
}

// OAuth2AuthMiddleware applies the default oauth2 OpenAPI security requirement
// and attaches the matching Gin middleware to this operation.
func OAuth2AuthMiddleware(mw gin.HandlerFunc, scopes ...string) OperationOption {
	return SecurityMiddleware("oauth2", mw, scopes...)
}

func appendOperationAuthMiddleware(op *operation, mw ...gin.HandlerFunc) {
	for _, handler := range mw {
		if handler == nil {
			panic("gin-ninja: auth middleware must not be nil")
		}
		op.route.ginMiddleware = append(op.route.ginMiddleware, handler)
	}
}

func requireAuthMiddleware(helper string, mw []gin.HandlerFunc) {
	if len(mw) == 0 {
		panic("gin-ninja: " + helper + " requires auth middleware; use Security/WithSecurity for documentation-only security metadata")
	}
}

// Deprecated marks the operation as deprecated in the docs.
func Deprecated() OperationOption {
	return func(op *operation) { op.spec.deprecated = true }
}

// Cache enables route-level response caching for safe read endpoints.
func Cache(ttl time.Duration, opts ...CacheOption) OperationOption {
	return func(op *operation) {
		op.cache.config = newRouteCacheConfig(ttl)
		for _, opt := range opts {
			opt(op.cache.config)
		}
		if op.cache.control == "" && ttl > 0 {
			op.cache.control = defaultCacheControl(ttl)
		}
		op.cache.etagEnabled = true
	}
}

// CacheControl sets the Cache-Control response header for successful responses.
func CacheControl(value string) OperationOption {
	return func(op *operation) { op.cache.control = value }
}

// ETag enables automatic ETag generation for successful responses.
func ETag() OperationOption {
	return func(op *operation) { op.cache.etagEnabled = true }
}

// ExcludeFromDocs omits the operation from the generated OpenAPI spec.
func ExcludeFromDocs() OperationOption {
	return func(op *operation) { op.spec.excludeFromDocs = true }
}

// SuccessStatus sets the HTTP status code used for successful responses.
// The default is 200 OK (201 Created is common for POST).
func SuccessStatus(code int) OperationOption {
	return func(op *operation) { op.spec.successStatus = code }
}

// Response documents an additional OpenAPI response for the operation.
// Pass model as nil for responses without a JSON response body.
func Response(status int, description string, model any) OperationOption {
	return func(op *operation) {
		var modelType reflect.Type
		if model != nil {
			modelType = reflect.TypeOf(model)
		}
		op.spec.responses = append(op.spec.responses, documentedResponse{
			status:       status,
			description:  description,
			responseType: modelType,
		})
	}
}

// ResponseModel declares the runtime and OpenAPI schema for the successful
// JSON response. When T embeds ModelSchema, returned model values are bound to
// T before being written so fields/exclude tags are enforced at runtime.
func ResponseModel[T any]() OperationOption {
	return func(op *operation) {
		op.spec.responseType = reflect.TypeOf((*T)(nil)).Elem()
		op.spec.responseSchema = nil
		op.spec.paginatedResponseSchema = nil
		op.spec.cursorPaginatedResponseSchema = nil
	}
}

// ResponseSchema declares a reusable model schema descriptor for the successful
// JSON response. Returned model values are serialized through the descriptor at
// runtime and documented with the same descriptor in OpenAPI.
func ResponseSchema[T any](descriptor ModelSchemaDescriptor[T]) OperationOption {
	return func(op *operation) {
		schema := descriptor.schemaDescriptor()
		op.spec.responseSchema = &schema
		op.spec.responseType = nil
		op.spec.paginatedResponseSchema = nil
		op.spec.cursorPaginatedResponseSchema = nil
	}
}

// Paginated declares a standard paginated success response schema. Items are
// bound and validated with the same runtime rules as ResponseModel.
func Paginated[T any]() OperationOption {
	return func(op *operation) {
		var item T
		op.spec.paginatedItemType = reflect.TypeOf(item)
		op.spec.paginatedResponseSchema = nil
	}
}

// PaginatedSchema declares a paginated success response whose items are
// serialized through the provided model schema descriptor at runtime.
func PaginatedSchema[T any](descriptor ModelSchemaDescriptor[T]) OperationOption {
	return func(op *operation) {
		schema := descriptor.schemaDescriptor()
		op.spec.paginatedResponseSchema = &schema
		op.spec.paginatedItemType = nil
	}
}

// CursorPaginated declares a standard cursor-paginated success response schema.
// Items are bound and validated with the same runtime rules as ResponseModel.
func CursorPaginated[T any]() OperationOption {
	return func(op *operation) {
		var item T
		op.spec.cursorPaginatedItemType = reflect.TypeOf(item)
		op.spec.cursorPaginatedResponseSchema = nil
	}
}

// CursorPaginatedSchema declares a cursor-paginated success response whose
// items are serialized through the provided model schema descriptor at runtime.
func CursorPaginatedSchema[T any](descriptor ModelSchemaDescriptor[T]) OperationOption {
	return func(op *operation) {
		schema := descriptor.schemaDescriptor()
		op.spec.cursorPaginatedResponseSchema = &schema
		op.spec.cursorPaginatedItemType = nil
	}
}

// PaginatedResponse documents an additional paginated OpenAPI response.
func PaginatedResponse[T any](status int, description string) OperationOption {
	return func(op *operation) {
		var item T
		op.spec.responses = append(op.spec.responses, documentedResponse{
			status:            status,
			description:       description,
			paginatedItemType: reflect.TypeOf(item),
		})
	}
}

// CursorPaginatedResponse documents an additional cursor-paginated OpenAPI response.
func CursorPaginatedResponse[T any](status int, description string) OperationOption {
	return func(op *operation) {
		var item T
		op.spec.responses = append(op.spec.responses, documentedResponse{
			status:                  status,
			description:             description,
			cursorPaginatedItemType: reflect.TypeOf(item),
		})
	}
}

// Timeout applies a context-based per-operation cooperative timeout.
// It cancels the request context and allows an early timeout response, but
// long-running handlers must observe ctx.Done() or ctx.Request.Context().Done()
// to stop promptly.
func Timeout(d time.Duration) OperationOption {
	return func(op *operation) { op.behavior.timeout = d }
}

// RateLimit applies a per-operation in-memory token-bucket rate limit.
func RateLimit(requestsPerSecond int, burst ...int) OperationOption {
	return func(op *operation) {
		if requestsPerSecond <= 0 {
			op.behavior.rateLimit = nil
			return
		}
		b := requestsPerSecond
		if len(burst) > 0 && burst[0] > 0 {
			b = burst[0]
		}
		op.behavior.rateLimit = newRateLimiter(float64(requestsPerSecond), float64(b))
	}
}

type documentedResponse struct {
	status                  int
	description             string
	responseType            reflect.Type
	paginatedItemType       reflect.Type
	cursorPaginatedItemType reflect.Type
}

type operationRoute struct {
	method        string
	path          string
	ginHandler    gin.HandlerFunc
	ginMiddleware []gin.HandlerFunc
	inputType     reflect.Type
	outputType    reflect.Type
}

type operationDocSpec struct {
	summary                       string
	description                   string
	operationID                   string
	tags                          []string
	tagDescriptions               map[string]string
	security                      []SecurityRequirement
	deprecated                    bool
	successStatus                 int
	responseType                  reflect.Type
	responseSchema                *modelSchemaDescriptor
	paginatedResponseSchema       *modelSchemaDescriptor
	cursorPaginatedResponseSchema *modelSchemaDescriptor
	responses                     []documentedResponse
	paginatedItemType             reflect.Type
	cursorPaginatedItemType       reflect.Type
	excludeFromDocs               bool
}

type operationBehavior struct {
	timeout         time.Duration
	rateLimit       *rateLimiter
	withTransaction bool
}

type operationCache struct {
	config      *routeCacheConfig
	control     string
	etagEnabled bool
}

type operationVersion struct {
	name string
	info *VersionConfig
}

type operationStream struct {
	config *streamConfig
}

// operation holds all metadata about a single API endpoint and the
// gin-compatible handler function that wraps the user-supplied typed handler.
type operation struct {
	route    operationRoute
	spec     operationDocSpec
	behavior operationBehavior
	cache    operationCache
	version  operationVersion
	stream   operationStream
}

// WithTransaction wraps the operation in a request-scoped database transaction.
func WithTransaction() OperationOption {
	return func(op *operation) { op.behavior.withTransaction = true }
}

func cloneSecurityRequirements(reqs []SecurityRequirement) []SecurityRequirement {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]SecurityRequirement, 0, len(reqs))
	for _, req := range reqs {
		cloned := make(SecurityRequirement, len(req))
		for name, scopes := range req {
			cloned[name] = append([]string{}, scopes...)
		}
		out = append(out, cloned)
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneVersionInfo(info *VersionConfig) *VersionConfig {
	if info == nil {
		return nil
	}
	cloned := *info
	return &cloned
}

// newOperation builds an operation and wraps the typed handler with
// parameter binding, error handling, and response serialization.
func newOperation[TIn any, TOut any](
	method, path string,
	handler func(ctx *Context, input *TIn) (*TOut, error),
	defaultTags []string,
) *operation {
	var zeroIn TIn
	var zeroOut TOut
	inputType := reflect.TypeOf(zeroIn)
	outputType := reflect.TypeOf(zeroOut)

	op := &operation{
		route: operationRoute{
			method:     method,
			path:       path,
			inputType:  inputType,
			outputType: outputType,
		},
		spec: operationDocSpec{
			tags:          append([]string(nil), defaultTags...),
			successStatus: http.StatusOK,
		},
	}

	op.route.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)

		// Allocate and populate the typed input.
		input := new(TIn)
		if err := bindInput(c, method, input); err != nil {
			WriteError(c, err)
			return
		}

		// Invoke the user handler.
		var (
			output *TOut
			err    error
		)
		invoke := func() error {
			output, err = handler(ctx, input)
			return err
		}
		if op.behavior.withTransaction {
			api, _ := currentAPI(c)
			_, _, _, withTransaction := apiTransactionHandlers(api)
			if withTransaction == nil {
				err = errTransactionUnavailable()
			} else {
				err = withTransaction(c, func() error {
					return invokeWithContextGuard(c, invoke)
				})
			}
		} else {
			err = invoke()
		}
		if err != nil {
			WriteError(c, err)
			return
		}

		op.writeSuccessResponse(c, output)
	}

	return op
}

// newVoidOperation builds an operation whose handler returns no typed body.
// Useful for DELETE endpoints that return 204 No Content.
func newVoidOperation[TIn any](
	method, path string,
	handler func(ctx *Context, input *TIn) error,
	defaultTags []string,
) *operation {
	var zeroIn TIn
	inputType := reflect.TypeOf(zeroIn)

	op := &operation{
		route: operationRoute{
			method:    method,
			path:      path,
			inputType: inputType,
		},
		spec: operationDocSpec{
			tags:          append([]string(nil), defaultTags...),
			successStatus: http.StatusNoContent,
		},
	}

	op.route.ginHandler = func(c *gin.Context) {
		ctx := newContext(c)

		input := new(TIn)
		if err := bindInput(c, method, input); err != nil {
			WriteError(c, err)
			return
		}

		invoke := func() error {
			return handler(ctx, input)
		}
		var err error
		if op.behavior.withTransaction {
			api, _ := currentAPI(c)
			_, _, _, withTransaction := apiTransactionHandlers(api)
			if withTransaction == nil {
				err = errTransactionUnavailable()
			} else {
				err = withTransaction(c, func() error {
					return invokeWithContextGuard(c, invoke)
				})
			}
		} else {
			err = invoke()
		}
		if err != nil {
			WriteError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}

	return op
}

func (op *operation) finalize() {
	if op.route.ginHandler == nil {
		return
	}

	handler := op.route.ginHandler
	if op.behavior.timeout > 0 {
		if op.usesDirectResponseWriter() {
			handler = wrapCooperativeTimeout(op.behavior.timeout, handler)
		} else {
			handler = wrapTimeout(op.behavior.timeout, handler)
		}
	}
	if op.cache.config != nil || op.cache.control != "" || op.cache.etagEnabled {
		handler = wrapCache(op, handler)
	}
	if op.behavior.rateLimit != nil {
		handler = wrapRateLimit(op.behavior.rateLimit, handler)
	}
	op.route.ginHandler = handler
}

func (op *operation) usesDirectResponseWriter() bool {
	if op.stream.config != nil {
		return true
	}
	if op.route.outputType == nil {
		return false
	}
	responseWriterType := reflect.TypeOf((*responseWriter)(nil)).Elem()
	if reflect.PointerTo(op.route.outputType).Implements(responseWriterType) {
		return true
	}
	return op.route.outputType.Implements(responseWriterType)
}

func (op *operation) successResponseType() reflect.Type {
	if op.spec.responseType != nil {
		return op.spec.responseType
	}
	return op.route.outputType
}

func (op *operation) writeSuccessResponse(c *gin.Context, output any) {
	if isNilResponse(output) {
		c.Status(http.StatusNoContent)
		return
	}
	if writer, ok := output.(responseWriter); ok {
		writer.writeTo(c, op.spec.successStatus)
		return
	}
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Status(op.spec.successStatus)
		return
	}

	body, err := op.serializeResponse(output)
	if err != nil {
		WriteError(c, err)
		return
	}
	c.JSON(op.spec.successStatus, body)
}

func isNilResponse(output any) bool {
	if output == nil {
		return true
	}
	value := reflect.ValueOf(output)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (op *operation) serializeResponse(output any) (any, error) {
	if op.spec.paginatedResponseSchema != nil {
		return serializePaginatedModelSchemaResponse(*op.spec.paginatedResponseSchema, output)
	}
	if op.spec.cursorPaginatedResponseSchema != nil {
		return serializePaginatedModelSchemaResponse(*op.spec.cursorPaginatedResponseSchema, output)
	}
	if op.spec.paginatedItemType != nil {
		return serializePaginatedResponseModel(op.spec.paginatedItemType, output)
	}
	if op.spec.cursorPaginatedItemType != nil {
		return serializePaginatedResponseModel(op.spec.cursorPaginatedItemType, output)
	}
	if op.spec.responseSchema != nil {
		return serializeModelSchemaResponse(*op.spec.responseSchema, output)
	}
	if op.spec.responseType == nil {
		return output, nil
	}
	body, err := bindResponseModel(op.spec.responseType, output)
	if err != nil {
		return nil, err
	}
	if err := validateResponseModel(body); err != nil {
		return nil, err
	}
	return body, nil
}

func serializePaginatedModelSchemaResponse(descriptor modelSchemaDescriptor, output any) (any, error) {
	value := reflect.ValueOf(output)
	if !value.IsValid() {
		return nil, fmt.Errorf("response value is invalid")
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, fmt.Errorf("response value is nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("paginated response must be a struct, got %s", value.Kind())
	}

	items := value.FieldByName("Items")
	if !items.IsValid() {
		return nil, fmt.Errorf("paginated response must have an Items field")
	}
	if !items.CanInterface() {
		return nil, fmt.Errorf("paginated response Items field is not accessible")
	}
	if err := validateModelSchemaResponseType(descriptor.target, items.Interface()); err != nil {
		return nil, err
	}
	if err := validateModelSchemaRequiredFields(items, descriptor.filter); err != nil {
		return nil, err
	}
	serializedItems, err := serializeModelSchemaValue(items, descriptor.filter)
	if err != nil {
		return nil, err
	}

	return serializePaginatedResponseEnvelope(value, serializedItems)
}

func serializePaginatedResponseModel(itemType reflect.Type, output any) (any, error) {
	value := reflect.ValueOf(output)
	if !value.IsValid() {
		return nil, fmt.Errorf("response value is invalid")
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, fmt.Errorf("response value is nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("paginated response must be a struct, got %s", value.Kind())
	}

	items := value.FieldByName("Items")
	if !items.IsValid() {
		return nil, fmt.Errorf("paginated response must have an Items field")
	}
	if !items.CanInterface() {
		return nil, fmt.Errorf("paginated response Items field is not accessible")
	}

	serializedItems, err := bindResponseModelItems(reflect.SliceOf(itemType), items.Interface())
	if err != nil {
		return nil, err
	}
	if err := validateResponseModel(serializedItems); err != nil {
		return nil, err
	}
	return serializePaginatedResponseEnvelope(value, serializedItems)
}

func serializePaginatedResponseEnvelope(value reflect.Value, serializedItems any) (map[string]any, error) {
	out := make(map[string]any)
	t := value.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		name := jsonFieldName(field)
		if name == "-" {
			continue
		}
		fieldValue := value.Field(i)
		if isJSONOmitEmpty(field) && fieldValue.IsZero() {
			continue
		}
		if field.Name == "Items" {
			out[name] = serializedItems
			continue
		}
		if marshaled, ok := preserveCustomJSONValue(fieldValue); ok {
			out[name] = marshaled
			continue
		}
		if !fieldValue.CanInterface() {
			continue
		}
		out[name] = fieldValue.Interface()
	}
	return out, nil
}

func serializeModelSchemaResponse(descriptor modelSchemaDescriptor, output any) (any, error) {
	if err := validateModelSchemaResponseType(descriptor.target, output); err != nil {
		return nil, err
	}
	if err := validateModelSchemaRequiredFields(reflect.ValueOf(output), descriptor.filter); err != nil {
		return nil, err
	}
	return serializeModelSchemaValue(reflect.ValueOf(output), descriptor.filter)
}

func validateModelSchemaRequiredFields(value reflect.Value, filter modelSchemaFilter) error {
	return validateModelSchemaRequiredFieldsAt(value, filter.normalized(), "")
}

func validateModelSchemaRequiredFieldsAt(value reflect.Value, filter modelSchemaFilter, path string) error {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if marshaled, ok := preserveCustomJSONValue(value); ok && marshaled != nil {
		return nil
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := validateModelSchemaRequiredFieldsAt(value.Index(i), filter, path); err != nil {
				return err
			}
		}
	case reflect.Struct:
		return validateModelSchemaStructRequiredFields(value, filter, path)
	}
	return nil
}

func validateModelSchemaStructRequiredFields(value reflect.Value, filter modelSchemaFilter, path string) error {
	t := value.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldValue := value.Field(i)
		if field.Anonymous {
			if err := validateModelSchemaRequiredFieldsAt(fieldValue, filter, path); err != nil {
				return err
			}
			continue
		}

		name := jsonFieldName(field)
		if name == "-" || !filter.includes(field, name) {
			continue
		}
		fieldPath := modelSchemaFieldPath(path, name)
		if isRequired(field) && fieldValue.IsZero() {
			return fmt.Errorf("response schema validation failed: %s is required", fieldPath)
		}
		if filter.depth > 0 && modelSchemaNestedField(fieldValue.Type()) {
			if err := validateModelSchemaRequiredFieldsAt(fieldValue, filter.child(), fieldPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func modelSchemaFieldPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func validateModelSchemaResponseType(target reflect.Type, output any) error {
	source := reflect.TypeOf(output)
	if source == nil {
		return fmt.Errorf("response value is invalid")
	}
	if isModelSchemaResponseType(target, source) {
		return nil
	}
	return fmt.Errorf("cannot serialize response type %s as model schema %s", source, target)
}

func isModelSchemaResponseType(target, source reflect.Type) bool {
	if target == nil || source == nil {
		return false
	}
	if source.AssignableTo(target) || source.ConvertibleTo(target) {
		return true
	}
	for source.Kind() == reflect.Ptr {
		source = source.Elem()
		if source.AssignableTo(target) || source.ConvertibleTo(target) {
			return true
		}
	}

	target = deref(target)
	if source.AssignableTo(target) || source.ConvertibleTo(target) {
		return true
	}
	if source.Kind() == reflect.Slice || source.Kind() == reflect.Array {
		return isModelSchemaResponseType(target, source.Elem())
	}
	return false
}

func bindResponseModel(schemaType reflect.Type, output any) (any, error) {
	schemaType = deref(schemaType)
	if value, ok := coerceResponseValue(schemaType, output); ok {
		return value, nil
	}

	if schemaType.Kind() == reflect.Slice || schemaType.Kind() == reflect.Array {
		return bindResponseModelItems(schemaType, output)
	}
	if schemaType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("response model must be a struct, got %s", schemaType.Kind())
	}
	if value, ok, err := bindModelSchemaForType(schemaType, output); ok || err != nil {
		return value, err
	}

	source := reflect.ValueOf(output)
	if !source.IsValid() {
		return nil, fmt.Errorf("response value is invalid")
	}
	return nil, fmt.Errorf("cannot serialize response type %s as %s", source.Type(), schemaType)
}

func coerceResponseValue(schemaType reflect.Type, output any) (any, bool) {
	source := reflect.ValueOf(output)
	if !source.IsValid() {
		return nil, false
	}
	if source.Type().AssignableTo(schemaType) {
		return source.Interface(), true
	}
	if source.Type().ConvertibleTo(schemaType) {
		return source.Convert(schemaType).Interface(), true
	}
	if source.Kind() == reflect.Ptr && !source.IsNil() {
		elem := source.Elem()
		if elem.Type().AssignableTo(schemaType) {
			return elem.Interface(), true
		}
		if elem.Type().ConvertibleTo(schemaType) {
			return elem.Convert(schemaType).Interface(), true
		}
	}
	return nil, false
}

func bindResponseModelItems(schemaType reflect.Type, output any) (any, error) {
	source := reflect.ValueOf(output)
	if !source.IsValid() {
		return nil, fmt.Errorf("response value is invalid")
	}
	for source.Kind() == reflect.Ptr {
		if source.IsNil() {
			return nil, fmt.Errorf("response value is nil")
		}
		source = source.Elem()
	}
	if source.Kind() != reflect.Slice && source.Kind() != reflect.Array {
		return nil, fmt.Errorf("cannot serialize response type %s as %s", source.Type(), schemaType)
	}
	if schemaType.Kind() == reflect.Array && source.Len() != schemaType.Len() {
		return nil, fmt.Errorf("cannot serialize response length %d as %s", source.Len(), schemaType)
	}

	out := reflect.MakeSlice(reflect.SliceOf(schemaType.Elem()), source.Len(), source.Len())
	for i := 0; i < source.Len(); i++ {
		item, err := bindResponseModel(schemaType.Elem(), source.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("response item %d: %w", i, err)
		}
		itemValue, err := responseModelValueForType(schemaType.Elem(), item)
		if err != nil {
			return nil, fmt.Errorf("response item %d: %w", i, err)
		}
		out.Index(i).Set(itemValue)
	}
	if schemaType.Kind() == reflect.Slice {
		return out.Interface(), nil
	}

	array := reflect.New(schemaType).Elem()
	reflect.Copy(array, out)
	return array.Interface(), nil
}

func responseModelValueForType(target reflect.Type, value any) (reflect.Value, error) {
	source := reflect.ValueOf(value)
	if !source.IsValid() {
		return reflect.Value{}, fmt.Errorf("response value is invalid")
	}
	if source.Type().AssignableTo(target) {
		return source, nil
	}
	if source.Type().ConvertibleTo(target) {
		return source.Convert(target), nil
	}
	if source.Kind() == reflect.Ptr && !source.IsNil() {
		elem := source.Elem()
		if elem.Type().AssignableTo(target) {
			return elem, nil
		}
		if elem.Type().ConvertibleTo(target) {
			return elem.Convert(target), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("cannot serialize response type %s as %s", source.Type(), target)
}

func validateResponseModel(value any) error {
	if value == nil {
		return nil
	}
	return validateResponseModelValue(reflect.ValueOf(value))
}

func validateResponseModelValue(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if !value.CanInterface() {
			return nil
		}
		if err := validate.Struct(value.Interface()); err != nil {
			return fmt.Errorf("response schema validation failed: %w", err)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := validateResponseModelValue(value.Index(i)); err != nil {
				return err
			}
		}
	default:
		return nil
	}
	return nil
}

func invokeWithContextGuard(c *gin.Context, invoke func() error) error {
	if err := invoke(); err != nil {
		return err
	}
	if err := c.Request.Context().Err(); err != nil {
		return err
	}
	return nil
}
