package espresso

import (
	"context"
	"net/http"
	"time"
)

// HTTPClient definisce l'interfaccia per il client HTTP
type HTTPClient[T any] interface {
	Get(ctx context.Context, url string, config RequestConfig) (*Response[T], error)
	Post(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Put(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Delete(ctx context.Context, url string, config RequestConfig) (*Response[T], error)
	Patch(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error)
	Request(url string) RequestBuilder[T]
}

// MetricsCollector definisce l'interfaccia per la raccolta metriche
type MetricsCollector interface {
	RecordRequest(method, endpoint string, duration time.Duration, statusCode int)
	RecordRetryAttempt(method, endpoint string, attempt int)
	RecordCircuitBreakerState(endpoint string, state CircuitState)
	RecordError(method, endpoint string, errorType string)
}

// RetryStrategy definisce l'interfaccia per le strategie di retry
type RetryStrategy interface {
	ShouldRetry(attempt int, err error, response *http.Response) bool
	GetDelay(attempt int) time.Duration
	MaxAttempts() int
}

// CircuitBreaker definisce l'interfaccia per il circuit breaker
type CircuitBreaker interface {
	Execute(fn func() error) error
	State() CircuitState
	Reset()
}

// AuthProvider definisce l'interfaccia per i provider di autenticazione
type AuthProvider interface {
	SetAuth(req *http.Request) error
	Name() string
}

// CacheProvider definisce l'interfaccia per il caching
type CacheProvider[T any] interface {
	Get(key string) (*T, bool)
	Set(key string, value T, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// Middleware definisce l'interfaccia per i middleware
type Middleware interface {
	Process(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)
	Name() string
}

// ResponseDecoder definisce l'interfaccia per decodificare risposte
type ResponseDecoder[T any] interface {
	Decode(data []byte, contentType string) (*T, error)
	CanDecode(contentType string) bool
}

// RequestEncoder definisce l'interfaccia per codificare richieste
type RequestEncoder interface {
	Encode(data any) ([]byte, string, error)
	CanEncode(data any) bool
}

// Transport definisce l'interfaccia per il transport layer
type Transport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
	Close() error
}

// HealthChecker definisce l'interfaccia per health checks
type HealthChecker interface {
	Check(ctx context.Context, endpoint string) error
	IsHealthy(endpoint string) bool
}

// Logger definisce l'interfaccia per il logging
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	With(fields ...any) Logger
}

// RateLimiter definisce l'interfaccia per il rate limiting
type RateLimiter interface {
	Allow(key string) bool
	Wait(ctx context.Context, key string) error
	Reset(key string)
}

// Tracer definisce l'interfaccia per il tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span definisce l'interfaccia per uno span di tracing
type Span interface {
	SetAttribute(key string, value any)
	SetStatus(code SpanStatusCode, description string)
	RecordError(err error)
	End()
}

// EventHook definisce l'interfaccia per gli eventi del client
type EventHook interface {
	OnRequestStart(ctx context.Context, req *http.Request)
	OnRequestEnd(ctx context.Context, req *http.Request, resp *http.Response, err error)
	OnRetryAttempt(ctx context.Context, req *http.Request, attempt int)
	OnCircuitBreakerStateChange(endpoint string, from, to CircuitState)
}
