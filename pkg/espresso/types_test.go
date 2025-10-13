package espresso

import (
	"net/http"
	"testing"
	"time"
)

func TestResponse_Struct(t *testing.T) {
	t.Parallel()

	// Test costruzione base
	data := "test data"
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	resp := Response[string]{
		Data:       &data,
		StatusCode: 200,
		Headers:    headers,
		RawBody:    []byte("raw"),
		Duration:   100 * time.Millisecond,
		Cached:     true,
		Attempts:   2,
	}

	if *resp.Data != "test data" {
		t.Fatalf("Data got %v want 'test data'", *resp.Data)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("StatusCode got %d want 200", resp.StatusCode)
	}
	if resp.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("Headers Content-Type got %s", resp.Headers.Get("Content-Type"))
	}
	if string(resp.RawBody) != "raw" {
		t.Fatalf("RawBody got %s want 'raw'", string(resp.RawBody))
	}
	if resp.Duration != 100*time.Millisecond {
		t.Fatalf("Duration got %v", resp.Duration)
	}
	if !resp.Cached {
		t.Fatalf("Cached should be true")
	}
	if resp.Attempts != 2 {
		t.Fatalf("Attempts got %d want 2", resp.Attempts)
	}
}

func TestResponse_GenericTypes(t *testing.T) {
	t.Parallel()

	t.Run("string type", func(t *testing.T) {
		data := "hello"
		resp := Response[string]{Data: &data}
		if *resp.Data != "hello" {
			t.Fatalf("string data got %v", *resp.Data)
		}
	})

	t.Run("int type", func(t *testing.T) {
		data := 42
		resp := Response[int]{Data: &data}
		if *resp.Data != 42 {
			t.Fatalf("int data got %v", *resp.Data)
		}
	})

	t.Run("struct type", func(t *testing.T) {
		type User struct{ Name string }
		user := User{Name: "Alice"}
		resp := Response[User]{Data: &user}
		if resp.Data.Name != "Alice" {
			t.Fatalf("struct data got %v", resp.Data.Name)
		}
	})

	t.Run("nil data", func(t *testing.T) {
		resp := Response[string]{Data: nil}
		if resp.Data != nil {
			t.Fatalf("nil data should remain nil")
		}
	})
}

func TestBasicAuth(t *testing.T) {
	t.Parallel()

	auth := BasicAuth{
		Username: "user123",
		Password: "secret456",
	}

	if auth.Username != "user123" {
		t.Fatalf("Username got %s want 'user123'", auth.Username)
	}
	if auth.Password != "secret456" {
		t.Fatalf("Password got %s want 'secret456'", auth.Password)
	}
}

func TestRetryConfig_Struct(t *testing.T) {
	t.Parallel()

	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		JitterEnabled:   true,
		JitterType:      JitterFull,
		RetriableStatus: []int{429, 500, 502},
		BackoffStrategy: BackoffExponential,
	}

	if cfg.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts got %d want 5", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 100*time.Millisecond {
		t.Fatalf("BaseDelay got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Fatalf("MaxDelay got %v", cfg.MaxDelay)
	}
	if !cfg.JitterEnabled {
		t.Fatalf("JitterEnabled should be true")
	}
	if cfg.JitterType != JitterFull {
		t.Fatalf("JitterType got %v want JitterFull", cfg.JitterType)
	}
	if len(cfg.RetriableStatus) != 3 {
		t.Fatalf("RetriableStatus length got %d want 3", len(cfg.RetriableStatus))
	}
	if cfg.BackoffStrategy != BackoffExponential {
		t.Fatalf("BackoffStrategy got %v", cfg.BackoffStrategy)
	}
}

func TestRequestConfig_Struct(t *testing.T) {
	t.Parallel()

	headers := map[string]string{"X-Custom": "value"}
	queryParams := map[string]string{"q": "search"}
	basicAuth := &BasicAuth{Username: "user", Password: "pass"}
	retryConfig := &RetryConfig{MaxAttempts: 3}

	cfg := RequestConfig{
		EnableRetry:          true,
		EnableCircuitBreaker: false,
		EnableMetrics:        true,
		EnableCache:          false,
		EnableTracing:        true,
		Headers:              headers,
		QueryParams:          queryParams,
		BearerToken:          "token123",
		BasicAuth:            basicAuth,
		Timeout:              5 * time.Second,
		CacheKey:             "key1",
		CacheTTL:             10 * time.Minute,
		ContentType:          "application/json",
		RetryConfig:          retryConfig,
	}

	if !cfg.EnableRetry {
		t.Fatalf("EnableRetry should be true")
	}
	if cfg.EnableCircuitBreaker {
		t.Fatalf("EnableCircuitBreaker should be false")
	}
	if !cfg.EnableMetrics {
		t.Fatalf("EnableMetrics should be true")
	}
	if cfg.EnableCache {
		t.Fatalf("EnableCache should be false")
	}
	if !cfg.EnableTracing {
		t.Fatalf("EnableTracing should be true")
	}
	if cfg.Headers["X-Custom"] != "value" {
		t.Fatalf("Headers X-Custom got %s", cfg.Headers["X-Custom"])
	}
	if cfg.QueryParams["q"] != "search" {
		t.Fatalf("QueryParams q got %s", cfg.QueryParams["q"])
	}
	if cfg.BearerToken != "token123" {
		t.Fatalf("BearerToken got %s", cfg.BearerToken)
	}
	if cfg.BasicAuth.Username != "user" {
		t.Fatalf("BasicAuth Username got %s", cfg.BasicAuth.Username)
	}
	if cfg.Timeout != 5*time.Second {
		t.Fatalf("Timeout got %v", cfg.Timeout)
	}
	if cfg.CacheKey != "key1" {
		t.Fatalf("CacheKey got %s", cfg.CacheKey)
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Fatalf("CacheTTL got %v", cfg.CacheTTL)
	}
	if cfg.ContentType != "application/json" {
		t.Fatalf("ContentType got %s", cfg.ContentType)
	}
	if cfg.RetryConfig.MaxAttempts != 3 {
		t.Fatalf("RetryConfig MaxAttempts got %d", cfg.RetryConfig.MaxAttempts)
	}
}

func TestClientConfig_Struct(t *testing.T) {
	t.Parallel()

	defaultHeaders := map[string]string{"User-Agent": "test"}
	retryConfig := &RetryConfig{MaxAttempts: 5}

	cfg := ClientConfig{
		BaseURL:         "https://api.example.com",
		Timeout:         30 * time.Second,
		DefaultHeaders:  defaultHeaders,
		RetryConfig:     retryConfig,
		MetricsEnabled:  true,
		TracingEnabled:  false,
		CacheEnabled:    true,
		MaxIdleConns:    100,
		MaxConnsPerHost: 10,
		IdleConnTimeout: 90 * time.Second,
	}

	if cfg.BaseURL != "https://api.example.com" {
		t.Fatalf("BaseURL got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("Timeout got %v", cfg.Timeout)
	}
	if cfg.DefaultHeaders["User-Agent"] != "test" {
		t.Fatalf("DefaultHeaders User-Agent got %s", cfg.DefaultHeaders["User-Agent"])
	}
	if cfg.RetryConfig.MaxAttempts != 5 {
		t.Fatalf("RetryConfig MaxAttempts got %d", cfg.RetryConfig.MaxAttempts)
	}
	if !cfg.MetricsEnabled {
		t.Fatalf("MetricsEnabled should be true")
	}
	if cfg.TracingEnabled {
		t.Fatalf("TracingEnabled should be false")
	}
	if !cfg.CacheEnabled {
		t.Fatalf("CacheEnabled should be true")
	}
	if cfg.MaxIdleConns != 100 {
		t.Fatalf("MaxIdleConns got %d", cfg.MaxIdleConns)
	}
	if cfg.MaxConnsPerHost != 10 {
		t.Fatalf("MaxConnsPerHost got %d", cfg.MaxConnsPerHost)
	}
	if cfg.IdleConnTimeout != 90*time.Second {
		t.Fatalf("IdleConnTimeout got %v", cfg.IdleConnTimeout)
	}
}

func TestJitterType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		jitter JitterType
		want   string
	}{
		{JitterFull, "full"},
		{JitterEqual, "equal"},
		{JitterDecorrelated, "decorrelated"},
		{JitterType(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.jitter.String(); got != tt.want {
			t.Fatalf("JitterType(%d).String() got %s want %s", tt.jitter, got, tt.want)
		}
	}
}

func TestBackoffStrategy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		strategy BackoffStrategy
		want     string
	}{
		{BackoffExponential, "exponential"},
		{BackoffLinear, "linear"},
		{BackoffFixed, "fixed"},
		{BackoffCustom, "custom"},
		{BackoffStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.strategy.String(); got != tt.want {
			t.Fatalf("BackoffStrategy(%d).String() got %s want %s", tt.strategy, got, tt.want)
		}
	}
}

func TestCircuitState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Fatalf("CircuitState(%d).String() got %s want %s", tt.state, got, tt.want)
		}
	}
}

func TestSpanStatusCode_Constants(t *testing.T) {
	t.Parallel()

	if SpanStatusOK != 0 {
		t.Fatalf("SpanStatusOK should be 0, got %d", SpanStatusOK)
	}
	if SpanStatusError != 1 {
		t.Fatalf("SpanStatusError should be 1, got %d", SpanStatusError)
	}
	if SpanStatusTimeout != 2 {
		t.Fatalf("SpanStatusTimeout should be 2, got %d", SpanStatusTimeout)
	}
}

func TestErrorType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errorType ErrorType
		want      string
	}{
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypeConnection, "connection"},
		{ErrorTypeRetryExhausted, "retry_exhausted"},
		{ErrorTypeCircuitOpen, "circuit_open"},
		{ErrorTypeRateLimit, "rate_limit"},
		{ErrorTypeAuth, "auth"},
		{ErrorTypeValidation, "validation"},
		{ErrorTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.errorType) != tt.want {
			t.Fatalf("ErrorType got %s want %s", string(tt.errorType), tt.want)
		}
	}
}

func TestHTTPMethod_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method HTTPMethod
		want   string
	}{
		{MethodGet, "GET"},
		{MethodPost, "POST"},
		{MethodPut, "PUT"},
		{MethodDelete, "DELETE"},
		{MethodPatch, "PATCH"},
		{MethodHead, "HEAD"},
		{MethodOptions, "OPTIONS"},
	}

	for _, tt := range tests {
		if string(tt.method) != tt.want {
			t.Fatalf("HTTPMethod got %s want %s", string(tt.method), tt.want)
		}
	}
}

func TestContentType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType ContentType
		want        string
	}{
		{ContentTypeJSON, "application/json"},
		{ContentTypeXML, "application/xml"},
		{ContentTypeForm, "application/x-www-form-urlencoded"},
		{ContentTypeText, "text/plain"},
	}

	for _, tt := range tests {
		if string(tt.contentType) != tt.want {
			t.Fatalf("ContentType got %s want %s", string(tt.contentType), tt.want)
		}
	}
}

func TestEnumValues_Uniqueness(t *testing.T) {
	t.Parallel()

	// Verifica che i valori delle enum siano univoci
	t.Run("JitterType uniqueness", func(t *testing.T) {
		values := []JitterType{JitterFull, JitterEqual, JitterDecorrelated}
		seen := make(map[JitterType]bool)
		for _, v := range values {
			if seen[v] {
				t.Fatalf("JitterType value %d duplicated", v)
			}
			seen[v] = true
		}
	})

	t.Run("BackoffStrategy uniqueness", func(t *testing.T) {
		values := []BackoffStrategy{BackoffExponential, BackoffLinear, BackoffFixed, BackoffCustom}
		seen := make(map[BackoffStrategy]bool)
		for _, v := range values {
			if seen[v] {
				t.Fatalf("BackoffStrategy value %d duplicated", v)
			}
			seen[v] = true
		}
	})

	t.Run("CircuitState uniqueness", func(t *testing.T) {
		values := []CircuitState{CircuitClosed, CircuitOpen, CircuitHalfOpen}
		seen := make(map[CircuitState]bool)
		for _, v := range values {
			if seen[v] {
				t.Fatalf("CircuitState value %d duplicated", v)
			}
			seen[v] = true
		}
	})
}

func TestZeroValues(t *testing.T) {
	t.Parallel()

	t.Run("Response zero value", func(t *testing.T) {
		var resp Response[string]
		if resp.Data != nil {
			t.Fatalf("zero Response.Data should be nil")
		}
		if resp.StatusCode != 0 {
			t.Fatalf("zero Response.StatusCode should be 0")
		}
		if resp.Headers != nil {
			t.Fatalf("zero Response.Headers should be nil")
		}
		if resp.RawBody != nil {
			t.Fatalf("zero Response.RawBody should be nil")
		}
		if resp.Duration != 0 {
			t.Fatalf("zero Response.Duration should be 0")
		}
		if resp.Cached {
			t.Fatalf("zero Response.Cached should be false")
		}
		if resp.Attempts != 0 {
			t.Fatalf("zero Response.Attempts should be 0")
		}
	})

	t.Run("RequestConfig zero value", func(t *testing.T) {
		var cfg RequestConfig
		if cfg.EnableRetry {
			t.Fatalf("zero RequestConfig.EnableRetry should be false")
		}
		if cfg.Headers != nil {
			t.Fatalf("zero RequestConfig.Headers should be nil")
		}
		if cfg.BearerToken != "" {
			t.Fatalf("zero RequestConfig.BearerToken should be empty")
		}
		if cfg.Timeout != 0 {
			t.Fatalf("zero RequestConfig.Timeout should be 0")
		}
	})

	t.Run("RetryConfig zero value", func(t *testing.T) {
		var cfg RetryConfig
		if cfg.MaxAttempts != 0 {
			t.Fatalf("zero RetryConfig.MaxAttempts should be 0")
		}
		if cfg.JitterEnabled {
			t.Fatalf("zero RetryConfig.JitterEnabled should be false")
		}
		if cfg.RetriableStatus != nil {
			t.Fatalf("zero RetryConfig.RetriableStatus should be nil")
		}
	})
}

func TestRequestBuilder_Generic(t *testing.T) {
	t.Parallel()

	// Test che RequestBuilder possa gestire diversi tipi generici
	t.Run("string builder", func(t *testing.T) {
		var builder RequestBuilder[string]
		builder.url = "test"
		if builder.url != "test" {
			t.Fatalf("RequestBuilder[string] url got %s", builder.url)
		}
	})

	t.Run("int builder", func(t *testing.T) {
		var builder RequestBuilder[int]
		builder.url = "test"
		if builder.url != "test" {
			t.Fatalf("RequestBuilder[int] url got %s", builder.url)
		}
	})

	t.Run("struct builder", func(t *testing.T) {
		type User struct{ ID int }
		var builder RequestBuilder[User]
		builder.body = User{ID: 123}
		if user, ok := builder.body.(User); !ok || user.ID != 123 {
			t.Fatalf("RequestBuilder[User] body error")
		}
	})
}
