package espresso

import (
	"errors"
	"net/http"
	"time"
)

// Common errors returned by the library.
var (
	ErrCacheClosed       = errors.New("cache is closed")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrCircuitOpen       = errors.New("circuit breaker is open")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrTimeout           = errors.New("request timeout")
)

// Response represents an HTTP response with typed data and metadata.
type Response[T any] struct {
	Data       *T
	StatusCode int
	Headers    http.Header
	RawBody    []byte
	Duration   time.Duration
	Cached     bool
	Attempts   int
}

// RequestConfig contains configuration for a single HTTP request.
type RequestConfig struct {
	// Feature flags
	EnableRetry          bool
	EnableCircuitBreaker bool
	EnableMetrics        bool
	EnableCache          bool
	EnableTracing        bool

	// Headers and query parameters
	Headers     map[string]string
	QueryParams map[string]string

	// Authentication
	BearerToken string
	BasicAuth   *BasicAuth
	CustomAuth  AuthProvider

	// Request-specific timeout override
	Timeout time.Duration

	// Cache settings
	CacheKey string
	CacheTTL time.Duration

	// Content-Type override
	ContentType string

	// Request-specific retry configuration
	RetryConfig *RetryConfig

	// Request-specific circuit breaker configuration
	CircuitBreakerConfig *CircuitBreakerConfig
}

// BasicAuth represents HTTP Basic Authentication credentials.
type BasicAuth struct {
	Username string
	Password string
}

// RetryConfig contains retry policy configuration.
type RetryConfig struct {
	MaxAttempts     int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	JitterEnabled   bool
	JitterType      JitterType
	RetriableStatus []int
	RetriableErrors []error
	BackoffStrategy BackoffStrategy
}

// RateLimiterConfig contains rate limiter configuration.
type RateLimiterConfig struct {
	RequestsPerSecond int
	Burst             int
	Limit             int
	Window            time.Duration
	LimiterType       string // "token_bucket", "sliding_window", "fixed_window"
}

// ClientConfig contains the global client configuration.
type ClientConfig struct {
	BaseURL           string
	Timeout           time.Duration
	DefaultHeaders    map[string]string
	DefaultAuth       AuthProvider
	RetryConfig       *RetryConfig
	CircuitConfig     *CircuitBreakerConfig
	RateLimiterConfig *RateLimiterConfig
	MetricsEnabled    bool
	TracingEnabled    bool
	CacheEnabled      bool
	Transport         Transport
	Middlewares       []Middleware
	EventHooks        []EventHook
	Logger            Logger
	MaxIdleConns      int
	MaxConnsPerHost   int
	IdleConnTimeout   time.Duration
}

// RequestBuilder provides a fluent API for building HTTP requests.
type RequestBuilder[T any] struct {
	client *Client[T]
	url    string
	config RequestConfig
	body   any
}

// JitterType defines the available jitter algorithms for retry delays.
type JitterType int

const (
	JitterFull JitterType = iota
	JitterEqual
	JitterDecorrelated
)

func (j JitterType) String() string {
	switch j {
	case JitterFull:
		return "full"
	case JitterEqual:
		return "equal"
	case JitterDecorrelated:
		return "decorrelated"
	default:
		return "unknown"
	}
}

// BackoffStrategy defines the available backoff strategies for retries.
type BackoffStrategy int

const (
	BackoffExponential BackoffStrategy = iota
	BackoffLinear
	BackoffFixed
	BackoffCustom
)

func (b BackoffStrategy) String() string {
	switch b {
	case BackoffExponential:
		return "exponential"
	case BackoffLinear:
		return "linear"
	case BackoffFixed:
		return "fixed"
	case BackoffCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (c CircuitState) String() string {
	switch c {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// SpanStatusCode represents the status of a tracing span.
type SpanStatusCode int

const (
	SpanStatusOK SpanStatusCode = iota
	SpanStatusError
	SpanStatusTimeout
)

// ErrorType defines categorized error types for monitoring and handling.
type ErrorType string

const (
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeConnection     ErrorType = "connection"
	ErrorTypeRetryExhausted ErrorType = "retry_exhausted"
	ErrorTypeCircuitOpen    ErrorType = "circuit_open"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeAuth           ErrorType = "auth"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeUnknown        ErrorType = "unknown"
)

// HTTPMethod defines the supported HTTP methods.
type HTTPMethod string

const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodDelete  HTTPMethod = "DELETE"
	MethodPatch   HTTPMethod = "PATCH"
	MethodHead    HTTPMethod = "HEAD"
	MethodOptions HTTPMethod = "OPTIONS"
)

// ContentType defines commonly used content types.
type ContentType string

const (
	ContentTypeJSON ContentType = "application/json"
	ContentTypeXML  ContentType = "application/xml"
	ContentTypeForm ContentType = "application/x-www-form-urlencoded"
	ContentTypeText ContentType = "text/plain"
)
