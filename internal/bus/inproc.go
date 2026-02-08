package bus

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrBusClosed            = errors.New("bus is closed")
	ErrNoSubscriberForTopic = errors.New("no subscriber for topic")
	ErrTopicAlreadyHandled  = errors.New("topic already has handler")
	ErrTopicFrozen          = errors.New("topic registry is frozen")
)

type HandlerFunc func(ctx context.Context, msg BusMessage) error

type DeliveryError struct {
	Message    BusMessage `json:"message"`
	Topic      string     `json:"topic"`
	ErrorText  string     `json:"error"`
	OccurredAt time.Time  `json:"occurred_at"`
}

type InprocOptions struct {
	MaxInFlight int
	Logger      *slog.Logger
}

type Inproc struct {
	maxInFlight int
	logger      *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	done      chan struct{}
	tokens    chan struct{}
	errs      chan DeliveryError
	closeOnce sync.Once

	mu          sync.RWMutex
	closed      bool
	started     bool
	subscribers map[string]HandlerFunc
	shards      []chan BusMessage

	wg sync.WaitGroup
}

func NewInproc(opts InprocOptions) (*Inproc, error) {
	if opts.MaxInFlight <= 0 {
		return nil, fmt.Errorf("max_inflight must be > 0")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	shardCount := deriveShardCount(opts.MaxInFlight)
	logger := opts.Logger
	ctx, cancel := context.WithCancel(context.Background())
	shards := make([]chan BusMessage, shardCount)
	for i := range shards {
		shards[i] = make(chan BusMessage, opts.MaxInFlight)
	}
	b := &Inproc{
		maxInFlight: opts.MaxInFlight,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		tokens:      make(chan struct{}, opts.MaxInFlight),
		errs:        make(chan DeliveryError, opts.MaxInFlight),
		subscribers: make(map[string]HandlerFunc),
		shards:      shards,
	}
	for i := 0; i < opts.MaxInFlight; i++ {
		b.tokens <- struct{}{}
	}
	b.logger.Debug(
		"bus_inproc_initialized",
		"max_inflight", opts.MaxInFlight,
		"shard_count", shardCount,
		"shard_queue_capacity", opts.MaxInFlight,
	)
	for shard := range b.shards {
		b.wg.Add(1)
		go b.runShardWorker(shard, b.shards[shard])
	}
	return b, nil
}

func (b *Inproc) Errors() <-chan DeliveryError {
	return b.errs
}

func (b *Inproc) Subscribe(topic string, handler HandlerFunc) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := ValidateTopic(topic); err != nil {
		return wrapError(CodeInvalidTopic, err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return wrapError(CodeBusClosed, ErrBusClosed)
	}
	if b.started {
		return wrapError(CodeTopicFrozen, ErrTopicFrozen)
	}
	if _, exists := b.subscribers[topic]; exists {
		return wrapError(CodeTopicAlreadyHandled, fmt.Errorf("%w: %s", ErrTopicAlreadyHandled, topic))
	}
	b.subscribers[topic] = handler
	b.logger.Debug("bus_subscribe", "topic", topic)
	return nil
}

func (b *Inproc) PublishValidated(ctx context.Context, msg BusMessage) error {
	if err := msg.Validate(); err != nil {
		return wrapError(CodeInvalidMessage, err)
	}
	return b.Publish(ctx, msg)
}

func (b *Inproc) Publish(ctx context.Context, msg BusMessage) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	// Publish runs on the internal fast path and expects boundary adapters
	// to validate BusMessage before calling into the bus.
	if err := b.preparePublish(msg.Topic); err != nil {
		return err
	}

	shardIndex, shardQueue, err := b.shardQueue(msg.ConversationKey)
	if err != nil {
		return err
	}
	b.logger.Debug("bus_publish_start",
		"id", msg.ID,
		"topic", msg.Topic,
		"channel", msg.Channel,
		"conversation_key", msg.ConversationKey,
		"idempotency_key", msg.IdempotencyKey,
		"correlation_id", msg.CorrelationID,
		"shard", shardIndex,
		"shard_queue_depth", len(shardQueue),
		"in_flight", b.maxInFlight-len(b.tokens),
	)
	if len(b.tokens) == 0 {
		b.logger.Debug("bus_publish_backpressure_wait",
			"id", msg.ID,
			"topic", msg.Topic,
			"conversation_key", msg.ConversationKey,
			"shard", shardIndex,
		)
	}

	select {
	case <-ctx.Done():
		return publishCtxError(ctx.Err())
	case <-b.done:
		return wrapError(CodeBusClosed, ErrBusClosed)
	case <-b.tokens:
	}

	select {
	case <-ctx.Done():
		b.releaseToken()
		return publishCtxError(ctx.Err())
	case <-b.done:
		b.releaseToken()
		return wrapError(CodeBusClosed, ErrBusClosed)
	case shardQueue <- msg:
		b.logger.Debug("bus_publish_enqueued",
			"id", msg.ID,
			"topic", msg.Topic,
			"conversation_key", msg.ConversationKey,
			"shard", shardIndex,
			"shard_queue_depth", len(shardQueue),
			"in_flight", b.maxInFlight-len(b.tokens),
		)
		return nil
	}
}

func publishCtxError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return wrapError(CodeQueueFull, err)
	}
	return err
}

func (b *Inproc) Close() error {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.closed = true
		b.mu.Unlock()
		close(b.done)
		b.cancel()
		b.logger.Debug("bus_close_requested")
	})
	b.wg.Wait()
	close(b.errs)
	b.logger.Debug("bus_closed")
	return nil
}

func (b *Inproc) preparePublish(topic string) error {
	if err := ValidateTopic(topic); err != nil {
		return wrapError(CodeInvalidTopic, err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return wrapError(CodeBusClosed, ErrBusClosed)
	}
	if !b.started {
		b.started = true
		b.logger.Debug("bus_topic_registry_frozen")
	}
	handler, ok := b.subscribers[topic]
	if !ok || handler == nil {
		return wrapError(CodeNoSubscriber, fmt.Errorf("%w: %s", ErrNoSubscriberForTopic, topic))
	}
	return nil
}

func (b *Inproc) shardQueue(conversationKey string) (int, chan BusMessage, error) {
	if err := validateRequiredCanonicalString("conversation_key", conversationKey); err != nil {
		return 0, nil, err
	}
	if len(b.shards) == 0 {
		return 0, nil, fmt.Errorf("bus shards are not initialized")
	}
	index := shardIndexFor(conversationKey, len(b.shards))
	return index, b.shards[index], nil
}

func (b *Inproc) runShardWorker(index int, queue chan BusMessage) {
	defer b.wg.Done()
	b.logger.Debug("bus_shard_worker_started", "shard", index)
	for {
		select {
		case <-b.done:
			b.logger.Debug("bus_shard_worker_stopped", "shard", index)
			return
		case msg := <-queue:
			err := b.deliver(index, msg)
			b.releaseToken()
			if err != nil {
				b.reportDeliveryError(msg, err)
			}
		}
	}
}

func (b *Inproc) deliver(shard int, msg BusMessage) error {
	handler, err := b.subscriberForTopic(msg.Topic)
	if err != nil {
		return err
	}
	if err := handler(b.ctx, msg); err != nil {
		return err
	}
	b.logger.Debug("bus_deliver_ok",
		"id", msg.ID,
		"topic", msg.Topic,
		"channel", msg.Channel,
		"conversation_key", msg.ConversationKey,
		"idempotency_key", msg.IdempotencyKey,
		"correlation_id", msg.CorrelationID,
		"shard", shard,
	)
	return nil
}

func (b *Inproc) subscriberForTopic(topic string) (HandlerFunc, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	handler, ok := b.subscribers[topic]
	if !ok {
		return nil, wrapError(CodeNoSubscriber, fmt.Errorf("%w: %s", ErrNoSubscriberForTopic, topic))
	}
	return handler, nil
}

func (b *Inproc) releaseToken() {
	select {
	case b.tokens <- struct{}{}:
	case <-b.done:
	}
}

func (b *Inproc) reportDeliveryError(msg BusMessage, err error) {
	b.logger.Warn("bus_deliver_failed",
		"id", msg.ID,
		"topic", msg.Topic,
		"channel", msg.Channel,
		"conversation_key", msg.ConversationKey,
		"idempotency_key", msg.IdempotencyKey,
		"correlation_id", msg.CorrelationID,
		"error", err.Error(),
	)
	select {
	case <-b.done:
	case b.errs <- DeliveryError{
		Message:    msg,
		Topic:      msg.Topic,
		ErrorText:  err.Error(),
		OccurredAt: time.Now().UTC(),
	}:
	}
}

func deriveShardCount(maxInFlight int) int {
	const defaultShardCount = 16
	if maxInFlight <= defaultShardCount {
		return maxInFlight
	}
	return defaultShardCount
}

func shardIndexFor(conversationKey string, shardCount int) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(conversationKey))
	return int(hasher.Sum32() % uint32(shardCount))
}
