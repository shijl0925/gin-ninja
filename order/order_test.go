package order

import "testing"

type taggedSortInput struct {
	Sort string `form:"sort" order:"name|created:created_at|score"`
}

type legacySortInput struct {
	Sort string
}

type embeddedTaggedSortInput struct {
	legacySortInput `order:"id|name|email|age|created_at"`
}

type equalsTaggedSortInput struct {
	Sort string `order:"display_name = users.name | created = users.created_at"`
}

func TestParseSort(t *testing.T) {
	t.Parallel()

	got := ParseSort(" +name , -created_at,score ,, ")
	want := []SortField{{Name: "name"}, {Name: "created_at", Desc: true}, {Name: "score"}}
	if len(got) != len(want) {
		t.Fatalf("expected %d fields, got %d (%+v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected field[%d]: %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestResolveSort(t *testing.T) {
	schema := NewSortSchema("name").Allow("created_at").Allow("score")
	cases := []struct {
		name    string
		sort    string
		want    []SortField
		wantErr bool
	}{
		{
			name: "mixed directions",
			sort: "name,-created_at",
			want: []SortField{{Name: "name"}, {Name: "created_at", Desc: true}},
		},
		{
			name: "whitespace empty segments and explicit plus",
			sort: " , +name , -score ,, ",
			want: []SortField{{Name: "name"}, {Name: "score", Desc: true}},
		},
		{
			name: "empty after prefix is skipped",
			sort: "+,-, ,",
		},
		{
			name:    "unknown field",
			sort:    "password",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := ResolveSort(tc.sort, schema)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected sort validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSort: %v", err)
			}
			if len(fields) != len(tc.want) {
				t.Fatalf("expected %d fields, got %d (%+v)", len(tc.want), len(fields), fields)
			}
			for i := range fields {
				if fields[i] != tc.want[i] {
					t.Fatalf("unexpected field[%d]: %+v want %+v", i, fields[i], tc.want[i])
				}
			}
		})
	}
}

func TestResolveOrder(t *testing.T) {
	cases := []struct {
		name    string
		input   any
		want    []SortField
		wantErr bool
	}{
		{
			name:  "standalone sort field with alias",
			input: &taggedSortInput{Sort: "name,-created"},
			want:  []SortField{{Name: "name"}, {Name: "created_at", Desc: true}},
		},
		{
			name:  "embedded legacy sort field",
			input: &embeddedTaggedSortInput{legacySortInput: legacySortInput{Sort: "-age,+created_at"}},
			want:  []SortField{{Name: "age", Desc: true}, {Name: "created_at"}},
		},
		{
			name:  "equals separator with explicit columns",
			input: &equalsTaggedSortInput{Sort: "display_name,-created"},
			want:  []SortField{{Name: "users.name"}, {Name: "users.created_at", Desc: true}},
		},
		{
			name:  "blank tokens ignored",
			input: &taggedSortInput{Sort: " , + , "},
		},
		{
			name:    "unknown field rejected",
			input:   &taggedSortInput{Sort: "password"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := ResolveOrder(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected sort validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveOrder: %v", err)
			}
			if len(fields) != len(tc.want) {
				t.Fatalf("expected %d fields, got %d (%+v)", len(tc.want), len(fields), fields)
			}
			for i := range fields {
				if fields[i] != tc.want[i] {
					t.Fatalf("unexpected field[%d]: %+v want %+v", i, fields[i], tc.want[i])
				}
			}
		})
	}
}

func TestResolveOrderRejectsInvalidOrderTagTarget(t *testing.T) {
	input := &struct {
		Page int `form:"page" order:"id"`
	}{Page: 1}

	if _, err := ResolveOrder(input); err == nil {
		t.Fatal("expected invalid order tag target error")
	}
}

func TestResolveOrderInputValidation(t *testing.T) {
	t.Parallel()

	if fields, err := ResolveOrder(nil); err != nil || len(fields) != 0 {
		t.Fatalf("ResolveOrder(nil) = (%+v, %v)", fields, err)
	}
	if _, err := ResolveOrder("name"); err == nil {
		t.Fatal("expected non-struct ResolveOrder error")
	}
}

func TestApplySortAndApplyOrderNilQuery(t *testing.T) {
	t.Parallel()

	schema := NewSortSchema("name")
	if err := ApplySort[struct{}](nil, "name", schema); err != nil {
		t.Fatalf("ApplySort(nil) error = %v", err)
	}
	if err := ApplyOrder[struct{}](nil, &taggedSortInput{Sort: "name"}); err != nil {
		t.Fatalf("ApplyOrder(nil) error = %v", err)
	}
}
