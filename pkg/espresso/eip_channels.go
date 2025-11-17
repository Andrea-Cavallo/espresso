package espresso

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PubSubChannel implements publish-subscribe pattern
type PubSubChannel[T any] struct {
	name        string
	subscribers []Channel[T]
	mu          sync.RWMutex
	closed      bool
}

// NewPubSubChannel creates a new publish-subscribe channel
func NewPubSubChannel[T any](name string) *PubSubChannel[T] {
	return &PubSubChannel[T]{
		name:        name,
		subscribers: make([]Channel[T], 0),
	}
}

// Subscribe adds a subscriber to the channel
func (p *PubSubChannel[T]) Subscribe(subscriber Channel[T]) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("channel is closed")
	}

	p.subscribers = append(p.subscribers, subscriber)
	return nil
}

// Unsubscribe removes a subscriber from the channel
func (p *PubSubChannel[T]) Unsubscribe(subscriber Channel[T]) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, sub := range p.subscribers {
		if sub == subscriber {
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			return nil
		}
	}

	return errors.New("subscriber not found")
}

// Send publishes a message to all subscribers
func (p *PubSubChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.New("channel is closed")
	}

	subscribers := make([]Channel[T], len(p.subscribers))
	copy(subscribers, p.subscribers)
	p.mu.RUnlock()

	msg.AddHistoryEntry(p.name, "pubsub_publish", map[string]interface{}{
		"subscriber_count": len(subscribers),
	})

	var wg sync.WaitGroup
	errChan := make(chan error, len(subscribers))

	for _, subscriber := range subscribers {
		wg.Add(1)
		go func(sub Channel[T]) {
			defer wg.Done()
			// Clone message for each subscriber
			if err := sub.Send(ctx, msg.Clone()); err != nil {
				errChan <- err
			}
		}(subscriber)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to publish to %d subscribers: %v", len(errs), errs)
	}

	return nil
}

// Receive is not supported for pub-sub (publish-only)
func (p *PubSubChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	return nil, errors.New("receive not supported on pub-sub channel")
}

// TryReceive is not supported for pub-sub
func (p *PubSubChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	return nil, false, errors.New("try receive not supported on pub-sub channel")
}

// Close closes the channel
func (p *PubSubChannel[T]) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("channel already closed")
	}

	p.closed = true
	p.subscribers = nil
	return nil
}

// SubscriberCount returns the number of subscribers
func (p *PubSubChannel[T]) SubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subscribers)
}

// DeadLetterChannelImpl implements a dead letter channel for failed messages
type DeadLetterChannelImpl[T any] struct {
	name           string
	mainChannel    Channel[T]
	dlqChannel     Channel[T]
	stats          DeadLetterStats
	errorReasons   map[string]int
	mu             sync.RWMutex
}

// NewDeadLetterChannel creates a new dead letter channel
func NewDeadLetterChannel[T any](name string, mainChannel Channel[T], dlqCapacity int) *DeadLetterChannelImpl[T] {
	return &DeadLetterChannelImpl[T]{
		name:         name,
		mainChannel:  mainChannel,
		dlqChannel:   NewInMemoryChannel[T](name+"-dlq", dlqCapacity),
		errorReasons: make(map[string]int),
		stats: DeadLetterStats{
			ErrorCounts: make(map[string]int),
		},
	}
}

// Send sends a message to the main channel
func (d *DeadLetterChannelImpl[T]) Send(ctx context.Context, msg *Message[T]) error {
	return d.mainChannel.Send(ctx, msg)
}

// Receive receives from the main channel
func (d *DeadLetterChannelImpl[T]) Receive(ctx context.Context) (*Message[T], error) {
	return d.mainChannel.Receive(ctx)
}

// TryReceive tries to receive from the main channel
func (d *DeadLetterChannelImpl[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	return d.mainChannel.TryReceive(ctx)
}

// Close closes both main and DLQ channels
func (d *DeadLetterChannelImpl[T]) Close() error {
	if err := d.mainChannel.Close(); err != nil {
		return err
	}
	return d.dlqChannel.Close()
}

// SendToDLQ sends a message to the dead letter queue
func (d *DeadLetterChannelImpl[T]) SendToDLQ(ctx context.Context, msg *Message[T], reason error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	msg.AddHistoryEntry(d.name, "sent_to_dlq", map[string]interface{}{
		"reason": reason.Error(),
	})

	reasonStr := reason.Error()
	d.errorReasons[reasonStr]++
	d.stats.ErrorCounts[reasonStr]++
	d.stats.TotalMessages++

	now := time.Now()
	if d.stats.OldestMessage == nil {
		d.stats.OldestMessage = &now
	}
	d.stats.NewestMessage = &now

	return d.dlqChannel.Send(ctx, msg)
}

// GetDLQStats returns dead letter queue statistics
func (d *DeadLetterChannelImpl[T]) GetDLQStats() DeadLetterStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	statsCopy := DeadLetterStats{
		TotalMessages: d.stats.TotalMessages,
		OldestMessage: d.stats.OldestMessage,
		NewestMessage: d.stats.NewestMessage,
		ErrorCounts:   make(map[string]int),
	}

	for k, v := range d.stats.ErrorCounts {
		statsCopy.ErrorCounts[k] = v
	}

	return statsCopy
}

// GetDLQChannel returns the DLQ channel for inspection
func (d *DeadLetterChannelImpl[T]) GetDLQChannel() Channel[T] {
	return d.dlqChannel
}

// WireTapImpl implements non-intrusive message monitoring
type WireTapImpl[T any] struct {
	name       string
	tapChannel Channel[T]
	mu         sync.RWMutex
}

// NewWireTap creates a new wire tap
func NewWireTap[T any](name string, tapChannel Channel[T]) *WireTapImpl[T] {
	return &WireTapImpl[T]{
		name:       name,
		tapChannel: tapChannel,
	}
}

// Tap captures a copy of the message
func (w *WireTapImpl[T]) Tap(ctx context.Context, msg *Message[T]) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.tapChannel == nil {
		return errors.New("tap channel not configured")
	}

	// Clone message to avoid interference
	tappedMsg := msg.Clone()
	tappedMsg.AddHistoryEntry(w.name, "wire_tap", nil)

	// Non-blocking send - we don't want to affect the main flow
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = w.tapChannel.Send(ctx, tappedMsg)
	}()

	return nil
}

// SetTapChannel sets the tap channel
func (w *WireTapImpl[T]) SetTapChannel(channel Channel[T]) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tapChannel = channel
}

// WireTapChannel wraps a channel with wire tap functionality
type WireTapChannel[T any] struct {
	name           string
	wrappedChannel Channel[T]
	wireTap        WireTap[T]
}

// NewWireTapChannel creates a new wire tap channel
func NewWireTapChannel[T any](name string, wrapped Channel[T], wireTap WireTap[T]) *WireTapChannel[T] {
	return &WireTapChannel[T]{
		name:           name,
		wrappedChannel: wrapped,
		wireTap:        wireTap,
	}
}

// Send sends a message and taps it
func (w *WireTapChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	// Tap the message first
	_ = w.wireTap.Tap(ctx, msg)

	// Then send to wrapped channel
	return w.wrappedChannel.Send(ctx, msg)
}

// Receive receives from the wrapped channel
func (w *WireTapChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	msg, err := w.wrappedChannel.Receive(ctx)
	if err == nil && msg != nil {
		_ = w.wireTap.Tap(ctx, msg)
	}
	return msg, err
}

// TryReceive tries to receive from the wrapped channel
func (w *WireTapChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	msg, ok, err := w.wrappedChannel.TryReceive(ctx)
	if err == nil && ok && msg != nil {
		_ = w.wireTap.Tap(ctx, msg)
	}
	return msg, ok, err
}

// Close closes the wrapped channel
func (w *WireTapChannel[T]) Close() error {
	return w.wrappedChannel.Close()
}

// GuaranteedDeliveryChannel ensures message persistence
type GuaranteedDeliveryChannel[T any] struct {
	name    string
	channel Channel[T]
	store   MessageStore[T]
	mu      sync.RWMutex
}

// NewGuaranteedDeliveryChannel creates a channel with guaranteed delivery
func NewGuaranteedDeliveryChannel[T any](
	name string,
	channel Channel[T],
	store MessageStore[T],
) *GuaranteedDeliveryChannel[T] {
	return &GuaranteedDeliveryChannel[T]{
		name:    name,
		channel: channel,
		store:   store,
	}
}

// Send persists and sends a message
func (g *GuaranteedDeliveryChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	// First, persist the message
	if err := g.store.Store(ctx, msg); err != nil {
		return fmt.Errorf("failed to persist message: %w", err)
	}

	msg.AddHistoryEntry(g.name, "guaranteed_delivery_stored", nil)

	// Then send
	if err := g.channel.Send(ctx, msg); err != nil {
		// Message is persisted, can be retried later
		return fmt.Errorf("failed to send message (persisted): %w", err)
	}

	// Successfully sent, can delete from store
	_ = g.store.Delete(ctx, msg.ID)

	msg.AddHistoryEntry(g.name, "guaranteed_delivery_sent", nil)
	return nil
}

// Receive receives from the channel
func (g *GuaranteedDeliveryChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	return g.channel.Receive(ctx)
}

// TryReceive tries to receive from the channel
func (g *GuaranteedDeliveryChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	return g.channel.TryReceive(ctx)
}

// Close closes the channel
func (g *GuaranteedDeliveryChannel[T]) Close() error {
	return g.channel.Close()
}

// InMemoryMessageStore implements in-memory message persistence
type InMemoryMessageStore[T any] struct {
	name     string
	messages map[string]*Message[T]
	mu       sync.RWMutex
}

// NewInMemoryMessageStore creates a new in-memory message store
func NewInMemoryMessageStore[T any](name string) *InMemoryMessageStore[T] {
	return &InMemoryMessageStore[T]{
		name:     name,
		messages: make(map[string]*Message[T]),
	}
}

// Store persists a message
func (s *InMemoryMessageStore[T]) Store(ctx context.Context, msg *Message[T]) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[msg.ID] = msg.Clone()
	return nil
}

// Retrieve retrieves a message by ID
func (s *InMemoryMessageStore[T]) Retrieve(ctx context.Context, messageID string) (*Message[T], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg, exists := s.messages[messageID]
	if !exists {
		return nil, fmt.Errorf("message not found: %s", messageID)
	}

	return msg.Clone(), nil
}

// Delete removes a message from the store
func (s *InMemoryMessageStore[T]) Delete(ctx context.Context, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.messages[messageID]; !exists {
		return fmt.Errorf("message not found: %s", messageID)
	}

	delete(s.messages, messageID)
	return nil
}

// List lists all stored messages matching a filter
func (s *InMemoryMessageStore[T]) List(ctx context.Context, filter MessagePredicate[T]) ([]*Message[T], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Message[T]
	for _, msg := range s.messages {
		if filter == nil || filter(msg) {
			result = append(result, msg.Clone())
		}
	}

	return result, nil
}

// Size returns the number of stored messages
func (s *InMemoryMessageStore[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// InvalidMessageChannel handles invalid/malformed messages
type InvalidMessageChannel[T any] struct {
	name            string
	validationFunc  func(*Message[T]) error
	validChannel    Channel[T]
	invalidChannel  Channel[T]
}

// NewInvalidMessageChannel creates a new invalid message channel
func NewInvalidMessageChannel[T any](
	name string,
	validationFunc func(*Message[T]) error,
	validChannel Channel[T],
	invalidChannel Channel[T],
) *InvalidMessageChannel[T] {
	return &InvalidMessageChannel[T]{
		name:           name,
		validationFunc: validationFunc,
		validChannel:   validChannel,
		invalidChannel: invalidChannel,
	}
}

// Send validates and routes to appropriate channel
func (i *InvalidMessageChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	err := i.validationFunc(msg)

	if err == nil {
		msg.AddHistoryEntry(i.name, "validation_passed", nil)
		return i.validChannel.Send(ctx, msg)
	}

	msg.AddHistoryEntry(i.name, "validation_failed", map[string]interface{}{
		"error": err.Error(),
	})
	return i.invalidChannel.Send(ctx, msg)
}

// Receive receives from the valid channel
func (i *InvalidMessageChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	return i.validChannel.Receive(ctx)
}

// TryReceive tries to receive from the valid channel
func (i *InvalidMessageChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	return i.validChannel.TryReceive(ctx)
}

// Close closes both channels
func (i *InvalidMessageChannel[T]) Close() error {
	if err := i.validChannel.Close(); err != nil {
		return err
	}
	return i.invalidChannel.Close()
}

// PriorityChannel implements a priority-based message channel
type PriorityChannel[T any] struct {
	name      string
	queues    map[int]chan *Message[T]
	priorities []int
	mu        sync.RWMutex
	closed    bool
}

// NewPriorityChannel creates a new priority channel
func NewPriorityChannel[T any](name string, priorities []int, capacity int) *PriorityChannel[T] {
	queues := make(map[int]chan *Message[T])
	for _, priority := range priorities {
		queues[priority] = make(chan *Message[T], capacity)
	}

	return &PriorityChannel[T]{
		name:       name,
		queues:     queues,
		priorities: priorities,
	}
}

// Send sends a message with priority
func (p *PriorityChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return errors.New("channel is closed")
	}

	// Get priority from message properties
	priorityVal, _ := msg.GetProperty("priority")
	priority, ok := priorityVal.(int)
	if !ok {
		priority = 0 // Default priority
	}

	queue, exists := p.queues[priority]
	if !exists {
		return fmt.Errorf("invalid priority: %d", priority)
	}

	select {
	case queue <- msg:
		msg.AddHistoryEntry(p.name, "priority_enqueue", map[string]interface{}{
			"priority": priority,
		})
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Receive receives the highest priority message available
func (p *PriorityChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Try to receive from queues in priority order
	for _, priority := range p.priorities {
		queue := p.queues[priority]
		select {
		case msg := <-queue:
			msg.AddHistoryEntry(p.name, "priority_dequeue", map[string]interface{}{
				"priority": priority,
			})
			return msg, nil
		default:
			continue
		}
	}

	// No messages available, wait for any
	cases := make([]chan *Message[T], 0, len(p.priorities))
	for _, priority := range p.priorities {
		cases = append(cases, p.queues[priority])
	}

	// Simple implementation - in production use reflect.Select for true prioritization
	select {
	case msg := <-cases[0]:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TryReceive tries to receive the highest priority message
func (p *PriorityChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, priority := range p.priorities {
		queue := p.queues[priority]
		select {
		case msg := <-queue:
			msg.AddHistoryEntry(p.name, "priority_try_dequeue", map[string]interface{}{
				"priority": priority,
			})
			return msg, true, nil
		default:
			continue
		}
	}

	return nil, false, nil
}

// Close closes all priority queues
func (p *PriorityChannel[T]) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("channel already closed")
	}

	p.closed = true
	for _, queue := range p.queues {
		close(queue)
	}

	return nil
}
