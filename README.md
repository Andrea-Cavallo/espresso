# Espresso - Libreria Go per Orchestrazione API

Espresso è una libreria Go moderna per orchestrare chiamate API con supporto completo per pattern avanzati,
 retry automatici, circuit breaker, caching.


## Installazione

```bash
go get github.com/Andrea-Cavallo/espresso
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/Andrea-Cavallo/espresso"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    ctx := context.Background()

    // Crea un client con configurazione base
    client := espresso.NewDefaultClient[User]()

    // Esegui una semplice richiesta GET
    response, err := client.Request("https://api.example.com/users/123").
        WithBearerToken("your-token").
        EnableRetry().
        Get(ctx)

    if err != nil {
        panic(err)
    }

    fmt.Printf("User: %+v\n", response.Data)
}
```

## Creazione Client

### Client Base

```go
// Client con configurazione di default
client := espresso.NewDefaultClient[MyType]()

// Client ottimizzato per resilienza
client := espresso.NewResilientClient[MyType]()

// Client ottimizzato per velocità
client := espresso.NewFastClient[MyType]()
```

### Client Personalizzato

```go
client := espresso.NewClientBuilder[User]().
    WithBaseURL("https://api.example.com").
    WithTimeout(30 * time.Second).
    WithBearerToken("your-token").
    WithExponentialBackoff(3, 100*time.Millisecond, 10*time.Second).
    WithDefaultCircuitBreaker().
    WithMetrics().
    WithCache().
    Build()
```

## Pattern di Orchestrazione

### 1. Chain Pattern

Esegui chiamate API in sequenza. Ogni chiamata dipende dal successo della precedente.

```go
result, err := client.Patterns().Chain().
    Call("authenticate", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/auth/login").
            WithBody(credentials)
    }).
    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
        ctx["token"] = response.Data.Token
        return nil
    }).
    Call("get_profile", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/user/profile").
            WithBearerToken(ctx["token"].(string))
    }).
    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
        ctx["user_id"] = response.Data.ID
        return nil
    }).
    Execute(ctx)
```

### 2. Fan-Out Pattern

Esegui chiamate parallele e aggrega i risultati.

```go
result, err := client.Patterns().FanOut().
    Add("user_data", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/user/123")
    }).
    Add("user_orders", func(ctx map[string]any) espresso.RequestBuilder[Order] {
        return client.Request("/orders?user_id=123")
    }).
    Add("user_reviews", func(ctx map[string]any) espresso.RequestBuilder[Review] {
        return client.Request("/reviews?user_id=123")
    }).
    WithAggregator(func(results []espresso.StepResult, ctx map[string]any) (any, error) {
        // Aggrega i risultati come preferisci
        return aggregatedData, nil
    }).
    Execute(ctx)
```

### 3. Saga Pattern

Gestisci transazioni distribuite con compensazioni automatiche in caso di errore.

```go
result, err := client.Patterns().Saga().
    Step("reserve_inventory",
        func(ctx map[string]any) espresso.RequestBuilder[Inventory] {
            return client.Request("/inventory/reserve").
                WithBody(reservationData)
        },
        // Compensazione in caso di errore
        func(ctx map[string]any) error {
            // Rilascia l'inventario
            return releaseInventory(ctx)
        }).
    Step("create_order",
        func(ctx map[string]any) espresso.RequestBuilder[Order] {
            return client.Request("/orders").WithBody(orderData)
        },
        func(ctx map[string]any) error {
            // Cancella l'ordine
            return cancelOrder(ctx)
        }).
    Step("process_payment",
        func(ctx map[string]any) espresso.RequestBuilder[Payment] {
            return client.Request("/payments").WithBody(paymentData)
        },
        func(ctx map[string]any) error {
            // Rimborsa il pagamento
            return refundPayment(ctx)
        }).
    Execute(ctx)
```

### 4. Conditional Pattern

Esegui logica condizionale basata sul contesto.

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
        "apply_discount",
        func(ctx map[string]any) espresso.RequestBuilder[Discount] {
            return client.Request("/discounts/premium")
        }).
    Else("standard_pricing",
        func(ctx map[string]any) espresso.RequestBuilder[Price] {
            return client.Request("/pricing/standard")
        }).
    Execute(ctx)
```

### 5. Batch Pattern

Elabora batch di elementi simili.

```go
items := []any{
    Product{ID: 1},
    Product{ID: 2},
    Product{ID: 3},
}

result, err := client.Patterns().Batch().
    WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[Product] {
        product := item.(Product)
        return client.Request(fmt.Sprintf("/products/%d", product.ID))
    }).
    WithItems(items).
    WithBatchSize(10).
    Execute(ctx)
```

## Orchestrazione Base

### Step Singolo

```go
result, err := client.Orchestrate().
    Step("get_user", func(ctx map[string]any) espresso.RequestBuilder[User] {
        return client.Request("/users/123")
    }).
    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
        fmt.Printf("User loaded: %s\n", response.Data.Name)
        ctx["user_name"] = response.Data.Name
        return nil
    }).
    OnFail(func(err error, ctx map[string]any) error {
        fmt.Printf("Failed to load user: %v\n", err)
        return err
    }).
    End().
    Execute(ctx)
```

### Step Condizionale

```go
result, err := client.Orchestrate().
    Step("conditional_step", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/data")
    }).
    When(func(ctx map[string]any) bool {
        // Esegui solo se la condizione è vera
        return ctx["should_execute"] == true
    }).
    End().
    Execute(ctx)
```

### Step Opzionale

```go
result, err := client.Orchestrate().
    Step("optional_step", func(ctx map[string]any) espresso.RequestBuilder[Data] {
        return client.Request("/optional-data")
    }).
    Optional(). // L'errore non blocca l'orchestrazione
    End().
    Execute(ctx)
```

### Esecuzione Parallela

```go
result, err := client.Orchestrate().
    Parallel(). // Abilita esecuzione parallela
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
    Execute(ctx)
```

## Feature Avanzate

### Retry Automatico

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

### Circuit Breaker

```go
response, err := client.Request("/api/endpoint").
    WithCircuitBreakerConfig(&espresso.CircuitBreakerConfig{
        MaxRequests:      10,
        Interval:         60 * time.Second,
        Timeout:          10 * time.Second,
        FailureThreshold: 5,
        SuccessThreshold: 3,
    }).
    Get(ctx)
```

### Caching

```go
response, err := client.Request("/api/users/123").
    WithCache("user:123", 5 * time.Minute).
    Get(ctx)

if response.Cached {
    fmt.Println("Risposta servita dalla cache")
}
```

### Timeout Personalizzato

```go
response, err := client.Request("/api/slow-endpoint").
    WithTimeout(5 * time.Second).
    Get(ctx)
```

### Autenticazione

```go
// Bearer Token
response, err := client.Request("/api/protected").
    WithBearerToken("your-token").
    Get(ctx)

// Basic Auth
response, err := client.Request("/api/protected").
    WithBasicAuth("username", "password").
    Get(ctx)

// Custom Auth
response, err := client.Request("/api/protected").
    WithCustomAuth(customAuthProvider).
    Get(ctx)
```

## Gestione Errori

### Callback di Orchestrazione

```go
result, err := client.Orchestrate().
    Step("step1", stepBuilder).End().
    Step("step2", stepBuilder).End().
    OnComplete(func(result *espresso.OrchestrationResult[T]) {
        fmt.Println("Orchestrazione completata")
    }).
    OnError(func(err error, result *espresso.OrchestrationResult[T]) {
        fmt.Printf("Errore: %v\n", err)
    }).
    Execute(ctx)
```

### Analisi Risultati

```go
result, err := client.Orchestrate().
    // ... steps ...
    Execute(ctx)

fmt.Printf("Durata totale: %v\n", result.Duration)
fmt.Printf("Step eseguiti: %d\n", len(result.Steps))

for _, step := range result.Steps {
    if step.Success {
        fmt.Printf("✓ %s: SUCCESS (durata: %v)\n", step.Name, step.Duration)
    } else if step.Skipped {
        fmt.Printf("⏭ %s: SKIPPED\n", step.Name)
    } else {
        fmt.Printf("✗ %s: FAILED (errore: %v)\n", step.Name, step.Error)
    }
}
```


## Best Practices

### 1. Usa i Generics Appropriati

```go
// ✓ Buono: Type-safe
client := espresso.NewClient[User](config)

// ✗ Evita: Type erasure
client := espresso.NewClient[any](config)
```

### 2. Gestisci il Context Correttamente

```go
// ✓ Buono: Passa il context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Request("/api").Get(ctx)
```

### 3. Riutilizza i Client

```go
// ✓ Buono: Crea una volta, riusa molte volte
var globalClient = espresso.NewDefaultClient[User]()

func getUser(id int) (*User, error) {
    response, err := globalClient.Request(fmt.Sprintf("/users/%d", id)).Get(ctx)
    return response.Data, err
}
```

### 4. Usa Pattern per Logica Complessa

```go
// ✓ Buono: Usa Chain per sequenze
result, err := client.Patterns().Chain().
    Call("step1", builder1).
    Call("step2", builder2).
    Execute(ctx)

// ✗ Evita: Nesting manuale
response1, err := client.Request("/step1").Get(ctx)
if err != nil {
    // ...
}
response2, err := client.Request("/step2").Get(ctx)
```

### 5. Configura Retry e Circuit Breaker

```go
// ✓ Buono: Per endpoint esterni instabili
client := espresso.NewClientBuilder[Data]().
    WithExponentialBackoff(3, 100*time.Millisecond, 5*time.Second).
    WithDefaultCircuitBreaker().
    Build()
```

