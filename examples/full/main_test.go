package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
)

func newFullTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := settings.Config{
		App: settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:" + t.Name() + "?mode=memory&cache=shared",
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	settings.Global.JWT = cfg.JWT

	log := bootstrap.InitLogger(&cfg.Log)
	db, err := initDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}

	return httptest.NewServer(buildAPI(cfg, db, log).Handler())
}

func doFullJSON(t *testing.T, server *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestFullExampleBuildsRoutesAndEndpoints(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	for _, path := range []string{"/docs", "/docs/v1", "/docs/v0", "/openapi.json", "/openapi/v1.json", "/health"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("Decode login: %v", err)
	}
	login.Body.Close()
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	list := doFullJSON(t, server, http.MethodGet, "/api/v1/users?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}
	list.Body.Close()

	adminMeta := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/users/meta", nil, auth.Token)
	if adminMeta.StatusCode != http.StatusOK {
		t.Fatalf("expected admin metadata 200, got %d", adminMeta.StatusCode)
	}
	adminMeta.Body.Close()

	update := doFullJSON(t, server, http.MethodPut, "/api/v1/users/1", map[string]any{
		"name": "Alicia",
		"age":  19,
	}, auth.Token)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("expected update 200, got %d", update.StatusCode)
	}
	update.Body.Close()

	versioned, err := http.Get(server.URL + "/api/v0/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned info: %v", err)
	}
	if versioned.StatusCode != http.StatusOK || versioned.Header.Get("Deprecation") == "" {
		t.Fatalf("expected deprecated version response, got status=%d headers=%v", versioned.StatusCode, versioned.Header)
	}
	versioned.Body.Close()
}

func TestFullExampleSmokeAuthDocsHealthAndVersioning(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	unauthorized := doFullJSON(t, server, http.MethodGet, "/api/v1/users", nil, "")
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized users list to return 401, got %d", unauthorized.StatusCode)
	}
	unauthorized.Body.Close()

	unauthorizedAdmin := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources", nil, "")
	if unauthorizedAdmin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized admin resources to return 401, got %d", unauthorizedAdmin.StatusCode)
	}
	unauthorizedAdmin.Body.Close()

	healthResp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", healthResp.StatusCode)
	}
	var health map[string]string
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("unexpected health payload: %+v", health)
	}

	for _, tc := range []struct {
		path        string
		wantTitle   string
		wantSpecURL string
	}{
		{path: "/docs", wantTitle: "Full Example - API Docs", wantSpecURL: "/openapi.json"},
		{path: "/docs/v1", wantTitle: "Full Example (v1) - API Docs", wantSpecURL: "/openapi/v1.json"},
		{path: "/docs/v0", wantTitle: "Full Example (v0) - API Docs", wantSpecURL: "/openapi/v0.json"},
	} {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read %s body: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.path, resp.StatusCode)
		}
		html := string(body)
		if !strings.Contains(html, tc.wantTitle) || !strings.Contains(html, tc.wantSpecURL) {
			t.Fatalf("%s: unexpected docs body %q", tc.path, html)
		}
	}

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()
	if auth.Token == "" {
		t.Fatal("expected login token")
	}

	list := doFullJSON(t, server, http.MethodGet, "/api/v1/users?sort=-age", nil, auth.Token)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list users 200, got %d", list.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(list.Body).Decode(&page); err != nil {
		t.Fatalf("decode user list: %v", err)
	}
	list.Body.Close()
	if _, ok := page["items"]; !ok {
		t.Fatalf("expected paginated users payload, got %+v", page)
	}

	detail := doFullJSON(t, server, http.MethodGet, "/api/v1/users/1", nil, auth.Token)
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("expected get user 200, got %d", detail.StatusCode)
	}
	detail.Body.Close()

	adminList := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/users?search=alice", nil, auth.Token)
	if adminList.StatusCode != http.StatusOK {
		t.Fatalf("expected admin users list 200, got %d", adminList.StatusCode)
	}
	adminList.Body.Close()

	v1, err := http.Get(server.URL + "/api/v1/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned v1: %v", err)
	}
	if v1.StatusCode != http.StatusOK {
		t.Fatalf("expected v1 versioned info 200, got %d", v1.StatusCode)
	}
	if v1.Header.Get("Deprecation") != "" || v1.Header.Get("Sunset") != "" || v1.Header.Get("Link") != "" {
		t.Fatalf("did not expect deprecation headers on v1, got %v", v1.Header)
	}
	v1.Body.Close()

	v0, err := http.Get(server.URL + "/api/v0/examples/versioned/info")
	if err != nil {
		t.Fatalf("GET versioned v0: %v", err)
	}
	if v0.StatusCode != http.StatusOK {
		t.Fatalf("expected v0 versioned info 200, got %d", v0.StatusCode)
	}
	if v0.Header.Get("Deprecation") == "" || v0.Header.Get("Sunset") == "" || v0.Header.Get("Link") == "" {
		t.Fatalf("expected deprecation headers on v0, got %v", v0.Header)
	}
	v0.Body.Close()

	for _, path := range []string{"/docs/v9", "/openapi/v9.json"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	docsHeadReq, err := http.NewRequest(http.MethodHead, server.URL+"/docs/v1", nil)
	if err != nil {
		t.Fatalf("new HEAD docs request: %v", err)
	}
	docsHeadResp, err := http.DefaultClient.Do(docsHeadReq)
	if err != nil {
		t.Fatalf("HEAD /docs/v1: %v", err)
	}
	if docsHeadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HEAD /docs/v1 to follow current internal-route behavior and return 404, got %d", docsHeadResp.StatusCode)
	}
	docsHeadResp.Body.Close()
}

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

func TestFullExampleAdminPrototypeAndProjectSelectors(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	loginPageResp, err := http.Get(server.URL + "/admin/login")
	if err != nil {
		t.Fatalf("GET /admin/login: %v", err)
	}
	loginPageBody, err := io.ReadAll(loginPageResp.Body)
	loginPageResp.Body.Close()
	if err != nil {
		t.Fatalf("read /admin/login body: %v", err)
	}
	if loginPageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /admin/login 200, got %d", loginPageResp.StatusCode)
	}
	loginHTML := string(loginPageBody)
	if !strings.Contains(loginHTML, "const adminLoginPath = '/admin/login'") {
		t.Fatalf("expected standalone login path in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "A cleaner sign-in for the standalone admin console.") {
		t.Fatalf("expected polished login marketing copy in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "Demo credentials") {
		t.Fatalf("expected demo credentials card in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "document.body.classList.toggle('standalone-login-page', isStandaloneLoginPage())") {
		t.Fatalf("expected standalone login body class toggle in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "[hidden] { display:none !important; }") {
		t.Fatalf("expected hidden css rule in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, ".two-col { display:grid; gap:20px; grid-template-columns:repeat(auto-fit, minmax(240px, 1fr)); }") {
		t.Fatalf("expected narrower two-column login layout in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "els.manualTokenTools.hidden = true;") {
		t.Fatalf("expected signed-out manual token tools to stay hidden in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "window.location.replace(adminPagePath)") {
		t.Fatalf("expected standalone login redirect to /admin in html: %q", loginHTML)
	}

	adminPageResp, err := http.Get(server.URL + "/admin")
	if err != nil {
		t.Fatalf("GET /admin: %v", err)
	}
	adminPageBody, err := io.ReadAll(adminPageResp.Body)
	adminPageResp.Body.Close()
	if err != nil {
		t.Fatalf("read /admin body: %v", err)
	}
	if adminPageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /admin 200, got %d", adminPageResp.StatusCode)
	}
	adminHTML := string(adminPageBody)
	if !strings.Contains(adminHTML, "const adminPagePath = '/admin'") {
		t.Fatalf("expected standalone admin path in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "Admin Workspace") {
		t.Fatalf("expected polished admin workspace header in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"workspace-path muted\"") {
		t.Fatalf("expected compact workspace summary markup in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.resourcePath.textContent = 'Browse, inspect, and edit ' + state.meta.label + '.';") {
		t.Fatalf("expected shorter admin workspace summary copy in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "Refresh workspace") {
		t.Fatalf("expected admin workspace toolbar action in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.loginForm.hidden = false;") {
		t.Fatalf("expected signed-out admin page to keep login form visible in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.adminShell.hidden = true;") {
		t.Fatalf("expected signed-out admin page to keep admin shell hidden in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "@media (max-width: 1380px)") {
		t.Fatalf("expected earlier admin-shell responsive collapse in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"openCreateModal\"") {
		t.Fatalf("expected create modal trigger in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "id=\"openBulkEditModal\"") {
		t.Fatalf("expected bulk edit trigger to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"modal-overlay\" hidden") {
		t.Fatalf("expected hidden modal overlay shell in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "id=\"authBadge\"") {
		t.Fatalf("expected top-right auth status card to be removed from html: %q", adminHTML)
	}

	prototypeResp, err := http.Get(server.URL + "/admin-prototype")
	if err != nil {
		t.Fatalf("GET /admin-prototype: %v", err)
	}
	body, err := io.ReadAll(prototypeResp.Body)
	prototypeResp.Body.Close()
	if err != nil {
		t.Fatalf("read /admin-prototype body: %v", err)
	}
	if prototypeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected /admin-prototype 200, got %d", prototypeResp.StatusCode)
	}
	html := string(body)
	if !strings.Contains(html, "Gin Ninja Admin") {
		t.Fatalf("expected prototype title in html: %q", html)
	}
	if !strings.Contains(html, "const apiBase = '/api/v1/admin'") {
		t.Fatalf("expected admin api base in html: %q", html)
	}
	if !strings.Contains(html, "const prototypePagePath = '/admin-prototype'") {
		t.Fatalf("expected prototype page path in html: %q", html)
	}
	if !strings.Contains(html, "selectRecord(row)") {
		t.Fatalf("expected record selection flow in html: %q", html)
	}
	if !strings.Contains(html, "renderFilterControls()") {
		t.Fatalf("expected filter controls in html: %q", html)
	}
	if !strings.Contains(html, "scheduleRelationSearch(") {
		t.Fatalf("expected relation search flow in html: %q", html)
	}
	if !strings.Contains(html, "localStorage.setItem(tokenStorageKey, token)") {
		t.Fatalf("expected token persistence in html: %q", html)
	}
	if !strings.Contains(html, "id=\"loginForm\"") {
		t.Fatalf("expected login form in html: %q", html)
	}
	if !strings.Contains(html, "els.loginForm.onsubmit") {
		t.Fatalf("expected login submit flow in html: %q", html)
	}
	if !strings.Contains(html, "skipAuthRedirect: true") {
		t.Fatalf("expected login request auth redirect bypass in html: %q", html)
	}
	if !strings.Contains(html, "els.sessionShell.hidden = true;") {
		t.Fatalf("expected signed-in pages to hide the login shell in html: %q", html)
	}
	if !strings.Contains(html, "body.modal-open { overflow:hidden; }") {
		t.Fatalf("expected modal body lock styling in html: %q", html)
	}
	if !strings.Contains(html, "closeModal(els.createModal);") {
		t.Fatalf("expected create modal close flow in html: %q", html)
	}
	if strings.Contains(html, "closeModal(els.bulkEditModal);") {
		t.Fatalf("expected bulk edit modal close flow to be removed from html: %q", html)
	}
	if !strings.Contains(html, "document.addEventListener('keydown'") {
		t.Fatalf("expected modal escape-key handling in html: %q", html)
	}
	if !strings.Contains(html, "Session expired. Please sign in again.") {
		t.Fatalf("expected expired session redirect flow in html: %q", html)
	}
	if !strings.Contains(html, "Signed out of the admin prototype.") {
		t.Fatalf("expected logout flow in html: %q", html)
	}
	if !strings.Contains(html, "/bulk-delete") {
		t.Fatalf("expected bulk delete endpoint usage in html: %q", html)
	}
	if !strings.Contains(html, "highlightMatch(") {
		t.Fatalf("expected relation highlight flow in html: %q", html)
	}
	if strings.Contains(html, "bulkEditForm") {
		t.Fatalf("expected bulk edit form to be removed from html: %q", html)
	}
	if !strings.Contains(html, "paginationInfo") {
		t.Fatalf("expected pagination controls in html: %q", html)
	}
	if !strings.Contains(html, "id=\"status\" class=\"status-banner\"") {
		t.Fatalf("expected status banner styling in html: %q", html)
	}
	if !strings.Contains(html, "button.className = 'nav-link'") {
		t.Fatalf("expected active resource navigation styling in html: %q", html)
	}
	if strings.Contains(html, "renderActionSummary()") {
		t.Fatalf("expected action pill rendering to be removed from html: %q", html)
	}
	if strings.Contains(html, "id=\"actions\"") {
		t.Fatalf("expected action summary container to be removed from html: %q", html)
	}
	if !strings.Contains(html, "detail-layout") {
		t.Fatalf("expected detail layout styles in html: %q", html)
	}
	if !strings.Contains(html, "Updated record #") {
		t.Fatalf("expected update flow in html: %q", html)
	}
	if !strings.Contains(html, "Deleted record #") {
		t.Fatalf("expected delete flow in html: %q", html)
	}

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()

	projectMeta := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/meta", nil, auth.Token)
	if projectMeta.StatusCode != http.StatusOK {
		t.Fatalf("expected project metadata 200, got %d", projectMeta.StatusCode)
	}
	var meta struct {
		Fields []struct {
			Name      string `json:"name"`
			Component string `json:"component"`
			Relation  *struct {
				Resource   string `json:"resource"`
				LabelField string `json:"label_field"`
			} `json:"relation"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(projectMeta.Body).Decode(&meta); err != nil {
		t.Fatalf("decode project metadata: %v", err)
	}
	projectMeta.Body.Close()
	var ownerFieldFound bool
	for _, field := range meta.Fields {
		if field.Name == "owner_id" && field.Component == "select" && field.Relation != nil && field.Relation.Resource == "users" && field.Relation.LabelField == "name" {
			ownerFieldFound = true
		}
	}
	if !ownerFieldFound {
		t.Fatalf("expected owner_id relation metadata, got %+v", meta.Fields)
	}

	options := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=ali", nil, auth.Token)
	if options.StatusCode != http.StatusOK {
		t.Fatalf("expected relation selector 200, got %d", options.StatusCode)
	}
	var optionsPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := json.NewDecoder(options.Body).Decode(&optionsPayload); err != nil {
		t.Fatalf("decode options: %v", err)
	}
	options.Body.Close()
	if len(optionsPayload.Items) != 1 || optionsPayload.Items[0].Value != 1 || optionsPayload.Items[0].Label != "Alice" {
		t.Fatalf("unexpected relation selector payload: %+v", optionsPayload.Items)
	}

	createProject := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "First Project",
		"summary":  "admin ui demo",
		"owner_id": 1,
	}, auth.Token)
	if createProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected create project 201, got %d body=%s", createProject.StatusCode, readBody(t, createProject.Body))
	}
	createProject.Body.Close()

	projectList := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects", nil, auth.Token)
	if projectList.StatusCode != http.StatusOK {
		t.Fatalf("expected project list 200, got %d", projectList.StatusCode)
	}
	projectListBody := readBody(t, projectList.Body)
	projectList.Body.Close()
	if !strings.Contains(projectListBody, "First Project") {
		t.Fatalf("expected created project in list, got %s", projectListBody)
	}

	createProjectTwo := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "A Project",
		"summary":  "admin ui demo",
		"owner_id": 1,
	}, auth.Token)
	if createProjectTwo.StatusCode != http.StatusCreated {
		t.Fatalf("expected second create project 201, got %d body=%s", createProjectTwo.StatusCode, readBody(t, createProjectTwo.Body))
	}
	createProjectTwo.Body.Close()

	sortedProjects := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects?search=Project&sort=-title", nil, auth.Token)
	if sortedProjects.StatusCode != http.StatusOK {
		t.Fatalf("expected sorted project list 200, got %d", sortedProjects.StatusCode)
	}
	sortedProjectsBody := readBody(t, sortedProjects.Body)
	sortedProjects.Body.Close()
	firstProjectIndex := strings.Index(sortedProjectsBody, "First Project")
	secondProjectIndex := strings.Index(sortedProjectsBody, "A Project")
	if firstProjectIndex == -1 || secondProjectIndex == -1 || firstProjectIndex > secondProjectIndex {
		t.Fatalf("expected sorted projects in descending title order, got %s", sortedProjectsBody)
	}

	pagedProjects := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects?page=2&size=1&sort=title", nil, auth.Token)
	if pagedProjects.StatusCode != http.StatusOK {
		t.Fatalf("expected paged project list 200, got %d", pagedProjects.StatusCode)
	}
	var paged struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		Page  int              `json:"page"`
		Size  int              `json:"size"`
		Pages int              `json:"pages"`
	}
	if err := json.NewDecoder(pagedProjects.Body).Decode(&paged); err != nil {
		t.Fatalf("decode paged projects: %v", err)
	}
	pagedProjects.Body.Close()
	if paged.Total != 2 || paged.Page != 2 || paged.Size != 1 || paged.Pages != 2 || len(paged.Items) != 1 {
		t.Fatalf("unexpected paged project payload: %+v", paged)
	}
	if paged.Items[0]["title"] != "First Project" {
		t.Fatalf("expected second page to contain First Project, got %+v", paged.Items)
	}

	updateProject := doFullJSON(t, server, http.MethodPut, "/api/v1/admin/resources/projects/1", map[string]any{
		"title":    "Renamed Project",
		"summary":  "updated via admin api",
		"owner_id": 1,
	}, auth.Token)
	if updateProject.StatusCode != http.StatusOK {
		t.Fatalf("expected update project 200, got %d body=%s", updateProject.StatusCode, readBody(t, updateProject.Body))
	}
	updateProject.Body.Close()

	projectDetail := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/1", nil, auth.Token)
	if projectDetail.StatusCode != http.StatusOK {
		t.Fatalf("expected project detail 200, got %d", projectDetail.StatusCode)
	}
	projectDetailBody := readBody(t, projectDetail.Body)
	projectDetail.Body.Close()
	if !strings.Contains(projectDetailBody, "Renamed Project") {
		t.Fatalf("expected updated project detail, got %s", projectDetailBody)
	}

	partialUpdate := doFullJSON(t, server, http.MethodPut, "/api/v1/admin/resources/projects/2", map[string]any{
		"summary": "bulk edit compatible partial update",
	}, auth.Token)
	if partialUpdate.StatusCode != http.StatusOK {
		t.Fatalf("expected partial update 200, got %d body=%s", partialUpdate.StatusCode, readBody(t, partialUpdate.Body))
	}
	partialUpdate.Body.Close()

	projectDetailTwo := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/2", nil, auth.Token)
	if projectDetailTwo.StatusCode != http.StatusOK {
		t.Fatalf("expected second project detail 200, got %d", projectDetailTwo.StatusCode)
	}
	projectDetailTwoBody := readBody(t, projectDetailTwo.Body)
	projectDetailTwo.Body.Close()
	if !strings.Contains(projectDetailTwoBody, "bulk edit compatible partial update") {
		t.Fatalf("expected partial update summary in second project detail, got %s", projectDetailTwoBody)
	}

	deleteProject := doFullJSON(t, server, http.MethodDelete, "/api/v1/admin/resources/projects/1", nil, auth.Token)
	if deleteProject.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete project 204, got %d body=%s", deleteProject.StatusCode, readBody(t, deleteProject.Body))
	}
	deleteProject.Body.Close()

	projectListAfterDelete := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects", nil, auth.Token)
	if projectListAfterDelete.StatusCode != http.StatusOK {
		t.Fatalf("expected project list after delete 200, got %d", projectListAfterDelete.StatusCode)
	}
	projectListAfterDeleteBody := readBody(t, projectListAfterDelete.Body)
	projectListAfterDelete.Body.Close()
	if strings.Contains(projectListAfterDeleteBody, "Renamed Project") {
		t.Fatalf("expected deleted project to be absent, got %s", projectListAfterDeleteBody)
	}

	bulkDeleteProjects := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects/bulk-delete", map[string]any{
		"ids": []int{2},
	}, auth.Token)
	if bulkDeleteProjects.StatusCode != http.StatusCreated {
		t.Fatalf("expected bulk delete project 201, got %d body=%s", bulkDeleteProjects.StatusCode, readBody(t, bulkDeleteProjects.Body))
	}
	var bulkDelete struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(bulkDeleteProjects.Body).Decode(&bulkDelete); err != nil {
		t.Fatalf("decode bulk delete: %v", err)
	}
	bulkDeleteProjects.Body.Close()
	if bulkDelete.Deleted != 1 {
		t.Fatalf("expected one bulk deleted project, got %+v", bulkDelete)
	}

	projectListAfterBulkDelete := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects?search=Project", nil, auth.Token)
	if projectListAfterBulkDelete.StatusCode != http.StatusOK {
		t.Fatalf("expected project list after bulk delete 200, got %d", projectListAfterBulkDelete.StatusCode)
	}
	projectListAfterBulkDeleteBody := readBody(t, projectListAfterBulkDelete.Body)
	projectListAfterBulkDelete.Body.Close()
	if strings.Contains(projectListAfterBulkDeleteBody, "A Project") {
		t.Fatalf("expected bulk deleted project to be absent, got %s", projectListAfterBulkDeleteBody)
	}
}

func TestFullExampleRunReturnsListenError(t *testing.T) {
	cfg := settings.Config{
		App: settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{
			Host: "127.0.0.1",
			Port: -1,
		},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:run-full?mode=memory&cache=shared",
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	settings.Global.JWT = cfg.JWT
	log := bootstrap.InitLogger(&cfg.Log)

	if err := run(cfg, log); err == nil {
		t.Fatal("expected run to fail for invalid address")
	}
}

func readBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}

func TestFullExampleInitDBAndMainHelpers(t *testing.T) {
	cfg := settings.Config{
		App:    settings.AppConfig{Name: "Full Example", Version: "1.0.0"},
		Server: settings.ServerConfig{Host: "127.0.0.1", Port: 8080},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:init-full?mode=memory&cache=shared",
		},
		Log: settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"},
	}
	log := bootstrap.InitLogger(&cfg.Log)
	db, err := initDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	api := buildAPI(cfg, db, log)
	if err := api.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if _, err := initDB(&settings.DatabaseConfig{Driver: "oracle"}); err == nil {
		t.Fatal("expected initDB to fail for unsupported driver")
	}

	originalRun := runFullMain
	originalFatal := fatalFull
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() {
		runFullMain = originalRun
		fatalFull = originalFatal
		_ = os.Chdir(wd)
	})

	called := false
	runFullMain = func(cfg settings.Config, log *zap.Logger) error {
		called = true
		return nil
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	main()
	if !called {
		t.Fatal("expected main to invoke injected runner")
	}

	runFullMain = func(cfg settings.Config, log *zap.Logger) error { return errors.New("boom") }
	fatalCalled := false
	fatalFull = func(v ...any) { fatalCalled = true }
	main()
	if !fatalCalled {
		t.Fatal("expected main to invoke injected fatal handler")
	}
}
