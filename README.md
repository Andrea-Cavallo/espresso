# Espresso

<div align="center">

 **A modern, type-safe HTTP client library for Go with advanced orchestration patterns**
![espresso](./docs/espresso.png)


[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[Features](#-features) •
[Installation](#-installation) •
[Quick Start](#-quick-start) •
[Documentation](#-documentation) •
[Examples](#-examples)

</div>

---

##  Features

- **Type-Safe**: Full generic support for compile-time type safety
- **Retry Logic**: Configurable retry strategies with exponential backoff and jitter
- **Circuit Breaker**: Automatic failure detection and recovery
- **Smart Caching**: Built-in caching with configurable TTL
- **Metrics**: Request tracking and performance monitoring
- **Rate Limiting**: Protect your APIs from overload
- **Authentication**: Support for Bearer, Basic, and custom authentication
- **Middleware**: Extensible middleware chain
- **Orchestration**: Complex API workflows made simple
- **Advanced Patterns**: Chain, Fan-Out, Saga, Batch, and Conditional patterns
- **Context Support**: Full context propagation and timeout handling
- **Response Switching**: Conditional branching based on response data

##  Installation

```bash
go get github.com/Andrea-Cavallo/espresso
```

**Requirements**: Go 1.21 or higher

##  Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/Andrea-Cavallo/espresso"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    ctx := context.Background()

    // Create a client with default configuration
    client := espresso.NewClient[User](espresso.ClientConfig{
        BaseURL: "https://api.example.com",
        Timeout: 30 * time.Second,
    })

    // Execute a simple GET request
    response, err := client.Request("/users/123").
        WithBearerToken("your-token-here").
        EnableRetry().
        Get(ctx)

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %+v\n", response.Data)
    fmt.Printf("Status Code: %d\n", response.StatusCode)
    fmt.Printf("Duration: %v\n", response.Duration)
}
```

##  Documentation

### Table of Contents

- [Client Creation](#client-creation)
- [Making Requests](#making-requests)
- [Authentication](#authentication)
- [Retry & Circuit Breaker](#retry--circuit-breaker)
- [Caching](#caching)
- [Orchestration](#orchestration)
- [Advanced Patterns](#advanced-patterns)
- [Response Switching](#response-switching)
- [Error Handling](#error-handling)

---

### Client Creation

#### Default Client

```go
// Basic client
client := espresso.NewClient[User](espresso.ClientConfig{
    BaseURL: "https://api.example.com",
    Timeout: 30 * time.Second,
})
```

#### Advanced Configuration

```go
client := espresso.NewClient[User](espresso.ClientConfig{
    BaseURL: "https://api.example.com",
    Timeout: 30 * time.Second,
    DefaultHeaders: map[string]string{
        "User-Agent": "MyApp/1.0",
        "Accept":     "application/json",
    },
    Transport: customTransport,
})

// Configure retry strategy
client.SetRetryStrategy(&espresso.ExponentialBackoffRetry{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    10 * time.Second,
})

// Configure circuit breaker
client.SetCircuitBreaker(espresso.NewCircuitBreaker("api",
    espresso.DefaultCircuitBreakerConfig()))

// Enable metrics
client.SetMetricsCollector(metricsCollector)

// Enable caching
client.SetCacheProvider(cacheProvider)
```

---

### Making Requests

#### GET Request

```go
response, err := client.Request("/users/123").
    WithHeader("X-Custom-Header", "value").
    WithQuery("include", "profile").
    Get(ctx)
```

#### POST Request

```go
userData := User{
    Name:  "John Doe",
    Email: "john@example.com",
}

response, err := client.Request("/users").
    WithBody(userData).
    WithContentType("application/json").
    Post(ctx)
```

#### PUT Request

```go
response, err := client.Request("/users/123").
    WithBody(updatedUser).
    Put(ctx)
```

#### DELETE Request

```go
response, err := client.Request("/users/123").
    Delete(ctx)
```

#### PATCH Request

```go
patch := map[string]any{
    "name": "Jane Doe",
}

response, err := client.Request("/users/123").
    WithBody(patch).
    Patch(ctx)
```

---

### Authentication

#### Bearer Token

```go
response, err := client.Request("/api/protected").
    WithBearerToken("eyJhbGciOiJIUzI1NiIs...").
    Get(ctx)
```

#### Basic Authentication

```go
response, err := client.Request("/api/protected").
    WithBasicAuth("username", "password").
    Get(ctx)
```

#### Custom Authentication

```go
type APIKeyAuth struct {
    APIKey string
}

func (a *APIKeyAuth) SetAuth(req *http.Request) error {
    req.Header.Set("X-API-Key", a.APIKey)
    return nil
}

func (a *APIKeyAuth) Name() string {
    return "api-key"
}

response, err := client.Request("/api/protected").
    WithCustomAuth(&APIKeyAuth{APIKey: "your-api-key"}).
    Get(ctx)
```

---

### Retry & Circuit Breaker

#### Automatic Retry

```go
response, err := client.Request("/api/endpoint").
    WithRetryConfig(&espresso.RetryConfig{
        MaxAttempts:     5,
        BaseDelay:       100 * time.Millisecond,
        MaxDelay:        10 * time.Second,
        BackoffStrategy: espresso.BackoffExponential,
        JitterEnabled:   true,
        RetriableStatus: []int{429, 500, 502, 503, 504},
    }).
    Get(ctx)
```

#### Circuit Breaker

```go
response, err := client.Request("/api/endpoint").
    WithCircuitBreakerConfig(&espresso.CircuitBreakerConfig{
        MaxRequests:      10,
        Interval:         60 * time.Second,
        Timeout:          10 * time.Second,
        FailureThreshold: 5,
        SuccessThreshold: 3,
        HalfOpenMaxReqs:  5,
    }).
    Get(ctx)
```

#### Combining Retry and Circuit Breaker

```go
response, err := client.Request("/api/endpoint").
    EnableRetry().
    EnableCircuitBreaker().
    Get(ctx)
```

---

### Caching

#### Simple Cache

```go
response, err := client.Request("/api/users/123").
    WithCache("user:123", 5 * time.Minute).
    Get(ctx)

if response.Cached {
    fmt.Println("Response served from cache")
}
```

#### Cache with Custom Key

```go
cacheKey := fmt.Sprintf("user:%d:profile", userID)

response, err := client.Request("/api/users/%d/profile", userID).
    WithCacheKey(cacheKey).
    WithCacheTTL(10 * time.Minute).
    EnableCache().
    Get(ctx)
```

---

### Orchestration

Orchestration allows you to build complex API workflows with multiple steps, conditional execution, and error handling.

#### Sequential Steps

```go
result, err := client.Orchestrate().
    Step("fetch-user", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
        ctx["user_name"] = response.Data.Name
        return nil
    }).
    End().
    Step("fetch-orders", func(ctx map[string]any) espresso.RequestBuilder[Order] {
        userName := ctx["user_name"].(string)
        return client.Request("/orders").
            WithQuery("user", userName)
    }).
    End().
    Execute(context.Background())
```

#### Parallel Execution

```go
result, err := client.Orchestrate().
    Parallel(). // Enable parallel execution
    Step("task1", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/task1")
    }).
    End().
    Step("task2", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/task2")
    }).
    End().
    Step("task3", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/task3")
    }).
    End().
    Execute(context.Background())
```

#### Conditional Steps

```go
result, err := client.Orchestrate().
    WithContext(map[string]any{
        "user_type": "premium",
    }).
    Step("check-premium", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/user/profile")
    }).
    When(func(ctx map[string]any) bool {
        return ctx["user_type"] == "premium"
    }).
    End().
    Execute(context.Background())
```

#### Optional Steps

```go
result, err := client.Orchestrate().
    Step("required-step", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/required")
    }).
    Required(true). // Failure stops orchestration
    End().
    Step("optional-step", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/optional")
    }).
    Optional(). // Failure doesn't stop orchestration
    End().
    Execute(context.Background())
```

#### Context Sharing

```go
result, err := client.Orchestrate().
    Step("step1", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    SaveToContext("user"). // Save response to context
    End().
    Step("step2", func(ctx map[string]any) espresso.RequestBuilder[Order] {
        user := ctx["user"].(User)
        return client.Request("/orders").
            WithQuery("user_id", user.ID)
    }).
    End().
    Execute(context.Background())
```

---

### Advanced Patterns

#### Chain Pattern

Execute API calls in sequence where each call depends on the previous one.

```go
result, err := client.Patterns().Chain().
    Call("authenticate", func(ctx map[string]any) espresso.RequestBuilder[AuthResponse] {
        return client.Request("/auth/login").
            WithBody(credentials)
    }).
    OnSuccess(func(response *espresso.Response[AuthResponse], ctx map[string]any) error {
        ctx["token"] = response.Data.Token
        return nil
    }).
    Call("fetch-profile", func(ctx map[string]any) espresso.RequestBuilder[User] {
        token := ctx["token"].(string)
        return client.Request("/user/profile").
            WithBearerToken(token)
    }).
    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
        ctx["user_id"] = response.Data.ID
        return nil
    }).
    Execute(context.Background())
```

#### Fan-Out Pattern

Execute multiple API calls in parallel and aggregate results.

```go
result, err := client.Patterns().FanOut().
    Add("user_data", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    Add("user_orders", func(ctx map[string]any) espresso.RequestBuilder[Order] {
        return client.Request("/orders?user_id=123")
    }).
    Add("user_reviews", func(ctx map[string]any) espresso.RequestBuilder[Review] {
        return client.Request("/reviews?user_id=123")
    }).
    WithAggregator(func(results []espresso.StepResult, ctx map[string]any) (any, error) {
        // Aggregate results as needed
        userData := results[0].Data.(User)
        userOrders := results[1].Data.([]Order)
        userReviews := results[2].Data.([]Review)

        return UserProfile{
            User:    userData,
            Orders:  userOrders,
            Reviews: userReviews,
        }, nil
    }).
    Execute(context.Background())
```

#### Saga Pattern

Handle distributed transactions with automatic compensation on failure.

```go
result, err := client.Patterns().Saga().
    Step("reserve-inventory",
        func(ctx map[string]any) espresso.RequestBuilder[Inventory] {
            return client.Request("/inventory/reserve").
                WithBody(reservationData)
        },
        // Compensation function
        func(ctx map[string]any) error {
            inventoryID := ctx["inventory_id"].(string)
            _, err := client.Request("/inventory/release/"+inventoryID).
                Delete(context.Background())
            return err
        }).
    Step("create-order",
        func(ctx map[string]any) espresso.RequestBuilder[Order] {
            return client.Request("/orders").
                WithBody(orderData)
        },
        func(ctx map[string]any) error {
            orderID := ctx["order_id"].(string)
            _, err := client.Request("/orders/"+orderID).
                Delete(context.Background())
            return err
        }).
    Step("process-payment",
        func(ctx map[string]any) espresso.RequestBuilder[Payment] {
            return client.Request("/payments").
                WithBody(paymentData)
        },
        func(ctx map[string]any) error {
            paymentID := ctx["payment_id"].(string)
            _, err := client.Request("/payments/"+paymentID+"/refund").
                Post(context.Background())
            return err
        }).
    Execute(context.Background())
```

#### Batch Pattern

Process batches of similar items efficiently.

```go
products := []any{
    Product{ID: 1},
    Product{ID: 2},
    Product{ID: 3},
    Product{ID: 4},
    Product{ID: 5},
}

result, err := client.Patterns().Batch().
    WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[Product] {
        product := item.(Product)
        return client.Request(fmt.Sprintf("/products/%d", product.ID))
    }).
    WithItems(products).
    WithBatchSize(10). // Process 10 items at a time
    WithConcurrency(3). // Use 3 concurrent workers
    Execute(context.Background())
```

#### Conditional Pattern

Execute conditional logic based on context or response data.

```go
result, err := client.Patterns().Conditional().
    WithContext(map[string]any{
        "user_type": "premium",
        "amount": 150.0,
    }).
    When(
        func(ctx map[string]any) bool {
            return ctx["user_type"] == "premium"
        },
        "apply-premium-discount",
        func(ctx map[string]any) espresso.RequestBuilder[Discount] {
            return client.Request("/discounts/premium")
        }).
    ElseWhen(
        func(ctx map[string]any) bool {
            amount := ctx["amount"].(float64)
            return amount > 100
        },
        "apply-bulk-discount",
        func(ctx map[string]any) espresso.RequestBuilder[Discount] {
            return client.Request("/discounts/bulk")
        }).
    Else("standard-pricing",
        func(ctx map[string]any) espresso.RequestBuilder[Price] {
            return client.Request("/pricing/standard")
        }).
    Execute(context.Background())
```

---

### Response Switching

Create branching logic based on response data.

#### Basic Response Switch

```go
result, err := client.Orchestrate().
    Step("check-status", func(ctx map[string]any) espresso.RequestBuilder[Status] {
        return client.Request("/status")
    }).
    ResponseSwitch("status_response").
    OnValue("Status", "active", "process-active", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/process/active")
    }).
    OnValue("Status", "pending", "process-pending", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/process/pending")
    }).
    Default("process-unknown", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/process/unknown")
    }).
    End().
    Execute(context.Background())
```

#### Status Code Switching

```go
result, err := client.Orchestrate().
    Step("create-resource", func(ctx map[string]any) espresso.RequestBuilder[Resource] {
        return client.Request("/resources").WithBody(data)
    }).
    ResponseSwitch("create_response").
    OnStatus(201, "handle-created", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/webhook/created")
    }).
    OnStatus(409, "handle-conflict", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/resources/update")
    }).
    End().
    Execute(context.Background())
```

#### Custom Conditions

```go
result, err := client.Orchestrate().
    Step("fetch-data", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/data")
    }).
    ResponseSwitch("data_response").
    OnCustom(func(responseData any) bool {
        if response, ok := responseData.(*espresso.Response[Data]); ok {
            return response.Data.Count > 100
        }
        return false
    }, "process-large-dataset", func(ctx map[string]any) espresso.RequestBuilder[Result] {
        return client.Request("/process/large")
    }).
    OnCustom(func(responseData any) bool {
        if response, ok := responseData.(*espresso.Response[Data]); ok {
            return response.Data.Count <= 100
        }
        return false
    }, "process-small-dataset", func(ctx map[string]any) espresso.RequestBuilder[Result] {
        return client.Request("/process/small")
    }).
    End().
    Execute(context.Background())
```

---

### Error Handling

#### Orchestration Callbacks

```go
result, err := client.Orchestrate().
    Step("step1", stepBuilder).End().
    Step("step2", stepBuilder).End().
    OnComplete(func(result *espresso.OrchestrationResult[T]) {
        fmt.Printf("Orchestration completed in %v\n", result.Duration)
        fmt.Printf("Total steps: %d\n", len(result.Steps))
    }).
    OnError(func(err error, result *espresso.OrchestrationResult[T]) {
        fmt.Printf("Orchestration failed: %v\n", err)
        for _, step := range result.Steps {
            if !step.Success {
                fmt.Printf("Failed step: %s - %v\n", step.Name, step.Error)
            }
        }
    }).
    Execute(context.Background())
```

#### Analyzing Results

```go
result, err := client.Orchestrate().
    // ... steps ...
    Execute(context.Background())

fmt.Printf("Total Duration: %v\n", result.Duration)
fmt.Printf("Success: %v\n", result.Success)

for _, step := range result.Steps {
    if step.Success {
        fmt.Printf("✓ %s: SUCCESS (duration: %v, status: %d)\n",
            step.Name, step.Duration, step.StatusCode)
    } else if step.Skipped {
        fmt.Printf("⏭ %s: SKIPPED\n", step.Name)
    } else {
        fmt.Printf("✗ %s: FAILED (error: %v)\n", step.Name, step.Error)
    }
}
```

#### Step-Level Error Handling

```go
result, err := client.Orchestrate().
    Step("risky-operation", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/risky")
    }).
    OnSuccess(func(response *espresso.Response[Data], ctx map[string]any) error {
        fmt.Println("Operation succeeded")
        return nil
    }).
    OnFail(func(err error, ctx map[string]any) error {
        fmt.Printf("Operation failed: %v\n", err)
        // Return nil to continue orchestration
        // Return error to stop orchestration
        return nil
    }).
    OnStatusCode(404, func(response *espresso.Response[Data], ctx map[string]any) error {
        fmt.Println("Resource not found, creating new one")
        return nil
    }).
    FailOn(500, 502, 503). // Fail orchestration on these status codes
    End().
    Execute(context.Background())
```


---

##  Examples

### Complete Example: E-commerce Checkout Flow

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/Andrea-Cavallo/espresso"
)

type (
    User struct {
        ID    int    `json:"id"`
        Email string `json:"email"`
    }

    Cart struct {
        ID     int     `json:"id"`
        Items  []Item  `json:"items"`
        Total  float64 `json:"total"`
    }

    Item struct {
        ProductID int     `json:"product_id"`
        Quantity  int     `json:"quantity"`
        Price     float64 `json:"price"`
    }

    Payment struct {
        ID     string  `json:"id"`
        Amount float64 `json:"amount"`
        Status string  `json:"status"`
    }

    Order struct {
        ID      int    `json:"id"`
        UserID  int    `json:"user_id"`
        Status  string `json:"status"`
    }
)

func main() {
    ctx := context.Background()

    // Create client with resilient configuration
    client := espresso.NewClient[any](espresso.ClientConfig{
        BaseURL: "https://api.example.com",
        Timeout: 30 * time.Second,
    })

    // Configure retry and circuit breaker
    client.SetRetryStrategy(&espresso.ExponentialBackoffRetry{
        MaxAttempts: 3,
        BaseDelay:   100 * time.Millisecond,
        MaxDelay:    5 * time.Second,
    })

    client.SetCircuitBreaker(espresso.NewCircuitBreaker("api",
        espresso.DefaultCircuitBreakerConfig()))

    // Execute checkout workflow using Saga pattern
    result, err := client.Patterns().Saga().
        Step("validate-cart",
            func(ctx map[string]any) espresso.RequestBuilder[Cart] {
                userID := ctx["user_id"].(int)
                return client.Request(fmt.Sprintf("/carts/user/%d", userID))
            },
            nil). // No compensation needed
        Step("reserve-inventory",
            func(ctx map[string]any) espresso.RequestBuilder[any] {
                cart := ctx["cart"].(Cart)
                return client.Request("/inventory/reserve").
                    WithBody(cart.Items)
            },
            func(ctx map[string]any) error {
                // Release inventory on failure
                reservationID := ctx["reservation_id"].(string)
                _, err := client.Request("/inventory/release/"+reservationID).
                    Delete(context.Background())
                return err
            }).
        Step("process-payment",
            func(ctx map[string]any) espresso.RequestBuilder[Payment] {
                cart := ctx["cart"].(Cart)
                return client.Request("/payments").
                    WithBody(map[string]any{
                        "amount":   cart.Total,
                        "currency": "USD",
                    })
            },
            func(ctx map[string]any) error {
                // Refund payment on failure
                paymentID := ctx["payment_id"].(string)
                _, err := client.Request("/payments/"+paymentID+"/refund").
                    Post(context.Background())
                return err
            }).
        Step("create-order",
            func(ctx map[string]any) espresso.RequestBuilder[Order] {
                userID := ctx["user_id"].(int)
                cart := ctx["cart"].(Cart)
                return client.Request("/orders").
                    WithBody(map[string]any{
                        "user_id": userID,
                        "items":   cart.Items,
                        "total":   cart.Total,
                    })
            },
            func(ctx map[string]any) error {
                // Cancel order on failure
                orderID := ctx["order_id"].(int)
                _, err := client.Request(fmt.Sprintf("/orders/%d", orderID)).
                    Delete(context.Background())
                return err
            }).
        Execute(ctx)

    if err != nil {
        log.Fatalf("Checkout failed: %v", err)
    }

    fmt.Printf("Checkout completed successfully in %v\n", result.Duration)
    fmt.Printf("Order ID: %v\n", result.Context["order_id"])
}
```

---

##  Best Practices

### 1. Use Appropriate Generic Types

```go
// ✓ Good: Type-safe
client := espresso.NewClient[User](config)
response, _ := client.Request("/users/123").Get(ctx)
user := response.Data // *User

// ✗ Avoid: Type erasure
client := espresso.NewClient[any](config)
response, _ := client.Request("/users/123").Get(ctx)
user := response.Data.(User) // Requires type assertion
```

### 2. Handle Context Properly

```go
// ✓ Good: With timeout and cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Request("/api").Get(ctx)

// ✗ Avoid: Using context.Background() everywhere
response, err := client.Request("/api").Get(context.Background())
```

### 3. Reuse Clients

```go
// ✓ Good: Create once, reuse many times
var globalClient = espresso.NewClient[User](config)

func getUser(ctx context.Context, id int) (*User, error) {
    response, err := globalClient.Request(fmt.Sprintf("/users/%d", id)).Get(ctx)
    if err != nil {
        return nil, err
    }
    return response.Data, nil
}

// ✗ Avoid: Creating clients in hot paths
func getUser(ctx context.Context, id int) (*User, error) {
    client := espresso.NewClient[User](config) // Don't do this!
    // ...
}
```

### 4. Use Patterns for Complex Logic

```go
// ✓ Good: Use Chain pattern for sequences
result, err := client.Patterns().Chain().
    Call("step1", builder1).
    Call("step2", builder2).
    Execute(ctx)

// ✗ Avoid: Manual nesting
response1, err := client.Request("/step1").Get(ctx)
if err != nil {
    return err
}
response2, err := client.Request("/step2").Get(ctx)
if err != nil {
    return err
}
```

### 5. Configure Retry and Circuit Breaker Appropriately

```go
// ✓ Good: For external unstable endpoints
client := espresso.NewClient[Data](config)
client.SetRetryStrategy(&espresso.ExponentialBackoffRetry{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    5 * time.Second,
})
client.SetCircuitBreaker(espresso.NewCircuitBreaker("external-api",
    espresso.AggressiveCircuitBreakerConfig()))

// ✗ Avoid: Using retry for internal endpoints
// Don't use aggressive retry for your own stable APIs
```

### 6. Use Structured Logging

```go
// ✓ Good: Use structured logging
client.Orchestrate().
    Step("fetch-data", builder).
    Log(logger). // Automatically logs success/failure
    End()

// ✗ Avoid: Manual logging everywhere
client.Orchestrate().
    Step("fetch-data", builder).
    OnSuccess(func(r *Response[T], ctx map[string]any) error {
        fmt.Println("Success") // Too basic
        return nil
    })
```

### 7. Handle Errors Gracefully

```go
// ✓ Good: Proper error handling with context
response, err := client.Request("/api").Get(ctx)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("request timeout: %w", err)
    }
    return fmt.Errorf("request failed: %w", err)
}

// ✗ Avoid: Ignoring errors
response, _ := client.Request("/api").Get(ctx)
// What if it failed?
```

---

##  Advanced Configuration

### Custom Transport

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
}

client := espresso.NewClient[User](espresso.ClientConfig{
    Transport: transport,
})
```

### Custom Middleware

```go
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(
    req *http.Request,
    next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
    start := time.Now()
    fmt.Printf("→ %s %s\n", req.Method, req.URL.Path)

    resp, err := next(req)

    duration := time.Since(start)
    if err != nil {
        fmt.Printf("← ERROR after %v: %v\n", duration, err)
    } else {
        fmt.Printf("← %d after %v\n", resp.StatusCode, duration)
    }

    return resp, err
}

func (m *LoggingMiddleware) Name() string {
    return "logging"
}

client := espresso.NewClient[User](espresso.ClientConfig{
    Middlewares: []espresso.Middleware{&LoggingMiddleware{}},
})
```

### Custom Cache Provider

```go
type RedisCache[T any] struct {
    client *redis.Client
}

func (r *RedisCache[T]) Get(key string) (*T, bool) {
    // Implementation
}

func (r *RedisCache[T]) Set(key string, value T, ttl time.Duration) error {
    // Implementation
}

func (r *RedisCache[T]) Delete(key string) error {
    // Implementation
}

func (r *RedisCache[T]) Clear() error {
    // Implementation
}

cache := &RedisCache[User]{client: redisClient}
client.SetCacheProvider(cache)
```

---

##  Metrics and Monitoring

```go
type MetricsCollector struct {
    requests       prometheus.Counter
    errors         prometheus.Counter
    duration       prometheus.Histogram
    circuitBreaker prometheus.Gauge
}

func (m *MetricsCollector) RecordRequest(method, path string, duration time.Duration, statusCode int) {
    m.requests.Inc()
    m.duration.Observe(duration.Seconds())
}

func (m *MetricsCollector) RecordError(method, path, errorType string) {
    m.errors.Inc()
}

func (m *MetricsCollector) RecordRetryAttempt(method, path string, attempt int) {
    // Record retry metrics
}

func (m *MetricsCollector) RecordCircuitBreakerState(endpoint string, state espresso.CircuitState) {
    m.circuitBreaker.Set(float64(state))
}

client.SetMetricsCollector(&MetricsCollector{
    // Initialize prometheus metrics
})
```

---

##  Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Andrea-Cavallo/espresso.git
cd espresso

# Install dependencies
go mod download

# Run tests
go test ./... -v

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run specific test
go test -run TestOrchestrate

# Run benchmarks
go test -bench=. -benchmem
```

---

##  License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

##  Acknowledgments

- Inspired by popular HTTP client libraries in the Go ecosystem
- Circuit Breaker pattern based on [gobreaker](https://github.com/sony/gobreaker)
- Retry logic inspired by [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp)

---


This project is licensed under the MIT License - see the LICENSE file for details.


<div align="center">

**Made with  by [Andrea Cavallo](https://github.com/Andrea-Cavallo)**

 Star this repository if you find it useful!

</div>