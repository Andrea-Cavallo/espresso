package espresso

import (
	"context"
	"fmt"
)

// OrchestrationPatterns contiene metodi per pattern comuni di orchestrazione
type OrchestrationPatterns[T any] struct {
	client *Client[T]
}

// Patterns restituisce un'istanza di OrchestrationPatterns per accedere ai pattern di orchestrazione.
//
// Esempio:
//
//	client := espresso.NewDefaultClient[MyType]()
//	patterns := client.Patterns()
//	result, err := patterns.Chain().
//	    Call("step1", func(ctx map[string]any) espresso.RequestBuilder[MyType] {
//	        return client.Request("https://api.example.com/endpoint")
//	    }).
//	    Execute(context.Background())
func (c *Client[T]) Patterns() *OrchestrationPatterns[T] {
	return &OrchestrationPatterns[T]{client: c}
}

// ChainBuilder permette di creare catene di chiamate API sequenziali.
// Ogni chiamata nella catena viene eseguita solo se la precedente ha successo.
type ChainBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	lastStepName  string
}

// Chain crea una nuova catena di chiamate API sequenziali.
// Ogni chiamata viene eseguita solo se la precedente ha successo.
// Il risultato di ogni step viene salvato nel contesto con le chiavi:
// - "<nome>_success" (bool): indica se lo step è riuscito
// - "<nome>_result" (any): contiene i dati della risposta
// - "<nome>_error" (error): contiene l'errore in caso di fallimento
//
// Esempio:
//
//	client := espresso.NewDefaultClient[User]()
//	result, err := client.Patterns().Chain().
//	    Call("auth", func(ctx map[string]any) espresso.RequestBuilder[User] {
//	        return client.Request("https://api.example.com/auth").
//	            WithBody(map[string]string{"user": "john", "pass": "secret"})
//	    }).
//	    Call("fetch_profile", func(ctx map[string]any) espresso.RequestBuilder[User] {
//	        token := ctx["auth_result"].(AuthResponse).Token
//	        return client.Request("https://api.example.com/user/profile").
//	            WithBearerToken(token)
//	    }).
//	    OnSuccess(func(response *espresso.Response[User], ctx map[string]any) error {
//	        fmt.Printf("Profilo caricato: %+v\n", response.Data)
//	        return nil
//	    }).
//	    Execute(context.Background())
func (op *OrchestrationPatterns[T]) Chain() *ChainBuilder[T] {
	return &ChainBuilder[T]{
		orchestration: op.client.Orchestrate(),
	}
}

// Call aggiunge una chiamata alla catena.
// La chiamata viene eseguita solo se la precedente ha successo.
// Il parametro 'name' identifica lo step e viene usato per salvare il risultato nel contesto.
// Il parametro 'builderFunc' è una funzione che costruisce la richiesta HTTP utilizzando il contesto condiviso.
//
// Esempio:
//
//	chain.Call("create_order", func(ctx map[string]any) espresso.RequestBuilder[Order] {
//	    userID := ctx["auth_result"].(User).ID
//	    return client.Request("https://api.example.com/orders").
//	        WithBody(map[string]any{
//	            "user_id": userID,
//	            "amount": 100.0,
//	        })
//	})
func (cb *ChainBuilder[T]) Call(name string, builderFunc func(ctx map[string]any) RequestBuilder[T]) *ChainBuilder[T] {
	step := cb.orchestration.Step(name, builderFunc)
	if cb.lastStepName != "" {
		// Aggiungi condizione che il passo precedente sia riuscito
		step.When(func(ctx map[string]any) bool {
			return ctx[cb.lastStepName+"_success"] == true
		})
	}

	// Salva il risultato del successo nel context
	step.OnSuccess(func(response *Response[T], ctx map[string]any) error {
		ctx[name+"_success"] = true
		ctx[name+"_result"] = response.Data
		return nil
	}).OnFail(func(err error, ctx map[string]any) error {
		ctx[name+"_success"] = false
		ctx[name+"_error"] = err
		return nil // Non propagare per permettere la continuazione della catena
	}).End()

	cb.lastStepName = name
	return cb
}

// OnSuccess aggiunge un handler di successo per l'ultima chiamata aggiunta alla catena.
// Il callback riceve la risposta HTTP e il contesto condiviso.
// Restituire un errore interrompe la catena.
//
// Esempio:
//
//	chain.Call("fetch_data", builderFunc).
//	    OnSuccess(func(response *espresso.Response[Data], ctx map[string]any) error {
//	        fmt.Printf("Dati ricevuti: %+v\n", response.Data)
//	        ctx["custom_field"] = "custom_value"
//	        return nil
//	    })
func (cb *ChainBuilder[T]) OnSuccess(callback func(*Response[T], map[string]any) error) *ChainBuilder[T] {
	if cb.lastStepName != "" {
		// Modifica l'ultimo step aggiunto
		lastStepIndex := len(cb.orchestration.steps) - 1
		if lastStepIndex >= 0 {
			originalOnSuccess := cb.orchestration.steps[lastStepIndex].OnSuccess
			cb.orchestration.steps[lastStepIndex].OnSuccess = func(response *Response[T], ctx map[string]any) error {
				if originalOnSuccess != nil {
					if err := originalOnSuccess(response, ctx); err != nil {
						return err
					}
				}
				return callback(response, ctx)
			}
		}
	}
	return cb
}

// OnFail aggiunge un handler di fallimento per l'ultima chiamata aggiunta alla catena.
// Il callback riceve l'errore e il contesto condiviso.
// Restituire un errore interrompe la catena, restituire nil permette di continuare.
//
// Esempio:
//
//	chain.Call("risky_operation", builderFunc).
//	    OnFail(func(err error, ctx map[string]any) error {
//	        log.Printf("Operazione fallita: %v\n", err)
//	        ctx["fallback_used"] = true
//	        // Ritorna nil per continuare con un valore di fallback
//	        return nil
//	    })
func (cb *ChainBuilder[T]) OnFail(callback func(error, map[string]any) error) *ChainBuilder[T] {
	if cb.lastStepName != "" {
		lastStepIndex := len(cb.orchestration.steps) - 1
		if lastStepIndex >= 0 {
			originalOnFail := cb.orchestration.steps[lastStepIndex].OnFail
			cb.orchestration.steps[lastStepIndex].OnFail = func(err error, ctx map[string]any) error {
				if originalOnFail != nil {
					if failErr := originalOnFail(err, ctx); failErr != nil {
						return failErr
					}
				}
				return callback(err, ctx)
			}
		}
	}
	return cb
}

// Execute esegue la catena di chiamate in sequenza.
// Restituisce il risultato dell'orchestrazione che contiene i risultati di tutti gli step
// e il contesto condiviso con tutti i dati raccolti.
//
// Esempio:
//
//	result, err := chain.Execute(context.Background())
//	if err != nil {
//	    log.Fatalf("Catena fallita: %v", err)
//	}
//	for _, step := range result.Steps {
//	    fmt.Printf("Step %s: success=%v, duration=%v\n",
//	        step.Name, step.Success, step.Duration)
//	}
func (cb *ChainBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	return cb.orchestration.Execute(ctx)
}

// FanOutBuilder permette di eseguire chiamate in parallelo e aggregare i risultati.
// Tutte le chiamate vengono eseguite simultaneamente (fan-out) e i risultati possono
// essere aggregati tramite una funzione personalizzata.
type FanOutBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	aggregator    func([]StepResult, map[string]any) (any, error)
}

// FanOut crea un pattern fan-out per eseguire chiamate in parallelo.
// Tutte le chiamate aggiunte vengono eseguite simultaneamente.
// Ogni chiamata è opzionale per default, quindi il fallimento di una non blocca le altre.
//
// Esempio:
//
//	client := espresso.NewDefaultClient[any]()
//	result, err := client.Patterns().FanOut().
//	    Add("service1", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api1.example.com/data")
//	    }).
//	    Add("service2", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api2.example.com/data")
//	    }).
//	    Add("service3", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api3.example.com/data")
//	    }).
//	    WithAggregator(func(steps []espresso.StepResult, ctx map[string]any) (any, error) {
//	        results := make([]any, 0)
//	        for _, step := range steps {
//	            if step.Success {
//	                results = append(results, step.Data)
//	            }
//	        }
//	        return results, nil
//	    }).
//	    Execute(context.Background())
func (op *OrchestrationPatterns[T]) FanOut() *FanOutBuilder[T] {
	orchestration := op.client.Orchestrate().Parallel()
	return &FanOutBuilder[T]{
		orchestration: orchestration,
	}
}

// Add aggiunge una chiamata al fan-out.
// Tutte le chiamate vengono eseguite in parallelo.
// Il risultato viene salvato nel contesto con la chiave "<nome>_result".
//
// Esempio:
//
//	fanOut.Add("weather_api", func(ctx map[string]any) espresso.RequestBuilder[Weather] {
//	    return client.Request("https://api.weather.com/current").
//	        WithQuery("city", "Rome")
//	})
func (fb *FanOutBuilder[T]) Add(name string, builderFunc func(ctx map[string]any) RequestBuilder[T]) *FanOutBuilder[T] {
	fb.orchestration.Step(name, builderFunc).
		Optional(). // Tutti gli step sono opzionali nel fan-out
		SaveToContext(name + "_result").
		End()
	return fb
}

// WithAggregator imposta una funzione per aggregare i risultati di tutte le chiamate parallele.
// La funzione riceve tutti i risultati degli step e il contesto condiviso.
// Il risultato aggregato viene salvato nel contesto con la chiave "aggregated_result".
//
// Esempio:
//
//	fanOut.WithAggregator(func(steps []espresso.StepResult, ctx map[string]any) (any, error) {
//	    // Conta i successi e fallimenti
//	    successes := 0
//	    failures := 0
//	    for _, step := range steps {
//	        if step.Success {
//	            successes++
//	        } else {
//	            failures++
//	        }
//	    }
//	    return map[string]int{
//	        "successes": successes,
//	        "failures": failures,
//	    }, nil
//	})
func (fb *FanOutBuilder[T]) WithAggregator(aggregator func([]StepResult, map[string]any) (any, error)) *FanOutBuilder[T] {
	fb.aggregator = aggregator
	return fb
}

// Execute esegue tutte le chiamate in parallelo e applica l'aggregatore se definito.
// Restituisce il risultato dell'orchestrazione con tutti i risultati individuali e
// il risultato aggregato (se l'aggregatore è stato impostato).
//
// Esempio:
//
//	result, err := fanOut.Execute(context.Background())
//	if err != nil {
//	    log.Fatalf("FanOut fallito: %v", err)
//	}
//	aggregated := result.Context["aggregated_result"]
//	fmt.Printf("Risultato aggregato: %+v\n", aggregated)
func (fb *FanOutBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	result, err := fb.orchestration.Execute(ctx)
	if err != nil {
		return result, err
	}

	// Applica aggregazione se definita
	if fb.aggregator != nil {
		aggregated, aggErr := fb.aggregator(result.Steps, result.Context)
		if aggErr != nil {
			return result, aggErr
		}
		result.Context["aggregated_result"] = aggregated
	}

	return result, nil
}

// SagaBuilder implementa il pattern Saga per gestire transazioni distribuite con compensazioni.
// Ogni step può avere una funzione di compensazione che viene eseguita in caso di fallimento
// di uno step successivo, permettendo il rollback di operazioni già completate.
type SagaBuilder[T any] struct {
	orchestration   *OrchestrationBuilder[T]
	compensations   map[string]func(map[string]any) error
	compensateOrder []string
}

// Saga crea un pattern saga per gestire transazioni distribuite.
// Ogni step può avere una funzione di compensazione che viene eseguita in ordine inverso
// se uno step successivo fallisce, implementando così un meccanismo di rollback distribuito.
//
// Esempio:
//
//	client := espresso.NewDefaultClient[Order]()
//	result, err := client.Patterns().Saga().
//	    Step("create_order", func(ctx map[string]any) espresso.RequestBuilder[Order] {
//	        return client.Request("https://api.example.com/orders").
//	            WithBody(orderData)
//	    }, func(ctx map[string]any) error {
//	        // Compensazione: cancella l'ordine
//	        orderID := ctx["create_order_result"].(Order).ID
//	        fmt.Printf("Cancellazione ordine %s\n", orderID)
//	        return nil
//	    }).
//	    Step("charge_payment", func(ctx map[string]any) espresso.RequestBuilder[Order] {
//	        order := ctx["create_order_result"].(Order)
//	        return client.Request("https://api.example.com/payments").
//	            WithBody(map[string]any{"order_id": order.ID, "amount": order.Total})
//	    }, func(ctx map[string]any) error {
//	        // Compensazione: rimborsa il pagamento
//	        fmt.Println("Rimborso pagamento")
//	        return nil
//	    }).
//	    Execute(context.Background())
func (op *OrchestrationPatterns[T]) Saga() *SagaBuilder[T] {
	return &SagaBuilder[T]{
		orchestration:   op.client.Orchestrate(),
		compensations:   make(map[string]func(map[string]any) error),
		compensateOrder: make([]string, 0),
	}
}

// Step aggiunge uno step alla saga con la sua funzione di compensazione.
// Se lo step ha successo, il risultato viene salvato nel contesto e la compensazione
// viene registrata. Se lo step fallisce, tutte le compensazioni degli step precedenti
// vengono eseguite in ordine inverso (LIFO).
//
// Parametri:
// - name: nome identificativo dello step
// - builderFunc: funzione che costruisce la richiesta HTTP
// - compensation: funzione di compensazione eseguita in caso di rollback (può essere nil)
//
// Esempio:
//
//	saga.Step("reserve_inventory", func(ctx map[string]any) espresso.RequestBuilder[Inventory] {
//	    return client.Request("https://api.example.com/inventory/reserve").
//	        WithBody(map[string]any{"product_id": 123, "quantity": 5})
//	}, func(ctx map[string]any) error {
//	    // Compensazione: rilascia la prenotazione
//	    reservation := ctx["reserve_inventory_result"].(Reservation)
//	    log.Printf("Rilascio prenotazione %s\n", reservation.ID)
//	    // Chiama API per rilasciare
//	    return nil
//	})
func (sb *SagaBuilder[T]) Step(name string, builderFunc func(ctx map[string]any) RequestBuilder[T], compensation func(map[string]any) error) *SagaBuilder[T] {
	step := sb.orchestration.Step(name, builderFunc)

	step.OnSuccess(func(response *Response[T], ctx map[string]any) error {
		ctx[name+"_completed"] = true
		ctx[name+"_result"] = response.Data
		return nil
	}).OnFail(func(err error, ctx map[string]any) error {
		// Se questo step fallisce, esegui tutte le compensazioni dei step precedenti
		return sb.executeCompensations(ctx)
	}).End()

	if compensation != nil {
		sb.compensations[name] = compensation
		sb.compensateOrder = append([]string{name}, sb.compensateOrder...) // Ordine inverso
	}

	return sb
}

// executeCompensations esegue le compensazioni in ordine inverso (LIFO).
// Viene chiamata automaticamente quando uno step fallisce.
func (sb *SagaBuilder[T]) executeCompensations(ctx map[string]any) error {
	for _, stepName := range sb.compensateOrder {
		if ctx[stepName+"_completed"] == true {
			if compensation, exists := sb.compensations[stepName]; exists {
				if err := compensation(ctx); err != nil {
					return fmt.Errorf("compensation failed for step %s: %w", stepName, err)
				}
			}
		}
	}
	return nil
}

// Execute esegue la saga.
// Se tutti gli step hanno successo, restituisce il risultato completo.
// Se uno step fallisce, esegue automaticamente le compensazioni in ordine inverso.
//
// Esempio:
//
//	result, err := saga.Execute(context.Background())
//	if err != nil {
//	    log.Printf("Saga fallita e rollback eseguito: %v", err)
//	} else {
//	    log.Printf("Saga completata con successo in %v", result.Duration)
//	}
func (sb *SagaBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	return sb.orchestration.Execute(ctx)
}

// ConditionalBuilder permette di creare branching condizionale nell'orchestrazione.
// Gli step vengono eseguiti in base a condizioni valutate sul contesto condiviso.
type ConditionalBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	conditions    []ConditionalBranch[T]
}

// ConditionalBranch rappresenta un ramo condizionale con la sua condizione e lo step da eseguire.
type ConditionalBranch[T any] struct {
	Condition   func(map[string]any) bool
	StepName    string
	BuilderFunc func(map[string]any) RequestBuilder[T]
}

// Conditional crea un builder per implementare logica condizionale (if-then-else).
// Permette di eseguire step diversi in base a condizioni valutate sul contesto.
//
// Esempio:
//
//	client := espresso.NewDefaultClient[any]()
//	result, err := client.Patterns().Conditional().
//	    WithContext(map[string]any{
//	        "user_type": "premium",
//	        "credit": 100,
//	    }).
//	    When(func(ctx map[string]any) bool {
//	        return ctx["user_type"] == "premium"
//	    }, "premium_endpoint", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/premium/data")
//	    }).
//	    When(func(ctx map[string]any) bool {
//	        return ctx["credit"].(int) > 50
//	    }, "credit_check", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/verify-credit")
//	    }).
//	    Else("default_endpoint", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/basic/data")
//	    }).
//	    Execute(context.Background())
func (op *OrchestrationPatterns[T]) Conditional() *ConditionalBuilder[T] {
	return &ConditionalBuilder[T]{
		orchestration: op.client.Orchestrate(),
		conditions:    make([]ConditionalBranch[T], 0),
	}
}

// WithContext imposta il contesto iniziale per la logica condizionale.
// I valori nel contesto vengono usati per valutare le condizioni.
//
// Esempio:
//
//	conditional.WithContext(map[string]any{
//	    "environment": "production",
//	    "feature_flag_x": true,
//	    "user_id": 12345,
//	})
func (cb *ConditionalBuilder[T]) WithContext(ctx map[string]any) *ConditionalBuilder[T] {
	cb.orchestration.WithContext(ctx)
	return cb
}

// When aggiunge un ramo condizionale (if).
// Lo step viene eseguito solo se la condizione restituisce true.
//
// Parametri:
// - condition: funzione che valuta una condizione sul contesto
// - stepName: nome identificativo dello step
// - builderFunc: funzione che costruisce la richiesta HTTP
//
// Esempio:
//
//	conditional.When(func(ctx map[string]any) bool {
//	    return ctx["environment"] == "production"
//	}, "prod_api", func(ctx map[string]any) espresso.RequestBuilder[Data] {
//	    return client.Request("https://prod-api.example.com/data")
//	})
func (cb *ConditionalBuilder[T]) When(condition func(map[string]any) bool, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ConditionalBuilder[T] {
	branch := ConditionalBranch[T]{
		Condition:   condition,
		StepName:    stepName,
		BuilderFunc: builderFunc,
	}
	cb.conditions = append(cb.conditions, branch)

	// Aggiungi lo step con la condizione
	cb.orchestration.Step(stepName, builderFunc).
		When(condition).
		End()

	return cb
}

// Else aggiunge un ramo di default che si esegue se nessun altro ramo When è stato eseguito.
//
// Esempio:
//
//	conditional.
//	    When(condition1, "step1", builder1).
//	    When(condition2, "step2", builder2).
//	    Else("fallback", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/fallback")
//	    })
func (cb *ConditionalBuilder[T]) Else(stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ConditionalBuilder[T] {
	// Il ramo else si esegue solo se nessun altro si è eseguito
	elseCondition := func(ctx map[string]any) bool {
		for _, branch := range cb.conditions {
			if branch.Condition(ctx) {
				return false // Un altro ramo si eseguirà
			}
		}
		return true
	}

	cb.orchestration.Step(stepName, builderFunc).
		When(elseCondition).
		End()

	return cb
}

// Execute esegue la logica condizionale.
// Solo gli step le cui condizioni sono soddisfatte vengono eseguiti.
//
// Esempio:
//
//	result, err := conditional.Execute(context.Background())
//	if err != nil {
//	    log.Fatalf("Orchestrazione condizionale fallita: %v", err)
//	}
//	for _, step := range result.Steps {
//	    if !step.Skipped {
//	        fmt.Printf("Step eseguito: %s\n", step.Name)
//	    }
//	}
func (cb *ConditionalBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	return cb.orchestration.Execute(ctx)
}

// RateLimitedBuilder applica rate limiting a livello di orchestrazione.
// Ogni step verifica il rate limiter prima di essere eseguito.
type RateLimitedBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	limiter       RateLimiter
	key           string
}

// RateLimited crea un'orchestrazione con rate limiting.
// Ogni step aggiunto verifica il rate limiter prima dell'esecuzione.
// Se il limite è superato, lo step viene saltato.
//
// Esempio:
//
//	// Assumendo di avere un rate limiter implementato
//	limiter := myRateLimiter // implementazione di espresso.RateLimiter
//	result, err := client.Patterns().RateLimited(limiter, "api_key_123").
//	    Step("call1", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/endpoint1")
//	    }).End().
//	    Step("call2", func(ctx map[string]any) espresso.RequestBuilder[any] {
//	        return client.Request("https://api.example.com/endpoint2")
//	    }).End().
//	    Execute(context.Background())
func (op *OrchestrationPatterns[T]) RateLimited(limiter RateLimiter, key string) *RateLimitedBuilder[T] {
	return &RateLimitedBuilder[T]{
		orchestration: op.client.Orchestrate(),
		limiter:       limiter,
		key:           key,
	}
}

// Step aggiunge uno step con rate limiting.
// Prima dell'esecuzione, lo step verifica il rate limiter.
// Se il limite è superato, viene salvato nel contesto "<nome>_rate_limited" = true
// e lo step viene saltato.
//
// Esempio:
//
//	rateLimited.Step("api_call", func(ctx map[string]any) espresso.RequestBuilder[Data] {
//	    return client.Request("https://api.example.com/data")
//	}).End()
func (rlb *RateLimitedBuilder[T]) Step(name string, builderFunc func(ctx map[string]any) RequestBuilder[T]) *StepBuilder[T] {
	// Wrapper che applica rate limiting
	wrappedBuilder := func(ctx map[string]any) RequestBuilder[T] {
		return builderFunc(ctx)
	}

	step := rlb.orchestration.Step(name, wrappedBuilder)

	// Aggiungi controllo rate limiting
	originalCondition := step.step.Condition
	step.When(func(ctx map[string]any) bool {
		// Controlla rate limiting
		if !rlb.limiter.Allow(rlb.key) {
			ctx[name+"_rate_limited"] = true
			return false
		}

		// Controlla condizione originale se presente
		if originalCondition != nil {
			return originalCondition(ctx)
		}
		return true
	})

	return step
}

// Execute esegue l'orchestrazione con rate limiting applicato a tutti gli step.
//
// Esempio:
//
//	result, err := rateLimited.Execute(context.Background())
//	if err != nil {
//	    log.Fatalf("Orchestrazione fallita: %v", err)
//	}
//	// Controlla quali step sono stati limitati
//	for stepName := range result.Context {
//	    if strings.HasSuffix(stepName, "_rate_limited") {
//	        fmt.Printf("Step limitato: %s\n", strings.TrimSuffix(stepName, "_rate_limited"))
//	    }
//	}
func (rlb *RateLimitedBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	return rlb.orchestration.Execute(ctx)
}

// BatchBuilder permette di eseguire batch di richieste simili applicando lo stesso template.
// Utile per operazioni bulk dove si deve eseguire la stessa chiamata su un set di dati.
type BatchBuilder[T any] struct {
	client    *Client[T]
	template  func(item any, ctx map[string]any) RequestBuilder[T]
	items     []any
	batchSize int
	parallel  bool
}

// Batch crea un builder per eseguire batch di richieste simili.
// Per default, le richieste vengono eseguite in parallelo con batch size di 10.
//
// Esempio:
//
//	client := espresso.NewDefaultClient[User]()
//	userIDs := []any{1, 2, 3, 4, 5}
//	result, err := client.Patterns().Batch().
//	    WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[User] {
//	        userID := item.(int)
//	        return client.Request(fmt.Sprintf("https://api.example.com/users/%d", userID))
//	    }).
//	    WithItems(userIDs).
//	    WithBatchSize(2). // Processa 2 alla volta
//	    Execute(context.Background())
//	
//	// Accedi ai risultati
//	for i, step := range result.Steps {
//	    if step.Success {
//	        fmt.Printf("User %d caricato con successo\n", userIDs[i])
//	    }
//	}
func (op *OrchestrationPatterns[T]) Batch() *BatchBuilder[T] {
	return &BatchBuilder[T]{
		client:    op.client,
		batchSize: 10,
		parallel:  true,
	}
}

// WithTemplate imposta il template per costruire le richieste del batch.
// Il template riceve ogni item della lista e il contesto condiviso.
//
// Esempio:
//
//	batch.WithTemplate(func(item any, ctx map[string]any) espresso.RequestBuilder[Product] {
//	    productID := item.(string)
//	    apiKey := ctx["api_key"].(string)
//	    return client.Request(fmt.Sprintf("https://api.example.com/products/%s", productID)).
//	        WithHeader("X-API-Key", apiKey)
//	})
func (bb *BatchBuilder[T]) WithTemplate(template func(item any, ctx map[string]any) RequestBuilder[T]) *BatchBuilder[T] {
	bb.template = template
	return bb
}

// WithItems imposta gli elementi da processare nel batch.
// Ogni elemento viene passato alla funzione template per costruire la richiesta.
//
// Esempio:
//
//	orderIDs := []any{"ORD001", "ORD002", "ORD003"}
//	batch.WithItems(orderIDs)
func (bb *BatchBuilder[T]) WithItems(items []any) *BatchBuilder[T] {
	bb.items = items
	return bb
}

// WithBatchSize imposta la dimensione del batch (numero di richieste per batch).
// Utile per controllare il carico e il parallelismo.
//
// Esempio:
//
//	batch.WithBatchSize(5) // Processa 5 elementi alla volta
func (bb *BatchBuilder[T]) WithBatchSize(size int) *BatchBuilder[T] {
	bb.batchSize = size
	return bb
}

// Sequential imposta l'esecuzione sequenziale invece che parallela.
// Utile quando l'ordine è importante o per limitare il carico sul server.
//
// Esempio:
//
//	batch.Sequential() // Esegui una richiesta alla volta
func (bb *BatchBuilder[T]) Sequential() *BatchBuilder[T] {
	bb.parallel = false
	return bb
}

// Execute esegue il batch di richieste.
// Ogni item genera uno step nell'orchestrazione con nome "batch_item_<indice>".
// I risultati sono salvati nel contesto con chiave "batch_item_<indice>_result".
//
// Esempio:
//
//	result, err := batch.Execute(context.Background())
//	if err != nil {
//	    log.Fatalf("Batch fallito: %v", err)
//	}
//	
//	successCount := 0
//	for _, step := range result.Steps {
//	    if step.Success {
//	        successCount++
//	    }
//	}
//	fmt.Printf
