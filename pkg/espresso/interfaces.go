package espresso

import (
	"context"
	"net/http"
	"time"
)

// HTTPClient defines the interface for type-safe HTTP operations.
// It provides methods for standard HTTP verbs and a fluent request builder.
type HTTPClient[T any] interface {
	Get(ctx context.Context, url string, config RequestConfig) (*Response[T], error)
	Post(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Put(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Delete(ctx context.Context, url string, config RequestConfig) (*Response[T], error)
	Patch(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Request(url string) RequestBuilder[T]
}

// MetricsCollector defines the interface for collecting request metrics.
// Implementations can integrate with monitoring systems like Prometheus or Datadog.
type MetricsCollector interface {
	RecordRequest(method, endpoint string, duration time.Duration, statusCode int)
	RecordRetryAttempt(method, endpoint string, attempt int)
	RecordCircuitBreakerState(endpoint string, state CircuitState)
	RecordError(method, endpoint string, errorType string)
}

// RetryStrategy defines the interface for implementing retry policies.
// It determines whether a request should be retried and calculates backoff delays.
type RetryStrategy interface {
	ShouldRetry(attempt int, err error, response *http.Response) bool
	GetDelay(attempt int) time.Duration
	MaxAttempts() int
}

// CircuitBreaker defines the interface for circuit breaker pattern implementation.
// It prevents cascading failures by temporarily blocking requests to failing services.
type CircuitBreaker interface {
	Execute(fn func() error) error
	State() CircuitState
	Reset()
}

// AuthProvider defines the interface for authentication mechanisms.
// Implementations can provide Bearer tokens, Basic auth, API keys, or custom schemes.
type AuthProvider interface {
	SetAuth(req *http.Request) error
	Name() string
}

// CacheProvider defines the interface for caching HTTP responses.
// Implementations can use in-memory storage, Redis, or other caching backends.
type CacheProvider[T any] interface {
	Get(key string) (*T, bool)
	Set(key string, value T, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// Middleware defines the interface for request/response middleware.
// Middlewares can modify requests, inspect responses, or add cross-cutting concerns.
type Middleware interface {
	Process(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)
	Name() string
}

// ResponseDecoder defines the interface for custom response deserialization.
// Implementations can handle JSON, XML, Protocol Buffers, or custom formats.
type ResponseDecoder[T any] interface {
	Decode(data []byte, contentType string) (*T, error)
	CanDecode(contentType string) bool
}

// RequestEncoder defines the interface for custom request serialization.
// Implementations can encode data as JSON, XML, form data, or custom formats.
type RequestEncoder interface {
	Encode(data any) ([]byte, string, error)
	CanEncode(data any) bool
}

// Transport defines the interface for the underlying HTTP transport layer.
// It allows customization of connection pooling, TLS, and low-level networking.
type Transport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
	Close() error
}

// HealthChecker defines the interface for service health monitoring.
// Implementations can perform active health checks and track service availability.
type HealthChecker interface {
	Check(ctx context.Context, endpoint string) error
	IsHealthy(endpoint string) bool
}

// Logger defines the interface for structured logging.
// Implementations can integrate with logging frameworks like zap, logrus, or slog.
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	With(fields ...any) Logger
}

// RateLimiter defines the interface for rate limiting implementations.
// It controls request throughput using algorithms like token bucket or sliding window.
type RateLimiter interface {
	Allow(key string) bool
	Wait(ctx context.Context, key string) error
	Reset(key string)
}

// Tracer defines the interface for distributed tracing integration.
// Implementations can integrate with OpenTelemetry, Jaeger, or Zipkin.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span defines the interface for a single trace span.
// It represents a unit of work in a distributed trace.
type Span interface {
	SetAttribute(key string, value any)
	SetStatus(code SpanStatusCode, description string)
	RecordError(err error)
	End()
}

// EventHook defines the interface for request lifecycle hooks.
// Implementations can add custom logic at various stages of request processing.
type EventHook interface {
	OnRequestStart(ctx context.Context, req *http.Request)
	OnRequestEnd(ctx context.Context, req *http.Request, resp *http.Response, err error)
	OnRetryAttempt(ctx context.Context, req *http.Request, attempt int)
	OnCircuitBreakerStateChange(endpoint string, from, to CircuitState)
}
