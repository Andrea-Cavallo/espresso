package espresso

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		BaseURL: "https://api.example.com",
		Timeout: 10 * time.Second,
	}

	client := NewClient[testData](config)

	if client == nil {
		t.Fatal("Client non dovrebbe essere nil")
	}

	if client.httpClient == nil {
		t.Error("HTTP client non inizializzato")
	}

	if client.config.BaseURL != config.BaseURL {
		t.Error("Configurazione non impostata correttamente")
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Metodo atteso GET, ricevuto %s", r.Method)
		}

		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})
	response, err := client.Get(context.Background(), server.URL, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.StatusCode != 200 {
		t.Errorf("Status code atteso 200, ricevuto %d", response.StatusCode)
	}

	if response.Data == nil {
		t.Fatal("Data non dovrebbe essere nil")
	}

	if response.Data.ID != 1 {
		t.Errorf("ID atteso 1, ricevuto %d", response.Data.ID)
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Metodo atteso POST, ricevuto %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var received testData
		json.Unmarshal(body, &received)

		if received.Name != "New Item" {
			t.Errorf("Nome atteso 'New Item', ricevuto %s", received.Name)
		}

		response := testData{ID: 2, Name: received.Name}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})
	body := testData{Name: "New Item"}

	response, err := client.Post(context.Background(), server.URL, body, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.ID != 2 {
		t.Errorf("ID atteso 2, ricevuto %d", response.Data.ID)
	}
}

func TestPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Metodo atteso PUT, ricevuto %s", r.Method)
		}

		response := testData{ID: 1, Name: "Updated"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})
	body := testData{ID: 1, Name: "Updated"}

	response, err := client.Put(context.Background(), server.URL, body, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.Name != "Updated" {
		t.Errorf("Nome atteso 'Updated', ricevuto %s", response.Data.Name)
	}
}

func TestDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Metodo atteso DELETE, ricevuto %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	response, err := client.Delete(context.Background(), server.URL, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.StatusCode != http.StatusNoContent {
		t.Errorf("Status code atteso 204, ricevuto %d", response.StatusCode)
	}
}

func TestPatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Metodo atteso PATCH, ricevuto %s", r.Method)
		}

		response := testData{ID: 1, Name: "Patched"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})
	body := map[string]string{"name": "Patched"}

	response, err := client.Patch(context.Background(), server.URL, body, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.Name != "Patched" {
		t.Errorf("Nome atteso 'Patched', ricevuto %s", response.Data.Name)
	}
}

func TestRequestBuilder(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	builder := client.Request("https://api.example.com/users")

	if builder.client != client {
		t.Error("Builder non collegato correttamente al client")
	}

	if builder.url != "https://api.example.com/users" {
		t.Error("URL non impostato correttamente nel builder")
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestURL  string
		queryParams map[string]string
		expected    string
	}{
		{
			name:        "URL completo senza base",
			baseURL:     "",
			requestURL:  "https://api.example.com/users",
			queryParams: nil,
			expected:    "https://api.example.com/users",
		},
		{
			name:        "URL relativo con base",
			baseURL:     "https://api.example.com",
			requestURL:  "/users",
			queryParams: nil,
			expected:    "https://api.example.com/users",
		},
		{
			name:       "Con query params",
			baseURL:    "https://api.example.com",
			requestURL: "/users",
			queryParams: map[string]string{
				"page":  "1",
				"limit": "10",
			},
			expected: "https://api.example.com/users?limit=10&page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient[testData](ClientConfig{BaseURL: tt.baseURL})
			result := client.buildURL(tt.requestURL, tt.queryParams)

			if result != tt.expected {
				t.Errorf("URL atteso %s, ricevuto %s", tt.expected, result)
			}
		})
	}
}

func TestSetHeaders(t *testing.T) {
	config := ClientConfig{
		DefaultHeaders: map[string]string{
			"X-Client-Version": "1.0",
		},
	}

	client := NewClient[testData](config)

	req, _ := http.NewRequest("GET", "https://api.example.com", nil)

	requestConfig := RequestConfig{
		Headers: map[string]string{
			"X-Custom": "value",
		},
	}

	client.setHeaders(req, requestConfig, "application/json")

	if req.Header.Get("X-Client-Version") != "1.0" {
		t.Error("Header di default non impostato")
	}

	if req.Header.Get("X-Custom") != "value" {
		t.Error("Header custom non impostato")
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type non impostato")
	}
}

func TestSetAuthenticationBearer(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)

	config := RequestConfig{
		BearerToken: "test-token",
	}

	err := client.setAuthentication(req, config)

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expected := "Bearer test-token"

	if authHeader != expected {
		t.Errorf("Authorization atteso %s, ricevuto %s", expected, authHeader)
	}
}

func TestSetAuthenticationBasic(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)

	config := RequestConfig{
		BasicAuth: &BasicAuth{
			Username: "user",
			Password: "pass",
		},
	}

	err := client.setAuthentication(req, config)

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	username, password, ok := req.BasicAuth()

	if !ok {
		t.Error("Basic auth non impostato")
	}

	if username != "user" || password != "pass" {
		t.Error("Credenziali basic auth non corrette")
	}
}

func TestSetAuthenticationCustom(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)

	customAuth := &mockAuthProvider{
		setAuthFunc: func(r *http.Request) error {
			r.Header.Set("X-API-Key", "custom-key")
			return nil
		},
	}

	config := RequestConfig{
		CustomAuth: customAuth,
	}

	err := client.setAuthentication(req, config)

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	if req.Header.Get("X-API-Key") != "custom-key" {
		t.Error("Custom auth non applicato correttamente")
	}
}

func TestCacheHit(t *testing.T) {
	client := NewClient[testData](ClientConfig{})

	cachedData := testData{ID: 1, Name: "Cached"}
	cache := newMockCache[testData]()
	cache.Set("test-key", cachedData, 5*time.Minute)

	client.SetCacheProvider(cache)

	config := RequestConfig{
		EnableCache: true,
		CacheKey:    "test-key",
	}

	response, err := client.Get(context.Background(), "https://api.example.com/users", config)

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !response.Cached {
		t.Error("Risposta dovrebbe essere dalla cache")
	}

	if response.Data.ID != 1 {
		t.Error("Dati dalla cache non corretti")
	}
}

func TestCacheMiss(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := testData{ID: 2, Name: "Fresh"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	cache := newMockCache[testData]()
	client.SetCacheProvider(cache)

	config := RequestConfig{
		EnableCache: true,
		CacheKey:    "test-key",
		CacheTTL:    5 * time.Minute,
	}

	response, err := client.Get(context.Background(), server.URL, config)

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Cached {
		t.Error("Risposta non dovrebbe essere dalla cache")
	}

	if response.Data.ID != 2 {
		t.Error("Dati non corretti")
	}

	cachedData, found := cache.Get("test-key")
	if !found {
		t.Error("Dati non salvati in cache")
	}

	if cachedData.ID != 2 {
		t.Error("Dati salvati in cache non corretti")
	}
}

func TestRateLimiting(t *testing.T) {
	client := NewClient[testData](ClientConfig{})

	rateLimiter := &mockRateLimiter{
		allowFunc: func(key string) bool {
			return false
		},
	}

	client.SetRateLimiter(rateLimiter)

	_, err := client.Get(context.Background(), "https://api.example.com/users", RequestConfig{})

	if err == nil {
		t.Error("Errore atteso per rate limit exceeded")
	}

	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Errore dovrebbe contenere 'rate limit exceeded', ricevuto: %s", err.Error())
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	client := NewClient[testData](ClientConfig{})

	circuitBreaker := &mockCircuitBreaker{
		executeFunc: func(fn func() error) error {
			return errors.New("circuit breaker is open")
		},
	}

	client.SetCircuitBreaker(circuitBreaker)

	config := RequestConfig{
		EnableCircuitBreaker: true,
	}

	_, err := client.Get(context.Background(), "https://api.example.com/users", config)

	if err == nil {
		t.Error("Errore atteso quando circuit breaker è open")
	}

	if !strings.Contains(err.Error(), "circuit breaker is open") {
		t.Errorf("Errore dovrebbe contenere 'circuit breaker is open', ricevuto: %s", err.Error())
	}
}

func TestCircuitBreakerClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := testData{ID: 1, Name: "Success"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	circuitBreaker := &mockCircuitBreaker{
		executeFunc: func(fn func() error) error {
			return fn()
		},
	}

	client.SetCircuitBreaker(circuitBreaker)

	config := RequestConfig{
		EnableCircuitBreaker: true,
	}

	response, err := client.Get(context.Background(), server.URL, config)

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.ID != 1 {
		t.Error("Risposta non corretta con circuit breaker closed")
	}
}

func TestMetricsRecording(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	metrics := &mockMetricsCollector{}
	client.SetMetricsCollector(metrics)

	config := RequestConfig{
		EnableMetrics: true,
	}

	client.Get(context.Background(), server.URL, config)

	if !metrics.recordRequestCalled {
		t.Error("RecordRequest non chiamato")
	}
}

func TestMiddleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Middleware") != "applied" {
			t.Error("Middleware non applicato")
		}

		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	middleware := &mockMiddleware{
		processFunc: func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
			req.Header.Set("X-Middleware", "applied")
			return next(req)
		},
	}

	config := ClientConfig{
		Middlewares: []Middleware{middleware},
	}

	client := NewClient[testData](config)

	_, err := client.Get(context.Background(), server.URL, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}
}

func TestMultipleMiddlewares(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-First") != "1" {
			t.Error("Primo middleware non applicato")
		}

		if r.Header.Get("X-Second") != "2" {
			t.Error("Secondo middleware non applicato")
		}

		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	middleware1 := &mockMiddleware{
		processFunc: func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
			req.Header.Set("X-First", "1")
			return next(req)
		},
	}

	middleware2 := &mockMiddleware{
		processFunc: func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
			req.Header.Set("X-Second", "2")
			return next(req)
		},
	}

	config := ClientConfig{
		Middlewares: []Middleware{middleware1, middleware2},
	}

	client := NewClient[testData](config)

	_, err := client.Get(context.Background(), server.URL, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}
}

func TestCustomEncoder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		if string(body) != "custom-encoded" {
			t.Errorf("Body atteso 'custom-encoded', ricevuto '%s'", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	encoder := &mockEncoder{
		canEncodeFunc: func(v any) bool {
			return true
		},
		encodeFunc: func(v any) ([]byte, string, error) {
			return []byte("custom-encoded"), "text/plain", nil
		},
	}

	client.SetEncoder(encoder)

	body := testData{ID: 1, Name: "Test"}

	_, err := client.Post(context.Background(), server.URL, body, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}
}

func TestCustomDecoder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("custom-data"))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	decoder := &mockDecoder[testData]{
		canDecodeFunc: func(contentType string) bool {
			return true
		},
		decodeFunc: func(data []byte, contentType string) (*testData, error) {
			return &testData{ID: 99, Name: "Custom Decoded"}, nil
		},
	}

	client.SetDecoder(decoder)

	response, err := client.Get(context.Background(), server.URL, RequestConfig{})

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.ID != 99 {
		t.Error("Decoder custom non applicato")
	}

	if response.Data.Name != "Custom Decoded" {
		t.Error("Dati decodificati non corretti")
	}
}

func TestEventHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	hook := &mockEventHook{}

	config := ClientConfig{
		EventHooks: []EventHook{hook},
	}

	client := NewClient[testData](config)

	client.Get(context.Background(), server.URL, RequestConfig{})

	if !hook.onRequestStartCalled {
		t.Error("OnRequestStart non chiamato")
	}

	if !hook.onRequestEndCalled {
		t.Error("OnRequestEnd non chiamato")
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		response := testData{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Get(ctx, server.URL, RequestConfig{})

	if err == nil {
		t.Error("Errore atteso per context cancellation")
	}
}

func TestClose(t *testing.T) {
	transport := &mockTransport{}

	config := ClientConfig{
		Transport: transport,
	}

	client := NewClient[testData](config)

	err := client.Close()

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	if !transport.closeCalled {
		t.Error("Transport.Close non chiamato")
	}
}

func TestBuildRequestError(t *testing.T) {
	client := NewClient[testData](ClientConfig{})

	invalidBody := make(chan int)

	_, err := client.buildRequest(context.Background(), MethodPost, "https://api.example.com", invalidBody, RequestConfig{})

	if err == nil {
		t.Error("Errore atteso per body non serializzabile")
	}
}

func TestSetRetryStrategy(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	strategy := &mockRetryStrategy{}

	client.SetRetryStrategy(strategy)

	if client.retry != strategy {
		t.Error("Retry strategy non impostata correttamente")
	}
}

func TestSetCircuitBreaker(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	breaker := &mockCircuitBreaker{}

	client.SetCircuitBreaker(breaker)

	if client.circuit != breaker {
		t.Error("Circuit breaker non impostato correttamente")
	}
}

func TestSetMetricsCollector(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	metrics := &mockMetricsCollector{}

	client.SetMetricsCollector(metrics)

	if client.metrics != metrics {
		t.Error("Metrics collector non impostato correttamente")
	}
}

func TestSetCacheProvider(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	cache := newMockCache[testData]()

	client.SetCacheProvider(cache)

	if client.cache != cache {
		t.Error("Cache provider non impostato correttamente")
	}
}

func TestSetTracer(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	tracer := &mockTracer{}

	client.SetTracer(tracer)

	if client.tracer != tracer {
		t.Error("Tracer non impostato correttamente")
	}
}

func TestSetDecoder(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	decoder := &mockDecoder[testData]{}

	client.SetDecoder(decoder)

	if client.decoder != decoder {
		t.Error("Decoder non impostato correttamente")
	}
}

func TestSetEncoder(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	encoder := &mockEncoder{}

	client.SetEncoder(encoder)

	if client.encoder != encoder {
		t.Error("Encoder non impostato correttamente")
	}
}

func TestSetRateLimiter(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	limiter := &mockRateLimiter{}

	client.SetRateLimiter(limiter)

	if client.rateLimiter != limiter {
		t.Error("Rate limiter non impostato correttamente")
	}
}

// Correggi TestHalfOpenFailureReOpensCircuit
func TestHalfOpenFailureReOpensCircuit(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      2,
		Interval:         60 * time.Second,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  5,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	// Accumula fallimenti per aprire il circuit
	for i := 0; i < 2; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != CircuitOpen {
		t.Error("Circuit dovrebbe essere open")
	}

	time.Sleep(150 * time.Millisecond)

	if cb.State() != CircuitHalfOpen {
		t.Error("Circuit dovrebbe essere half-open")
	}

	// In half-open, accumula abbastanza fallimenti per riaprire
	for i := 0; i < 2; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != CircuitOpen {
		t.Error("Circuit dovrebbe riaprirsi dopo i fallimenti in half-open")
	}
}

// Correggi TestRetrySuccess - fai in modo che il server generi un errore di connessione
func TestRetrySuccess(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if attempts < 3 {
			// Chiudi la connessione per generare un errore vero
			hj, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}

		response := testData{ID: 1, Name: "Success"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	retryStrategy := &mockRetryStrategy{
		maxAttempts: 3,
		shouldRetry: func(attempt int, err error, resp *http.Response) bool {
			return attempt < 3
		},
		getDelay: func(attempt int) time.Duration {
			return 10 * time.Millisecond
		},
	}

	client.SetRetryStrategy(retryStrategy)

	config := RequestConfig{
		EnableRetry: true,
	}

	response, err := client.Get(context.Background(), server.URL, config)

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if response.Data.ID != 1 {
		t.Error("Risposta non corretta dopo retry")
	}

	if attempts != 3 {
		t.Errorf("Tentativi attesi 3, ricevuti %d", attempts)
	}
}

type mockAuthProvider struct {
	setAuthFunc func(*http.Request) error
}

func (m *mockAuthProvider) SetAuth(req *http.Request) error {
	if m.setAuthFunc != nil {
		return m.setAuthFunc(req)
	}
	return nil
}

func (m *mockAuthProvider) Name() string {
	return "mock-auth"
}

type mockCache[T any] struct {
	data map[string]*T
}

func newMockCache[T any]() *mockCache[T] {
	return &mockCache[T]{
		data: make(map[string]*T),
	}
}

func (m *mockCache[T]) Get(key string) (*T, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *mockCache[T]) Set(key string, value T, ttl time.Duration) error {
	m.data[key] = &value
	return nil
}

func (m *mockCache[T]) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache[T]) Clear() error {
	m.data = make(map[string]*T)
	return nil
}

type mockRateLimiter struct {
	allowFunc func(string) bool
}

func (m *mockRateLimiter) Allow(key string) bool {
	if m.allowFunc != nil {
		return m.allowFunc(key)
	}
	return true
}

func (m *mockRateLimiter) Wait(ctx context.Context, key string) error {
	return nil
}

func (m *mockRateLimiter) Reset(key string) {}

type mockRetryStrategy struct {
	maxAttempts int
	shouldRetry func(int, error, *http.Response) bool
	getDelay    func(int) time.Duration
}

func (m *mockRetryStrategy) MaxAttempts() int {
	return m.maxAttempts
}

func (m *mockRetryStrategy) ShouldRetry(attempt int, err error, resp *http.Response) bool {
	if m.shouldRetry != nil {
		return m.shouldRetry(attempt, err, resp)
	}
	return false
}

func (m *mockRetryStrategy) GetDelay(attempt int) time.Duration {
	if m.getDelay != nil {
		return m.getDelay(attempt)
	}
	return 0
}

type mockCircuitBreaker struct {
	executeFunc func(func() error) error
}

func (m *mockCircuitBreaker) Execute(fn func() error) error {
	if m.executeFunc != nil {
		return m.executeFunc(fn)
	}
	return fn()
}

func (m *mockCircuitBreaker) State() CircuitState {
	return CircuitClosed
}

func (m *mockCircuitBreaker) Reset() {}

type mockMetricsCollector struct {
	recordRequestCalled bool
}

func (m *mockMetricsCollector) RecordRequest(method, path string, duration time.Duration, statusCode int) {
	m.recordRequestCalled = true
}

func (m *mockMetricsCollector) RecordError(method, path, errorType string) {}

func (m *mockMetricsCollector) RecordRetryAttempt(method, path string, attempt int) {}

func (m *mockMetricsCollector) RecordCircuitBreakerState(endpoint string, state CircuitState) {}

type mockMiddleware struct {
	processFunc func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error)
}

func (m *mockMiddleware) Process(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	if m.processFunc != nil {
		return m.processFunc(req, next)
	}
	return next(req)
}

func (m *mockMiddleware) Name() string {
	return "mock-middleware"
}

type mockEncoder struct {
	canEncodeFunc func(any) bool
	encodeFunc    func(any) ([]byte, string, error)
}

func (m *mockEncoder) CanEncode(v any) bool {
	if m.canEncodeFunc != nil {
		return m.canEncodeFunc(v)
	}
	return false
}

func (m *mockEncoder) Encode(v any) ([]byte, string, error) {
	if m.encodeFunc != nil {
		return m.encodeFunc(v)
	}
	return nil, "", errors.New("not implemented")
}

type mockDecoder[T any] struct {
	canDecodeFunc func(string) bool
	decodeFunc    func([]byte, string) (*T, error)
}

func (m *mockDecoder[T]) CanDecode(contentType string) bool {
	if m.canDecodeFunc != nil {
		return m.canDecodeFunc(contentType)
	}
	return false
}

func (m *mockDecoder[T]) Decode(data []byte, contentType string) (*T, error) {
	if m.decodeFunc != nil {
		return m.decodeFunc(data, contentType)
	}
	return nil, errors.New("not implemented")
}

type mockEventHook struct {
	onRequestStartCalled bool
	onRequestEndCalled   bool
}

func (m *mockEventHook) OnRequestStart(ctx context.Context, req *http.Request) {
	m.onRequestStartCalled = true
}

func (m *mockEventHook) OnRequestEnd(ctx context.Context, req *http.Request, resp *http.Response, err error) {
	m.onRequestEndCalled = true
}

func (m *mockEventHook) OnRetryAttempt(ctx context.Context, req *http.Request, attempt int) {}

func (m *mockEventHook) OnCircuitBreakerStateChange(endpoint string, from, to CircuitState) {}

type mockTransport struct {
	closeCalled bool
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTransport) Close() error {
	m.closeCalled = true
	return nil
}

type mockTracer struct{}

func (m *mockTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &mockSpan{}
}

type mockSpan struct{}

func (m *mockSpan) End()                                              {}
func (m *mockSpan) SetAttribute(key string, value any)                {}
func (m *mockSpan) RecordError(err error)                             {}
func (m *mockSpan) SetStatus(code SpanStatusCode, description string) {}
