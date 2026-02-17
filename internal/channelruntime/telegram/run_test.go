package telegram

import "testing"

func TestNormalizeAllowedChatIDs(t *testing.T) {
	got := normalizeAllowedChatIDs([]int64{1, 0, 2, 1})
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (%#v)", len(got), got)
	}
	if got[0] != 1 || got[1] != 2 {
		t.Fatalf("got = %#v, want [1 2]", got)
	}
}

func TestNormalizeRunStringSlice(t *testing.T) {
	got := normalizeRunStringSlice([]string{" /ip4/1 ", "", " /ip4/2 ", "/ip4/1"})
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (%#v)", len(got), got)
	}
	if got[0] != "/ip4/1" || got[1] != "/ip4/2" {
		t.Fatalf("got = %#v, want [/ip4/1 /ip4/2]", got)
	}
}
