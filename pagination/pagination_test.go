package pagination

import "testing"

type taggedSortInput struct {
	Sort string `form:"sort" order:"name|created_at|score"`
}

type embeddedTaggedSortInput struct {
	PageInput `order:"id|name|email|age|created_at"`
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
			fields, err := (PageInput{Sort: tc.sort}).ResolveSort(schema)
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
			name:  "standalone sort field",
			input: &taggedSortInput{Sort: "name,-created_at"},
			want:  []SortField{{Name: "name"}, {Name: "created_at", Desc: true}},
		},
		{
			name:  "embedded page input legacy sort",
			input: &embeddedTaggedSortInput{PageInput: PageInput{Sort: "-age,+created_at"}},
			want:  []SortField{{Name: "age", Desc: true}, {Name: "created_at"}},
		},
		{
			name:  "empty sort ignored",
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

func TestResolveSortRejectsUnknownField(t *testing.T) {
	_, err := (PageInput{Sort: "password"}).ResolveSort(NewSortSchema("name"))
	if err == nil {
		t.Fatal("expected sort validation error")
	}
}

func TestNewPageZeroTotalHasZeroPages(t *testing.T) {
	page := NewPage([]int{}, 0, PageInput{Page: 1, Size: 20})
	if page.Pages != 0 {
		t.Fatalf("expected zero pages for zero total, got %d", page.Pages)
	}
}
