package espresso

import (
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	name := "test-breaker"
	config := CircuitBreakerConfig{
		MaxRequests:      5,
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  3,
	}

	cb := NewCircuitBreaker(name, config)

	if cb.name != name {
		t.Errorf("Nome atteso %s, ricevuto %s", name, cb.name)
	}

	if cb.state != CircuitClosed {
		t.Errorf("Stato iniziale atteso %v, ricevuto %v", CircuitClosed, cb.state)
	}

	if cb.config.MaxRequests != 5 {
		t.Error("Configurazione non impostata correttamente")
	}
}

func TestNewCircuitBreakerWithDefaults(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{})

	if cb.config.MaxRequests == 0 {
		t.Error("MaxRequests dovrebbe avere un valore di default")
	}

	if cb.config.Interval == 0 {
		t.Error("Interval dovrebbe avere un valore di default")
	}

	if cb.config.Timeout == 0 {
		t.Error("Timeout dovrebbe avere un valore di default")
	}

	if cb.config.FailureThreshold == 0 {
		t.Error("FailureThreshold dovrebbe avere un valore di default")
	}

	if cb.config.SuccessThreshold == 0 {
		t.Error("SuccessThreshold dovrebbe avere un valore di default")
	}

	if cb.config.HalfOpenMaxReqs == 0 {
		t.Error("HalfOpenMaxReqs dovrebbe avere un valore di default")
	}
}

func TestExecuteSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	executed := false
	fn := func() error {
		executed = true
		return nil
	}

	err := cb.Execute(fn)

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	if !executed {
		t.Error("La funzione non è stata eseguita")
	}

	counts := cb.GetCounts()
	if counts.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses atteso 1, ricevuto %d", counts.TotalSuccesses)
	}
}

func TestExecuteFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	expectedError := errors.New("test error")
	fn := func() error {
		return expectedError
	}

	err := cb.Execute(fn)

	if err != expectedError {
		t.Errorf("Errore atteso %v, ricevuto %v", expectedError, err)
	}

	counts := cb.GetCounts()
	if counts.TotalFailures != 1 {
		t.Errorf("TotalFailures atteso 1, ricevuto %d", counts.TotalFailures)
	}
}

func TestCircuitOpenAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	state := cb.State()
	if state != CircuitOpen {
		t.Errorf("Stato atteso %v, ricevuto %v", CircuitOpen, state)
	}

	err := cb.Execute(failFn)
	if err == nil {
		t.Error("Errore atteso quando il circuit è open")
	}
}

func TestCircuitHalfOpenAfterTimeout(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	if cb.State() != CircuitOpen {
		t.Error("Circuit dovrebbe essere open dopo i fallimenti")
	}

	time.Sleep(150 * time.Millisecond)

	state := cb.State()
	if state != CircuitHalfOpen {
		t.Errorf("Stato atteso %v dopo timeout, ricevuto %v", CircuitHalfOpen, state)
	}
}

func TestCircuitClosedAfterSuccesses(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  5,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	successFn := func() error {
		return nil
	}

	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	time.Sleep(150 * time.Millisecond)

	if cb.State() != CircuitHalfOpen {
		t.Error("Circuit dovrebbe essere half-open")
	}

	for i := 0; i < 2; i++ {
		cb.Execute(successFn)
	}

	state := cb.State()
	if state != CircuitClosed {
		t.Errorf("Stato atteso %v dopo successi, ricevuto %v", CircuitClosed, state)
	}
}

func TestReset(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	failFn := func() error {
		return errors.New("failure")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(failFn)
	}

	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Error("Circuit dovrebbe essere closed dopo reset")
	}

	counts := cb.GetCounts()
	if counts.TotalFailures != 0 {
		t.Error("I contatori dovrebbero essere resettati")
	}
}

func TestBeforeRequestClosed(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	allowed, err := cb.beforeRequest()

	if !allowed {
		t.Error("La richiesta dovrebbe essere permessa quando il circuit è closed")
	}

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	if cb.counts.Requests != 1 {
		t.Error("Il contatore delle richieste dovrebbe essere incrementato")
	}
}

func TestBeforeRequestOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      2,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	for i := 0; i < 2; i++ {
		cb.Execute(failFn)
	}

	allowed, err := cb.beforeRequest()

	if allowed {
		t.Error("La richiesta non dovrebbe essere permessa quando il circuit è open")
	}

	if err == nil {
		t.Error("Errore atteso quando il circuit è open")
	}
}

func TestBeforeRequestHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      2,
		Interval:         60 * time.Second,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	failFn := func() error {
		return errors.New("failure")
	}

	for i := 0; i < 2; i++ {
		cb.Execute(failFn)
	}

	time.Sleep(150 * time.Millisecond)

	allowed, err := cb.beforeRequest()

	if !allowed {
		t.Error("La prima richiesta dovrebbe essere permessa in half-open")
	}

	if err != nil {
		t.Errorf("Errore non atteso: %v", err)
	}

	cb.beforeRequest()

	allowed, err = cb.beforeRequest()

	if allowed {
		t.Error("Le richieste oltre HalfOpenMaxReqs non dovrebbero essere permesse")
	}

	if err == nil {
		t.Error("Errore atteso quando si supera HalfOpenMaxReqs")
	}
}

func TestAfterRequestSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	cb.afterRequest(true)

	counts := cb.GetCounts()
	if counts.TotalSuccesses != 1 {
		t.Error("TotalSuccesses non incrementato correttamente")
	}

	if counts.ConsecutiveSuccesses != 1 {
		t.Error("ConsecutiveSuccesses non incrementato correttamente")
	}

	if counts.ConsecutiveFailures != 0 {
		t.Error("ConsecutiveFailures dovrebbe essere 0")
	}
}

func TestAfterRequestFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	cb.afterRequest(false)

	counts := cb.GetCounts()
	if counts.TotalFailures != 1 {
		t.Error("TotalFailures non incrementato correttamente")
	}

	if counts.ConsecutiveFailures != 1 {
		t.Error("ConsecutiveFailures non incrementato correttamente")
	}

	if counts.ConsecutiveSuccesses != 0 {
		t.Error("ConsecutiveSuccesses dovrebbe essere 0")
	}
}

func TestOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	cb.onSuccess()

	if cb.counts.ConsecutiveSuccesses != 1 {
		t.Error("ConsecutiveSuccesses non incrementato")
	}

	if cb.counts.ConsecutiveFailures != 0 {
		t.Error("ConsecutiveFailures non resettato")
	}
}

func TestOnFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	cb.onFailure()

	if cb.counts.ConsecutiveFailures != 1 {
		t.Error("ConsecutiveFailures non incrementato")
	}

	if cb.counts.ConsecutiveSuccesses != 0 {
		t.Error("ConsecutiveSuccesses non resettato")
	}
}

func TestShouldTrip(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	cb.counts.Requests = 3
	cb.counts.ConsecutiveFailures = 2

	if !cb.shouldTrip() {
		t.Error("shouldTrip dovrebbe restituire true")
	}

	cb.counts.ConsecutiveFailures = 1

	if cb.shouldTrip() {
		t.Error("shouldTrip dovrebbe restituire false")
	}
}

func TestShouldTripWithCustomFunction(t *testing.T) {
	called := false
	config := CircuitBreakerConfig{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
		ReadyToTrip: func(counts Counts) bool {
			called = true
			return counts.ConsecutiveFailures > 1
		},
	}

	cb := NewCircuitBreaker("test", config)
	cb.counts.ConsecutiveFailures = 2

	result := cb.shouldTrip()

	if !called {
		t.Error("La funzione ReadyToTrip non è stata chiamata")
	}

	if !result {
		t.Error("ReadyToTrip dovrebbe restituire true")
	}
}

func TestUpdateStateClosedExpiry(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxRequests:      5,
		Interval:         100 * time.Millisecond,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
	}

	cb := NewCircuitBreaker("test", config)

	cb.counts.Requests = 3
	cb.expiry = time.Now().Add(-1 * time.Millisecond)

	cb.updateState()

	if cb.counts.Requests != 0 {
		t.Error("I contatori dovrebbero essere resettati dopo l'expiry")
	}
}

func TestSetState(t *testing.T) {
	stateChanged := false
	var oldStateReceived CircuitState
	var newStateReceived CircuitState

	config := CircuitBreakerConfig{
		MaxRequests:      5,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  2,
		OnStateChange: func(name string, oldState, newState CircuitState) {
			stateChanged = true
			oldStateReceived = oldState
			newStateReceived = newState
		},
	}

	cb := NewCircuitBreaker("test", config)

	cb.setState(CircuitOpen)

	time.Sleep(50 * time.Millisecond)

	if !stateChanged {
		t.Error("OnStateChange non è stato chiamato")
	}

	if oldStateReceived != CircuitClosed {
		t.Errorf("Old state atteso %v, ricevuto %v", CircuitClosed, oldStateReceived)
	}

	if newStateReceived != CircuitOpen {
		t.Errorf("New state atteso %v, ricevuto %v", CircuitOpen, newStateReceived)
	}
}

func TestGetCounts(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	successFn := func() error {
		return nil
	}

	cb.Execute(successFn)
	cb.Execute(successFn)

	counts := cb.GetCounts()

	if counts.TotalSuccesses != 2 {
		t.Errorf("TotalSuccesses atteso 2, ricevuto %d", counts.TotalSuccesses)
	}

	if counts.Requests != 2 {
		t.Errorf("Requests atteso 2, ricevuto %d", counts.Requests)
	}
}

func TestNewCircuitBreakerManager(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	manager := NewCircuitBreakerManager(config)

	if manager == nil {
		t.Fatal("Manager non dovrebbe essere nil")
	}

	if manager.breakers == nil {
		t.Error("Breakers map non inizializzata")
	}

	if manager.config.MaxRequests != config.MaxRequests {
		t.Error("Configurazione non impostata correttamente")
	}
}

func TestGetBreaker(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())
	endpoint := "https://api.example.com/users"

	breaker := manager.GetBreaker(endpoint)

	if breaker == nil {
		t.Fatal("Breaker non dovrebbe essere nil")
	}

	if breaker.name != endpoint {
		t.Errorf("Nome breaker atteso %s, ricevuto %s", endpoint, breaker.name)
	}

	breaker2 := manager.GetBreaker(endpoint)

	if breaker != breaker2 {
		t.Error("GetBreaker dovrebbe restituire la stessa istanza")
	}
}

func TestGetBreakerWithConfig(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())
	endpoint := "https://api.example.com/products"

	customConfig := AggressiveCircuitBreakerConfig()
	breaker := manager.GetBreakerWithConfig(endpoint, customConfig)

	if breaker == nil {
		t.Fatal("Breaker non dovrebbe essere nil")
	}

	if breaker.config.MaxRequests != customConfig.MaxRequests {
		t.Error("Configurazione custom non applicata")
	}
}

func TestResetBreaker(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())
	endpoint := "https://api.example.com/users"

	breaker := manager.GetBreaker(endpoint)

	failFn := func() error {
		return errors.New("failure")
	}

	breaker.Execute(failFn)

	manager.ResetBreaker(endpoint)

	counts := breaker.GetCounts()
	if counts.TotalFailures != 0 {
		t.Error("Il breaker non è stato resettato")
	}
}

func TestResetBreakerNonExistent(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	manager.ResetBreaker("non-existent")
}

func TestResetAll(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	breaker1 := manager.GetBreaker("endpoint1")
	breaker2 := manager.GetBreaker("endpoint2")

	failFn := func() error {
		return errors.New("failure")
	}

	breaker1.Execute(failFn)
	breaker2.Execute(failFn)

	manager.ResetAll()

	if breaker1.GetCounts().TotalFailures != 0 {
		t.Error("Breaker1 non resettato")
	}

	if breaker2.GetCounts().TotalFailures != 0 {
		t.Error("Breaker2 non resettato")
	}
}

func TestGetAllStates(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	manager.GetBreaker("endpoint1")
	manager.GetBreaker("endpoint2")

	states := manager.GetAllStates()

	if len(states) != 2 {
		t.Errorf("Stati attesi 2, ricevuti %d", len(states))
	}

	if states["endpoint1"] != CircuitClosed {
		t.Error("Stato endpoint1 non corretto")
	}

	if states["endpoint2"] != CircuitClosed {
		t.Error("Stato endpoint2 non corretto")
	}
}

func TestGetAllCounts(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	breaker1 := manager.GetBreaker("endpoint1")
	breaker2 := manager.GetBreaker("endpoint2")

	successFn := func() error {
		return nil
	}

	breaker1.Execute(successFn)
	breaker2.Execute(successFn)
	breaker2.Execute(successFn)

	counts := manager.GetAllCounts()

	if len(counts) != 2 {
		t.Errorf("Counts attesi 2, ricevuti %d", len(counts))
	}

	if counts["endpoint1"].TotalSuccesses != 1 {
		t.Error("Count endpoint1 non corretto")
	}

	if counts["endpoint2"].TotalSuccesses != 2 {
		t.Error("Count endpoint2 non corretto")
	}
}

func TestRemoveBreaker(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	manager.GetBreaker("endpoint1")
	manager.GetBreaker("endpoint2")

	manager.RemoveBreaker("endpoint1")

	states := manager.GetAllStates()

	if len(states) != 1 {
		t.Errorf("Stati attesi 1, ricevuti %d", len(states))
	}

	if _, exists := states["endpoint1"]; exists {
		t.Error("endpoint1 non dovrebbe esistere dopo la rimozione")
	}
}

func TestClear(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	manager.GetBreaker("endpoint1")
	manager.GetBreaker("endpoint2")
	manager.GetBreaker("endpoint3")

	manager.Clear()

	states := manager.GetAllStates()

	if len(states) != 0 {
		t.Errorf("Stati attesi 0, ricevuti %d", len(states))
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.MaxRequests == 0 {
		t.Error("MaxRequests non dovrebbe essere 0")
	}

	if config.Interval == 0 {
		t.Error("Interval non dovrebbe essere 0")
	}

	if config.Timeout == 0 {
		t.Error("Timeout non dovrebbe essere 0")
	}

	if config.FailureThreshold == 0 {
		t.Error("FailureThreshold non dovrebbe essere 0")
	}

	if config.SuccessThreshold == 0 {
		t.Error("SuccessThreshold non dovrebbe essere 0")
	}

	if config.HalfOpenMaxReqs == 0 {
		t.Error("HalfOpenMaxReqs non dovrebbe essere 0")
	}
}

func TestAggressiveCircuitBreakerConfig(t *testing.T) {
	config := AggressiveCircuitBreakerConfig()

	defaultConfig := DefaultCircuitBreakerConfig()

	if config.FailureThreshold >= defaultConfig.FailureThreshold {
		t.Error("Aggressive config dovrebbe avere una soglia di fallimento più bassa")
	}

	if config.Timeout >= defaultConfig.Timeout {
		t.Error("Aggressive config dovrebbe avere un timeout più breve")
	}
}

func TestConservativeCircuitBreakerConfig(t *testing.T) {
	config := ConservativeCircuitBreakerConfig()

	defaultConfig := DefaultCircuitBreakerConfig()

	if config.FailureThreshold <= defaultConfig.FailureThreshold {
		t.Error("Conservative config dovrebbe avere una soglia di fallimento più alta")
	}

	if config.Timeout <= defaultConfig.Timeout {
		t.Error("Conservative config dovrebbe avere un timeout più lungo")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			endpoint := "endpoint"
			breaker := manager.GetBreaker(endpoint)

			successFn := func() error {
				return nil
			}

			for j := 0; j < 100; j++ {
				breaker.Execute(successFn)
			}

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	breaker := manager.GetBreaker("endpoint")
	counts := breaker.GetCounts()

	if counts.TotalSuccesses != 1000 {
		t.Errorf("TotalSuccesses atteso 1000, ricevuto %d", counts.TotalSuccesses)
	}
}
