package ninja

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func FuzzNormalizeVersionParam(f *testing.F) {
	for _, seed := range []string{"v1", " v2.json ", "", "版本1.json", "v1.0"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		_ = normalizeVersionParam(value)
	})
}

func FuzzRequestVersion(f *testing.F) {
	gin.SetMode(gin.TestMode)
	for _, seed := range []struct {
		version     string
		versionJSON string
	}{
		{version: "v1", versionJSON: ""},
		{version: "", versionJSON: "v2.json"},
		{version: " 版本3 ", versionJSON: "v9.json"},
	} {
		f.Add(seed.version, seed.versionJSON)
	}

	f.Fuzz(func(t *testing.T, version, versionJSON string) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/docs", nil)
		c.Params = gin.Params{
			{Key: "version", Value: version},
			{Key: "version.json", Value: versionJSON},
		}
		_ = requestVersion(c)
	})
}
