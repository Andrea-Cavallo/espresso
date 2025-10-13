package espresso

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerConfig rappresenta la configurazione di un Circuit Breaker
type CircuitBreakerConfig struct {
	MaxRequests      uint32
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold uint32
	SuccessThreshold uint32
	HalfOpenMaxReqs  uint32
	ReadyToTrip      func(Counts) bool
	OnStateChange    func(name string, oldState, newState CircuitState)
}

// Counts tiene traccia delle statistiche del circuit breaker
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// DefaultCircuitBreaker implementa l'interfaccia CircuitBreaker
type DefaultCircuitBreaker struct {
	name   string
	config CircuitBreakerConfig
	mutex  sync.RWMutex
	state  CircuitState
	counts Counts
	expiry time.Time
}

// NewCircuitBreaker crea un nuovo circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *DefaultCircuitBreaker {
	if config.MaxRequests == 0 {
		config.MaxRequests = 10
	}
	if config.Interval == 0 {
		config.Interval = 60 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 3
	}
	if config.HalfOpenMaxReqs == 0 {
		config.HalfOpenMaxReqs = 5
	}

	cb := &DefaultCircuitBreaker{
		name:   name,
		config: config,
		state:  CircuitClosed,
		counts: Counts{},
		expiry: time.Now().Add(config.Interval),
	}

	return cb
}

// Execute esegue una funzione attraverso il circuit breaker
func (cb *DefaultCircuitBreaker) Execute(fn func() error) error {
	allowed, err := cb.beforeRequest()
	if !allowed {
		return err
	}

	err = fn()

	cb.afterRequest(err == nil)

	return err
}

// State restituisce lo stato corrente del circuit breaker
func (cb *DefaultCircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	cb.updateState()
	return cb.state
}

// Reset resetta il circuit breaker allo stato closed
func (cb *DefaultCircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.setState(CircuitClosed)
	cb.counts = Counts{}
	cb.expiry = time.Now().Add(cb.config.Interval)
}

// beforeRequest controlla se la richiesta può essere eseguita
func (cb *DefaultCircuitBreaker) beforeRequest() (bool, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.updateState()

	switch cb.state {
	case CircuitClosed:
		cb.counts.Requests++
		return true, nil

	case CircuitOpen:
		return false, fmt.Errorf("circuit breaker '%s' is open", cb.name)

	case CircuitHalfOpen:
		if cb.counts.Requests >= cb.config.HalfOpenMaxReqs {
			return false, fmt.Errorf("circuit breaker '%s' is half-open and max requests reached", cb.name)
		}
		cb.counts.Requests++
		return true, nil

	default:
		return false, fmt.Errorf("circuit breaker '%s' in unknown state", cb.name)
	}
}

// afterRequest aggiorna le statistiche dopo l'esecuzione
func (cb *DefaultCircuitBreaker) afterRequest(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}

	cb.updateState()
}

// onSuccess gestisce una richiesta riuscita
func (cb *DefaultCircuitBreaker) onSuccess() {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if cb.state == CircuitHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.config.SuccessThreshold {
		cb.setState(CircuitClosed)
		cb.counts = Counts{}
		cb.expiry = time.Now().Add(cb.config.Interval)
	}
}

// onFailure gestisce una richiesta fallita
func (cb *DefaultCircuitBreaker) onFailure() {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.shouldTrip() {
		cb.setState(CircuitOpen)
		cb.expiry = time.Now().Add(cb.config.Timeout)
	}
}

// shouldTrip determina se il circuit dovrebbe aprirsi
func (cb *DefaultCircuitBreaker) shouldTrip() bool {
	if cb.config.ReadyToTrip != nil {
		return cb.config.ReadyToTrip(cb.counts)
	}

	return cb.counts.Requests >= cb.config.MaxRequests &&
		cb.counts.ConsecutiveFailures >= cb.config.FailureThreshold
}

// updateState aggiorna lo stato del circuit breaker
func (cb *DefaultCircuitBreaker) updateState() {
	switch cb.state {
	case CircuitClosed:
		if time.Now().After(cb.expiry) {
			cb.counts = Counts{}
			cb.expiry = time.Now().Add(cb.config.Interval)
		}

	case CircuitOpen:
		if time.Now().After(cb.expiry) {
			cb.setState(CircuitHalfOpen)
			cb.counts = Counts{}
		}

	case CircuitHalfOpen:
	}
}

// setState cambia lo stato e notifica i listener
func (cb *DefaultCircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	if oldState != newState && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.name, oldState, newState)
	}
}

// GetCounts restituisce le statistiche correnti
func (cb *DefaultCircuitBreaker) GetCounts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}

// CircuitBreakerManager gestisce multiple istanze di circuit breaker
type CircuitBreakerManager struct {
	breakers map[string]*DefaultCircuitBreaker
	mutex    sync.RWMutex
	config   CircuitBreakerConfig
}

// NewCircuitBreakerManager crea un nuovo manager
func NewCircuitBreakerManager(defaultConfig CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*DefaultCircuitBreaker),
		config:   defaultConfig,
	}
}

// GetBreaker restituisce o crea un circuit breaker per un endpoint
func (cbm *CircuitBreakerManager) GetBreaker(endpoint string) *DefaultCircuitBreaker {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[endpoint]
	cbm.mutex.RUnlock()

	if exists {
		return breaker
	}

	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if breaker, exists := cbm.breakers[endpoint]; exists {
		return breaker
	}

	breaker = NewCircuitBreaker(endpoint, cbm.config)
	cbm.breakers[endpoint] = breaker

	return breaker
}

// GetBreakerWithConfig restituisce o crea un circuit breaker con configurazione specifica
func (cbm *CircuitBreakerManager) GetBreakerWithConfig(endpoint string, config CircuitBreakerConfig) *DefaultCircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	breaker := NewCircuitBreaker(endpoint, config)
	cbm.breakers[endpoint] = breaker

	return breaker
}

// ResetBreaker resetta un circuit breaker specifico
func (cbm *CircuitBreakerManager) ResetBreaker(endpoint string) {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[endpoint]
	cbm.mutex.RUnlock()

	if exists {
		breaker.Reset()
	}
}

// ResetAll resetta tutti i circuit breaker
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	for _, breaker := range cbm.breakers {
		breaker.Reset()
	}
}

// GetAllStates restituisce lo stato di tutti i circuit breaker
func (cbm *CircuitBreakerManager) GetAllStates() map[string]CircuitState {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	states := make(map[string]CircuitState)
	for endpoint, breaker := range cbm.breakers {
		states[endpoint] = breaker.State()
	}

	return states
}

// GetAllCounts restituisce le statistiche di tutti i circuit breaker
func (cbm *CircuitBreakerManager) GetAllCounts() map[string]Counts {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	counts := make(map[string]Counts)
	for endpoint, breaker := range cbm.breakers {
		counts[endpoint] = breaker.GetCounts()
	}

	return counts
}

// RemoveBreaker rimuove un circuit breaker
func (cbm *CircuitBreakerManager) RemoveBreaker(endpoint string) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	delete(cbm.breakers, endpoint)
}

// Clear rimuove tutti i circuit breaker
func (cbm *CircuitBreakerManager) Clear() {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	cbm.breakers = make(map[string]*DefaultCircuitBreaker)
}

// DefaultCircuitBreakerConfig restituisce una configurazione di default
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests:      10,
		Interval:         60 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		HalfOpenMaxReqs:  5,
	}
}

// AggressiveCircuitBreakerConfig restituisce una configurazione aggressiva
func AggressiveCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests:      5,
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		HalfOpenMaxReqs:  3,
	}
}

// ConservativeCircuitBreakerConfig restituisce una configurazione conservativa
func ConservativeCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests:      20,
		Interval:         120 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 10,
		SuccessThreshold: 5,
		HalfOpenMaxReqs:  10,
	}
}
