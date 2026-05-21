package settings

import ninja "github.com/shijl0925/gin-ninja"

// StartupConfig returns the lightweight core startup banner metadata for cfg.
func StartupConfig(cfg Config) ninja.StartupConfig {
	return ninja.StartupConfig{
		Env:     cfg.App.Env,
		Version: cfg.App.Version,
		Server:  ninja.StartupServerConfig{Port: cfg.Server.Port},
		Database: ninja.StartupDatabaseConfig{
			Driver: cfg.Database.Driver,
			DSN:    cfg.Database.DSN,
			MySQL: ninja.StartupMySQLConfig{
				Host: cfg.Database.MySQL.Host,
				Port: cfg.Database.MySQL.Port,
				User: cfg.Database.MySQL.User,
				Name: cfg.Database.MySQL.Name,
			},
			Postgres: ninja.StartupPostgresConfig{
				Host:     cfg.Database.Postgres.Host,
				Port:     cfg.Database.Postgres.Port,
				User:     cfg.Database.Postgres.User,
				Name:     cfg.Database.Postgres.Name,
				SSLMode:  cfg.Database.Postgres.SSLMode,
				TimeZone: cfg.Database.Postgres.TimeZone,
			},
		},
	}
}
