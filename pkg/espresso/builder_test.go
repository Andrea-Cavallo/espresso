package espresso

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewRequestBuilder(t *testing.T) {
	client := &Client[string]{}
	url := "https://api.example.com/users"

	builder := NewRequestBuilder[string](client, url)

	if builder.client != client {
		t.Error("Client non impostato correttamente")
	}

	if builder.url != url {
		t.Errorf("URL atteso %s, ricevuto %s", url, builder.url)
	}

	if builder.config.Headers == nil {
		t.Error("Headers map non inizializzata")
	}

	if builder.config.QueryParams == nil {
		t.Error("QueryParams map non inizializzata")
	}
}

func TestWithHeader(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.WithHeader("Content-Type", "application/json")

	if result.config.Headers["Content-Type"] != "application/json" {
		t.Error("Header non aggiunto correttamente")
	}
}

func TestWithHeaders(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token123",
	}

	result := builder.WithHeaders(headers)

	for key, value := range headers {
		if result.config.Headers[key] != value {
			t.Errorf("Header %s non impostato correttamente", key)
		}
	}
}

func TestWithBearerToken(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	token := "my-secret-token"
	result := builder.WithBearerToken(token)

	if result.config.BearerToken != token {
		t.Errorf("Bearer token atteso %s, ricevuto %s", token, result.config.BearerToken)
	}
}

func TestWithBasicAuth(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	username := "user"
	password := "pass"

	result := builder.WithBasicAuth(username, password)

	if result.config.BasicAuth == nil {
		t.Fatal("BasicAuth non impostato")
	}

	if result.config.BasicAuth.Username != username {
		t.Errorf("Username atteso %s, ricevuto %s", username, result.config.BasicAuth.Username)
	}

	if result.config.BasicAuth.Password != password {
		t.Errorf("Password attesa %s, ricevuta %s", password, result.config.BasicAuth.Password)
	}
}

func TestWithQuery(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.WithQuery("page", "1")

	if result.config.QueryParams["page"] != "1" {
		t.Error("Query param non aggiunto correttamente")
	}
}

func TestWithQueryParams(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	params := map[string]string{
		"page":  "1",
		"limit": "10",
	}

	result := builder.WithQueryParams(params)

	for key, value := range params {
		if result.config.QueryParams[key] != value {
			t.Errorf("Query param %s non impostato correttamente", key)
		}
	}
}

func TestWithTimeout(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	timeout := 5 * time.Second
	result := builder.WithTimeout(timeout)

	if result.config.Timeout != timeout {
		t.Errorf("Timeout atteso %v, ricevuto %v", timeout, result.config.Timeout)
	}
}

func TestWithContentType(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	contentType := "application/xml"
	result := builder.WithContentType(contentType)

	if result.config.ContentType != contentType {
		t.Errorf("ContentType atteso %s, ricevuto %s", contentType, result.config.ContentType)
	}
}

func TestWithBody(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	body := map[string]string{"name": "John"}
	result := builder.WithBody(body)

	if !reflect.DeepEqual(result.body, body) {
		t.Error("Body non impostato correttamente")
	}
}

func TestEnableRetry(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableRetry()

	if !result.config.EnableRetry {
		t.Error("Retry non abilitato")
	}
}

func TestWithRetryConfig(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	retryConfig := &RetryConfig{MaxAttempts: 5}
	result := builder.WithRetryConfig(retryConfig)

	if !result.config.EnableRetry {
		t.Error("Retry non abilitato")
	}

	if result.config.RetryConfig != retryConfig {
		t.Error("RetryConfig non impostato correttamente")
	}
}

func TestEnableCircuitBreaker(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableCircuitBreaker()

	if !result.config.EnableCircuitBreaker {
		t.Error("CircuitBreaker non abilitato")
	}
}

func TestEnableMetrics(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableMetrics()

	if !result.config.EnableMetrics {
		t.Error("Metrics non abilitate")
	}
}

func TestEnableCache(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableCache()

	if !result.config.EnableCache {
		t.Error("Cache non abilitata")
	}
}

func TestWithCacheKey(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	cacheKey := "user:123"
	result := builder.WithCacheKey(cacheKey)

	if !result.config.EnableCache {
		t.Error("Cache non abilitata")
	}

	if result.config.CacheKey != cacheKey {
		t.Errorf("CacheKey attesa %s, ricevuta %s", cacheKey, result.config.CacheKey)
	}
}

func TestWithCacheTTL(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	ttl := 10 * time.Minute
	result := builder.WithCacheTTL(ttl)

	if result.config.CacheTTL != ttl {
		t.Errorf("CacheTTL atteso %v, ricevuto %v", ttl, result.config.CacheTTL)
	}
}

func TestWithCache(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	cacheKey := "product:456"
	ttl := 15 * time.Minute

	result := builder.WithCache(cacheKey, ttl)

	if !result.config.EnableCache {
		t.Error("Cache non abilitata")
	}

	if result.config.CacheKey != cacheKey {
		t.Errorf("CacheKey attesa %s, ricevuta %s", cacheKey, result.config.CacheKey)
	}

	if result.config.CacheTTL != ttl {
		t.Errorf("CacheTTL atteso %v, ricevuto %v", ttl, result.config.CacheTTL)
	}
}

func TestEnableTracing(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableTracing()

	if !result.config.EnableTracing {
		t.Error("Tracing non abilitato")
	}
}

func TestEnableAll(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	result := builder.EnableAll()

	if !result.config.EnableRetry {
		t.Error("Retry non abilitato")
	}

	if !result.config.EnableCircuitBreaker {
		t.Error("CircuitBreaker non abilitato")
	}

	if !result.config.EnableMetrics {
		t.Error("Metrics non abilitate")
	}

	if !result.config.EnableCache {
		t.Error("Cache non abilitata")
	}

	if !result.config.EnableTracing {
		t.Error("Tracing non abilitato")
	}
}

func TestWithConfig(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com")

	config := RequestConfig{
		Headers:              map[string]string{"X-Custom": "value"},
		QueryParams:          map[string]string{"filter": "active"},
		BearerToken:          "token123",
		Timeout:              30 * time.Second,
		ContentType:          "application/json",
		EnableRetry:          true,
		EnableCircuitBreaker: true,
		EnableMetrics:        true,
		EnableCache:          true,
		EnableTracing:        true,
		CacheKey:             "test-key",
		CacheTTL:             5 * time.Minute,
	}

	result := builder.WithConfig(config)

	if result.config.Headers["X-Custom"] != "value" {
		t.Error("Headers non impostati correttamente")
	}

	if result.config.QueryParams["filter"] != "active" {
		t.Error("QueryParams non impostati correttamente")
	}

	if result.config.BearerToken != "token123" {
		t.Error("BearerToken non impostato correttamente")
	}

	if result.config.Timeout != 30*time.Second {
		t.Error("Timeout non impostato correttamente")
	}

	if !result.config.EnableRetry {
		t.Error("EnableRetry non impostato correttamente")
	}
}

func TestBuild(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com").
		WithHeader("Content-Type", "application/json").
		WithQuery("page", "1").
		EnableRetry()

	config := builder.Build()

	if config.Headers["Content-Type"] != "application/json" {
		t.Error("Build non restituisce la configurazione corretta")
	}

	if config.QueryParams["page"] != "1" {
		t.Error("Build non restituisce query params corretti")
	}

	if !config.EnableRetry {
		t.Error("Build non restituisce EnableRetry corretto")
	}
}

func TestClone(t *testing.T) {
	client := &Client[string]{}
	builder := NewRequestBuilder[string](client, "https://api.example.com").
		WithHeader("Authorization", "Bearer token").
		WithQuery("limit", "10").
		WithTimeout(5 * time.Second).
		EnableRetry().
		EnableMetrics()

	cloned := builder.Clone()

	if cloned.client != builder.client {
		t.Error("Client non clonato correttamente")
	}

	if cloned.url != builder.url {
		t.Error("URL non clonato correttamente")
	}

	if cloned.config.Headers["Authorization"] != "Bearer token" {
		t.Error("Headers non clonati correttamente")
	}

	if cloned.config.QueryParams["limit"] != "10" {
		t.Error("QueryParams non clonati correttamente")
	}

	if cloned.config.Timeout != 5*time.Second {
		t.Error("Timeout non clonato correttamente")
	}

	if !cloned.config.EnableRetry {
		t.Error("EnableRetry non clonato correttamente")
	}

	if !cloned.config.EnableMetrics {
		t.Error("EnableMetrics non clonato correttamente")
	}

	cloned.WithHeader("X-New", "value")

	if builder.config.Headers["X-New"] == "value" {
		t.Error("La modifica del clone ha influenzato l'originale")
	}
}

func TestFluentInterface(t *testing.T) {
	client := &Client[string]{}

	result := NewRequestBuilder[string](client, "https://api.example.com").
		WithHeader("Content-Type", "application/json").
		WithQuery("page", "1").
		WithTimeout(10 * time.Second).
		EnableRetry().
		EnableMetrics()

	if result.config.Headers["Content-Type"] != "application/json" {
		t.Error("Fluent interface non mantiene gli headers")
	}

	if result.config.QueryParams["page"] != "1" {
		t.Error("Fluent interface non mantiene i query params")
	}

	if result.config.Timeout != 10*time.Second {
		t.Error("Fluent interface non mantiene il timeout")
	}

	if !result.config.EnableRetry {
		t.Error("Fluent interface non mantiene EnableRetry")
	}

	if !result.config.EnableMetrics {
		t.Error("Fluent interface non mantiene EnableMetrics")
	}
}

func (m *mockAuthProvider) ApplyAuth(config *RequestConfig) error {
	return nil
}

type mockClient[T any] struct{}

func (m *mockClient[T]) Get(ctx context.Context, url string, config RequestConfig) (*Response[T], error) {
	return &Response[T]{}, nil
}

func (m *mockClient[T]) Post(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return &Response[T]{}, nil
}

func (m *mockClient[T]) Put(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return &Response[T]{}, nil
}

func (m *mockClient[T]) Delete(ctx context.Context, url string, config RequestConfig) (*Response[T], error) {
	return &Response[T]{}, nil
}

func (m *mockClient[T]) Patch(ctx context.Context, url string, body any, config RequestConfig) (*Response[T], error) {
	return &Response[T]{}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
