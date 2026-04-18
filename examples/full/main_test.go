package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
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

// compactWhitespace normalizes HTML and script snippets so assertions ignore formatting-only changes.
func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func chromiumExecPath(t *testing.T) string {
	t.Helper()

	for _, candidate := range []string{
		"/usr/bin/chromium-browser",
		"/usr/bin/chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Skip("chromium browser not available")
	return ""
}

func newFullBrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromiumExecPath(t)),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 90*time.Second)
	if err := chromedp.Run(timeoutCtx, chromedp.Navigate("about:blank")); err != nil {
		cancelTimeout()
		cancelBrowser()
		cancelAllocator()
		if isBrowserStartupInfraError(err) {
			t.Skipf("skipping browser test because chromium failed to start in this environment: %v", err)
		}
		t.Fatalf("start chromium: %v", err)
	}
	return timeoutCtx, func() {
		cancelTimeout()
		cancelBrowser()
		cancelAllocator()
	}
}

func isBrowserStartupInfraError(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	for _, token := range []string{
		"chrome failed to start",
		"ThreadCache::IsValid",
		"scheduler_loop_quarantine_support.h",
	} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func runBrowser(t *testing.T, ctx context.Context, actions ...chromedp.Action) {
	t.Helper()
	if err := chromedp.Run(ctx, actions...); err != nil {
		t.Fatalf("chromedp run: %v", err)
	}
}

func waitForBrowserCondition(t *testing.T, ctx context.Context, description, expression string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	var last bool
	for time.Now().Before(deadline) {
		if err := chromedp.Run(ctx, chromedp.Evaluate(expression, &last)); err == nil && last {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func waitForBrowserText(t *testing.T, ctx context.Context, selector, want string) {
	t.Helper()
	waitForBrowserCondition(t, ctx, fmt.Sprintf("%s to contain %q", selector, want), fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		return !!el && String(el.textContent || "").includes(%q);
	})()`, selector, want))
}

func waitForBrowserPath(t *testing.T, ctx context.Context, want string) {
	t.Helper()
	waitForBrowserCondition(t, ctx, "browser path "+want, fmt.Sprintf(`window.location.pathname === %q`, want))
}

func waitForBrowserEnabled(t *testing.T, ctx context.Context, selector string) {
	t.Helper()
	waitForBrowserCondition(t, ctx, selector+" enabled", fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		return !!el && !el.disabled;
	})()`, selector))
}

func waitForBrowserExists(t *testing.T, ctx context.Context, selector string) {
	t.Helper()
	waitForBrowserCondition(t, ctx, selector+" exists", fmt.Sprintf(`document.querySelector(%q) !== null`, selector))
}

func waitForBrowserVisible(t *testing.T, ctx context.Context, selector string) {
	t.Helper()
	waitForBrowserCondition(t, ctx, selector+" visible", fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el || el.hidden) return false;
		const style = window.getComputedStyle(el);
		return style.display !== "none" && style.visibility !== "hidden" && style.opacity !== "0";
	})()`, selector))
}

func setBrowserValue(t *testing.T, ctx context.Context, selector, value string) {
	t.Helper()
	var ok bool
	runBrowser(t, ctx, chromedp.Evaluate(fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return false;
		el.focus();
		el.value = %q;
		el.dispatchEvent(new Event("input", { bubbles: true }));
		el.dispatchEvent(new Event("change", { bubbles: true }));
		return true;
	})()`, selector, value), &ok))
	if !ok {
		t.Fatalf("failed to set %s", selector)
	}
}

func clickBrowser(t *testing.T, ctx context.Context, selector string) {
	t.Helper()
	var ok bool
	runBrowser(t, ctx, chromedp.Evaluate(fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return false;
		el.click();
		return true;
	})()`, selector), &ok))
	if !ok {
		t.Fatalf("failed to click %s", selector)
	}
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
	if !strings.Contains(loginHTML, `const adminLoginPath = "/admin/login"`) {
		t.Fatalf("expected standalone login path in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "Gin Ninja") {
		t.Fatalf("expected brand name in login marketing panel in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "login-brand-mark") {
		t.Fatalf("expected login brand mark in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "document.body.classList.toggle('standalone-login-page', isStandaloneLoginPage())") {
		t.Fatalf("expected standalone login body class toggle in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "body.standalone-login-page .topbar { display:none; }") {
		t.Fatalf("expected standalone login page to hide the top header in html: %q", loginHTML)
	}
	if !strings.Contains(loginHTML, "body.standalone-login-page .app-main {\n      max-width:1200px;\n      margin:0 auto;\n      width:100%;\n      min-height:100vh;\n      align-content:center;") {
		t.Fatalf("expected standalone login page to vertically center the main login shell in html: %q", loginHTML)
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
	if !strings.Contains(loginHTML, "[data-theme=\"dark\"] body.standalone-login-page .session-panel {") {
		t.Fatalf("expected standalone login dark mode card override in html: %q", loginHTML)
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
	if !strings.Contains(adminHTML, `const adminPagePath = "/admin"`) {
		t.Fatalf("expected standalone admin path in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Admin Workspace") {
		t.Fatalf("expected compact admin workspace header copy to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"workspace-path muted\"") {
		t.Fatalf("expected compact workspace summary markup in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.resourcePath.textContent = 'Browse, inspect, and edit ' + state.meta.label + '.';") {
		t.Fatalf("expected shorter admin workspace summary copy in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.loginForm.hidden = false;") {
		t.Fatalf("expected signed-out admin page to keep login form visible in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "els.adminShell.hidden = true;") {
		t.Fatalf("expected signed-out admin page to keep admin shell hidden in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "Resource navigation") || !strings.Contains(adminHTML, "Switch workspaces") {
		t.Fatalf("expected compact resource strip copy in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "Move between admin resources from a left-hand menu while keeping the workspace focused.") {
		t.Fatalf("expected left-hand resource navigation copy in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "sidebar-brand-mark") || !strings.Contains(adminHTML, "Alexander Pierce") {
		t.Fatalf("expected AdminLTE-style sidebar brand and user panel in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "aria-label=\"Admin navigation shortcuts\"") || !strings.Contains(adminHTML, "aria-label=\"Admin quick actions\"") {
		t.Fatalf("expected AdminLTE-style topbar navigation chrome in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "class=\"topbar-link-icon\"") {
		t.Fatalf("expected Home shortcut icon markup to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Control panel") {
		t.Fatalf("expected Control panel label to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Standalone workspace") {
		t.Fatalf("expected Standalone workspace label to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "class=\"topbar-context\"") {
		t.Fatalf("expected topbar context container to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "aria-label=\"Search sidebar navigation\"") {
		t.Fatalf("expected AdminLTE-style sidebar search box in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"sidebarResourceSearch\"") || !strings.Contains(adminHTML, "id=\"sidebarResourceSearchButton\"") {
		t.Fatalf("expected searchable sidebar resource controls in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"sidebarDashboardLink\"") {
		t.Fatalf("expected AdminLTE-style dashboard entry in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"sidebar-section-label\">Overview</div>") {
		t.Fatalf("expected AdminLTE-style overview section label in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"sidebar-treeview-toggle-copy\"") || !strings.Contains(adminHTML, "class=\"sidebar-treeview-toggle-icon\"") || !strings.Contains(adminHTML, "class=\"sidebar-treeview-toggle-text\">Resources</span>") {
		t.Fatalf("expected AdminLTE-style resource treeview toggle markup in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "icon.innerHTML = dashboardTileMeta(resource, index).icon;") {
		t.Fatalf("expected sidebar resource entries to reuse dashboard tile icons: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"resourceTreeviewBadge\"") {
		t.Fatalf("expected AdminLTE-style sidebar badge markup in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "class=\"nav-link-suffix\"") {
		t.Fatalf("expected dashboard and child resource sidebar chevrons to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "function filteredResources()") || !strings.Contains(adminHTML, "state.resourceSearch = els.sidebarResourceSearch.value.trim();") {
		t.Fatalf("expected sidebar resource search filtering logic in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, ">Navigation<") {
		t.Fatalf("expected old sidebar navigation label to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Choose a resource to manage records, filters, and bulk actions.") {
		t.Fatalf("expected old sidebar helper copy to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"panel resource-strip stack sidebar-shell\"") {
		t.Fatalf("expected admin sidebar resource shell in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"openCreateModal\"") {
		t.Fatalf("expected create modal trigger in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Workspace</span>") {
		t.Fatalf("expected workspace eyebrow to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "AdminLTE workspace chrome") {
		t.Fatalf("expected workspace chrome heading to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Operate resources with dashboard, filters, and table controls in one place.") {
		t.Fatalf("expected workspace chrome description to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"resource-header-actions\"") {
		t.Fatalf("expected create action to move into the resource header actions area in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"workspaceHeader\"") || !strings.Contains(adminHTML, "id=\"recordsShell\"") {
		t.Fatalf("expected dashboard-toggleable workspace header and records shells in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"card-header section-card-header\"") || !strings.Contains(adminHTML, "class=\"card-footer section-card-footer\"") {
		t.Fatalf("expected AdminLTE-style card header/footer treatment in html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "Record workspace") {
		t.Fatalf("expected record workspace card to be removed from html: %q", adminHTML)
	}
	if strings.Contains(adminHTML, "id=\"openBulkEditModal\"") {
		t.Fatalf("expected bulk edit trigger to be removed from html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "class=\"modal-overlay\" hidden") {
		t.Fatalf("expected hidden modal overlay shell in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "id=\"recordModal\"") || !strings.Contains(adminHTML, "id=\"editModal\"") {
		t.Fatalf("expected record and edit modal shells in html: %q", adminHTML)
	}
	if !strings.Contains(adminHTML, "actionHead.textContent = 'Actions';") {
		t.Fatalf("expected row action column in html: %q", adminHTML)
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
	if !strings.Contains(html, `const apiBase = "/api/v1/admin"`) {
		t.Fatalf("expected admin api base in html: %q", html)
	}
	if !strings.Contains(html, `const prototypePagePath = "/admin-prototype"`) {
		t.Fatalf("expected prototype page path in html: %q", html)
	}
	if !strings.Contains(html, "selectRecord(row, { openModal: 'record' })") {
		t.Fatalf("expected record selection flow in html: %q", html)
	}
	if !strings.Contains(html, "renderFilterControls()") {
		t.Fatalf("expected filter controls in html: %q", html)
	}
	if !strings.Contains(html, "function scheduleSearchReload()") {
		t.Fatalf("expected debounced resource search helper in html: %q", html)
	}
	if !strings.Contains(html, "els.search.addEventListener('input'") {
		t.Fatalf("expected search input to trigger live reloads in html: %q", html)
	}
	if !strings.Contains(html, "function renderSearchPlaceholder()") {
		t.Fatalf("expected search placeholder renderer in html: %q", html)
	}
	for _, needle := range []string{"Search by ", "labels.join(', ')", "Search current resource"} {
		if !strings.Contains(html, needle) {
			t.Fatalf("expected search placeholder component %q in html: %q", needle, html)
		}
	}
	if !strings.Contains(html, "scheduleRelationSearch(") {
		t.Fatalf("expected relation search flow in html: %q", html)
	}
	if !strings.Contains(html, "resolveRelationSelection(field, items, selectedRelationValues(select, field), term)") {
		t.Fatalf("expected relation exact-id auto-selection flow in html: %q", html)
	}
	if !strings.Contains(html, "option.textContent = 'Choose ' + placeholderLabel;") {
		t.Fatalf("expected relation selects to include an explicit empty choice in html: %q", html)
	}
	if !strings.Contains(html, "preview.hidden = true;") {
		t.Fatalf("expected relation preview to stay hidden until searching in html: %q", html)
	}
	if !strings.Contains(html, "const numericFieldPattern = /^-?\\d+(?:\\.\\d+)?$/;") {
		t.Fatalf("expected relation numeric parsing helper in html: %q", html)
	}
	if !strings.Contains(html, "payload[key] = numericFieldPattern.test(value) ? Number(value) : value;") {
		t.Fatalf("expected relation values to be serialized with numeric JSON types in html: %q", html)
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
	if !strings.Contains(html, "closeAllModals();") {
		t.Fatalf("expected shared modal close flow in html: %q", html)
	}
	if strings.Contains(html, "closeModal(els.bulkEditModal);") {
		t.Fatalf("expected bulk edit modal close flow to be removed from html: %q", html)
	}
	if !strings.Contains(html, "document.addEventListener('keydown'") {
		t.Fatalf("expected modal escape-key handling in html: %q", html)
	}
	if !strings.Contains(html, "openButton.onclick = () => selectRecord(row, { openModal: 'record' });") {
		t.Fatalf("expected row open action to launch the record modal in html: %q", html)
	}
	if !strings.Contains(html, "selectRecord(row, { openModal: 'edit' })") {
		t.Fatalf("expected row edit action to launch the edit modal in html: %q", html)
	}
	if !strings.Contains(html, "deleteRecordByID(id)") {
		t.Fatalf("expected row delete action in html: %q", html)
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
	if !strings.Contains(html, "Dashboard cards") || !strings.Contains(html, "Table tools") {
		t.Fatalf("expected richer AdminLTE-style dashboard and table section labels in html: %q", html)
	}
	if !strings.Contains(html, "dashboard-tile-description") || !strings.Contains(html, "dashboard-tile-icon-badge") || !strings.Contains(html, "Open workspace") {
		t.Fatalf("expected upgraded dashboard resource card chrome in html: %q", html)
	}
	if !strings.Contains(html, "Connected admin workspace") || !strings.Contains(html, "function dashboardTileMeta(resource, index)") {
		t.Fatalf("expected richer dashboard resource metadata helpers in html: %q", html)
	}
	if !strings.Contains(html, "els.workspaceHeader.hidden = true;") || !strings.Contains(html, "els.recordsShell.hidden = true;") {
		t.Fatalf("expected dashboard state to hide workspace header and records shell in html: %q", html)
	}
	if !strings.Contains(html, "els.workspaceHeader.hidden = false;") || !strings.Contains(html, "els.recordsShell.hidden = false;") {
		t.Fatalf("expected resource state to restore workspace header and records shell in html: %q", html)
	}
	if !strings.Contains(html, "id=\"status\" class=\"visually-hidden\" aria-live=\"polite\" aria-atomic=\"true\"") {
		t.Fatalf("expected hidden live status region in html: %q", html)
	}
	if strings.Contains(html, "class=\"status-banner\"") {
		t.Fatalf("expected visible status banner card to be removed from html: %q", html)
	}
	if !strings.Contains(html, "button.className = 'nav-link'") {
		t.Fatalf("expected active resource navigation styling in html: %q", html)
	}
	if !strings.Contains(html, "sidebar-treeview-toggle-badge") || !strings.Contains(html, "box-shadow:0 10px 18px rgba(0, 123, 255, 0.3);") {
		t.Fatalf("expected AdminLTE-style sidebar menu styling in html: %q", html)
	}
	if strings.Contains(html, "nav-link-caret") {
		t.Fatalf("expected sidebar submenu leaf caret to be removed from html: %q", html)
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
	if !strings.Contains(html, "id=\"toastContainer\"") || !strings.Contains(html, "class=\"toast-container\"") {
		t.Fatalf("expected toast notification container in html: %q", html)
	}
	if !strings.Contains(html, "function showToast(message, tone, durationMs)") {
		t.Fatalf("expected showToast function in html: %q", html)
	}
	if !strings.Contains(html, "toast.dataset.tone = tone || inferStatusTone(message)") {
		t.Fatalf("expected toast tone assignment in html: %q", html)
	}
	if !strings.Contains(html, "event.key === '/' && state.current") {
		t.Fatalf("expected '/' keyboard shortcut to focus search in html: %q", html)
	}
	if !strings.Contains(html, "event.key === 'n' && !event.shiftKey") {
		t.Fatalf("expected 'n' keyboard shortcut to open create modal in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"]") {
		t.Fatalf("expected dark mode CSS custom properties in html: %q", html)
	}
	if !strings.Contains(html, "id=\"darkModeToggle\"") {
		t.Fatalf("expected dark mode toggle button in html: %q", html)
	}
	if !strings.Contains(html, "function applyTheme(dark)") {
		t.Fatalf("expected applyTheme function in html: %q", html)
	}
	if !strings.Contains(html, "function toggleDarkMode()") {
		t.Fatalf("expected toggleDarkMode function in html: %q", html)
	}
	if !strings.Contains(html, "function restoreTheme()") {
		t.Fatalf("expected restoreTheme function in html: %q", html)
	}
	if !strings.Contains(html, "localStorage.setItem(themeStorageKey") {
		t.Fatalf("expected dark mode localStorage persistence in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"] body.standalone-login-page {") {
		t.Fatalf("expected standalone login dark mode background override in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"] body.standalone-login-page .login-marketing {") {
		t.Fatalf("expected standalone login dark mode marketing panel override in html: %q", html)
	}
	if !strings.Contains(html, "id=\"topbarSearchInput\"") {
		t.Fatalf("expected topbar search input in html: %q", html)
	}
	if !strings.Contains(html, "id=\"topbarSearchResults\"") {
		t.Fatalf("expected topbar search results panel in html: %q", html)
	}
	if !strings.Contains(html, "function globalSearch(query)") {
		t.Fatalf("expected globalSearch function in html: %q", html)
	}
	if !strings.Contains(html, "function closeGlobalSearch()") {
		t.Fatalf("expected closeGlobalSearch function in html: %q", html)
	}
	if !strings.Contains(html, "topbar-search-results") {
		t.Fatalf("expected topbar-search-results CSS class in html: %q", html)
	}
	if !strings.Contains(html, ".topbar-search-expand.has-results input { border-radius:0.35rem 0.35rem 0 0; }") {
		t.Fatalf("expected topbar search radius override when results are visible in html: %q", html)
	}
	if !strings.Contains(html, "sortable-th") {
		t.Fatalf("expected sortable-th CSS class in html: %q", html)
	}
	if !strings.Contains(html, "function applySortFromHeader(field)") {
		t.Fatalf("expected applySortFromHeader function in html: %q", html)
	}
	if !strings.Contains(html, "function activeSortField()") {
		t.Fatalf("expected activeSortField function in html: %q", html)
	}
	if !strings.Contains(html, "function closeActionMenuPortal()") {
		t.Fatalf("expected closeActionMenuPortal function in html: %q", html)
	}
	if !strings.Contains(html, "function openActionMenuAt(triggerEl") {
		t.Fatalf("expected openActionMenuAt function in html: %q", html)
	}
	if !strings.Contains(html, "action-menu-portal") {
		t.Fatalf("expected action-menu-portal in html: %q", html)
	}
	if !strings.Contains(html, ".table-shell { background: var(--admin-surface)") {
		t.Fatalf("expected dark mode table-shell override in html: %q", html)
	}
	if !strings.Contains(html, ".detail-card { background: var(--admin-surface)") {
		t.Fatalf("expected dark mode detail-card override in html: %q", html)
	}
	if !strings.Contains(html, ".action-menu-trigger,") {
		t.Fatalf("expected dark mode action-menu-trigger override in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"] th { background: #22253a; color: var(--admin-muted); }") {
		t.Fatalf("expected dark mode th override in html: %q", html)
	}
	if !strings.Contains(html, ".topbar-user-avatar { background: #2d3242") {
		t.Fatalf("expected dark mode topbar-user-avatar override in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"] .multi-relation-dropdown summary,") {
		t.Fatalf("expected dark mode multi-relation dropdown override in html: %q", html)
	}
	if !strings.Contains(html, "[data-theme=\"dark\"] .multi-relation-option input { accent-color: var(--admin-primary-dark); }") {
		t.Fatalf("expected dark mode multi-relation checkbox accent override in html: %q", html)
	}
	if !strings.Contains(html, ".relation-search { width:100%; }") {
		t.Fatalf("expected embedded multi-relation search styling in html: %q", html)
	}
	if !strings.Contains(html, "dropdownMenu.appendChild(searchInput);") {
		t.Fatalf("expected multi-relation search input rendered inside dropdown menu in html: %q", html)
	}
	if !strings.Contains(html, "const dropdown = document.createElement('details');") {
		t.Fatalf("expected relation fields to render with dropdown controls in html: %q", html)
	}
	if !strings.Contains(html, "document.createElement(multiRelation ? 'label' : 'button')") {
		t.Fatalf("expected single relation dropdown options to render as buttons in html: %q", html)
	}
	if !strings.Contains(html, "button:hover:not(:disabled) { filter: brightness(1.15)") {
		t.Fatalf("expected dark mode button hover filter reversal in html: %q", html)
	}
	if !strings.Contains(html, "const cb = pendingConfirmCallback; pendingConfirmCallback = null; if (cb) cb()") {
		t.Fatalf("expected safe confirm callback invocation in html: %q", html)
	}
	if !strings.Contains(html, ".action-menu-item.danger { background:transparent; color:var(--admin-danger); border-color:transparent; }") {
		t.Fatalf("expected action menu danger item override in html: %q", html)
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

	optionsByID := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=1", nil, auth.Token)
	if optionsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected relation selector by id 200, got %d", optionsByID.StatusCode)
	}
	var optionsByIDPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := json.NewDecoder(optionsByID.Body).Decode(&optionsByIDPayload); err != nil {
		t.Fatalf("decode options by id: %v", err)
	}
	optionsByID.Body.Close()
	if len(optionsByIDPayload.Items) != 1 || optionsByIDPayload.Items[0].Value != 1 || optionsByIDPayload.Items[0].Label != "Alice" {
		t.Fatalf("unexpected relation selector by id payload: %+v", optionsByIDPayload.Items)
	}

	missingOptionsByID := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=999", nil, auth.Token)
	if missingOptionsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected missing relation selector by id 200, got %d", missingOptionsByID.StatusCode)
	}
	var missingOptionsByIDPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := json.NewDecoder(missingOptionsByID.Body).Decode(&missingOptionsByIDPayload); err != nil {
		t.Fatalf("decode missing options by id: %v", err)
	}
	missingOptionsByID.Body.Close()
	if len(missingOptionsByIDPayload.Items) != 0 {
		t.Fatalf("expected empty relation selector by missing id payload: %+v", missingOptionsByIDPayload.Items)
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

	sortedProjectsByID := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects?sort=-id", nil, auth.Token)
	if sortedProjectsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected id-sorted project list 200, got %d", sortedProjectsByID.StatusCode)
	}
	var sortedByID struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(sortedProjectsByID.Body).Decode(&sortedByID); err != nil {
		t.Fatalf("decode id-sorted projects: %v", err)
	}
	sortedProjectsByID.Body.Close()
	if len(sortedByID.Items) < 2 || sortedByID.Items[0]["id"] != float64(2) || sortedByID.Items[1]["id"] != float64(1) {
		t.Fatalf("unexpected id-sorted projects payload: %+v", sortedByID.Items)
	}

	filteredProjectsByID := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects?id=2", nil, auth.Token)
	if filteredProjectsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected id-filtered project list 200, got %d", filteredProjectsByID.StatusCode)
	}
	var filteredByID struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.NewDecoder(filteredProjectsByID.Body).Decode(&filteredByID); err != nil {
		t.Fatalf("decode id-filtered projects: %v", err)
	}
	filteredProjectsByID.Body.Close()
	if filteredByID.Total != 1 || len(filteredByID.Items) != 1 || filteredByID.Items[0]["id"] != float64(2) {
		t.Fatalf("unexpected id-filtered projects payload: %+v", filteredByID)
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

func TestFullExampleAdminAPIUsersAndProjectPermissions(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	for _, user := range []map[string]any{
		{
			"name":     "Alice",
			"email":    "alice@example.com",
			"password": "password123",
			"age":      18,
		},
		{
			"name":     "Bob",
			"email":    "bob@example.com",
			"password": "password123",
			"age":      22,
		},
	} {
		register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", user, "")
		if register.StatusCode != http.StatusCreated {
			t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
		}
		register.Body.Close()
	}

	login := func(email string) string {
		t.Helper()
		resp := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": "password123",
		}, "")
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected login 201 for %s, got %d body=%s", email, resp.StatusCode, readBody(t, resp.Body))
		}
		var auth struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
			t.Fatalf("decode login for %s: %v", email, err)
		}
		resp.Body.Close()
		if auth.Token == "" {
			t.Fatalf("expected login token for %s", email)
		}
		return auth.Token
	}

	aliceToken := login("alice@example.com")
	bobToken := login("bob@example.com")

	for _, role := range []map[string]any{
		{
			"name":   "Administrators",
			"code":   "admin",
			"status": 1,
			"remark": "full access",
		},
		{
			"name":   "Editors",
			"code":   "editor",
			"status": 1,
			"remark": "content editors",
		},
	} {
		createRoleResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/roles", role, aliceToken)
		if createRoleResp.StatusCode != http.StatusCreated {
			t.Fatalf("expected role create 201, got %d body=%s", createRoleResp.StatusCode, readBody(t, createRoleResp.Body))
		}
		createRoleResp.Body.Close()
	}

	resourceIndexResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources", nil, aliceToken)
	if resourceIndexResp.StatusCode != http.StatusOK {
		t.Fatalf("expected admin resource index 200, got %d", resourceIndexResp.StatusCode)
	}
	var resourceIndex struct {
		Resources []struct {
			Name  string `json:"name"`
			Label string `json:"label"`
			Path  string `json:"path"`
		} `json:"resources"`
	}
	if err := json.NewDecoder(resourceIndexResp.Body).Decode(&resourceIndex); err != nil {
		t.Fatalf("decode resource index: %v", err)
	}
	resourceIndexResp.Body.Close()
	if len(resourceIndex.Resources) != 3 {
		t.Fatalf("expected 3 admin resources, got %+v", resourceIndex.Resources)
	}
	resourcePaths := map[string]string{}
	for _, resource := range resourceIndex.Resources {
		resourcePaths[resource.Name] = resource.Path
	}
	if resourcePaths["users"] != "/users" || resourcePaths["roles"] != "/roles" || resourcePaths["projects"] != "/projects" {
		t.Fatalf("unexpected admin resources: %+v", resourceIndex.Resources)
	}

	usersMetaResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/users/meta", nil, aliceToken)
	if usersMetaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected users metadata 200, got %d", usersMetaResp.StatusCode)
	}
	var usersMeta struct {
		Actions      []string `json:"actions"`
		CreateFields []string `json:"create_fields"`
		UpdateFields []string `json:"update_fields"`
		Fields       []struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			Component string `json:"component"`
			Relation  *struct {
				Resource   string `json:"resource"`
				LabelField string `json:"label_field"`
			} `json:"relation"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(usersMetaResp.Body).Decode(&usersMeta); err != nil {
		t.Fatalf("decode users metadata: %v", err)
	}
	usersMetaResp.Body.Close()
	actionSet := map[string]bool{}
	for _, action := range usersMeta.Actions {
		actionSet[action] = true
	}
	for _, action := range []string{"list", "detail", "create", "update", "delete", "bulk_delete"} {
		if !actionSet[action] {
			t.Fatalf("expected action %q in users metadata, got %+v", action, usersMeta.Actions)
		}
	}
	if !strings.Contains(strings.Join(usersMeta.CreateFields, ","), "password") || !strings.Contains(strings.Join(usersMeta.UpdateFields, ","), "password") {
		t.Fatalf("expected password field in users metadata create/update fields, got %+v", usersMeta)
	}
	if !strings.Contains(strings.Join(usersMeta.CreateFields, ","), "role_ids") || !strings.Contains(strings.Join(usersMeta.UpdateFields, ","), "role_ids") {
		t.Fatalf("expected role_ids field in users metadata create/update fields, got %+v", usersMeta)
	}
	var roleIDsFieldFound bool
	for _, field := range usersMeta.Fields {
		if field.Name == "role_ids" && field.Type == "array" && field.Component == "select" && field.Relation != nil && field.Relation.Resource == "roles" && field.Relation.LabelField == "name" {
			roleIDsFieldFound = true
		}
	}
	if !roleIDsFieldFound {
		t.Fatalf("expected role_ids relation metadata, got %+v", usersMeta.Fields)
	}

	roleOptionsResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/users/fields/role_ids/options?search=adm", nil, aliceToken)
	if roleOptionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected role relation selector 200, got %d", roleOptionsResp.StatusCode)
	}
	var roleOptions struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := json.NewDecoder(roleOptionsResp.Body).Decode(&roleOptions); err != nil {
		t.Fatalf("decode role options: %v", err)
	}
	roleOptionsResp.Body.Close()
	if len(roleOptions.Items) != 1 || roleOptions.Items[0].Value != 1 || roleOptions.Items[0].Label != "Administrators" {
		t.Fatalf("unexpected role relation selector payload: %+v", roleOptions.Items)
	}

	createUserResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/users", map[string]any{
		"name":     "  Carol Admin  ",
		"email":    "  CAROL@EXAMPLE.COM ",
		"password": "password123",
		"age":      27,
		"is_admin": true,
		"role_ids": []int{1, 2},
	}, aliceToken)
	if createUserResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected admin user create 201, got %d body=%s", createUserResp.StatusCode, readBody(t, createUserResp.Body))
	}
	var createdUser struct {
		Item map[string]any `json:"item"`
	}
	if err := json.NewDecoder(createUserResp.Body).Decode(&createdUser); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	createUserResp.Body.Close()
	if createdUser.Item["name"] != "Carol Admin" || createdUser.Item["email"] != "carol@example.com" {
		t.Fatalf("expected normalized created user payload, got %+v", createdUser.Item)
	}
	if createdUser.Item["is_admin"] != true {
		t.Fatalf("expected created user to preserve is_admin=true, got %+v", createdUser.Item)
	}
	roleIDs, ok := createdUser.Item["role_ids"].([]any)
	if !ok || len(roleIDs) != 2 || roleIDs[0] != float64(1) || roleIDs[1] != float64(2) {
		t.Fatalf("expected created user role_ids [1 2], got %+v", createdUser.Item["role_ids"])
	}
	if _, ok := createdUser.Item["password"]; ok {
		t.Fatalf("expected password to stay hidden in admin response, got %+v", createdUser.Item)
	}
	if createdUser.Item["id"] != float64(3) {
		t.Fatalf("expected created user id 3, got %+v", createdUser.Item)
	}

	_ = login("carol@example.com")

	updateUserResp := doFullJSON(t, server, http.MethodPut, "/api/v1/admin/resources/users/3", map[string]any{
		"name":     "  Carol Updated  ",
		"email":    "  CAROL.UPDATED@EXAMPLE.COM ",
		"age":      28,
		"role_ids": []int{2},
	}, aliceToken)
	if updateUserResp.StatusCode != http.StatusOK {
		t.Fatalf("expected admin user update 200, got %d body=%s", updateUserResp.StatusCode, readBody(t, updateUserResp.Body))
	}
	var updatedUser struct {
		Item map[string]any `json:"item"`
	}
	if err := json.NewDecoder(updateUserResp.Body).Decode(&updatedUser); err != nil {
		t.Fatalf("decode updated user: %v", err)
	}
	updateUserResp.Body.Close()
	if updatedUser.Item["name"] != "Carol Updated" || updatedUser.Item["email"] != "carol.updated@example.com" {
		t.Fatalf("expected normalized updated user payload, got %+v", updatedUser.Item)
	}
	updatedRoleIDs, ok := updatedUser.Item["role_ids"].([]any)
	if !ok || len(updatedRoleIDs) != 1 || updatedRoleIDs[0] != float64(2) {
		t.Fatalf("expected updated user role_ids [2], got %+v", updatedUser.Item["role_ids"])
	}

	_ = login("carol.updated@example.com")

	invalidUserResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/users", map[string]any{
		"name":  "No Password",
		"email": "nopassword@example.com",
		"age":   19,
	}, aliceToken)
	if invalidUserResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected admin user validation 400, got %d body=%s", invalidUserResp.StatusCode, readBody(t, invalidUserResp.Body))
	}
	invalidUserBody := readBody(t, invalidUserResp.Body)
	invalidUserResp.Body.Close()
	if !strings.Contains(invalidUserBody, "password") || !strings.Contains(invalidUserBody, "required") {
		t.Fatalf("expected missing-password validation message, got %s", invalidUserBody)
	}

	createAliceProject := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "Alice Private Project",
		"summary":  "owned by alice",
		"owner_id": 1,
	}, aliceToken)
	if createAliceProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected alice project create 201, got %d body=%s", createAliceProject.StatusCode, readBody(t, createAliceProject.Body))
	}
	var aliceProject struct {
		Item map[string]any `json:"item"`
	}
	if err := json.NewDecoder(createAliceProject.Body).Decode(&aliceProject); err != nil {
		t.Fatalf("decode alice project: %v", err)
	}
	createAliceProject.Body.Close()

	createBobProject := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "Bob Visible Project",
		"summary":  "owned by bob",
		"owner_id": 2,
	}, bobToken)
	if createBobProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected bob project create 201, got %d body=%s", createBobProject.StatusCode, readBody(t, createBobProject.Body))
	}
	var bobProject struct {
		Item map[string]any `json:"item"`
	}
	if err := json.NewDecoder(createBobProject.Body).Decode(&bobProject); err != nil {
		t.Fatalf("decode bob project: %v", err)
	}
	createBobProject.Body.Close()

	bobListResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects", nil, bobToken)
	if bobListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected bob project list 200, got %d", bobListResp.StatusCode)
	}
	bobListBody := readBody(t, bobListResp.Body)
	bobListResp.Body.Close()
	if strings.Contains(bobListBody, "Alice Private Project") || !strings.Contains(bobListBody, "Bob Visible Project") {
		t.Fatalf("expected bob project list to be row-scoped, got %s", bobListBody)
	}

	aliceProjectID := int(aliceProject.Item["id"].(float64))
	bobProjectID := int(bobProject.Item["id"].(float64))

	bobReadsAliceResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), nil, bobToken)
	if bobReadsAliceResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected bob to get 404 for alice project detail, got %d body=%s", bobReadsAliceResp.StatusCode, readBody(t, bobReadsAliceResp.Body))
	}
	bobReadsAliceResp.Body.Close()

	bobUpdatesAliceResp := doFullJSON(t, server, http.MethodPut, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), map[string]any{
		"title": "blocked",
	}, bobToken)
	if bobUpdatesAliceResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected bob to get 404 for alice project update, got %d body=%s", bobUpdatesAliceResp.StatusCode, readBody(t, bobUpdatesAliceResp.Body))
	}
	bobUpdatesAliceResp.Body.Close()

	bobBulkDeleteResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/projects/bulk-delete", map[string]any{
		"ids": []int{aliceProjectID, bobProjectID},
	}, bobToken)
	if bobBulkDeleteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected bob bulk delete 201, got %d body=%s", bobBulkDeleteResp.StatusCode, readBody(t, bobBulkDeleteResp.Body))
	}
	var bobBulkDelete struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(bobBulkDeleteResp.Body).Decode(&bobBulkDelete); err != nil {
		t.Fatalf("decode bob bulk delete: %v", err)
	}
	bobBulkDeleteResp.Body.Close()
	if bobBulkDelete.Deleted != 1 {
		t.Fatalf("expected bob bulk delete to remove only his own project, got %+v", bobBulkDelete)
	}

	aliceProjectStillThereResp := doFullJSON(t, server, http.MethodGet, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), nil, aliceToken)
	if aliceProjectStillThereResp.StatusCode != http.StatusOK {
		t.Fatalf("expected alice project to remain after bob bulk delete, got %d body=%s", aliceProjectStillThereResp.StatusCode, readBody(t, aliceProjectStillThereResp.Body))
	}
	aliceProjectStillThereBody := readBody(t, aliceProjectStillThereResp.Body)
	aliceProjectStillThereResp.Body.Close()
	if !strings.Contains(aliceProjectStillThereBody, "Alice Private Project") {
		t.Fatalf("expected alice project detail after bob bulk delete, got %s", aliceProjectStillThereBody)
	}
}

func TestFullExampleAdminPrototypeBrowserCRUDFlow(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserText(t, ctx, "#resources", "Projects")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")

	setBrowserValue(t, ctx, "#sidebarResourceSearch", "proj")
	waitForBrowserCondition(t, ctx, "sidebar resource search filters navigation", `(() => {
		const resources = document.querySelector("#resources");
		return !!resources && resources.textContent.includes("Projects") && !resources.textContent.includes("Users");
	})()`)
	clickBrowser(t, ctx, "#sidebarResourceSearchButton")
	waitForBrowserCondition(t, ctx, "sidebar resource search reset restores navigation", `(() => {
		const resources = document.querySelector("#resources");
		const search = document.querySelector("#sidebarResourceSearch");
		return !!resources && !!search && search.value === "" && resources.textContent.includes("Users") && resources.textContent.includes("Projects");
	})()`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const button = Array.from(document.querySelectorAll('#resources .nav-link'))
			.find((node) => node.textContent && node.textContent.includes('Projects'));
		if (button) button.click();
		return !!button;
	})()`, nil))
	waitForBrowserText(t, ctx, "#resourceTitle", "Projects")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")
	waitForBrowserExists(t, ctx, "#createForm textarea[name='title']")

	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserExists(t, ctx, "#createForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "owner relation dropdown options loaded", `(() => {
		const menu = document.querySelector("#createForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Alice"));
	})()`)
	clickBrowser(t, ctx, "#createForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "owner relation dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		const search = document.querySelector("#createForm .relation-search");
		const options = document.querySelectorAll("#createForm .multi-relation-option");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search && options.length > 0;
	})()`)

	setBrowserValue(t, ctx, "#createForm textarea[name='title']", "Black Box Project")
	setBrowserValue(t, ctx, "#createForm textarea[name='summary']", "created via browser integration")
	setBrowserValue(t, ctx, "#createForm .relation-search", "ali")
	waitForBrowserCondition(t, ctx, "owner relation option filtered", `(() => {
		const options = Array.from(document.querySelectorAll("#createForm .multi-relation-option"));
		return options.length > 0 && options.every((option) => option.textContent.includes("Ali"));
	})()`)
	clickBrowser(t, ctx, "#createForm .multi-relation-option")
	waitForBrowserCondition(t, ctx, "owner relation value selected", `(() => {
		const select = document.querySelector("#createForm select[name='owner_id']");
		const summary = document.querySelector("#createForm details.multi-relation-dropdown summary");
		return !!select && select.value === "1" && !!summary && summary.textContent.includes("Alice");
	})()`)
	clickBrowser(t, ctx, "#createForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Created a new projects record.")
	waitForBrowserText(t, ctx, "#list", "Black Box Project")
	// Toast should appear for successful create
	waitForBrowserCondition(t, ctx, "create toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Created a new projects record.");
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-btn-view")
	waitForBrowserVisible(t, ctx, "#recordModal")
	waitForBrowserText(t, ctx, "#detailTitle", "Projects #1")
	waitForBrowserText(t, ctx, "#detailFields", "created via browser integration")

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	clickBrowser(t, ctx, ".action-menu-list.open .action-menu-item")
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserExists(t, ctx, "#updateForm textarea[name='summary']")

	setBrowserValue(t, ctx, "#updateForm textarea[name='summary']", "updated through browser flow")
	clickBrowser(t, ctx, "#updateForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Updated record #1.")
	waitForBrowserText(t, ctx, "#list", "Black Box Project")
	// Toast should appear for successful update
	waitForBrowserCondition(t, ctx, "update toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Updated record #1.");
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-btn-view")
	waitForBrowserText(t, ctx, "#detailFields", "updated through browser flow")
	clickBrowser(t, ctx, "#closeRecordModal")

	// Verify '/' keyboard shortcut focuses the search input
	waitForBrowserCondition(t, ctx, "search input exists before shortcut", `document.getElementById('search') !== null`)

	clickBrowser(t, ctx, "#list tbody tr:first-child td:first-child input[type='checkbox']")
	waitForBrowserText(t, ctx, "#selectedCountBadge", "1 selected")
	waitForBrowserEnabled(t, ctx, "#bulkDelete")
	clickBrowser(t, ctx, "#bulkDelete")

	// Confirm the bulk delete in the confirm dialog
	waitForBrowserVisible(t, ctx, "#confirmModal")
	waitForBrowserExists(t, ctx, "#confirmModalConfirm")
	clickBrowser(t, ctx, "#confirmModalConfirm")

	waitForBrowserText(t, ctx, "#status", "Bulk deleted 1 record(s).")
	waitForBrowserText(t, ctx, "#list", "No records matched the current filters.")
	// Toast should appear for successful bulk delete
	waitForBrowserCondition(t, ctx, "bulk delete toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Bulk deleted 1 record(s).");
	})()`)
}

func TestFullExampleAdminPrototypeDarkModeToggle(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#darkModeToggle")

	// By default the page should NOT be in dark mode
	waitForBrowserCondition(t, ctx, "page starts in light mode", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Click the toggle — should enter dark mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "dark mode activated after toggle", `document.documentElement.getAttribute('data-theme') === 'dark'`)

	// Sun icon should be visible, moon icon should be hidden
	waitForBrowserCondition(t, ctx, "sun icon visible in dark mode", `!document.getElementById('darkModeIconSun').hidden`)
	waitForBrowserCondition(t, ctx, "moon icon hidden in dark mode", `document.getElementById('darkModeIconMoon').hidden`)

	// Click again — should return to light mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "light mode restored after second toggle", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Moon icon should be visible, sun icon hidden
	waitForBrowserCondition(t, ctx, "moon icon visible in light mode", `!document.getElementById('darkModeIconMoon').hidden`)
	waitForBrowserCondition(t, ctx, "sun icon hidden in light mode", `document.getElementById('darkModeIconSun').hidden`)
}

func TestFullExampleAdminPrototypeUserRoleMultiSelect(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d body=%s", login.StatusCode, readBody(t, login.Body))
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()

	for _, role := range []map[string]any{
		{"name": "Administrators", "code": "admin", "status": 1, "remark": "full access"},
		{"name": "Editors", "code": "editor", "status": 1, "remark": "content editors"},
	} {
		createRoleResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/roles", role, auth.Token)
		if createRoleResp.StatusCode != http.StatusCreated {
			t.Fatalf("expected role create 201, got %d body=%s", createRoleResp.StatusCode, readBody(t, createRoleResp.Body))
		}
		createRoleResp.Body.Close()
	}

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")

	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserExists(t, ctx, "#createForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "role multiselect dropdown options loaded", `(() => {
		const menu = document.querySelector("#createForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Administrators")) && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Editors"));
	})()`)
	clickBrowser(t, ctx, "#createForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "create multiselect dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		const search = document.querySelector("#createForm .relation-search");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search;
	})()`)

	setBrowserValue(t, ctx, "#createForm textarea[name='name']", "Role User")
	setBrowserValue(t, ctx, "#createForm input[name='email']", "role.user@example.com")
	setBrowserValue(t, ctx, "#createForm input[name='password']", "password123")
	setBrowserValue(t, ctx, "#createForm input[name='age']", "31")
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		if (!dropdown) return "";
		Array.from(dropdown.querySelectorAll(".multi-relation-option")).forEach((option) => {
			const checkbox = option.querySelector("input[type='checkbox']");
			const shouldSelect = option.textContent.includes("Administrators") || option.textContent.includes("Editors");
			if (checkbox && checkbox.checked !== shouldSelect) checkbox.click();
		});
		const select = document.querySelector("#createForm select[name='role_ids']");
		return select ? Array.from(select.selectedOptions).map((option) => option.value).join(",") : "";
	})()`, nil))
	clickBrowser(t, ctx, "#createForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Created a new users record.")
	waitForBrowserText(t, ctx, "#list", "Role User")
	waitForBrowserCondition(t, ctx, "created user visible with role ids", `document.getElementById('list').textContent.includes('Role User')`)

	clickBrowser(t, ctx, "#list tbody tr:last-child .action-btn-view")
	waitForBrowserVisible(t, ctx, "#recordModal")
	waitForBrowserText(t, ctx, "#detailFields", "[1,2]")
	clickBrowser(t, ctx, "#closeRecordModal")

	clickBrowser(t, ctx, "#list tbody tr:last-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	clickBrowser(t, ctx, ".action-menu-list.open .action-menu-item")
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserExists(t, ctx, "#updateForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "update multiselect dropdown options loaded", `(() => {
		const menu = document.querySelector("#updateForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Administrators")) && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Editors"));
	})()`)
	clickBrowser(t, ctx, "#updateForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "update multiselect dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#updateForm details.multi-relation-dropdown");
		const search = document.querySelector("#updateForm .relation-search");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search;
	})()`)
	setBrowserValue(t, ctx, "#updateForm .relation-search", "Admin")
	waitForBrowserCondition(t, ctx, "update multiselect options filtered", `(() => {
		const options = Array.from(document.querySelectorAll("#updateForm .multi-relation-option"));
		return options.length === 1 && options[0].textContent.includes("Administrators");
	})()`)
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const dropdown = document.querySelector("#updateForm details.multi-relation-dropdown");
		if (!dropdown) return "";
		Array.from(dropdown.querySelectorAll(".multi-relation-option")).forEach((option) => {
			const checkbox = option.querySelector("input[type='checkbox']");
			const shouldSelect = option.textContent.includes("Editors");
			if (checkbox && checkbox.checked !== shouldSelect) checkbox.click();
		});
		const select = document.querySelector("#updateForm select[name='role_ids']");
		return select ? Array.from(select.selectedOptions).map((option) => option.value).join(",") : "";
	})()`, nil))
	clickBrowser(t, ctx, "#updateForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Updated record #2.")
	clickBrowser(t, ctx, "#list tbody tr:last-child .action-btn-view")
	waitForBrowserText(t, ctx, "#detailFields", "[2]")
}

func TestFullExampleAdminPrototypeActionMenuPortal(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserCondition(t, ctx, "list table loaded", `document.querySelector('#list table') !== null`)

	// Clicking the ··· trigger should open the portal menu (not inside the table)
	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserCondition(t, ctx, "action menu portal opened", `document.querySelector('#action-menu-portal .action-menu-list.open') !== null`)

	// The portal menu must be a direct child of body (not inside table-shell)
	waitForBrowserCondition(t, ctx, "action menu portal is direct child of body", `
		document.getElementById('action-menu-portal') !== null &&
		document.getElementById('action-menu-portal').parentElement === document.body
	`)

	// Clicking outside should close the portal
	runBrowser(t, ctx, chromedp.Evaluate(`document.body.click()`, nil))
	waitForBrowserCondition(t, ctx, "action menu portal closed after outside click", `document.querySelector('#action-menu-portal .action-menu-list.open') === null`)

	// Pressing Escape after re-opening should also close it
	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserCondition(t, ctx, "action menu portal re-opened", `document.querySelector('#action-menu-portal .action-menu-list.open') !== null`)
	runBrowser(t, ctx, chromedp.KeyEvent("\x1b"))
	waitForBrowserCondition(t, ctx, "action menu portal closed by Escape", `document.querySelector('#action-menu-portal .action-menu-list.open') === null`)
}

func TestFullExampleAdminPrototypeGlobalSearch(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	// Register Alice
	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	// Navigate to admin prototype and sign in via the login form
	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	// Wait for resources to load
	waitForBrowserText(t, ctx, "#resources", "Users")

	// The topbar search input should exist in the page
	waitForBrowserCondition(t, ctx, "topbar search input exists", `!!document.getElementById('topbarSearchInput')`)
	// The results panel should start hidden (no has-results class)
	waitForBrowserCondition(t, ctx, "search results panel initially hidden", `!document.getElementById('topbarSearchResults').classList.contains('has-results')`)

	// Open the search expand by clicking the search toggle
	clickBrowser(t, ctx, "#topbarSearchToggle")
	waitForBrowserCondition(t, ctx, "search expand opens", `document.getElementById('topbarSearchExpand').classList.contains('open')`)
	waitForBrowserVisible(t, ctx, "#topbarSearchInput")

	// Pressing Escape should close the search expand
	runBrowser(t, ctx, chromedp.Focus("#topbarSearchInput"))
	runBrowser(t, ctx, chromedp.KeyEvent("\x1b"))
	waitForBrowserCondition(t, ctx, "search expand closes on Escape", `!document.getElementById('topbarSearchExpand').classList.contains('open')`)
}

func TestFullExampleAdminPrototypeSortableColumns(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	// Register and sign in
	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	// Wait for resources and list to load
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserCondition(t, ctx, "list has at least one row", `document.querySelector('#list table') !== null`)

	// There should be sortable column headers in the Users table
	waitForBrowserCondition(t, ctx, "sortable header exists", `document.querySelector('th.sortable-th') !== null`)

	// Clicking a sortable header should update the sort dropdown to ascending
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))

	// The sort select should now have a non-empty value (ascending sort was applied)
	waitForBrowserCondition(t, ctx, "sort select updated after header click", `document.getElementById('sort').value !== ''`)
	// Wait for list to finish re-rendering before next click
	waitForBrowserCondition(t, ctx, "list table re-rendered after first click", `document.querySelector('#list th.sortable-th') !== null`)

	// Clicking again should switch to descending
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))
	waitForBrowserCondition(t, ctx, "sort select shows descending after second click", `document.getElementById('sort').value.startsWith('-')`)
	// Wait for list to finish re-rendering before next click
	waitForBrowserCondition(t, ctx, "list table re-rendered after second click", `document.querySelector('#list th.sortable-th.sort-desc') !== null`)

	// Clicking a third time should clear the sort (return to default)
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))
	waitForBrowserCondition(t, ctx, "sort select cleared after third click", `document.getElementById('sort').value === ''`)
}

func TestFullExampleStandaloneAdminBrowserRedirectFlow(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserText(t, ctx, "#resources", "Projects")
	waitForBrowserCondition(t, ctx, "#adminShell visible", `(() => {
		const el = document.querySelector("#adminShell");
		return !!el && !el.hidden;
	})()`)
	waitForBrowserCondition(t, ctx, "#loginForm hidden", `(() => {
		const el = document.querySelector("#loginForm");
		return !!el && el.hidden;
	})()`)

	clickBrowser(t, ctx, "#clearToken")
	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserText(t, ctx, "#status", "Signed out of the admin console.")
	waitForBrowserVisible(t, ctx, "#loginForm")
}

func TestFullExampleStandaloneAdminLoginShowsErrorFeedback(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "wrong-password")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserText(t, ctx, "#status", "invalid email or password")
	waitForBrowserCondition(t, ctx, "login error toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("invalid email or password");
	})()`)
}

func TestFullExampleStandaloneAdminDashboardBackNavigation(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")

	clickBrowser(t, ctx, "#sidebarDashboardLink")
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserText(t, ctx, "#dashboardTiles", "Users")
	waitForBrowserCondition(t, ctx, "workspace header hidden on dashboard", `document.getElementById('workspaceHeader').hidden === true`)
	waitForBrowserCondition(t, ctx, "records shell hidden on dashboard", `document.getElementById('recordsShell').hidden === true`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const tiles = Array.from(document.querySelectorAll('#dashboardTiles .dashboard-tile'));
		const tile = tiles.find((item) => String(item.textContent || '').includes('Users'));
		if (tile) tile.click();
		return !!tile;
	})()`, nil))
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserCondition(t, ctx, "workspace header visible for resource", `document.getElementById('workspaceHeader').hidden === false`)
	waitForBrowserCondition(t, ctx, "records shell visible for resource", `document.getElementById('recordsShell').hidden === false`)

	runBrowser(t, ctx, chromedp.Evaluate(`history.back()`, nil))
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserText(t, ctx, "#dashboardTiles", "Users")
	waitForBrowserCondition(t, ctx, "workspace header hidden again on dashboard", `document.getElementById('workspaceHeader').hidden === true`)
	waitForBrowserCondition(t, ctx, "records shell hidden again on dashboard", `document.getElementById('recordsShell').hidden === true`)
}

func TestFullExampleAdminPrototypeLiveSearchScript(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/admin-prototype")
	if err != nil {
		t.Fatalf("GET /admin-prototype: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read /admin-prototype body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected /admin-prototype 200, got %d", resp.StatusCode)
	}

	html := string(body)
	normalizedHTML := compactWhitespace(html)
	for _, needle := range []string{
		"searchTimer: null,",
		"function cancelScheduledSearchReload()",
		"clearTimeout(state.searchTimer);",
		"state.searchTimer = null;",
		"function scheduleSearchReload()",
		"state.searchTimer = setTimeout(() => {",
		"els.search.addEventListener('input', () => {",
	} {
		if !strings.Contains(normalizedHTML, compactWhitespace(needle)) {
			t.Fatalf("expected live-search integration marker %q in html: %q", needle, html)
		}
	}

	for _, needle := range []string{
		"els.clearFilters.onclick = () => { if (!state.current) return; cancelScheduledSearchReload();",
		"els.filtersForm.onsubmit = (event) => { event.preventDefault(); cancelScheduledSearchReload();",
		"els.search.onkeydown = (event) => { if (event.key === 'Enter') { event.preventDefault(); cancelScheduledSearchReload();",
		"els.sort.onchange = () => { if (!state.current) return; cancelScheduledSearchReload();",
		"els.filtersForm.onchange = () => { if (!state.current) return; cancelScheduledSearchReload();",
	} {
		if !strings.Contains(normalizedHTML, compactWhitespace(needle)) {
			t.Fatalf("expected live-search cancellation context %q in html: %q", needle, html)
		}
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

// readBody loads an HTTP response body into a string for test assertions.
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
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
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
