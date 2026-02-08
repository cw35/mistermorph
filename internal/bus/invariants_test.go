package bus

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInvariantConversationOrder(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 16, Logger: newTestLogger()})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	var (
		mu   sync.Mutex
		seen []seenPair
		done = make(chan struct{})
	)
	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error {
		mu.Lock()
		seen = append(seen, seenPair{conv: msg.ConversationKey, id: msg.ID})
		if len(seen) == 6 {
			close(done)
		}
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	messages := []BusMessage{
		testMessageForConversation(t, "conv:x", "x1", "ix1"),
		testMessageForConversation(t, "conv:y", "y1", "iy1"),
		testMessageForConversation(t, "conv:x", "x2", "ix2"),
		testMessageForConversation(t, "conv:y", "y2", "iy2"),
		testMessageForConversation(t, "conv:x", "x3", "ix3"),
		testMessageForConversation(t, "conv:y", "y3", "iy3"),
	}
	for _, msg := range messages {
		if err := b.Publish(context.Background(), msg); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for invariant order check")
	}
	mu.Lock()
	defer mu.Unlock()
	if got := extractIDs(seen, "conv:x"); got != "x1,x2,x3" {
		t.Fatalf("conv:x order mismatch: got %s", got)
	}
	if got := extractIDs(seen, "conv:y"); got != "y1,y2,y3" {
		t.Fatalf("conv:y order mismatch: got %s", got)
	}
}

func TestInvariantBackpressure(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 1, Logger: newTestLogger()})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	block := make(chan struct{})
	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error {
		<-block
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	first := testMessageForConversation(t, "conv:block", "blk1", "idem_blk1")
	if err := b.Publish(context.Background(), first); err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	second := testMessageForConversation(t, "conv:block", "blk2", "idem_blk2")
	err = b.Publish(ctx, second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Publish(second) error = %v, want context deadline exceeded", err)
	}
	close(block)
}

func TestInvariantTopicSingleHandler(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 2, Logger: newTestLogger()})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error { return nil }); err != nil {
		t.Fatalf("Subscribe(first) error = %v", err)
	}
	err = b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error { return nil })
	if err == nil || !strings.Contains(err.Error(), ErrTopicAlreadyHandled.Error()) {
		t.Fatalf("Subscribe(second) error = %v, want ErrTopicAlreadyHandled", err)
	}
}
