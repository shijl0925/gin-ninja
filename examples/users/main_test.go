package main

import "testing"

func TestUsersExampleGlobals(t *testing.T) {
	if fatalUsers == nil {
		t.Fatal("expected fatalUsers to be initialized")
	}
}
