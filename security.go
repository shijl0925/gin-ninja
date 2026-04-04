package ninja

// SecurityScheme describes an OpenAPI reusable security scheme.
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// SecurityRequirement maps an OpenAPI security scheme name to required scopes.
type SecurityRequirement map[string][]string

// HTTPBearerSecurityScheme returns a standard JWT bearer auth scheme.
func HTTPBearerSecurityScheme(bearerFormat string) SecurityScheme {
	scheme := SecurityScheme{
		Type:   "http",
		Scheme: "bearer",
	}
	if bearerFormat != "" {
		scheme.BearerFormat = bearerFormat
	}
	return scheme
}
