package espresso

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ControlBusImpl provides administrative control over the messaging system
type ControlBusImpl struct {
	name       string
	commands   chan ControlCommand
	handlers   []ControlCommandHandler
	running    bool
	mu         sync.RWMutex
	stopChan   chan struct{}
}

// NewControlBus creates a new control bus
func NewControlBus(name string) *ControlBusImpl {
	return &ControlBusImpl{
		name:     name,
		commands: make(chan ControlCommand, 100),
		handlers: make([]ControlCommandHandler, 0),
		stopChan: make(chan struct{}),
	}
}

// SendCommand sends a control command
func (c *ControlBusImpl) SendCommand(ctx context.Context, command ControlCommand) error {
	c.mu.RLock()
	if !c.running {
		c.mu.RUnlock()
		return errors.New("control bus not running")
	}
	c.mu.RUnlock()

	select {
	case c.commands <- command:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Subscribe subscribes to control events
func (c *ControlBusImpl) Subscribe(handler ControlCommandHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers = append(c.handlers, handler)
	return nil
}

// Start starts processing control commands
func (c *ControlBusImpl) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("control bus already running")
	}
	c.running = true
	c.mu.Unlock()

	go c.processCommands(ctx)

	return nil
}

// processCommands processes incoming commands
func (c *ControlBusImpl) processCommands(ctx context.Context) {
	for {
		select {
		case <-c.stopChan:
			return
		case <-ctx.Done():
			return
		case cmd := <-c.commands:
			c.mu.RLock()
			handlers := make([]ControlCommandHandler, len(c.handlers))
			copy(handlers, c.handlers)
			c.mu.RUnlock()

			for _, handler := range handlers {
				_ = handler(ctx, cmd)
			}
		}
	}
}

// Stop stops the control bus
func (c *ControlBusImpl) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return errors.New("control bus not running")
	}

	c.running = false
	close(c.stopChan)
	return nil
}

// ProcessManagerImpl manages complex routing with state
type ProcessManagerImpl[T any] struct {
	name      string
	processes map[string]*processInstance[T]
	mu        sync.RWMutex
}

type processInstance[T any] struct {
	ProcessID       string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CurrentStep     string
	CompletedSteps  []string
	PendingMessages []*Message[T]
	Metadata        map[string]interface{}
	mu              sync.Mutex
}

// NewProcessManager creates a new process manager
func NewProcessManager[T any](name string) *ProcessManagerImpl[T] {
	return &ProcessManagerImpl[T]{
		name:      name,
		processes: make(map[string]*processInstance[T]),
	}
}

// StartProcess starts a new process instance
func (p *ProcessManagerImpl[T]) StartProcess(ctx context.Context, processID string, initialMessage *Message[T]) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.processes[processID]; exists {
		return fmt.Errorf("process already exists: %s", processID)
	}

	now := time.Now()
	instance := &processInstance[T]{
		ProcessID:       processID,
		Status:          "running",
		CreatedAt:       now,
		UpdatedAt:       now,
		CurrentStep:     "initial",
		CompletedSteps:  make([]string, 0),
		PendingMessages: []*Message[T]{initialMessage},
		Metadata:        make(map[string]interface{}),
	}

	p.processes[processID] = instance

	initialMessage.AddHistoryEntry(p.name, "process_started", map[string]interface{}{
		"process_id": processID,
	})

	return nil
}

// HandleMessage processes a message within a process instance
func (p *ProcessManagerImpl[T]) HandleMessage(ctx context.Context, processID string, msg *Message[T]) error {
	p.mu.RLock()
	instance, exists := p.processes[processID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", processID)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	instance.PendingMessages = append(instance.PendingMessages, msg)
	instance.UpdatedAt = time.Now()

	msg.AddHistoryEntry(p.name, "process_message_added", map[string]interface{}{
		"process_id":       processID,
		"pending_messages": len(instance.PendingMessages),
	})

	return nil
}

// GetProcessState returns the current state of a process
func (p *ProcessManagerImpl[T]) GetProcessState(processID string) (ProcessState, error) {
	p.mu.RLock()
	instance, exists := p.processes[processID]
	p.mu.RUnlock()

	if !exists {
		return ProcessState{}, fmt.Errorf("process not found: %s", processID)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	return ProcessState{
		ProcessID:       instance.ProcessID,
		Status:          instance.Status,
		CreatedAt:       instance.CreatedAt,
		UpdatedAt:       instance.UpdatedAt,
		CurrentStep:     instance.CurrentStep,
		CompletedSteps:  append([]string{}, instance.CompletedSteps...),
		PendingMessages: len(instance.PendingMessages),
		Metadata:        instance.Metadata,
	}, nil
}

// CompleteProcess marks a process as completed
func (p *ProcessManagerImpl[T]) CompleteProcess(ctx context.Context, processID string) error {
	p.mu.RLock()
	instance, exists := p.processes[processID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", processID)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	instance.Status = "completed"
	instance.UpdatedAt = time.Now()

	return nil
}

// UpdateProcessStep updates the current step of a process
func (p *ProcessManagerImpl[T]) UpdateProcessStep(processID string, step string) error {
	p.mu.RLock()
	instance, exists := p.processes[processID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", processID)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	if instance.CurrentStep != "" {
		instance.CompletedSteps = append(instance.CompletedSteps, instance.CurrentStep)
	}
	instance.CurrentStep = step
	instance.UpdatedAt = time.Now()

	return nil
}

// GetProcessCount returns the number of active processes
func (p *ProcessManagerImpl[T]) GetProcessCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.processes)
}

// RoutingSlipProcessor processes messages using routing slips
type RoutingSlipProcessor[T any] struct {
	name       string
	components map[string]MessageHandler[T, T]
	mu         sync.RWMutex
}

// NewRoutingSlipProcessor creates a new routing slip processor
func NewRoutingSlipProcessor[T any](name string) *RoutingSlipProcessor[T] {
	return &RoutingSlipProcessor[T]{
		name:       name,
		components: make(map[string]MessageHandler[T, T]),
	}
}

// RegisterComponent registers a component that can be used in routing slips
func (r *RoutingSlipProcessor[T]) RegisterComponent(name string, handler MessageHandler[T, T]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.components[name] = handler
}

// Process processes a message using a routing slip
func (r *RoutingSlipProcessor[T]) Process(ctx context.Context, msg *Message[T], slip *RoutingSlip) (*Message[T], error) {
	msg.AddHistoryEntry(r.name, "routing_slip_start", map[string]interface{}{
		"total_steps": len(slip.Steps),
	})

	currentMsg := msg

	for slip.Current < len(slip.Steps) {
		step := slip.Steps[slip.Current]

		r.mu.RLock()
		handler, exists := r.components[step.Component]
		r.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("component not found: %s", step.Component)
		}

		currentMsg.AddHistoryEntry(r.name, "routing_slip_step", map[string]interface{}{
			"step":      slip.Current,
			"component": step.Component,
			"action":    step.Action,
		})

		result, err := handler.Handle(ctx, currentMsg)
		if err != nil {
			return nil, fmt.Errorf("routing slip step %d failed: %w", slip.Current, err)
		}

		currentMsg = result
		slip.Current++
	}

	currentMsg.AddHistoryEntry(r.name, "routing_slip_complete", map[string]interface{}{
		"steps_processed": len(slip.Steps),
	})

	return currentMsg, nil
}

// DetourRouter conditionally routes messages for monitoring
type DetourRouter[T any] struct {
	name              string
	normalChannel     Channel[T]
	monitoringChannel Channel[T]
	detourPredicate   MessagePredicate[T]
}

// NewDetourRouter creates a new detour router
func NewDetourRouter[T any](
	name string,
	normalChannel Channel[T],
	monitoringChannel Channel[T],
	detourPredicate MessagePredicate[T],
) *DetourRouter[T] {
	return &DetourRouter[T]{
		name:              name,
		normalChannel:     normalChannel,
		monitoringChannel: monitoringChannel,
		detourPredicate:   detourPredicate,
	}
}

// Route routes a message to normal or monitoring channel
func (d *DetourRouter[T]) Route(ctx context.Context, msg *Message[T]) error {
	if d.detourPredicate(msg) {
		msg.AddHistoryEntry(d.name, "detour_to_monitoring", nil)
		return d.monitoringChannel.Send(ctx, msg)
	}

	msg.AddHistoryEntry(d.name, "detour_normal_flow", nil)
	return d.normalChannel.Send(ctx, msg)
}

// MessageHistoryTracker tracks message history across the system
type MessageHistoryTracker[T any] struct {
	name    string
	history map[string][]*Message[T]
	mu      sync.RWMutex
}

// NewMessageHistoryTracker creates a new message history tracker
func NewMessageHistoryTracker[T any](name string) *MessageHistoryTracker[T] {
	return &MessageHistoryTracker[T]{
		name:    name,
		history: make(map[string][]*Message[T]),
	}
}

// Track tracks a message
func (m *MessageHistoryTracker[T]) Track(msg *Message[T]) {
	m.mu.Lock()
	defer m.mu.Unlock()

	correlationID := msg.CorrelationID
	if correlationID == "" {
		correlationID = msg.ID
	}

	m.history[correlationID] = append(m.history[correlationID], msg.Clone())
}

// GetHistory returns the history for a correlation ID
func (m *MessageHistoryTracker[T]) GetHistory(correlationID string) []*Message[T] {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.history[correlationID]
	if !exists {
		return nil
	}

	result := make([]*Message[T], len(history))
	for i, msg := range history {
		result[i] = msg.Clone()
	}

	return result
}

// GetMessagePath returns the path a message took through the system
func (m *MessageHistoryTracker[T]) GetMessagePath(correlationID string) []MessageHistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.history[correlationID]
	if !exists {
		return nil
	}

	var allEntries []MessageHistoryEntry
	for _, msg := range history {
		allEntries = append(allEntries, msg.History...)
	}

	return allEntries
}

// Clear clears the history for a correlation ID
func (m *MessageHistoryTracker[T]) Clear(correlationID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.history, correlationID)
}

// Size returns the number of tracked correlation IDs
func (m *MessageHistoryTracker[T]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.history)
}

// SmartProxy tracks message interactions
type SmartProxy[T any, R any] struct {
	name       string
	target     MessageHandler[T, R]
	tracker    *MessageHistoryTracker[T]
	metrics    *ProxyMetrics
	mu         sync.Mutex
}

// ProxyMetrics contains metrics about proxied calls
type ProxyMetrics struct {
	TotalCalls      int64
	SuccessfulCalls int64
	FailedCalls     int64
	TotalDuration   time.Duration
	AvgDuration     time.Duration
	LastCallTime    time.Time
}

// NewSmartProxy creates a new smart proxy
func NewSmartProxy[T any, R any](
	name string,
	target MessageHandler[T, R],
	tracker *MessageHistoryTracker[T],
) *SmartProxy[T, R] {
	return &SmartProxy[T, R]{
		name:    name,
		target:  target,
		tracker: tracker,
		metrics: &ProxyMetrics{},
	}
}

// Handle handles a message and tracks it
func (s *SmartProxy[T, R]) Handle(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	s.mu.Lock()
	start := time.Now()
	s.metrics.TotalCalls++
	s.mu.Unlock()

	// Track incoming message
	if s.tracker != nil {
		s.tracker.Track(msg)
	}

	msg.AddHistoryEntry(s.name, "smart_proxy_before", nil)

	// Call target
	result, err := s.target.Handle(ctx, msg)

	duration := time.Since(start)

	s.mu.Lock()
	s.metrics.LastCallTime = time.Now()
	s.metrics.TotalDuration += duration
	s.metrics.AvgDuration = time.Duration(int64(s.metrics.TotalDuration) / s.metrics.TotalCalls)

	if err != nil {
		s.metrics.FailedCalls++
	} else {
		s.metrics.SuccessfulCalls++
	}
	s.mu.Unlock()

	if result != nil {
		result.AddHistoryEntry(s.name, "smart_proxy_after", map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"success":     err == nil,
		})
	}

	return result, err
}

// GetMetrics returns proxy metrics
func (s *SmartProxy[T, R]) GetMetrics() ProxyMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	return ProxyMetrics{
		TotalCalls:      s.metrics.TotalCalls,
		SuccessfulCalls: s.metrics.SuccessfulCalls,
		FailedCalls:     s.metrics.FailedCalls,
		TotalDuration:   s.metrics.TotalDuration,
		AvgDuration:     s.metrics.AvgDuration,
		LastCallTime:    s.metrics.LastCallTime,
	}
}

// TestMessageGenerator generates test messages for testing
type TestMessageGenerator[T any] struct {
	name      string
	generator func(sequence int) *Message[T]
	sequence  int
	mu        sync.Mutex
}

// NewTestMessageGenerator creates a new test message generator
func NewTestMessageGenerator[T any](
	name string,
	generator func(sequence int) *Message[T],
) *TestMessageGenerator[T] {
	return &TestMessageGenerator[T]{
		name:      name,
		generator: generator,
	}
}

// Generate generates a test message
func (t *TestMessageGenerator[T]) Generate() *Message[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sequence++
	msg := t.generator(t.sequence)

	msg.AddHistoryEntry(t.name, "test_message_generated", map[string]interface{}{
		"sequence": t.sequence,
	})

	return msg
}

// GenerateBatch generates multiple test messages
func (t *TestMessageGenerator[T]) GenerateBatch(count int) []*Message[T] {
	messages := make([]*Message[T], count)
	for i := 0; i < count; i++ {
		messages[i] = t.Generate()
	}
	return messages
}

// ChannelPurger cleans up messages from channels
type ChannelPurger[T any] struct {
	name string
}

// NewChannelPurger creates a new channel purger
func NewChannelPurger[T any](name string) *ChannelPurger[T] {
	return &ChannelPurger[T]{
		name: name,
	}
}

// Purge purges all messages from a channel
func (c *ChannelPurger[T]) Purge(ctx context.Context, channel Channel[T]) (int, error) {
	count := 0

	for {
		msg, ok, err := channel.TryReceive(ctx)
		if err != nil {
			return count, err
		}

		if !ok {
			break // No more messages
		}

		if msg != nil {
			msg.AddHistoryEntry(c.name, "message_purged", nil)
		}
		count++
	}

	return count, nil
}

// PurgeWithPredicate purges messages matching a predicate
func (c *ChannelPurger[T]) PurgeWithPredicate(
	ctx context.Context,
	channel Channel[T],
	outputChannel Channel[T],
	predicate MessagePredicate[T],
) (int, error) {
	purged := 0
	kept := 0

	for {
		msg, ok, err := channel.TryReceive(ctx)
		if err != nil {
			return purged, err
		}

		if !ok {
			break
		}

		if predicate(msg) {
			msg.AddHistoryEntry(c.name, "message_purged_predicate", nil)
			purged++
		} else {
			// Put back messages that don't match
			if err := outputChannel.Send(ctx, msg); err != nil {
				return purged, err
			}
			kept++
		}
	}

	return purged, nil
}

// MessageBridge connects different messaging systems
type MessageBridge[T any, R any] struct {
	name              string
	sourceChannel     Channel[T]
	destinationChannel Channel[R]
	translator        MessageTransformer[T, R]
	running           bool
	stopChan          chan struct{}
	mu                sync.Mutex
}

// NewMessageBridge creates a new message bridge
func NewMessageBridge[T any, R any](
	name string,
	source Channel[T],
	destination Channel[R],
	translator MessageTransformer[T, R],
) *MessageBridge[T, R] {
	return &MessageBridge[T, R]{
		name:               name,
		sourceChannel:      source,
		destinationChannel: destination,
		translator:         translator,
		stopChan:           make(chan struct{}),
	}
}

// Start starts bridging messages
func (b *MessageBridge[T, R]) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return errors.New("bridge already running")
	}
	b.running = true
	b.mu.Unlock()

	go b.bridge(ctx)

	return nil
}

// bridge bridges messages between channels
func (b *MessageBridge[T, R]) bridge(ctx context.Context) {
	for {
		select {
		case <-b.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := b.sourceChannel.Receive(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}

			msg.AddHistoryEntry(b.name, "message_bridge_receive", nil)

			// Translate the message
			translated, err := b.translator.Transform(ctx, msg)
			if err != nil {
				msg.AddHistoryEntry(b.name, "message_bridge_translation_error", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			translated.AddHistoryEntry(b.name, "message_bridge_send", nil)

			// Send to destination
			if err := b.destinationChannel.Send(ctx, translated); err != nil {
				translated.AddHistoryEntry(b.name, "message_bridge_send_error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// Stop stops the bridge
func (b *MessageBridge[T, R]) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return errors.New("bridge not running")
	}

	b.running = false
	close(b.stopChan)
	return nil
}
