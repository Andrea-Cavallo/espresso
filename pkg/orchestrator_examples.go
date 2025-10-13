package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"espresso/pkg/espresso"
)

// AuthResponse Strutture di esempio per gli esempi di orchestrazione
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

type UserProfile struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Settings struct {
		Notifications bool   `json:"notifications"`
		Theme         string `json:"theme"`
	} `json:"settings"`
}

type OrderRequest struct {
	UserID    int     `json:"user_id"`
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Amount    float64 `json:"amount"`
}

type OrderResponse struct {
	OrderID   string  `json:"order_id"`
	Status    string  `json:"status"`
	Total     float64 `json:"total"`
	CreatedAt string  `json:"created_at"`
}

type PaymentRequest struct {
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
	Method    string  `json:"method"`
	CardToken string  `json:"card_token"`
}

type PaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Orchestrazione API Examples (Mock Version) ===\n")

	// Esempi base di orchestrazione con simulazione
	mockBasicOrchestrationExample(ctx)
	mockChainOrchestrationExample(ctx)

	// Esempi di pattern avanzati
	mockConditionalOrchestrationExample(ctx)
	mockSagaPatternExample(ctx)

	// Esempi pratici
	mockE2EWorkflowExample(ctx)
	mockErrorHandlingExample(ctx)

	fmt.Println("\n=== Mock Orchestration Examples completed ===")
}

// mockBasicOrchestrationExample dimostra orchestrazione base con simulazioni
func mockBasicOrchestrationExample(ctx context.Context) {
	fmt.Println("--- Mock Basic Orchestration Example ---")

	client := espresso.NewDefaultClient[any]()

	// Orchestrazione: Simula Auth -> Get Profile -> Update Settings
	result, err := client.Orchestrate().
		Step("simulate_auth", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula una chiamata di autenticazione
			return client.Request("https://httpbin.org/json").
				WithHeader("Content-Type", "application/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			// Simula l'estrazione del token
			ctx["auth_token"] = "mock_token_12345"
			fmt.Printf("✓ Authentication successful, token: %s\n", ctx["auth_token"])
			return nil
		}).
		OnFail(func(err error, ctx map[string]any) error {
			fmt.Printf("✗ Authentication failed: %v\n", err)
			// Non propagare l'errore per continuare con mock data
			ctx["auth_token"] = "fallback_token"
			return nil
		}).
		End().
		Step("simulate_profile", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula caricamento profilo
			return client.Request("https://httpbin.org/json").
				WithBearerToken(ctx["auth_token"].(string))
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			// Simula dati profilo
			ctx["user_profile"] = map[string]any{
				"id":    123,
				"name":  "John Doe",
				"email": "john@example.com",
			}
			fmt.Printf("✓ Profile loaded successfully: %v\n", ctx["user_profile"])
			return nil
		}).
		SaveToContext("profile_data").
		End().
		Step("simulate_settings", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula aggiornamento impostazioni
			return client.Request("https://httpbin.org/json").
				WithBearerToken(ctx["auth_token"].(string)).
				WithBody(map[string]any{
					"notifications": true,
					"theme":         "dark",
				})
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			fmt.Printf("✓ Settings updated successfully\n")
			ctx["settings_updated"] = true
			return nil
		}).
		End().
		WithTimeout(30 * time.Second).
		Execute(ctx)

	if err != nil {
		log.Printf("Orchestration failed: %v", err)
		return
	}

	fmt.Printf("🎉 Orchestration completed in %v with %d steps\n", result.Duration, len(result.Steps))
	for _, step := range result.Steps {
		status := "SUCCESS"
		if !step.Success {
			status = "FAILED"
		}
		if step.Skipped {
			status = "SKIPPED"
		}
		fmt.Printf("  - %s: %s (duration: %v)\n", step.Name, status, step.Duration)
	}
	fmt.Println()
}

// mockChainOrchestrationExample dimostra il pattern Chain con simulazioni
func mockChainOrchestrationExample(ctx context.Context) {
	fmt.Println("--- Mock Chain Orchestration Example ---")

	client := espresso.NewDefaultClient[any]()

	// Chain: Validate User -> Create Order -> Process Payment -> Send Confirmation
	result, err := client.Patterns().Chain().
		Call("validate_user", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula validazione utente
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			ctx["user_valid"] = true
			ctx["user_id"] = 123
			fmt.Println("✓ User validation successful")
			return nil
		}).
		Call("create_order", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula creazione ordine
			return client.Request("https://httpbin.org/json").
				WithBody(OrderRequest{
					UserID:    ctx["user_id"].(int),
					ProductID: 456,
					Quantity:  2,
					Amount:    99.99,
				})
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			// Simula risposta ordine
			orderData := map[string]any{
				"order_id": "ORDER_789",
				"total":    99.99,
				"status":   "created",
			}
			ctx["order_id"] = orderData["order_id"]
			ctx["total_amount"] = orderData["total"]
			fmt.Printf("✓ Order created: %s\n", orderData["order_id"])
			return nil
		}).
		Call("process_payment", func(ctx map[string]any) espresso.RequestBuilder[any] {
			orderID := ctx["order_id"].(string)
			amount := ctx["total_amount"].(float64)

			// Simula processamento pagamento
			return client.Request("https://httpbin.org/json").
				WithBody(PaymentRequest{
					OrderID:   orderID,
					Amount:    amount,
					Method:    "credit_card",
					CardToken: "card_token_123",
				})
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			// Simula risposta pagamento
			paymentData := map[string]any{
				"payment_id":     "PAY_ABC123",
				"status":         "completed",
				"transaction_id": "TXN_DEF456",
			}
			ctx["payment_id"] = paymentData["payment_id"]
			fmt.Printf("✓ Payment processed: %s\n", paymentData["payment_id"])
			return nil
		}).
		Call("send_confirmation", func(ctx map[string]any) espresso.RequestBuilder[any] {
			orderID := ctx["order_id"].(string)
			// Simula invio conferma
			return client.Request("https://httpbin.org/json").
				WithBody(map[string]any{
					"type":     "order_confirmation",
					"order_id": orderID,
					"user_id":  ctx["user_id"],
				})
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			fmt.Println("✓ Confirmation sent successfully")
			ctx["confirmation_sent"] = true
			return nil
		}).
		Execute(ctx)

	if err != nil {
		log.Printf("Chain failed: %v", err)
		return
	}

	fmt.Printf("🎉 Chain completed successfully in %v\n", result.Duration)
	fmt.Printf("📊 Order ID: %v, Payment ID: %v\n",
		result.Context["order_id"],
		result.Context["payment_id"])
	fmt.Println()
}

// mockConditionalOrchestrationExample dimostra orchestrazione condizionale
func mockConditionalOrchestrationExample(ctx context.Context) {
	fmt.Println("--- Mock Conditional Orchestration Example ---")

	client := espresso.NewDefaultClient[any]()

	// Simuliamo un context con dati utente
	initialContext := map[string]any{
		"user_type":    "premium",
		"order_amount": 150.0,
		"country":      "US",
	}

	result, err := client.Patterns().Conditional().
		WithContext(initialContext).
		When(
			func(ctx map[string]any) bool {
				userType, userTypeOk := ctx["user_type"].(string)
				orderAmount, amountOk := ctx["order_amount"].(float64)
				return userTypeOk && amountOk && userType == "premium" && orderAmount > 100
			},
			"apply_premium_discount",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				return client.Request("https://httpbin.org/json").
					WithBody(map[string]any{
						"user_id": 123,
						"amount":  ctx["order_amount"],
					})
			}).
		When(
			func(ctx map[string]any) bool {
				country, ok := ctx["country"].(string)
				return ok && country == "US"
			},
			"calculate_tax_us",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				amount := ctx["order_amount"].(float64)
				return client.Request("https://httpbin.org/json").
					WithQuery("amount", fmt.Sprintf("%.2f", amount))
			}).
		When(
			func(ctx map[string]any) bool {
				country, ok := ctx["country"].(string)
				return ok && country == "EU"
			},
			"calculate_tax_eu",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				amount := ctx["order_amount"].(float64)
				return client.Request("https://httpbin.org/json").
					WithQuery("amount", fmt.Sprintf("%.2f", amount))
			}).
		Else("calculate_tax_default",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				amount := ctx["order_amount"].(float64)
				return client.Request("https://httpbin.org/json").
					WithQuery("amount", fmt.Sprintf("%.2f", amount))
			}).
		Execute(ctx)

	if err != nil {
		log.Printf("Conditional orchestration failed: %v", err)
		return
	}

	fmt.Printf("🎉 Conditional orchestration completed in %v\n", result.Duration)
	fmt.Printf("📋 Initial context: user_type=%s, amount=%.2f, country=%s\n",
		initialContext["user_type"],
		initialContext["order_amount"],
		initialContext["country"])

	for _, step := range result.Steps {
		if !step.Skipped {
			status := "SUCCESS"
			if !step.Success {
				status = "FAILED"
			}
			fmt.Printf("  ✓ Executed: %s (%s)\n", step.Name, status)
		} else {
			fmt.Printf("  ⏭ Skipped: %s\n", step.Name)
		}
	}
	fmt.Println()
}

// mockSagaPatternExample dimostra il pattern Saga con simulazioni
func mockSagaPatternExample(ctx context.Context) {
	fmt.Println("--- Mock Saga Pattern Example ---")

	client := espresso.NewDefaultClient[any]()

	// Simuliamo un fallimento nel terzo step per vedere le compensazioni
	stepCount := 0

	result, err := client.Patterns().Saga().
		Step("reserve_inventory",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				return client.Request("https://httpbin.org/json").
					WithBody(map[string]any{
						"product_id": 456,
						"quantity":   2,
					})
			},
			// Compensazione: rilascia l'inventario
			func(ctx map[string]any) error {
				fmt.Println("🔄 COMPENSATING: Releasing inventory reservation")
				return nil
			}).
		Step("create_order",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				return client.Request("https://httpbin.org/json").
					WithBody(OrderRequest{
						UserID:    123,
						ProductID: 456,
						Quantity:  2,
						Amount:    99.99,
					})
			},
			// Compensazione: cancella l'ordine
			func(ctx map[string]any) error {
				fmt.Println("🔄 COMPENSATING: Cancelling order")
				return nil
			}).
		Step("process_payment",
			func(ctx map[string]any) espresso.RequestBuilder[any] {
				stepCount++
				// Simula un fallimento nel pagamento
				if stepCount >= 3 {
					// Forza un fallimento per dimostrare le compensazioni
					return client.Request("https://httpbin.org/status/500")
				}
				return client.Request("https://httpbin.org/json")
			},
			// Compensazione: rimborsa il pagamento
			func(ctx map[string]any) error {
				fmt.Println("🔄 COMPENSATING: Refunding payment")
				return nil
			}).
		Execute(ctx)

	if err != nil {
		fmt.Printf("⚠️ Saga failed as expected: %v\n", err)
		fmt.Println("✓ Compensations were executed to rollback the transaction")
	} else {
		fmt.Printf("🎉 Saga completed successfully in %v\n", result.Duration)
	}

	fmt.Printf("📊 Execution summary:\n")
	for _, step := range result.Steps {
		status := "SUCCESS"
		if !step.Success {
			status = "FAILED"
		}
		fmt.Printf("  - %s: %s (duration: %v)\n", step.Name, status, step.Duration)
	}
	fmt.Println()
}

// mockE2EWorkflowExample dimostra un workflow end-to-end realistico
func mockE2EWorkflowExample(ctx context.Context) {
	fmt.Println("--- Mock E2E Workflow Example ---")

	client := espresso.NewDefaultClient[any]()

	// Workflow: E-commerce checkout completo
	result, err := client.Orchestrate().
		Step("validate_cart", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			ctx["cart_valid"] = true
			ctx["cart_total"] = 199.99
			ctx["items_count"] = 3
			fmt.Printf("✓ Cart validated: %d items, total: $%.2f\n",
				ctx["items_count"], ctx["cart_total"])
			return nil
		}).
		End().
		Step("check_inventory", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		When(func(ctx map[string]any) bool {
			return ctx["cart_valid"] == true
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			ctx["inventory_available"] = true
			fmt.Printf("✓ Inventory check passed\n")
			return nil
		}).
		End().
		Step("apply_promotions", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			originalTotal := ctx["cart_total"].(float64)
			discount := originalTotal * 0.10 // 10% sconto
			newTotal := originalTotal - discount
			ctx["discount_applied"] = discount
			ctx["final_total"] = newTotal
			fmt.Printf("✓ Promotion applied: $%.2f discount, new total: $%.2f\n",
				discount, newTotal)
			return nil
		}).
		End().
		Step("calculate_shipping", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			shippingCost := 9.99
			finalTotal := ctx["final_total"].(float64) + shippingCost
			ctx["shipping_cost"] = shippingCost
			ctx["total_with_shipping"] = finalTotal
			fmt.Printf("✓ Shipping calculated: $%.2f, total: $%.2f\n",
				shippingCost, finalTotal)
			return nil
		}).
		End().
		Step("process_order", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			orderID := fmt.Sprintf("ORD_%d", time.Now().Unix())
			ctx["order_id"] = orderID
			ctx["order_status"] = "confirmed"
			fmt.Printf("✓ Order processed: %s\n", orderID)
			return nil
		}).
		End().
		WithTimeout(60 * time.Second).
		Execute(ctx)

	if err != nil {
		log.Printf("E2E Workflow failed: %v", err)
		return
	}

	fmt.Printf("🎉 E2E Workflow completed successfully in %v\n", result.Duration)
	fmt.Printf("📋 Final summary:\n")
	fmt.Printf("  Order ID: %v\n", result.Context["order_id"])
	fmt.Printf("  Final Total: $%.2f\n", result.Context["total_with_shipping"])
	fmt.Printf("  Discount: $%.2f\n", result.Context["discount_applied"])
	fmt.Printf("  Steps executed: %d\n", len(result.Steps))
	fmt.Println()
}

// mockErrorHandlingExample dimostra gestione degli errori
func mockErrorHandlingExample(ctx context.Context) {
	fmt.Println("--- Mock Error Handling Example ---")

	client := espresso.NewDefaultClient[any]()

	// Workflow con gestione errori avanzata
	result, err := client.Orchestrate().
		Step("risky_operation", func(ctx map[string]any) espresso.RequestBuilder[any] {
			// Simula un'operazione che potrebbe fallire
			return client.Request("https://httpbin.org/status/500")
		}).
		OnFail(func(err error, ctx map[string]any) error {
			fmt.Printf("⚠️ Risky operation failed: %v\n", err)
			ctx["fallback_used"] = true
			return nil // Non propagare l'errore
		}).
		Optional(). // Rendi questo step opzionale
		End().
		Step("fallback_operation", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		When(func(ctx map[string]any) bool {
			return ctx["fallback_used"] == true
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			fmt.Printf("✓ Fallback operation successful\n")
			ctx["operation_completed"] = true
			return nil
		}).
		End().
		Step("final_step", func(ctx map[string]any) espresso.RequestBuilder[any] {
			return client.Request("https://httpbin.org/json")
		}).
		OnSuccess(func(response *espresso.Response[any], ctx map[string]any) error {
			fmt.Printf("✓ Final step completed\n")
			return nil
		}).
		End().
		OnComplete(func(result *espresso.OrchestrationResult[any]) {
			fmt.Printf("🔔 Orchestration completed callback triggered\n")
		}).
		OnError(func(err error, result *espresso.OrchestrationResult[any]) {
			fmt.Printf("🚨 Orchestration error callback: %v\n", err)
		}).
		Execute(ctx)

	if err != nil {
		log.Printf("Error Handling example failed: %v", err)
		return
	}

	fmt.Printf("🎉 Error Handling example completed in %v\n", result.Duration)
	fmt.Printf("📊 Error handling summary:\n")
	for _, step := range result.Steps {
		status := "SUCCESS"
		if !step.Success {
			status = "FAILED"
		}
		if step.Skipped {
			status = "SKIPPED"
		}
		fmt.Printf("  - %s: %s\n", step.Name, status)
	}
	fmt.Printf("  Fallback used: %v\n", result.Context["fallback_used"])
	fmt.Printf("  Operation completed: %v\n", result.Context["operation_completed"])
	fmt.Println()
}
