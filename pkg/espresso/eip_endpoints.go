package espresso

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// IdempotentReceiverImpl ensures messages are processed only once
type IdempotentReceiverImpl[T any] struct {
	name         string
	processedIDs map[string]time.Time
	ttl          time.Duration
	mu           sync.RWMutex
}

// NewIdempotentReceiver creates a new idempotent receiver
func NewIdempotentReceiver[T any](name string, ttl time.Duration) *IdempotentReceiverImpl[T] {
	receiver := &IdempotentReceiverImpl[T]{
		name:         name,
		processedIDs: make(map[string]time.Time),
		ttl:          ttl,
	}

	// Start cleanup goroutine
	go receiver.cleanupExpired()

	return receiver
}

// Receive receives and processes a message idempotently
func (r *IdempotentReceiverImpl[T]) Receive(ctx context.Context, msg *Message[T], handler MessageHandler[T, T]) error {
	r.mu.Lock()
	processTime, alreadyProcessed := r.processedIDs[msg.ID]
	if alreadyProcessed {
		r.mu.Unlock()
		msg.AddHistoryEntry(r.name, "idempotent_duplicate_detected", map[string]interface{}{
			"original_process_time": processTime,
		})
		return nil // Silently ignore duplicate
	}

	// Mark as processing
	r.processedIDs[msg.ID] = time.Now()
	r.mu.Unlock()

	msg.AddHistoryEntry(r.name, "idempotent_processing", nil)

	// Process the message
	_, err := handler.Handle(ctx, msg)
	if err != nil {
		// Remove from processed if handler fails
		r.mu.Lock()
		delete(r.processedIDs, msg.ID)
		r.mu.Unlock()
		return fmt.Errorf("handler failed: %w", err)
	}

	msg.AddHistoryEntry(r.name, "idempotent_processed", nil)
	return nil
}

// IsProcessed checks if a message has already been processed
func (r *IdempotentReceiverImpl[T]) IsProcessed(messageID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, processed := r.processedIDs[messageID]
	return processed
}

// cleanupExpired removes expired message IDs
func (r *IdempotentReceiverImpl[T]) cleanupExpired() {
	ticker := time.NewTicker(r.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for id, processTime := range r.processedIDs {
			if now.Sub(processTime) > r.ttl {
				delete(r.processedIDs, id)
			}
		}
		r.mu.Unlock()
	}
}

// Size returns the number of tracked message IDs
func (r *IdempotentReceiverImpl[T]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.processedIDs)
}

// CompetingConsumersImpl manages multiple consumers competing for messages
type CompetingConsumersImpl[T any] struct {
	name       string
	consumers  map[string]*consumer[T]
	stats      ConsumerPoolStats
	running    atomic.Bool
	stopChan   chan struct{}
	mu         sync.RWMutex
	nextID     int
}

type consumer[T any] struct {
	id        string
	handler   MessageHandler[T, T]
	processed atomic.Int64
	errors    atomic.Int64
	totalTime atomic.Int64
}

// NewCompetingConsumers creates a new competing consumers pool
func NewCompetingConsumers[T any](name string) *CompetingConsumersImpl[T] {
	return &CompetingConsumersImpl[T]{
		name:      name,
		consumers: make(map[string]*consumer[T]),
		stopChan:  make(chan struct{}),
	}
}

// AddConsumer adds a consumer to the pool
func (c *CompetingConsumersImpl[T]) AddConsumer(handler MessageHandler[T, T]) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	consumerID := fmt.Sprintf("%s-consumer-%d", c.name, c.nextID)

	c.consumers[consumerID] = &consumer[T]{
		id:      consumerID,
		handler: handler,
	}

	return nil
}

// RemoveConsumer removes a consumer from the pool
func (c *CompetingConsumersImpl[T]) RemoveConsumer(consumerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.consumers[consumerID]; !exists {
		return fmt.Errorf("consumer not found: %s", consumerID)
	}

	delete(c.consumers, consumerID)
	return nil
}

// Start starts all consumers
func (c *CompetingConsumersImpl[T]) Start(ctx context.Context, channel Channel[T]) error {
	if c.running.Load() {
		return errors.New("consumers already running")
	}

	c.running.Store(true)

	c.mu.RLock()
	consumerList := make([]*consumer[T], 0, len(c.consumers))
	for _, consumer := range c.consumers {
		consumerList = append(consumerList, consumer)
	}
	c.mu.RUnlock()

	// Start each consumer in a goroutine
	for _, cons := range consumerList {
		go c.runConsumer(ctx, cons, channel)
	}

	return nil
}

// runConsumer runs a single consumer
func (c *CompetingConsumersImpl[T]) runConsumer(ctx context.Context, cons *consumer[T], channel Channel[T]) {
	for {
		select {
		case <-c.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			// Try to receive a message
			msg, ok, err := channel.TryReceive(ctx)
			if err != nil {
				cons.errors.Add(1)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if !ok {
				// No message available, wait a bit
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Process the message
			start := time.Now()
			msg.AddHistoryEntry(c.name, "competing_consumer_processing", map[string]interface{}{
				"consumer_id": cons.id,
			})

			_, err = cons.handler.Handle(ctx, msg)
			duration := time.Since(start)

			if err != nil {
				cons.errors.Add(1)
				msg.AddHistoryEntry(c.name, "competing_consumer_error", map[string]interface{}{
					"consumer_id": cons.id,
					"error":       err.Error(),
				})
			} else {
				cons.processed.Add(1)
				cons.totalTime.Add(int64(duration))
				msg.AddHistoryEntry(c.name, "competing_consumer_success", map[string]interface{}{
					"consumer_id": cons.id,
					"duration_ms": duration.Milliseconds(),
				})
			}
		}
	}
}

// Stop stops all consumers
func (c *CompetingConsumersImpl[T]) Stop() error {
	if !c.running.Load() {
		return errors.New("consumers not running")
	}

	c.running.Store(false)
	close(c.stopChan)

	return nil
}

// GetStats returns statistics about the consumer pool
func (c *CompetingConsumersImpl[T]) GetStats() ConsumerPoolStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalProcessed int64
	var totalErrors int64
	var totalTime int64

	for _, cons := range c.consumers {
		totalProcessed += cons.processed.Load()
		totalErrors += cons.errors.Load()
		totalTime += cons.totalTime.Load()
	}

	avgProcessTime := time.Duration(0)
	if totalProcessed > 0 {
		avgProcessTime = time.Duration(totalTime / totalProcessed)
	}

	return ConsumerPoolStats{
		ActiveConsumers: len(c.consumers),
		TotalProcessed:  totalProcessed,
		TotalErrors:     totalErrors,
		AvgProcessTime:  avgProcessTime,
	}
}

// PollingConsumer polls a channel at regular intervals
type PollingConsumer[T any] struct {
	name     string
	channel  Channel[T]
	handler  MessageHandler[T, T]
	interval time.Duration
	running  atomic.Bool
	stopChan chan struct{}
}

// NewPollingConsumer creates a new polling consumer
func NewPollingConsumer[T any](
	name string,
	channel Channel[T],
	handler MessageHandler[T, T],
	interval time.Duration,
) *PollingConsumer[T] {
	return &PollingConsumer[T]{
		name:     name,
		channel:  channel,
		handler:  handler,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start starts polling
func (p *PollingConsumer[T]) Start(ctx context.Context) error {
	if p.running.Load() {
		return errors.New("already running")
	}

	p.running.Store(true)

	go p.poll(ctx)

	return nil
}

// poll polls the channel
func (p *PollingConsumer[T]) poll(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Try to receive a message
			msg, ok, err := p.channel.TryReceive(ctx)
			if err != nil || !ok {
				continue
			}

			msg.AddHistoryEntry(p.name, "polling_consumer_received", nil)

			// Process the message
			_, err = p.handler.Handle(ctx, msg)
			if err != nil {
				msg.AddHistoryEntry(p.name, "polling_consumer_error", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				msg.AddHistoryEntry(p.name, "polling_consumer_success", nil)
			}
		}
	}
}

// Stop stops polling
func (p *PollingConsumer[T]) Stop(ctx context.Context) error {
	if !p.running.Load() {
		return errors.New("not running")
	}

	p.running.Store(false)
	close(p.stopChan)

	return nil
}

// IsRunning returns true if polling
func (p *PollingConsumer[T]) IsRunning() bool {
	return p.running.Load()
}

// EventDrivenConsumer consumes messages as they arrive
type EventDrivenConsumer[T any] struct {
	name     string
	channel  Channel[T]
	handler  MessageHandler[T, T]
	running  atomic.Bool
	stopChan chan struct{}
}

// NewEventDrivenConsumer creates a new event-driven consumer
func NewEventDrivenConsumer[T any](
	name string,
	channel Channel[T],
	handler MessageHandler[T, T],
) *EventDrivenConsumer[T] {
	return &EventDrivenConsumer[T]{
		name:     name,
		channel:  channel,
		handler:  handler,
		stopChan: make(chan struct{}),
	}
}

// Start starts consuming
func (e *EventDrivenConsumer[T]) Start(ctx context.Context) error {
	if e.running.Load() {
		return errors.New("already running")
	}

	e.running.Store(true)

	go e.consume(ctx)

	return nil
}

// consume consumes messages
func (e *EventDrivenConsumer[T]) consume(ctx context.Context) {
	for {
		select {
		case <-e.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			// Blocking receive
			msg, err := e.channel.Receive(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}

			msg.AddHistoryEntry(e.name, "event_driven_consumer_received", nil)

			// Process the message
			_, err = e.handler.Handle(ctx, msg)
			if err != nil {
				msg.AddHistoryEntry(e.name, "event_driven_consumer_error", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				msg.AddHistoryEntry(e.name, "event_driven_consumer_success", nil)
			}
		}
	}
}

// Stop stops consuming
func (e *EventDrivenConsumer[T]) Stop(ctx context.Context) error {
	if !e.running.Load() {
		return errors.New("not running")
	}

	e.running.Store(false)
	close(e.stopChan)

	return nil
}

// IsRunning returns true if consuming
func (e *EventDrivenConsumer[T]) IsRunning() bool {
	return e.running.Load()
}

// ServiceActivator invokes a service in response to a message
type ServiceActivator[T any, R any] struct {
	name          string
	inputChannel  Channel[T]
	outputChannel Channel[R]
	service       MessageHandler[T, R]
	running       atomic.Bool
	stopChan      chan struct{}
}

// NewServiceActivator creates a new service activator
func NewServiceActivator[T any, R any](
	name string,
	inputChannel Channel[T],
	outputChannel Channel[R],
	service MessageHandler[T, R],
) *ServiceActivator[T, R] {
	return &ServiceActivator[T, R]{
		name:          name,
		inputChannel:  inputChannel,
		outputChannel: outputChannel,
		service:       service,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the service activator
func (s *ServiceActivator[T, R]) Start(ctx context.Context) error {
	if s.running.Load() {
		return errors.New("already running")
	}

	s.running.Store(true)

	go s.activate(ctx)

	return nil
}

// activate activates the service
func (s *ServiceActivator[T, R]) activate(ctx context.Context) {
	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := s.inputChannel.Receive(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}

			msg.AddHistoryEntry(s.name, "service_activator_invoke", nil)

			// Invoke the service
			result, err := s.service.Handle(ctx, msg)
			if err != nil {
				msg.AddHistoryEntry(s.name, "service_activator_error", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			result.AddHistoryEntry(s.name, "service_activator_success", nil)

			// Send result to output channel
			if s.outputChannel != nil {
				_ = s.outputChannel.Send(ctx, result)
			}
		}
	}
}

// Stop stops the service activator
func (s *ServiceActivator[T, R]) Stop(ctx context.Context) error {
	if !s.running.Load() {
		return errors.New("not running")
	}

	s.running.Store(false)
	close(s.stopChan)

	return nil
}

// IsRunning returns true if running
func (s *ServiceActivator[T, R]) IsRunning() bool {
	return s.running.Load()
}

// MessageDispatcher distributes messages to performers
type MessageDispatcher[T any] struct {
	name       string
	performers []MessageHandler[T, T]
	strategy   DispatchStrategy
	currentIdx atomic.Int32
	mu         sync.RWMutex
}

// DispatchStrategy defines how messages are dispatched
type DispatchStrategy int

const (
	// RoundRobin distributes messages in round-robin fashion
	RoundRobin DispatchStrategy = iota
	// Random distributes messages randomly
	Random
	// LeastBusy distributes to the least busy performer (not implemented in this simple version)
	LeastBusy
)

// NewMessageDispatcher creates a new message dispatcher
func NewMessageDispatcher[T any](name string, strategy DispatchStrategy) *MessageDispatcher[T] {
	return &MessageDispatcher[T]{
		name:       name,
		performers: make([]MessageHandler[T, T], 0),
		strategy:   strategy,
	}
}

// AddPerformer adds a performer
func (d *MessageDispatcher[T]) AddPerformer(performer MessageHandler[T, T]) *MessageDispatcher[T] {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.performers = append(d.performers, performer)
	return d
}

// Dispatch dispatches a message to a performer
func (d *MessageDispatcher[T]) Dispatch(ctx context.Context, msg *Message[T]) (*Message[T], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.performers) == 0 {
		return nil, errors.New("no performers available")
	}

	var performer MessageHandler[T, T]

	switch d.strategy {
	case RoundRobin:
		idx := d.currentIdx.Add(1) - 1
		performer = d.performers[int(idx)%len(d.performers)]
	case Random:
		// Simple random - in production use crypto/rand
		idx := time.Now().UnixNano() % int64(len(d.performers))
		performer = d.performers[idx]
	default:
		performer = d.performers[0]
	}

	msg.AddHistoryEntry(d.name, "message_dispatcher_dispatch", map[string]interface{}{
		"strategy": d.strategy,
	})

	return performer.Handle(ctx, msg)
}

// SelectiveConsumer filters messages before consuming
type SelectiveConsumer[T any] struct {
	name      string
	channel   Channel[T]
	filter    MessagePredicate[T]
	handler   MessageHandler[T, T]
	rejected  Channel[T]
	running   atomic.Bool
	stopChan  chan struct{}
}

// NewSelectiveConsumer creates a new selective consumer
func NewSelectiveConsumer[T any](
	name string,
	channel Channel[T],
	filter MessagePredicate[T],
	handler MessageHandler[T, T],
	rejectedChannel Channel[T],
) *SelectiveConsumer[T] {
	return &SelectiveConsumer[T]{
		name:     name,
		channel:  channel,
		filter:   filter,
		handler:  handler,
		rejected: rejectedChannel,
		stopChan: make(chan struct{}),
	}
}

// Start starts the selective consumer
func (s *SelectiveConsumer[T]) Start(ctx context.Context) error {
	if s.running.Load() {
		return errors.New("already running")
	}

	s.running.Store(true)

	go s.consume(ctx)

	return nil
}

// consume consumes and filters messages
func (s *SelectiveConsumer[T]) consume(ctx context.Context) {
	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := s.channel.Receive(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}

			// Filter the message
			if !s.filter(msg) {
				msg.AddHistoryEntry(s.name, "selective_consumer_rejected", nil)
				if s.rejected != nil {
					_ = s.rejected.Send(ctx, msg)
				}
				continue
			}

			msg.AddHistoryEntry(s.name, "selective_consumer_accepted", nil)

			// Process the message
			_, err = s.handler.Handle(ctx, msg)
			if err != nil {
				msg.AddHistoryEntry(s.name, "selective_consumer_error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// Stop stops the selective consumer
func (s *SelectiveConsumer[T]) Stop(ctx context.Context) error {
	if !s.running.Load() {
		return errors.New("not running")
	}

	s.running.Store(false)
	close(s.stopChan)

	return nil
}

// IsRunning returns true if running
func (s *SelectiveConsumer[T]) IsRunning() bool {
	return s.running.Load()
}
