package ninja

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// OpenAPI 3.0 spec types
// ---------------------------------------------------------------------------

// openAPISpec is the root OpenAPI 3.0 document.
type openAPISpec struct {
	OpenAPI    string               `json:"openapi"`
	Info       openAPIInfo          `json:"info"`
	Paths      map[string]*pathItem `json:"paths"`
	Components openAPIComponents    `json:"components"`
	Tags       []openAPITag         `json:"tags,omitempty"`

	// Internal state – not serialised.
	config          Config
	registry        *schemaRegistry
	tagDescriptions map[string]string
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type openAPIComponents struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// pathItem holds the operations for a single URL path.
type pathItem struct {
	Get    *operationSpec `json:"get,omitempty"`
	Post   *operationSpec `json:"post,omitempty"`
	Put    *operationSpec `json:"put,omitempty"`
	Patch  *operationSpec `json:"patch,omitempty"`
	Delete *operationSpec `json:"delete,omitempty"`
}

// operationSpec is the OpenAPI representation of a single operation.
type operationSpec struct {
	OperationID string                  `json:"operationId,omitempty"`
	Summary     string                  `json:"summary,omitempty"`
	Description string                  `json:"description,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Security    []SecurityRequirement   `json:"security,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty"`
	Parameters  []parameterSpec         `json:"parameters,omitempty"`
	RequestBody *requestBodySpec        `json:"requestBody,omitempty"`
	Responses   map[string]responseSpec `json:"responses"`
}

type parameterSpec struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // path | query | header | cookie
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

type requestBodySpec struct {
	Description string                   `json:"description,omitempty"`
	Required    bool                     `json:"required"`
	Content     map[string]mediaTypeSpec `json:"content"`
}

type mediaTypeSpec struct {
	Schema *Schema `json:"schema"`
}

type responseSpec struct {
	Description string                   `json:"description"`
	Content     map[string]mediaTypeSpec `json:"content,omitempty"`
}

// ---------------------------------------------------------------------------
// Constructor + build
// ---------------------------------------------------------------------------

func newOpenAPISpec(cfg Config) *openAPISpec {
	return &openAPISpec{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       cfg.Title,
			Version:     cfg.Version,
			Description: cfg.Description,
		},
		Paths: make(map[string]*pathItem),
		Components: openAPIComponents{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: cloneSecuritySchemes(cfg.SecuritySchemes),
		},
		config:          cfg,
		registry:        newSchemaRegistry(),
		tagDescriptions: map[string]string{},
	}
}

// build returns the final spec ready for JSON serialisation.
func (s *openAPISpec) build() *openAPISpec {
	built := *s
	built.Paths = make(map[string]*pathItem, len(s.Paths))
	for path, item := range s.Paths {
		built.Paths[path] = item
	}
	built.Components = openAPIComponents{
		Schemas:         make(map[string]*Schema, len(s.registry.schemas)),
		SecuritySchemes: cloneSecuritySchemes(s.Components.SecuritySchemes),
	}
	for name, schema := range s.registry.schemas {
		built.Components.Schemas[name] = schema
	}
	if len(s.tagDescriptions) > 0 {
		names := make([]string, 0, len(s.tagDescriptions))
		for name := range s.tagDescriptions {
			names = append(names, name)
		}
		sort.Strings(names)
		built.Tags = make([]openAPITag, 0, len(names))
		for _, name := range names {
			built.Tags = append(built.Tags, openAPITag{
				Name:        name,
				Description: s.tagDescriptions[name],
			})
		}
	}
	return &built
}

// addOperation registers an operation in the spec.
func (s *openAPISpec) addOperation(op *operation) {
	if op.excludeFromDocs {
		return
	}
	s.registerTags(op.tags, op.tagDescriptions)

	// op.path is already the fully-qualified router path, including any global
	// API prefix applied during router registration.
	openapiPath := ginPathToOpenAPI(op.path)

	item, ok := s.Paths[openapiPath]
	if !ok {
		item = &pathItem{}
		s.Paths[openapiPath] = item
	}

	spec := s.buildOperationSpec(op)
	switch strings.ToUpper(op.method) {
	case http.MethodGet:
		item.Get = spec
	case http.MethodPost:
		item.Post = spec
	case http.MethodPut:
		item.Put = spec
	case http.MethodPatch:
		item.Patch = spec
	case http.MethodDelete:
		item.Delete = spec
	}
}

// buildOperationSpec converts an operation into an operationSpec.
func (s *openAPISpec) buildOperationSpec(op *operation) *operationSpec {
	spec := &operationSpec{
		OperationID: op.operationID,
		Summary:     op.summary,
		Description: op.description,
		Tags:        op.tags,
		Security:    cloneSecurityRequirements(op.security),
		Deprecated:  op.deprecated,
		Responses:   make(map[string]responseSpec),
	}

	// Parameters (path, query, header) from the input type.
	if op.inputType != nil {
		inputType := deref(op.inputType)
		if inputType.Kind() == reflect.Struct {
			params, bodySchema := s.extractParams(op.method, inputType)
			spec.Parameters = params
			if bodySchema != nil {
				spec.RequestBody = &requestBodySpec{
					Required: true,
					Content: map[string]mediaTypeSpec{
						"application/json": {Schema: bodySchema},
					},
				}
			}
		}
	}

	// Success response.
	successCode := fmt.Sprintf("%d", op.successStatus)
	if op.paginatedItemType != nil {
		spec.Responses[successCode] = responseSpec{
			Description: http.StatusText(op.successStatus),
			Content: map[string]mediaTypeSpec{
				"application/json": {Schema: paginatedSchema(s.registry.schemaForType(op.paginatedItemType))},
			},
		}
	} else if op.outputType != nil {
		schema := s.registry.schemaForType(op.outputType)
		spec.Responses[successCode] = responseSpec{
			Description: http.StatusText(op.successStatus),
			Content: map[string]mediaTypeSpec{
				"application/json": {Schema: schema},
			},
		}
	} else {
		spec.Responses[successCode] = responseSpec{
			Description: http.StatusText(op.successStatus),
		}
	}

	// Standard error responses.
	spec.Responses["422"] = responseSpec{Description: "Validation Error"}
	spec.Responses["500"] = responseSpec{Description: "Internal Server Error"}
	if op.timeout > 0 {
		spec.Responses["408"] = responseSpec{Description: http.StatusText(http.StatusRequestTimeout)}
	}
	if op.rateLimit != nil {
		spec.Responses["429"] = responseSpec{Description: http.StatusText(http.StatusTooManyRequests)}
	}

	for _, documented := range op.responses {
		response := responseSpec{
			Description: documented.description,
		}
		if response.Description == "" {
			response.Description = http.StatusText(documented.status)
		}
		if documented.paginatedItemType != nil {
			response.Content = map[string]mediaTypeSpec{
				"application/json": {Schema: paginatedSchema(s.registry.schemaForType(documented.paginatedItemType))},
			}
		} else if documented.responseType != nil {
			response.Content = map[string]mediaTypeSpec{
				"application/json": {Schema: s.registry.schemaForType(documented.responseType)},
			}
		}
		spec.Responses[fmt.Sprintf("%d", documented.status)] = response
	}

	return spec
}

// extractParams inspects the input struct and returns parameter specs plus
// an optional request-body schema for body methods.
func (s *openAPISpec) extractParams(method string, t reflect.Type) ([]parameterSpec, *Schema) {
	var params []parameterSpec
	bodyFields := make(map[string]*Schema)
	bodyRequired := []string{}
	hasBody := isBodyMethod(method)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			// Flatten embedded structs.
			ep, embeddedBodySchema := s.extractParams(method, deref(f.Type))
			params = append(params, ep...)
			if hasBody && embeddedBodySchema != nil {
				for name, schema := range embeddedBodySchema.Properties {
					bodyFields[name] = schema
				}
				bodyRequired = append(bodyRequired, embeddedBodySchema.Required...)
			}
			continue
		}

		fieldSchema := annotateSchema(s.registry.schemaForType(f.Type), f)

		// Path parameter.
		if pathTag := f.Tag.Get("path"); pathTag != "" {
			params = append(params, parameterSpec{
				Name:     pathTag,
				In:       "path",
				Required: true,
				Schema:   fieldSchema,
			})
			continue
		}

		// Header parameter.
		if headerTag := f.Tag.Get("header"); headerTag != "" {
			params = append(params, parameterSpec{
				Name:        headerTag,
				In:          "header",
				Required:    isRequired(f),
				Description: f.Tag.Get("description"),
				Schema:      fieldSchema,
			})
			continue
		}

		// Cookie parameter.
		if cookieTag := f.Tag.Get("cookie"); cookieTag != "" {
			params = append(params, parameterSpec{
				Name:        cookieTag,
				In:          "cookie",
				Required:    isRequired(f),
				Description: f.Tag.Get("description"),
				Schema:      fieldSchema,
			})
			continue
		}

		// Query / form parameter.
		if formTag := f.Tag.Get("form"); formTag != "" {
			params = append(params, parameterSpec{
				Name:        formTag,
				In:          "query",
				Required:    isRequired(f),
				Description: f.Tag.Get("description"),
				Schema:      fieldSchema,
			})
			continue
		}

		// For body methods, any remaining fields with a json tag go into the
		// request body schema.
		if hasBody {
			fieldName := jsonFieldName(f)
			if fieldName == "-" {
				continue
			}
			bodyFields[fieldName] = annotateSchema(fieldSchema, f)
			if isRequired(f) {
				bodyRequired = append(bodyRequired, fieldName)
			}
		}
	}

	var bodySchema *Schema
	if hasBody && len(bodyFields) > 0 {
		bodySchema = &Schema{
			Type:       "object",
			Properties: bodyFields,
			Required:   bodyRequired,
		}
	}

	return params, bodySchema
}

func (s *openAPISpec) registerTags(tags []string, descriptions map[string]string) {
	for _, tag := range tags {
		if _, ok := s.tagDescriptions[tag]; !ok {
			s.tagDescriptions[tag] = ""
		}
		if desc := descriptions[tag]; desc != "" {
			s.tagDescriptions[tag] = desc
		}
	}
}

// ---------------------------------------------------------------------------
// Swagger UI HTML
// ---------------------------------------------------------------------------

func swaggerUIHTML(openapiURL, title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%s - API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" >
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"> </script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"> </script>
<script>
window.onload = function() {
  const ui = SwaggerUIBundle({
    url: "%s",
    dom_id: '#swagger-ui',
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout"
  })
  window.ui = ui
}
</script>
</body>
</html>`, title, openapiURL)
}

// ginPathToOpenAPI converts a gin-style path ("/users/:id") to an OpenAPI
// path ("/users/{id}").
func ginPathToOpenAPI(ginPath string) string {
	parts := strings.Split(ginPath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func cloneSecuritySchemes(in map[string]SecurityScheme) map[string]SecurityScheme {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]SecurityScheme, len(in))
	for name, scheme := range in {
		out[name] = scheme
	}
	return out
}
