package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestFullExampleAdminPrototypeAndProjectSelectors(t *testing.T) {
	client := newFullTestClient(t)

	fetchHTML := func(path string) string {
		t.Helper()
		resp := client.Get(path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
		return resp.String()
	}

	loginHTML := fetchHTML("/admin/login")
	if !strings.Contains(loginHTML, `id="loginForm"`) {
		t.Fatalf("expected login form in /admin/login html")
	}

	for _, path := range []string{"/admin", "/admin"} {
		html := fetchHTML(path)
		for _, marker := range []string{
			`id="loginForm"`,
			`id="resources"`,
			`id="tableDensity"`,
			`id="columnToggle"`,
			`id="columnMenu"`,
			`id="toggleFilters"`,
			`id="openCreateModal"`,
			`id="createModal"`,
			`id="toastContainer"`,
		} {
			if !strings.Contains(html, marker) {
				t.Fatalf("%s: expected marker %q in html", path, marker)
			}
		}
	}
}
