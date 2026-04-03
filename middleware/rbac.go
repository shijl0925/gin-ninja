package middleware

import (
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
)

// PermissionResolver returns the permission codes granted to the current user.
type PermissionResolver func(*ninja.Context) ([]string, error)

// RequirePermissions enforces that the current user has all required
// permission codes. It is intended for use with ninja.WithMiddleware.
func RequirePermissions(resolve PermissionResolver, permissions ...string) func(*ninja.Context) error {
	required := normalizeCodes(permissions)
	return func(ctx *ninja.Context) error {
		if len(required) == 0 {
			return nil
		}
		if ctx == nil || ctx.GetUserID() == 0 {
			return ninja.ErrUnauthorized
		}
		if resolve == nil {
			return ninja.ErrInternal
		}

		granted, err := resolve(ctx)
		if err != nil {
			return err
		}

		lookup := make(map[string]struct{}, len(granted))
		for _, code := range normalizeCodes(granted) {
			lookup[code] = struct{}{}
		}
		for _, code := range required {
			if _, ok := lookup[code]; !ok {
				return ninja.ErrForbidden
			}
		}
		return nil
	}
}

func normalizeCodes(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		out = append(out, code)
	}
	return out
}
