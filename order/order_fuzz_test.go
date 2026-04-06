package order

import "testing"

func FuzzParseSort(f *testing.F) {
	for _, seed := range []string{"name,-created_at", " , +name , -score ,, ", "+", "", "中文,-时间"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, sort string) {
		_ = ParseSort(sort)
	})
}
