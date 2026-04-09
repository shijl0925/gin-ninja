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

func FuzzParseListInput(f *testing.F) {
	for _, seed := range []struct {
		search string
		age    int
	}{
		{search: "alice", age: 18},
		{search: "", age: 0},
		{search: "中文", age: -1},
	} {
		f.Add(seed.search, seed.age)
	}

	f.Fuzz(func(t *testing.T, search string, age int) {
		admin := age%2 == 0
		_, _ = Parse(&listInput{
			embeddedFilter: embeddedFilter{IsAdmin: &admin},
			Search:         search,
			AgeMin:         age,
		})
	})
}
