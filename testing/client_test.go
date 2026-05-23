package ninjatest_test

import (
	"net/http"
	"net/url"
	"testing"

	ninja "github.com/shijl0925/gin-ninja"
	ninjatest "github.com/shijl0925/gin-ninja/testing"
)

type testHelloInput struct {
	Name string `query:"name"`
}

type testHelloOutput struct {
	Message string `json:"message"`
}

func testHello(_ *ninja.Context, in *testHelloInput) (*testHelloOutput, error) {
	return &testHelloOutput{Message: "hello " + in.Name}, nil
}

func TestClientRouterTarget(t *testing.T) {
	router := ninja.NewRouter("/users")
	ninja.Get(router, "/", testHello)

	client := ninjatest.NewWithT(t, router)
	resp := client.Get("/users/", ninjatest.Query("name", "alice"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.String())
	}

	var out testHelloOutput
	if err := resp.DecodeJSON(&out); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if out.Message != "hello alice" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

type testCreateInput struct {
	Name  string `json:"name"`
	Trace string `header:"X-Trace-ID"`
	Mode  string `cookie:"mode"`
}

type testCreateOutput struct {
	Name  string `json:"name"`
	Trace string `json:"trace"`
	Mode  string `json:"mode"`
}

func testCreate(_ *ninja.Context, in *testCreateInput) (*testCreateOutput, error) {
	return &testCreateOutput{Name: in.Name, Trace: in.Trace, Mode: in.Mode}, nil
}

func TestClientJSONHeadersAndCookies(t *testing.T) {
	api := ninja.New(ninja.Config{Title: "test"})
	router := ninja.NewRouter("/users")
	ninja.Post(router, "/", testCreate)
	api.AddRouter(router)

	client := ninjatest.NewWithT(t, api, ninjatest.WithHeader("X-Trace-ID", "default"))
	resp := client.Post("/users/", map[string]string{"name": "alice"},
		ninjatest.Header("X-Trace-ID", "request"),
		ninjatest.Cookie(&http.Cookie{Name: "mode", Value: "test"}),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.String())
	}

	var out testCreateOutput
	if err := resp.JSON(&out); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if out.Name != "alice" || out.Trace != "request" || out.Mode != "test" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

type testFormInput struct {
	Name string `form:"name"`
}

func testForm(_ *ninja.Context, in *testFormInput) (*testHelloOutput, error) {
	return &testHelloOutput{Message: "form " + in.Name}, nil
}

func TestClientFormBodyAndRawRequest(t *testing.T) {
	router := ninja.NewRouter("/forms")
	ninja.Post(router, "/", testForm)

	client := ninjatest.NewWithT(t, router)
	req := client.NewRequest(http.MethodPost, "/forms/", url.Values{"name": {"alice"}})
	resp := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.String())
	}

	var out testHelloOutput
	if err := resp.DecodeJSON(&out); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if out.Message != "form alice" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestClientHTTPHandlerTarget(t *testing.T) {
	client := ninjatest.NewWithT(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Path", r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))

	resp := client.Get("/ping")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Path") != "/ping" || resp.String() != "ok" {
		t.Fatalf("unexpected response: header=%q body=%q", resp.Header.Get("X-Path"), resp.String())
	}
}
