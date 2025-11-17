package espresso

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPMessage wraps HTTP request and response data
type HTTPMessage struct {
	Request  *http.Request
	Response *http.Response
	Body     []byte
	Error    error
	Metadata map[string]interface{}
}

// HTTPChannelAdapter adapts HTTP operations to EIP channels
type HTTPChannelAdapter struct {
	name   string
	client *Client[any]
}

// NewHTTPChannelAdapter creates a new HTTP channel adapter
func NewHTTPChannelAdapter(name string, client *Client[any]) *HTTPChannelAdapter {
	return &HTTPChannelAdapter{
		name:   name,
		client: client,
	}
}

// SendToExternal sends a message via HTTP
func (h *HTTPChannelAdapter) SendToExternal(ctx context.Context, msg *Message[HTTPMessage]) error {
	msg.AddHistoryEntry(h.name, "http_send_start", map[string]interface{}{
		"url":    msg.Body.Request.URL.String(),
		"method": msg.Body.Request.Method,
	})

	// Execute HTTP request using the underlying client
	// Note: This is a simplified implementation
	// In production, you'd want to properly integrate with the Client's request builder

	msg.AddHistoryEntry(h.name, "http_send_complete", nil)
	return nil
}

// ReceiveFromExternal receives a response from HTTP
func (h *HTTPChannelAdapter) ReceiveFromExternal(ctx context.Context) (*Message[HTTPMessage], error) {
	// This would be implemented based on your specific HTTP polling needs
	return nil, fmt.Errorf("not implemented")
}

// HTTPToMessageTransformer transforms HTTP responses to EIP messages
type HTTPToMessageTransformer[T any] struct {
	name          string
	unmarshaller  func([]byte) (T, error)
}

// NewHTTPToMessageTransformer creates a transformer for HTTP to EIP messages
func NewHTTPToMessageTransformer[T any](
	name string,
	unmarshaller func([]byte) (T, error),
) *HTTPToMessageTransformer[T] {
	return &HTTPToMessageTransformer[T]{
		name:         name,
		unmarshaller: unmarshaller,
	}
}

// Transform transforms an HTTP message to a typed message
func (h *HTTPToMessageTransformer[T]) Transform(
	ctx context.Context,
	msg *Message[HTTPMessage],
) (*Message[T], error) {
	msg.AddHistoryEntry(h.name, "http_transform_start", nil)

	// Unmarshal the response body
	body, err := h.unmarshaller(msg.Body.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Create new typed message
	result := NewMessage(body)
	result.ID = msg.ID
	result.CorrelationID = msg.CorrelationID
	result.Timestamp = msg.Timestamp

	// Copy headers from HTTP response
	if msg.Body.Response != nil {
		for key, values := range msg.Body.Response.Header {
			if len(values) > 0 {
				result.SetHeader(key, values[0])
			}
		}
	}

	// Merge history
	result.History = append(msg.History, result.History...)
	result.AddHistoryEntry(h.name, "http_transform_complete", nil)

	return result, nil
}

// MessageToHTTPTransformer transforms EIP messages to HTTP requests
type MessageToHTTPTransformer[T any] struct {
	name       string
	marshaller func(T) ([]byte, error)
	baseURL    string
	method     string
}

// NewMessageToHTTPTransformer creates a transformer for EIP messages to HTTP
func NewMessageToHTTPTransformer[T any](
	name string,
	marshaller func(T) ([]byte, error),
	baseURL string,
	method string,
) *MessageToHTTPTransformer[T] {
	return &MessageToHTTPTransformer[T]{
		name:       name,
		marshaller: marshaller,
		baseURL:    baseURL,
		method:     method,
	}
}

// Transform transforms a typed message to an HTTP message
func (m *MessageToHTTPTransformer[T]) Transform(
	ctx context.Context,
	msg *Message[T],
) (*Message[HTTPMessage], error) {
	msg.AddHistoryEntry(m.name, "message_to_http_start", nil)

	// Marshal the body
	body, err := m.marshaller(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, m.method, m.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from message
	for key, value := range msg.Headers {
		req.Header.Set(key, value)
	}

	// Create HTTP message
	httpMsg := HTTPMessage{
		Request:  req,
		Body:     body,
		Metadata: make(map[string]interface{}),
	}

	// Create new message
	result := NewMessage(httpMsg)
	result.ID = msg.ID
	result.CorrelationID = msg.CorrelationID
	result.Timestamp = msg.Timestamp

	// Merge history
	result.History = append(msg.History, result.History...)
	result.AddHistoryEntry(m.name, "message_to_http_complete", nil)

	return result, nil
}

// HTTPOrchestrationBridge bridges EIP patterns with existing HTTP orchestration
type HTTPOrchestrationBridge[T any] struct {
	name              string
	messageChannel    Channel[T]
	httpOrchestrator  *Orchestrator[any]
}

// NewHTTPOrchestrationBridge creates a bridge between EIP and HTTP orchestration
func NewHTTPOrchestrationBridge[T any](
	name string,
	messageChannel Channel[T],
	httpOrchestrator *Orchestrator[any],
) *HTTPOrchestrationBridge[T] {
	return &HTTPOrchestrationBridge[T]{
		name:             name,
		messageChannel:   messageChannel,
		httpOrchestrator: httpOrchestrator,
	}
}

// ProcessMessage processes a message through HTTP orchestration
func (b *HTTPOrchestrationBridge[T]) ProcessMessage(ctx context.Context, msg *Message[T]) error {
	msg.AddHistoryEntry(b.name, "http_orchestration_bridge_start", nil)

	// In a real implementation, you would:
	// 1. Convert the message to an HTTP-compatible format
	// 2. Execute the orchestration
	// 3. Convert the result back to a message
	// 4. Send to the output channel

	msg.AddHistoryEntry(b.name, "http_orchestration_bridge_complete", nil)
	return nil
}

// EIPPatternBuilder provides a fluent API for building EIP patterns
type EIPPatternBuilder[T any] struct {
	name           string
	inputChannel   Channel[T]
	outputChannel  Channel[T]
	transformers   []MessageTransformer[T, T]
	filters        []MessagePredicate[T]
	enrichers      []ContentEnricher[T]
	errorChannel   Channel[T]
}

// NewEIPPatternBuilder creates a new EIP pattern builder
func NewEIPPatternBuilder[T any](name string) *EIPPatternBuilder[T] {
	return &EIPPatternBuilder[T]{
		name:         name,
		transformers: make([]MessageTransformer[T, T], 0),
		filters:      make([]MessagePredicate[T], 0),
		enrichers:    make([]ContentEnricher[T], 0),
	}
}

// WithInputChannel sets the input channel
func (b *EIPPatternBuilder[T]) WithInputChannel(channel Channel[T]) *EIPPatternBuilder[T] {
	b.inputChannel = channel
	return b
}

// WithOutputChannel sets the output channel
func (b *EIPPatternBuilder[T]) WithOutputChannel(channel Channel[T]) *EIPPatternBuilder[T] {
	b.outputChannel = channel
	return b
}

// WithErrorChannel sets the error channel
func (b *EIPPatternBuilder[T]) WithErrorChannel(channel Channel[T]) *EIPPatternBuilder[T] {
	b.errorChannel = channel
	return b
}

// AddTransformer adds a transformer to the pipeline
func (b *EIPPatternBuilder[T]) AddTransformer(transformer MessageTransformer[T, T]) *EIPPatternBuilder[T] {
	b.transformers = append(b.transformers, transformer)
	return b
}

// AddFilter adds a filter predicate
func (b *EIPPatternBuilder[T]) AddFilter(filter MessagePredicate[T]) *EIPPatternBuilder[T] {
	b.filters = append(b.filters, filter)
	return b
}

// AddEnricher adds a content enricher
func (b *EIPPatternBuilder[T]) AddEnricher(enricher ContentEnricher[T]) *EIPPatternBuilder[T] {
	b.enrichers = append(b.enrichers, enricher)
	return b
}

// Build builds the pattern pipeline
func (b *EIPPatternBuilder[T]) Build() (*EIPPattern[T], error) {
	if b.inputChannel == nil {
		return nil, fmt.Errorf("input channel is required")
	}

	if b.outputChannel == nil {
		return nil, fmt.Errorf("output channel is required")
	}

	return &EIPPattern[T]{
		name:          b.name,
		inputChannel:  b.inputChannel,
		outputChannel: b.outputChannel,
		transformers:  b.transformers,
		filters:       b.filters,
		enrichers:     b.enrichers,
		errorChannel:  b.errorChannel,
	}, nil
}

// EIPPattern represents a complete EIP pattern
type EIPPattern[T any] struct {
	name          string
	inputChannel  Channel[T]
	outputChannel Channel[T]
	transformers  []MessageTransformer[T, T]
	filters       []MessagePredicate[T]
	enrichers     []ContentEnricher[T]
	errorChannel  Channel[T]
	running       bool
	stopChan      chan struct{}
}

// Start starts processing messages
func (p *EIPPattern[T]) Start(ctx context.Context) error {
	if p.running {
		return fmt.Errorf("pattern already running")
	}

	p.running = true
	p.stopChan = make(chan struct{})

	go p.process(ctx)

	return nil
}

// process processes messages through the pattern
func (p *EIPPattern[T]) process(ctx context.Context) {
	for {
		select {
		case <-p.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := p.inputChannel.Receive(ctx)
			if err != nil {
				continue
			}

			// Apply filters
			shouldProcess := true
			for _, filter := range p.filters {
				if !filter(msg) {
					shouldProcess = false
					break
				}
			}

			if !shouldProcess {
				continue
			}

			// Apply enrichers
			for _, enricher := range p.enrichers {
				enriched, err := enricher.Enrich(ctx, msg)
				if err != nil {
					if p.errorChannel != nil {
						_ = p.errorChannel.Send(ctx, msg)
					}
					continue
				}
				msg = enriched
			}

			// Apply transformers
			currentMsg := msg
			for _, transformer := range p.transformers {
				transformed, err := transformer.Transform(ctx, currentMsg)
				if err != nil {
					if p.errorChannel != nil {
						_ = p.errorChannel.Send(ctx, msg)
					}
					continue
				}
				currentMsg = transformed
			}

			// Send to output
			_ = p.outputChannel.Send(ctx, currentMsg)
		}
	}
}

// Stop stops processing
func (p *EIPPattern[T]) Stop() error {
	if !p.running {
		return fmt.Errorf("pattern not running")
	}

	p.running = false
	close(p.stopChan)

	return nil
}

// ScatterGather implements the scatter-gather pattern
type ScatterGather[T any, R any] struct {
	name          string
	recipients    []MessageHandler[T, R]
	aggregator    MessageAggregator[R, R]
	timeout       time.Duration
}

// NewScatterGather creates a new scatter-gather pattern
func NewScatterGather[T any, R any](
	name string,
	aggregator MessageAggregator[R, R],
	timeout time.Duration,
) *ScatterGather[T, R] {
	return &ScatterGather[T, R]{
		name:       name,
		recipients: make([]MessageHandler[T, R], 0),
		aggregator: aggregator,
		timeout:    timeout,
	}
}

// AddRecipient adds a recipient handler
func (s *ScatterGather[T, R]) AddRecipient(handler MessageHandler[T, R]) *ScatterGather[T, R] {
	s.recipients = append(s.recipients, handler)
	return s
}

// Execute executes scatter-gather
func (s *ScatterGather[T, R]) Execute(ctx context.Context, msg *Message[T]) (*Message[R], error) {
	msg.AddHistoryEntry(s.name, "scatter_gather_start", map[string]interface{}{
		"recipients": len(s.recipients),
	})

	// Create context with timeout
	scatterCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Scatter - send to all recipients in parallel
	type result struct {
		msg *Message[R]
		err error
	}

	results := make(chan result, len(s.recipients))

	for _, recipient := range s.recipients {
		go func(handler MessageHandler[T, R]) {
			res, err := handler.Handle(scatterCtx, msg.Clone())
			results <- result{msg: res, err: err}
		}(recipient)
	}

	// Gather results
	var responses []*Message[R]
	for i := 0; i < len(s.recipients); i++ {
		select {
		case res := <-results:
			if res.err == nil && res.msg != nil {
				responses = append(responses, res.msg)
			}
		case <-scatterCtx.Done():
			// Timeout or cancellation
			break
		}
	}

	msg.AddHistoryEntry(s.name, "scatter_gather_gathered", map[string]interface{}{
		"responses": len(responses),
	})

	// Aggregate responses
	if len(responses) == 0 {
		return nil, fmt.Errorf("no responses received")
	}

	aggregated, err := s.aggregator.Aggregate(ctx, responses)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}

	aggregated.AddHistoryEntry(s.name, "scatter_gather_complete", nil)
	return aggregated, nil
}
