package espresso

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"time"
)

// DefaultRetryStrategy implementa RetryStrategy con jitter avanzato
type DefaultRetryStrategy struct {
	config RetryConfig
	random *rand.Rand
}

// NewRetryStrategy crea una nuova strategia di retry
func NewRetryStrategy(config RetryConfig) *DefaultRetryStrategy {
	// Imposta valori di default se non specificati
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = 100 * time.Millisecond
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.BackoffStrategy == 0 {
		config.BackoffStrategy = BackoffExponential
	}
	if config.JitterType == 0 {
		config.JitterType = JitterFull
	}
	if len(config.RetriableStatus) == 0 {
		config.RetriableStatus = []int{
			http.StatusTooManyRequests,     // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
		}
	}

	return &DefaultRetryStrategy{
		config: config,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldRetry determina se dovremmo riprovare dopo un errore
func (r *DefaultRetryStrategy) ShouldRetry(attempt int, err error, response *http.Response) bool {
	if attempt >= r.config.MaxAttempts {
		return false
	}

	if response != nil {
		return r.isRetriableStatusCode(response.StatusCode)
	}

	if err != nil {
		return r.isRetriableError(err)
	}

	return false
}

// GetDelay calcola il delay per il prossimo tentativo con jitter
func (r *DefaultRetryStrategy) GetDelay(attempt int) time.Duration {
	baseDelay := r.calculateBaseDelay(attempt)

	if !r.config.JitterEnabled {
		return baseDelay
	}

	return r.applyJitter(baseDelay, attempt)
}

// MaxAttempts restituisce il numero massimo di tentativi
func (r *DefaultRetryStrategy) MaxAttempts() int {
	return r.config.MaxAttempts
}

// calculateBaseDelay calcola il delay base senza jitter
func (r *DefaultRetryStrategy) calculateBaseDelay(attempt int) time.Duration {
	switch r.config.BackoffStrategy {
	case BackoffExponential:
		delay := time.Duration(float64(r.config.BaseDelay) * math.Pow(2, float64(attempt-1)))
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
		return delay

	case BackoffLinear:
		delay := time.Duration(int64(r.config.BaseDelay) * int64(attempt))
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
		return delay

	case BackoffFixed:
		return r.config.BaseDelay

	default:

		delay := time.Duration(float64(r.config.BaseDelay) * math.Pow(2, float64(attempt-1)))
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
		return delay
	}
}

// applyJitter applica il jitter al delay base
func (r *DefaultRetryStrategy) applyJitter(baseDelay time.Duration, attempt int) time.Duration {
	switch r.config.JitterType {
	case JitterFull:
		return r.applyFullJitter(baseDelay)
	case JitterEqual:
		return r.applyEqualJitter(baseDelay)
	case JitterDecorrelated:
		return r.applyDecorrelatedJitter(baseDelay, attempt)
	default:
		return r.applyFullJitter(baseDelay)
	}
}

// applyFullJitter applica full jitter: delay = random(0, baseDelay)
func (r *DefaultRetryStrategy) applyFullJitter(baseDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		return 0
	}
	return time.Duration(r.random.Int63n(int64(baseDelay)))
}

// applyEqualJitter applica equal jitter: delay = baseDelay/2 + random(0, baseDelay/2)
func (r *DefaultRetryStrategy) applyEqualJitter(baseDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		return 0
	}
	half := int64(baseDelay) / 2
	return time.Duration(half + r.random.Int63n(half))
}

// applyDecorrelatedJitter applica decorrelated jitter
// delay = random(baseDelay, previousDelay * 3)
func (r *DefaultRetryStrategy) applyDecorrelatedJitter(baseDelay time.Duration, attempt int) time.Duration {
	if attempt <= 1 {
		return r.applyFullJitter(baseDelay)
	}

	prevDelay := r.calculateBaseDelay(attempt - 1)
	maxDelay := time.Duration(float64(prevDelay) * 3)

	if maxDelay > r.config.MaxDelay {
		maxDelay = r.config.MaxDelay
	}

	if baseDelay >= maxDelay {
		return baseDelay
	}

	delta := int64(maxDelay - baseDelay)
	if delta <= 0 {
		return baseDelay
	}

	return baseDelay + time.Duration(r.random.Int63n(delta))
}

// isRetriableStatusCode controlla se uno status code è retriable
func (r *DefaultRetryStrategy) isRetriableStatusCode(statusCode int) bool {
	for _, code := range r.config.RetriableStatus {
		if statusCode == code {
			return true
		}
	}
	return false
}

// isRetriableError controlla se un errore è retriable
func (r *DefaultRetryStrategy) isRetriableError(err error) bool {
	for _, retriableErr := range r.config.RetriableErrors {
		if errors.Is(err, retriableErr) {
			return true
		}
	}

	// Check for common error types that are retriable
	switch {
	case isTimeoutError(err):
		return true
	case isConnectionError(err):
		return true
	default:
		return false
	}
}

// isTimeoutError checks if the error is a timeout
func isTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

// isConnectionError checks if the error is a connection error
func isConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		var opErr *net.OpError
		if errors.As(netErr, &opErr) {
			switch {
			case errors.Is(opErr.Err, syscall.ECONNREFUSED), errors.Is(opErr.Err, syscall.ECONNRESET), errors.Is(opErr.Err, syscall.ECONNABORTED):
				return true
			}
		}
	}
	return false
}

// ExponentialBackoffRetry crea una strategia con backoff esponenziale
func ExponentialBackoffRetry(maxAttempts int, baseDelay, maxDelay time.Duration, enableJitter bool) *DefaultRetryStrategy {
	config := RetryConfig{
		MaxAttempts:     maxAttempts,
		BaseDelay:       baseDelay,
		MaxDelay:        maxDelay,
		JitterEnabled:   enableJitter,
		JitterType:      JitterFull,
		BackoffStrategy: BackoffExponential,
		RetriableStatus: []int{429, 500, 502, 503, 504},
	}
	return NewRetryStrategy(config)
}

// LinearBackoffRetry crea una strategia con backoff lineare
func LinearBackoffRetry(maxAttempts int, baseDelay, maxDelay time.Duration, enableJitter bool) *DefaultRetryStrategy {
	config := RetryConfig{
		MaxAttempts:     maxAttempts,
		BaseDelay:       baseDelay,
		MaxDelay:        maxDelay,
		JitterEnabled:   enableJitter,
		JitterType:      JitterEqual,
		BackoffStrategy: BackoffLinear,
		RetriableStatus: []int{429, 500, 502, 503, 504},
	}
	return NewRetryStrategy(config)
}

// FixedDelayRetry crea una strategia con delay fisso
func FixedDelayRetry(maxAttempts int, delay time.Duration, enableJitter bool) *DefaultRetryStrategy {
	config := RetryConfig{
		MaxAttempts:     maxAttempts,
		BaseDelay:       delay,
		MaxDelay:        delay,
		JitterEnabled:   enableJitter,
		JitterType:      JitterFull,
		BackoffStrategy: BackoffFixed,
		RetriableStatus: []int{429, 500, 502, 503, 504},
	}
	return NewRetryStrategy(config)
}

// CustomRetry crea una strategia con configurazione personalizzata
func CustomRetry(config RetryConfig) *DefaultRetryStrategy {
	return NewRetryStrategy(config)
}

// NoRetry crea una strategia che non riprova mai
func NoRetry() *DefaultRetryStrategy {
	config := RetryConfig{
		MaxAttempts: 1,
	}
	return NewRetryStrategy(config)
}
