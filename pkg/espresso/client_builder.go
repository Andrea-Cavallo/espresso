package espresso

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type transportWrapper struct {
	transport *http.Transport
}

func (tw *transportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	return tw.transport.RoundTrip(req)
}

func (tw *transportWrapper) Close() error {
	return nil
}

type ClientBuilder[T any] struct {
	config ClientConfig
}

func NewClientBuilder[T any]() *ClientBuilder[T] {
	return &ClientBuilder[T]{
		config: ClientConfig{
			Timeout:         30 * time.Second,
			DefaultHeaders:  make(map[string]string),
			MaxIdleConns:    100,
			MaxConnsPerHost: 10,
			IdleConnTimeout: 90 * time.Second,
		},
	}
}

func (cb *ClientBuilder[T]) WithBaseURL(baseURL string) *ClientBuilder[T] {
	cb.config.BaseURL = baseURL
	return cb
}

func (cb *ClientBuilder[T]) WithTimeout(timeout time.Duration) *ClientBuilder[T] {
	cb.config.Timeout = timeout
	return cb
}

func (cb *ClientBuilder[T]) WithDefaultHeaders(headers map[string]string) *ClientBuilder[T] {
	cb.config.DefaultHeaders = headers
	return cb
}

func (cb *ClientBuilder[T]) WithDefaultHeader(key, value string) *ClientBuilder[T] {
	if cb.config.DefaultHeaders == nil {
		cb.config.DefaultHeaders = make(map[string]string)
	}
	cb.config.DefaultHeaders[key] = value
	return cb
}

func (cb *ClientBuilder[T]) WithDefaultAuth(auth AuthProvider) *ClientBuilder[T] {
	cb.config.DefaultAuth = auth
	return cb
}

func (cb *ClientBuilder[T]) WithBearerToken(token string) *ClientBuilder[T] {
	cb.config.DefaultAuth = &BearerTokenAuth{Token: token}
	return cb
}

func (cb *ClientBuilder[T]) WithBasicAuth(username, password string) *ClientBuilder[T] {
	cb.config.DefaultAuth = &BasicAuthProvider{
		Username: username,
		Password: password,
	}
	return cb
}

func (cb *ClientBuilder[T]) WithRetry(config RetryConfig) *ClientBuilder[T] {
	cb.config.RetryConfig = &config
	return cb
}

func (cb *ClientBuilder[T]) WithExponentialBackoff(maxAttempts int, baseDelay, maxDelay time.Duration) *ClientBuilder[T] {
	cb.config.RetryConfig = &RetryConfig{
		MaxAttempts:     maxAttempts,
		BaseDelay:       baseDelay,
		MaxDelay:        maxDelay,
		JitterEnabled:   true,
		JitterType:      JitterFull,
		BackoffStrategy: BackoffExponential,
		RetriableStatus: []int{429, 500, 502, 503, 504},
	}
	return cb
}

func (cb *ClientBuilder[T]) WithCircuitBreaker(config CircuitBreakerConfig) *ClientBuilder[T] {
	cb.config.CircuitConfig = &config
	return cb
}

func (cb *ClientBuilder[T]) WithDefaultCircuitBreaker() *ClientBuilder[T] {
	cb.config.CircuitConfig = &CircuitBreakerConfig{
		MaxRequests:      10,
		Interval:         60 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		HalfOpenMaxReqs:  5,
	}
	return cb
}

func (cb *ClientBuilder[T]) WithMetrics() *ClientBuilder[T] {
	cb.config.MetricsEnabled = true
	return cb
}

func (cb *ClientBuilder[T]) WithPrometheusMetrics(namespace string) *ClientBuilder[T] {
	cb.config.MetricsEnabled = true
	return cb
}

func (cb *ClientBuilder[T]) WithTracing() *ClientBuilder[T] {
	cb.config.TracingEnabled = true
	return cb
}

func (cb *ClientBuilder[T]) WithCache() *ClientBuilder[T] {
	cb.config.CacheEnabled = true
	return cb
}

func (cb *ClientBuilder[T]) WithRateLimiter(rps, burst int) *ClientBuilder[T] {
	cb.config.RateLimiterConfig = &RateLimiterConfig{
		RequestsPerSecond: rps,
		Burst:             burst,
	}
	return cb
}

func (cb *ClientBuilder[T]) WithTokenBucketRateLimit(rps, burst int) *ClientBuilder[T] {
	return cb.WithRateLimiter(rps, burst)
}

func (cb *ClientBuilder[T]) WithSlidingWindowRateLimit(limit int, window time.Duration) *ClientBuilder[T] {
	cb.config.RateLimiterConfig = &RateLimiterConfig{
		Limit:      limit,
		Window:     window,
		LimiterType: "sliding_window",
	}
	return cb
}

func (cb *ClientBuilder[T]) WithFixedWindowRateLimit(limit int, window time.Duration) *ClientBuilder[T] {
	cb.config.RateLimiterConfig = &RateLimiterConfig{
		Limit:      limit,
		Window:     window,
		LimiterType: "fixed_window",
	}
	return cb
}

func (cb *ClientBuilder[T]) WithTransport(transport Transport) *ClientBuilder[T] {
	cb.config.Transport = transport
	return cb
}

func (cb *ClientBuilder[T]) WithTLS(config *tls.Config) *ClientBuilder[T] {
	if cb.config.Transport == nil {
		cb.config.Transport = &transportWrapper{transport: cb.createDefaultTransport()}
	}

	if tw, ok := cb.config.Transport.(*transportWrapper); ok {
		tw.transport.TLSClientConfig = config
	}
	return cb
}

func (cb *ClientBuilder[T]) WithInsecureTLS() *ClientBuilder[T] {
	return cb.WithTLS(&tls.Config{InsecureSkipVerify: true})
}

func (cb *ClientBuilder[T]) WithConnectionPooling(maxIdleConns, maxConnsPerHost int, idleTimeout time.Duration) *ClientBuilder[T] {
	cb.config.MaxIdleConns = maxIdleConns
	cb.config.MaxConnsPerHost = maxConnsPerHost
	cb.config.IdleConnTimeout = idleTimeout
	return cb
}

func (cb *ClientBuilder[T]) WithUserAgent(userAgent string) *ClientBuilder[T] {
	return cb.WithDefaultHeader("User-Agent", userAgent)
}

func (cb *ClientBuilder[T]) WithMiddleware(middleware Middleware) *ClientBuilder[T] {
	cb.config.Middlewares = append(cb.config.Middlewares, middleware)
	return cb
}

func (cb *ClientBuilder[T]) WithEventHook(hook EventHook) *ClientBuilder[T] {
	cb.config.EventHooks = append(cb.config.EventHooks, hook)
	return cb
}

func (cb *ClientBuilder[T]) WithLogger(logger Logger) *ClientBuilder[T] {
	cb.config.Logger = logger
	return cb
}

func (cb *ClientBuilder[T]) Build() *Client[T] {
	if cb.config.Transport == nil {
		cb.config.Transport = &transportWrapper{transport: cb.createDefaultTransport()}
	}

	client := &Client[T]{
		config: cb.config,
		httpClient: &http.Client{
			Timeout:   cb.config.Timeout,
			Transport: cb.config.Transport,
		},
	}

	// Configura retry strategy se presente
	if cb.config.RetryConfig != nil {
		client.SetRetryStrategy(NewRetryStrategy(*cb.config.RetryConfig))
	}

	// Configura circuit breaker se presente
	if cb.config.CircuitConfig != nil {
		client.SetCircuitBreaker(NewCircuitBreaker("default", *cb.config.CircuitConfig))
	}

	// Configura cache di default se abilitata
	if cb.config.CacheEnabled {
		client.SetCacheProvider(NewInMemoryCache[T]())
	}

	// Configura rate limiter se presente
	if cb.config.RateLimiterConfig != nil {
		cfg := cb.config.RateLimiterConfig
		var limiter RateLimiter

		switch cfg.LimiterType {
		case "sliding_window":
			limiter = NewSlidingWindowRateLimiter(cfg.Limit, cfg.Window)
		case "fixed_window":
			limiter = NewFixedWindowRateLimiter(cfg.Limit, cfg.Window)
		default: // token_bucket o default
			rps := cfg.RequestsPerSecond
			burst := cfg.Burst
			if rps == 0 {
				rps = 100 // default 100 req/sec
			}
			if burst == 0 {
				burst = 200 // default burst 200
			}
			limiter = NewTokenBucketRateLimiter(rps, burst)
		}

		client.SetRateLimiter(limiter)
	}

	return client
}

func (cb *ClientBuilder[T]) createDefaultTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          cb.config.MaxIdleConns,
		MaxIdleConnsPerHost:   cb.config.MaxConnsPerHost,
		IdleConnTimeout:       cb.config.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    false,
	}
}

type BearerTokenAuth struct {
	Token string
}

func (b *BearerTokenAuth) SetAuth(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+b.Token)
	return nil
}

func (b *BearerTokenAuth) Name() string {
	return "bearer"
}

type BasicAuthProvider struct {
	Username string
	Password string
}

func (b *BasicAuthProvider) SetAuth(req *http.Request) error {
	req.SetBasicAuth(b.Username, b.Password)
	return nil
}

func (b *BasicAuthProvider) Name() string {
	return "basic"
}

func NewDefaultClient[T any]() *Client[T] {
	return NewClientBuilder[T]().
		WithTimeout(30 * time.Second).
		WithUserAgent("espresso/1.0").
		WithMetrics().
		Build()
}

func NewRetryClient[T any](maxAttempts int) *Client[T] {
	return NewClientBuilder[T]().
		WithTimeout(30*time.Second).
		WithExponentialBackoff(maxAttempts, 100*time.Millisecond, 10*time.Second).
		WithMetrics().
		Build()
}

func NewResilientClient[T any]() *Client[T] {
	return NewClientBuilder[T]().
		WithTimeout(30*time.Second).
		WithExponentialBackoff(3, 100*time.Millisecond, 10*time.Second).
		WithDefaultCircuitBreaker().
		WithMetrics().
		WithCache().
		Build()
}

func NewFastClient[T any]() *Client[T] {
	return NewClientBuilder[T]().
		WithTimeout(5*time.Second).
		WithConnectionPooling(200, 50, 60*time.Second).
		WithMetrics().
		Build()
}
