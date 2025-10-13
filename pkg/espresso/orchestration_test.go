package espresso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestOrchestrate(t *testing.T) {
	client := NewClient[testData](ClientConfig{})
	builder := client.Orchestrate()

	if builder == nil {
		t.Fatal("OrchestrationBuilder non dovrebbe essere nil")
	}

	if builder.client != client {
		t.Error("Client non impostato correttamente")
	}

	if builder.steps == nil {
		t.Error("Steps non inizializzati")
	}

	if builder.context == nil {
		t.Error("Context non inizializzato")
	}

	if builder.timeout == 0 {
		t.Error("Timeout non impostato")
	}
}

func TestStepBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("fetch-data", func(ctx map[string]any) RequestBuilder[testData] {
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

	if len(result.Steps) != 1 {
		t.Errorf("Steps attesi 1, ricevuti %d", len(result.Steps))
	}

	if !result.Steps[0].Success {
		t.Error("Step dovrebbe avere successo")
	}

	if result.Steps[0].Name != "fetch-data" {
		t.Errorf("Nome step atteso 'fetch-data', ricevuto '%s'", result.Steps[0].Name)
	}
}

func TestMultipleStepsSequential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Step("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Step("step3", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if len(result.Steps) != 3 {
		t.Errorf("Steps attesi 3, ricevuti %d", len(result.Steps))
	}

	for i, step := range result.Steps {
		if !step.Success {
			t.Errorf("Step %d dovrebbe avere successo", i)
		}
	}
}

func TestParallelExecution(t *testing.T) {
	callCount := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	start := time.Now()

	result, err := client.Orchestrate().
		Parallel().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Step("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Step("step3", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if len(result.Steps) != 3 {
		t.Errorf("Steps attesi 3, ricevuti %d", len(result.Steps))
	}

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	if finalCount != 3 {
		t.Errorf("Call count atteso 3, ricevuto %d", finalCount)
	}

	if duration > 200*time.Millisecond {
		t.Errorf("Esecuzione parallela dovrebbe essere più veloce, durata: %v", duration)
	}
}

func TestWhenCondition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		When(func(ctx map[string]any) bool {
			return true
		}).
		End().
		Step("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		When(func(ctx map[string]any) bool {
			return false
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if len(result.Steps) != 2 {
		t.Errorf("Steps attesi 2, ricevuti %d", len(result.Steps))
	}

	if !result.Steps[0].Success {
		t.Error("Step1 dovrebbe essere eseguito")
	}

	if !result.Steps[1].Skipped {
		t.Error("Step2 dovrebbe essere skipped")
	}
}

func TestOnSuccess1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	successCalled := false

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnSuccess(func(response *Response[testData], ctx map[string]any) error {
			successCalled = true
			return nil
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !successCalled {
		t.Error("OnSuccess callback non chiamato")
	}
}

func TestOnFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	failCalled := false

	client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnFail(func(err error, ctx map[string]any) error {
			failCalled = true
			return nil
		}).
		End().
		Execute(context.Background())

	if !failCalled {
		t.Error("OnFail callback non chiamato")
	}
}

func TestOnComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	completeCalled := false

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		OnComplete(func(result *OrchestrationResult[testData]) {
			completeCalled = true
		}).
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !completeCalled {
		t.Error("OnComplete callback non chiamato")
	}
}

func TestOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	errorCalled := false

	client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		OnError(func(err error, result *OrchestrationResult[testData]) {
			errorCalled = true
		}).
		Execute(context.Background())

	if !errorCalled {
		t.Error("OnError callback non chiamato")
	}
}

func TestOnStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	statusHandlerCalled := false

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnStatusCode(201, func(response *Response[testData], ctx map[string]any) error {
			statusHandlerCalled = true
			return nil
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !statusHandlerCalled {
		t.Error("Status handler non chiamato")
	}
}

func TestTransform(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		Transform(func(response *Response[testData], ctx map[string]any) (any, error) {
			return "transformed", nil
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if result.Steps[0].Data != "transformed" {
		t.Errorf("Data atteso 'transformed', ricevuto %v", result.Steps[0].Data)
	}
}

func TestRequiredStep(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		Required(true).
		End().
		Execute(context.Background())

	if err == nil {
		t.Error("Errore atteso per step required fallito")
	}

	if result.Success {
		t.Error("Orchestrazione non dovrebbe avere successo")
	}
}

func TestOptionalStep(t *testing.T) {
	callOrder := []string{}
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer successServer.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		Optional().
		OnSuccess(func(response *Response[testData], ctx map[string]any) error {
			mu.Lock()
			callOrder = append(callOrder, "step1-success")
			mu.Unlock()
			return nil
		}).
		OnFail(func(err error, ctx map[string]any) error {
			mu.Lock()
			callOrder = append(callOrder, "step1-fail")
			mu.Unlock()
			return nil
		}).
		End().
		Step("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(successServer.URL)
		}).
		OnSuccess(func(response *Response[testData], ctx map[string]any) error {
			mu.Lock()
			callOrder = append(callOrder, "step2-success")
			mu.Unlock()
			return nil
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if len(result.Steps) != 2 {
		t.Errorf("Steps attesi 2, ricevuti %d", len(result.Steps))
	}

	mu.Lock()
	orderLen := len(callOrder)
	mu.Unlock()

	if orderLen != 2 {
		t.Errorf("Call order atteso 2, ricevuto %d", orderLen)
	}
}

func TestWithContext(t *testing.T) {
	client := NewClient[testData](ClientConfig{})

	initialContext := map[string]any{
		"userId": 123,
		"token":  "abc",
	}

	builder := client.Orchestrate().WithContext(initialContext)

	if builder.context["userId"] != 123 {
		t.Error("Context non impostato correttamente")
	}

	if builder.context["token"] != "abc" {
		t.Error("Context non impostato correttamente")
	}
}

func TestSaveToContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":42,"name":"saved"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		SaveToContext("userData").
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	userData, exists := result.Context["userData"]
	if !exists {
		t.Fatal("userData non trovato nel context")
	}

	savedData, ok := userData.(testData)
	if !ok {
		t.Fatal("userData non è del tipo corretto")
	}

	if savedData.ID != 42 {
		t.Errorf("ID atteso 42, ricevuto %d", savedData.ID)
	}

	if savedData.Name != "saved" {
		t.Errorf("Name atteso 'saved', ricevuto '%s'", savedData.Name)
	}
}

func TestSaveFieldToContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":99,"name":"field"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		SaveFieldToContext("savedId", func(response *Response[testData]) any {
			return response.Data.ID
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	savedId, exists := result.Context["savedId"]
	if !exists {
		t.Fatal("savedId non trovato nel context")
	}

	if savedId != 99 {
		t.Errorf("savedId atteso 99, ricevuto %v", savedId)
	}
}

func TestFailOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		FailOn(404, 500).
		End().
		Execute(context.Background())

	if err == nil {
		t.Error("Errore atteso per status code 404")
	}

	if result.Success {
		t.Error("Orchestrazione non dovrebbe avere successo")
	}

	if result.Steps[0].Success {
		t.Error("Step non dovrebbe avere successo")
	}
}

func TestContextSharing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnSuccess(func(response *Response[testData], ctx map[string]any) error {
			ctx["step1Data"] = "from-step1"
			return nil
		}).
		End().
		Step("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnSuccess(func(response *Response[testData], ctx map[string]any) error {
			if ctx["step1Data"] != "from-step1" {
				return errors.New("context non condiviso correttamente")
			}
			ctx["step2Data"] = "from-step2"
			return nil
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if result.Context["step1Data"] != "from-step1" {
		t.Error("Context step1 non salvato")
	}

	if result.Context["step2Data"] != "from-step2" {
		t.Error("Context step2 non salvato")
	}
}

func TestStepTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	_, err := client.Orchestrate().
		Step("slow-step", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		WithTimeout(50 * time.Millisecond).
		End().
		Execute(context.Background())

	if err == nil {
		t.Error("Errore atteso per timeout")
	}
}

func TestOrchestrationTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	_, err := client.Orchestrate().
		WithTimeout(50*time.Millisecond).
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err == nil {
		t.Error("Errore atteso per orchestration timeout")
	}
}

func TestThen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Then("step2", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if len(result.Steps) != 2 {
		t.Errorf("Steps attesi 2, ricevuti %d", len(result.Steps))
	}

	if !result.Steps[0].Success || !result.Steps[1].Success {
		t.Error("Tutti gli step dovrebbero avere successo")
	}
}

func TestOnStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	handler202Called := false
	handler404Called := false

	handlers := map[int]func(*Response[testData], map[string]any) error{
		202: func(response *Response[testData], ctx map[string]any) error {
			handler202Called = true
			return nil
		},
		404: func(response *Response[testData], ctx map[string]any) error {
			handler404Called = true
			return nil
		},
	}

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		OnStatus(handlers).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !handler202Called {
		t.Error("Handler 202 non chiamato")
	}

	if handler404Called {
		t.Error("Handler 404 non dovrebbe essere chiamato")
	}
}

func TestLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	logger := &mockLogger{}

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		Log(logger).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if !logger.infoCalled {
		t.Error("Logger.Info non chiamato")
	}
}

func TestComplexOrchestration(t *testing.T) {
	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":100,"name":"John"}`))
	}))
	defer userServer.Close()

	orderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":200,"name":"Order1"}`))
	}))
	defer orderServer.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("fetch-user", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(userServer.URL)
		}).
		SaveToContext("user").
		End().
		Step("fetch-orders", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(orderServer.URL)
		}).
		When(func(ctx map[string]any) bool {
			_, exists := ctx["user"]
			return exists
		}).
		SaveToContext("orders").
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if !result.Success {
		t.Error("Orchestrazione dovrebbe avere successo")
	}

	if len(result.Steps) != 2 {
		t.Errorf("Steps attesi 2, ricevuti %d", len(result.Steps))
	}

	user, userExists := result.Context["user"]
	if !userExists {
		t.Fatal("User non trovato nel context")
	}

	userData := user.(testData)
	if userData.ID != 100 {
		t.Errorf("User ID atteso 100, ricevuto %d", userData.ID)
	}

	orders, ordersExist := result.Context["orders"]
	if !ordersExist {
		t.Fatal("Orders non trovato nel context")
	}

	ordersData := orders.(testData)
	if ordersData.ID != 200 {
		t.Errorf("Orders ID atteso 200, ricevuto %d", ordersData.ID)
	}
}

func TestOrchestrationResultDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	client := NewClient[testData](ClientConfig{})

	result, err := client.Orchestrate().
		Step("step1", func(ctx map[string]any) RequestBuilder[testData] {
			return client.Request(server.URL)
		}).
		End().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Errore non atteso: %v", err)
	}

	if result.Duration < 100*time.Millisecond {
		t.Errorf("Duration dovrebbe essere almeno 100ms, ricevuto %v", result.Duration)
	}

	if result.Steps[0].Duration < 100*time.Millisecond {
		t.Errorf("Step duration dovrebbe essere almeno 100ms, ricevuto %v", result.Steps[0].Duration)
	}
}

type mockLogger struct {
	infoCalled  bool
	errorCalled bool
}

func (m *mockLogger) Debug(msg string, fields ...any) {}

func (m *mockLogger) Info(msg string, fields ...any) {
	m.infoCalled = true
}

func (m *mockLogger) Warn(msg string, fields ...any) {}

func (m *mockLogger) Error(msg string, fields ...any) {
	m.errorCalled = true
}

func (m *mockLogger) With(fields ...any) Logger {
	return m
}
