package ninja

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func FuzzSetFieldFromString(f *testing.F) {
	for _, seed := range []string{"42", "true", "3.14", "", "中文", " \n "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		values := []reflect.Value{
			reflect.New(reflect.TypeOf("")).Elem(),
			reflect.New(reflect.TypeOf(int(0))).Elem(),
			reflect.New(reflect.TypeOf(uint(0))).Elem(),
			reflect.New(reflect.TypeOf(bool(false))).Elem(),
			reflect.New(reflect.TypeOf(float64(0))).Elem(),
		}
		for _, value := range values {
			_ = setFieldFromString(value, raw)
		}
	})
}

func FuzzBindInputFormLikeValues(f *testing.F) {
	gin.SetMode(gin.TestMode)
	for _, seed := range []struct {
		query  string
		header string
	}{
		{query: "flag=true&page=2", header: "alice"},
		{query: "flag=false&page=bad", header: ""},
		{query: "name=%E4%B8%AD%E6%96%87", header: "机器人"},
	} {
		f.Add(seed.query, seed.header)
	}

	type input struct {
		Flag   bool   `form:"flag"`
		Page   int    `form:"page"`
		Name   string `form:"name"`
		Header string `header:"X-Name"`
	}

	f.Fuzz(func(t *testing.T, query, header string) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		req.Header.Set("X-Name", header)
		c.Request = req

		var in input
		_ = bindInput(c, http.MethodGet, &in)
	})
}
