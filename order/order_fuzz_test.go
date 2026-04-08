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

func FuzzResolveOrder(f *testing.F) {
	for _, seed := range []string{"name,-created", "display_name", "password", "", " , +name "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, sort string) {
		_, _ = ResolveOrder(&taggedSortInput{Sort: sort})
		_, _ = ResolveOrder(&equalsTaggedSortInput{Sort: sort})
	})
}
