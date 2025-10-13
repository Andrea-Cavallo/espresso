package espresso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseSwitch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"active"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	executed := false

	result, err := client.Orchestrate().
		Step("fetch-data", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("response").
		OnValue("Name", "active", "process-active", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		OnComplete(func(result *OrchestrationResult[testData]) {
			executed = true
		}).
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !executed {
		t.Error("OnComplete non eseguito")
	}
}

func TestResponseSwitchOnField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":100,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	stepExecuted := false

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		OnField("ID", OpEquals, 100, "match-id", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Step("verify", func(ctx map[string]any) RequestBuilder[testData] {
			stepExecuted = true
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !stepExecuted {
		t.Error("Step verify non eseguito")
	}
}

func TestResponseSwitchOnStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"name":"created"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	statusMatched := false

	result, err := client.Orchestrate().
		Step("create", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		OnStatus(201, "handle-created", func(ctx map[string]any) RequestBuilder[testData] {
			statusMatched = true
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !statusMatched {
		t.Error("Status 201 non matchato")
	}
}

func TestResponseSwitchOnCustom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":50,"name":"custom"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	customMatched := false

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		OnCustom(func(responseData any) bool {
			if response, ok := responseData.(*Response[testData]); ok {
				return response.Data != nil && response.Data.ID == 50
			}
			return false
		}, "custom-handler", func(ctx map[string]any) RequestBuilder[testData] {
			customMatched = true
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !customMatched {
		t.Error("Custom condition non matchata")
	}
}

func TestResponseSwitchDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":999,"name":"unknown"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	defaultExecuted := false

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		OnValue("ID", 100, "match-100", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnValue("ID", 200, "match-200", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		Default("default-handler", func(ctx map[string]any) RequestBuilder[testData] {
			defaultExecuted = true
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !defaultExecuted {
		t.Error("Default case non eseguito")
	}
}

func TestResponseSwitchMultipleCases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":10,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	case1Executed := false
	case2Executed := false

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		Case(FieldEquals("ID", 10), "case1", func(ctx map[string]any) RequestBuilder[testData] {
			case1Executed = true
			return client.Request(server.URL)
		}).
		Case(FieldEquals("Name", "test"), "case2", func(ctx map[string]any) RequestBuilder[testData] {
			case2Executed = true
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !case1Executed {
		t.Error("Case1 non eseguito")
	}

	if !case2Executed {
		t.Error("Case2 non eseguito")
	}
}

func TestEvaluateConditionEquals(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpEquals,
		Value:    42,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpEquals dovrebbe essere vera")
	}
}

func TestEvaluateConditionNotEquals(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpNotEquals,
		Value:    100,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpNotEquals dovrebbe essere vera")
	}
}

func TestEvaluateConditionGreaterThan(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 100, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpGreaterThan,
		Value:    50,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpGreaterThan dovrebbe essere vera")
	}
}

func TestEvaluateConditionLessThan(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 25, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpLessThan,
		Value:    50,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpLessThan dovrebbe essere vera")
	}
}

func TestEvaluateConditionIsNull(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	response := &Response[testData]{Data: nil}

	condition := ResponseCondition{
		Operator: OpIsNull,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpIsNull dovrebbe essere vera")
	}
}

func TestEvaluateConditionIsNotNull(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 1, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Operator: OpIsNotNull,
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpIsNotNull dovrebbe essere vera")
	}
}

func TestEvaluateConditionIn(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 2, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpIn,
		Value:    []any{1, 2, 3},
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpIn dovrebbe essere vera")
	}
}

func TestEvaluateConditionNotIn(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 10, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Field:    "ID",
		Operator: OpNotIn,
		Value:    []any{1, 2, 3},
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Condizione OpNotIn dovrebbe essere vera")
	}
}

func TestExtractField(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "example"}

	idValue := rsb.extractField(data, "ID")
	if idValue != 42 {
		t.Errorf("ID atteso 42, ricevuto %v", idValue)
	}

	nameValue := rsb.extractField(data, "Name")
	if nameValue != "example" {
		t.Errorf("Name atteso 'example', ricevuto %v", nameValue)
	}
}

func TestExtractFieldFromMap(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := map[string]any{
		"id":   123,
		"name": "mapped",
	}

	idValue := rsb.extractField(data, "id")
	if idValue != 123 {
		t.Errorf("id atteso 123, ricevuto %v", idValue)
	}

	nameValue := rsb.extractField(data, "name")
	if nameValue != "mapped" {
		t.Errorf("name atteso 'mapped', ricevuto %v", nameValue)
	}
}

func TestContains(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	result := rsb.contains("hello world", "world")
	if !result {
		t.Error("Contains dovrebbe essere true")
	}

	result = rsb.contains("hello", "xyz")
	if result {
		t.Error("Contains dovrebbe essere false")
	}
}

func TestCompare(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	tests := []struct {
		name     string
		a        any
		b        any
		expected int
	}{
		{"greater", 10, 5, 1},
		{"less", 5, 10, -1},
		{"equal", 10, 10, 0},
		{"float greater", 10.5, 5.2, 1},
		{"float less", 5.2, 10.5, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rsb.compare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Compare(%v, %v) atteso %d, ricevuto %d", tt.a, tt.b, tt.expected, result)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	tests := []struct {
		name     string
		value    any
		expected float64
		ok       bool
	}{
		{"int", 42, 42.0, true},
		{"int64", int64(100), 100.0, true},
		{"float32", float32(3.14), 3.14, true},
		{"float64", 2.71, 2.71, true},
		{"string", "not a number", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := rsb.toFloat64(tt.value)

			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok atteso %v, ricevuto %v", tt.value, tt.ok, ok)
			}

			if ok && result != tt.expected {
				t.Errorf("toFloat64(%v) atteso %v, ricevuto %v", tt.value, tt.expected, result)
			}
		})
	}
}

func TestIn(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	slice := []any{1, 2, 3, 4, 5}

	if !rsb.in(3, slice) {
		t.Error("in(3, slice) dovrebbe essere true")
	}

	if rsb.in(10, slice) {
		t.Error("in(10, slice) dovrebbe essere false")
	}
}

func TestConditionalFlowBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"flow"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})
	patterns := &OrchestrationPatterns[testData]{client: client}

	result, err := patterns.Flow().
		Call("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Flow dovrebbe avere successo")
	}
}

func TestFieldEqualsHelper(t *testing.T) {
	condition := FieldEquals("status", "active")

	if condition.Field != "status" {
		t.Errorf("Field atteso 'status', ricevuto '%s'", condition.Field)
	}

	if condition.Operator != OpEquals {
		t.Error("Operator dovrebbe essere OpEquals")
	}

	if condition.Value != "active" {
		t.Errorf("Value atteso 'active', ricevuto %v", condition.Value)
	}
}

func TestFieldContainsHelper(t *testing.T) {
	condition := FieldContains("message", "error")

	if condition.Field != "message" {
		t.Errorf("Field atteso 'message', ricevuto '%s'", condition.Field)
	}

	if condition.Operator != OpContains {
		t.Error("Operator dovrebbe essere OpContains")
	}

	if condition.Value != "error" {
		t.Errorf("Value atteso 'error', ricevuto %v", condition.Value)
	}
}

func TestFieldGreaterThanHelper(t *testing.T) {
	condition := FieldGreaterThan("count", 100)

	if condition.Field != "count" {
		t.Errorf("Field atteso 'count', ricevuto '%s'", condition.Field)
	}

	if condition.Operator != OpGreaterThan {
		t.Error("Operator dovrebbe essere OpGreaterThan")
	}

	if condition.Value != 100 {
		t.Errorf("Value atteso 100, ricevuto %v", condition.Value)
	}
}

func TestFieldInHelper(t *testing.T) {
	values := []int{1, 2, 3}
	condition := FieldIn("id", values)

	if condition.Field != "id" {
		t.Errorf("Field atteso 'id', ricevuto '%s'", condition.Field)
	}

	if condition.Operator != OpIn {
		t.Error("Operator dovrebbe essere OpIn")
	}
}

func TestCustomConditionHelper(t *testing.T) {
	customFn := func(data any) bool {
		return true
	}

	condition := CustomCondition(customFn)

	if condition.Custom == nil {
		t.Error("Custom function non impostata")
	}

	result := condition.Custom("test")
	if !result {
		t.Error("Custom function dovrebbe restituire true")
	}
}

func TestResponseSwitchNoMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":999,"name":"nomatch"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("resp").
		OnValue("ID", 100, "case1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnValue("ID", 200, "case2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo anche senza match")
	}

	matchFound := false
	for _, step := range result.Steps {
		if step.Name == "case1" && !step.Skipped {
			matchFound = true
		}
		if step.Name == "case2" && !step.Skipped {
			matchFound = true
		}
	}

	if matchFound {
		t.Error("Nessun case dovrebbe essere eseguito")
	}
}

func TestEvaluateConditionCustomFunction(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "test"}
	response := &Response[testData]{Data: &data}

	condition := ResponseCondition{
		Custom: func(responseData any) bool {
			if resp, ok := responseData.(*Response[testData]); ok {
				return resp.Data != nil && resp.Data.ID > 40
			}
			return false
		},
	}

	result := rsb.evaluateCondition(condition, response)

	if !result {
		t.Error("Custom condition dovrebbe essere vera")
	}
}

func TestExtractFieldEmptyField(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "test"}

	result := rsb.extractField(data, "")

	dataResult, ok := result.(testData)
	if !ok {
		t.Fatal("Risultato dovrebbe essere testData")
	}

	if dataResult.ID != 42 {
		t.Error("Dovrebbe restituire l'intero oggetto")
	}
}

func TestExtractFieldNonexistent(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	data := testData{ID: 42, Name: "test"}

	result := rsb.extractField(data, "NonExistent")

	if result != nil {
		t.Error("Campo inesistente dovrebbe restituire nil")
	}
}

func TestEvaluateOperatorGreaterOrEqual(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	if !rsb.evaluateOperator(10, OpGreaterOrEqual, 10) {
		t.Error("10 >= 10 dovrebbe essere true")
	}

	if !rsb.evaluateOperator(15, OpGreaterOrEqual, 10) {
		t.Error("15 >= 10 dovrebbe essere true")
	}

	if rsb.evaluateOperator(5, OpGreaterOrEqual, 10) {
		t.Error("5 >= 10 dovrebbe essere false")
	}
}

func TestEvaluateOperatorLessOrEqual(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	if !rsb.evaluateOperator(10, OpLessOrEqual, 10) {
		t.Error("10 <= 10 dovrebbe essere true")
	}

	if !rsb.evaluateOperator(5, OpLessOrEqual, 10) {
		t.Error("5 <= 10 dovrebbe essere true")
	}

	if rsb.evaluateOperator(15, OpLessOrEqual, 10) {
		t.Error("15 <= 10 dovrebbe essere false")
	}
}

func TestResponseSwitchSavesResponseInContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":123,"name":"saved"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("fetch", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		ResponseSwitch("myresponse").
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	responseData, exists := result.Context["myresponse"]
	if !exists {
		t.Fatal("Response non salvata nel context")
	}

	response, ok := responseData.(*Response[testData])
	if !ok {
		t.Fatal("Response nel context non del tipo corretto")
	}

	if response.Data == nil || response.Data.ID != 123 {
		t.Error("Response data non corretto nel context")
	}
}

func TestInWithNonSlice(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	result := rsb.in(3, "not a slice")

	if result {
		t.Error("in() con non-slice dovrebbe restituire false")
	}
}

func TestCompareWithStrings(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	result := rsb.compare("apple", "banana")
	if result >= 0 {
		t.Error("'apple' < 'banana' dovrebbe restituire valore negativo")
	}

	result = rsb.compare("banana", "apple")
	if result <= 0 {
		t.Error("'banana' > 'apple' dovrebbe restituire valore positivo")
	}
}

func TestContainsEdgeCases(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	rsb := &ResponseSwitchBuilder[testData]{
		orchestration: client.Orchestrate(),
	}

	if rsb.contains("", "test") {
		t.Error("Empty string non dovrebbe contenere nulla")
	}

	if rsb.contains("test", "") {
		t.Error("Non dovrebbe contenere empty string")
	}

	if rsb.contains("same", "same") {
		t.Error("Stringhe identiche non dovrebbero passare contains")
	}

	if rsb.contains("short", "longer string") {
		t.Error("String corta non può contenere string più lunga")
	}
}
