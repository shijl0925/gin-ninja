package common

import "strings"

func ShouldIgnoreImplicitDefaultDSN(dsn, driver string, hasStructuredConfig bool) bool {
	trimmedDriver := strings.TrimSpace(driver)
	return hasStructuredConfig && trimmedDriver != "sqlite" && trimmedDriver != "sqlite3" && strings.TrimSpace(dsn) == "app.db"
}

func SanitizeParams(params map[string]string) map[string]string {
	values := make(map[string]string, len(params))
	for key, value := range params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		values[trimmedKey] = value
	}
	return values
}
