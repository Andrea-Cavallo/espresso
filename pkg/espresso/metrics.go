package espresso

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetricsCollector struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	errorsTotal     *prometheus.CounterVec
	retryAttempts   *prometheus.CounterVec
	circuitState    *prometheus.GaugeVec
	cacheHits       *prometheus.CounterVec
	activeRequests  *prometheus.GaugeVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	namespace       string
}

func NewPrometheusMetricsCollector(namespace string) *PrometheusMetricsCollector {
	if namespace == "" {
		namespace = "espresso"
	}

	collector := &PrometheusMetricsCollector{
		namespace: namespace,
	}

	collector.initMetrics()

	return collector
}

func (p *PrometheusMetricsCollector) initMetrics() {
	p.requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: p.namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	p.requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: p.namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	p.errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: p.namespace,
			Name:      "http_errors_total",
			Help:      "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	p.retryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: p.namespace,
			Name:      "http_retry_attempts_total",
			Help:      "Total number of retry attempts",
		},
		[]string{"method", "endpoint", "attempt"},
	)

	p.circuitState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: p.namespace,
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"endpoint"},
	)

	p.cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: p.namespace,
			Name:      "cache_operations_total",
			Help:      "Total number of cache operations",
		},
		[]string{"endpoint", "operation"}, // operation: hit, miss, set, delete
	)

	// Richieste attive
	p.activeRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: p.namespace,
			Name:      "http_active_requests",
			Help:      "Number of active HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	// Dimensione delle richieste
	p.requestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: p.namespace,
			Name:      "http_request_size_bytes",
			Help:      "Size of HTTP requests in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	// Dimensione delle risposte
	p.responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: p.namespace,
			Name:      "http_response_size_bytes",
			Help:      "Size of HTTP responses in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint", "status_code"},
	)
}

// RecordRequest registra una richiesta HTTP completata
func (p *PrometheusMetricsCollector) RecordRequest(method, endpoint string, duration time.Duration, statusCode int) {
	statusStr := strconv.Itoa(statusCode)

	p.requestDuration.WithLabelValues(method, endpoint, statusStr).Observe(duration.Seconds())
	p.requestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
}

// RecordRetryAttempt registra un tentativo di retry
func (p *PrometheusMetricsCollector) RecordRetryAttempt(method, endpoint string, attempt int) {
	attemptStr := strconv.Itoa(attempt)
	p.retryAttempts.WithLabelValues(method, endpoint, attemptStr).Inc()
}

// RecordCircuitBreakerState registra lo stato del circuit breaker
func (p *PrometheusMetricsCollector) RecordCircuitBreakerState(endpoint string, state CircuitState) {
	var stateValue float64
	switch state {
	case CircuitClosed:
		stateValue = 0
	case CircuitOpen:
		stateValue = 1
	case CircuitHalfOpen:
		stateValue = 2
	}
	p.circuitState.WithLabelValues(endpoint).Set(stateValue)
}

// RecordError registra un errore
func (p *PrometheusMetricsCollector) RecordError(method, endpoint string, errorType string) {
	p.errorsTotal.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordCacheHit registra un cache hit
func (p *PrometheusMetricsCollector) RecordCacheHit(endpoint string) {
	p.cacheHits.WithLabelValues(endpoint, "hit").Inc()
}

// RecordCacheMiss registra un cache miss
func (p *PrometheusMetricsCollector) RecordCacheMiss(endpoint string) {
	p.cacheHits.WithLabelValues(endpoint, "miss").Inc()
}

// RecordCacheSet registra un'operazione di cache set
func (p *PrometheusMetricsCollector) RecordCacheSet(endpoint string) {
	p.cacheHits.WithLabelValues(endpoint, "set").Inc()
}

// RecordCacheDelete registra un'operazione di cache delete
func (p *PrometheusMetricsCollector) RecordCacheDelete(endpoint string) {
	p.cacheHits.WithLabelValues(endpoint, "delete").Inc()
}

// IncActiveRequests incrementa il contatore delle richieste attive
func (p *PrometheusMetricsCollector) IncActiveRequests(method, endpoint string) {
	p.activeRequests.WithLabelValues(method, endpoint).Inc()
}

// DecActiveRequests decrementa il contatore delle richieste attive
func (p *PrometheusMetricsCollector) DecActiveRequests(method, endpoint string) {
	p.activeRequests.WithLabelValues(method, endpoint).Dec()
}

// RecordRequestSize registra la dimensione di una richiesta
func (p *PrometheusMetricsCollector) RecordRequestSize(method, endpoint string, size int64) {
	p.requestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordResponseSize registra la dimensione di una risposta
func (p *PrometheusMetricsCollector) RecordResponseSize(method, endpoint string, statusCode int, size int64) {
	statusStr := strconv.Itoa(statusCode)
	p.responseSize.WithLabelValues(method, endpoint, statusStr).Observe(float64(size))
}

// GetRegistry restituisce il registry Prometheus per l'export
func (p *PrometheusMetricsCollector) GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}

// SimpleMetricsCollector implementazione semplice in memoria per test
type SimpleMetricsCollector struct {
	requests     map[string]int64
	errors       map[string]int64
	retries      map[string]int64
	circuitState map[string]CircuitState
}

// NewSimpleMetricsCollector crea un collector semplice
func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		requests:     make(map[string]int64),
		errors:       make(map[string]int64),
		retries:      make(map[string]int64),
		circuitState: make(map[string]CircuitState),
	}
}

// RecordRequest implementa MetricsCollector
func (s *SimpleMetricsCollector) RecordRequest(method, endpoint string, duration time.Duration, statusCode int) {
	key := method + ":" + endpoint + ":" + strconv.Itoa(statusCode)
	s.requests[key]++
}

// RecordRetryAttempt implementa MetricsCollector
func (s *SimpleMetricsCollector) RecordRetryAttempt(method, endpoint string, attempt int) {
	key := method + ":" + endpoint + ":" + strconv.Itoa(attempt)
	s.retries[key]++
}

// RecordCircuitBreakerState implementa MetricsCollector
func (s *SimpleMetricsCollector) RecordCircuitBreakerState(endpoint string, state CircuitState) {
	s.circuitState[endpoint] = state
}

// RecordError implementa MetricsCollector
func (s *SimpleMetricsCollector) RecordError(method, endpoint string, errorType string) {
	key := method + ":" + endpoint + ":" + errorType
	s.errors[key]++
}

// GetRequestCount restituisce il numero di richieste per una chiave
func (s *SimpleMetricsCollector) GetRequestCount(method, endpoint string, statusCode int) int64 {
	key := method + ":" + endpoint + ":" + strconv.Itoa(statusCode)
	return s.requests[key]
}

// GetErrorCount restituisce il numero di errori per una chiave
func (s *SimpleMetricsCollector) GetErrorCount(method, endpoint string, errorType string) int64 {
	key := method + ":" + endpoint + ":" + errorType
	return s.errors[key]
}

// GetRetryCount restituisce il numero di retry per una chiave
func (s *SimpleMetricsCollector) GetRetryCount(method, endpoint string, attempt int) int64 {
	key := method + ":" + endpoint + ":" + strconv.Itoa(attempt)
	return s.retries[key]
}

// GetCircuitBreakerState restituisce lo stato del circuit breaker
func (s *SimpleMetricsCollector) GetCircuitBreakerState(endpoint string) CircuitState {
	return s.circuitState[endpoint]
}

// Reset resetta tutte le metriche
func (s *SimpleMetricsCollector) Reset() {
	s.requests = make(map[string]int64)
	s.errors = make(map[string]int64)
	s.retries = make(map[string]int64)
	s.circuitState = make(map[string]CircuitState)
}

// NoOpMetricsCollector implementazione che non fa nulla (per disabilitare le metriche)
type NoOpMetricsCollector struct{}

// NewNoOpMetricsCollector crea un collector che non fa nulla
func NewNoOpMetricsCollector() *NoOpMetricsCollector {
	return &NoOpMetricsCollector{}
}

// RecordRequest implementa MetricsCollector (no-op)
func (n *NoOpMetricsCollector) RecordRequest(method, endpoint string, duration time.Duration, statusCode int) {
}

// RecordRetryAttempt implementa MetricsCollector (no-op)
func (n *NoOpMetricsCollector) RecordRetryAttempt(method, endpoint string, attempt int) {}

// RecordCircuitBreakerState implementa MetricsCollector (no-op)
func (n *NoOpMetricsCollector) RecordCircuitBreakerState(endpoint string, state CircuitState) {}

// RecordError implementa MetricsCollector (no-op)
func (n *NoOpMetricsCollector) RecordError(method, endpoint string, errorType string) {}
