package espresso

import (
	"context"
	"fmt"
	"time"
)

// NewRequestBuilder crea un nuovo RequestBuilder
func NewRequestBuilder[T any](client *Client[T], url string) RequestBuilder[T] {
	return RequestBuilder[T]{
		client: client,
		url:    url,
		config: RequestConfig{
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
		},
	}
}

// WithHeader aggiunge un header alla richiesta
func (rb RequestBuilder[T]) WithHeader(key, value string) RequestBuilder[T] {
	rb.config.Headers[key] = value
	return rb
}

// WithHeaders aggiunge multiple headers alla richiesta
func (rb RequestBuilder[T]) WithHeaders(headers map[string]string) RequestBuilder[T] {
	for key, value := range headers {
		rb.config.Headers[key] = value
	}
	return rb
}

// WithBearerToken imposta il bearer token per l'autenticazione
func (rb RequestBuilder[T]) WithBearerToken(token string) RequestBuilder[T] {
	rb.config.BearerToken = token
	return rb
}

// WithBasicAuth imposta l'autenticazione basic
func (rb RequestBuilder[T]) WithBasicAuth(username, password string) RequestBuilder[T] {
	rb.config.BasicAuth = &BasicAuth{
		Username: username,
		Password: password,
	}
	return rb
}

// WithCustomAuth imposta un provider di autenticazione personalizzato
func (rb RequestBuilder[T]) WithCustomAuth(auth AuthProvider) RequestBuilder[T] {
	rb.config.CustomAuth = auth
	return rb
}

// WithQuery aggiunge un parametro di query
func (rb RequestBuilder[T]) WithQuery(key, value string) RequestBuilder[T] {
	rb.config.QueryParams[key] = value
	return rb
}

// WithQueryParams aggiunge multiple parametri di query
func (rb RequestBuilder[T]) WithQueryParams(params map[string]string) RequestBuilder[T] {
	for key, value := range params {
		rb.config.QueryParams[key] = value
	}
	return rb
}

// WithTimeout imposta il timeout per questa richiesta specifica
func (rb RequestBuilder[T]) WithTimeout(timeout time.Duration) RequestBuilder[T] {
	rb.config.Timeout = timeout
	return rb
}

// WithContentType imposta il content type
func (rb RequestBuilder[T]) WithContentType(contentType string) RequestBuilder[T] {
	rb.config.ContentType = contentType
	return rb
}

// WithBody imposta il body della richiesta
func (rb RequestBuilder[T]) WithBody(body any) RequestBuilder[T] {
	rb.body = body
	return rb
}

// EnableRetry abilita il retry per questa richiesta
func (rb RequestBuilder[T]) EnableRetry() RequestBuilder[T] {
	rb.config.EnableRetry = true
	return rb
}

// WithRetryConfig abilita il retry con configurazione specifica
func (rb RequestBuilder[T]) WithRetryConfig(config *RetryConfig) RequestBuilder[T] {
	rb.config.EnableRetry = true
	rb.config.RetryConfig = config
	return rb
}

// EnableCircuitBreaker abilita il circuit breaker per questa richiesta
func (rb RequestBuilder[T]) EnableCircuitBreaker() RequestBuilder[T] {
	rb.config.EnableCircuitBreaker = true
	return rb
}

// WithCircuitBreakerConfig abilita il circuit breaker con configurazione specifica
func (rb RequestBuilder[T]) WithCircuitBreakerConfig(config *CircuitBreakerConfig) RequestBuilder[T] {
	rb.config.EnableCircuitBreaker = true
	rb.config.CircuitBreakerConfig = config
	return rb
}

// EnableMetrics abilita la raccolta di metriche per questa richiesta
func (rb RequestBuilder[T]) EnableMetrics() RequestBuilder[T] {
	rb.config.EnableMetrics = true
	return rb
}

// EnableCache abilita il caching per questa richiesta
func (rb RequestBuilder[T]) EnableCache() RequestBuilder[T] {
	rb.config.EnableCache = true
	return rb
}

// WithCacheKey imposta la chiave di cache e abilita il caching
func (rb RequestBuilder[T]) WithCacheKey(key string) RequestBuilder[T] {
	rb.config.EnableCache = true
	rb.config.CacheKey = key
	return rb
}

// WithCacheTTL imposta il TTL della cache
func (rb RequestBuilder[T]) WithCacheTTL(ttl time.Duration) RequestBuilder[T] {
	rb.config.CacheTTL = ttl
	return rb
}

// WithCache abilita il caching con chiave e TTL
func (rb RequestBuilder[T]) WithCache(key string, ttl time.Duration) RequestBuilder[T] {
	rb.config.EnableCache = true
	rb.config.CacheKey = key
	rb.config.CacheTTL = ttl
	return rb
}

// EnableTracing abilita il tracing per questa richiesta
func (rb RequestBuilder[T]) EnableTracing() RequestBuilder[T] {
	rb.config.EnableTracing = true
	return rb
}

// EnableAll abilita tutte le feature (retry, circuit breaker, metrics, cache, tracing)
func (rb RequestBuilder[T]) EnableAll() RequestBuilder[T] {
	rb.config.EnableRetry = true
	rb.config.EnableCircuitBreaker = true
	rb.config.EnableMetrics = true
	rb.config.EnableCache = true
	rb.config.EnableTracing = true
	return rb
}

// WithConfig imposta la configurazione completa
func (rb RequestBuilder[T]) WithConfig(config RequestConfig) RequestBuilder[T] {
	if len(config.Headers) > 0 {
		for key, value := range config.Headers {
			rb.config.Headers[key] = value
		}
	}
	if len(config.QueryParams) > 0 {
		for key, value := range config.QueryParams {
			rb.config.QueryParams[key] = value
		}
	}

	if config.BearerToken != "" {
		rb.config.BearerToken = config.BearerToken
	}
	if config.BasicAuth != nil {
		rb.config.BasicAuth = config.BasicAuth
	}
	if config.CustomAuth != nil {
		rb.config.CustomAuth = config.CustomAuth
	}
	if config.Timeout > 0 {
		rb.config.Timeout = config.Timeout
	}
	if config.ContentType != "" {
		rb.config.ContentType = config.ContentType
	}
	if config.CacheKey != "" {
		rb.config.CacheKey = config.CacheKey
	}
	if config.CacheTTL > 0 {
		rb.config.CacheTTL = config.CacheTTL
	}
	if config.RetryConfig != nil {
		rb.config.RetryConfig = config.RetryConfig
	}
	if config.CircuitBreakerConfig != nil {
		rb.config.CircuitBreakerConfig = config.CircuitBreakerConfig
	}

	rb.config.EnableRetry = config.EnableRetry
	rb.config.EnableCircuitBreaker = config.EnableCircuitBreaker
	rb.config.EnableMetrics = config.EnableMetrics
	rb.config.EnableCache = config.EnableCache
	rb.config.EnableTracing = config.EnableTracing

	return rb
}

// Get esegue una richiesta GET
func (rb RequestBuilder[T]) Get(ctx context.Context) (*Response[T], error) {
	return rb.client.Get(ctx, rb.url, rb.config)
}

// Post esegue una richiesta POST
func (rb RequestBuilder[T]) Post(ctx context.Context) (*Response[T], error) {
	return rb.client.Post(ctx, rb.url, rb.body, rb.config)
}

// Put esegue una richiesta PUT
func (rb RequestBuilder[T]) Put(ctx context.Context) (*Response[T], error) {
	return rb.client.Put(ctx, rb.url, rb.body, rb.config)
}

// Delete esegue una richiesta DELETE
func (rb RequestBuilder[T]) Delete(ctx context.Context) (*Response[T], error) {
	return rb.client.Delete(ctx, rb.url, rb.config)
}

// Patch esegue una richiesta PATCH
func (rb RequestBuilder[T]) Patch(ctx context.Context) (*Response[T], error) {
	return rb.client.Patch(ctx, rb.url, rb.body, rb.config)
}

// Execute esegue la richiesta con il metodo HTTP specificato
func (rb RequestBuilder[T]) Execute(ctx context.Context, method HTTPMethod) (*Response[T], error) {
	switch method {
	case MethodGet:
		return rb.Get(ctx)
	case MethodPost:
		return rb.Post(ctx)
	case MethodPut:
		return rb.Put(ctx)
	case MethodDelete:
		return rb.Delete(ctx)
	case MethodPatch:
		return rb.Patch(ctx)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
}

// Build restituisce la configurazione costruita (utile per debug o per uso con metodi diretti)
func (rb RequestBuilder[T]) Build() RequestConfig {
	return rb.config
}

// Clone crea una copia del builder (utile per riutilizzare configurazioni base)
func (rb RequestBuilder[T]) Clone() RequestBuilder[T] {
	newBuilder := RequestBuilder[T]{
		client: rb.client,
		url:    rb.url,
		body:   rb.body,
		config: RequestConfig{
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
		},
	}

	for key, value := range rb.config.Headers {
		newBuilder.config.Headers[key] = value
	}

	for key, value := range rb.config.QueryParams {
		newBuilder.config.QueryParams[key] = value
	}

	newBuilder.config.EnableRetry = rb.config.EnableRetry
	newBuilder.config.EnableCircuitBreaker = rb.config.EnableCircuitBreaker
	newBuilder.config.EnableMetrics = rb.config.EnableMetrics
	newBuilder.config.EnableCache = rb.config.EnableCache
	newBuilder.config.EnableTracing = rb.config.EnableTracing
	newBuilder.config.BearerToken = rb.config.BearerToken
	newBuilder.config.BasicAuth = rb.config.BasicAuth
	newBuilder.config.CustomAuth = rb.config.CustomAuth
	newBuilder.config.Timeout = rb.config.Timeout
	newBuilder.config.ContentType = rb.config.ContentType
	newBuilder.config.CacheKey = rb.config.CacheKey
	newBuilder.config.CacheTTL = rb.config.CacheTTL
	newBuilder.config.RetryConfig = rb.config.RetryConfig
	newBuilder.config.CircuitBreakerConfig = rb.config.CircuitBreakerConfig

	return newBuilder
}
