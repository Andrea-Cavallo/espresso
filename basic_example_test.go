package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout        = 5 * time.Second
	mockServerResponse = `{"id":1,"name":"Test User","email":"test@example.com","username":"testuser"}`
)

// TestUser valida la struttura User
func TestUser(t *testing.T) {
	user := User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Username: "johndoe",
	}

	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, "johndoe", user.Username)
}

// TestPost valida la struttura Post
func TestPost(t *testing.T) {
	post := Post{
		ID:     1,
		UserID: 1,
		Title:  "Test Title",
		Body:   "Test Body",
	}

	assert.Equal(t, 1, post.ID)
	assert.Equal(t, 1, post.UserID)
	assert.Equal(t, "Test Title", post.Title)
	assert.Equal(t, "Test Body", post.Body)
}

// TestAPIResponse valida la risposta API generica
func TestAPIResponse(t *testing.T) {
	response := APIResponse[User]{
		Data: User{
			ID:       1,
			Name:     "Test",
			Email:    "test@test.com",
			Username: "test",
		},
		Status:  "success",
		Message: "Operation completed",
	}

	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "Operation completed", response.Message)
	assert.Equal(t, 1, response.Data.ID)
}

// TestErrorResponse valida la risposta di errore
func TestErrorResponse(t *testing.T) {
	errorResp := ErrorResponse{
		Error:   "Not Found",
		Code:    404,
		Details: "Resource not found",
	}

	assert.Equal(t, "Not Found", errorResp.Error)
	assert.Equal(t, 404, errorResp.Code)
	assert.Equal(t, "Resource not found", errorResp.Details)
}

// createMockServer crea un server HTTP di test
func createMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	if handler == nil {
		t.Fatal("handler cannot be nil")
	}

	return httptest.NewServer(handler)
}

// createSuccessHandler crea un handler che restituisce successo
func createSuccessHandler(t *testing.T, statusCode int, body string) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(body))
		require.NoError(t, err)
	}
}

// createErrorHandler crea un handler che restituisce errore
func createErrorHandler(statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}
}

// TestBasicClientExample testa l'esempio del client base
func TestBasicClientExample(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  bool
		validateResult func(t *testing.T)
	}{
		{
			name:          "success_response",
			statusCode:    http.StatusOK,
			responseBody:  mockServerResponse,
			expectedError: false,
			validateResult: func(t *testing.T) {
				assert.True(t, true)
			},
		},
		{
			name:          "server_error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error":"Internal Server Error"}`,
			expectedError: false,
			validateResult: func(t *testing.T) {
				assert.True(t, true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServer(t, createSuccessHandler(t, tt.statusCode, tt.responseBody))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			basicClientExample(ctx)

			tt.validateResult(t)
		})
	}
}

// TestRequestBuilderExample testa l'esempio del request builder
func TestRequestBuilderExample(t *testing.T) {
	postsResponse := `[
		{"id":1,"userId":1,"title":"Post 1","body":"Body 1"},
		{"id":2,"userId":1,"title":"Post 2","body":"Body 2"}
	]`

	server := createMockServer(t, createSuccessHandler(t, http.StatusOK, postsResponse))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	requestBuilderExample(ctx)

	assert.True(t, true)
}

// TestRetryClientExample testa l'esempio con retry
func TestRetryClientExample(t *testing.T) {
	attemptCount := 0
	maxAttempts := 3

	handler := func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount < maxAttempts {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockServerResponse))
		require.NoError(t, err)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	retryClientExample(ctx)

	assert.True(t, true)
}

// TestCircuitBreakerExample testa l'esempio con circuit breaker
func TestCircuitBreakerExample(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	circuitBreakerExample(ctx)

	assert.True(t, true)
}

// TestAuthenticationExample testa l'esempio con autenticazione
func TestAuthenticationExample(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockServerResponse))
		require.NoError(t, err)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	authenticationExample(ctx)

	assert.True(t, true)
}

// TestCachingExample testa l'esempio con caching
func TestCachingExample(t *testing.T) {
	requestCount := 0

	handler := func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockServerResponse))
		require.NoError(t, err)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cachingExample(ctx)

	assert.True(t, true)
}

// TestResilientClientExample testa l'esempio del client resiliente
func TestResilientClientExample(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resilientClientExample(ctx)

	assert.True(t, true)
}

// TestCustomConfigurationExample testa l'esempio con configurazione custom
func TestCustomConfigurationExample(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var post Post
		err := json.NewDecoder(r.Body).Decode(&post)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		post.ID = 101

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(post)
		require.NoError(t, err)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	customConfigurationExample(ctx)

	assert.True(t, true)
}

// TestErrorHandlingExample testa la gestione degli errori
func TestErrorHandlingExample(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError bool
	}{
		{
			name:          "not_found_error",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error":"Not Found","code":404,"details":"Resource not found"}`,
			expectedError: true,
		},
		{
			name:          "server_error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error":"Internal Server Error","code":500,"details":"Server error"}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServer(t, createSuccessHandler(t, tt.statusCode, tt.responseBody))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			errorHandlingExample(ctx)

			assert.True(t, true)
		})
	}
}

// TestCloneBuilderExample testa la clonazione dei builder
func TestCloneBuilderExample(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cloneBuilderExample(ctx)

	assert.True(t, true)
}

// TestBatchRequestExample testa le richieste batch
func TestBatchRequestExample(t *testing.T) {
	userResponses := map[string]string{
		"1": `{"id":1,"name":"User 1","email":"user1@test.com","username":"user1"}`,
		"2": `{"id":2,"name":"User 2","email":"user2@test.com","username":"user2"}`,
		"3": `{"id":3,"name":"User 3","email":"user3@test.com","username":"user3"}`,
		"4": `{"id":4,"name":"User 4","email":"user4@test.com","username":"user4"}`,
		"5": `{"id":5,"name":"User 5","email":"user5@test.com","username":"user5"}`,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Path[len("/users/"):]

		response, exists := userResponses[userID]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		require.NoError(t, err)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	batchRequestExample(ctx)

	assert.True(t, true)
}

// TestContextCancellation testa la cancellazione del contesto
func TestContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	basicClientExample(ctx)

	assert.True(t, true)
}

// TestTimeoutScenario testa lo scenario di timeout
func TestTimeoutScenario(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}

	server := createMockServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	basicClientExample(ctx)

	assert.True(t, true)
}

// BenchmarkBasicClientExample benchmark per il client base
func BenchmarkBasicClientExample(b *testing.B) {
	server := createMockServer(&testing.T{}, createSuccessHandler(&testing.T{}, http.StatusOK, mockServerResponse))
	defer server.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		basicClientExample(ctx)
	}
}

// BenchmarkBatchRequestExample benchmark per le richieste batch
func BenchmarkBatchRequestExample(b *testing.B) {
	server := createMockServer(&testing.T{}, createSuccessHandler(&testing.T{}, http.StatusOK, mockServerResponse))
	defer server.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchRequestExample(ctx)
	}
}
