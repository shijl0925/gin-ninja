package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadWithOverrides(f *testing.F) {
	for _, seed := range []struct {
		base     string
		override string
	}{
		{
			base:     "app:\n  name: demo\nserver:\n  port: 8080\n",
			override: "server:\n  port: 8081\n",
		},
		{
			base:     "app:\n  env: production\n",
			override: "server: [broken",
		},
		{
			base:     "",
			override: "",
		},
	} {
		f.Add(seed.base, seed.override)
	}

	f.Fuzz(func(t *testing.T, baseContent, overrideContent string) {
		dir := t.TempDir()
		base := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(base, []byte(baseContent), 0o644); err != nil {
			t.Fatalf("write base: %v", err)
		}

		override := filepath.Join(dir, "config.local.yaml")
		if overrideContent != "" {
			if err := os.WriteFile(override, []byte(overrideContent), 0o644); err != nil {
				t.Fatalf("write override: %v", err)
			}
			_, _ = LoadWithOverrides(base, override)
			return
		}

		_, _ = LoadWithOverrides(base, override)
	})
}
