package bootstrap

import (
	"fmt"
	"strings"
	"sync"

	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/gorm"
)

// DialectorBuilder builds a GORM dialector for a database config.
type DialectorBuilder func(settings.DatabaseConfig) (gorm.Dialector, error)

var (
	dialectorRegistryMu sync.RWMutex
	dialectorRegistry   = map[string]DialectorBuilder{}
)

// RegisterDialector registers a dialector builder for the given driver names.
func RegisterDialector(builder DialectorBuilder, names ...string) error {
	if builder == nil {
		return fmt.Errorf("bootstrap: dialector builder must not be nil")
	}
	if len(names) == 0 {
		return fmt.Errorf("bootstrap: at least one database driver name is required")
	}

	dialectorRegistryMu.Lock()
	defer dialectorRegistryMu.Unlock()

	for _, name := range names {
		normalized := normalizeDriverName(name)
		if normalized == "" {
			return fmt.Errorf("bootstrap: database driver name must not be empty")
		}
		if _, exists := dialectorRegistry[normalized]; exists {
			return fmt.Errorf("bootstrap: database driver %q is already registered", normalized)
		}
		dialectorRegistry[normalized] = builder
	}
	return nil
}

// MustRegisterDialector registers a dialector builder and panics on error.
func MustRegisterDialector(builder DialectorBuilder, names ...string) {
	if err := RegisterDialector(builder, names...); err != nil {
		panic(err)
	}
}

func registeredDialector(cfg *settings.DatabaseConfig) (DialectorBuilder, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("bootstrap: database config must not be nil")
	}

	driver := normalizeDriverName(cfg.Driver)
	if driver == "" {
		return nil, "", fmt.Errorf("bootstrap: database driver must not be empty")
	}

	dialectorRegistryMu.RLock()
	builder, ok := dialectorRegistry[driver]
	dialectorRegistryMu.RUnlock()
	if !ok {
		return nil, driver, fmt.Errorf("bootstrap: database driver %q is not registered; import the matching bootstrap/drivers package", driver)
	}
	return builder, driver, nil
}

func normalizeDriverName(driver string) string {
	return strings.ToLower(strings.TrimSpace(driver))
}
