package main

import "testing"

func TestUsersMainUsesFatalOnRunError(t *testing.T) {
	originalFatal := fatalUsers
	t.Cleanup(func() { fatalUsers = originalFatal })

	t.Setenv("SERVER__PORT", "-1")

	called := false
	fatalUsers = func(v ...any) { called = true }
	main()
	if !called {
		t.Fatal("expected main to invoke fatal handler on run error")
	}
}
