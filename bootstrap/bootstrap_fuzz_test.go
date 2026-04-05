package bootstrap

import (
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
)

func FuzzMySQLDialector(f *testing.F) {
	for _, seed := range []string{
		"root:p%40ss%3Aword@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local",
		"root:p%2Bss@tcp(localhost:3306)/app?loc=UTC+8",
		"bad%zz",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = mysqlDialector(settings.DatabaseConfig{DSN: raw})
	})
}
