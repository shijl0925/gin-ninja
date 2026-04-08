package ninja

import "testing"

func BenchmarkNormalizeVersionParam(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = normalizeVersionParam(" v2026.json ")
	}
}

func BenchmarkSSEDataJSONMap(b *testing.B) {
	value := map[string]any{
		"name":  "alice",
		"count": 3,
		"ok":    true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sseData(value)
	}
}
