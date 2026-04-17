package main

import "testing"

func TestAdminMainUsesFatalOnRunError(t *testing.T) {
	originalFatal := fatalAdmin
	t.Cleanup(func() { fatalAdmin = originalFatal })

	t.Setenv("SERVER__PORT", "-1")

	called := false
	fatalAdmin = func(v ...any) { called = true }
	main()
	if !called {
		t.Fatal("expected main to invoke fatal handler on run error")
	}
}
