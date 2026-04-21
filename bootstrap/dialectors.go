package bootstrap

import (
	"fmt"

	"github.com/shijl0925/gin-ninja/settings"
	"gorm.io/gorm"
)

// buildDialector returns the GORM Dialector for the given DatabaseConfig.
// Drivers are resolved via registration so that only the drivers you actually
// import contribute to your binary.
func buildDialector(cfg *settings.DatabaseConfig) (gorm.Dialector, error) {
	builder, driver, err := registeredDialector(cfg)
	if err != nil {
		return nil, err
	}

	dialector, err := builder(*cfg)
	if err != nil {
		return nil, err
	}
	if dialector == nil {
		return nil, fmt.Errorf("bootstrap: database driver %q returned a nil dialector", driver)
	}
	return dialector, nil
}
