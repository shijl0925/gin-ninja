package bootstrap

import (
	drivermysql "github.com/go-sql-driver/mysql"
	"net/url"
	"strings"
	"testing"

	applogger "github.com/shijl0925/gin-ninja/pkg/logger"
	"github.com/shijl0925/gin-ninja/settings"
	gormmysql "gorm.io/driver/mysql"
	driverpg "gorm.io/driver/postgres"
)

func TestBuildDialector(t *testing.T) {
	cases := []struct {
		name    string
		cfg     settings.DatabaseConfig
		wantErr bool
	}{
		{name: "sqlite", cfg: settings.DatabaseConfig{Driver: "sqlite", DSN: "file::memory:?cache=shared"}},
		{name: "sqlite3", cfg: settings.DatabaseConfig{Driver: "sqlite3", DSN: "file::memory:?cache=shared"}},
		{name: "mysql", cfg: settings.DatabaseConfig{Driver: "mysql", DSN: "user:pass@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"}},
		{name: "mysql structured", cfg: settings.DatabaseConfig{Driver: "mysql", MySQL: settings.MySQLConfig{Host: "localhost", User: "user", Password: "p@ss:word+plus", Name: "app", Charset: "utf8mb4", ParseTime: true, Loc: "Local"}}},
		{name: "postgres", cfg: settings.DatabaseConfig{Driver: "postgres", DSN: "host=localhost user=postgres password=postgres dbname=app port=5432 sslmode=disable TimeZone=Asia/Shanghai"}},
		{name: "postgres structured", cfg: settings.DatabaseConfig{Driver: "postgres", Postgres: settings.PostgresConfig{Host: "localhost", User: "postgres", Password: "p@ss word", Name: "app", SSLMode: "disable", TimeZone: "Asia/Shanghai"}}},
		{name: "unsupported", cfg: settings.DatabaseConfig{Driver: "oracle", DSN: "dsn"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dialector, err := buildDialector(&tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || dialector == nil {
				t.Fatalf("expected dialector, got dialector=%v err=%v", dialector, err)
			}
			if tc.name == "mysql structured" {
				mysqlDial, ok := dialector.(*gormmysql.Dialector)
				if !ok {
					t.Fatalf("expected *mysql.Dialector, got %T", dialector)
				}
				parsed, err := drivermysql.ParseDSN(mysqlDial.DSN)
				if err != nil {
					t.Fatalf("ParseDSN: %v", err)
				}
				if parsed.Passwd != "p@ss:word+plus" {
					t.Fatalf("expected structured mysql password to round-trip, got %q", parsed.Passwd)
				}
			}
		})
	}
}

func TestSQLiteDialectorRequiresDSN(t *testing.T) {
	if _, err := sqliteDialector(""); err == nil {
		t.Fatal("expected sqlite dsn validation error")
	}
}

func TestMySQLDialectorRequiresDSN(t *testing.T) {
	if _, err := mysqlDialector(settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected mysql dsn validation error")
	}
}

func TestMySQLDialectorHandlesBoundaryDSNs(t *testing.T) {
	cases := []struct {
		name    string
		cfg     settings.DatabaseConfig
		wantDSN string
		verify  func(*testing.T, string)
		wantErr string
	}{
		{
			name:    "encoded reserved characters",
			cfg:     settings.DatabaseConfig{DSN: "root:p%40ss%3Aword@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local"},
			wantDSN: "root:p@ss:word@tcp(127.0.0.1:3306)/gin_ninja?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name:    "plus sign stays plus",
			cfg:     settings.DatabaseConfig{DSN: "root:p%2Bss@tcp(127.0.0.1:3306)/gin_ninja?loc=UTC+8"},
			wantDSN: "root:p+ss@tcp(127.0.0.1:3306)/gin_ninja?loc=UTC+8",
		},
		{
			name: "structured config builds escaped password",
			cfg:  settings.DatabaseConfig{MySQL: settings.MySQLConfig{Host: "127.0.0.1", User: "root", Password: "p@ss:word+plus", Name: "gin_ninja", Charset: "utf8mb4", ParseTime: true, Loc: "Local"}},
			verify: func(t *testing.T, dsn string) {
				t.Helper()
				parsed, err := drivermysql.ParseDSN(dsn)
				if err != nil {
					t.Fatalf("ParseDSN: %v", err)
				}
				if parsed.User != "root" || parsed.Passwd != "p@ss:word+plus" {
					t.Fatalf("unexpected credentials in DSN %q", dsn)
				}
				if parsed.Addr != "127.0.0.1:3306" || parsed.DBName != "gin_ninja" {
					t.Fatalf("unexpected address/db in DSN %q", dsn)
				}
				if !parsed.ParseTime || parsed.Params["charset"] != "utf8mb4" || parsed.Loc.String() != "Local" {
					t.Fatalf("unexpected mysql options in DSN %q", dsn)
				}
			},
		},
		{
			name: "structured config preserves at sign password",
			cfg:  settings.DatabaseConfig{MySQL: settings.MySQLConfig{Host: "127.0.0.1", User: "root", Password: "root@123", Name: "gin_ninja", Charset: "utf8mb4", ParseTime: true, Loc: "Local"}},
			verify: func(t *testing.T, dsn string) {
				t.Helper()
				parsed, err := drivermysql.ParseDSN(dsn)
				if err != nil {
					t.Fatalf("ParseDSN: %v", err)
				}
				if parsed.Passwd != "root@123" {
					t.Fatalf("expected password to round-trip, got %q", parsed.Passwd)
				}
			},
		},
		{
			name: "structured config wins over default sqlite dsn",
			cfg:  settings.DatabaseConfig{Driver: "mysql", DSN: "app.db", MySQL: settings.MySQLConfig{Host: "127.0.0.1", User: "root", Password: "root@123", Name: "gin_ninja", Charset: "utf8mb4", ParseTime: true, Loc: "Local"}},
			verify: func(t *testing.T, dsn string) {
				t.Helper()
				parsed, err := drivermysql.ParseDSN(dsn)
				if err != nil {
					t.Fatalf("ParseDSN: %v", err)
				}
				if parsed.Net != "tcp" || parsed.DBName != "gin_ninja" || parsed.Passwd != "root@123" {
					t.Fatalf("expected structured mysql config to override default sqlite dsn, got %q", dsn)
				}
			},
		},
		{
			name:    "bad escape",
			cfg:     settings.DatabaseConfig{DSN: "root:bad%zz@tcp(localhost:3306)/app"},
			wantErr: "decode mysql DSN",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dialector, err := mysqlDialector(tc.cfg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("mysqlDialector: %v", err)
			}

			mysqlDial, ok := dialector.(*gormmysql.Dialector)
			if !ok {
				t.Fatalf("expected *mysql.Dialector, got %T", dialector)
			}
			if tc.wantDSN != "" && mysqlDial.DSN != tc.wantDSN {
				t.Fatalf("expected DSN %q, got %q", tc.wantDSN, mysqlDial.DSN)
			}
			if tc.verify != nil {
				tc.verify(t, mysqlDial.DSN)
			}
		})
	}
}

func TestPostgresDialectorRequiresDSN(t *testing.T) {
	if _, err := postgresDialector(settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected postgres dsn validation error")
	}
}

func TestPostgresDialectorBuildsStructuredDSN(t *testing.T) {
	dialector, err := postgresDialector(settings.DatabaseConfig{
		Postgres: settings.PostgresConfig{
			Host:     "localhost",
			User:     "postgres",
			Password: "p@ss word",
			Name:     "gin_ninja",
			SSLMode:  "disable",
			TimeZone: "Asia/Shanghai",
		},
	})
	if err != nil {
		t.Fatalf("postgresDialector: %v", err)
	}

	pgDial, ok := dialector.(*driverpg.Dialector)
	if !ok {
		t.Fatalf("expected *postgres.Dialector, got %T", dialector)
	}
	parsed, err := url.Parse(pgDial.DSN)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	if parsed.Scheme != "postgres" || parsed.Host != "localhost:5432" || parsed.Path != "/gin_ninja" {
		t.Fatalf("unexpected postgres dsn: %q", pgDial.DSN)
	}
	if user := parsed.User.Username(); user != "postgres" {
		t.Fatalf("expected postgres user, got %q", user)
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss word" {
		t.Fatalf("expected postgres password to round-trip, got %q ok=%v", password, ok)
	}
	query := parsed.Query()
	if query.Get("sslmode") != "disable" || query.Get("TimeZone") != "Asia/Shanghai" {
		t.Fatalf("unexpected postgres query params: %v", query)
	}
}

func TestInitDBAndMustInitDB(t *testing.T) {
	cfg := &settings.DatabaseConfig{
		Driver:                 "sqlite",
		DSN:                    "file::memory:?cache=shared",
		MaxIdleConns:           1,
		MaxOpenConns:           2,
		ConnMaxLifetimeMinutes: 1,
	}

	db, err := InitDB(cfg)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("expected query to succeed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 2 {
		t.Fatalf("expected max open connections to be set, got %d", sqlDB.Stats().MaxOpenConnections)
	}

	if MustInitDB(cfg) == nil {
		t.Fatal("expected MustInitDB to return db")
	}
}

func TestMustInitDBPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustInitDB(&settings.DatabaseConfig{Driver: "sqlite"})
}

func TestInitLoggerSetsGlobal(t *testing.T) {
	cfg := &settings.LogConfig{Level: "debug", Format: "json", Output: "stdout"}
	logger := InitLogger(cfg)
	if logger == nil {
		t.Fatal("expected logger")
	}
	if applogger.Global() != logger {
		t.Fatal("expected InitLogger to replace global logger")
	}
}
