package filter

import (
	"reflect"
	"testing"
)

func FuzzParseTag(f *testing.F) {
	for _, seed := range []string{"name,eq", "name|email,like", " status ", "name,,", ""} {
		f.Add(seed)
	}

	field := reflect.TypeOf(struct {
		Value string `filter:"value,eq"`
	}{}).Field(0)

	f.Fuzz(func(t *testing.T, tag string) {
		_, _, _, _ = parseTag(tag, field)
	})
}
