package fullapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/settings"
	"go.uber.org/zap"
)

func testConfig(dsn string) settings.Config {
	return settings.Config{
		App: settings.AppConfig{
			Name:    "fullapp-test",
			Version: "1.0.0",
		},
		Server: settings.ServerConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},
		Database: settings.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
		JWT: settings.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 24,
			Issuer:      "gin-ninja",
		},
	}
}

func doFullappRequest(api *ninja.NinjaAPI, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, req)
	return w
}

func TestFullappOptionsAndHelpers(t *testing.T) {
	if !FullOptions().IncludeAdminPages || !UsersOptions().IncludeUsersV2 || !FeaturesOptions().IncludeFeatureDemos || !AdminOptions().IncludeAdminAPI {
		t.Fatal("expected option constructors to enable their documented features")
	}

	if clone := cloneVersions(nil); clone != nil {
		t.Fatalf("expected nil clone for nil versions, got %v", clone)
	}
	versions := cloneVersions(FullOptions().Versions)
	versions["v1"] = ninja.VersionConfig{Prefix: "/mutated"}
	if FullOptions().Versions["v1"].Prefix == "/mutated" {
		t.Fatal("expected cloneVersions to return an independent copy")
	}

	cfg := testConfig("full_example.db")
	normalizeConfigPaths(filepath.Join("/tmp", "config.yaml"), &cfg)
	if want := filepath.Join("/tmp", "full_example.db"); cfg.Database.DSN != want {
		t.Fatalf("expected normalized sqlite dsn %q, got %q", want, cfg.Database.DSN)
	}
	cfg.Database.DSN = "file:embedded.db?mode=memory&cache=shared"
	normalizeConfigPaths(filepath.Join("/tmp", "config.yaml"), &cfg)
	if cfg.Database.DSN != "file:embedded.db?mode=memory&cache=shared" {
		t.Fatalf("expected file: DSN to remain unchanged, got %q", cfg.Database.DSN)
	}
	cfg.Database.DSN = filepath.Join("/tmp", "absolute.db")
	normalizeConfigPaths(filepath.Join("/tmp", "config.yaml"), &cfg)
	if cfg.Database.DSN != filepath.Join("/tmp", "absolute.db") {
		t.Fatalf("expected absolute DSN to remain unchanged, got %q", cfg.Database.DSN)
	}
	normalizeConfigPaths(filepath.Join("/tmp", "config.yaml"), nil)
}

func TestFullappInitCacheStoreCoverage(t *testing.T) {
	cfg := testConfig("file:cache?mode=memory&cache=shared")

	store, shutdown := initCacheStore(cfg)
	if _, ok := store.(*ninja.MemoryCacheStore); !ok {
		t.Fatalf("expected memory cache store by default, got %T", store)
	}
	if shutdown != nil {
		t.Fatalf("expected nil shutdown for default memory store, got nil=%t", shutdown == nil)
	}

	redisServer := miniredis.RunT(t)
	cfg.Redis = settings.RedisConfig{
		Enabled: true,
		Addr:    redisServer.Addr(),
		Prefix:  "fullapp:",
	}
	store, shutdown = initCacheStore(cfg)
	if _, ok := store.(*ninja.RedisCacheStore); !ok {
		t.Fatalf("expected redis cache store, got %T", store)
	}
	if shutdown == nil {
		t.Fatal("expected redis shutdown hook")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown(): %v", err)
	}

	cfg.Redis.Addr = ""
	store, shutdown = initCacheStore(cfg)
	if _, ok := store.(*ninja.MemoryCacheStore); !ok {
		t.Fatalf("expected memory fallback store, got %T", store)
	}
	if shutdown != nil {
		t.Fatalf("expected nil shutdown after fallback, got nil=%t", shutdown == nil)
	}

	cfg.Redis.Addr = "127.0.0.1:1"
	store, shutdown = initCacheStore(cfg)
	if _, ok := store.(*ninja.MemoryCacheStore); !ok {
		t.Fatalf("expected ping failure fallback store, got %T", store)
	}
	if shutdown != nil {
		t.Fatalf("expected nil shutdown after ping failure fallback, got nil=%t", shutdown == nil)
	}
}

func TestFullappBuildAPIAndRunCoverage(t *testing.T) {
	cfg := testConfig("file:fullapp-build?mode=memory&cache=shared")
	db, err := InitDB(&cfg.Database)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	api := BuildAPI(cfg, db, zap.NewNop(), FullOptions())
	for _, tc := range []struct {
		path string
		want int
	}{
		{path: "/health", want: http.StatusOK},
		{path: "/admin", want: http.StatusOK},
		{path: "/admin/login", want: http.StatusOK},
		{path: "/admin-prototype", want: http.StatusOK},
		{path: "/openapi/v0.json", want: http.StatusOK},
		{path: "/openapi/v1.json", want: http.StatusOK},
		{path: "/openapi/v2.json", want: http.StatusOK},
		{path: "/api/v1/examples/hidden", want: http.StatusOK},
		{path: "/api/v2/users/", want: http.StatusUnauthorized},
	} {
		w := doFullappRequest(api, http.MethodGet, tc.path)
		if w.Code != tc.want {
			t.Fatalf("%s status = %d, want %d body=%s", tc.path, w.Code, tc.want, w.Body.String())
		}
	}

	runCfg := testConfig("file:fullapp-run?mode=memory&cache=shared")
	runCfg.Server.Port = -1
	if err := Run(runCfg, zap.NewNop(), Options{
		Description: "run failure coverage",
		Versions: map[string]ninja.VersionConfig{
			"v1": {Prefix: "/v1"},
		},
		IncludeHealth: true,
	}); err == nil {
		t.Fatal("expected Run() to fail when binding the default occupied address is impossible")
	}

	if _, err := InitDB(&settings.DatabaseConfig{Driver: "oracle"}); err == nil {
		t.Fatal("expected InitDB to fail for missing driver registration")
	}
}

func TestMustLoadConfigNormalizesSQLitePaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`
app:
  name: "fullapp"
  version: "1.0.0"
database:
  driver: "sqlite"
  dsn: "fullapp.db"
`), 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	cfg := MustLoadConfig(configPath)
	if cfg.Database.DSN != filepath.Join(dir, "fullapp.db") {
		t.Fatalf("expected normalized config dsn, got %q", cfg.Database.DSN)
	}
}
