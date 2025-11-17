package espresso

import (
	"context"
	"sync"
	"time"
)

// Message represents a generic message in the EIP framework.
// It abstracts away transport-specific details (HTTP, AMQP, etc.)
// and provides a unified interface for all messaging patterns.
type Message[T any] struct {
	// ID is a unique identifier for this message
	ID string

	// CorrelationID links related messages together (e.g., request/response pairs)
	CorrelationID string

	// Headers contains message metadata
	Headers map[string]string

	// Body is the message payload
	Body T

	// Timestamp when the message was created
	Timestamp time.Time

	// ExpiresAt defines when this message should be considered expired (optional)
	ExpiresAt *time.Time

	// ReplyTo specifies where responses should be sent (Request-Reply pattern)
	ReplyTo string

	// History tracks the path this message has taken through the system
	History []MessageHistoryEntry

	// Properties for additional metadata
	Properties map[string]interface{}

	// RetryCount tracks how many times this message has been retried
	RetryCount int

	// mu protects concurrent access to message fields
	mu sync.RWMutex
}

// MessageHistoryEntry records a step in the message's journey
type MessageHistoryEntry struct {
	Component string
	Timestamp time.Time
	Action    string
	Details   map[string]interface{}
}

// NewMessage creates a new message with the given body
func NewMessage[T any](body T) *Message[T] {
	return &Message[T]{
		ID:         generateMessageID(),
		Body:       body,
		Headers:    make(map[string]string),
		Properties: make(map[string]interface{}),
		Timestamp:  time.Now(),
		History:    make([]MessageHistoryEntry, 0),
	}
}

// Clone creates a deep copy of the message
func (m *Message[T]) Clone() *Message[T] {
	m.mu.RLock()
	defer m.mu.RUnlock()

	headers := make(map[string]string, len(m.Headers))
	for k, v := range m.Headers {
		headers[k] = v
	}

	properties := make(map[string]interface{}, len(m.Properties))
	for k, v := range m.Properties {
		properties[k] = v
	}

	history := make([]MessageHistoryEntry, len(m.History))
	copy(history, m.History)

	return &Message[T]{
		ID:            m.ID,
		CorrelationID: m.CorrelationID,
		Headers:       headers,
		Body:          m.Body,
		Timestamp:     m.Timestamp,
		ExpiresAt:     m.ExpiresAt,
		ReplyTo:       m.ReplyTo,
		History:       history,
		Properties:    properties,
		RetryCount:    m.RetryCount,
	}
}

// AddHistoryEntry adds an entry to the message history
func (m *Message[T]) AddHistoryEntry(component, action string, details map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.History = append(m.History, MessageHistoryEntry{
		Component: component,
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
	})
}

// IsExpired checks if the message has expired
func (m *Message[T]) IsExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*m.ExpiresAt)
}

// SetHeader sets a header value (thread-safe)
func (m *Message[T]) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Headers[key] = value
}

// GetHeader gets a header value (thread-safe)
func (m *Message[T]) GetHeader(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.Headers[key]
	return val, ok
}

// SetProperty sets a property value (thread-safe)
func (m *Message[T]) SetProperty(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Properties[key] = value
}

// GetProperty gets a property value (thread-safe)
func (m *Message[T]) GetProperty(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.Properties[key]
	return val, ok
}

// Channel represents a communication channel for messages.
// This is the foundation for all messaging patterns.
type Channel[T any] interface {
	// Send sends a message to the channel
	Send(ctx context.Context, msg *Message[T]) error

	// Receive receives a message from the channel (blocking)
	Receive(ctx context.Context) (*Message[T], error)

	// TryReceive attempts to receive a message without blocking
	TryReceive(ctx context.Context) (*Message[T], bool, error)

	// Close closes the channel
	Close() error
}

// MessageHandler processes messages
type MessageHandler[T any, R any] interface {
	// Handle processes a message and returns a result
	Handle(ctx context.Context, msg *Message[T]) (*Message[R], error)
}

// MessageHandlerFunc is a function adapter for MessageHandler
type MessageHandlerFunc[T any, R any] func(ctx context.Context, msg *Message[T]) (*Message[R], error)

func (f MessageHandlerFunc[T, R]) Handle(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	return f(ctx, msg)
}

// MessagePredicate evaluates a condition on a message
type MessagePredicate[T any] func(*Message[T]) bool

// MessageTransformer transforms a message from one type to another
type MessageTransformer[T any, R any] interface {
	// Transform converts a message of type T to type R
	Transform(ctx context.Context, msg *Message[T]) (*Message[R], error)
}

// MessageTransformerFunc is a function adapter for MessageTransformer
type MessageTransformerFunc[T any, R any] func(ctx context.Context, msg *Message[T]) (*Message[R], error)

func (f MessageTransformerFunc[T, R]) Transform(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	return f(ctx, msg)
}

// Router routes messages to different channels based on conditions
type Router[T any] interface {
	// Route determines which channel(s) should receive the message
	Route(ctx context.Context, msg *Message[T]) ([]Channel[T], error)

	// AddRoute adds a routing rule
	AddRoute(name string, predicate MessagePredicate[T], channel Channel[T]) Router[T]
}

// Pipeline represents a series of message transformations
type Pipeline[T any] interface {
	// Process processes a message through the pipeline
	Process(ctx context.Context, msg *Message[T]) (*Message[T], error)

	// AddStage adds a transformation stage to the pipeline
	AddStage(name string, transformer MessageTransformer[T, T]) Pipeline[T]
}

// MessageAggregator combines multiple messages into one
type MessageAggregator[T any, R any] interface {
	// Aggregate combines messages into a single result
	Aggregate(ctx context.Context, messages []*Message[T]) (*Message[R], error)
}

// MessageSplitter splits one message into multiple messages
type MessageSplitter[T any, R any] interface {
	// Split breaks a message into multiple messages
	Split(ctx context.Context, msg *Message[T]) ([]*Message[R], error)
}

// MessageFilter filters messages based on a predicate
type MessageFilter[T any] interface {
	// Filter returns true if the message should pass through
	Filter(ctx context.Context, msg *Message[T]) (bool, error)
}

// MessageFilterFunc is a function adapter for MessageFilter
type MessageFilterFunc[T any] func(ctx context.Context, msg *Message[T]) (bool, error)

func (f MessageFilterFunc[T]) Filter(ctx context.Context, msg *Message[T]) (bool, error) {
	return f(ctx, msg)
}

// ContentEnricher enriches messages with additional data
type ContentEnricher[T any] interface {
	// Enrich adds additional data to the message
	Enrich(ctx context.Context, msg *Message[T]) (*Message[T], error)
}

// MessageEndpoint represents a messaging endpoint
type MessageEndpoint[T any] interface {
	// Start starts the endpoint
	Start(ctx context.Context) error

	// Stop stops the endpoint
	Stop(ctx context.Context) error

	// IsRunning returns true if the endpoint is running
	IsRunning() bool
}

// ChannelAdapter connects the EIP framework to external systems
type ChannelAdapter[T any, R any] interface {
	// SendToExternal sends a message to an external system
	SendToExternal(ctx context.Context, msg *Message[T]) error

	// ReceiveFromExternal receives a message from an external system
	ReceiveFromExternal(ctx context.Context) (*Message[R], error)
}

// DeadLetterChannel handles messages that cannot be processed
type DeadLetterChannel[T any] interface {
	Channel[T]

	// SendToDLQ sends a message to the dead letter queue with an error reason
	SendToDLQ(ctx context.Context, msg *Message[T], reason error) error

	// GetDLQStats returns statistics about the dead letter queue
	GetDLQStats() DeadLetterStats
}

// DeadLetterStats contains statistics about dead letter messages
type DeadLetterStats struct {
	TotalMessages int
	OldestMessage *time.Time
	NewestMessage *time.Time
	ErrorCounts   map[string]int
}

// IdempotentReceiver ensures messages are processed only once
type IdempotentReceiver[T any] interface {
	// Receive receives and processes a message idempotently
	Receive(ctx context.Context, msg *Message[T], handler MessageHandler[T, T]) error

	// IsProcessed checks if a message has already been processed
	IsProcessed(messageID string) bool
}

// CompetingConsumers manages multiple consumers competing for messages
type CompetingConsumers[T any] interface {
	// AddConsumer adds a consumer to the pool
	AddConsumer(consumer MessageHandler[T, T]) error

	// RemoveConsumer removes a consumer from the pool
	RemoveConsumer(consumerID string) error

	// Start starts all consumers
	Start(ctx context.Context, channel Channel[T]) error

	// Stop stops all consumers
	Stop() error

	// GetStats returns statistics about the consumer pool
	GetStats() ConsumerPoolStats
}

// ConsumerPoolStats contains statistics about competing consumers
type ConsumerPoolStats struct {
	ActiveConsumers int
	TotalProcessed  int64
	TotalErrors     int64
	AvgProcessTime  time.Duration
}

// WireTap allows non-intrusive monitoring of messages
type WireTap[T any] interface {
	// Tap captures a copy of the message without affecting the flow
	Tap(ctx context.Context, msg *Message[T]) error

	// SetTapChannel sets the channel where tapped messages are sent
	SetTapChannel(channel Channel[T])
}

// ControlBus provides administrative control over the messaging system
type ControlBus interface {
	// SendCommand sends a control command
	SendCommand(ctx context.Context, command ControlCommand) error

	// Subscribe subscribes to control events
	Subscribe(handler ControlCommandHandler) error
}

// ControlCommand represents an administrative command
type ControlCommand struct {
	Type       string
	Target     string
	Parameters map[string]interface{}
	Timestamp  time.Time
}

// ControlCommandHandler handles control commands
type ControlCommandHandler func(ctx context.Context, cmd ControlCommand) error

// MessageStore persists messages for guaranteed delivery
type MessageStore[T any] interface {
	// Store persists a message
	Store(ctx context.Context, msg *Message[T]) error

	// Retrieve retrieves a message by ID
	Retrieve(ctx context.Context, messageID string) (*Message[T], error)

	// Delete removes a message from the store
	Delete(ctx context.Context, messageID string) error

	// List lists all stored messages
	List(ctx context.Context, filter MessagePredicate[T]) ([]*Message[T], error)
}

// ClaimCheck stores and retrieves large message payloads
type ClaimCheck[T any] interface {
	// Store stores a payload and returns a claim check (reference)
	Store(ctx context.Context, payload T) (string, error)

	// Retrieve retrieves a payload using a claim check
	Retrieve(ctx context.Context, claimCheckID string) (T, error)

	// Remove removes a stored payload
	Remove(ctx context.Context, claimCheckID string) error
}

// RoutingSlip defines a dynamic message routing sequence
type RoutingSlip struct {
	Steps    []RoutingStep
	Current  int
	Metadata map[string]interface{}
}

// RoutingStep represents a step in the routing slip
type RoutingStep struct {
	Component string
	Action    string
	Config    map[string]interface{}
}

// ProcessManager manages complex routing with state
type ProcessManager[T any] interface {
	// StartProcess starts a new process instance
	StartProcess(ctx context.Context, processID string, initialMessage *Message[T]) error

	// HandleMessage processes a message within a process instance
	HandleMessage(ctx context.Context, processID string, msg *Message[T]) error

	// GetProcessState returns the current state of a process
	GetProcessState(processID string) (ProcessState, error)

	// CompleteProcess marks a process as completed
	CompleteProcess(ctx context.Context, processID string) error
}

// ProcessState represents the state of a process instance
type ProcessState struct {
	ProcessID       string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CurrentStep     string
	CompletedSteps  []string
	PendingMessages int
	Metadata        map[string]interface{}
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	// Simple implementation - in production, use UUID or similar
	return time.Now().Format("20060102150405.000000")
}
