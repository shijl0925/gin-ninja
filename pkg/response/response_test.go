package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newResponseContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestResponseConstructors(t *testing.T) {
	if got := OK("data"); got.Code != CodeOK || got.Message != "success" || got.Data != "data" {
		t.Fatalf("unexpected OK response: %+v", got)
	}
	if got := OKWithMessage("done", 1); got.Message != "done" || got.Data != 1 {
		t.Fatalf("unexpected OKWithMessage response: %+v", got)
	}
	if got := Fail(CodeForbidden, "forbidden"); got.Code != CodeForbidden || got.Data != nil {
		t.Fatalf("unexpected Fail response: %+v", got)
	}
	if got := FailWithData(CodeValidation, "invalid", gin.H{"field": "name"}); got.Data == nil {
		t.Fatalf("expected FailWithData payload: %+v", got)
	}
	if got := Error("boom"); got.Code != CodeError || got.Message != "boom" {
		t.Fatalf("unexpected Error response: %+v", got)
	}
}

func TestResponseHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("json and success", func(t *testing.T) {
		c, w := newResponseContext()
		JSON(c, OK("value"))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"message":"success"`) {
			t.Fatalf("unexpected JSON response: %d %s", w.Code, w.Body.String())
		}

		c, w = newResponseContext()
		Success(c, gin.H{"ok": true})
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":0`) {
			t.Fatalf("unexpected Success response: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("error helpers", func(t *testing.T) {
		cases := []struct {
			name        string
			fn          func(*gin.Context, string)
			msg         string
			status      int
			wantMessage string
		}{
			{name: "unauthorized", fn: Unauthorized, status: http.StatusUnauthorized, wantMessage: "unauthorized"},
			{name: "forbidden", fn: Forbidden, status: http.StatusForbidden, wantMessage: "forbidden"},
			{name: "notfound", fn: NotFound, status: http.StatusNotFound, wantMessage: "not found"},
			{name: "badrequest", fn: BadRequest, status: http.StatusBadRequest, wantMessage: "bad request"},
			{name: "servererror", fn: ServerError, status: http.StatusInternalServerError, wantMessage: "internal server error"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c, w := newResponseContext()
				tc.fn(c, "")
				if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.wantMessage) {
					t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
				}
			})
		}
	})
}
