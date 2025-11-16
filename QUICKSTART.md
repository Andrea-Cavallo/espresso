# Espresso Quick Start Guide

> Orchestrate API calls effortlessly - Apache Camel style, for Go

##  5-Minute Start

### 1. Install

```bash
go get github.com/Andrea-Cavallo/espresso
```

### 2. Your First API Call

```go
package main

import (
    "context"
    "fmt"
    "github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Create a resilient client (with retry, circuit breaker, cache)
    client := espresso.NewResilientClient[User]()

    // Make a request
    response, err := client.Request("https://api.example.com/users/1").
        WithBearerToken("your-token").
        Get(context.Background())

    if err != nil {
        panic(err)
    }

    fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
}
```

##  Common Patterns

### Simple Chain (Sequential Calls)

Perfect for: Login → Get Profile → Fetch Data workflows

```go
client := espresso.NewResilientClient[any]()

result, err := client.Patterns().Chain().
    Call("login", func(ctx map[string]any) espresso.RequestBuilder[AuthToken] {
        return client.Request("/auth/login").WithBody(credentials)
    }).
    OnSuccess(func(resp *espresso.Response[AuthToken], ctx map[string]any) error {
        ctx["token"] = resp.Data.Token
        return nil
    }).
    Call("get-profile", func(ctx map[string]any) espresso.RequestBuilder[User] {
        token := ctx["token"].(string)
        return client.Request("/profile").WithBearerToken(token)
    }).
    Execute(context.Background())
```

### Fan-Out (Parallel Calls)

Perfect for: Dashboard data, aggregating multiple APIs

```go
result, err := client.Patterns().FanOut().
    Add("user", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    Add("orders", func(ctx map[string]any) espresso.RequestBuilder[[]Order] {
        return client.Request("/orders?userId=123")
    }).
    Add("stats", func(ctx map[string]any) espresso.RequestBuilder[Stats] {
        return client.Request("/stats/123")
    }).
    WithAggregator(func(results []espresso.StepResult, ctx map[string]any) (any, error) {
        return Dashboard{
            User:   results[0].Data,
            Orders: results[1].Data,
            Stats:  results[2].Data,
        }, nil
    }).
    Execute(context.Background())
```

### Saga (Distributed Transaction)

Perfect for: E-commerce checkout, booking systems

```go
result, err := client.Patterns().Saga().
    Step("reserve-inventory",
        func(ctx map[string]any) espresso.RequestBuilder[Reservation] {
            return client.Request("/inventory/reserve").WithBody(items)
        },
        func(ctx map[string]any) error {
            // Compensation: release inventory
            reservationID := ctx["reservation_id"].(string)
            _, err := client.Request("/inventory/release/"+reservationID).Delete(context.Background())
            return err
        }).
    Step("process-payment",
        func(ctx map[string]any) espresso.RequestBuilder[Payment] {
            return client.Request("/payments").WithBody(paymentData)
        },
        func(ctx map[string]any) error {
            // Compensation: refund payment
            paymentID := ctx["payment_id"].(string)
            _, err := client.Request("/payments/"+paymentID+"/refund").Post(context.Background())
            return err
        }).
    Execute(context.Background())
```

### Content-Based Router

Perfect for: Multi-tenant systems, conditional workflows

```go
result, err := client.Orchestrate().
    Step("check-user", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    ResponseSwitch("user").
    OnValue("Subscription", "premium", "premium-flow", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/premium/data")
    }).
    OnValue("Subscription", "basic", "basic-flow", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/basic/data")
    }).
    Default("free-flow", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/free/data")
    }).
    End().
    Execute(context.Background())
```

### Batch Processing

Perfect for: Bulk operations, data migrations

```go
userIDs := []any{1, 2, 3, 4, 5}

result, err := client.Patterns().Batch().
    WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[User] {
        userID := item.(int)
        return client.Request(fmt.Sprintf("/users/%d", userID))
    }).
    WithItems(userIDs).
    WithBatchSize(10).      // Process 10 at a time
    WithConcurrency(3).     // 3 parallel workers
    Execute(context.Background())
```

##  Client Configurations

### Resilient Client (Recommended)

Includes: Retry + Circuit Breaker + Cache

```go
client := espresso.NewResilientClient[User]()
```

### Custom Client

```go
client := espresso.NewClientBuilder[User]().
    WithTimeout(30 * time.Second).
    WithExponentialBackoff(3, 100*time.Millisecond, 10*time.Second).
    WithDefaultCircuitBreaker().
    WithCache().
    WithRateLimiter(100, 200).  // 100 req/sec, burst 200
    Build()
```

##  Features Comparison

| Feature | Simple Client | Retry Client | Resilient Client |
|---------|--------------|--------------|------------------|
| Basic HTTP | ✅ | ✅ | ✅ |
| Retry Logic | ❌ | ✅ | ✅ |
| Circuit Breaker | ❌ | ❌ | ✅ |
| Caching | ❌ | ❌ | ✅ |
| Metrics | ✅ | ✅ | ✅ |
| Rate Limiting | ❌ | ❌ | ❌* |

*Add with `.WithRateLimiter()`

##  Common Use Cases

### 1. Microservices Communication

```go
// Service A calls Service B and C
client := espresso.NewResilientClient[any]()

client.Patterns().FanOut().
    Add("service-b", func(ctx map[string]any) espresso.RequestBuilder[ResponseB] {
        return client.Request("http://service-b/api/data")
    }).
    Add("service-c", func(ctx map[string]any) espresso.RequestBuilder[ResponseC] {
        return client.Request("http://service-c/api/data")
    }).
    Execute(context.Background())
```

### 2. API Gateway Aggregation

```go
// Aggregate multiple backend services
client.Patterns().FanOut().
    Add("users", func(ctx map[string]any) espresso.RequestBuilder[[]User] {
        return client.Request("/api/users")
    }).
    Add("products", func(ctx map[string]any) espresso.RequestBuilder[[]Product] {
        return client.Request("/api/products")
    }).
    WithAggregator(func(results []espresso.StepResult, ctx map[string]any) (any, error) {
        return AggregatedResponse{
            Users:    results[0].Data,
            Products: results[1].Data,
        }, nil
    }).
    Execute(context.Background())
```

### 3. External API Integration

```go
client := espresso.NewClientBuilder[any]().
    WithBaseURL("https://api.external.com").
    WithBearerToken(apiKey).
    WithExponentialBackoff(5, 1*time.Second, 30*time.Second).
    WithDefaultCircuitBreaker().
    WithCache().
    Build()

response, err := client.Request("/v1/data").
    WithCache("data-cache", 5*time.Minute).
    Get(context.Background())
```

### 4. E-commerce Checkout

```go
client.Patterns().Saga().
    Step("validate-cart", cartValidation, nil).
    Step("reserve-inventory", reserveInventory, releaseInventory).
    Step("process-payment", processPayment, refundPayment).
    Step("create-order", createOrder, cancelOrder).
    Step("send-confirmation", sendEmail, nil).
    Execute(context.Background())
```

##  Advanced Features

### Nested Field Access (NEW!)

```go
// Access nested fields in response switching
client.Orchestrate().
    Step("get-user", getUserBuilder).
    ResponseSwitch("user").
    OnValue("Address.City", "New York", "ny-handler", nyBuilder).
    OnValue("Profile.Settings.Theme", "dark", "dark-theme", darkBuilder).
    End()
```

### Rate Limiting (NEW!)

```go
// Token Bucket (default)
client := espresso.NewClientBuilder[User]().
    WithRateLimiter(100, 200).  // 100 req/sec, burst 200
    Build()

// Sliding Window
client := espresso.NewClientBuilder[User]().
    WithSlidingWindowRateLimit(1000, 1*time.Minute).  // 1000 req/min
    Build()

// Fixed Window
client := espresso.NewClientBuilder[User]().
    WithFixedWindowRateLimit(100, 1*time.Second).
    Build()
```

### Custom Cache (NEW!)

```go
// In-memory cache (default)
cache := espresso.NewInMemoryCache[User]()

// LRU cache with capacity
cache := espresso.NewLRUCache[User](1000)

client.SetCacheProvider(cache)
```

##  Best Practices

### 1. ✅ Use Type-Safe Clients

```go
// Good
userClient := espresso.NewResilientClient[User]()
postClient := espresso.NewResilientClient[Post]()

// Avoid
client := espresso.NewResilientClient[any]()  // Loses type safety
```

### 2. ✅ Reuse Clients

```go
// Good: Create once, use everywhere
var globalClient = espresso.NewResilientClient[User]()

func GetUser(ctx context.Context, id int) (*User, error) {
    response, err := globalClient.Request(fmt.Sprintf("/users/%d", id)).Get(ctx)
    // ...
}

// Avoid: Creating clients in loops
for _, id := range userIDs {
    client := espresso.NewResilientClient[User]()  // BAD!
    // ...
}
```

### 3. ✅ Handle Contexts Properly

```go
// Good: With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Request("/api").Get(ctx)

// Avoid: Using context.Background() everywhere
```

### 4. ✅ Use Patterns for Complex Flows

```go
// Good: Use Saga for transactions
client.Patterns().Saga().Step(...).Execute(ctx)

// Avoid: Manual error handling and rollback
```

##  Troubleshooting

### Issue: Requests timing out

```go
// Solution: Increase timeout
client := espresso.NewClientBuilder[User]().
    WithTimeout(60 * time.Second).  // Increase from default 30s
    Build()
```

### Issue: Too many retries

```go
// Solution: Configure retry strategy
client := espresso.NewClientBuilder[User]().
    WithExponentialBackoff(
        3,                      // Max 3 attempts
        100*time.Millisecond,   // Start with 100ms
        5*time.Second,          // Max 5s delay
    ).
    Build()
```

### Issue: Circuit breaker always open

```go
// Solution: Adjust thresholds
config := espresso.CircuitBreakerConfig{
    FailureThreshold: 10,  // Increase from default 5
    SuccessThreshold: 3,
    Timeout:         30 * time.Second,
}

client := espresso.NewClientBuilder[User]().
    WithCircuitBreaker(config).
    Build()
```

##  Next Steps

1. Check [examples/camel_style_examples.go](examples/camel_style_examples.go) for more patterns
2. Read the [full documentation](README.md)
3. Explore [orchestration patterns](docs/patterns.md)
4. Join our community on GitHub

##  Support

- GitHub Issues: https://github.com/Andrea-Cavallo/espresso/issues
- Documentation: Full README.md
- Examples: `/examples` directory

---

**Happy Orchestrating! ** Made with  by Andrea Cavallo
