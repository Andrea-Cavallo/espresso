# Espresso

A production-ready, type-safe HTTP client library for Go with advanced orchestration capabilities and complete Enterprise Integration Patterns (EIP) support.

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Andrea-Cavallo/espresso)](https://goreportcard.com/report/github.com/Andrea-Cavallo/espresso)

## Overview

Espresso is a comprehensive HTTP client library designed for building resilient, observable, and maintainable API integrations. It provides enterprise-grade features including retry logic, circuit breakers, rate limiting, caching, and complex orchestration patterns.

### Key Features

- **Type Safety**: Full generics support for compile-time type checking
- **Enterprise Integration Patterns**: 45+ EIP patterns including routing, transformation, channels, and endpoints
- **Message-Oriented Middleware**: Complete messaging abstraction layer with channels, routers, and transformers
- **Resilience**: Built-in retry logic with exponential backoff and jitter
- **Circuit Breaker**: Automatic failure detection and recovery
- **Caching**: Multiple caching strategies (in-memory, LRU)
- **Rate Limiting**: Token bucket, sliding window, and fixed window algorithms
- **Observability**: Metrics collection and distributed tracing support
- **Authentication**: Bearer tokens, Basic auth, and extensible custom schemes
- **Middleware**: Composable request/response pipeline
- **Orchestration**: Chain, fan-out, saga, batch, and conditional execution patterns
- **Context Support**: Full context propagation and cancellation

## Installation

```bash
go get github.com/Andrea-Cavallo/espresso
```

**Requirements**: Go 1.21 or higher

## Quick Start

### Minimal Example (3 lines!)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    client := espresso.NewDefaultClient[User]()
    response, err := client.Request("https://api.example.com/users/1").Get(context.Background())

    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", response.Data.Name)
}
```

### With Automatic Retry

```go
// Create a client with automatic retry (3 attempts)
client := espresso.NewRetryClient[User](3)

response, err := client.Request("https://api.example.com/users/1").
    EnableRetry().
    Get(context.Background())
```

### Full-Featured Client

```go
// Client with retry, circuit breaker, and cache
client := espresso.NewResilientClient[User]()

response, err := client.Request("https://api.example.com/users/1").
    WithBearerToken("your-token").
    WithCache("user-1", 5*time.Minute).
    Get(context.Background())
```

### Three Easy Ways to Create a Client

```go
// 1. Default - for simple use cases
client := espresso.NewDefaultClient[User]()

// 2. With Retry - automatically retries failed requests
client := espresso.NewRetryClient[User](3) // 3 retry attempts

// 3. Resilient - includes retry, circuit breaker, and cache
client := espresso.NewResilientClient[User]()

// 4. Custom - full control over configuration
client := espresso.NewClientBuilder[User]().
    WithBaseURL("https://api.example.com").
    WithTimeout(30 * time.Second).
    WithExponentialBackoff(3, 100*time.Millisecond, 10*time.Second).
    WithDefaultCircuitBreaker().
    WithCache().
    WithRateLimiter(100, 200). // 100 req/sec, burst 200
    Build()
```

## Core Features

### Retry Logic

Automatic retry with configurable backoff strategies:

```go
response, err := client.Request("/api/resource").
    WithRetryConfig(&espresso.RetryConfig{
        MaxAttempts:     5,
        BaseDelay:       100 * time.Millisecond,
        MaxDelay:        10 * time.Second,
        BackoffStrategy: espresso.BackoffExponential,
        JitterEnabled:   true,
        JitterType:      espresso.JitterFull,
        RetriableStatus: []int{429, 500, 502, 503, 504},
    }).
    Get(ctx)
```

### Circuit Breaker

Prevent cascading failures with circuit breaker pattern:

```go
client := espresso.NewClientBuilder[Response]().
    WithCircuitBreaker(espresso.CircuitBreakerConfig{
        MaxRequests:      10,
        Interval:         60 * time.Second,
        Timeout:          10 * time.Second,
        FailureThreshold: 5,
        SuccessThreshold: 3,
        HalfOpenMaxReqs:  5,
    }).
    Build()
```

### Caching

Multiple caching strategies available:

```go
// In-memory cache with TTL
cache := espresso.NewInMemoryCache[User]()
client.SetCacheProvider(cache)

response, err := client.Request("/users/123").
    WithCache("user:123", 5*time.Minute).
    Get(ctx)

// LRU cache with capacity limit
lruCache := espresso.NewLRUCache[User](1000)
client.SetCacheProvider(lruCache)
```

### Rate Limiting

Control request throughput with multiple algorithms:

```go
// Token bucket (recommended for most use cases)
client := espresso.NewClientBuilder[Data]().
    WithRateLimiter(100, 200). // 100 req/sec, burst 200
    Build()

// Sliding window
client := espresso.NewClientBuilder[Data]().
    WithSlidingWindowRateLimit(1000, 1*time.Minute). // 1000 req/min
    Build()

// Fixed window
client := espresso.NewClientBuilder[Data]().
    WithFixedWindowRateLimit(100, 1*time.Second). // 100 req/sec
    Build()
```

## Orchestration Patterns

### Chain Pattern

Execute requests sequentially where each step depends on the previous:

```go
result, err := client.Patterns().Chain().
    Call("authenticate", func(ctx map[string]any) espresso.RequestBuilder[AuthToken] {
        return client.Request("/auth/login").WithBody(credentials)
    }).
    OnSuccess(func(resp *espresso.Response[AuthToken], ctx map[string]any) error {
        ctx["token"] = resp.Data.Token
        return nil
    }).
    Call("fetch-profile", func(ctx map[string]any) espresso.RequestBuilder[User] {
        token := ctx["token"].(string)
        return client.Request("/profile").WithBearerToken(token)
    }).
    Execute(context.Background())
```

### Fan-Out Pattern

Execute multiple requests in parallel and aggregate results:

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
        return DashboardData{
            User:   results[0].Data,
            Orders: results[1].Data,
            Stats:  results[2].Data,
        }, nil
    }).
    Execute(context.Background())
```

### Saga Pattern

Handle distributed transactions with automatic compensation:

```go
result, err := client.Patterns().Saga().
    Step("reserve-inventory",
        func(ctx map[string]any) espresso.RequestBuilder[Reservation] {
            return client.Request("/inventory/reserve").WithBody(items)
        },
        func(ctx map[string]any) error {
            // Compensation: release inventory on failure
            reservationID := ctx["reservation_id"].(string)
            _, err := client.Request("/inventory/release/"+reservationID).
                Delete(context.Background())
            return err
        }).
    Step("process-payment",
        func(ctx map[string]any) espresso.RequestBuilder[Payment] {
            return client.Request("/payments").WithBody(paymentData)
        },
        func(ctx map[string]any) error {
            // Compensation: refund payment
            paymentID := ctx["payment_id"].(string)
            _, err := client.Request("/payments/"+paymentID+"/refund").
                Post(context.Background())
            return err
        }).
    Execute(context.Background())
```

### Batch Processing

Process multiple items with controlled concurrency:

```go
userIDs := []any{1, 2, 3, 4, 5}

result, err := client.Patterns().Batch().
    WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[User] {
        userID := item.(int)
        return client.Request(fmt.Sprintf("/users/%d", userID))
    }).
    WithItems(userIDs).
    WithBatchSize(10).      // Process 10 items per batch
    WithConcurrency(3).     // Use 3 parallel workers
    Execute(context.Background())
```

### Conditional Pattern

Execute different flows based on runtime conditions:

```go
result, err := client.Patterns().Conditional().
    WithContext(map[string]any{
        "user_type": "premium",
    }).
    When(
        func(ctx map[string]any) bool {
            return ctx["user_type"] == "premium"
        },
        "premium-flow",
        func(ctx map[string]any) espresso.RequestBuilder[Data] {
            return client.Request("/premium/data")
        }).
    ElseWhen(
        func(ctx map[string]any) bool {
            return ctx["user_type"] == "standard"
        },
        "standard-flow",
        func(ctx map[string]any) espresso.RequestBuilder[Data] {
            return client.Request("/standard/data")
        }).
    Else("default-flow",
        func(ctx map[string]any) espresso.RequestBuilder[Data] {
            return client.Request("/default/data")
        }).
    Execute(context.Background())
```

## Enterprise Integration Patterns (EIP)

Espresso provides a **complete implementation of 45+ Enterprise Integration Patterns** based on the canonical "Enterprise Integration Patterns" book by Gregor Hohpe and Bobby Woolf.

### Message-Oriented Architecture

EIP patterns work with a generic `Message[T]` abstraction instead of HTTP-specific requests:

```go
// Create a typed message
order := Order{ID: "123", Amount: 100.0}
msg := espresso.NewMessage(order)
msg.SetHeader("priority", "high")
msg.CorrelationID = "order-flow-123"

// Messages track their complete history
for _, entry := range msg.History {
    fmt.Printf("%s: %s at %s\n",
        entry.Component,
        entry.Action,
        entry.Timestamp)
}
```

### Core EIP Categories

#### 1. Message Routing Patterns

**Content-Based Router** - Route messages based on content:

```go
router := espresso.NewContentBasedRouter[Order]("order-router")
router.AddRoute("high-priority",
    func(msg *espresso.Message[Order]) bool {
        return msg.Body.Priority == "high"
    },
    highPriorityChannel)
router.AddRoute("standard",
    func(msg *espresso.Message[Order]) bool {
        return msg.Body.Priority == "standard"
    },
    standardChannel)

router.RouteAndSend(ctx, orderMessage)
```

**Recipient List** - Send to multiple recipients:

```go
recipients := espresso.NewRecipientListRouter[Event]("event-broadcast",
    analyticsChannel,
    loggingChannel,
    metricsChannel,
)

recipients.Send(ctx, eventMessage)
```

**Splitter and Aggregator** - Split messages and aggregate results:

```go
// Split order into items
splitter := espresso.NewSplitter[Order, OrderItem]("order-splitter",
    func(ctx context.Context, msg *espresso.Message[Order]) ([]*espresso.Message[OrderItem], error) {
        var items []*espresso.Message[OrderItem]
        for _, item := range msg.Body.Items {
            items = append(items, espresso.NewMessage(item))
        }
        return items, nil
    },
    itemChannel,
)

// Aggregate processed items
aggregator := espresso.NewAggregator[Item, Result](espresso.AggregatorConfig{
    Name: "result-aggregator",
    CompletionFunc: func(messages []*espresso.Message[Item]) bool {
        return len(messages) >= expectedCount
    },
    AggregateFunc: func(ctx context.Context, messages []*espresso.Message[Item]) (*espresso.Message[Result], error) {
        // Combine results
        return espresso.NewMessage(combinedResult), nil
    },
    Timeout: 10 * time.Second,
})
```

#### 2. Message Transformation Patterns

**Content Enricher** - Add data from external sources:

```go
enricher := espresso.NewContentEnricher[User]("user-enricher",
    func(ctx context.Context, msg *espresso.Message[User]) (*espresso.Message[User], error) {
        // Fetch additional data
        profile := fetchUserProfile(msg.Body.ID)
        msg.Body.Profile = profile
        return msg, nil
    },
)
```

**Message Translator** - Transform between types:

```go
translator := espresso.NewMessageTranslator[DTO, Entity]("dto-to-entity",
    func(ctx context.Context, msg *espresso.Message[DTO]) (*espresso.Message[Entity], error) {
        entity := convertDTOToEntity(msg.Body)
        return espresso.NewMessage(entity), nil
    },
)
```

**Pipeline** - Chain transformations:

```go
pipeline := espresso.NewPipeline[User]("user-pipeline")
pipeline.AddStage("validate", validationTransformer)
pipeline.AddStage("sanitize", sanitizationTransformer)
pipeline.AddStage("enrich", enrichmentTransformer)

result := pipeline.Process(ctx, userMessage)
```

#### 3. Messaging Channels Patterns

**Publish-Subscribe** - Broadcast to multiple subscribers:

```go
pubsub := espresso.NewPubSubChannel[Event]("events")

pubsub.Subscribe(analyticsChannel)
pubsub.Subscribe(loggingChannel)
pubsub.Subscribe(metricsChannel)

pubsub.Send(ctx, eventMessage) // All subscribers receive it
```

**Dead Letter Channel** - Handle failed messages:

```go
dlc := espresso.NewDeadLetterChannel[Order]("orders", mainChannel, 50)

// On processing error
dlc.SendToDLQ(ctx, failedMessage, err)

// Monitor failed messages
stats := dlc.GetDLQStats()
fmt.Printf("Failed: %d messages\n", stats.TotalMessages)
```

**Wire Tap** - Non-intrusive monitoring:

```go
wireTap := espresso.NewWireTap[Transaction]("txn-monitor", auditChannel)
tappedChannel := espresso.NewWireTapChannel("txns", mainChannel, wireTap)

// All messages are automatically copied to audit channel
tappedChannel.Send(ctx, txnMessage)
```

**Priority Channel** - Priority-based processing:

```go
priorities := []int{1, 2, 3, 4, 5} // 1 = highest
priorityChannel := espresso.NewPriorityChannel[Task]("tasks", priorities, 100)

msg.SetProperty("priority", 1) // High priority
priorityChannel.Send(ctx, msg)

// Receives highest priority first
task, _ := priorityChannel.Receive(ctx)
```

#### 4. Messaging Endpoints Patterns

**Idempotent Receiver** - Process messages exactly once:

```go
receiver := espresso.NewIdempotentReceiver[Payment]("payment-processor", 24*time.Hour)

// Duplicate messages are automatically ignored
receiver.Receive(ctx, paymentMessage, paymentHandler)
```

**Competing Consumers** - Load balancing across workers:

```go
pool := espresso.NewCompetingConsumers[Task]("worker-pool")

pool.AddConsumer(worker1Handler)
pool.AddConsumer(worker2Handler)
pool.AddConsumer(worker3Handler)

pool.Start(ctx, taskChannel)

stats := pool.GetStats()
fmt.Printf("Processed: %d, Errors: %d\n",
    stats.TotalProcessed,
    stats.TotalErrors)
```

**Service Activator** - Invoke services from messages:

```go
activator := espresso.NewServiceActivator[Request, Response]("api-service",
    requestChannel,
    responseChannel,
    serviceHandler,
)

activator.Start(ctx)
```

#### 5. System Management Patterns

**Process Manager** - Manage complex workflows with state:

```go
pm := espresso.NewProcessManager[OrderEvent]("order-fulfillment")

// Start process
pm.StartProcess(ctx, "order-123", initialEvent)

// Track progress
pm.UpdateProcessStep("order-123", "payment_received")
pm.UpdateProcessStep("order-123", "shipped")

// Get state
state, _ := pm.GetProcessState("order-123")
fmt.Printf("Status: %s, Steps: %d\n",
    state.Status,
    len(state.CompletedSteps))
```

**Control Bus** - Administrative control:

```go
controlBus := espresso.NewControlBus("system-control")

controlBus.Subscribe(func(ctx context.Context, cmd espresso.ControlCommand) error {
    switch cmd.Type {
    case "pause_channel":
        // Pause operations
    case "purge_dlq":
        // Clean dead letter queue
    }
    return nil
})

controlBus.SendCommand(ctx, espresso.ControlCommand{
    Type:   "pause_channel",
    Target: "orders",
})
```

**Message History Tracker** - Full message tracing:

```go
tracker := espresso.NewMessageHistoryTracker[Order]("order-tracker")

tracker.Track(orderMessage)

// Get complete message path
path := tracker.GetMessagePath(correlationID)
for _, entry := range path {
    fmt.Printf("%s: %s at %s\n",
        entry.Component,
        entry.Action,
        entry.Timestamp)
}
```

### Complete Pattern List

Espresso implements **45+ Enterprise Integration Patterns**:

**Routing (11)**: Content-Based Router, Message Router, Recipient List, Splitter, Aggregator, Resequencer, Message Filter, Composed Message Processor, Routing Slip, Scatter-Gather, Detour

**Transformation (7)**: Message Translator, Content Enricher, Content Filter, Claim Check, Normalizer, Canonical Data Model, Pipeline

**Channels (8)**: Point-to-Point, Publish-Subscribe, Dead Letter Channel, Invalid Message Channel, Guaranteed Delivery, Wire Tap, Priority Channel, Message Bridge

**Endpoints (9)**: Message Endpoint, Polling Consumer, Event-Driven Consumer, Competing Consumers, Message Dispatcher, Selective Consumer, Idempotent Receiver, Service Activator, Transactional Client

**System Management (10)**: Control Bus, Detour, Wire Tap, Message History, Message Store, Smart Proxy, Test Message, Channel Purger, Process Manager, Routing Slip

### EIP Examples

Complete working examples are available in [`examples/eip_patterns_example.go`](examples/eip_patterns_example.go).

For detailed documentation of all patterns, see [`docs/EIP_PATTERNS.md`](docs/EIP_PATTERNS.md).

### Mixing HTTP and EIP Patterns

You can seamlessly combine HTTP orchestration with EIP patterns:

```go
// Create HTTP client
httpClient := espresso.NewResilientClient[User]()

// Create EIP message channel
userChannel := espresso.NewInMemoryChannel[User]("users", 100)

// Route users based on properties
router := espresso.NewContentBasedRouter[User]("user-router")
router.AddRoute("premium", isPremium, premiumChannel)
router.AddRoute("standard", isStandard, standardChannel)

// Process through pipeline
pipeline := espresso.NewPipeline[User]("user-processing")
pipeline.AddStage("validate", validator)
pipeline.AddStage("enrich", enricher)
pipeline.AddStage("transform", transformer)
```

## Advanced Features

### Response Switching

Branch execution based on response data, including nested fields:

```go
result, err := client.Orchestrate().
    Step("check-user", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    ResponseSwitch("user").
    OnValue("Address.City", "New York", "ny-handler", nyBuilder).
    OnValue("Profile.Type", "premium", "premium-handler", premiumBuilder).
    Default("default-handler", defaultBuilder).
    End().
    Execute(context.Background())
```

### Authentication

Multiple authentication schemes supported:

```go
// Bearer token
response, err := client.Request("/api/protected").
    WithBearerToken("eyJhbGci...").
    Get(ctx)

// Basic auth
response, err := client.Request("/api/protected").
    WithBasicAuth("username", "password").
    Get(ctx)

// Custom authentication
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
    WithCustomAuth(&APIKeyAuth{APIKey: "your-key"}).
    Get(ctx)
```

### Middleware

Extend client functionality with custom middleware:

```go
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(
    req *http.Request,
    next func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
    start := time.Now()
    log.Printf("Request: %s %s", req.Method, req.URL.Path)

    resp, err := next(req)

    duration := time.Since(start)
    if err != nil {
        log.Printf("Error after %v: %v", duration, err)
    } else {
        log.Printf("Response: %d after %v", resp.StatusCode, duration)
    }

    return resp, err
}

func (m *LoggingMiddleware) Name() string {
    return "logging"
}

client := espresso.NewClientBuilder[User]().
    WithMiddleware(&LoggingMiddleware{}).
    Build()
```

### Metrics Integration

Collect metrics for monitoring:

```go
type PrometheusMetrics struct {
    requestCounter  prometheus.Counter
    requestDuration prometheus.Histogram
}

func (m *PrometheusMetrics) RecordRequest(method, path string, duration time.Duration, statusCode int) {
    m.requestCounter.Inc()
    m.requestDuration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordError(method, path, errorType string) {
    // Record error metrics
}

func (m *PrometheusMetrics) RecordRetryAttempt(method, path string, attempt int) {
    // Record retry metrics
}

func (m *PrometheusMetrics) RecordCircuitBreakerState(endpoint string, state espresso.CircuitState) {
    // Record circuit breaker state
}

client.SetMetricsCollector(&PrometheusMetrics{
    // Initialize Prometheus metrics
})
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestCircuitBreaker ./pkg/espresso
```

## Best Practices

### Use Type-Safe Clients

Always use specific types instead of `any` for better type safety:

```go
// Recommended
userClient := espresso.NewResilientClient[User]()
postClient := espresso.NewResilientClient[Post]()

// Avoid
genericClient := espresso.NewResilientClient[any]()
```

### Reuse Client Instances

Create clients once and reuse them:

```go
// Good: Create once, use everywhere
var globalClient = espresso.NewResilientClient[User]()

func GetUser(ctx context.Context, id int) (*User, error) {
    response, err := globalClient.Request(fmt.Sprintf("/users/%d", id)).Get(ctx)
    if err != nil {
        return nil, err
    }
    return response.Data, nil
}

// Avoid: Creating clients repeatedly
for _, id := range userIDs {
    client := espresso.NewResilientClient[User]() // Inefficient
    // ...
}
```

### Use Context Properly

Always use context with timeouts or cancellation:

```go
// Recommended
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Request("/api").Get(ctx)

// Avoid
response, err := client.Request("/api").Get(context.Background())
```

### Configure Appropriately

Tune retry and circuit breaker settings based on your use case:

```go
// For external unreliable services
client := espresso.NewClientBuilder[Data]().
    WithExponentialBackoff(5, 200*time.Millisecond, 30*time.Second).
    WithCircuitBreaker(espresso.CircuitBreakerConfig{
        FailureThreshold: 10,
        Timeout:         30 * time.Second,
    }).
    Build()

// For internal stable services
client := espresso.NewClientBuilder[Data]().
    WithExponentialBackoff(2, 50*time.Millisecond, 1*time.Second).
    Build()
```

## Architecture

Espresso follows clean architecture principles with clear separation of concerns:

- **Client Layer**: HTTP client with resilience features
- **Orchestration Layer**: Complex workflow patterns
- **Provider Layer**: Pluggable cache, rate limiting, and metrics
- **Middleware Layer**: Request/response pipeline
- **Type Layer**: Type-safe generic wrappers

All components are designed to be:
- **Extensible**: Implement interfaces to add custom behavior
- **Composable**: Mix and match features as needed
- **Testable**: Easy to mock and test in isolation
- **Observable**: Built-in metrics and tracing hooks

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run `go fmt` and `go vet`
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Espresso was inspired by battle-tested patterns from the distributed systems and microservices ecosystem. The library incorporates proven resilience patterns including:

- Circuit Breaker pattern from Michael Nygard's "Release It!"
- Retry strategies with jitter from AWS Architecture Blog
- Rate limiting algorithms from IETF and industry best practices
- Saga pattern from Hector Garcia-Molina and Kenneth Salem

## Support

- GitHub Issues: https://github.com/Andrea-Cavallo/espresso/issues
- Documentation: See this README and inline Go documentation
- Examples: Check the `/examples` directory

---

Made by Andrea Cavallo
