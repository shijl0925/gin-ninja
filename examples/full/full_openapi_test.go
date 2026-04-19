package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestFullExampleOpenAPIContracts(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	openAPIResp, err := http.Get(server.URL + "/openapi.json")
	if err != nil {
		t.Fatalf("GET /openapi.json: %v", err)
	}
	defer openAPIResp.Body.Close()
	if openAPIResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /openapi.json 200, got %d", openAPIResp.StatusCode)
	}

	var spec map[string]any
	if err := json.NewDecoder(openAPIResp.Body).Decode(&spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}

	components := spec["components"].(map[string]any)
	securitySchemes := components["securitySchemes"].(map[string]any)
	bearerAuth := securitySchemes["bearerAuth"].(map[string]any)
	if bearerAuth["type"] != "http" || bearerAuth["scheme"] != "bearer" || bearerAuth["bearerFormat"] != "JWT" {
		t.Fatalf("unexpected bearer auth scheme: %+v", bearerAuth)
	}

	paths := spec["paths"].(map[string]any)
	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/users/",
		"/api/v1/admin/resources",
		"/api/v1/admin/resources/users",
		"/api/v1/examples/request-meta",
		"/api/v0/examples/versioned/info",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected path %s in root spec, got keys=%v", path, paths)
		}
	}

	usersGet := paths["/api/v1/users/"].(map[string]any)["get"].(map[string]any)
	security := usersGet["security"].([]any)
	if len(security) != 1 {
		t.Fatalf("expected one security requirement, got %v", security)
	}
	if _, ok := security[0].(map[string]any)["bearerAuth"]; !ok {
		t.Fatalf("expected bearerAuth security requirement, got %v", security[0])
	}

	for _, tc := range []struct {
		path        string
		wantPath    string
		missingPath string
	}{
		{path: "/openapi/v1.json", wantPath: "/api/v1/users/", missingPath: "/api/v0/examples/versioned/info"},
		{path: "/openapi/v0.json", wantPath: "/api/v0/examples/versioned/info", missingPath: "/api/v1/users/"},
	} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, resp.StatusCode)
		}
		var versionedSpec map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&versionedSpec); err != nil {
			resp.Body.Close()
			t.Fatalf("decode %s: %v", tc.path, err)
		}
		resp.Body.Close()

		versionedPaths := versionedSpec["paths"].(map[string]any)
		if _, ok := versionedPaths[tc.wantPath]; !ok {
			t.Fatalf("%s: expected path %s, got %v", tc.path, tc.wantPath, versionedPaths)
		}
		if _, ok := versionedPaths[tc.missingPath]; ok {
			t.Fatalf("%s: did not expect path %s, got %v", tc.path, tc.missingPath, versionedPaths)
		}
		if tc.path == "/openapi/v0.json" {
			responses := versionedPaths["/api/v0/examples/versioned/info"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)
			headers := responses["200"].(map[string]any)["headers"].(map[string]any)
			for _, name := range []string{"Deprecation", "Sunset", "Link"} {
				if _, ok := headers[name]; !ok {
					t.Fatalf("%s: expected header %s in deprecated version docs, got %v", tc.path, name, headers)
				}
			}
		}
	}
}
