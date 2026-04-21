package ninja

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shijl0925/gin-ninja/settings"
)

func TestSanitizeStartupDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "mysql raw dsn",
			dsn:  "user:secret@tcp(localhost:3306)/demo?charset=utf8mb4",
			want: "user:xxxxx@tcp(localhost:3306)/demo?charset=utf8mb4",
		},
		{
			name: "postgres url dsn",
			dsn:  "postgres://app:secret@localhost:5432/demo?sslmode=disable",
			want: "postgres://app:xxxxx@localhost:5432/demo?sslmode=disable",
		},
		{
			name: "postgres keyword dsn",
			dsn:  "host=localhost port=5432 user=app password='secret value' dbname=demo sslmode=disable",
			want: "host=localhost port=5432 user=app password='xxxxx' dbname=demo sslmode=disable",
		},
		{
			name: "query password redacted",
			dsn:  "postgres://app@localhost:5432/demo?password=secret&sslmode=disable",
			want: "postgres://app@localhost:5432/demo?password=xxxxx&sslmode=disable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeStartupDSN(tc.dsn); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestStartupDSNStructuredConfigOmitsPasswords(t *testing.T) {
	mysqlDSN := startupDSN(settings.DatabaseConfig{
		Driver: "mysql",
		MySQL: settings.MySQLConfig{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "demo",
			Password: "secret",
			Name:     "app",
		},
	})
	if mysqlDSN != "demo@tcp(127.0.0.1:3306)/app" {
		t.Fatalf("unexpected mysql dsn %q", mysqlDSN)
	}
	if strings.Contains(mysqlDSN, "secret") {
		t.Fatalf("mysql dsn leaked password: %q", mysqlDSN)
	}

	postgresDSN := startupDSN(settings.DatabaseConfig{
		Driver: "postgres",
		Postgres: settings.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "demo",
			Password: "secret",
			Name:     "app",
			SSLMode:  "disable",
		},
	})
	if postgresDSN != "host=127.0.0.1 port=5432 dbname=app user=demo sslmode=disable" {
		t.Fatalf("unexpected postgres dsn %q", postgresDSN)
	}
	if strings.Contains(postgresDSN, "secret") {
		t.Fatalf("postgres dsn leaked password: %q", postgresDSN)
	}
}

func TestServe_PrintsStartupBanner(t *testing.T) {
	prev := settings.GetGlobal()
	t.Cleanup(func() { settings.SetGlobal(prev) })
	settings.SetGlobal(settings.Config{
		App: settings.AppConfig{
			Env: "demo",
		},
		Database: settings.DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://app:secret@localhost:5432/demo?sslmode=disable",
		},
	})

	api := New(Config{Title: "Banner Test", Version: "1.0.0-test", DisableGinDefault: true})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	done := make(chan error, 1)
	go func() {
		done <- api.Serve(listener)
	}()

	if err := waitForServer(listener.Addr().String()); err != nil {
		t.Fatalf("wait for server: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := api.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("serve: %v", err)
	}

	_ = w.Close()
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	banner := string(output)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}

	for _, want := range []string{
		"dsn: postgres://app:xxxxx@localhost:5432/demo?sslmode=disable",
		"port: " + port,
		"env: demo",
		"version: 1.0.0-test",
		" ██████╗ ██╗███╗   ██╗",
	} {
		if !strings.Contains(banner, want) {
			t.Fatalf("expected banner to contain %q, got %q", want, banner)
		}
	}
	if strings.Contains(banner, "secret") {
		t.Fatalf("banner leaked secret: %q", banner)
	}
}

func waitForServer(addr string) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return context.DeadlineExceeded
}
