package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

// Domain models
type Order struct {
	ID          string
	CustomerID  string
	Items       []OrderItem
	TotalAmount float64
	Priority    string
	Status      string
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

type OrderConfirmation struct {
	OrderID     string
	Status      string
	ProcessedAt time.Time
}

type InventoryReservation struct {
	OrderID   string
	ProductID string
	Quantity  int
	Reserved  bool
}

// Example 1: Content-Based Router
func exampleContentBasedRouter(ctx context.Context) {
	fmt.Println("\n=== Example 1: Content-Based Router ===")

	// Create channels for different priority levels
	highPriorityChannel := espresso.NewInMemoryChannel[Order]("high-priority", 10)
	standardPriorityChannel := espresso.NewInMemoryChannel[Order]("standard-priority", 10)
	lowPriorityChannel := espresso.NewInMemoryChannel[Order]("low-priority", 10)

	// Create content-based router
	router := espresso.NewContentBasedRouter[Order]("order-router")
	router.AddRoute("high", func(msg *espresso.Message[Order]) bool {
		return msg.Body.Priority == "high"
	}, highPriorityChannel)
	router.AddRoute("standard", func(msg *espresso.Message[Order]) bool {
		return msg.Body.Priority == "standard"
	}, standardPriorityChannel)
	router.SetDefaultChannel(lowPriorityChannel)

	// Create and route messages
	orders := []Order{
		{ID: "1", Priority: "high", TotalAmount: 1000},
		{ID: "2", Priority: "standard", TotalAmount: 500},
		{ID: "3", Priority: "low", TotalAmount: 100},
	}

	for _, order := range orders {
		msg := espresso.NewMessage(order)
		err := router.RouteAndSend(ctx, msg)
		if err != nil {
			fmt.Printf("Error routing order %s: %v\n", order.ID, err)
		} else {
			fmt.Printf("Order %s routed to %s priority channel\n", order.ID, order.Priority)
		}
	}

	// Receive and process messages
	if msg, err := highPriorityChannel.Receive(ctx); err == nil {
		fmt.Printf("Processed high priority order: %s (Amount: $%.2f)\n",
			msg.Body.ID, msg.Body.TotalAmount)
	}
}

// Example 2: Splitter and Aggregator
func exampleSplitterAggregator(ctx context.Context) {
	fmt.Println("\n=== Example 2: Splitter and Aggregator ===")

	// Create channels
	itemChannel := espresso.NewInMemoryChannel[OrderItem]("items", 100)
	resultChannel := espresso.NewInMemoryChannel[InventoryReservation]("reservations", 100)

	// Create splitter to split order into items
	splitter := espresso.NewSplitter[Order, OrderItem]("order-splitter",
		func(ctx context.Context, msg *espresso.Message[Order]) ([]*espresso.Message[OrderItem], error) {
			var items []*espresso.Message[OrderItem]
			for _, item := range msg.Body.Items {
				itemMsg := espresso.NewMessage(item)
				itemMsg.CorrelationID = msg.ID
				items = append(items, itemMsg)
			}
			return items, nil
		},
		itemChannel,
	)

	// Create order with items
	order := Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Items: []OrderItem{
			{ProductID: "prod-1", Quantity: 2, Price: 10.0},
			{ProductID: "prod-2", Quantity: 1, Price: 20.0},
			{ProductID: "prod-3", Quantity: 3, Price: 5.0},
		},
	}

	orderMsg := espresso.NewMessage(order)

	// Split the order
	err := splitter.ProcessAndSend(ctx, orderMsg)
	if err != nil {
		fmt.Printf("Error splitting order: %v\n", err)
		return
	}

	fmt.Printf("Split order %s into %d items\n", order.ID, len(order.Items))

	// Process each item (simulated inventory reservation)
	for i := 0; i < len(order.Items); i++ {
		item, err := itemChannel.Receive(ctx)
		if err != nil {
			continue
		}

		reservation := InventoryReservation{
			OrderID:   orderMsg.ID,
			ProductID: item.Body.ProductID,
			Quantity:  item.Body.Quantity,
			Reserved:  true,
		}
		reservationMsg := espresso.NewMessage(reservation)
		reservationMsg.CorrelationID = item.CorrelationID
		resultChannel.Send(ctx, reservationMsg)

		fmt.Printf("  Reserved %d units of %s\n", item.Body.Quantity, item.Body.ProductID)
	}

	// Create aggregator to combine reservations
	aggregator := espresso.NewAggregator[InventoryReservation, OrderConfirmation](
		espresso.AggregatorConfig[InventoryReservation, OrderConfirmation]{
			Name: "reservation-aggregator",
			CompletionFunc: func(messages []*espresso.Message[InventoryReservation]) bool {
				return len(messages) >= 3
			},
			AggregateFunc: func(ctx context.Context, messages []*espresso.Message[InventoryReservation]) (*espresso.Message[OrderConfirmation], error) {
				allReserved := true
				for _, msg := range messages {
					if !msg.Body.Reserved {
						allReserved = false
						break
					}
				}

				status := "confirmed"
				if !allReserved {
					status = "failed"
				}

				confirmation := OrderConfirmation{
					OrderID:     messages[0].Body.OrderID,
					Status:      status,
					ProcessedAt: time.Now(),
				}

				return espresso.NewMessage(confirmation), nil
			},
			Timeout:       10 * time.Second,
			OutputChannel: nil,
		},
	)

	// Aggregate reservations
	for i := 0; i < len(order.Items); i++ {
		reservation, _ := resultChannel.Receive(ctx)
		aggregator.Aggregate(ctx, reservation)
	}

	fmt.Printf("Order aggregated successfully\n")
}

// Example 3: Pub-Sub Pattern
func examplePubSub(ctx context.Context) {
	fmt.Println("\n=== Example 3: Publish-Subscribe ===")

	// Create subscriber channels
	analyticsChannel := espresso.NewInMemoryChannel[Order]("analytics", 10)
	loggingChannel := espresso.NewInMemoryChannel[Order]("logging", 10)
	metricsChannel := espresso.NewInMemoryChannel[Order]("metrics", 10)

	// Create pub-sub channel
	pubsub := espresso.NewPubSubChannel[Order]("order-events")

	// Subscribe channels
	pubsub.Subscribe(analyticsChannel)
	pubsub.Subscribe(loggingChannel)
	pubsub.Subscribe(metricsChannel)

	// Publish an event
	order := Order{
		ID:          "order-789",
		CustomerID:  "customer-123",
		TotalAmount: 250.0,
		Status:      "completed",
	}

	orderMsg := espresso.NewMessage(order)
	err := pubsub.Send(ctx, orderMsg)
	if err != nil {
		fmt.Printf("Error publishing: %v\n", err)
		return
	}

	fmt.Printf("Published order event to %d subscribers\n", pubsub.SubscriberCount())

	// Each subscriber receives the message
	if msg, err := analyticsChannel.Receive(ctx); err == nil {
		fmt.Printf("  Analytics: Received order %s ($%.2f)\n", msg.Body.ID, msg.Body.TotalAmount)
	}

	if msg, err := loggingChannel.Receive(ctx); err == nil {
		fmt.Printf("  Logging: Received order %s (Status: %s)\n", msg.Body.ID, msg.Body.Status)
	}

	if msg, err := metricsChannel.Receive(ctx); err == nil {
		fmt.Printf("  Metrics: Received order %s\n", msg.Body.ID)
	}
}

// Example 4: Dead Letter Channel
func exampleDeadLetterChannel(ctx context.Context) {
	fmt.Println("\n=== Example 4: Dead Letter Channel ===")

	mainChannel := espresso.NewInMemoryChannel[Order]("orders", 10)
	dlc := espresso.NewDeadLetterChannel[Order]("order-processing", mainChannel, 10)

	// Simulate processing with errors
	orders := []Order{
		{ID: "1", Status: "valid"},
		{ID: "2", Status: "invalid"},
		{ID: "3", Status: "valid"},
	}

	for _, order := range orders {
		msg := espresso.NewMessage(order)
		dlc.Send(ctx, msg)
	}

	// Process messages
	for i := 0; i < 3; i++ {
		msg, err := dlc.Receive(ctx)
		if err != nil {
			continue
		}

		if msg.Body.Status == "invalid" {
			// Send to DLQ
			dlc.SendToDLQ(ctx, msg, fmt.Errorf("invalid order status"))
			fmt.Printf("Order %s sent to DLQ (invalid status)\n", msg.Body.ID)
		} else {
			fmt.Printf("Order %s processed successfully\n", msg.Body.ID)
		}
	}

	// Check DLQ stats
	stats := dlc.GetDLQStats()
	fmt.Printf("DLQ contains %d failed messages\n", stats.TotalMessages)
}

// Example 5: Content Enricher
func exampleContentEnricher(ctx context.Context) {
	fmt.Println("\n=== Example 5: Content Enricher ===")

	// Simulated customer database
	customerDB := map[string]string{
		"customer-123": "Premium",
		"customer-456": "Standard",
	}

	// Create enricher
	enricher := espresso.NewContentEnricher[Order]("order-enricher",
		func(ctx context.Context, msg *espresso.Message[Order]) (*espresso.Message[Order], error) {
			// Enrich with customer tier
			tier := customerDB[msg.Body.CustomerID]
			msg.SetProperty("customer_tier", tier)

			// Calculate discount based on tier
			discount := 0.0
			if tier == "Premium" {
				discount = 0.10
			}
			msg.SetProperty("discount", discount)

			fmt.Printf("Enriched order %s: Customer Tier=%s, Discount=%.0f%%\n",
				msg.Body.ID, tier, discount*100)

			return msg, nil
		},
	)

	// Create and enrich order
	order := Order{
		ID:          "order-999",
		CustomerID:  "customer-123",
		TotalAmount: 100.0,
	}

	orderMsg := espresso.NewMessage(order)
	enriched, err := enricher.Enrich(ctx, orderMsg)
	if err != nil {
		fmt.Printf("Error enriching: %v\n", err)
		return
	}

	tier, _ := enriched.GetProperty("customer_tier")
	discount, _ := enriched.GetProperty("discount")
	fmt.Printf("Final order: Amount=$%.2f, Tier=%s, Discount=%.0f%%\n",
		enriched.Body.TotalAmount, tier, discount.(float64)*100)
}

// Example 6: Message Translator
func exampleMessageTranslator(ctx context.Context) {
	fmt.Println("\n=== Example 6: Message Translator ===")

	type OrderDTO struct {
		OrderNumber string  `json:"order_number"`
		Total       float64 `json:"total"`
		Priority    int     `json:"priority"`
	}

	// Create translator
	translator := espresso.NewMessageTranslator[OrderDTO, Order]("dto-to-order",
		func(ctx context.Context, msg *espresso.Message[OrderDTO]) (*espresso.Message[Order], error) {
			priority := "standard"
			if msg.Body.Priority == 1 {
				priority = "high"
			} else if msg.Body.Priority == 3 {
				priority = "low"
			}

			order := Order{
				ID:          msg.Body.OrderNumber,
				TotalAmount: msg.Body.Total,
				Priority:    priority,
			}

			return espresso.NewMessage(order), nil
		},
	)

	// Translate DTO to Order
	dto := OrderDTO{
		OrderNumber: "ORD-12345",
		Total:       150.50,
		Priority:    1,
	}

	dtoMsg := espresso.NewMessage(dto)
	orderMsg, err := translator.Transform(ctx, dtoMsg)
	if err != nil {
		fmt.Printf("Error translating: %v\n", err)
		return
	}

	fmt.Printf("Translated: DTO{%s, $%.2f, priority=%d} -> Order{%s, $%.2f, priority=%s}\n",
		dto.OrderNumber, dto.Total, dto.Priority,
		orderMsg.Body.ID, orderMsg.Body.TotalAmount, orderMsg.Body.Priority)
}

// Example 7: Idempotent Receiver
func exampleIdempotentReceiver(ctx context.Context) {
	fmt.Println("\n=== Example 7: Idempotent Receiver ===")

	receiver := espresso.NewIdempotentReceiver[Order]("payment-processor", 1*time.Hour)

	handler := espresso.MessageHandlerFunc[Order, Order](
		func(ctx context.Context, msg *espresso.Message[Order]) (*espresso.Message[Order], error) {
			fmt.Printf("  Processing payment for order %s\n", msg.Body.ID)
			return msg, nil
		},
	)

	// Process same message multiple times
	order := Order{ID: "order-pay-123", TotalAmount: 99.99}
	orderMsg := espresso.NewMessage(order)
	orderMsg.ID = "msg-123" // Fixed ID to demonstrate idempotency

	fmt.Println("First attempt:")
	receiver.Receive(ctx, orderMsg, handler)

	fmt.Println("Second attempt (duplicate):")
	receiver.Receive(ctx, orderMsg, handler)

	fmt.Println("Third attempt (duplicate):")
	receiver.Receive(ctx, orderMsg, handler)

	fmt.Printf("Message processed only once (idempotent)\n")
}

// Example 8: Wire Tap
func exampleWireTap(ctx context.Context) {
	fmt.Println("\n=== Example 8: Wire Tap ===")

	mainChannel := espresso.NewInMemoryChannel[Order]("orders", 10)
	auditChannel := espresso.NewInMemoryChannel[Order]("audit", 10)

	wireTap := espresso.NewWireTap[Order]("order-audit", auditChannel)
	tappedChannel := espresso.NewWireTapChannel[Order]("orders-tapped", mainChannel, wireTap)

	// Send order through tapped channel
	order := Order{ID: "order-tap-1", TotalAmount: 500.0}
	orderMsg := espresso.NewMessage(order)

	tappedChannel.Send(ctx, orderMsg)

	fmt.Printf("Sent order %s through wire tap\n", order.ID)

	// Process from main channel
	mainMsg, _ := mainChannel.Receive(ctx)
	fmt.Printf("  Main channel received: Order %s\n", mainMsg.Body.ID)

	// Audit channel also receives (non-blocking)
	time.Sleep(100 * time.Millisecond)
	auditMsg, ok, _ := auditChannel.TryReceive(ctx)
	if ok {
		fmt.Printf("  Audit channel received: Order %s\n", auditMsg.Body.ID)
	}
}

// Example 9: Process Manager
func exampleProcessManager(ctx context.Context) {
	fmt.Println("\n=== Example 9: Process Manager ===")

	type OrderEvent struct {
		Type    string
		OrderID string
		Details string
	}

	pm := espresso.NewProcessManager[OrderEvent]("order-fulfillment")

	// Start a process
	orderID := "order-pm-123"
	initialEvent := OrderEvent{Type: "created", OrderID: orderID, Details: "New order"}
	initialMsg := espresso.NewMessage(initialEvent)

	pm.StartProcess(ctx, orderID, initialMsg)
	fmt.Printf("Started process for order %s\n", orderID)

	// Handle events
	events := []OrderEvent{
		{Type: "payment_received", OrderID: orderID, Details: "Payment confirmed"},
		{Type: "inventory_reserved", OrderID: orderID, Details: "Items reserved"},
		{Type: "shipped", OrderID: orderID, Details: "Package shipped"},
	}

	for _, event := range events {
		pm.UpdateProcessStep(orderID, event.Type)
		pm.HandleMessage(ctx, orderID, espresso.NewMessage(event))
		fmt.Printf("  Step: %s - %s\n", event.Type, event.Details)
	}

	// Get process state
	state, _ := pm.GetProcessState(orderID)
	fmt.Printf("Process state: %s (completed steps: %d)\n", state.Status, len(state.CompletedSteps))

	// Complete process
	pm.CompleteProcess(ctx, orderID)
	finalState, _ := pm.GetProcessState(orderID)
	fmt.Printf("Final state: %s\n", finalState.Status)
}

// Example 10: Complete EIP Pipeline
func exampleCompletePipeline(ctx context.Context) {
	fmt.Println("\n=== Example 10: Complete EIP Pipeline ===")

	// Step 1: Validation transformer
	validateTransformer := espresso.MessageTransformerFunc[Order, Order](
		func(ctx context.Context, msg *espresso.Message[Order]) (*espresso.Message[Order], error) {
			if msg.Body.TotalAmount <= 0 {
				return nil, fmt.Errorf("invalid amount")
			}
			fmt.Printf("  ✓ Validated order %s\n", msg.Body.ID)
			return msg, nil
		},
	)

	// Step 2: Enrichment transformer
	enrichTransformer := espresso.MessageTransformerFunc[Order, Order](
		func(ctx context.Context, msg *espresso.Message[Order]) (*espresso.Message[Order], error) {
			msg.SetProperty("processing_timestamp", time.Now())
			msg.SetProperty("processed_by", "pipeline-v1")
			fmt.Printf("  ✓ Enriched order %s\n", msg.Body.ID)
			return msg, nil
		},
	)

	// Step 3: Persistence transformer
	persistTransformer := espresso.MessageTransformerFunc[Order, Order](
		func(ctx context.Context, msg *espresso.Message[Order]) (*espresso.Message[Order], error) {
			// Simulate database save
			data, _ := json.Marshal(msg.Body)
			fmt.Printf("  ✓ Persisted order %s: %s\n", msg.Body.ID, string(data))
			return msg, nil
		},
	)

	// Create pipeline
	pipeline := espresso.NewPipeline[Order]("order-processing-pipeline")
	pipeline.AddStage("validate", validateTransformer)
	pipeline.AddStage("enrich", enrichTransformer)
	pipeline.AddStage("persist", persistTransformer)

	// Process order through pipeline
	order := Order{
		ID:          "order-pipe-1",
		CustomerID:  "customer-789",
		TotalAmount: 299.99,
		Priority:    "high",
	}

	orderMsg := espresso.NewMessage(order)
	fmt.Printf("Processing order %s through pipeline...\n", order.ID)

	result, err := pipeline.Process(ctx, orderMsg)
	if err != nil {
		fmt.Printf("Pipeline error: %v\n", err)
		return
	}

	fmt.Printf("✓ Pipeline complete! Order %s processed through %d stages\n",
		result.Body.ID, len(result.History))

	// Print message history
	fmt.Println("\nMessage History:")
	for i, entry := range result.History {
		fmt.Printf("  %d. %s: %s at %s\n",
			i+1, entry.Component, entry.Action, entry.Timestamp.Format("15:04:05"))
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  Espresso EIP Patterns - Comprehensive Examples          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")

	// Run all examples
	exampleContentBasedRouter(ctx)
	exampleSplitterAggregator(ctx)
	examplePubSub(ctx)
	exampleDeadLetterChannel(ctx)
	exampleContentEnricher(ctx)
	exampleMessageTranslator(ctx)
	exampleIdempotentReceiver(ctx)
	exampleWireTap(ctx)
	exampleProcessManager(ctx)
	exampleCompletePipeline(ctx)

	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║  All EIP pattern examples completed successfully!        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
}
