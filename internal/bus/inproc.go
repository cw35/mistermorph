package bus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrBusClosed            = errors.New("bus is closed")
	ErrNoSubscriberForTopic = errors.New("no subscriber for topic")
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
	ingress   chan BusMessage
	tokens    chan struct{}
	errs      chan DeliveryError
	closeOnce sync.Once

	mu          sync.RWMutex
	closed      bool
	subscribers map[string][]HandlerFunc
	workers     map[string]*conversationWorker

	wg sync.WaitGroup
}

type conversationWorker struct {
	key   string
	queue chan BusMessage
}

func NewInproc(opts InprocOptions) (*Inproc, error) {
	if opts.MaxInFlight <= 0 {
		return nil, fmt.Errorf("max_inflight must be > 0")
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &Inproc{
		maxInFlight: opts.MaxInFlight,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		ingress:     make(chan BusMessage, opts.MaxInFlight),
		tokens:      make(chan struct{}, opts.MaxInFlight),
		errs:        make(chan DeliveryError, opts.MaxInFlight),
		subscribers: make(map[string][]HandlerFunc),
		workers:     make(map[string]*conversationWorker),
	}
	for i := 0; i < opts.MaxInFlight; i++ {
		b.tokens <- struct{}{}
	}
	b.logger.Debug("bus_inproc_initialized", "max_inflight", opts.MaxInFlight)
	b.wg.Add(1)
	go b.runDispatcher()
	return b, nil
}

func (b *Inproc) Errors() <-chan DeliveryError {
	return b.errs
}

func (b *Inproc) Subscribe(topic string, handler HandlerFunc) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := validateRequiredCanonicalString("topic", topic); err != nil {
		return err
	}
	if !topicPattern.MatchString(topic) {
		return fmt.Errorf("topic is invalid")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrBusClosed
	}
	b.subscribers[topic] = append(b.subscribers[topic], handler)
	b.logger.Debug("bus_subscribe", "topic", topic, "subscriber_count", len(b.subscribers[topic]))
	return nil
}

func (b *Inproc) Publish(ctx context.Context, msg BusMessage) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := msg.Validate(); err != nil {
		return err
	}
	if err := b.ensureHasSubscriber(msg.Topic); err != nil {
		return err
	}
	b.logger.Debug("bus_publish_start",
		"id", msg.ID,
		"topic", msg.Topic,
		"channel", msg.Channel,
		"conversation_key", msg.ConversationKey,
		"idempotency_key", msg.IdempotencyKey,
		"correlation_id", msg.CorrelationID,
		"ingress_depth", len(b.ingress),
		"in_flight", b.maxInFlight-len(b.tokens),
	)
	if len(b.tokens) == 0 {
		b.logger.Debug("bus_publish_backpressure_wait",
			"id", msg.ID,
			"topic", msg.Topic,
			"conversation_key", msg.ConversationKey,
		)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.done:
		return ErrBusClosed
	case <-b.tokens:
	}

	select {
	case <-ctx.Done():
		b.releaseToken()
		return ctx.Err()
	case <-b.done:
		b.releaseToken()
		return ErrBusClosed
	case b.ingress <- msg:
		b.logger.Debug("bus_publish_enqueued",
			"id", msg.ID,
			"topic", msg.Topic,
			"conversation_key", msg.ConversationKey,
			"ingress_depth", len(b.ingress),
			"in_flight", b.maxInFlight-len(b.tokens),
		)
		return nil
	}
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

func (b *Inproc) ensureHasSubscriber(topic string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return ErrBusClosed
	}
	if len(b.subscribers[topic]) == 0 {
		return fmt.Errorf("%w: %s", ErrNoSubscriberForTopic, topic)
	}
	return nil
}

func (b *Inproc) runDispatcher() {
	defer b.wg.Done()
	b.logger.Debug("bus_dispatcher_started")
	for {
		select {
		case <-b.done:
			b.logger.Debug("bus_dispatcher_stopping")
			b.closeWorkerQueues()
			return
		case msg := <-b.ingress:
			if err := b.dispatch(msg); err != nil {
				b.releaseToken()
				b.reportDeliveryError(msg, err)
			}
		}
	}
}

func (b *Inproc) dispatch(msg BusMessage) error {
	worker, err := b.getOrCreateWorker(msg.ConversationKey)
	if err != nil {
		return err
	}
	b.logger.Debug("bus_dispatch",
		"id", msg.ID,
		"topic", msg.Topic,
		"conversation_key", msg.ConversationKey,
		"worker_queue_depth", len(worker.queue),
	)
	select {
	case <-b.done:
		return ErrBusClosed
	case worker.queue <- msg:
		return nil
	}
}

func (b *Inproc) getOrCreateWorker(conversationKey string) (*conversationWorker, error) {
	if err := validateRequiredCanonicalString("conversation_key", conversationKey); err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrBusClosed
	}
	if worker, ok := b.workers[conversationKey]; ok {
		return worker, nil
	}
	worker := &conversationWorker{
		key:   conversationKey,
		queue: make(chan BusMessage, b.maxInFlight),
	}
	b.workers[conversationKey] = worker
	b.logger.Debug("bus_worker_created", "conversation_key", conversationKey, "worker_count", len(b.workers))
	b.wg.Add(1)
	go b.runConversationWorker(worker)
	return worker, nil
}

func (b *Inproc) runConversationWorker(worker *conversationWorker) {
	defer b.wg.Done()
	b.logger.Debug("bus_worker_started", "conversation_key", worker.key)
	for msg := range worker.queue {
		err := b.deliver(msg)
		b.releaseToken()
		if err != nil {
			b.reportDeliveryError(msg, err)
		}
	}
	b.logger.Debug("bus_worker_stopped", "conversation_key", worker.key)
}

func (b *Inproc) deliver(msg BusMessage) error {
	handlers, err := b.subscribersForTopic(msg.Topic)
	if err != nil {
		return err
	}
	for _, handler := range handlers {
		if err := handler(b.ctx, msg); err != nil {
			return err
		}
	}
	b.logger.Debug("bus_deliver_ok",
		"id", msg.ID,
		"topic", msg.Topic,
		"channel", msg.Channel,
		"conversation_key", msg.ConversationKey,
		"idempotency_key", msg.IdempotencyKey,
		"correlation_id", msg.CorrelationID,
		"handler_count", len(handlers),
	)
	return nil
}

func (b *Inproc) subscribersForTopic(topic string) ([]HandlerFunc, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	handlers := b.subscribers[topic]
	if len(handlers) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoSubscriberForTopic, topic)
	}
	out := make([]HandlerFunc, len(handlers))
	copy(out, handlers)
	return out, nil
}

func (b *Inproc) closeWorkerQueues() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.logger.Debug("bus_worker_close_all", "worker_count", len(b.workers))
	for _, worker := range b.workers {
		close(worker.queue)
	}
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
