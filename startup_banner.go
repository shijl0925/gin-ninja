package ninja

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shijl0925/gin-ninja/settings"
)

const startupLogo = ` ██████╗ ██╗███╗   ██╗      ███╗   ██╗██╗███╗   ██╗     ██╗ █████╗ 
██╔════╝ ██║████╗  ██║      ████╗  ██║██║████╗  ██║     ██║██╔══██╗
██║  ███╗██║██╔██╗ ██║█████╗██╔██╗ ██║██║██╔██╗ ██║     ██║███████║
██║   ██║██║██║╚██╗██║╚════╝██║╚██╗██║██║██║╚██╗██║██   ██║██╔══██║
╚██████╔╝██║██║ ╚████║      ██║ ╚████║██║██║ ╚████║╚█████╔╝██║  ██║
 ╚═════╝ ╚═╝╚═╝  ╚═══╝      ╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝ ╚════╝ ╚═╝  ╚═╝
`

var startupSensitiveKVPattern = regexp.MustCompile(`(?i)\b((?:password|passwd|pwd|pass|secret|token)\s*=\s*)('[^']*'|"[^"]*"|[^ ]+)`)

func (api *NinjaAPI) printStartupBanner(listener net.Listener) {
	_, _ = io.WriteString(os.Stdout, api.startupBanner(listener))
}

func (api *NinjaAPI) startupBanner(listener net.Listener) string {
	cfg := settings.GetGlobal()

	var b strings.Builder
	b.WriteString("dsn: ")
	b.WriteString(startupDSN(cfg.Database))
	b.WriteString("\nport: ")
	b.WriteString(startupPort(listener, cfg.Server.Port))
	b.WriteString("\nenv: ")
	b.WriteString(startupEnv(cfg))
	b.WriteString("\nversion: ")
	b.WriteString(startupVersion(api, cfg))
	b.WriteString("\n---\n\n")
	b.WriteString(startupLogo)
	return b.String()
}

func startupDSN(cfg settings.DatabaseConfig) string {
	if dsn := strings.TrimSpace(cfg.DSN); dsn != "" {
		return sanitizeStartupDSN(dsn)
	}

	switch strings.TrimSpace(cfg.Driver) {
	case "mysql":
		if !cfg.MySQL.IsConfigured() {
			return "-"
		}
		host := strings.TrimSpace(cfg.MySQL.Host)
		if host == "" {
			host = "-"
		}
		name := strings.TrimSpace(cfg.MySQL.Name)
		if name == "" {
			name = "-"
		}
		port := cfg.MySQL.Port
		if port <= 0 {
			port = 3306
		}
		user := strings.TrimSpace(cfg.MySQL.User)
		if user == "" {
			return fmt.Sprintf("tcp(%s:%d)/%s", host, port, name)
		}
		return fmt.Sprintf("%s@tcp(%s:%d)/%s", user, host, port, name)
	case "postgres", "postgresql":
		if !cfg.Postgres.IsConfigured() {
			return "-"
		}
		host := strings.TrimSpace(cfg.Postgres.Host)
		if host == "" {
			host = "-"
		}
		name := strings.TrimSpace(cfg.Postgres.Name)
		if name == "" {
			name = "-"
		}
		port := cfg.Postgres.Port
		if port <= 0 {
			port = 5432
		}
		var parts []string
		parts = append(parts, "host="+host, "port="+strconv.Itoa(port), "dbname="+name)
		if user := strings.TrimSpace(cfg.Postgres.User); user != "" {
			parts = append(parts, "user="+user)
		}
		if sslmode := strings.TrimSpace(cfg.Postgres.SSLMode); sslmode != "" {
			parts = append(parts, "sslmode="+sslmode)
		}
		if timeZone := strings.TrimSpace(cfg.Postgres.TimeZone); timeZone != "" {
			parts = append(parts, "TimeZone="+timeZone)
		}
		return strings.Join(parts, " ")
	default:
		return "-"
	}
}

func startupPort(listener net.Listener, fallback int) string {
	if listener != nil {
		if _, port, err := net.SplitHostPort(listener.Addr().String()); err == nil && port != "" {
			return port
		}
	}
	if fallback > 0 {
		return strconv.Itoa(fallback)
	}
	return "-"
}

func startupEnv(cfg settings.Config) string {
	if env := strings.TrimSpace(cfg.App.Env); env != "" {
		return env
	}
	if mode := strings.TrimSpace(gin.Mode()); mode != "" {
		return mode
	}
	return "-"
}

func startupVersion(api *NinjaAPI, cfg settings.Config) string {
	if api != nil {
		if version := strings.TrimSpace(api.config.Version); version != "" {
			return version
		}
	}
	if version := strings.TrimSpace(cfg.App.Version); version != "" {
		return version
	}
	return "-"
}

func sanitizeStartupDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return "-"
	}
	if sanitized, ok := sanitizeStartupDSNURL(trimmed); ok {
		return sanitized
	}
	trimmed = startupSensitiveKVPattern.ReplaceAllStringFunc(trimmed, redactStartupKeyValue)
	trimmed = sanitizeStartupMySQLCredentials(trimmed)
	return sanitizeStartupQuery(trimmed)
}

func redactStartupKeyValue(match string) string {
	submatches := startupSensitiveKVPattern.FindStringSubmatch(match)
	if len(submatches) != 3 {
		return match
	}
	prefix := submatches[1]
	value := submatches[2]
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return prefix + "'xxxxx'"
	}
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return prefix + `"xxxxx"`
	}
	return prefix + "xxxxx"
}

func sanitizeStartupDSNURL(dsn string) (string, bool) {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" || (!strings.Contains(dsn, "://") && !strings.HasPrefix(dsn, "file:")) {
		return "", false
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if _, ok := parsed.User.Password(); ok {
			parsed.User = url.UserPassword(username, "xxxxx")
		} else {
			parsed.User = url.User(username)
		}
	}
	values := parsed.Query()
	changed := false
	for key := range values {
		if isStartupSensitiveKey(key) {
			values.Set(key, "xxxxx")
			changed = true
		}
	}
	if changed {
		parsed.RawQuery = values.Encode()
	}
	return parsed.String(), true
}

func sanitizeStartupMySQLCredentials(dsn string) string {
	at := strings.IndexByte(dsn, '@')
	if at < 0 {
		return dsn
	}
	credentials := dsn[:at]
	colon := strings.IndexByte(credentials, ':')
	if colon < 0 {
		return dsn
	}
	return credentials[:colon] + ":xxxxx" + dsn[at:]
}

func sanitizeStartupQuery(dsn string) string {
	queryIndex := strings.IndexByte(dsn, '?')
	if queryIndex < 0 || queryIndex == len(dsn)-1 {
		return dsn
	}
	values, err := url.ParseQuery(dsn[queryIndex+1:])
	if err != nil {
		return dsn
	}
	changed := false
	for key := range values {
		if isStartupSensitiveKey(key) {
			values.Set(key, "xxxxx")
			changed = true
		}
	}
	if !changed {
		return dsn
	}
	return dsn[:queryIndex+1] + values.Encode()
}

func isStartupSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "password", "passwd", "pwd", "pass", "secret", "token", "access_token", "auth_token":
		return true
	default:
		return false
	}
}
