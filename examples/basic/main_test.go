package main

import (
	"errors"
	"net/http"
	"testing"

	ninjatest "github.com/shijl0925/gin-ninja/testing"
)

func newBasicTestAPI(t *testing.T) *ninjatest.TestClient {
	t.Helper()

	db, err := initDB("file:" + t.Name() + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	return ninjatest.NewWithT(t, buildAPI(db))
}

func doBasicJSON(t *testing.T, client *ninjatest.TestClient, method, path string, body any) *ninjatest.Response {
	t.Helper()

	return client.Request(method, path, body)
}

func decodeBasicBody(t *testing.T, resp *ninjatest.Response, out any) {
	t.Helper()
	if err := resp.DecodeJSON(out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
}

func TestBasicExampleRoutesAndCRUD(t *testing.T) {
	client := newBasicTestAPI(t)

	resp := client.Get("/docs")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", resp.StatusCode)
	}

	resp = client.Get("/openapi.json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", resp.StatusCode)
	}

	resp = client.Get("/health")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.StatusCode)
	}

	create := doBasicJSON(t, client, http.MethodPost, "/api/v1/users/", CreateUserInput{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   18,
	})
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", create.StatusCode)
	}
	var created map[string]any
	decodeBasicBody(t, create, &created)
	if created["id"] == nil || created["name"] != "Alice" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	list := client.Get("/api/v1/users/?search=Ali")
	if list.StatusCode != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.StatusCode)
	}
	var page map[string]any
	decodeBasicBody(t, list, &page)
	if page["total"] != float64(1) {
		t.Fatalf("unexpected list page: %+v", page)
	}

	get := client.Get("/api/v1/users/1")
	if get.StatusCode != http.StatusOK {
		t.Fatalf("expected get 200, got %d", get.StatusCode)
	}
	var got map[string]any
	decodeBasicBody(t, get, &got)
	if got["email"] != "alice@example.com" {
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

	deleteResp := doBasicJSON(t, client, http.MethodDelete, "/api/v1/users/1", nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", deleteResp.StatusCode)
	}

	missing := client.Get("/api/v1/users/1")
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
