package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

// User represents a simple user structure
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Example 1: Simple GET request
	simpleGet()

	// Example 2: GET with retry
	getWithRetry()

	// Example 3: POST request
	createUser()

	// Example 4: Using builder for complex configuration
	advancedClient()
}

// Example 1: Simple GET request - the easiest way
func simpleGet() {
	fmt.Println("=== Simple GET Request ===")

	// Create a default client
	client := espresso.NewDefaultClient[User]()

	// Make a simple GET request
	response, err := client.Request("https://jsonplaceholder.typicode.com/users/1").Get(context.Background())

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
		fmt.Printf("Response time: %v\n\n", response.Duration)
	}
}

// Example 2: GET with automatic retry
func getWithRetry() {
	fmt.Println("=== GET with Retry ===")

	// Create a client with automatic retry (3 attempts)
	client := espresso.NewRetryClient[User](3)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.Request("https://jsonplaceholder.typicode.com/users/2").
		EnableRetry().
		Get(ctx)

	if err != nil {
		log.Printf("Error after retries: %v\n", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("User: %s (ID: %d)\n", response.Data.Name, response.Data.ID)
		fmt.Printf("Attempts: %d\n\n", response.Attempts)
	}
}

// Example 3: POST request to create a user
func createUser() {
	fmt.Println("=== POST Request ===")

	client := espresso.NewDefaultClient[User]()

	newUser := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	response, err := client.Request("https://jsonplaceholder.typicode.com/users").
		WithBody(newUser).
		WithContentType("application/json").
		Post(context.Background())

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created user with ID: %d\n", response.StatusCode)
	fmt.Println()
}

// Example 4: Advanced client with multiple features
func advancedClient() {
	fmt.Println("=== Advanced Client ===")

	// Build a client with all features
	client := espresso.NewClientBuilder[User]().
		WithBaseURL("https://jsonplaceholder.typicode.com").
		WithTimeout(30 * time.Second).
		WithExponentialBackoff(3, 100*time.Millisecond, 5*time.Second).
		WithDefaultCircuitBreaker().
		WithCache().
		WithUserAgent("MyApp/1.0").
		Build()

	// Make a cached request
	response, err := client.Request("/users/1").
		WithCache("user-1", 5*time.Minute).
		Get(context.Background())

	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	if response.Data != nil {
		fmt.Printf("User: %s\n", response.Data.Name)
		fmt.Printf("Cached: %v\n", response.Cached)
	}
}
