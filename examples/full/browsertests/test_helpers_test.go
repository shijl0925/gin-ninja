package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/shijl0925/gin-ninja/bootstrap"
	_ "github.com/shijl0925/gin-ninja/bootstrap/drivers/sqlite"
	"github.com/shijl0925/gin-ninja/examples/internal/fullapp"
	"github.com/shijl0925/gin-ninja/settings"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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
	db, err := fullapp.InitDB(&cfg.Database)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}

	return httptest.NewServer(fullapp.BuildAPI(cfg, db, log, fullapp.FullOptions()).Handler())
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

func readBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
