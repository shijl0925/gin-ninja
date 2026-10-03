package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDialectorRegistryErrorBranches(t *testing.T) {
	if err := RegisterDialector(nil, "coverage-nil-builder"); err == nil {
		t.Fatal("expected nil builder registration to fail")
	}
	if err := RegisterDialector(func(settings.DatabaseConfig) (gorm.Dialector, error) { return sqlite.Open(":memory:"), nil }); err == nil {
		t.Fatal("expected registration without names to fail")
	}
	if err := RegisterDialector(func(settings.DatabaseConfig) (gorm.Dialector, error) { return sqlite.Open(":memory:"), nil }, " "); err == nil {
		t.Fatal("expected blank driver name to fail")
	}

	name := normalizeDriverName("coverage-" + strings.ReplaceAll(t.Name(), "/", "-"))
	builder := func(settings.DatabaseConfig) (gorm.Dialector, error) { return sqlite.Open(":memory:"), nil }
	if err := RegisterDialector(builder, name); err != nil {
		t.Fatalf("RegisterDialector: %v", err)
	}
	if err := RegisterDialector(builder, name); err == nil {
		t.Fatal("expected duplicate driver registration to fail")
	}
	if got, driver, err := registeredDialector(&settings.DatabaseConfig{Driver: " " + name + " "}); err != nil || got == nil || driver != name {
		t.Fatalf("registeredDialector = (%v, %q, %v)", got, driver, err)
	}
	if _, _, err := registeredDialector(nil); err == nil {
		t.Fatal("expected nil config error")
	}
	if _, _, err := registeredDialector(&settings.DatabaseConfig{}); err == nil {
		t.Fatal("expected empty driver error")
	}

	nilName := name + "-nil"
	if err := RegisterDialector(func(settings.DatabaseConfig) (gorm.Dialector, error) { return nil, nil }, nilName); err != nil {
		t.Fatalf("RegisterDialector nil dialector: %v", err)
	}
	if _, err := buildDialector(&settings.DatabaseConfig{Driver: nilName}); err == nil {
		t.Fatal("expected nil dialector error")
	}

	errName := name + "-err"
	if err := RegisterDialector(func(settings.DatabaseConfig) (gorm.Dialector, error) { return nil, fmt.Errorf("boom") }, errName); err != nil {
		t.Fatalf("RegisterDialector err builder: %v", err)
	}
	if _, err := buildDialector(&settings.DatabaseConfig{Driver: errName}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected builder error, got %v", err)
	}
}
