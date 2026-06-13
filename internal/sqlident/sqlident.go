package sqlident

import "strings"

// IsSafeFieldName reports whether field is a simple SQL identifier or table.column pair.
func IsSafeFieldName(field string) bool {
	if field == "" {
		return false
	}
	table, name, ok := strings.Cut(field, ".")
	if !ok {
		return IsSafeIdentifier(field)
	}
	return IsSafeIdentifier(table) && IsSafeIdentifier(name) && !strings.Contains(name, ".")
}

// IsSafeIdentifier reports whether identifier contains only ASCII identifier characters.
func IsSafeIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	for i := 0; i < len(identifier); i++ {
		ch := identifier[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_' || i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}
