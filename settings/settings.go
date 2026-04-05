// Package settings provides Viper-based configuration management for
// gin-ninja applications.
//
// The configuration can be loaded from a YAML file and overridden by
// environment variables.  A typical config.yaml looks like:
//
//	app:
//	  name: "My API"
//	  env: "development"
//	  debug: true
//
//	server:
//	  host: "0.0.0.0"
//	  port: 8080
//	  read_timeout:  60
//	  write_timeout: 60
//
//	database:
//	  driver: "sqlite"
//	  dsn: "app.db"
//	  max_idle_conns: 10
//	  max_open_conns: 100
//
//	jwt:
//	  secret: "change-me-in-production"
//	  expire_hours: 24
//	  issuer: "gin-ninja"
//
//	log:
//	  level: "info"
//	  format: "json"
//	  output: "stdout"
package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root application configuration.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig holds general application metadata.
type AppConfig struct {
	// Name is the application name (used in OpenAPI info).
	Name string `mapstructure:"name"`
	// Env is the runtime environment ("development", "staging", "production").
	Env string `mapstructure:"env"`
	// Debug enables debug mode in Gin when true.
	Debug bool `mapstructure:"debug"`
	// Version is the application version string.
	Version string `mapstructure:"version"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	// Host is the listen address (default: "0.0.0.0").
	Host string `mapstructure:"host"`
	// Port is the listen port (default: 8080).
	Port int `mapstructure:"port"`
	// ReadTimeout is the maximum duration for reading the full request (seconds).
	ReadTimeout int `mapstructure:"read_timeout"`
	// WriteTimeout is the maximum duration for writing the response (seconds).
	WriteTimeout int `mapstructure:"write_timeout"`
}

// Addr returns the host:port string, e.g. "0.0.0.0:8080".
func (s ServerConfig) Addr() string {
	host := s.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := s.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// ReadTimeoutDuration returns the read timeout as a time.Duration.
func (s ServerConfig) ReadTimeoutDuration() time.Duration {
	if s.ReadTimeout <= 0 {
		return 60 * time.Second
	}
	return time.Duration(s.ReadTimeout) * time.Second
}

// WriteTimeoutDuration returns the write timeout as a time.Duration.
func (s ServerConfig) WriteTimeoutDuration() time.Duration {
	if s.WriteTimeout <= 0 {
		return 60 * time.Second
	}
	return time.Duration(s.WriteTimeout) * time.Second
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	// Driver is the GORM driver name ("sqlite", "mysql", "postgres", "sqlserver").
	Driver string `mapstructure:"driver"`
	// DSN is the data source name / connection string.
	DSN string `mapstructure:"dsn"`
	// MySQL optionally defines structured MySQL connection settings.
	MySQL MySQLConfig `mapstructure:"mysql"`
	// Postgres optionally defines structured PostgreSQL connection settings.
	Postgres PostgresConfig `mapstructure:"postgres"`
	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int `mapstructure:"max_idle_conns"`
	// MaxOpenConns is the maximum number of open connections in the pool.
	MaxOpenConns int `mapstructure:"max_open_conns"`
	// ConnMaxLifetimeMinutes is the maximum lifetime of a connection (minutes).
	ConnMaxLifetimeMinutes int `mapstructure:"conn_max_lifetime_minutes"`
}

// MySQLConfig holds structured MySQL connection settings.
type MySQLConfig struct {
	Host      string            `mapstructure:"host"`
	Port      int               `mapstructure:"port"`
	User      string            `mapstructure:"user"`
	Password  string            `mapstructure:"password"`
	Name      string            `mapstructure:"name"`
	Charset   string            `mapstructure:"charset"`
	ParseTime bool              `mapstructure:"parse_time"`
	Loc       string            `mapstructure:"loc"`
	Params    map[string]string `mapstructure:"params"`
}

// PostgresConfig holds structured PostgreSQL connection settings.
type PostgresConfig struct {
	Host     string            `mapstructure:"host"`
	Port     int               `mapstructure:"port"`
	User     string            `mapstructure:"user"`
	Password string            `mapstructure:"password"`
	Name     string            `mapstructure:"name"`
	SSLMode  string            `mapstructure:"sslmode"`
	TimeZone string            `mapstructure:"time_zone"`
	Params   map[string]string `mapstructure:"params"`
}

// JWTConfig holds JSON Web Token settings.
type JWTConfig struct {
	// Secret is the HMAC signing secret.  Must be set in production.
	Secret string `mapstructure:"secret"`
	// ExpireHours is the token lifetime in hours (default: 24).
	ExpireHours int `mapstructure:"expire_hours"`
	// Issuer is the "iss" claim value.
	Issuer string `mapstructure:"issuer"`
}

// ExpireDuration returns the token TTL as a time.Duration.
func (j JWTConfig) ExpireDuration() time.Duration {
	h := j.ExpireHours
	if h <= 0 {
		h = 24
	}
	return time.Duration(h) * time.Hour
}

// LogConfig holds logging settings.
type LogConfig struct {
	// Level is the minimum log level ("debug", "info", "warn", "error").
	Level string `mapstructure:"level"`
	// Format is the log format ("json" or "console").
	Format string `mapstructure:"format"`
	// Output is where logs are written ("stdout", "stderr", or a file path).
	Output string `mapstructure:"output"`
}

// Global holds the application-wide configuration loaded by Load.
var Global Config

// Load reads configuration from the given file path and merges in
// environment variables.  Environment variables are mapped using the
// pattern APP__SERVER__PORT → app.server.port (double underscore separator).
//
// If cfgFile is empty, Load searches for a file named "config" in the
// current working directory and common config directories.
func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Set defaults.
	setDefaults(v)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/app")
	}

	// Read the config file.
	if err := v.ReadInConfig(); err != nil {
		// It is fine if the file is absent when cfgFile was not explicitly set.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("settings: read config: %w", err)
		}
	}

	// Allow environment variables to override, using double underscore as
	// the key delimiter so APP__SERVER__PORT maps to app.server.port.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("settings: unmarshal: %w", err)
	}

	Global = cfg
	return &cfg, nil
}

// MustLoad calls Load and panics on error.  Useful in main() where
// a missing config should be a fatal error.
func MustLoad(cfgFile string) *Config {
	cfg, err := Load(cfgFile)
	if err != nil {
		panic(fmt.Sprintf("settings: MustLoad: %v", err))
	}
	return cfg
}

// setDefaults registers sensible defaults into the viper instance.
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "Gin Ninja App")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.debug", true)
	v.SetDefault("app.version", "1.0.0")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 60)
	v.SetDefault("server.write_timeout", 60)

	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "app.db")
	v.SetDefault("database.mysql.port", 3306)
	v.SetDefault("database.mysql.charset", "utf8mb4")
	v.SetDefault("database.mysql.parse_time", true)
	v.SetDefault("database.mysql.loc", "Local")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.sslmode", "disable")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime_minutes", 60)

	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expire_hours", 24)
	v.SetDefault("jwt.issuer", "gin-ninja")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}
