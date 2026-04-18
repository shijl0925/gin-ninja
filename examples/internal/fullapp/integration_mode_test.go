//go:build !integration

package fullapp

import "testing"

func requireIntegration(t *testing.T) {
	t.Helper()
	t.Skip("requires go test -tags=integration")
}
