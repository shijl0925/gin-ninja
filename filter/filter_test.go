package filter

import "testing"

type embeddedFilter struct {
	IsAdmin *bool `filter:"is_admin,eq"`
}

type listInput struct {
	embeddedFilter
	Search string `filter:"name,like"`
	AgeMin int    `filter:"age,ge"`
}

func TestParse(t *testing.T) {
	admin := true
	clauses, err := Parse(&listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
		Search:         "alice",
		AgeMin:         18,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 3 {
		t.Fatalf("expected 3 clauses, got %d", len(clauses))
	}
	if clauses[0].Field != "is_admin" || clauses[0].Op != OpEq || clauses[0].Value != true {
		t.Fatalf("unexpected clause[0]: %+v", clauses[0])
	}
	if clauses[1].Field != "name" || clauses[1].Op != OpLike || clauses[1].Value != "alice" {
		t.Fatalf("unexpected clause[1]: %+v", clauses[1])
	}
	if clauses[2].Field != "age" || clauses[2].Op != OpGe || clauses[2].Value != 18 {
		t.Fatalf("unexpected clause[2]: %+v", clauses[2])
	}
}

func TestParseSkipsZeroValues(t *testing.T) {
	clauses, err := Parse(&listInput{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 0 {
		t.Fatalf("expected no clauses, got %+v", clauses)
	}
}

func TestParseKeepsFalseBoolPointers(t *testing.T) {
	admin := false
	clauses, err := Parse(&listInput{
		embeddedFilter: embeddedFilter{IsAdmin: &admin},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(clauses) != 1 || clauses[0].Value != false {
		t.Fatalf("expected false bool clause, got %+v", clauses)
	}
}
