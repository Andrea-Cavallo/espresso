package espresso

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client implementa l'interfaccia HTTPClient con supporto ai generics
type Client[T any] struct {
	httpClient  *http.Client
	config      ClientConfig
	retry       RetryStrategy
	circuit     CircuitBreaker
	metrics     MetricsCollector
	cache       CacheProvider[T]
	tracer      Tracer
	decoder     ResponseDecoder[T]
	encoder     RequestEncoder
	rateLimiter RateLimiter
}

// NewClient crea una nuova istanza del client
func NewClient[T any](config ClientConfig) *Client[T] {
	client := &Client[T]{
		config: config,
	}

	client.httpClient = &http.Client{
		Timeout:   config.Timeout,
		Transport: config.Transport,
	}

	return client
}

// Get esegue una richiesta GET
func (c *Client[T]) Get(ctx context.Context, url string, config RequestConfig) (*Response[T], error) {
	return c.executeRequest(ctx, MethodGet, url, nil, config)
}

// Post esegue una richiesta POST
func (c *Client[T]) Post(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return c.executeRequest(ctx, MethodPost, url, body, config)
}

// Put esegue una richiesta PUT
func (c *Client[T]) Put(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return c.executeRequest(ctx, MethodPut, url, body, config)
}

// Delete esegue una richiesta DELETE
func (c *Client[T]) Delete(ctx context.Context, url string, config RequestConfig) (*Response[T], error) {
	return c.executeRequest(ctx, MethodDelete, url, nil, config)
}

// Patch esegue una richiesta PATCH
func (c *Client[T]) Patch(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return c.executeRequest(ctx, MethodPatch, url, body, config)
}

// Request crea un nuovo RequestBuilder
func (c *Client[T]) Request(url string) RequestBuilder[T] {
	return NewRequestBuilder(c, url)
}

// executeRequest è il metodo principale che esegue le richieste HTTP
func (c *Client[T]) executeRequest(ctx context.Context, method HTTPMethod, requestURL string, body any, config RequestConfig) (*Response[T], error) {
	start := time.Now()

	if config.EnableCache && c.cache != nil && method == MethodGet && config.CacheKey != "" {
		if cached, found := c.cache.Get(config.CacheKey); found {
			return &Response[T]{
				Data:       cached,
				StatusCode: 200,
				Duration:   time.Since(start),
				Cached:     true,
			}, nil
		}
	}

	// Rate limiting
	if c.rateLimiter != nil {
		if !c.rateLimiter.Allow(requestURL) {
			return nil, fmt.Errorf("rate limit exceeded for %s", requestURL)
		}
	}

	req, err := c.buildRequest(ctx, method, requestURL, body, config)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	if config.EnableRetry && c.retry != nil {
		return c.executeWithRetry(ctx, req, config, start)
	}

	if config.EnableCircuitBreaker && c.circuit != nil {
		var response *Response[T]
		err := c.circuit.Execute(func() error {
			var execErr error
			response, execErr = c.executeSingleRequest(ctx, req, config, start, 1)
			return execErr
		})
		return response, err
	}

	return c.executeSingleRequest(ctx, req, config, start, 1)
}

// buildRequest costruisce la richiesta HTTP
func (c *Client[T]) buildRequest(ctx context.Context, method HTTPMethod, requestURL string, body any, config RequestConfig) (*http.Request, error) {
	fullURL := c.buildURL(requestURL, config.QueryParams)

	var bodyReader io.Reader
	var contentType string

	if body != nil {
		if c.encoder != nil && c.encoder.CanEncode(body) {
			bodyBytes, ct, err := c.encoder.Encode(body)
			if err != nil {
				return nil, fmt.Errorf("failed to encode body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
			contentType = ct
		} else {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body to JSON: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
			contentType = string(ContentTypeJSON)
		}
	}

	req, err := http.NewRequestWithContext(ctx, string(method), fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req, config, contentType)

	if err := c.setAuthentication(req, config); err != nil {
		return nil, fmt.Errorf("failed to set authentication: %w", err)
	}

	return req, nil
}

// executeWithRetry esegue la richiesta con retry
func (c *Client[T]) executeWithRetry(ctx context.Context, req *http.Request, config RequestConfig, start time.Time) (*Response[T], error) {
	var lastErr error
	var lastResponse *Response[T]

	for attempt := 1; attempt <= c.retry.MaxAttempts(); attempt++ {
		clonedReq := req.Clone(ctx)

		response, err := c.executeSingleRequest(ctx, clonedReq, config, start, attempt)
		if err == nil {
			return response, nil
		}

		lastErr = err
		lastResponse = response

		if !c.retry.ShouldRetry(attempt, err, nil) {
			break
		}

		if config.EnableMetrics && c.metrics != nil {
			c.metrics.RecordRetryAttempt(string(HTTPMethod(req.Method)), req.URL.Path, attempt)
		}

		if attempt < c.retry.MaxAttempts() {
			delay := c.retry.GetDelay(attempt)
			select {
			case <-ctx.Done():
				return lastResponse, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastResponse, lastErr
}

// executeSingleRequest esegue una singola richiesta HTTP
func (c *Client[T]) executeSingleRequest(ctx context.Context, req *http.Request, config RequestConfig, start time.Time, attempt int) (*Response[T], error) {
	if config.EnableTracing && c.tracer != nil {
		spanCtx, span := c.tracer.StartSpan(ctx, fmt.Sprintf("HTTP %s", req.Method))
		defer span.End()
		req = req.WithContext(spanCtx)

		span.SetAttribute("http.method", req.Method)
		span.SetAttribute("http.url", req.URL.String())
		span.SetAttribute("attempt", attempt)
	}

	for _, hook := range c.config.EventHooks {
		hook.OnRequestStart(req.Context(), req)
	}

	response, err := c.applyMiddleware(req, func(r *http.Request) (*http.Response, error) {
		return c.httpClient.Do(r)
	})

	duration := time.Since(start)

	for _, hook := range c.config.EventHooks {
		hook.OnRequestEnd(req.Context(), req, response, err)
	}

	if err != nil {
		if config.EnableMetrics && c.metrics != nil {
			c.metrics.RecordError(req.Method, req.URL.Path, string(ErrorTypeConnection))
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if config.EnableMetrics && c.metrics != nil {
		c.metrics.RecordRequest(req.Method, req.URL.Path, duration, response.StatusCode)
	}

	var data *T
	if len(bodyBytes) > 0 && response.StatusCode >= 200 && response.StatusCode < 300 {
		if c.decoder != nil && c.decoder.CanDecode(response.Header.Get("Content-Type")) {
			decoded, err := c.decoder.Decode(bodyBytes, response.Header.Get("Content-Type"))
			if err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			data = decoded
		} else {
			var decoded T
			decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
			decoder.UseNumber()
			if err := decoder.Decode(&decoded); err == nil {
				data = &decoded
			}
		}
	}

	result := &Response[T]{
		Data:       data,
		StatusCode: response.StatusCode,
		Headers:    response.Header,
		RawBody:    bodyBytes,
		Duration:   duration,
		Attempts:   attempt,
	}

	if config.EnableCache && c.cache != nil && req.Method == "GET" &&
		response.StatusCode >= 200 && response.StatusCode < 300 &&
		config.CacheKey != "" && data != nil {
		c.cache.Set(config.CacheKey, *data, config.CacheTTL)
	}

	return result, nil
}

// buildURL costruisce l'URL completo con query parameters
func (c *Client[T]) buildURL(requestURL string, queryParams map[string]string) string {
	if c.config.BaseURL != "" && !strings.HasPrefix(requestURL, "http") {
		requestURL = strings.TrimSuffix(c.config.BaseURL, "/") + "/" + strings.TrimPrefix(requestURL, "/")
	}

	if len(queryParams) == 0 {
		return requestURL
	}

	u, err := url.Parse(requestURL)
	if err != nil {
		return requestURL
	}

	q := u.Query()
	for key, value := range queryParams {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// setHeaders imposta gli headers della richiesta
func (c *Client[T]) setHeaders(req *http.Request, config RequestConfig, contentType string) {
	for key, value := range c.config.DefaultHeaders {
		req.Header.Set(key, value)
	}

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	if contentType != "" {
		if config.ContentType != "" {
			req.Header.Set("Content-Type", config.ContentType)
		} else {
			req.Header.Set("Content-Type", contentType)
		}
	}
}

// setAuthentication imposta l'autenticazione della richiesta
func (c *Client[T]) setAuthentication(req *http.Request, config RequestConfig) error {
	if config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.BearerToken)
		return nil
	}

	if config.BasicAuth != nil {
		req.SetBasicAuth(config.BasicAuth.Username, config.BasicAuth.Password)
		return nil
	}

	if config.CustomAuth != nil {
		return config.CustomAuth.SetAuth(req)
	}

	if c.config.DefaultAuth != nil {
		return c.config.DefaultAuth.SetAuth(req)
	}

	return nil
}

// applyMiddleware applica i middleware alla richiesta
func (c *Client[T]) applyMiddleware(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	if len(c.config.Middlewares) == 0 {
		return next(req)
	}

	handler := next
	for i := len(c.config.Middlewares) - 1; i >= 0; i-- {
		middleware := c.config.Middlewares[i]
		currentHandler := handler
		handler = func(r *http.Request) (*http.Response, error) {
			return middleware.Process(r, currentHandler)
		}
	}

	return handler(req)
}

// SetRetryStrategy imposta la strategia di retry
func (c *Client[T]) SetRetryStrategy(strategy RetryStrategy) {
	c.retry = strategy
}

// SetCircuitBreaker imposta il circuit breaker
func (c *Client[T]) SetCircuitBreaker(breaker CircuitBreaker) {
	c.circuit = breaker
}

// SetMetricsCollector imposta il collector delle metriche
func (c *Client[T]) SetMetricsCollector(collector MetricsCollector) {
	c.metrics = collector
}

// SetCacheProvider imposta il provider di cache
func (c *Client[T]) SetCacheProvider(cache CacheProvider[T]) {
	c.cache = cache
}

// SetTracer imposta il tracer
func (c *Client[T]) SetTracer(tracer Tracer) {
	c.tracer = tracer
}

// SetDecoder imposta il decoder per le risposte
func (c *Client[T]) SetDecoder(decoder ResponseDecoder[T]) {
	c.decoder = decoder
}

// SetEncoder imposta l'encoder per le richieste
func (c *Client[T]) SetEncoder(encoder RequestEncoder) {
	c.encoder = encoder
}

// SetRateLimiter imposta il rate limiter
func (c *Client[T]) SetRateLimiter(limiter RateLimiter) {
	c.rateLimiter = limiter
}

// Close chiude il client e le risorse associate
func (c *Client[T]) Close() error {
	if c.config.Transport != nil {
		return c.config.Transport.Close()
	}
	return nil
}
