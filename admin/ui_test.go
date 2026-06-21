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
			`const adminPagePath = "/console";`,
			`const adminLoginPath = "/console/login";`,
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
		`id="tableDensity"`,
		`id="columnToggle"`,
		`id="columnMenu"`,
		`function updateColumnToggleLabel`,
		`function clearSavedColumnVisibility`,
		`column-menu-actions`,
		`const authIdentityStorageKey = 'gin-ninja-admin-auth-identity';`,
		`function persistAuthIdentity`,
		`function restoreAuthIdentity`,
		`id="toggleFilters"`,
		`const filtersCollapsedStorageKey = 'gin-ninja-admin-filters-collapsed';`,
		`/resources/stats`,
		`/search?q=`,
		`id="exportList"`,
		`/export`,
		`function buildExportQuery`,
		`id="copyRecordJSON"`,
		`function copySelectedRecordJSON`,
		`id="activeListState"`,
		`function renderActiveListState`,
		`const listStateStoragePrefix`,
		`function applySavedListState`,
		`id="savedViewSelect"`,
		`const savedViewsStoragePrefix`,
		`function saveCurrentViewPreset`,
		`function applySavedViewByID`,
		`id="clearSelection"`,
		`id="copySelectedIDs"`,
		`function copyBulkSelectedIDs`,
		`function clearBulkSelection`,
		`function renderListEmptyState`,
		`empty-state-actions`,
		`function openModalAndFocus`,
		`function focusFirstFormError`,
		`function isInteractiveTableTarget`,
		`role', 'button'`,
		`id="copyViewLink"`,
		`function copyCurrentViewLink`,
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
}
