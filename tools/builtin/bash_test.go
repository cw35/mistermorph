package builtin

import "testing"

func TestContainsTokenBoundary(t *testing.T) {
	cases := []struct {
		name   string
		cmd    string
		needle string
		want   bool
	}{
		{name: "plain", cmd: "cat config.yaml", needle: "config.yaml", want: true},
		{name: "quoted", cmd: "cat \"config.yaml\"", needle: "config.yaml", want: true},
		{name: "subpath", cmd: "cat ./config.yaml", needle: "config.yaml", want: true},
		{name: "parent", cmd: "cat ../config.yaml", needle: "config.yaml", want: true},
		{name: "redir", cmd: "grep x <config.yaml", needle: "config.yaml", want: true},
		{name: "assignment", cmd: "X=config.yaml; echo $X", needle: "config.yaml", want: true},
		{name: "nonmatch_prefix", cmd: "cat myconfig.yaml", needle: "config.yaml", want: false},
		{name: "nonmatch_suffix", cmd: "cat config.yaml.bak", needle: "config.yaml", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsTokenBoundary(tc.cmd, tc.needle)
			if got != tc.want {
				t.Fatalf("containsTokenBoundary(%q,%q)=%v, want %v", tc.cmd, tc.needle, got, tc.want)
			}
		})
	}
}

func TestBashCommandDenied(t *testing.T) {
	offending, ok := bashCommandDenied("cat ./config.yaml", []string{"config.yaml"})
	if !ok {
		t.Fatal("expected denied=true")
	}
	if offending != "config.yaml" {
		t.Fatalf("expected offending=config.yaml, got %q", offending)
	}

	if _, ok := bashCommandDenied("echo hello", []string{"config.yaml"}); ok {
		t.Fatal("expected allowed command")
	}
}
