package sqlident

import "testing"

func TestIsSafeFieldName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want bool
	}{
		{name: "name", want: true},
		{name: "users.email", want: true},
		{name: "_private1", want: true},
		{name: "", want: false},
		{name: "1name", want: false},
		{name: "users.profile.email", want: false},
		{name: "name desc", want: false},
		{name: "`name`", want: false},
		{name: "name;drop", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSafeFieldName(tc.name); got != tc.want {
				t.Fatalf("IsSafeFieldName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
