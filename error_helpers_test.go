package ninja

import (
	"errors"
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

func TestErrorFactoryAndCloneHelpers(t *testing.T) {
	t.Parallel()

	if got := NewError(http.StatusForbidden, "denied"); got.Status != http.StatusForbidden || got.Message != "denied" {
		t.Fatalf("NewError() = %+v", got)
	}
	if got := NewErrorWithCode(http.StatusConflict, "CONFLICT", "taken"); got.Code != "CONFLICT" || got.Message != "taken" {
		t.Fatalf("NewErrorWithCode() = %+v", got)
	}
	if cloneBuiltinError(nil) != nil {
		t.Fatal("expected cloneBuiltinError(nil) to be nil")
	}
}

func TestErrorMapperHelpers(t *testing.T) {
	sentinel := errors.New("sentinel")

	t.Run("map error nil", func(t *testing.T) {
		t.Parallel()
		if got := mapError(nil); got != nil {
			t.Fatalf("mapError(nil) = %v", got)
		}
	})

	t.Run("api register appends mapper and ignores nil", func(t *testing.T) {
		api := New(Config{DisableGinDefault: true})
		initial := len(api.errorMappers)
		api.RegisterErrorMapper(nil)
		api.RegisterErrorMapper(func(err error) error {
			if errors.Is(err, sentinel) {
				return NewError(http.StatusTeapot, "mapped")
			}
			return nil
		})

		if count := len(api.errorMappers); count != initial+1 {
			t.Fatalf("expected one registered mapper, got %d", count-initial)
		}

		got := api.mapError(sentinel)
		if !errors.Is(got, NewError(http.StatusTeapot, "mapped")) {
			t.Fatalf("expected registered mapper to be applied, got %v", got)
		}
	})

	t.Run("first mapper wins and nil mappers skipped", func(t *testing.T) {
		t.Parallel()
		got := mapErrorWithMappers(sentinel, []ErrorMapper{
			nil,
			func(err error) error { return NewError(http.StatusBadRequest, "first") },
			func(err error) error { return NewError(http.StatusTeapot, "second") },
		})
		if !errors.Is(got, NewError(http.StatusBadRequest, "first")) {
			t.Fatalf("expected first mapper to win, got %v", got)
		}
	})

	t.Run("api mappers are instance scoped", func(t *testing.T) {
		api := New(Config{DisableGinDefault: true})
		if got := api.mapError(sentinel); !errors.Is(got, sentinel) {
			t.Fatalf("expected unmapped error, got %v", got)
		}
		api.RegisterErrorMapper(func(err error) error {
			if errors.Is(err, sentinel) {
				return NewError(http.StatusBadRequest, "local")
			}
			return nil
		})
		if got := api.mapError(sentinel); !errors.Is(got, NewError(http.StatusBadRequest, "local")) {
			t.Fatalf("expected instance mapper to apply, got %v", got)
		}
		other := New(Config{DisableGinDefault: true})
		if got := other.mapError(sentinel); !errors.Is(got, sentinel) {
			t.Fatalf("expected mapper not to leak to other API, got %v", got)
		}
	})
}
