package main

import "testing"

func TestFeaturesExampleGlobals(t *testing.T) {
	if fatalFeatures == nil {
		t.Fatal("expected fatalFeatures to be initialized")
	}
}
