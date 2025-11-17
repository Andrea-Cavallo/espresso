# Enterprise Integration Patterns (EIP) - Complete Implementation

This document provides a comprehensive mapping of all Enterprise Integration Patterns implemented in the Espresso framework.

## Overview

Espresso now provides a **complete, type-safe implementation** of Enterprise Integration Patterns for Go, going beyond simple HTTP orchestration to offer a full-featured messaging and integration framework.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  EIP Layer (Message-Oriented Middleware)                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Core Abstractions                                    │   │
│  │ - Message[T]   - Channel[T]   - Router[T]          │   │
│  │ - Handler[T,R] - Transformer  - Aggregator         │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Pattern Implementations                                    │
│  ├─ Routing      ├─ Transformation  ├─ Channels            │
│  ├─ Endpoints    ├─ Management      └─ HTTP Adapters       │
├─────────────────────────────────────────────────────────────┤
│  HTTP Orchestration Layer (Existing)                        │
│  ├─ Chain   ├─ Fan-Out   ├─ Saga   ├─ Batch   ├─ Switch   │
└─────────────────────────────────────────────────────────────┘
```

## Pattern Categories

### 1. Message Construction Patterns

#### Message[T] - Core Message Abstraction
**File**: `eip_core.go:13-112`

```go
msg := NewMessage(userData)
msg.SetHeader("content-type", "application/json")
msg.CorrelationID = "order-12345"
msg.ExpiresAt = &expiryTime
```

**Features**:
- Unique message ID
- Correlation ID for related messages
- Headers and properties
- Message history tracking
- Expiration support
- Reply-to addressing
- Thread-safe operations

---

### 2. Message Routing Patterns

#### Content-Based Router
**File**: `eip_routing.go:67-123`

Routes messages to different channels based on content predicates.

```go
router := NewContentBasedRouter[Order]("order-router")
router.AddRoute("priority", func(msg *Message[Order]) bool {
    return msg.Body.Priority == "high"
}, highPriorityChannel)
router.AddRoute("standard", func(msg *Message[Order]) bool {
    return msg.Body.Priority == "standard"
}, standardChannel)
router.SetDefaultChannel(defaultChannel)

router.RouteAndSend(ctx, orderMessage)
```

**Use Cases**:
- Priority-based routing
- Customer segmentation
- Geographic routing
- Service level routing

#### Recipient List Router
**File**: `eip_routing.go:125-204`

Sends messages to a static or dynamic list of recipients.

```go
router := NewRecipientListRouter[Event]("event-broadcast",
    analyticsChannel,
    loggingChannel,
    metricsChannel,
)

router.Send(ctx, eventMessage)
```

**Use Cases**:
- Event broadcasting
- Parallel processing
- Fan-out notifications
- Multi-system updates

#### Message Filter
**File**: `eip_routing.go:206-256`

Filters messages before passing them through.

```go
filter := NewMessageFilterChannel[User]("active-users",
    upstreamChannel,
    func(msg *Message[User]) bool {
        return msg.Body.Active && !msg.Body.Deleted
    },
)
filter.SetRejectedChannel(inactiveUsersChannel)
```

**Use Cases**:
- Data validation
- Access control
- Quality filtering
- Rate limiting

#### Resequencer
**File**: `eip_routing.go:258-328`

Reorders out-of-sequence messages.

```go
resequencer := NewResequencer[Transaction]("txn-resequencer",
    outputChannel,
    func(msg *Message[Transaction]) int {
        return msg.Body.SequenceNumber
    },
)

resequencer.ProcessMessage(ctx, txnMessage)
```

**Use Cases**:
- Ordered message processing
- Transaction sequencing
- Event replay
- Stream processing

---

### 3. Message Transformation Patterns

#### Splitter
**File**: `eip_transformation.go:13-63`

Splits one message into multiple messages.

```go
splitter := NewSplitter[Order, OrderItem]("order-splitter",
    func(ctx context.Context, msg *Message[Order]) ([]*Message[OrderItem], error) {
        var items []*Message[OrderItem]
        for _, item := range msg.Body.Items {
            items = append(items, NewMessage(item))
        }
        return items, nil
    },
    itemChannel,
)

splitter.ProcessAndSend(ctx, orderMessage)
```

**Use Cases**:
- Batch decomposition
- Array processing
- Multi-recipient messages
- Distributed processing

#### Aggregator
**File**: `eip_transformation.go:65-177`

Aggregates multiple related messages into one.

```go
aggregator := NewAggregator[OrderItem, Order](AggregatorConfig{
    Name: "order-aggregator",
    CompletionFunc: func(messages []*Message[OrderItem]) bool {
        return len(messages) >= expectedCount
    },
    AggregateFunc: func(ctx context.Context, messages []*Message[OrderItem]) (*Message[Order], error) {
        var items []OrderItem
        for _, msg := range messages {
            items = append(items, msg.Body)
        }
        return NewMessage(Order{Items: items}), nil
    },
    Timeout: 10 * time.Second,
    OutputChannel: orderChannel,
})

aggregator.Aggregate(ctx, itemMessage)
```

**Use Cases**:
- Batch composition
- Parallel result gathering
- Multi-step workflows
- Response correlation

#### Content Enricher
**File**: `eip_transformation.go:179-205`

Enriches messages with additional data from external sources.

```go
enricher := NewContentEnricher[User]("user-enricher",
    func(ctx context.Context, msg *Message[User]) (*Message[User], error) {
        // Fetch additional data
        profile := fetchUserProfile(msg.Body.ID)
        msg.Body.Profile = profile
        return msg, nil
    },
)

enriched := enricher.Enrich(ctx, userMessage)
```

**Use Cases**:
- Data augmentation
- Reference lookups
- Cache hydration
- Metadata addition

#### Message Translator
**File**: `eip_transformation.go:207-243`

Transforms messages from one type to another.

```go
translator := NewMessageTranslator[UserDTO, User]("dto-to-entity",
    func(ctx context.Context, msg *Message[UserDTO]) (*Message[User], error) {
        user := User{
            ID:   msg.Body.UserID,
            Name: msg.Body.FullName,
        }
        return NewMessage(user), nil
    },
)

userMsg := translator.Transform(ctx, dtoMsg)
```

**Use Cases**:
- Format conversion
- Protocol translation
- Version migration
- Schema mapping

#### Normalizer
**File**: `eip_transformation.go:245-287`

Normalizes different message formats to a canonical format.

```go
normalizer := NewNormalizer[CanonicalOrder]("order-normalizer",
    func(msg *Message[any]) string {
        // Detect format based on headers or content
        return msg.GetHeader("format")
    },
)

normalizer.RegisterFormat("xml", xmlToCanonicalTranslator)
normalizer.RegisterFormat("json", jsonToCanonicalTranslator)
normalizer.RegisterFormat("csv", csvToCanonicalTranslator)

canonical := normalizer.Normalize(ctx, incomingMessage)
```

**Use Cases**:
- Multi-format ingestion
- Legacy system integration
- Protocol normalization
- Standard API gateway

#### Claim Check
**File**: `eip_transformation.go:289-334`

Stores large payloads externally and passes a reference.

```go
claimCheck := NewClaimCheck[LargePayload]("payload-store")

// Store large payload
claimID, _ := claimCheck.Store(ctx, largeData)

// Later, retrieve it
payload, _ := claimCheck.Retrieve(ctx, claimID)
```

**Use Cases**:
- Large file handling
- Memory optimization
- Attachment processing
- Document management

#### Pipeline
**File**: `eip_transformation.go:336-391`

Chains multiple transformations together.

```go
pipeline := NewPipeline[User]("user-pipeline")
pipeline.AddStage("validate", validationTransformer)
pipeline.AddStage("sanitize", sanitizationTransformer)
pipeline.AddStage("enrich", enrichmentTransformer)

processed := pipeline.Process(ctx, userMessage)
```

**Use Cases**:
- Multi-step transformation
- Data cleaning pipelines
- ETL workflows
- Message processing chains

#### Composed Message Processor
**File**: `eip_transformation.go:393-431`

Combines Splitter, Router, and Aggregator.

```go
processor := NewComposedMessageProcessor[Batch, Item, Result](
    "batch-processor",
    splitter,
    router,
    aggregator,
)

result := processor.Process(ctx, batchMessage)
```

**Use Cases**:
- Complex batch processing
- Distributed computation
- Map-reduce patterns
- Parallel workflows

---

### 4. Messaging Channels Patterns

#### In-Memory Channel
**File**: `eip_routing.go:12-65`

Basic buffered channel implementation.

```go
channel := NewInMemoryChannel[Order]("orders", 100)

channel.Send(ctx, orderMessage)
msg, _ := channel.Receive(ctx)
```

#### Pub-Sub Channel
**File**: `eip_channels.go:12-116`

Publish-subscribe messaging pattern.

```go
pubsub := NewPubSubChannel[Event]("events")

pubsub.Subscribe(analyticsChannel)
pubsub.Subscribe(loggingChannel)
pubsub.Subscribe(metricsChannel)

pubsub.Send(ctx, eventMessage) // Broadcasts to all subscribers
```

**Use Cases**:
- Event broadcasting
- Real-time notifications
- Multi-consumer patterns
- Event sourcing

#### Dead Letter Channel
**File**: `eip_channels.go:118-198`

Handles messages that cannot be processed.

```go
dlc := NewDeadLetterChannel[Order]("orders", mainChannel, 50)

// On error
dlc.SendToDLQ(ctx, failedMessage, err)

// Monitor DLQ
stats := dlc.GetDLQStats()
fmt.Printf("Failed messages: %d\n", stats.TotalMessages)
```

**Use Cases**:
- Error handling
- Poison message isolation
- Retry queue management
- Failure analysis

#### Wire Tap
**File**: `eip_channels.go:200-267`

Non-intrusive message monitoring.

```go
wireTap := NewWireTap[Transaction]("txn-monitor", monitoringChannel)

// Wrap channel with wire tap
tappedChannel := NewWireTapChannel("txns", mainChannel, wireTap)

// All messages are automatically copied to monitoring
tappedChannel.Send(ctx, txnMessage)
```

**Use Cases**:
- Message auditing
- Performance monitoring
- Debug logging
- Analytics collection

#### Guaranteed Delivery Channel
**File**: `eip_channels.go:269-326`

Ensures message persistence for reliability.

```go
store := NewInMemoryMessageStore[Order]("order-store")
guaranteed := NewGuaranteedDeliveryChannel("orders", channel, store)

// Messages are persisted before sending
guaranteed.Send(ctx, orderMessage)
```

**Use Cases**:
- Critical transactions
- Disaster recovery
- Audit trails
- Durable messaging

#### Invalid Message Channel
**File**: `eip_channels.go:366-426`

Routes invalid messages to a separate channel.

```go
invalidChannel := NewInvalidMessageChannel[User]("user-validation",
    func(msg *Message[User]) error {
        if msg.Body.Email == "" {
            return errors.New("email required")
        }
        return nil
    },
    validChannel,
    invalidChannel,
)
```

**Use Cases**:
- Input validation
- Data quality control
- Schema enforcement
- Error segregation

#### Priority Channel
**File**: `eip_channels.go:428-514`

Priority-based message processing.

```go
priorities := []int{1, 2, 3, 4, 5} // 1 = highest
priorityChannel := NewPriorityChannel[Task]("tasks", priorities, 100)

// Set priority on message
msg.SetProperty("priority", 1)
priorityChannel.Send(ctx, msg)

// Receives highest priority message first
task, _ := priorityChannel.Receive(ctx)
```

**Use Cases**:
- SLA-based processing
- Critical path optimization
- Emergency handling
- Tiered service levels

---

### 5. Messaging Endpoints Patterns

#### Idempotent Receiver
**File**: `eip_endpoints.go:13-78`

Ensures messages are processed only once.

```go
receiver := NewIdempotentReceiver[Payment]("payment-processor", 24*time.Hour)

receiver.Receive(ctx, paymentMessage, paymentHandler)
// Duplicate messages are silently ignored

if receiver.IsProcessed(messageID) {
    // Already processed
}
```

**Use Cases**:
- Payment processing
- Duplicate prevention
- Exactly-once semantics
- Retry safety

#### Competing Consumers
**File**: `eip_endpoints.go:80-211`

Multiple consumers compete for messages from a single channel.

```go
pool := NewCompetingConsumers[Task]("task-pool")

pool.AddConsumer(worker1Handler)
pool.AddConsumer(worker2Handler)
pool.AddConsumer(worker3Handler)

pool.Start(ctx, taskChannel)

// Get statistics
stats := pool.GetStats()
fmt.Printf("Processed: %d, Errors: %d\n", stats.TotalProcessed, stats.TotalErrors)
```

**Use Cases**:
- Load balancing
- Parallel processing
- Worker pools
- Throughput scaling

#### Polling Consumer
**File**: `eip_endpoints.go:213-274`

Polls a channel at regular intervals.

```go
consumer := NewPollingConsumer[Email]("email-poller",
    emailChannel,
    emailHandler,
    5*time.Second, // Poll every 5 seconds
)

consumer.Start(ctx)
```

**Use Cases**:
- Scheduled processing
- Batch jobs
- Rate-limited consumption
- Controlled throughput

#### Event-Driven Consumer
**File**: `eip_endpoints.go:276-338`

Consumes messages as they arrive (push-based).

```go
consumer := NewEventDrivenConsumer[Order]("order-processor",
    orderChannel,
    orderHandler,
)

consumer.Start(ctx)
```

**Use Cases**:
- Real-time processing
- Low-latency requirements
- Event streaming
- Reactive systems

#### Service Activator
**File**: `eip_endpoints.go:340-412`

Invokes a service in response to a message.

```go
activator := NewServiceActivator[Request, Response]("api-service",
    requestChannel,
    responseChannel,
    serviceHandler,
)

activator.Start(ctx)
```

**Use Cases**:
- Microservice invocation
- Async service calls
- Request-response pattern
- RPC-style messaging

#### Message Dispatcher
**File**: `eip_endpoints.go:414-486`

Distributes messages to performers using various strategies.

```go
dispatcher := NewMessageDispatcher[Task]("task-dispatcher", RoundRobin)

dispatcher.AddPerformer(performer1)
dispatcher.AddPerformer(performer2)
dispatcher.AddPerformer(performer3)

result := dispatcher.Dispatch(ctx, taskMessage)
```

**Strategies**:
- Round-robin
- Random
- Least busy (future)

**Use Cases**:
- Load distribution
- Work assignment
- Failover routing
- Resource optimization

#### Selective Consumer
**File**: `eip_endpoints.go:488-559`

Filters messages before consuming.

```go
consumer := NewSelectiveConsumer[Transaction]("high-value",
    txnChannel,
    func(msg *Message[Transaction]) bool {
        return msg.Body.Amount > 1000
    },
    highValueHandler,
    rejectedChannel,
)

consumer.Start(ctx)
```

**Use Cases**:
- Conditional processing
- Message filtering
- Subscription filtering
- Interest-based routing

---

### 6. System Management Patterns

#### Control Bus
**File**: `eip_management.go:13-81`

Administrative control over the messaging system.

```go
controlBus := NewControlBus("system-control")

controlBus.Subscribe(func(ctx context.Context, cmd ControlCommand) error {
    switch cmd.Type {
    case "pause_channel":
        // Pause a channel
    case "resume_channel":
        // Resume a channel
    case "purge_dlq":
        // Clean dead letter queue
    }
    return nil
})

controlBus.Start(ctx)

// Send control command
controlBus.SendCommand(ctx, ControlCommand{
    Type:   "pause_channel",
    Target: "orders",
})
```

**Use Cases**:
- Runtime configuration
- System administration
- Emergency controls
- Dynamic reconfiguration

#### Process Manager
**File**: `eip_management.go:83-194`

Manages complex routing with state.

```go
pm := NewProcessManager[OrderEvent]("order-process")

// Start a process
pm.StartProcess(ctx, "order-12345", initialEvent)

// Handle subsequent events
pm.HandleMessage(ctx, "order-12345", paymentEvent)
pm.HandleMessage(ctx, "order-12345", shippingEvent)

// Update process state
pm.UpdateProcessStep("order-12345", "shipped")

// Get process state
state := pm.GetProcessState("order-12345")

// Complete process
pm.CompleteProcess(ctx, "order-12345")
```

**Use Cases**:
- Long-running workflows
- Saga orchestration
- Stateful processing
- Business process management

#### Routing Slip
**File**: `eip_management.go:196-247`

Dynamic message routing sequence.

```go
processor := NewRoutingSlipProcessor[Order]("order-processor")

processor.RegisterComponent("validate", validationHandler)
processor.RegisterComponent("enrich", enrichmentHandler)
processor.RegisterComponent("persist", persistenceHandler)

slip := &RoutingSlip{
    Steps: []RoutingStep{
        {Component: "validate", Action: "check"},
        {Component: "enrich", Action: "add_data"},
        {Component: "persist", Action: "save"},
    },
}

result := processor.Process(ctx, orderMessage, slip)
```

**Use Cases**:
- Dynamic workflows
- Runtime routing decisions
- Configurable pipelines
- Business rule routing

#### Detour
**File**: `eip_management.go:249-275`

Conditionally routes messages for monitoring.

```go
detour := NewDetourRouter[Transaction]("txn-detour",
    normalChannel,
    monitoringChannel,
    func(msg *Message[Transaction]) bool {
        return msg.Body.Amount > 10000 // Monitor high-value txns
    },
)

detour.Route(ctx, txnMessage)
```

**Use Cases**:
- Selective monitoring
- Debug mode routing
- Compliance monitoring
- Conditional auditing

#### Message History Tracker
**File**: `eip_management.go:277-338`

Tracks message paths through the system.

```go
tracker := NewMessageHistoryTracker[Order]("order-tracker")

// Track messages
tracker.Track(orderMessage)

// Get full history
history := tracker.GetHistory(correlationID)

// Get message path
path := tracker.GetMessagePath(correlationID)

for _, entry := range path {
    fmt.Printf("%s: %s at %s\n",
        entry.Component,
        entry.Action,
        entry.Timestamp,
    )
}
```

**Use Cases**:
- Message tracing
- Audit trails
- Debugging
- Performance analysis

#### Smart Proxy
**File**: `eip_management.go:340-403`

Tracks message interactions with metrics.

```go
proxy := NewSmartProxy[Request, Response]("api-proxy",
    apiHandler,
    historyTracker,
)

result := proxy.Handle(ctx, requestMessage)

metrics := proxy.GetMetrics()
fmt.Printf("Total calls: %d, Avg duration: %v\n",
    metrics.TotalCalls,
    metrics.AvgDuration,
)
```

**Use Cases**:
- Performance monitoring
- Service metrics
- Call tracing
- SLA monitoring

#### Test Message Generator
**File**: `eip_management.go:405-439`

Generates test messages for testing.

```go
generator := NewTestMessageGenerator[TestData]("test-gen",
    func(sequence int) *Message[TestData] {
        return NewMessage(TestData{
            ID:       sequence,
            Payload:  fmt.Sprintf("test-%d", sequence),
        })
    },
)

// Generate single message
msg := generator.Generate()

// Generate batch
batch := generator.GenerateBatch(100)
```

**Use Cases**:
- Integration testing
- Load testing
- System validation
- Performance testing

#### Channel Purger
**File**: `eip_management.go:441-487`

Cleans up messages from channels.

```go
purger := NewChannelPurger[Order]("order-purger")

// Purge all messages
count := purger.Purge(ctx, orderChannel)

// Purge with predicate
count = purger.PurgeWithPredicate(ctx,
    orderChannel,
    cleanChannel,
    func(msg *Message[Order]) bool {
        return msg.IsExpired()
    },
)
```

**Use Cases**:
- Channel maintenance
- Expired message cleanup
- System reset
- Testing cleanup

#### Message Bridge
**File**: `eip_management.go:489-558`

Connects different messaging systems.

```go
bridge := NewMessageBridge[SourceFormat, TargetFormat]("system-bridge",
    sourceChannel,
    targetChannel,
    translatorFunc,
)

bridge.Start(ctx)
```

**Use Cases**:
- System integration
- Protocol bridging
- Legacy system connection
- Multi-cloud messaging

---

### 7. HTTP Integration Patterns

#### Scatter-Gather
**File**: `eip_http_adapter.go:288-354`

Sends to multiple recipients and aggregates responses.

```go
scatterGather := NewScatterGather[Request, Response]("api-scatter",
    aggregator,
    5*time.Second,
)

scatterGather.AddRecipient(api1Handler)
scatterGather.AddRecipient(api2Handler)
scatterGather.AddRecipient(api3Handler)

result := scatterGather.Execute(ctx, requestMessage)
```

**Use Cases**:
- Multi-API calls
- Price comparison
- Redundancy/failover
- Best-of-N results

## Pattern Summary by File

| File | Patterns | LOC |
|------|----------|-----|
| `eip_core.go` | Core abstractions (Message, Channel, Router, etc.) | ~470 |
| `eip_routing.go` | Content Router, Recipient List, Filter, Resequencer | ~328 |
| `eip_transformation.go` | Splitter, Aggregator, Enricher, Translator, Normalizer, Pipeline | ~431 |
| `eip_channels.go` | Pub-Sub, Dead Letter, Wire Tap, Priority, Guaranteed Delivery | ~514 |
| `eip_endpoints.go` | Idempotent Receiver, Competing Consumers, Service Activator | ~559 |
| `eip_management.go` | Control Bus, Process Manager, Routing Slip, History Tracker | ~558 |
| `eip_http_adapter.go` | HTTP adapters, Scatter-Gather, Pattern Builder | ~354 |

**Total EIP Implementation**: ~3,214 lines of code

## Complete Pattern List (65+ patterns)

### ✅ Fully Implemented (45 patterns)

**Routing (11)**:
- Content-Based Router
- Message Router
- Recipient List (Static & Dynamic)
- Splitter
- Aggregator
- Resequencer
- Message Filter
- Composed Message Processor
- Routing Slip
- Scatter-Gather
- Detour

**Transformation (7)**:
- Message Translator
- Content Enricher
- Content Filter
- Claim Check
- Normalizer
- Canonical Data Model
- Pipeline

**Channels (8)**:
- Point-to-Point Channel (InMemoryChannel)
- Publish-Subscribe Channel
- Dead Letter Channel
- Invalid Message Channel
- Guaranteed Delivery Channel
- Wire Tap
- Priority Channel
- Message Bridge

**Endpoints (9)**:
- Message Endpoint (base interface)
- Polling Consumer
- Event-Driven Consumer
- Competing Consumers
- Message Dispatcher
- Selective Consumer
- Idempotent Receiver
- Service Activator
- Transactional Client (via context)

**System Management (10)**:
- Control Bus
- Detour
- Wire Tap
- Message History
- Message Store
- Smart Proxy
- Test Message
- Channel Purger
- Process Manager
- Routing Slip Processor

### ⚡ Partially Implemented (via existing features)

- **Request-Reply**: Via Chain pattern + context sharing
- **Return Address**: Via Message.ReplyTo field
- **Correlation Identifier**: Via Message.CorrelationID
- **Message Expiration**: Via Message.ExpiresAt
- **Format Indicator**: Via headers

### 🔄 Can Be Built Using Existing Primitives

- **Channel Adapter**: Base interface provided, users implement
- **Messaging Gateway**: Can be built with Channel + Transformer
- **Messaging Mapper**: Via MessageTransformer
- **Durable Subscriber**: Via GuaranteedDeliveryChannel + EventDrivenConsumer

## Migration Guide

### From HTTP Orchestration to EIP

```go
// Old: HTTP-specific Chain
chain := orchestrator.Chain("user-flow").
    Step(loginRequest).
    OnSuccess(fetchProfileRequest).
    OnSuccess(updateSettingsRequest)

// New: EIP Pipeline with type safety
pipeline := NewPipeline[UserContext]("user-flow")
pipeline.AddStage("login", loginTransformer)
pipeline.AddStage("fetch-profile", profileTransformer)
pipeline.AddStage("update-settings", settingsTransformer)

result := pipeline.Process(ctx, userMessage)
```

### Mixing HTTP and EIP Patterns

```go
// Create HTTP adapter
httpAdapter := NewHTTPChannelAdapter("http", client)

// Create message from HTTP response
transformer := NewHTTPToMessageTransformer[User]("http-to-msg", jsonUnmarshaller)

// Route through EIP patterns
router := NewContentBasedRouter[User]("user-router")
router.AddRoute("premium", isPremiumPredicate, premiumChannel)
router.AddRoute("standard", isStandardPredicate, standardChannel)

// Bridge back to HTTP
httpBridge := NewHTTPOrchestrationBridge("bridge", outputChannel, orchestrator)
```

## Best Practices

1. **Type Safety**: Always use generics for compile-time type safety
2. **Context Propagation**: Pass `context.Context` through all operations
3. **Message History**: Use `AddHistoryEntry` for debugging and tracing
4. **Error Channels**: Configure error/dead-letter channels for resilience
5. **Idempotency**: Use IdempotentReceiver for critical operations
6. **Monitoring**: Leverage Wire Tap and Smart Proxy for observability
7. **Resource Cleanup**: Always call `Close()` on channels and `Stop()` on endpoints

## Performance Characteristics

- **In-Memory Channels**: O(1) send/receive, bounded by buffer size
- **Routers**: O(n) where n = number of routes
- **Aggregators**: O(1) per message, O(n) for aggregation
- **History Tracking**: O(1) append, O(n) retrieval
- **Idempotent Receiver**: O(1) duplicate check with map

## Testing Support

All patterns include:
- Unit test structure in `*_test.go` files
- Test message generators
- Channel purgers for cleanup
- Mock-friendly interfaces

## Future Enhancements

Potential additions:
- Persistent message stores (Redis, PostgreSQL)
- Distributed tracing integration (OpenTelemetry)
- Cloud messaging adapters (SQS, Kafka, RabbitMQ)
- GraphQL integration
- gRPC channel adapters
- WebSocket support

## References

- **Book**: "Enterprise Integration Patterns" by Gregor Hohpe and Bobby Woolf
- **Website**: https://www.enterpriseintegrationpatterns.com/
- **Pattern Catalog**: 65+ patterns for messaging and integration

---

**Espresso EIP Framework** - Complete, type-safe, Go-native implementation of Enterprise Integration Patterns
