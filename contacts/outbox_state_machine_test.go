package contacts

import (
	"testing"
	"time"
)

func TestNextOutboxRecord_StartAttemptFromNew(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	base := testOutboxBase()
	record, err := NextOutboxRecord(nil, base, OutboxTransition{Type: OutboxTransitionStartAttempt}, now)
	if err != nil {
		t.Fatalf("NextOutboxRecord() error = %v", err)
	}
	if record.Status != BusDeliveryStatusPending {
		t.Fatalf("status mismatch: got %s want %s", record.Status, BusDeliveryStatusPending)
	}
	if record.Attempts != 1 {
		t.Fatalf("attempts mismatch: got %d want 1", record.Attempts)
	}
	if record.LastAttemptAt == nil || !record.LastAttemptAt.Equal(now) {
		t.Fatalf("last_attempt_at mismatch")
	}
}

func TestNextOutboxRecord_StartAttemptFromFailed(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	current := testOutboxBase()
	current.Status = BusDeliveryStatusFailed
	current.Attempts = 2
	current.CreatedAt = now.Add(-2 * time.Hour)

	record, err := NextOutboxRecord(&current, testOutboxBase(), OutboxTransition{Type: OutboxTransitionStartAttempt}, now)
	if err != nil {
		t.Fatalf("NextOutboxRecord() error = %v", err)
	}
	if record.Attempts != 3 {
		t.Fatalf("attempts mismatch: got %d want 3", record.Attempts)
	}
	if !record.CreatedAt.Equal(current.CreatedAt) {
		t.Fatalf("created_at mismatch")
	}
}

func TestNextOutboxRecord_RejectStartFromSent(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	current := testOutboxBase()
	current.Status = BusDeliveryStatusSent
	current.Attempts = 5
	if _, err := NextOutboxRecord(&current, testOutboxBase(), OutboxTransition{Type: OutboxTransitionStartAttempt}, now); err == nil {
		t.Fatalf("NextOutboxRecord() expected error")
	}
}

func TestNextOutboxRecord_MarkSent(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	pending := testOutboxBase()
	pending.Status = BusDeliveryStatusPending
	pending.Attempts = 1

	record, err := NextOutboxRecord(&pending, testOutboxBase(), OutboxTransition{
		Type:     OutboxTransitionMarkSent,
		Accepted: true,
		Deduped:  false,
	}, now)
	if err != nil {
		t.Fatalf("NextOutboxRecord() error = %v", err)
	}
	if record.Status != BusDeliveryStatusSent {
		t.Fatalf("status mismatch: got %s want %s", record.Status, BusDeliveryStatusSent)
	}
	if !record.Accepted || record.Deduped {
		t.Fatalf("accepted/deduped mismatch")
	}
	if record.SentAt == nil || !record.SentAt.Equal(now) {
		t.Fatalf("sent_at mismatch")
	}
}

func TestNextOutboxRecord_MarkFailed(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	pending := testOutboxBase()
	pending.Status = BusDeliveryStatusPending
	pending.Attempts = 1

	record, err := NextOutboxRecord(&pending, testOutboxBase(), OutboxTransition{
		Type:      OutboxTransitionMarkFailed,
		ErrorText: "network timeout",
	}, now)
	if err != nil {
		t.Fatalf("NextOutboxRecord() error = %v", err)
	}
	if record.Status != BusDeliveryStatusFailed {
		t.Fatalf("status mismatch: got %s want %s", record.Status, BusDeliveryStatusFailed)
	}
	if record.LastError != "network timeout" {
		t.Fatalf("last_error mismatch: got %q", record.LastError)
	}
}

func TestNextOutboxRecord_RejectFailedWithoutError(t *testing.T) {
	now := time.Date(2026, 2, 8, 22, 0, 0, 0, time.UTC)
	pending := testOutboxBase()
	pending.Status = BusDeliveryStatusPending
	if _, err := NextOutboxRecord(&pending, testOutboxBase(), OutboxTransition{
		Type: OutboxTransitionMarkFailed,
	}, now); err == nil {
		t.Fatalf("NextOutboxRecord() expected error")
	}
}

func testOutboxBase() BusOutboxRecord {
	return BusOutboxRecord{
		Channel:        ChannelTelegram,
		IdempotencyKey: "manual:k1",
		ContactID:      "tg:1001",
		ItemID:         "item-1",
		ContentType:    "text/plain",
		PayloadBase64:  "aGVsbG8",
	}
}
