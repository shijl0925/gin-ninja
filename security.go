package ninja

// SecurityScheme describes an OpenAPI reusable security scheme.
type SecurityScheme struct {
	Type         string      `json:"type"`
	Description  string      `json:"description,omitempty"`
	Name         string      `json:"name,omitempty"`
	In           string      `json:"in,omitempty"`
	Scheme       string      `json:"scheme,omitempty"`
	BearerFormat string      `json:"bearerFormat,omitempty"`
	Flows        *OAuthFlows `json:"flows,omitempty"`
}

// SecurityRequirement maps an OpenAPI security scheme name to required scopes.
type SecurityRequirement map[string][]string

// OAuthFlows describes the OAuth2 flows supported by an OpenAPI security scheme.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow describes one OAuth2 authorization flow in OpenAPI.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

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

// HTTPBasicSecurityScheme returns a standard HTTP Basic auth scheme.
func HTTPBasicSecurityScheme() SecurityScheme {
	return SecurityScheme{
		Type:   "http",
		Scheme: "basic",
	}
}

// APIKeyHeaderSecurityScheme returns an API key scheme read from a header.
func APIKeyHeaderSecurityScheme(name string) SecurityScheme {
	return SecurityScheme{
		Type: "apiKey",
		Name: name,
		In:   "header",
	}
}

// APIKeyCookieSecurityScheme returns an API key scheme read from a cookie.
func APIKeyCookieSecurityScheme(name string) SecurityScheme {
	return SecurityScheme{
		Type: "apiKey",
		Name: name,
		In:   "cookie",
	}
}

// APIKeyQuerySecurityScheme returns an API key scheme read from a query parameter.
func APIKeyQuerySecurityScheme(name string) SecurityScheme {
	return SecurityScheme{
		Type: "apiKey",
		Name: name,
		In:   "query",
	}
}

// OAuth2SecurityScheme returns an OAuth2 OpenAPI security scheme.
func OAuth2SecurityScheme(flows OAuthFlows) SecurityScheme {
	return SecurityScheme{
		Type:  "oauth2",
		Flows: &flows,
	}
}
