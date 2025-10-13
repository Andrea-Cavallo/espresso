package espresso

import (
	_ "bytes"
	"crypto/tls"
	_ "io"
	"net/http"
	_ "net/http/httptest"
	"reflect"
	_ "strings"
	"testing"
	"time"
)

type MockLogger struct {
	Logs []string
}

func (m *MockLogger) Debug(msg string, fields ...any) {
	m.Logs = append(m.Logs, "DEBUG: "+msg)
}

func (m *MockLogger) Info(msg string, fields ...any) {
	m.Logs = append(m.Logs, "INFO: "+msg)
}

func (m *MockLogger) Warn(msg string, fields ...any) {
	m.Logs = append(m.Logs, "WARN: "+msg)
}

func (m *MockLogger) Error(msg string, fields ...any) {
	m.Logs = append(m.Logs, "ERROR: "+msg)
}

// MockRoundTripper simula un http.RoundTripper
type MockRoundTripper struct {
	Response *http.Response
	Error    error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, m.Error
}

func (m *MockRoundTripper) Close() error {
	return nil
}

func TestClientBuilder_WithBaseURL(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithBaseURL("https://api.example.com").Build()
	if client.config.BaseURL != "https://api.example.com" {
		t.Errorf("expected BaseURL https://api.example.com, got %s", client.config.BaseURL)
	}
}

func TestClientBuilder_WithTimeout(t *testing.T) {
	builder := NewClientBuilder[any]()
	timeout := 5 * time.Second
	client := builder.WithTimeout(timeout).Build()
	if client.httpClient.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestClientBuilder_WithDefaultHeaders(t *testing.T) {
	builder := NewClientBuilder[any]()
	headers := map[string]string{"X-Test": "test-value"}
	client := builder.WithDefaultHeaders(headers).Build()
	if !reflect.DeepEqual(client.config.DefaultHeaders, headers) {
		t.Errorf("expected headers %v, got %v", headers, client.config.DefaultHeaders)
	}
}

func TestClientBuilder_WithDefaultHeader(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithDefaultHeader("X-Key", "X-Value").Build()
	if client.config.DefaultHeaders["X-Key"] != "X-Value" {
		t.Errorf("expected X-Key=X-Value, got %v", client.config.DefaultHeaders)
	}
}

func TestClientBuilder_WithBearerToken(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithBearerToken("abc123").Build()
	if client.config.DefaultAuth == nil {
		t.Fatal("expected DefaultAuth to be set")
	}
	bearer, ok := client.config.DefaultAuth.(*BearerTokenAuth)
	if !ok {
		t.Fatalf("expected *BearerTokenAuth, got %T", client.config.DefaultAuth)
	}
	if bearer.Token != "abc123" {
		t.Errorf("expected token abc123, got %s", bearer.Token)
	}
}

func TestClientBuilder_WithBasicAuth(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithBasicAuth("user", "pass").Build()
	if client.config.DefaultAuth == nil {
		t.Fatal("expected DefaultAuth to be set")
	}
	basic, ok := client.config.DefaultAuth.(*BasicAuthProvider)
	if !ok {
		t.Fatalf("expected *BasicAuthProvider, got %T", client.config.DefaultAuth)
	}
	if basic.Username != "user" || basic.Password != "pass" {
		t.Errorf("expected user=pass, got %s=%s", basic.Username, basic.Password)
	}
}

func TestClientBuilder_WithRetry(t *testing.T) {
	builder := NewClientBuilder[any]()
	config := RetryConfig{MaxAttempts: 3}
	client := builder.WithRetry(config).Build()
	if client.config.RetryConfig == nil {
		t.Fatal("expected RetryConfig to be set")
	}
	if client.config.RetryConfig.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", client.config.RetryConfig.MaxAttempts)
	}
}

func TestClientBuilder_WithExponentialBackoff(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithExponentialBackoff(3, 100*time.Millisecond, 1*time.Second).Build()
	if client.config.RetryConfig == nil {
		t.Fatal("expected RetryConfig to be set")
	}
	if client.config.RetryConfig.BackoffStrategy != BackoffExponential {
		t.Errorf("expected BackoffExponential, got %d", client.config.RetryConfig.BackoffStrategy)
	}
}

func TestClientBuilder_WithDefaultCircuitBreaker(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithDefaultCircuitBreaker().Build()
	if client.config.CircuitConfig == nil {
		t.Fatal("expected CircuitConfig to be set")
	}
	if client.config.CircuitConfig.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold=5, got %d", client.config.CircuitConfig.FailureThreshold)
	}
}

func TestClientBuilder_WithTLS(t *testing.T) {
	builder := NewClientBuilder[any]()
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	client := builder.WithTLS(tlsConfig).Build()
	if tw, ok := client.config.Transport.(*transportWrapper); ok {
		if tw.transport.TLSClientConfig != tlsConfig {
			t.Errorf("expected TLS config to be set")
		}
	} else {
		t.Error("expected transportWrapper")
	}
}

func TestClientBuilder_WithInsecureTLS(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithInsecureTLS().Build()
	if tw, ok := client.config.Transport.(*transportWrapper); ok {
		if !tw.transport.TLSClientConfig.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify to be true")
		}
	}
}

func TestClientBuilder_WithConnectionPooling(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithConnectionPooling(200, 50, 60*time.Second).Build()
	if client.config.MaxIdleConns != 200 {
		t.Errorf("expected MaxIdleConns=200, got %d", client.config.MaxIdleConns)
	}
	if client.config.MaxConnsPerHost != 50 {
		t.Errorf("expected MaxConnsPerHost=50, got %d", client.config.MaxConnsPerHost)
	}
}

func TestClientBuilder_WithUserAgent(t *testing.T) {
	builder := NewClientBuilder[any]()
	client := builder.WithUserAgent("test-agent").Build()
	userAgent := client.config.DefaultHeaders["User-Agent"]
	if userAgent != "test-agent" {
		t.Errorf("expected User-Agent=test-agent, got %s", userAgent)
	}
}

func TestClientBuilder_Build(t *testing.T) {
	client := NewClientBuilder[any]().Build()
	if client.httpClient == nil {
		t.Fatal("expected httpClient to be created")
	}
	if client.config.DefaultHeaders == nil {
		t.Fatal("expected DefaultHeaders to be initialized")
	}
}

func TestClientBuilder_WithTransport(t *testing.T) {
	mockRT := &MockRoundTripper{}
	builder := NewClientBuilder[any]()
	client := builder.WithTransport(mockRT).Build()
	if client.config.Transport != mockRT {
		t.Error("expected transport to be set to mockRT")
	}
}

// Testa le funzioni factory
func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient[any]()
	if client.config.Timeout != 30*time.Second {
		t.Error("expected timeout 30s")
	}
	if client.config.DefaultHeaders["User-Agent"] != "espresso/1.0" {
		t.Error("expected User-Agent espresso/1.0")
	}
	if !client.config.MetricsEnabled {
		t.Error("expected MetricsEnabled true")
	}
}

func TestNewRetryClient(t *testing.T) {
	client := NewRetryClient[any](3)
	if client.config.RetryConfig == nil {
		t.Fatal("expected RetryConfig")
	}
	if client.config.RetryConfig.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3")
	}
}

func TestNewResilientClient(t *testing.T) {
	client := NewResilientClient[any]()
	if client.config.CircuitConfig == nil {
		t.Fatal("expected CircuitConfig")
	}
	if !client.config.CacheEnabled {
		t.Fatal("expected CacheEnabled true")
	}
}

func TestNewFastClient(t *testing.T) {
	client := NewFastClient[any]()
	if client.httpClient.Timeout != 5*time.Second {
		t.Error("expected Timeout=5s")
	}
	if client.config.MaxIdleConns != 200 {
		t.Error("expected MaxIdleConns=200")
	}
}
