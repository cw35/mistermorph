package builtin

import "testing"

func TestDenyPath(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		deny     []string
		wantDeny bool
	}{
		{name: "basename_exact", path: "config.yaml", deny: []string{"config.yaml"}, wantDeny: true},
		{name: "basename_nested", path: "./sub/config.yaml", deny: []string{"config.yaml"}, wantDeny: true},
		{name: "basename_other", path: "./sub/config.yml", deny: []string{"config.yaml"}, wantDeny: false},
		{name: "basename_suffix_not_match", path: "config.yaml.bak", deny: []string{"config.yaml"}, wantDeny: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := denyPath(tc.path, tc.deny)
			if got != tc.wantDeny {
				t.Fatalf("denyPath(%q,%v)=%v, want %v", tc.path, tc.deny, got, tc.wantDeny)
			}
		})
	}
}
