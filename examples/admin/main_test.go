package main

import "testing"

func TestAdminExampleGlobals(t *testing.T) {
	if fatalAdmin == nil {
		t.Fatal("expected fatalAdmin to be initialized")
	}
}
