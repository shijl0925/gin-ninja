package ninja

import (
	"net/http"
	"testing"
)

func TestErrorStringFormatting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "code and message",
			err:  &Error{Status: http.StatusTeapot, Code: "TEAPOT", Message: "short and stout"},
			want: "[418] TEAPOT: short and stout",
		},
		{
			name: "message only",
			err:  &Error{Status: http.StatusBadRequest, Message: "bad request"},
			want: "[400] bad request",
		},
		{
			name: "code only",
			err:  &Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED"},
			want: "[401] UNAUTHORIZED",
		},
		{
			name: "status only",
			err:  &Error{Status: http.StatusInternalServerError},
			want: "[500]",
		},
		{
			name: "nil error",
			err:  nil,
			want: "<nil>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuiltinErrorHelpers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		errFn  func() *Error
		isFn   func(error) bool
		status int
		code   string
	}{
		{"bad request", BadRequestError, IsBadRequest, http.StatusBadRequest, "BAD_REQUEST"},
		{"unauthorized", UnauthorizedError, IsUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"forbidden", ForbiddenError, IsForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"not found", NotFoundError, IsNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"conflict", ConflictError, IsConflict, http.StatusConflict, "CONFLICT"},
		{"internal", InternalError, IsInternal, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.errFn()
			if err.Status != tc.status || err.Code != tc.code {
				t.Fatalf("unexpected builtin error: %+v", err)
			}
			if !tc.isFn(err) {
				t.Fatalf("expected classifier to match %+v", err)
			}

			err.Message = "changed"
			fresh := tc.errFn()
			if fresh.Message == "changed" {
				t.Fatalf("expected %s helper to return a clone", tc.name)
			}
		})
	}
}
