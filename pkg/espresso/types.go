package espresso

import (
	"net/http"
	"time"
)

// Response rappresenta una risposta HTTP generica
type Response[T any] struct {
	Data       *T
	StatusCode int
	Headers    http.Header
	RawBody    []byte
	Duration   time.Duration
	Cached     bool
	Attempts   int
}

// RequestConfig contiene la configurazione per una richiesta
type RequestConfig struct {
	// Feature flags
	EnableRetry          bool
	EnableCircuitBreaker bool
	EnableMetrics        bool
	EnableCache          bool
	EnableTracing        bool

	// Headers e parametri
	Headers     map[string]string
	QueryParams map[string]string

	// Authentication
	BearerToken string
	BasicAuth   *BasicAuth
	CustomAuth  AuthProvider

	// Timeout specifico per questa richiesta
	Timeout time.Duration

	// Cache settings
	CacheKey string
	CacheTTL time.Duration

	// Content-Type override
	ContentType string

	// Retry specifico per questa richiesta
	RetryConfig *RetryConfig

	// Circuit breaker specifico
	CircuitBreakerConfig *CircuitBreakerConfig
}

// BasicAuth rappresenta l'autenticazione basic
type BasicAuth struct {
	Username string
	Password string
}

// RetryConfig contiene la configurazione del retry
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

// CircuitBreakerConfig contiene la configurazione del circuit breaker

// ClientConfig contiene la configurazione globale del client
type ClientConfig struct {
	BaseURL         string
	Timeout         time.Duration
	DefaultHeaders  map[string]string
	DefaultAuth     AuthProvider
	RetryConfig     *RetryConfig
	CircuitConfig   *CircuitBreakerConfig
	MetricsEnabled  bool
	TracingEnabled  bool
	CacheEnabled    bool
	Transport       Transport
	Middlewares     []Middleware
	EventHooks      []EventHook
	Logger          Logger
	MaxIdleConns    int
	MaxConnsPerHost int
	IdleConnTimeout time.Duration
}

// RequestBuilder fornisce un'API fluente per costruire richieste
type RequestBuilder[T any] struct {
	client *Client[T]
	url    string
	config RequestConfig
	body   any
}

// JitterType definisce i tipi di jitter disponibili
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

// BackoffStrategy definisce le strategie di backoff
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

// CircuitState rappresenta lo stato del circuit breaker
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

// SpanStatusCode rappresenta lo stato di uno span
type SpanStatusCode int

const (
	SpanStatusOK SpanStatusCode = iota
	SpanStatusError
	SpanStatusTimeout
)

// ErrorType definisce i tipi di errori
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

// HTTPMethod definisce i metodi HTTP supportati
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

// ContentType definisce i content type comuni
type ContentType string

const (
	ContentTypeJSON ContentType = "application/json"
	ContentTypeXML  ContentType = "application/xml"
	ContentTypeForm ContentType = "application/x-www-form-urlencoded"
	ContentTypeText ContentType = "text/plain"
)
