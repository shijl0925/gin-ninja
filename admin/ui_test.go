package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMountUIUsesConfiguredPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		Title:         `Admin <Console>`,
		APIBasePath:   "/custom/api/admin",
		AuthLoginPath: "/custom/api/auth/login",
		AuthMePath:    "/custom/api/auth/me",
		AdminPath:     "/console",
		LoginPath:     "/console/login",
	})

	for _, path := range []string{"/console", "/console/login"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body=%s", path, w.Code, w.Body.String())
		}
		body := w.Body.String()
		for _, snippet := range []string{
			"<title>Admin &lt;Console&gt;</title>",
			`const apiBase = "/custom/api/admin";`,
			`const authMePath = "/custom/api/auth/me";`,
			`const adminPagePath = "/console";`,
			`const adminLoginPath = "/console/login";`,
			`<a class="topbar-link nav-link" href="/console"><span>Home</span></a>`,
			`<a id="topbarBreadcrumbRoot" class="topbar-breadcrumb-root" href="/console">Dashboard</a>`,
			`await request("/custom/api/auth/login", {`,
			`Paste a token from /custom/api/auth/login`,
			// default extract expressions
			`const loginTokenExtractExpr = "payload.token";`,
			`const loginNameExtractExpr = "payload.name";`,
			`const loginUserIDExtractExpr = "payload.user_id || payload.userID";`,
		} {
			if !strings.Contains(body, snippet) {
				t.Fatalf("GET %s missing %q", path, snippet)
			}
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/console/users?search=t&page=1&size=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /console/users status = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `const adminPagePath = "/console";`) {
		t.Fatal("resource route did not render admin UI shell")
	}

	req = httptest.NewRequest(http.MethodGet, "/missing", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /missing status = %d, want 404", w.Code)
	}
}

func TestMountUICustomTokenExtract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath:           "/admin",
		LoginPath:           "/admin/login",
		TokenExtractExpr:    "payload.data && payload.data.accessToken",
		UserNameExtractExpr: "payload.data && payload.data.userName",
		UserIDExtractExpr:   "payload.data && payload.data.id",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d", w.Code)
	}
	body := w.Body.String()
	for _, snippet := range []string{
		`const loginTokenExtractExpr = "payload.data \u0026\u0026 payload.data.accessToken";`,
		`const loginNameExtractExpr = "payload.data \u0026\u0026 payload.data.userName";`,
		`const loginUserIDExtractExpr = "payload.data \u0026\u0026 payload.data.id";`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("GET /admin missing %q", snippet)
		}
	}
}

func TestRenderUIHTMLAssemblesSplitAssets(t *testing.T) {
	body := renderUIHTML(DefaultUIConfig())
	for _, snippet := range []string{
		`<style>`,
		`:root {`,
		`<script>`,
		`const apiBase = "/api/v1/admin";`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("rendered UI missing assembled asset snippet %q", snippet)
		}
	}
	for _, placeholder := range []string{
		"__GIN_NINJA_ADMIN_INLINE_CSS__",
		"__GIN_NINJA_ADMIN_INLINE_JS__",
	} {
		if strings.Contains(body, placeholder) {
			t.Fatalf("rendered UI still contains asset placeholder %q", placeholder)
		}
	}
}

func TestMountUIUsesBrandThemeAndStorageOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		Title:        "Ops Desk",
		BrandName:    "Acme Ops",
		LogoText:     "ACME",
		Locale:       "zh-CN",
		DefaultTheme: "system",
		TokenStorage: "session",
		AdminPath:    "/admin",
		LoginPath:    "/admin/login",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d", w.Code)
	}
	body := w.Body.String()
	for _, snippet := range []string{
		`<html lang="zh-CN">`,
		`const adminTitle = "Ops Desk";`,
		`const adminBrandName = "Acme Ops";`,
		`const adminLogoText = "ACM";`,
		`const adminLocale = "zh-CN";`,
		`const adminDefaultTheme = "system";`,
		`const tokenStorageDriver = "session";`,
		`<strong>Acme Ops</strong>`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("GET /admin missing %q", snippet)
		}
	}
}

func TestMountUIDeduplicatesPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath: "/admin",
		LoginPath: "/admin",
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestMountUIEscapesExtractorExpressionsAsData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	MountUI(router, UIConfig{
		AdminPath:        "/admin",
		LoginPath:        "/admin/login",
		TokenExtractExpr: `payload.token"; window.evil = true; "`,
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin status = %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, `const loginTokenExtractExpr = "payload.token"; window.evil = true; "";`) {
		t.Fatalf("extractor expression was injected as executable JavaScript: %s", body)
	}
	if !strings.Contains(body, `const loginTokenExtractExpr = "payload.token\"; window.evil = true; \"";`) {
		t.Fatal("expected extractor expression to be JSON string data")
	}
}

func TestServeDefaultUI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin", nil)

	ServeDefaultUI(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	body := w.Body.String()
	for _, snippet := range []string{
		"Gin Ninja Admin",
		`const apiBase = "/api/v1/admin";`,
		`id="columnToggle"`,
		`id="columnMenu"`,
		`function updateColumnToggleLabel`,
		`function saveColumnVisibility`,
		`Toggle columns`,
		`column-option-check`,
		`const authIdentityStorageKey = 'gin-ninja-admin-auth-identity';`,
		`function persistAuthIdentity`,
		`function restoreAuthIdentity`,
		`/resources/stats`,
		`/search?q=`,
		`placeholder="Search"`,
		`function globalResourceSearch`,
		`Records / `,
		`function moveGlobalSearchSelection`,
		`function openActiveGlobalSearchResult`,
		`id="copyRecordJSON"`,
		`function copySelectedRecordJSON`,
		`detail-value`,
		`const listStateStoragePrefix`,
		`function applySavedListState`,
		`id="clearSelection"`,
		`id="copySelectedIDs"`,
		`id="bulkActionBar"`,
		`id="bulkActionLabel"`,
		`id="bulkActionExport"`,
		`id="bulkActionDelete"`,
		`id="errorShell"`,
		`Unauthorized Access`,
		`Access Forbidden`,
		`Oops! Page Not Found!`,
		`Oops! Something went wrong :')`,
		`Website is under maintenance!`,
		`function copyBulkSelectedIDs`,
		`function exportSelectedRecordsCSV`,
		`function bulkDeleteSelectedRecords`,
		`function clearBulkSelection`,
		`function syncBulkRowSelection`,
		`bulk-action-bar`,
		`row-bulk-selected`,
		`function renderListEmptyState`,
		`empty-state-actions`,
		`function openModalAndFocus`,
		`function focusFirstFormError`,
		`id="resourceActionSummary"`,
		`function renderResourceActionSummary`,
		`function formatFieldDisplay`,
		`function formatRelativeTime`,
		`function paginationSummaryText`,
		`Updated `,
		`function rewindPageIfCurrentPageEmptied`,
		`function buildValidatedFormPayload`,
		`data-form-status`,
		`Unsaved changes`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("default UI body missing %q: %s", snippet, body)
		}
	}
	for _, snippet := range []string{
		`Open the user workspace to review profiles, roles, and sign-in ready records.`,
		`Open this admin resource to review records, filters, and available actions.`,
		`Browse, inspect, and edit `,
		`Dashboard cards`,
		`Connected admin workspace`,
		`dashboard-tile-hint`,
		`dashboard-tile-meta`,
		`detail-value copyable`,
		`function copyDetailFieldValue`,
		`Copy value`,
	} {
		if strings.Contains(body, snippet) {
			t.Fatalf("default UI should not contain framework-authored resource description %q", snippet)
		}
	}
}
