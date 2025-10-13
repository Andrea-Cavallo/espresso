package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

// User rappresenta un utente di esempio
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Post rappresenta un post di esempio
type Post struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// APIResponse rappresenta una risposta API generica
type APIResponse[T any] struct {
	Data    T      `json:"data"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorResponse rappresenta una risposta di errore
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details"`
}

func main() {
	ctx := context.Background()

	// Esempi base
	basicClientExample(ctx)
	requestBuilderExample(ctx)

	// Esempi con funzionalità avanzate
	retryClientExample(ctx)
	circuitBreakerExample(ctx)
	authenticationExample(ctx)
	cachingExample(ctx)

	// Esempi di configurazione avanzata
	resilientClientExample(ctx)
	customConfigurationExample(ctx)

	// Esempi di gestione errori
	errorHandlingExample(ctx)

	// Esempi aggiuntivi
	cloneBuilderExample(ctx)
	batchRequestExample(ctx)

	fmt.Println("\n=== Examples completed ===")
}

// basicClientExample dimostra l'uso base del client
func basicClientExample(ctx context.Context) {
	fmt.Println("--- Basic Client Example ---")

	// Client semplice con configurazione di default
	client := espresso.NewDefaultClient[User]()

	// Richiesta GET semplice
	response, err := client.Get(ctx, "https://jsonplaceholder.typicode.com/users/1", espresso.RequestConfig{})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("User: %+v\n", *response.Data)
	}
	fmt.Printf("Status Code: %d\n", response.StatusCode)
	fmt.Printf("Response Time: %v\n\n", response.Duration)
}

// requestBuilderExample dimostra l'uso del request builder
func requestBuilderExample(ctx context.Context) {
	fmt.Println("--- Request Builder Example ---")

	client := espresso.NewDefaultClient[[]Post]()

	// Utilizzo del request builder per costruire richieste complesse
	response, err := client.Request("https://jsonplaceholder.typicode.com/posts").
		WithHeader("Accept", "application/json").
		WithHeader("User-Agent", "Espresso-Example/1.0").
		WithQuery("userId", "1").
		WithTimeout(10 * time.Second).
		EnableMetrics().
		Get(ctx)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("Found %d posts\n", len(*response.Data))
		if len(*response.Data) > 0 {
			fmt.Printf("First post: %+v\n\n", (*response.Data)[0])
		}
	}
}

// retryClientExample dimostra l'uso del retry
func retryClientExample(ctx context.Context) {
	fmt.Println("--- Retry Client Example ---")

	// Client con retry automatico
	client := espresso.NewRetryClient[User](3)

	// Richiesta che potrebbe fallire e venire ritentata
	response, err := client.Request("https://jsonplaceholder.typicode.com/users/1").
		WithTimeout(5 * time.Second).
		EnableRetry().
		Get(ctx)

	if err != nil {
		log.Printf("Error after retries: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("User retrieved with retry: %+v\n\n", *response.Data)
	}
}

// circuitBreakerExample dimostra l'uso del circuit breaker
func circuitBreakerExample(ctx context.Context) {
	fmt.Println("--- Circuit Breaker Example ---")

	// Client con circuit breaker
	client := espresso.NewClientBuilder[User]().
		WithTimeout(30 * time.Second).
		WithDefaultCircuitBreaker().
		WithMetrics().
		Build()

	// Simulazione di richieste multiple per testare il circuit breaker
	for i := 0; i < 5; i++ {
		response, err := client.Request("https://jsonplaceholder.typicode.com/users/1").
			EnableCircuitBreaker().
			EnableMetrics().
			Get(ctx)

		if err != nil {
			log.Printf("Request %d failed: %v", i+1, err)
			continue
		}

		if response.Data != nil {
			fmt.Printf("Request %d succeeded: User ID %d\n", i+1, response.Data.ID)
		}
	}
	fmt.Println()
}

// authenticationExample dimostra diversi tipi di autenticazione
func authenticationExample(ctx context.Context) {
	fmt.Println("--- Authentication Example ---")

	// Utilizzo del request builder per autenticazione per-richiesta
	publicClient := espresso.NewDefaultClient[User]()

	response, err := publicClient.Request("https://jsonplaceholder.typicode.com/users/1").
		WithBearerToken("request-specific-token").
		WithHeader("X-API-Version", "v1").
		Get(ctx)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("Authenticated request succeeded: %+v\n\n", *response.Data)
	}
}

// cachingExample dimostra l'uso del caching
func cachingExample(ctx context.Context) {
	fmt.Println("--- Caching Example ---")

	client := espresso.NewClientBuilder[User]().
		WithCache().
		WithTimeout(30 * time.Second).
		Build()

	// Prima richiesta (verrà cachata)
	start := time.Now()
	response1, err := client.Request("https://jsonplaceholder.typicode.com/users/1").
		WithCache("user-1", 5*time.Minute).
		Get(ctx)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("First request (cached): %v\n", time.Since(start))

	// Seconda richiesta (dovrebbe usare la cache)
	start = time.Now()
	response2, err := client.Request("https://jsonplaceholder.typicode.com/users/1").
		WithCache("user-1", 5*time.Minute).
		Get(ctx)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Second request (from cache): %v\n", time.Since(start))

	if response1.Data != nil && response2.Data != nil {
		fmt.Printf("Same data: %t\n\n", response1.Data.ID == response2.Data.ID)
	}
}

// resilientClientExample dimostra un client con tutte le funzionalità
func resilientClientExample(ctx context.Context) {
	fmt.Println("--- Resilient Client Example ---")

	// Client con tutte le funzionalità di resilienza
	client := espresso.NewResilientClient[User]()

	// Richiesta con tutte le funzionalità abilitate
	response, err := client.Request("https://jsonplaceholder.typicode.com/users/1").
		EnableAll().
		WithTimeout(15 * time.Second).
		WithCacheKey("resilient-user-1").
		WithCacheTTL(2 * time.Minute).
		Get(ctx)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("Resilient request succeeded: %+v\n\n", *response.Data)
	}
}

// customConfigurationExample dimostra configurazioni personalizzate
func customConfigurationExample(ctx context.Context) {
	fmt.Println("--- Custom Configuration Example ---")

	// Configurazione retry personalizzata
	customRetryConfig := espresso.RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       200 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		JitterEnabled:   true,
		JitterType:      espresso.JitterFull,
		BackoffStrategy: espresso.BackoffExponential,
		RetriableStatus: []int{429, 500, 502, 503, 504},
	}

	// Configurazione circuit breaker personalizzata
	customCircuitConfig := espresso.CircuitBreakerConfig{
		MaxRequests:      15,
		Interval:         120 * time.Second,
		Timeout:          20 * time.Second,
		FailureThreshold: 8,
		SuccessThreshold: 4,
		HalfOpenMaxReqs:  8,
	}

	// Client con configurazioni personalizzate
	client := espresso.NewClientBuilder[Post]().
		WithTimeout(45 * time.Second).
		WithRetry(customRetryConfig).
		WithCircuitBreaker(customCircuitConfig).
		WithDefaultHeaders(map[string]string{
			"Accept":     "application/json",
			"User-Agent": "Custom-Espresso-Client/2.0",
		}).
		WithMetrics().
		WithTracing().
		Build()

	// Creazione di un post
	newPost := Post{
		UserID: 1,
		Title:  "Example Post",
		Body:   "This is an example post created with Espresso client",
	}

	response, err := client.Request("https://jsonplaceholder.typicode.com/posts").
		WithBody(newPost).
		WithContentType("application/json").
		WithRetryConfig(&customRetryConfig).
		Post(ctx)

	if err != nil {
		log.Printf("Error creating post: %v", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("Created post: %+v\n\n", *response.Data)
	}
}

// errorHandlingExample dimostra la gestione degli errori
func errorHandlingExample(ctx context.Context) {
	fmt.Println("--- Error Handling Example ---")

	client := espresso.NewDefaultClient[ErrorResponse]()

	// Richiesta a un endpoint che non esiste
	response, err := client.Request("https://jsonplaceholder.typicode.com/nonexistent").
		WithTimeout(5 * time.Second).
		Get(ctx)

	if err != nil {
		fmt.Printf("Expected error occurred: %v\n", err)
	}

	if response != nil {
		fmt.Printf("Response status: %d\n", response.StatusCode)
		if response.StatusCode >= 400 && response.Data != nil {
			fmt.Printf("Error response: %+v\n", *response.Data)
		}
	}

	// Esempio di timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	_, err = client.Request("https://jsonplaceholder.typicode.com/users/1").
		Get(timeoutCtx)

	if err != nil {
		fmt.Printf("Timeout error: %v\n", err)
	}

	fmt.Println()
}

// cloneBuilderExample dimostra la clonazione di builder
func cloneBuilderExample(ctx context.Context) {
	fmt.Println("--- Clone Builder Example ---")

	client := espresso.NewDefaultClient[User]()

	// Builder base con configurazione comune
	baseBuilder := client.Request("https://jsonplaceholder.typicode.com").
		WithHeader("Accept", "application/json").
		WithHeader("User-Agent", "Espresso-Clone-Example/1.0").
		WithTimeout(10 * time.Second).
		EnableMetrics()

	// Clone per richieste specifiche
	userBuilder := baseBuilder.Clone()
	response1, err := userBuilder.
		WithQuery("id", "1").
		Get(ctx)

	if err != nil {
		log.Printf("User request error: %v", err)
		return
	}

	if response1.Data != nil {
		fmt.Printf("User from cloned builder: %+v\n", *response1.Data)
	}

	// Altro clone per richieste diverse
	adminBuilder := baseBuilder.Clone()
	response2, err := adminBuilder.
		WithBearerToken("admin-token").
		WithQuery("role", "admin").
		Get(ctx)

	if err != nil {
		log.Printf("Admin request error: %v", err)
		return
	}

	if response2.Data != nil {
		fmt.Printf("Admin user: %+v\n\n", *response2.Data)
	}
}

// batchRequestExample dimostra richieste multiple
func batchRequestExample(ctx context.Context) {
	fmt.Println("--- Batch Request Example ---")

	client := espresso.NewFastClient[User]()

	userIDs := []string{"1", "2", "3", "4", "5"}
	results := make(chan struct {
		ID   string
		User *User
		Err  error
	}, len(userIDs))

	// Richieste parallele
	for _, id := range userIDs {
		go func(userID string) {
			response, err := client.Request("https://jsonplaceholder.typicode.com/users/" + userID).
				WithTimeout(5 * time.Second).
				EnableMetrics().
				Get(ctx)

			result := struct {
				ID   string
				User *User
				Err  error
			}{
				ID:  userID,
				Err: err,
			}

			if err == nil && response.Data != nil {
				result.User = response.Data
			}

			results <- result
		}(id)
	}

	// Raccolta risultati
	for i := 0; i < len(userIDs); i++ {
		result := <-results
		if result.Err != nil {
			fmt.Printf("User %s failed: %v\n", result.ID, result.Err)
		} else {
			fmt.Printf("User %s: %s (%s)\n", result.ID, result.User.Name, result.User.Email)
		}
	}

	fmt.Println()
}
