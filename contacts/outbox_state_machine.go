package contacts

import (
	"fmt"
	"strings"
	"time"
)

type OutboxTransitionType string

const (
	OutboxTransitionStartAttempt OutboxTransitionType = "start_attempt"
	OutboxTransitionMarkSent     OutboxTransitionType = "mark_sent"
	OutboxTransitionMarkFailed   OutboxTransitionType = "mark_failed"
)

type OutboxTransition struct {
	Type      OutboxTransitionType
	Accepted  bool
	Deduped   bool
	ErrorText string
}

func NextOutboxRecord(current *BusOutboxRecord, base BusOutboxRecord, transition OutboxTransition, now time.Time) (BusOutboxRecord, error) {
	if _, err := busOutboxRecordKey(base.Channel, base.IdempotencyKey); err != nil {
		return BusOutboxRecord{}, err
	}
	if current != nil {
		if current.Channel != base.Channel || current.IdempotencyKey != base.IdempotencyKey {
			return BusOutboxRecord{}, fmt.Errorf("outbox identity mismatch")
		}
	}
	switch transition.Type {
	case OutboxTransitionStartAttempt:
		return nextOutboxStartAttempt(current, base, now)
	case OutboxTransitionMarkSent:
		return nextOutboxMarkSent(current, transition, now)
	case OutboxTransitionMarkFailed:
		return nextOutboxMarkFailed(current, transition, now)
	default:
		return BusOutboxRecord{}, fmt.Errorf("unsupported outbox transition: %q", transition.Type)
	}
}

func nextOutboxStartAttempt(current *BusOutboxRecord, base BusOutboxRecord, now time.Time) (BusOutboxRecord, error) {
	attempts := 1
	createdAt := now
	if current != nil {
		switch current.Status {
		case BusDeliveryStatusPending, BusDeliveryStatusFailed:
		case BusDeliveryStatusSent:
			return BusOutboxRecord{}, fmt.Errorf("cannot start attempt from status=%s", current.Status)
		default:
			return BusOutboxRecord{}, fmt.Errorf("unsupported current outbox status: %q", current.Status)
		}
		attempts = current.Attempts + 1
		createdAt = current.CreatedAt.UTC()
	}

	record := base
	record.Status = BusDeliveryStatusPending
	record.Attempts = attempts
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	lastAttemptAt := now
	record.LastAttemptAt = &lastAttemptAt
	record.SentAt = nil
	record.LastError = ""
	record.Accepted = false
	record.Deduped = false
	return record, nil
}

func nextOutboxMarkSent(current *BusOutboxRecord, transition OutboxTransition, now time.Time) (BusOutboxRecord, error) {
	if current == nil {
		return BusOutboxRecord{}, fmt.Errorf("cannot mark sent without pending record")
	}
	if current.Status != BusDeliveryStatusPending {
		return BusOutboxRecord{}, fmt.Errorf("cannot mark sent from status=%s", current.Status)
	}
	record := *current
	record.Status = BusDeliveryStatusSent
	record.Accepted = transition.Accepted
	record.Deduped = transition.Deduped
	record.LastError = ""
	record.UpdatedAt = now
	sentAt := now
	record.SentAt = &sentAt
	return record, nil
}

func nextOutboxMarkFailed(current *BusOutboxRecord, transition OutboxTransition, now time.Time) (BusOutboxRecord, error) {
	if current == nil {
		return BusOutboxRecord{}, fmt.Errorf("cannot mark failed without pending record")
	}
	if current.Status != BusDeliveryStatusPending {
		return BusOutboxRecord{}, fmt.Errorf("cannot mark failed from status=%s", current.Status)
	}
	transition.ErrorText = strings.TrimSpace(transition.ErrorText)
	if transition.ErrorText == "" {
		return BusOutboxRecord{}, fmt.Errorf("error_text is required for failed transition")
	}
	record := *current
	record.Status = BusDeliveryStatusFailed
	record.LastError = transition.ErrorText
	record.Accepted = false
	record.Deduped = false
	record.UpdatedAt = now
	record.SentAt = nil
	return record, nil
}
