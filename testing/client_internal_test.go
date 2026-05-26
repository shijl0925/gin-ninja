package ninjatest

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
)

func TestClientOptionsInitializeDefaultHeaders(t *testing.T) {
	var headerConfig clientConfig
	WithHeader("X-Test", "one")(&headerConfig)
	if got := headerConfig.defaultHeaders.Get("X-Test"); got != "one" {
		t.Fatalf("expected WithHeader to initialize headers, got %q", got)
	}

	var headersConfig clientConfig
	WithHeaders(http.Header{"X-Test": {"one", "two"}})(&headersConfig)
	if got := headersConfig.defaultHeaders.Values("X-Test"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("expected WithHeaders to initialize headers, got %v", got)
	}
}

func TestEncodeMultipartBodyReturnsCopyError(t *testing.T) {
	_, _, err := encodeMultipartBody(MultipartBody{
		Fields: url.Values{"title": {"demo"}},
		Files: []MultipartFile{{
			FieldName: "file",
			FileName:  "bad.txt",
			Body:      errorReader{},
		}},
	})
	if err == nil {
		t.Fatalf("expected copy error")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
