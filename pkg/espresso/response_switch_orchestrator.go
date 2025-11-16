package espresso

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// ResponseSwitchBuilder permette di creare branching basato sui contenuti della response
type ResponseSwitchBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	responseKey   string
	cases         []ResponseCase[T]
	defaultCase   *ResponseCase[T]
}

// ResponseCase rappresenta un caso nel response switch
type ResponseCase[T any] struct {
	Condition   ResponseCondition
	StepName    string
	BuilderFunc func(map[string]any) RequestBuilder[T]
	OnExecute   func(*Response[T], map[string]any) error
}

// ResponseCondition definisce la condizione da verificare sulla response
type ResponseCondition struct {
	Field    string         // Campo da controllare nella response
	Operator Operator       // Operatore di confronto
	Value    any            // Valore da confrontare
	Custom   func(any) bool // Funzione personalizzata di verifica
}

// Operator definisce gli operatori di confronto
type Operator int

const (
	OpEquals Operator = iota
	OpNotEquals
	OpContains
	OpGreaterThan
	OpLessThan
	OpGreaterOrEqual
	OpLessOrEqual
	OpIsNull
	OpIsNotNull
	OpIn
	OpNotIn
	OpMatches // per regex
)

// ResponseSwitch crea un nuovo ResponseSwitchBuilder
func (sb *StepBuilder[T]) ResponseSwitch(responseKey string) *ResponseSwitchBuilder[T] {
	// Modifica lo step per salvare la response nel context
	originalOnSuccess := sb.step.OnSuccess
	sb.step.OnSuccess = func(response *Response[T], ctx map[string]any) error {
		if originalOnSuccess != nil {
			if err := originalOnSuccess(response, ctx); err != nil {
				return err
			}
		}
		// Salva la response per il branching
		ctx[responseKey] = response
		return nil
	}

	// Termina questo step e crea il response switch
	sb.orchestration.steps = append(sb.orchestration.steps, *sb.step)

	return &ResponseSwitchBuilder[T]{
		orchestration: sb.orchestration,
		responseKey:   responseKey,
		cases:         make([]ResponseCase[T], 0),
	}
}

// Case aggiunge un caso al response switch
func (rsb *ResponseSwitchBuilder[T]) Case(condition ResponseCondition, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	case_ := ResponseCase[T]{
		Condition:   condition,
		StepName:    stepName,
		BuilderFunc: builderFunc,
	}
	rsb.cases = append(rsb.cases, case_)
	return rsb
}

// OnField helper per creare condizioni sui campi
func (rsb *ResponseSwitchBuilder[T]) OnField(field string, operator Operator, value any, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	return rsb.Case(ResponseCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	}, stepName, builderFunc)
}

// OnValue helper per verificare un valore specifico
func (rsb *ResponseSwitchBuilder[T]) OnValue(field string, value any, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	return rsb.OnField(field, OpEquals, value, stepName, builderFunc)
}

// OnStatus helper per verificare status code
func (rsb *ResponseSwitchBuilder[T]) OnStatus(statusCode int, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	return rsb.Case(ResponseCondition{
		Custom: func(responseData any) bool {
			if response, ok := responseData.(*Response[T]); ok {
				return response.StatusCode == statusCode
			}
			return false
		},
	}, stepName, builderFunc)
}

// OnCustom permette logica personalizzata
func (rsb *ResponseSwitchBuilder[T]) OnCustom(condition func(any) bool, stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	return rsb.Case(ResponseCondition{
		Custom: condition,
	}, stepName, builderFunc)
}

// Default aggiunge un caso di default
func (rsb *ResponseSwitchBuilder[T]) Default(stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *ResponseSwitchBuilder[T] {
	rsb.defaultCase = &ResponseCase[T]{
		StepName:    stepName,
		BuilderFunc: builderFunc,
	}
	return rsb
}

// End completa il response switch e aggiunge gli step condizionali
func (rsb *ResponseSwitchBuilder[T]) End() *OrchestrationBuilder[T] {
	// Aggiungi tutti i casi come step condizionali
	for _, case_ := range rsb.cases {
		rsb.orchestration.Step(case_.StepName, case_.BuilderFunc).
			When(func(ctx map[string]any) bool {
				responseData := ctx[rsb.responseKey]
				return rsb.evaluateCondition(case_.Condition, responseData)
			}).
			OnSuccess(case_.OnExecute).
			End()
	}

	// Aggiungi il caso default se presente
	if rsb.defaultCase != nil {
		rsb.orchestration.Step(rsb.defaultCase.StepName, rsb.defaultCase.BuilderFunc).
			When(func(ctx map[string]any) bool {
				responseData := ctx[rsb.responseKey]
				// Esegui il default solo se nessun altro caso è stato soddisfatto
				for _, case_ := range rsb.cases {
					if rsb.evaluateCondition(case_.Condition, responseData) {
						return false
					}
				}
				return true
			}).
			OnSuccess(rsb.defaultCase.OnExecute).
			End()
	}

	return rsb.orchestration
}

// evaluateCondition valuta una condizione sulla response
func (rsb *ResponseSwitchBuilder[T]) evaluateCondition(condition ResponseCondition, responseData any) bool {
	// Se c'è una funzione personalizzata, usala
	if condition.Custom != nil {
		return condition.Custom(responseData)
	}

	// Estrai i dati dalla response
	var data any
	if response, ok := responseData.(*Response[T]); ok {
		if response.Data != nil {
			data = *response.Data
		}
	}

	if data == nil {
		return condition.Operator == OpIsNull
	}

	// Estrai il campo specificato
	fieldValue := rsb.extractField(data, condition.Field)

	// Valuta l'operatore
	return rsb.evaluateOperator(fieldValue, condition.Operator, condition.Value)
}

// extractField estrae un campo dai dati (supporta nested fields con notazione dot)
func (rsb *ResponseSwitchBuilder[T]) extractField(data any, field string) any {
	if field == "" {
		return data
	}

	// Converte in map se possibile
	dataMap, ok := data.(map[string]any)
	if !ok {
		// Prova con reflection per struct
		return rsb.extractFieldReflection(data, field)
	}

	// Supporta notazione dot per campi nested
	if field == "." {
		return dataMap
	}

	// Supporto per campi nested: "field.subfield.subsubfield"
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		var current any = dataMap

		for _, part := range parts {
			if currentMap, ok := current.(map[string]any); ok {
				current = currentMap[part]
			} else {
				return nil
			}
		}

		return current
	}

	// Campo singolo di primo livello
	return dataMap[field]
}

// extractFieldReflection estrae campo usando reflection con supporto per campi nested
func (rsb *ResponseSwitchBuilder[T]) extractFieldReflection(data any, field string) any {
	// Supporto per campi nested: "User.Name" o "Address.City.Name"
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		var current any = data

		for _, part := range parts {
			val := reflect.ValueOf(current)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}

			if val.Kind() != reflect.Struct {
				return nil
			}

			fieldVal := val.FieldByName(part)
			if !fieldVal.IsValid() {
				return nil
			}

			current = fieldVal.Interface()
		}

		return current
	}

	// Campo singolo
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	fieldVal := val.FieldByName(field)
	if !fieldVal.IsValid() {
		return nil
	}

	return fieldVal.Interface()
}

// evaluateOperator valuta un operatore
func (rsb *ResponseSwitchBuilder[T]) evaluateOperator(fieldValue any, operator Operator, compareValue any) bool {
	switch operator {
	case OpEquals:
		return fieldValue == compareValue
	case OpNotEquals:
		return fieldValue != compareValue
	case OpIsNull:
		return fieldValue == nil
	case OpIsNotNull:
		return fieldValue != nil
	case OpContains:
		return rsb.contains(fieldValue, compareValue)
	case OpGreaterThan:
		return rsb.compare(fieldValue, compareValue) > 0
	case OpLessThan:
		return rsb.compare(fieldValue, compareValue) < 0
	case OpGreaterOrEqual:
		return rsb.compare(fieldValue, compareValue) >= 0
	case OpLessOrEqual:
		return rsb.compare(fieldValue, compareValue) <= 0
	case OpIn:
		return rsb.in(fieldValue, compareValue)
	case OpNotIn:
		return !rsb.in(fieldValue, compareValue)
	default:
		return false
	}
}

// Helper methods per gli operatori
func (rsb *ResponseSwitchBuilder[T]) contains(haystack, needle any) bool {
	haystackStr, ok1 := haystack.(string)
	needleStr, ok2 := needle.(string)
	if ok1 && ok2 {
		return len(haystackStr) > 0 && len(needleStr) > 0 &&
			haystackStr != needleStr &&
			len(haystackStr) > len(needleStr)
	}
	return false
}

func (rsb *ResponseSwitchBuilder[T]) compare(a, b any) int {
	// Implementazione semplificata per numeri
	aFloat, aOk := rsb.toFloat64(a)
	bFloat, bOk := rsb.toFloat64(b)

	if aOk && bOk {
		if aFloat > bFloat {
			return 1
		} else if aFloat < bFloat {
			return -1
		}
		return 0
	}

	// Fallback su string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr > bStr {
		return 1
	} else if aStr < bStr {
		return -1
	}
	return 0
}

func (rsb *ResponseSwitchBuilder[T]) toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

func (rsb *ResponseSwitchBuilder[T]) in(item any, slice any) bool {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < sliceVal.Len(); i++ {
		if sliceVal.Index(i).Interface() == item {
			return true
		}
	}
	return false
}

// ConditionalFlowBuilder permette di creare flussi complessi basati su response
type ConditionalFlowBuilder[T any] struct {
	orchestration *OrchestrationBuilder[T]
	lastResponse  string
}

// Flow crea un nuovo ConditionalFlowBuilder
func (op *OrchestrationPatterns[T]) Flow() *ConditionalFlowBuilder[T] {
	return &ConditionalFlowBuilder[T]{
		orchestration: op.client.Orchestrate(),
	}
}

// Call esegue una chiamata e salva la response per il branching
func (cfb *ConditionalFlowBuilder[T]) Call(stepName string, builderFunc func(map[string]any) RequestBuilder[T]) *StepBuilder[T] {
	cfb.lastResponse = stepName + "_response"

	step := cfb.orchestration.Step(stepName, builderFunc)

	// Modifica per salvare automaticamente la response
	originalOnSuccess := step.step.OnSuccess
	step.step.OnSuccess = func(response *Response[T], ctx map[string]any) error {
		if originalOnSuccess != nil {
			if err := originalOnSuccess(response, ctx); err != nil {
				return err
			}
		}
		ctx[cfb.lastResponse] = response
		return nil
	}

	return step
}

// Switch crea un response switch sull'ultima chiamata
func (cfb *ConditionalFlowBuilder[T]) Switch() *ResponseSwitchBuilder[T] {
	return &ResponseSwitchBuilder[T]{
		orchestration: cfb.orchestration,
		responseKey:   cfb.lastResponse,
		cases:         make([]ResponseCase[T], 0),
	}
}

// Execute esegue il flusso condizionale
func (cfb *ConditionalFlowBuilder[T]) Execute(ctx context.Context) (*OrchestrationResult[T], error) {
	return cfb.orchestration.Execute(ctx)
}

// FieldEquals Helper functions per creare condizioni comuni
func FieldEquals(field string, value any) ResponseCondition {
	return ResponseCondition{Field: field, Operator: OpEquals, Value: value}
}

func FieldContains(field string, value any) ResponseCondition {
	return ResponseCondition{Field: field, Operator: OpContains, Value: value}
}

func FieldGreaterThan(field string, value any) ResponseCondition {
	return ResponseCondition{Field: field, Operator: OpGreaterThan, Value: value}
}

func FieldIn(field string, values any) ResponseCondition {
	return ResponseCondition{Field: field, Operator: OpIn, Value: values}
}

func StatusCodeEquals(code int) ResponseCondition {
	return ResponseCondition{
		Custom: func(responseData any) bool {
			if response, ok := responseData.(*Response[any]); ok {
				return response.StatusCode == code
			}
			return false
		},
	}
}

func CustomCondition(fn func(any) bool) ResponseCondition {
	return ResponseCondition{Custom: fn}
}
