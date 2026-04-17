package main

import "testing"

func TestFeaturesMainUsesFatalOnRunError(t *testing.T) {
	originalFatal := fatalFeatures
	t.Cleanup(func() { fatalFeatures = originalFatal })

	t.Setenv("SERVER__PORT", "-1")
	t.Setenv("DATABASE__DSN", "file:features-main-test?mode=memory&cache=shared")

	called := false
	fatalFeatures = func(v ...any) { called = true }
	main()
	if !called {
		t.Fatal("expected main to invoke fatal handler on run error")
	}
}
