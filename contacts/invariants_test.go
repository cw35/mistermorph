package contacts

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"
)

func TestInvariantOutboxIdempotency(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "contacts")
	store := NewFileStore(root)
	svc := NewService(store)
	now := time.Date(2026, 2, 8, 21, 0, 0, 0, time.UTC)

	if _, err := svc.UpsertContact(ctx, Contact{
		ContactID:          "maep:inv",
		Kind:               KindAgent,
		Status:             StatusActive,
		PeerID:             "12D3KooWInv",
		TrustState:         "verified",
		UnderstandingDepth: 50,
		ReciprocityNorm:    0.7,
		TopicWeights: map[string]float64{
			"maep": 0.9,
		},
	}, now); err != nil {
		t.Fatalf("UpsertContact() error = %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte("hello"))
	if _, err := svc.AddCandidate(ctx, ShareCandidate{
		ItemID:        "cand-inv-1",
		Topic:         "maep",
		ContentType:   "text/plain",
		PayloadBase64: payload,
	}, now); err != nil {
		t.Fatalf("AddCandidate() error = %v", err)
	}

	sender := &mockSender{accepted: true}
	if _, err := svc.RunTick(ctx, now, TickOptions{
		MaxTargets:      1,
		FreshnessWindow: 72 * time.Hour,
		Send:            true,
	}, sender); err != nil {
		t.Fatalf("RunTick(first) error = %v", err)
	}
	second, err := svc.RunTick(ctx, now.Add(1*time.Minute), TickOptions{
		MaxTargets:      1,
		FreshnessWindow: 72 * time.Hour,
		Send:            true,
	}, sender)
	if err != nil {
		t.Fatalf("RunTick(second) error = %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("sender calls mismatch: got %d want 1", sender.calls)
	}
	if len(second.Outcomes) == 0 || !second.Outcomes[0].Deduped {
		t.Fatalf("second run expected deduped outcome, got=%v", second.Outcomes)
	}
}
