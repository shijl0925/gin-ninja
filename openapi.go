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
	Headers     map[string]headerSpec    `json:"headers,omitempty"`
}

type headerSpec struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
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
			params, bodySchema, requestContentType := s.extractParams(op.method, inputType)
			spec.Parameters = params
			if bodySchema != nil {
				if requestContentType == "" {
					requestContentType = "application/json"
				}
				spec.RequestBody = &requestBodySpec{
					Required: true,
					Content: map[string]mediaTypeSpec{
						requestContentType: {Schema: bodySchema},
					},
				}
			}
		}
	}

	// Success response.
	successCode := fmt.Sprintf("%d", op.successStatus)
	successResponse := responseSpec{
		Description: http.StatusText(op.successStatus),
	}
	if op.stream != nil {
		if op.stream.kind == streamKindSSE {
			successResponse.Content = map[string]mediaTypeSpec{
				"text/event-stream": {Schema: &Schema{Type: "string"}},
			}
			successResponse.Description = "Server-Sent Events stream"
		} else {
			successResponse.Description = http.StatusText(http.StatusSwitchingProtocols)
		}
	} else if op.paginatedItemType != nil {
		successResponse.Content = map[string]mediaTypeSpec{
			"application/json": {Schema: paginatedSchema(s.registry.schemaForType(op.paginatedItemType))},
		}
	} else if op.outputType != nil {
		contentType, schema := s.responseSchemaForType(op.outputType)
		successResponse.Content = map[string]mediaTypeSpec{
			contentType: {Schema: schema},
		}
	}
	successResponse.Headers = s.responseHeadersForOperation(op)
	spec.Responses[successCode] = successResponse
	if op.stream != nil && op.stream.kind == streamKindWebSocket && successCode != "101" {
		spec.Responses["101"] = successResponse
		delete(spec.Responses, successCode)
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
			contentType, schema := s.responseSchemaForType(documented.responseType)
			response.Content = map[string]mediaTypeSpec{
				contentType: {Schema: schema},
			}
		}
		spec.Responses[fmt.Sprintf("%d", documented.status)] = response
	}

	return spec
}

func (s *openAPISpec) responseHeadersForOperation(op *operation) map[string]headerSpec {
	headers := map[string]headerSpec{}
	if op.etagEnabled {
		headers["ETag"] = headerSpec{
			Description: "Entity tag for conditional requests",
			Schema:      &Schema{Type: "string"},
		}
	}
	if op.cache != nil || op.cacheControl != "" {
		headers["Cache-Control"] = headerSpec{
			Description: "Cache policy for the response",
			Schema:      &Schema{Type: "string"},
		}
	}
	if op.versionInfo != nil && op.versionInfo.Deprecated {
		headers["Deprecation"] = headerSpec{
			Description: "Version deprecation signal",
			Schema:      &Schema{Type: "string"},
		}
		if op.versionInfo.Sunset != "" || !op.versionInfo.SunsetTime.IsZero() {
			headers["Sunset"] = headerSpec{
				Description: "Version sunset timestamp",
				Schema:      &Schema{Type: "string"},
			}
		}
		if op.versionInfo.MigrationURL != "" {
			headers["Link"] = headerSpec{
				Description: "Migration guidance for a deprecated version",
				Schema:      &Schema{Type: "string"},
			}
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// extractParams inspects the input struct and returns parameter specs plus
// an optional request-body schema for body methods.
func (s *openAPISpec) extractParams(method string, t reflect.Type) ([]parameterSpec, *Schema, string) {
	var params []parameterSpec
	bodyFields := make(map[string]*Schema)
	bodyRequired := []string{}
	hasBody := isBodyMethod(method)
	isMultipart := hasMultipartBody(t)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			// Flatten embedded structs.
			ep, embeddedBodySchema, _ := s.extractParams(method, deref(f.Type))
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

		if fileTag := f.Tag.Get("file"); fileTag != "" && hasBody {
			bodyFields[fileTag] = fieldSchema
			if isRequired(f) {
				bodyRequired = append(bodyRequired, fileTag)
			}
			continue
		}

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
			if hasBody && isMultipart {
				bodyFields[formTag] = fieldSchema
				if isRequired(f) {
					bodyRequired = append(bodyRequired, formTag)
				}
				continue
			}
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
	contentType := ""
	if hasBody && len(bodyFields) > 0 {
		bodySchema = &Schema{
			Type:       "object",
			Properties: bodyFields,
			Required:   bodyRequired,
		}
		if isMultipart {
			contentType = "multipart/form-data"
		} else {
			contentType = "application/json"
		}
	}

	return params, bodySchema, contentType
}

func (s *openAPISpec) responseSchemaForType(t reflect.Type) (string, *Schema) {
	if isDownloadType(t) {
		return "application/octet-stream", &Schema{Type: "string", Format: "binary"}
	}
	return "application/json", s.registry.schemaForType(t)
}

func hasMultipartBody(t reflect.Type) bool {
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && hasMultipartBody(deref(f.Type)) {
			return true
		}
		if f.Tag.Get("file") != "" {
			return true
		}
	}
	return false
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

// homepageHTML returns the HTML for the Gin Ninja welcome homepage.
// docsURL and adminURL may be empty strings to hide the respective buttons.
func homepageHTML(title, docsURL, adminURL string) string {
	docsButton := ""
	if docsURL != "" {
		docsButton = fmt.Sprintf(`
        <a href="%s" class="btn btn-docs">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
            <polyline points="10 9 9 9 8 9"/>
          </svg>
          API Docs
        </a>`, docsURL)
	}
	adminButton := ""
	if adminURL != "" {
		adminButton = fmt.Sprintf(`
        <a href="%s" class="btn btn-admin">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
          Admin
        </a>`, adminURL)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    :root {
      color-scheme: light;
      --bg: #f7f8fb;
      --bg-grid: rgba(15, 23, 42, 0.045);
      --panel: rgba(255,255,255,0.78);
      --panel-border: rgba(15, 23, 42, 0.08);
      --text: #0a0a0f;
      --muted: #52525b;
      --subtle: #71717a;
      --primary: #111111;
      --primary-hover: #000000;
      --accent: #635bff;
      --accent-soft: rgba(99, 91, 255, 0.10);
      --success: #0f766e;
      --success-soft: rgba(15, 118, 110, 0.09);
      --shadow: 0 18px 50px rgba(15, 23, 42, 0.07);
    }
    body {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 32px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      color: var(--text);
      background:
        linear-gradient(to right, transparent 0, transparent 47px, var(--bg-grid) 48px),
        linear-gradient(to bottom, transparent 0, transparent 47px, var(--bg-grid) 48px),
        radial-gradient(circle at top, rgba(99, 91, 255, 0.12), transparent 30%%),
        radial-gradient(circle at bottom right, rgba(59, 130, 246, 0.07), transparent 28%%),
        var(--bg);
      background-size: 48px 48px, 48px 48px, auto, auto, auto;
    }
    .shell {
      width: 100%%;
      max-width: 980px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--panel-border);
      border-radius: 28px;
      box-shadow: var(--shadow);
      padding: 56px;
      backdrop-filter: blur(20px);
      position: relative;
      overflow: hidden;
    }
    .card::before {
      content: "";
      position: absolute;
      inset: 0 0 auto 0;
      height: 1px;
      background: linear-gradient(90deg, transparent, rgba(255,255,255,0.9), transparent);
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 10px;
      padding: 6px 12px;
      border-radius: 999px;
      background: rgba(255,255,255,0.72);
      border: 1px solid rgba(15, 23, 42, 0.08);
      color: var(--subtle);
      font-size: 0.76rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .eyebrow::before {
      content: "";
      width: 6px;
      height: 6px;
      border-radius: 50%%;
      background: var(--accent);
      box-shadow: 0 0 0 4px rgba(99, 91, 255, 0.10);
    }
    .hero {
      margin-top: 28px;
      display: flex;
      align-items: flex-start;
      gap: 22px;
    }
    .logo-ring {
      width: 68px;
      height: 68px;
      flex-shrink: 0;
      border-radius: 18px;
      background:
        linear-gradient(135deg, rgba(255,255,255,0.98), rgba(244,244,245,0.92)),
        linear-gradient(135deg, rgba(99,91,255,0.08), rgba(59,130,246,0.04));
      border: 1px solid rgba(15, 23, 42, 0.08);
      display: flex;
      align-items: center;
      justify-content: center;
      box-shadow:
        inset 0 1px 0 rgba(255,255,255,0.85),
        0 8px 20px rgba(15, 23, 42, 0.04);
    }
    .logo-svg {
      width: 30px;
      height: 30px;
      fill: var(--accent);
    }
    .hero-copy {
      min-width: 0;
    }
    h1 {
      font-size: clamp(2.25rem, 5vw, 3.5rem);
      line-height: 0.98;
      letter-spacing: -0.055em;
      font-weight: 750;
      color: var(--text);
      max-width: 640px;
    }
    .tagline {
      margin-top: 18px;
      max-width: 640px;
      font-size: 1.04rem;
      line-height: 1.72;
      color: var(--muted);
    }
    .meta-band {
      margin-top: 36px;
      display: grid;
      grid-template-columns: minmax(0, 1.45fr) minmax(280px, 0.85fr);
      border-radius: 24px;
      border: 1px solid rgba(15, 23, 42, 0.08);
      background: rgba(255,255,255,0.66);
      box-shadow: inset 0 1px 0 rgba(255,255,255,0.72);
      overflow: hidden;
    }
    .status-panel,
    .quicklinks-panel {
      padding: 24px 24px 22px;
      min-width: 0;
    }
    .quicklinks-panel {
      display: flex;
      flex-direction: column;
      justify-content: center;
      border-left: 1px solid rgba(15, 23, 42, 0.08);
      background: linear-gradient(180deg, rgba(255,255,255,0.52), rgba(248,250,252,0.78));
    }
    .meta-label {
      font-size: 0.72rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--subtle);
    }
    .status-row {
      display: flex;
      align-items: center;
      gap: 14px;
      margin-top: 16px;
      flex-wrap: wrap;
    }
    .status {
      display: inline-flex;
      align-items: center;
      gap: 10px;
      padding: 0;
      border: 0;
      background: transparent;
      color: var(--success);
      font-size: 1.02rem;
      font-weight: 650;
      letter-spacing: 0.01em;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      padding: 7px 11px;
      border-radius: 999px;
      border: 1px solid rgba(15, 23, 42, 0.08);
      background: rgba(255,255,255,0.72);
      color: var(--muted);
      font-size: 0.8rem;
      font-weight: 600;
      letter-spacing: 0.01em;
    }
    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%%;
      background: currentColor;
      box-shadow: 0 0 0 6px var(--success-soft);
    }
    .meta-copy {
      margin-top: 12px;
      max-width: 46ch;
      color: var(--muted);
      font-size: 0.92rem;
      line-height: 1.65;
    }
    .quicklinks-copy {
      margin-top: 10px;
      max-width: 28ch;
      color: var(--muted);
      font-size: 0.9rem;
      line-height: 1.6;
    }
    .buttons {
      display: flex;
      gap: 12px;
      flex-direction: column;
      margin-top: 18px;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      min-height: 46px;
      padding: 0 18px;
      border-radius: 14px;
      border: 1px solid transparent;
      text-decoration: none;
      font-size: 0.94rem;
      font-weight: 600;
      transition: background-color 0.18s ease, border-color 0.18s ease, color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
      box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
      width: 100%%;
    }
    .btn:hover {
      transform: translateY(-1px);
      box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
    }
    .btn-docs {
      background: var(--primary);
      color: #fff;
    }
    .btn-docs:hover {
      background: var(--primary-hover);
    }
    .btn-admin {
      background: rgba(255,255,255,0.7);
      color: var(--text);
      border-color: rgba(15, 23, 42, 0.10);
    }
    .btn-admin:hover {
      background: rgba(255,255,255,0.92);
      border-color: rgba(15, 23, 42, 0.16);
    }
    .btn-icon {
      width: 18px;
      height: 18px;
      flex-shrink: 0;
    }
    .footer {
      margin-top: 40px;
      padding-top: 24px;
      border-top: 1px solid rgba(15, 23, 42, 0.07);
      color: var(--subtle);
      font-size: 0.86rem;
    }
    .footer a {
      color: var(--text);
      text-decoration: none;
      font-weight: 600;
    }
    .footer a:hover {
      color: var(--accent);
    }
    @media (max-width: 640px) {
      body {
        padding: 18px;
      }
      .card {
        padding: 30px 22px;
        border-radius: 20px;
      }
      .hero {
        flex-direction: column;
      }
      .logo-ring {
        width: 58px;
        height: 58px;
        border-radius: 16px;
      }
      h1 {
        font-size: 2rem;
      }
      .meta-band {
        grid-template-columns: 1fr;
      }
      .quicklinks-panel {
        border-left: 0;
        border-top: 1px solid rgba(15, 23, 42, 0.08);
      }
      .buttons {
        flex-direction: column;
      }
      .btn {
        width: 100%%;
      }
    }
  </style>
</head>
<body>
<main class="shell">
<div class="card">
  <div class="eyebrow">Gin Ninja</div>
  <div class="hero">
    <div class="logo-ring" aria-hidden="true">
      <svg class="logo-svg" viewBox="0 0 64 64" xmlns="http://www.w3.org/2000/svg">
        <path d="M32 4 L40 28 L64 32 L40 36 L32 60 L24 36 L0 32 L24 28 Z"/>
      </svg>
    </div>
    <div class="hero-copy">
      <h1>%s</h1>
      <p class="tagline">A fast, typed REST framework powered by Gin &amp; Go generics</p>
    </div>
  </div>

  <div class="meta-band">
    <section class="status-panel" aria-label="Server status">
      <div class="meta-label">Status</div>
      <div class="status-row">
        <div class="status">
          <span class="status-dot"></span>
          Server is running
        </div>
        <div class="status-pill">Ready for requests</div>
      </div>
      <p class="meta-copy">Ready to serve requests and expose typed API routes with a clean default setup.</p>
    </section>
    <section class="quicklinks-panel" aria-label="Quick links">
      <div class="meta-label">Quick links</div>
      <p class="quicklinks-copy">Open the interactive docs and jump into the default API workspace.</p>
      <div class="buttons">%s
%s
      </div>
    </section>
  </div>
 
  <div class="footer">
    Powered by <a href="https://github.com/shijl0925/gin-ninja" target="_blank" rel="noopener">gin-ninja</a>
  </div>
</div>
</main>
</body>
</html>`, title, title, docsButton, adminButton)
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
