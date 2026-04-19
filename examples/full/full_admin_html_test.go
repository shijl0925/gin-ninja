package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFullExampleAdminPrototypeAndProjectSelectors(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	fetchHTML := func(path string) string {
		t.Helper()
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read %s body: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
		return string(body)
	}

	loginHTML := fetchHTML("/admin/login")
	if !strings.Contains(loginHTML, `id="loginForm"`) {
		t.Fatalf("expected login form in /admin/login html")
	}

	for _, path := range []string{"/admin", "/admin-prototype"} {
		html := fetchHTML(path)
		for _, marker := range []string{
			`id="loginForm"`,
			`id="resources"`,
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
