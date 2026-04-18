package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBasicTestAPI(t *testing.T) *httptest.Server {
	t.Helper()

	db, err := initDB("file:" + t.Name() + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	return httptest.NewServer(buildAPI(db).Handler())
}

func doBasicJSON(t *testing.T, server *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func decodeBasicBody(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
}

func TestBasicExampleRoutesAndCRUD(t *testing.T) {
	requireIntegration(t)

	server := newBasicTestAPI(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/docs")
	if err != nil {
		t.Fatalf("GET /docs: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = http.Get(server.URL + "/openapi.json")
	if err != nil {
		t.Fatalf("GET /openapi.json: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	create := doBasicJSON(t, server, http.MethodPost, "/api/v1/users/", CreateUserInput{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   18,
	})
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.StatusCode)
	}
	var created UserOut
	decodeBasicBody(t, create, &created)
	if created.ID == 0 || created.Name != "Alice" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	list, err := http.Get(server.URL + "/api/v1/users/?search=Ali")
	if err != nil {
		t.Fatalf("GET list: %v", err)
	}
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.StatusCode)
	}
	var page map[string]any
	decodeBasicBody(t, list, &page)
	if page["total"] != float64(1) {
		t.Fatalf("unexpected list page: %+v", page)
	}

	get, err := http.Get(server.URL + "/api/v1/users/1")
	if err != nil {
		t.Fatalf("GET user: %v", err)
	}
	if get.StatusCode != http.StatusOK {
		t.Fatalf("expected get 200, got %d", get.StatusCode)
	}
	var got UserOut
	decodeBasicBody(t, get, &got)
	if got.Email != "alice@example.com" {
		t.Fatalf("unexpected fetched user: %+v", got)
	}

	updatedDirect, err := updateUser(nil, &UpdateUserInput{
		UserID: 1,
		Name:   "Alicia",
		Email:  "alice@example.com",
		Age:    19,
	})
	if err != nil {
		t.Fatalf("updateUser: %v", err)
	}
	if updatedDirect.Name != "Alicia" || updatedDirect.Age != 19 {
		t.Fatalf("unexpected updated user: %+v", updatedDirect)
	}

	deleteResp := doBasicJSON(t, server, http.MethodDelete, "/api/v1/users/1", nil)
	deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", deleteResp.StatusCode)
	}

	missing, err := http.Get(server.URL + "/api/v1/users/1")
	if err != nil {
		t.Fatalf("GET missing user: %v", err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing user 404, got %d", missing.StatusCode)
	}
}

func TestBasicExampleRunReturnsListenError(t *testing.T) {
	if err := run("file:run-basic?mode=memory&cache=shared", ":-1"); err == nil {
		t.Fatal("expected run to fail for invalid address")
	}
}

func TestBasicMainUsesInjectedRunner(t *testing.T) {
	originalRun := runBasicMain
	originalFatal := fatalBasic
	t.Cleanup(func() {
		runBasicMain = originalRun
		fatalBasic = originalFatal
	})

	called := false
	runBasicMain = func(dsn, addr string) error {
		called = dsn == "users.db" && addr == ":8080"
		return nil
	}
	main()
	if !called {
		t.Fatal("expected main to invoke injected runner")
	}

	runBasicMain = func(dsn, addr string) error { return errors.New("boom") }
	fatalCalled := false
	fatalBasic = func(v ...any) { fatalCalled = true }
	main()
	if !fatalCalled {
		t.Fatal("expected main to invoke injected fatal handler")
	}
}
