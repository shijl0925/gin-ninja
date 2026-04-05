package ninja

import "testing"

func FuzzNormalizeVersionParam(f *testing.F) {
	for _, seed := range []string{"v1", " v2.json ", "", "版本1.json", "v1.0"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		_ = normalizeVersionParam(value)
	})
}
