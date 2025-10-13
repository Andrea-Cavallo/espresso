package espresso

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OrchestrationResult contiene il risultato di un'orchestrazione
type OrchestrationResult[T any] struct {
	Steps    []StepResult
	Success  bool
	Error    error
	Duration time.Duration
	Context  map[string]any
}

// StepResult rappresenta il risultato di un singolo step
type StepResult struct {
	Name       string
	Success    bool
	StatusCode int
	Error      error
	Duration   time.Duration
	Data       any
	Skipped    bool
}

// OrchestrationBuilder fornisce un'API fluente per orchestrare chiamate API
type OrchestrationBuilder[T any] struct {
	client     *Client[T]
	steps      []OrchestrationStep[T]
	context    map[string]any
	timeout    time.Duration
	onComplete func(*OrchestrationResult[T])
	onError    func(error, *OrchestrationResult[T])
	parallel   bool
}

// OrchestrationStep rappresenta un singolo step nell'orchestrazione
type OrchestrationStep[T any] struct {
	Name        string
	Builder     func(ctx map[string]any) RequestBuilder[T]
	Condition   func(ctx map[string]any) bool
	OnSuccess   func(*Response[T], map[string]any) error
	OnFail      func(error, map[string]any) error
	OnStatus    map[int]func(*Response[T], map[string]any) error
	Transform   func(*Response[T], map[string]any) (any, error)
	Required    bool
	Timeout     time.Duration
	RetryConfig *RetryConfig
}

// Orchestrate crea un nuovo OrchestrationBuilder dal client
func (c *Client[T]) Orchestrate() *OrchestrationBuilder[T] {
	return &OrchestrationBuilder[T]{
		client:  c,
		steps:   make([]OrchestrationStep[T], 0),
		context: make(map[string]any),
		timeout: 30 * time.Second,
	}
}

// Step aggiunge un nuovo step all'orchestrazione
func (ob *OrchestrationBuilder[T]) Step(name string, builderFunc func(ctx map[string]any) RequestBuilder[T]) *StepBuilder[T] {
	step := OrchestrationStep[T]{
		Name:     name,
		Builder:  builderFunc,
		OnStatus: make(map[int]func(*Response[T], map[string]any) error),
		Required: true,
	}

	return &StepBuilder[T]{
		orchestration: ob,
		step:          &step,
	}
}

// Then aggiunge un step condizionale che si esegue dopo il successo del precedente
func (ob *OrchestrationBuilder[T]) Then(name string, builderFunc func(ctx map[string]any) RequestBuilder[T]) *StepBuilder[T] {
	return ob.Step(name, builderFunc).When(func(ctx map[string]any) bool {
		if len(ob.steps) == 0 {
			return true
		}
		return true
	})
}

// OnComplete imposta un callback da eseguire al completamento dell'orchestrazione
func (ob *OrchestrationBuilder[T]) OnComplete(callback func(*OrchestrationResult[T])) *OrchestrationBuilder[T] {
	ob.onComplete = callback
	return ob
}

// OnError imposta un callback da eseguire in caso di errore
func (ob *OrchestrationBuilder[T]) OnError(callback func(error, *OrchestrationResult[T])) *OrchestrationBuilder[T] {
	ob.onError = callback
	return ob
}

// WithTimeout imposta il timeout globale per l'orchestrazione
func (ob *OrchestrationBuilder[T]) WithTimeout(timeout time.Duration) *OrchestrationBuilder[T] {
	ob.timeout = timeout
	return ob
}

// WithContext imposta il context iniziale
func (ob *OrchestrationBuilder[T]) WithContext(ctx map[string]any) *OrchestrationBuilder[T] {
	for k, v := range ctx {
		ob.context[k] = v
	}
	return ob
}

// Parallel abilita l'esecuzione parallela degli step
func (ob *OrchestrationBuilder[T]) Parallel() *OrchestrationBuilder[T] {
	ob.parallel = true
	return ob
}

// Execute esegue l'orchestrazione
func (ob *OrchestrationBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	start := time.Now()

	result := &OrchestrationResult[T]{
		Steps:   make([]StepResult, 0),
		Context: make(map[string]any),
		Success: true,
	}

	for k, v := range ob.context {
		result.Context[k] = v
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, ob.timeout)
	defer cancel()

	var err error
	if ob.parallel {
		err = ob.executeParallel(timeoutCtx, result)
	} else {
		err = ob.executeSequential(timeoutCtx, result)
	}

	result.Duration = time.Since(start)
	result.Error = err
	result.Success = err == nil

	if err != nil && ob.onError != nil {
		ob.onError(err, result)
	}
	if ob.onComplete != nil {
		ob.onComplete(result)
	}

	return result, err
}

// executeSequential esegue gli step in sequenza
func (ob *OrchestrationBuilder[T]) executeSequential(ctx context.Context, result *OrchestrationResult[T]) error {
	for _, step := range ob.steps {
		stepResult, err := ob.executeStep(ctx, step, result.Context)
		result.Steps = append(result.Steps, stepResult)

		if err != nil && step.Required {
			return fmt.Errorf("required step '%s' failed: %w", step.Name, err)
		}

		if err != nil {
			continue
		}
	}
	return nil
}

// executeParallel esegue gli step in parallelo
func (ob *OrchestrationBuilder[T]) executeParallel(ctx context.Context, result *OrchestrationResult[T]) error {
	var wg sync.WaitGroup
	resultsChan := make(chan StepResult, len(ob.steps))
	errorsChan := make(chan error, len(ob.steps))

	for _, step := range ob.steps {
		wg.Add(1)
		go func(s OrchestrationStep[T]) {
			defer wg.Done()

			stepResult, err := ob.executeStep(ctx, s, result.Context)
			resultsChan <- stepResult

			if err != nil && s.Required {
				errorsChan <- fmt.Errorf("required step '%s' failed: %w", s.Name, err)
			}
		}(step)
	}

	wg.Wait()
	close(resultsChan)
	close(errorsChan)

	for stepResult := range resultsChan {
		result.Steps = append(result.Steps, stepResult)
	}

	for err := range errorsChan {
		return err
	}

	return nil
}

// executeStep esegue un singolo step
func (ob *OrchestrationBuilder[T]) executeStep(ctx context.Context, step OrchestrationStep[T], sharedContext map[string]any) (StepResult, error) {
	start := time.Now()

	stepResult := StepResult{
		Name: step.Name,
	}

	if step.Condition != nil && !step.Condition(sharedContext) {
		stepResult.Skipped = true
		stepResult.Success = true
		stepResult.Duration = time.Since(start)
		return stepResult, nil
	}

	builder := step.Builder(sharedContext)

	if step.Timeout > 0 {
		builder = builder.WithTimeout(step.Timeout)
	}
	if step.RetryConfig != nil {
		builder = builder.WithRetryConfig(step.RetryConfig)
	}

	response, err := builder.Get(ctx)

	stepResult.Duration = time.Since(start)

	if err != nil {
		stepResult.Error = err
		stepResult.Success = false

		if step.OnFail != nil {
			if failErr := step.OnFail(err, sharedContext); failErr != nil {
				return stepResult, failErr
			}
		}

		return stepResult, err
	}

	stepResult.Success = true
	stepResult.StatusCode = response.StatusCode
	stepResult.Data = response.Data

	if handler, exists := step.OnStatus[response.StatusCode]; exists {
		if err := handler(response, sharedContext); err != nil {
			stepResult.Success = false
			stepResult.Error = err
			return stepResult, err
		}
	}

	if step.OnSuccess != nil {
		if err := step.OnSuccess(response, sharedContext); err != nil {
			stepResult.Success = false
			stepResult.Error = err
			return stepResult, err
		}
	}

	if step.Transform != nil {
		transformed, err := step.Transform(response, sharedContext)
		if err != nil {
			stepResult.Success = false
			stepResult.Error = err
			return stepResult, err
		}
		stepResult.Data = transformed
	}

	return stepResult, nil
}

// StepBuilder fornisce un'API fluente per configurare un singolo step
type StepBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	step          *OrchestrationStep[T]
}

// When imposta una condizione per l'esecuzione dello step
func (sb *StepBuilder[T]) When(condition func(ctx map[string]any) bool) *StepBuilder[T] {
	sb.step.Condition = condition
	return sb
}

// OnSuccess imposta un callback da eseguire in caso di successo
func (sb *StepBuilder[T]) OnSuccess(callback func(*Response[T], map[string]any) error) *StepBuilder[T] {
	sb.step.OnSuccess = callback
	return sb
}

// OnFail imposta un callback da eseguire in caso di fallimento
func (sb *StepBuilder[T]) OnFail(callback func(error, map[string]any) error) *StepBuilder[T] {
	sb.step.OnFail = callback
	return sb
}

// OnStatusCode imposta un callback per uno specifico status code
func (sb *StepBuilder[T]) OnStatusCode(statusCode int, callback func(*Response[T], map[string]any) error) *StepBuilder[T] {
	sb.step.OnStatus[statusCode] = callback
	return sb
}

// OnStatus imposta callback per multiple status code
func (sb *StepBuilder[T]) OnStatus(handlers map[int]func(*Response[T], map[string]any) error) *StepBuilder[T] {
	for code, handler := range handlers {
		sb.step.OnStatus[code] = handler
	}
	return sb
}

// Transform imposta una funzione di trasformazione per il risultato
func (sb *StepBuilder[T]) Transform(transformer func(*Response[T], map[string]any) (any, error)) *StepBuilder[T] {
	sb.step.Transform = transformer
	return sb
}

// Required indica se lo step è obbligatorio
func (sb *StepBuilder[T]) Required(required bool) *StepBuilder[T] {
	sb.step.Required = required
	return sb
}

// Optional rende lo step opzionale
func (sb *StepBuilder[T]) Optional() *StepBuilder[T] {
	sb.step.Required = false
	return sb
}

// WithTimeout imposta un timeout specifico per questo step
func (sb *StepBuilder[T]) WithTimeout(timeout time.Duration) *StepBuilder[T] {
	sb.step.Timeout = timeout
	return sb
}

// WithRetry imposta una configurazione di retry specifica per questo step
func (sb *StepBuilder[T]) WithRetry(config *RetryConfig) *StepBuilder[T] {
	sb.step.RetryConfig = config
	return sb
}

// End termina la configurazione dello step e ritorna all'orchestration builder
func (sb *StepBuilder[T]) End() *OrchestrationBuilder[T] {
	sb.orchestration.steps = append(sb.orchestration.steps, *sb.step)
	return sb.orchestration
}

// SaveToContext salva il risultato nel context condiviso
func (sb *StepBuilder[T]) SaveToContext(key string) *StepBuilder[T] {
	return sb.OnSuccess(func(response *Response[T], ctx map[string]any) error {
		if response.Data != nil {
			ctx[key] = *response.Data
		}
		return nil
	})
}

// SaveFieldToContext salva un campo specifico nel context
func (sb *StepBuilder[T]) SaveFieldToContext(contextKey string, extractor func(*Response[T]) any) *StepBuilder[T] {
	return sb.OnSuccess(func(response *Response[T], ctx map[string]any) error {
		ctx[contextKey] = extractor(response)
		return nil
	})
}

// FailOn fa fallire lo step per specifici status code
func (sb *StepBuilder[T]) FailOn(statusCodes ...int) *StepBuilder[T] {
	for _, code := range statusCodes {
		sb.OnStatusCode(code, func(response *Response[T], ctx map[string]any) error {
			return fmt.Errorf("step failed with status code %d", code)
		})
	}
	return sb
}

// RetryOn configura retry per specifici status code
func (sb *StepBuilder[T]) RetryOn(statusCodes []int, maxAttempts int, delay time.Duration) *StepBuilder[T] {
	config := &RetryConfig{
		MaxAttempts:     maxAttempts,
		BaseDelay:       delay,
		MaxDelay:        delay * 10,
		BackoffStrategy: BackoffExponential,
		RetriableStatus: statusCodes,
	}
	return sb.WithRetry(config)
}

// Log aggiunge logging automatico al step
func (sb *StepBuilder[T]) Log(logger Logger) *StepBuilder[T] {
	sb.OnSuccess(func(response *Response[T], ctx map[string]any) error {
		logger.Info("Step succeeded", "step", sb.step.Name, "status", response.StatusCode)
		return nil
	})
	sb.OnFail(func(err error, ctx map[string]any) error {
		logger.Error("Step failed", "step", sb.step.Name, "error", err)
		return nil
	})
	return sb
}
