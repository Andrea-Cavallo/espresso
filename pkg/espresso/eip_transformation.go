package espresso

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Splitter splits a message into multiple messages
type Splitter[T any, R any] struct {
	name       string
	splitFunc  func(ctx context.Context, msg *Message[T]) ([]*Message[R], error)
	outputChan Channel[R]
}

// NewSplitter creates a new splitter
func NewSplitter[T any, R any](
	name string,
	splitFunc func(ctx context.Context, msg *Message[T]) ([]*Message[R], error),
	outputChan Channel[R],
) *Splitter[T, R] {
	return &Splitter[T, R]{
		name:       name,
		splitFunc:  splitFunc,
		outputChan: outputChan,
	}
}

// Split splits a message into multiple messages
func (s *Splitter[T, R]) Split(ctx context.Context, msg *Message[T]) ([]*Message[R], error) {
	msg.AddHistoryEntry(s.name, "split_start", nil)

	messages, err := s.splitFunc(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("split function failed: %w", err)
	}

	// Set correlation ID for all split messages
	for i, splitMsg := range messages {
		if splitMsg.CorrelationID == "" {
			splitMsg.CorrelationID = msg.ID
		}
		splitMsg.AddHistoryEntry(s.name, "split_created", map[string]interface{}{
			"original_message_id": msg.ID,
			"split_index":         i,
			"total_splits":        len(messages),
		})
	}

	msg.AddHistoryEntry(s.name, "split_complete", map[string]interface{}{
		"split_count": len(messages),
	})

	return messages, nil
}

// ProcessAndSend splits a message and sends all parts to the output channel
func (s *Splitter[T, R]) ProcessAndSend(ctx context.Context, msg *Message[T]) error {
	messages, err := s.Split(ctx, msg)
	if err != nil {
		return err
	}

	for _, splitMsg := range messages {
		if err := s.outputChan.Send(ctx, splitMsg); err != nil {
			return fmt.Errorf("failed to send split message: %w", err)
		}
	}

	return nil
}

// Aggregator aggregates multiple messages into one
type Aggregator[T any, R any] struct {
	name             string
	correlationIDKey string
	buffer           map[string][]*Message[T]
	completionFunc   func([]*Message[T]) bool
	aggregateFunc    func(ctx context.Context, messages []*Message[T]) (*Message[R], error)
	timeout          time.Duration
	outputChan       Channel[R]
	mu               sync.Mutex
}

// AggregatorConfig configures an aggregator
type AggregatorConfig[T any, R any] struct {
	Name           string
	CompletionFunc func([]*Message[T]) bool
	AggregateFunc  func(ctx context.Context, messages []*Message[T]) (*Message[R], error)
	Timeout        time.Duration
	OutputChannel  Channel[R]
}

// NewAggregator creates a new aggregator
func NewAggregator[T any, R any](config AggregatorConfig[T, R]) *Aggregator[T, R] {
	return &Aggregator[T, R]{
		name:           config.Name,
		buffer:         make(map[string][]*Message[T]),
		completionFunc: config.CompletionFunc,
		aggregateFunc:  config.AggregateFunc,
		timeout:        config.Timeout,
		outputChan:     config.OutputChannel,
	}
}

// Aggregate adds a message to the aggregation and processes if complete
func (a *Aggregator[T, R]) Aggregate(ctx context.Context, msg *Message[T]) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	correlationID := msg.CorrelationID
	if correlationID == "" {
		return errors.New("message must have correlation ID for aggregation")
	}

	msg.AddHistoryEntry(a.name, "aggregate_buffer", map[string]interface{}{
		"correlation_id": correlationID,
	})

	// Add to buffer
	a.buffer[correlationID] = append(a.buffer[correlationID], msg)

	// Check if aggregation is complete
	messages := a.buffer[correlationID]
	if a.completionFunc(messages) {
		return a.completeAggregation(ctx, correlationID, messages)
	}

	// Set timeout for aggregation if configured
	if a.timeout > 0 {
		go a.handleTimeout(ctx, correlationID, a.timeout)
	}

	return nil
}

// completeAggregation performs the aggregation and sends the result
func (a *Aggregator[T, R]) completeAggregation(ctx context.Context, correlationID string, messages []*Message[T]) error {
	// Remove from buffer
	delete(a.buffer, correlationID)

	// Perform aggregation
	result, err := a.aggregateFunc(ctx, messages)
	if err != nil {
		return fmt.Errorf("aggregation failed: %w", err)
	}

	result.CorrelationID = correlationID
	result.AddHistoryEntry(a.name, "aggregate_complete", map[string]interface{}{
		"correlation_id":  correlationID,
		"message_count":   len(messages),
		"result_message":  result.ID,
	})

	// Send aggregated result
	if a.outputChan != nil {
		return a.outputChan.Send(ctx, result)
	}

	return nil
}

// handleTimeout handles aggregation timeout
func (a *Aggregator[T, R]) handleTimeout(ctx context.Context, correlationID string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		a.mu.Lock()
		defer a.mu.Unlock()

		messages, exists := a.buffer[correlationID]
		if !exists {
			return // Already completed
		}

		// Force aggregation on timeout
		_ = a.completeAggregation(ctx, correlationID, messages)
	case <-ctx.Done():
		return
	}
}

// GetBufferSize returns the number of correlation groups being aggregated
func (a *Aggregator[T, R]) GetBufferSize() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.buffer)
}

// ContentEnricherImpl enriches messages with additional data
type ContentEnricherImpl[T any] struct {
	name       string
	enrichFunc func(ctx context.Context, msg *Message[T]) (*Message[T], error)
}

// NewContentEnricher creates a new content enricher
func NewContentEnricher[T any](
	name string,
	enrichFunc func(ctx context.Context, msg *Message[T]) (*Message[T], error),
) *ContentEnricherImpl[T] {
	return &ContentEnricherImpl[T]{
		name:       name,
		enrichFunc: enrichFunc,
	}
}

// Enrich enriches a message with additional data
func (e *ContentEnricherImpl[T]) Enrich(ctx context.Context, msg *Message[T]) (*Message[T], error) {
	msg.AddHistoryEntry(e.name, "enrich_start", nil)

	enriched, err := e.enrichFunc(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("enrichment failed: %w", err)
	}

	enriched.AddHistoryEntry(e.name, "enrich_complete", nil)
	return enriched, nil
}

// MessageTranslatorImpl transforms messages from one type to another
type MessageTranslatorImpl[T any, R any] struct {
	name          string
	translateFunc func(ctx context.Context, msg *Message[T]) (*Message[R], error)
}

// NewMessageTranslator creates a new message translator
func NewMessageTranslator[T any, R any](
	name string,
	translateFunc func(ctx context.Context, msg *Message[T]) (*Message[R], error),
) *MessageTranslatorImpl[T, R] {
	return &MessageTranslatorImpl[T, R]{
		name:          name,
		translateFunc: translateFunc,
	}
}

// Transform translates a message from type T to type R
func (t *MessageTranslatorImpl[T, R]) Transform(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	msg.AddHistoryEntry(t.name, "translate_start", nil)

	translated, err := t.translateFunc(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	// Preserve message metadata
	translated.ID = msg.ID
	translated.CorrelationID = msg.CorrelationID
	translated.Timestamp = msg.Timestamp
	translated.ExpiresAt = msg.ExpiresAt
	translated.ReplyTo = msg.ReplyTo

	// Copy headers
	for k, v := range msg.Headers {
		translated.SetHeader(k, v)
	}

	// Merge history
	translated.History = append(msg.History, translated.History...)
	translated.AddHistoryEntry(t.name, "translate_complete", nil)

	return translated, nil
}

// NormalizerImpl normalizes different message formats to a canonical format
type NormalizerImpl[R any] struct {
	name         string
	translators  map[string]func(ctx context.Context, msg *Message[any]) (*Message[R], error)
	formatDetector func(*Message[any]) string
}

// NewNormalizer creates a new normalizer
func NewNormalizer[R any](
	name string,
	formatDetector func(*Message[any]) string,
) *NormalizerImpl[R] {
	return &NormalizerImpl[R]{
		name:           name,
		translators:    make(map[string]func(ctx context.Context, msg *Message[any]) (*Message[R], error)),
		formatDetector: formatDetector,
	}
}

// RegisterFormat registers a translator for a specific format
func (n *NormalizerImpl[R]) RegisterFormat(
	format string,
	translator func(ctx context.Context, msg *Message[any]) (*Message[R], error),
) *NormalizerImpl[R] {
	n.translators[format] = translator
	return n
}

// Normalize normalizes a message to the canonical format
func (n *NormalizerImpl[R]) Normalize(ctx context.Context, msg *Message[any]) (*Message[R], error) {
	format := n.formatDetector(msg)
	msg.AddHistoryEntry(n.name, "normalize_start", map[string]interface{}{
		"detected_format": format,
	})

	translator, exists := n.translators[format]
	if !exists {
		return nil, fmt.Errorf("no translator registered for format: %s", format)
	}

	normalized, err := translator(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("normalization failed for format %s: %w", format, err)
	}

	normalized.AddHistoryEntry(n.name, "normalize_complete", map[string]interface{}{
		"from_format": format,
	})

	return normalized, nil
}

// ClaimCheckImpl implements the claim check pattern for large payloads
type ClaimCheckImpl[T any] struct {
	name    string
	storage map[string]T
	mu      sync.RWMutex
}

// NewClaimCheck creates a new claim check storage
func NewClaimCheck[T any](name string) *ClaimCheckImpl[T] {
	return &ClaimCheckImpl[T]{
		name:    name,
		storage: make(map[string]T),
	}
}

// Store stores a payload and returns a claim check ID
func (c *ClaimCheckImpl[T]) Store(ctx context.Context, payload T) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	claimCheckID := generateMessageID() // Reuse ID generator
	c.storage[claimCheckID] = payload

	return claimCheckID, nil
}

// Retrieve retrieves a payload using a claim check ID
func (c *ClaimCheckImpl[T]) Retrieve(ctx context.Context, claimCheckID string) (T, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	payload, exists := c.storage[claimCheckID]
	if !exists {
		var zero T
		return zero, fmt.Errorf("claim check not found: %s", claimCheckID)
	}

	return payload, nil
}

// Remove removes a stored payload
func (c *ClaimCheckImpl[T]) Remove(ctx context.Context, claimCheckID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.storage[claimCheckID]; !exists {
		return fmt.Errorf("claim check not found: %s", claimCheckID)
	}

	delete(c.storage, claimCheckID)
	return nil
}

// Size returns the number of stored payloads
func (c *ClaimCheckImpl[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.storage)
}

// PipelineImpl implements a message transformation pipeline
type PipelineImpl[T any] struct {
	name   string
	stages []pipelineStage[T]
	mu     sync.RWMutex
}

type pipelineStage[T any] struct {
	name        string
	transformer MessageTransformer[T, T]
}

// NewPipeline creates a new transformation pipeline
func NewPipeline[T any](name string) *PipelineImpl[T] {
	return &PipelineImpl[T]{
		name:   name,
		stages: make([]pipelineStage[T], 0),
	}
}

// AddStage adds a transformation stage to the pipeline
func (p *PipelineImpl[T]) AddStage(name string, transformer MessageTransformer[T, T]) Pipeline[T] {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stages = append(p.stages, pipelineStage[T]{
		name:        name,
		transformer: transformer,
	})

	return p
}

// Process processes a message through all pipeline stages
func (p *PipelineImpl[T]) Process(ctx context.Context, msg *Message[T]) (*Message[T], error) {
	p.mu.RLock()
	stages := make([]pipelineStage[T], len(p.stages))
	copy(stages, p.stages)
	p.mu.RUnlock()

	msg.AddHistoryEntry(p.name, "pipeline_start", map[string]interface{}{
		"stage_count": len(stages),
	})

	currentMsg := msg
	for i, stage := range stages {
		transformed, err := stage.transformer.Transform(ctx, currentMsg)
		if err != nil {
			return nil, fmt.Errorf("pipeline stage %d (%s) failed: %w", i, stage.name, err)
		}

		currentMsg = transformed
		currentMsg.AddHistoryEntry(p.name, "pipeline_stage_complete", map[string]interface{}{
			"stage":       stage.name,
			"stage_index": i,
		})
	}

	currentMsg.AddHistoryEntry(p.name, "pipeline_complete", map[string]interface{}{
		"stages_processed": len(stages),
	})

	return currentMsg, nil
}

// ComposedMessageProcessor combines Splitter, Router, and Aggregator
type ComposedMessageProcessor[T any, S any, R any] struct {
	name       string
	splitter   MessageSplitter[T, S]
	router     Router[S]
	aggregator MessageAggregator[S, R]
}

// NewComposedMessageProcessor creates a new composed message processor
func NewComposedMessageProcessor[T any, S any, R any](
	name string,
	splitter MessageSplitter[T, S],
	router Router[S],
	aggregator MessageAggregator[S, R],
) *ComposedMessageProcessor[T, S, R] {
	return &ComposedMessageProcessor[T, S, R]{
		name:       name,
		splitter:   splitter,
		router:     router,
		aggregator: aggregator,
	}
}

// Process processes a message through split, route, and aggregate
func (c *ComposedMessageProcessor[T, S, R]) Process(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	msg.AddHistoryEntry(c.name, "composed_start", nil)

	// Step 1: Split
	splitMessages, err := c.splitter.Split(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("split failed: %w", err)
	}

	// Step 2: Route each split message
	for _, splitMsg := range splitMessages {
		channels, err := c.router.Route(ctx, splitMsg)
		if err != nil {
			return nil, fmt.Errorf("route failed: %w", err)
		}

		// Send to all matched channels
		for _, ch := range channels {
			if err := ch.Send(ctx, splitMsg); err != nil {
				return nil, fmt.Errorf("send failed: %w", err)
			}
		}
	}

	// Step 3: Aggregate (caller needs to collect results and call aggregator)
	aggregated, err := c.aggregator.Aggregate(ctx, splitMessages)
	if err != nil {
		return nil, fmt.Errorf("aggregate failed: %w", err)
	}

	aggregated.AddHistoryEntry(c.name, "composed_complete", nil)
	return aggregated, nil
}
