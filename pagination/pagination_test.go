package pagination

import "testing"

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
