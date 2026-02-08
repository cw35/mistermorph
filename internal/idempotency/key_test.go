package idempotency

import (
	"strings"
	"testing"
)

func TestManualContactKey_Format(t *testing.T) {
	key := ManualContactKey("tg:@Alice")
	if !strings.HasPrefix(key, "manual:tg__alice:") {
		t.Fatalf("ManualContactKey() = %q, want prefix manual:tg__alice:", key)
	}
}

func TestProactiveShareKey_Deterministic(t *testing.T) {
	got := ProactiveShareKey("maep:Peer-A", "cand-1", 42, "group")
	want := "proactive:maep_peer_a:cand_1:42:group"
	if got != want {
		t.Fatalf("ProactiveShareKey() = %q, want %q", got, want)
	}
}

func TestMessageEnvelopeKey_UsesSharedAlgorithm(t *testing.T) {
	got := MessageEnvelopeKey("MSG-001")
	want := "msg:msg_001"
	if got != want {
		t.Fatalf("MessageEnvelopeKey() = %q, want %q", got, want)
	}
}
