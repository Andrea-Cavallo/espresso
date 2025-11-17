package espresso

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// InMemoryChannel is an in-memory implementation of Channel
type InMemoryChannel[T any] struct {
	name     string
	buffer   chan *Message[T]
	closed   bool
	mu       sync.RWMutex
	capacity int
}

// NewInMemoryChannel creates a new in-memory channel with the specified capacity
func NewInMemoryChannel[T any](name string, capacity int) *InMemoryChannel[T] {
	return &InMemoryChannel[T]{
		name:     name,
		buffer:   make(chan *Message[T], capacity),
		capacity: capacity,
	}
}

// Send sends a message to the channel
func (c *InMemoryChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return errors.New("channel is closed")
	}

	msg.AddHistoryEntry(c.name, "send", map[string]interface{}{
		"channel_type": "in_memory",
	})

	select {
	case c.buffer <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Receive receives a message from the channel (blocking)
func (c *InMemoryChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	select {
	case msg, ok := <-c.buffer:
		if !ok {
			return nil, errors.New("channel is closed")
		}
		msg.AddHistoryEntry(c.name, "receive", map[string]interface{}{
			"channel_type": "in_memory",
		})
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TryReceive attempts to receive a message without blocking
func (c *InMemoryChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	select {
	case msg, ok := <-c.buffer:
		if !ok {
			return nil, false, errors.New("channel is closed")
		}
		msg.AddHistoryEntry(c.name, "try_receive", map[string]interface{}{
			"channel_type": "in_memory",
		})
		return msg, true, nil
	default:
		return nil, false, nil
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

// Close closes the channel
func (c *InMemoryChannel[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("channel already closed")
	}

	c.closed = true
	close(c.buffer)
	return nil
}

// Name returns the channel name
func (c *InMemoryChannel[T]) Name() string {
	return c.name
}

// Len returns the number of messages currently in the channel
func (c *InMemoryChannel[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.buffer)
}

// Cap returns the capacity of the channel
func (c *InMemoryChannel[T]) Cap() int {
	return c.capacity
}

// ContentBasedRouter routes messages based on their content
type ContentBasedRouter[T any] struct {
	name          string
	routes        []routingRule[T]
	defaultChannel Channel[T]
	mu            sync.RWMutex
}

type routingRule[T any] struct {
	name      string
	predicate MessagePredicate[T]
	channel   Channel[T]
}

// NewContentBasedRouter creates a new content-based router
func NewContentBasedRouter[T any](name string) *ContentBasedRouter[T] {
	return &ContentBasedRouter[T]{
		name:   name,
		routes: make([]routingRule[T], 0),
	}
}

// AddRoute adds a routing rule
func (r *ContentBasedRouter[T]) AddRoute(name string, predicate MessagePredicate[T], channel Channel[T]) Router[T] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = append(r.routes, routingRule[T]{
		name:      name,
		predicate: predicate,
		channel:   channel,
	})

	return r
}

// SetDefaultChannel sets a default channel for messages that don't match any route
func (r *ContentBasedRouter[T]) SetDefaultChannel(channel Channel[T]) *ContentBasedRouter[T] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultChannel = channel
	return r
}

// Route determines which channel(s) should receive the message
func (r *ContentBasedRouter[T]) Route(ctx context.Context, msg *Message[T]) ([]Channel[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msg.AddHistoryEntry(r.name, "route_start", map[string]interface{}{
		"router_type": "content_based",
		"routes":      len(r.routes),
	})

	var matchedChannels []Channel[T]

	for _, rule := range r.routes {
		if rule.predicate(msg) {
			matchedChannels = append(matchedChannels, rule.channel)
			msg.AddHistoryEntry(r.name, "route_matched", map[string]interface{}{
				"rule": rule.name,
			})
		}
	}

	// If no routes matched, use default channel
	if len(matchedChannels) == 0 && r.defaultChannel != nil {
		matchedChannels = append(matchedChannels, r.defaultChannel)
		msg.AddHistoryEntry(r.name, "route_default", nil)
	}

	if len(matchedChannels) == 0 {
		return nil, fmt.Errorf("no matching routes found for message %s", msg.ID)
	}

	return matchedChannels, nil
}

// RouteAndSend routes a message and sends it to all matching channels
func (r *ContentBasedRouter[T]) RouteAndSend(ctx context.Context, msg *Message[T]) error {
	channels, err := r.Route(ctx, msg)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(channels))

	for _, ch := range channels {
		wg.Add(1)
		go func(channel Channel[T]) {
			defer wg.Done()
			if err := channel.Send(ctx, msg.Clone()); err != nil {
				errChan <- err
			}
		}(ch)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("routing errors: %v", errs)
	}

	return nil
}

// RecipientListRouter sends messages to a static list of recipients
type RecipientListRouter[T any] struct {
	name       string
	recipients []Channel[T]
	mu         sync.RWMutex
}

// NewRecipientListRouter creates a new recipient list router
func NewRecipientListRouter[T any](name string, recipients ...Channel[T]) *RecipientListRouter[T] {
	return &RecipientListRouter[T]{
		name:       name,
		recipients: recipients,
	}
}

// AddRecipient adds a recipient to the list
func (r *RecipientListRouter[T]) AddRecipient(channel Channel[T]) *RecipientListRouter[T] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recipients = append(r.recipients, channel)
	return r
}

// RemoveRecipient removes a recipient from the list
func (r *RecipientListRouter[T]) RemoveRecipient(channel Channel[T]) *RecipientListRouter[T] {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, recipient := range r.recipients {
		// Simple pointer comparison - in production you'd want better identity
		if recipient == channel {
			r.recipients = append(r.recipients[:i], r.recipients[i+1:]...)
			break
		}
	}

	return r
}

// Route returns all recipients
func (r *RecipientListRouter[T]) Route(ctx context.Context, msg *Message[T]) ([]Channel[T], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	msg.AddHistoryEntry(r.name, "recipient_list_route", map[string]interface{}{
		"recipients": len(r.recipients),
	})

	return r.recipients, nil
}

// Send sends a message to all recipients
func (r *RecipientListRouter[T]) Send(ctx context.Context, msg *Message[T]) error {
	r.mu.RLock()
	recipients := make([]Channel[T], len(r.recipients))
	copy(recipients, r.recipients)
	r.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(recipients))

	for _, recipient := range recipients {
		wg.Add(1)
		go func(ch Channel[T]) {
			defer wg.Done()
			// Clone the message for each recipient to avoid race conditions
			if err := ch.Send(ctx, msg.Clone()); err != nil {
				errChan <- err
			}
		}(recipient)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("recipient list errors: %v", errs)
	}

	return nil
}

// DynamicRecipientListRouter determines recipients at runtime
type DynamicRecipientListRouter[T any] struct {
	name              string
	recipientSelector func(ctx context.Context, msg *Message[T]) ([]Channel[T], error)
}

// NewDynamicRecipientListRouter creates a new dynamic recipient list router
func NewDynamicRecipientListRouter[T any](
	name string,
	recipientSelector func(ctx context.Context, msg *Message[T]) ([]Channel[T], error),
) *DynamicRecipientListRouter[T] {
	return &DynamicRecipientListRouter[T]{
		name:              name,
		recipientSelector: recipientSelector,
	}
}

// Route determines recipients dynamically
func (r *DynamicRecipientListRouter[T]) Route(ctx context.Context, msg *Message[T]) ([]Channel[T], error) {
	msg.AddHistoryEntry(r.name, "dynamic_recipient_list_route", nil)
	return r.recipientSelector(ctx, msg)
}

// Send sends a message to dynamically determined recipients
func (r *DynamicRecipientListRouter[T]) Send(ctx context.Context, msg *Message[T]) error {
	recipients, err := r.Route(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to determine recipients: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(recipients))

	for _, recipient := range recipients {
		wg.Add(1)
		go func(ch Channel[T]) {
			defer wg.Done()
			if err := ch.Send(ctx, msg.Clone()); err != nil {
				errChan <- err
			}
		}(recipient)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("dynamic recipient list errors: %v", errs)
	}

	return nil
}

// MessageFilterChannel filters messages before passing them through
type MessageFilterChannel[T any] struct {
	name            string
	upstreamChannel Channel[T]
	filter          MessagePredicate[T]
	rejectedChannel Channel[T] // Optional channel for rejected messages
}

// NewMessageFilterChannel creates a new message filter channel
func NewMessageFilterChannel[T any](
	name string,
	upstream Channel[T],
	filter MessagePredicate[T],
) *MessageFilterChannel[T] {
	return &MessageFilterChannel[T]{
		name:            name,
		upstreamChannel: upstream,
		filter:          filter,
	}
}

// SetRejectedChannel sets a channel for rejected messages
func (c *MessageFilterChannel[T]) SetRejectedChannel(channel Channel[T]) *MessageFilterChannel[T] {
	c.rejectedChannel = channel
	return c
}

// Send sends a message through the filter
func (c *MessageFilterChannel[T]) Send(ctx context.Context, msg *Message[T]) error {
	if c.filter(msg) {
		msg.AddHistoryEntry(c.name, "filter_passed", nil)
		return c.upstreamChannel.Send(ctx, msg)
	}

	msg.AddHistoryEntry(c.name, "filter_rejected", nil)

	if c.rejectedChannel != nil {
		return c.rejectedChannel.Send(ctx, msg)
	}

	// Message is silently dropped if no rejected channel is configured
	return nil
}

// Receive receives from the upstream channel
func (c *MessageFilterChannel[T]) Receive(ctx context.Context) (*Message[T], error) {
	return c.upstreamChannel.Receive(ctx)
}

// TryReceive tries to receive from the upstream channel
func (c *MessageFilterChannel[T]) TryReceive(ctx context.Context) (*Message[T], bool, error) {
	return c.upstreamChannel.TryReceive(ctx)
}

// Close closes the upstream channel
func (c *MessageFilterChannel[T]) Close() error {
	return c.upstreamChannel.Close()
}

// Resequencer reorders out-of-sequence messages
type Resequencer[T any] struct {
	name           string
	outputChannel  Channel[T]
	buffer         map[int]*Message[T]
	nextSequence   int
	sequenceGetter func(*Message[T]) int
	mu             sync.Mutex
}

// NewResequencer creates a new resequencer
func NewResequencer[T any](
	name string,
	outputChannel Channel[T],
	sequenceGetter func(*Message[T]) int,
) *Resequencer[T] {
	return &Resequencer[T]{
		name:           name,
		outputChannel:  outputChannel,
		buffer:         make(map[int]*Message[T]),
		nextSequence:   0,
		sequenceGetter: sequenceGetter,
	}
}

// ProcessMessage processes a message and outputs in correct sequence
func (r *Resequencer[T]) ProcessMessage(ctx context.Context, msg *Message[T]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sequence := r.sequenceGetter(msg)
	msg.AddHistoryEntry(r.name, "resequencer_received", map[string]interface{}{
		"sequence":      sequence,
		"next_expected": r.nextSequence,
	})

	// Buffer the message
	r.buffer[sequence] = msg

	// Output all sequential messages starting from nextSequence
	for {
		msg, exists := r.buffer[r.nextSequence]
		if !exists {
			break
		}

		msg.AddHistoryEntry(r.name, "resequencer_output", map[string]interface{}{
			"sequence": r.nextSequence,
		})

		if err := r.outputChannel.Send(ctx, msg); err != nil {
			return fmt.Errorf("failed to send message %d: %w", r.nextSequence, err)
		}

		delete(r.buffer, r.nextSequence)
		r.nextSequence++
	}

	return nil
}

// GetBufferSize returns the number of buffered messages
func (r *Resequencer[T]) GetBufferSize() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.buffer)
}

// Flush sends all buffered messages regardless of sequence
func (r *Resequencer[T]) Flush(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for seq, msg := range r.buffer {
		msg.AddHistoryEntry(r.name, "resequencer_flush", map[string]interface{}{
			"sequence": seq,
		})
		if err := r.outputChannel.Send(ctx, msg); err != nil {
			return err
		}
		delete(r.buffer, seq)
	}

	return nil
}
