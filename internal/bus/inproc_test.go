package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type seenPair struct {
	conv string
	id   string
}

func TestInprocPublishSubscribe(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 8})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	var (
		mu   sync.Mutex
		got  []string
		done = make(chan struct{})
	)
	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error {
		mu.Lock()
		got = append(got, msg.ID)
		if len(got) == 3 {
			close(done)
		}
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	base := validMessage(t)
	for i := 0; i < 3; i++ {
		msg := base
		msg.ID = fmt.Sprintf("bus_%d", i+1)
		msg.IdempotencyKey = fmt.Sprintf("idem_%d", i+1)
		if err := b.Publish(context.Background(), msg); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for messages")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("message count mismatch: got %d want 3", len(got))
	}
}

func TestInprocConversationOrder(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 16})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	var (
		mu      sync.Mutex
		seen    = make([]seenPair, 0, 8)
		done    = make(chan struct{})
		seenCnt int
	)
	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, seenPair{conv: msg.ConversationKey, id: msg.ID})
		seenCnt++
		if seenCnt == 6 {
			close(done)
		}
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	messages := []BusMessage{
		testMessageForConversation(t, "conv:a", "a1", "i1"),
		testMessageForConversation(t, "conv:b", "b1", "i2"),
		testMessageForConversation(t, "conv:a", "a2", "i3"),
		testMessageForConversation(t, "conv:b", "b2", "i4"),
		testMessageForConversation(t, "conv:a", "a3", "i5"),
		testMessageForConversation(t, "conv:b", "b3", "i6"),
	}
	for _, msg := range messages {
		if err := b.Publish(context.Background(), msg); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for ordered deliveries")
	}

	mu.Lock()
	defer mu.Unlock()
	if extractIDs(seen, "conv:a") != "a1,a2,a3" {
		t.Fatalf("conv:a order mismatch: got %s", extractIDs(seen, "conv:a"))
	}
	if extractIDs(seen, "conv:b") != "b1,b2,b3" {
		t.Fatalf("conv:b order mismatch: got %s", extractIDs(seen, "conv:b"))
	}
}

func TestInprocBackpressure(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 1})
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

	first := testMessageForConversation(t, "conv:block", "m1", "idem1")
	if err := b.Publish(context.Background(), first); err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	second := testMessageForConversation(t, "conv:block", "m2", "idem2")
	err = b.Publish(ctx, second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Publish(second) error = %v, want context deadline exceeded", err)
	}
	close(block)
}

func TestInprocPublishWithoutSubscriberFails(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 2})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	msg := validMessage(t)
	err = b.Publish(context.Background(), msg)
	if err == nil || !strings.Contains(err.Error(), ErrNoSubscriberForTopic.Error()) {
		t.Fatalf("Publish() error = %v, want ErrNoSubscriberForTopic", err)
	}
}

func TestInprocPublishAfterCloseFails(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 2})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	if err := b.Subscribe(TopicChatMessage, func(ctx context.Context, msg BusMessage) error { return nil }); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	err = b.Publish(context.Background(), validMessage(t))
	if !errors.Is(err, ErrBusClosed) {
		t.Fatalf("Publish() error = %v, want ErrBusClosed", err)
	}
}

func TestRouterHandleAndPublish(t *testing.T) {
	b, err := NewInproc(InprocOptions{MaxInFlight: 4})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer b.Close()

	router, err := NewRouter(b)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	done := make(chan struct{})
	if err := router.Handle(TopicChatMessage, func(ctx context.Context, msg BusMessage) error {
		close(done)
		return nil
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if err := router.Publish(context.Background(), validMessage(t)); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for router delivery")
	}
}

func testMessageForConversation(t *testing.T, conversationKey string, id string, idem string) BusMessage {
	t.Helper()
	msg := validMessage(t)
	msg.ConversationKey = conversationKey
	msg.ID = id
	msg.IdempotencyKey = idem
	return msg
}

func extractIDs(pairs []seenPair, conv string) string {
	out := make([]string, 0, len(pairs))
	for _, item := range pairs {
		if item.conv == conv {
			out = append(out, item.id)
		}
	}
	return strings.Join(out, ",")
}
