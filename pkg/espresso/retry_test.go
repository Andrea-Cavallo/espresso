package espresso

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

// helper per costruire una strategia con sorgente random deterministica
func newDeterministicStrategy(cfg RetryConfig, seed int64) *DefaultRetryStrategy {
	strat := NewRetryStrategy(cfg)
	strat.random = rand.New(rand.NewSource(seed))
	return strat
}

// mock net.Error per timeout/temporary
type mockNetErr struct {
	timeout   bool
	temporary bool
	wrapped   error
}

func (m mockNetErr) Error() string {
	if m.wrapped != nil {
		return m.wrapped.Error()
	}
	return "mock net err"
}
func (m mockNetErr) Timeout() bool   { return m.timeout }
func (m mockNetErr) Temporary() bool { return m.temporary }

// mock response helper
func resp(status int) *http.Response { return &http.Response{StatusCode: status} }

func Test_NewRetryStrategy_Defaults(t *testing.T) {
	t.Parallel()

	strat := NewRetryStrategy(RetryConfig{})
	if strat.config.MaxAttempts != 3 {
		t.Fatalf("MaxAttempts default, got %d want 3", strat.config.MaxAttempts)
	}
	if strat.config.BaseDelay != 100*time.Millisecond {
		t.Fatalf("BaseDelay default, got %v", strat.config.BaseDelay)
	}
	if strat.config.MaxDelay != 30*time.Second {
		t.Fatalf("MaxDelay default, got %v", strat.config.MaxDelay)
	}
	if strat.config.BackoffStrategy != BackoffExponential {
		t.Fatalf("BackoffStrategy default, got %v", strat.config.BackoffStrategy)
	}
	if strat.config.JitterType != JitterFull {
		t.Fatalf("JitterType default, got %v", strat.config.JitterType)
	}
	wantStatuses := []int{429, 500, 502, 503, 504}
	if len(strat.config.RetriableStatus) != len(wantStatuses) {
		t.Fatalf("RetriableStatus default size mismatch")
	}
	for i, s := range wantStatuses {
		if strat.config.RetriableStatus[i] != s {
			t.Fatalf("RetriableStatus default mismatch at %d got %d want %d", i, strat.config.RetriableStatus[i], s)
		}
	}
}

func Test_MaxAttempts(t *testing.T) {
	t.Parallel()
	s := NewRetryStrategy(RetryConfig{MaxAttempts: 5})
	if s.MaxAttempts() != 5 {
		t.Fatalf("MaxAttempts returned %d want 5", s.MaxAttempts())
	}
}

func Test_ShouldRetry_RespectsMaxAttempts(t *testing.T) {
	t.Parallel()
	s := NewRetryStrategy(RetryConfig{MaxAttempts: 2})
	if !s.ShouldRetry(1, nil, nil) {
		t.Fatalf("should retry at attempt 1")
	}
	if s.ShouldRetry(2, nil, nil) {
		t.Fatalf("should NOT retry at attempt == max")
	}
}

func Test_ShouldRetry_ByStatusCodes(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     3,
		RetriableStatus: []int{429, 503},
	}
	s := NewRetryStrategy(cfg)

	if !s.ShouldRetry(1, nil, resp(429)) {
		t.Fatalf("should retry on 429")
	}
	if !s.ShouldRetry(1, nil, resp(503)) {
		t.Fatalf("should retry on 503")
	}
	if s.ShouldRetry(1, nil, resp(404)) {
		t.Fatalf("should NOT retry su 404")
	}
}

func Test_ShouldRetry_ByErrors(t *testing.T) {
	t.Parallel()
	baseCfg := RetryConfig{MaxAttempts: 3}

	t.Run("custom retriable errors list", func(t *testing.T) {
		custom := errors.New("boom")
		cfg := baseCfg
		cfg.RetriableErrors = []error{custom}
		s := NewRetryStrategy(cfg)
		if !s.ShouldRetry(1, custom, nil) {
			t.Fatalf("should retry su errore custom in lista")
		}
	})

	t.Run("timeout error", func(t *testing.T) {
		s := NewRetryStrategy(baseCfg)
		err := mockNetErr{timeout: true}
		if !s.ShouldRetry(1, err, nil) {
			t.Fatalf("should retry su timeout")
		}
	})

	t.Run("temporary error", func(t *testing.T) {
		s := NewRetryStrategy(baseCfg)
		err := mockNetErr{temporary: true}
		if !s.ShouldRetry(1, err, nil) {
			t.Fatalf("should retry su temporary")
		}
	})

	t.Run("connection errors ECONNREFUSED/RESET/ABORTED", func(t *testing.T) {
		s := NewRetryStrategy(baseCfg)

		// net.OpError che wrappa syscall errors
		connRefused := &net.OpError{Err: syscall.ECONNREFUSED}
		if !s.ShouldRetry(1, connRefused, nil) {
			t.Fatalf("should retry su ECONNREFUSED")
		}

		connReset := &net.OpError{Err: syscall.ECONNRESET}
		if !s.ShouldRetry(1, connReset, nil) {
			t.Fatalf("should retry su ECONNRESET")
		}

		connAborted := &net.OpError{Err: syscall.ECONNABORTED}
		if !s.ShouldRetry(1, connAborted, nil) {
			t.Fatalf("should retry su ECONNABORTED")
		}
	})

	t.Run("non retriable error", func(t *testing.T) {
		s := NewRetryStrategy(baseCfg)
		if s.ShouldRetry(1, errors.New("not retriable"), nil) {
			t.Fatalf("should NOT retry su errore generico")
		}
	})
}

func Test_CalculateBaseDelay_Exponential(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        1500 * time.Millisecond,
		BackoffStrategy: BackoffExponential,
	}
	s := NewRetryStrategy(cfg)

	// attempt 1 -> base
	if d := s.calculateBaseDelay(1); d != 100*time.Millisecond {
		t.Fatalf("exp a1 got %v want 100ms", d)
	}
	// attempt 2 -> 200ms
	if d := s.calculateBaseDelay(2); d != 200*time.Millisecond {
		t.Fatalf("exp a2 got %v want 200ms", d)
	}
	// attempt 3 -> 400ms
	if d := s.calculateBaseDelay(3); d != 400*time.Millisecond {
		t.Fatalf("exp a3 got %v want 400ms", d)
	}
	// attempt 4 -> 800ms
	if d := s.calculateBaseDelay(4); d != 800*time.Millisecond {
		t.Fatalf("exp a4 got %v want 800ms", d)
	}
	// attempt 5 -> 1600ms capped a 1500ms
	if d := s.calculateBaseDelay(5); d != 1500*time.Millisecond {
		t.Fatalf("exp a5 got %v want 1500ms cap", d)
	}
}

func Test_CalculateBaseDelay_Linear(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        350 * time.Millisecond,
		BackoffStrategy: BackoffLinear,
	}
	s := NewRetryStrategy(cfg)

	if d := s.calculateBaseDelay(1); d != 100*time.Millisecond {
		t.Fatalf("lin a1 got %v want 100ms", d)
	}
	if d := s.calculateBaseDelay(2); d != 200*time.Millisecond {
		t.Fatalf("lin a2 got %v want 200ms", d)
	}
	if d := s.calculateBaseDelay(3); d != 300*time.Millisecond {
		t.Fatalf("lin a3 got %v want 300ms", d)
	}
	// attempt 4 -> 400ms capped a 350ms
	if d := s.calculateBaseDelay(4); d != 350*time.Millisecond {
		t.Fatalf("lin a4 got %v want 350ms cap", d)
	}
}

func Test_CalculateBaseDelay_Fixed(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       250 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		BackoffStrategy: BackoffFixed,
	}
	s := NewRetryStrategy(cfg)

	for a := 1; a <= 5; a++ {
		if d := s.calculateBaseDelay(a); d != 250*time.Millisecond {
			t.Fatalf("fixed got %v want 250ms", d)
		}
	}
}

func Test_GetDelay_NoJitter(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       200 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		JitterEnabled:   false,
		BackoffStrategy: BackoffExponential,
	}
	s := NewRetryStrategy(cfg)

	if d := s.GetDelay(1); d != 200*time.Millisecond {
		t.Fatalf("nojitter a1 got %v want 200ms", d)
	}
	if d := s.GetDelay(2); d != 400*time.Millisecond {
		t.Fatalf("nojitter a2 got %v want 400ms", d)
	}
}

func Test_Jitter_Full(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       1000 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		JitterEnabled:   true,
		JitterType:      JitterFull,
		BackoffStrategy: BackoffFixed,
	}
	s := newDeterministicStrategy(cfg, 42)

	// full jitter: random [0, base)
	for i := 0; i < 5; i++ {
		d := s.GetDelay(1)
		if d < 0 || d >= 1000*time.Millisecond {
			t.Fatalf("full jitter fuori range: %v", d)
		}
	}
}

func Test_Jitter_Equal(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       1000 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		JitterEnabled:   true,
		JitterType:      JitterEqual,
		BackoffStrategy: BackoffFixed,
	}
	s := newDeterministicStrategy(cfg, 7)

	// equal jitter: base/2 + random(0, base/2) ∈ [500ms, 1000ms)
	for i := 0; i < 5; i++ {
		d := s.GetDelay(1)
		if d < 500*time.Millisecond || d >= 1000*time.Millisecond {
			t.Fatalf("equal jitter fuori range: %v", d)
		}
	}
}

func Test_Jitter_Decorrelated(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{
		MaxAttempts:     5,
		BaseDelay:       200 * time.Millisecond,
		MaxDelay:        1500 * time.Millisecond,
		JitterEnabled:   true,
		JitterType:      JitterDecorrelated,
		BackoffStrategy: BackoffExponential,
	}
	s := newDeterministicStrategy(cfg, 99)

	// attempt 1 usa full jitter su baseDelay(1)=200ms: [0,200)
	d1 := s.GetDelay(1)
	if d1 < 0 || d1 >= 200*time.Millisecond {
		t.Fatalf("decorrelated a1 fuori range: %v", d1)
	}

	// attempt 2: prev base delay = 200ms, max= min(3*200, 1500)=600ms
	// baseDelay(2) = 400ms; random tra [base=400, max=600) -> [400,600)
	d2 := s.GetDelay(2)
	if d2 < 400*time.Millisecond || d2 >= 600*time.Millisecond {
		t.Fatalf("decorrelated a2 fuori range: %v", d2)
	}

	// attempt 5: verifica capping a MaxDelay
	// baseDelay(5) exp: 200 * 2^(4)=3200ms cap a 1500ms
	// prev baseDelay(4) = 1600 cap 1500, max= min(3*1500, 1500)=1500; base>=max => ritorna base (1500)
	d5 := s.GetDelay(5)
	if d5 != 1500*time.Millisecond {
		t.Fatalf("decorrelated a5 cap err: got %v want 1500ms", d5)
	}
}

func Test_isRetriableStatusCode(t *testing.T) {
	t.Parallel()
	cfg := RetryConfig{RetriableStatus: []int{429, 500}}
	s := NewRetryStrategy(cfg)
	if !s.isRetriableStatusCode(429) || !s.isRetriableStatusCode(500) {
		t.Fatalf("expected retriable for 429/500")
	}
	if s.isRetriableStatusCode(404) {
		t.Fatalf("404 non retriable")
	}
}

func Test_NoRetry(t *testing.T) {
	t.Parallel()
	s := NoRetry()
	if s.MaxAttempts() != 1 {
		t.Fatalf("NoRetry MaxAttempts got %d want 1", s.MaxAttempts())
	}
	if s.ShouldRetry(1, nil, nil) {
		t.Fatalf("NoRetry should never retry when attempt==1")
	}
}

func Test_ConvenienceConstructors(t *testing.T) {
	t.Parallel()

	// Exponential
	e := ExponentialBackoffRetry(4, 100*time.Millisecond, 1*time.Second, true)
	if e.config.BackoffStrategy != BackoffExponential || !e.config.JitterEnabled || e.config.JitterType != JitterFull {
		t.Fatalf("ExponentialBackoffRetry config err: %+v", e.config)
	}

	// Linear
	l := LinearBackoffRetry(4, 100*time.Millisecond, 1*time.Second, true)
	if l.config.BackoffStrategy != BackoffLinear || !l.config.JitterEnabled || l.config.JitterType != JitterEqual {
		t.Fatalf("LinearBackoffRetry config err: %+v", l.config)
	}

	// Fixed
	f := FixedDelayRetry(5, 250*time.Millisecond, false)
	if f.config.BackoffStrategy != BackoffFixed || f.config.BaseDelay != 250*time.Millisecond || f.config.MaxDelay != 250*time.Millisecond || f.config.JitterEnabled {
		t.Fatalf("FixedDelayRetry config err: %+v", f.config)
	}

	// Custom passthrough
	c := CustomRetry(RetryConfig{MaxAttempts: 9})
	if c.config.MaxAttempts != 9 {
		t.Fatalf("CustomRetry err: %+v", c.config)
	}
}

func Test_Jitter_BaseDelayZeroAndNegatives(t *testing.T) {
	t.Parallel()
	// BaseDelay zero deve produrre 0 in applyFull/Equal
	cfg := RetryConfig{
		MaxAttempts:     3,
		BaseDelay:       0,
		MaxDelay:        0,
		JitterEnabled:   true,
		JitterType:      JitterFull,
		BackoffStrategy: BackoffFixed,
	}
	s := newDeterministicStrategy(cfg, 1)

	if d := s.applyFullJitter(0); d != 0 {
		t.Fatalf("full jitter base 0 expected 0 got %v", d)
	}
	if d := s.applyEqualJitter(0); d != 0 {
		t.Fatalf("equal jitter base 0 expected 0 got %v", d)
	}
}

func Test_ShouldRetry_Priority_ResponseOverError(t *testing.T) {
	t.Parallel()
	// Se response presente, decide sullo status e ignora l'errore
	cfg := RetryConfig{
		MaxAttempts:     3,
		RetriableStatus: []int{503},
	}
	s := NewRetryStrategy(cfg)

	nonRetErr := errors.New("irrelevant")
	if !s.ShouldRetry(1, nonRetErr, resp(503)) {
		t.Fatalf("response 503 deve forzare retry")
	}
	if s.ShouldRetry(1, nonRetErr, resp(200)) {
		t.Fatalf("response 200 deve impedire retry anche se c'è errore")
	}
}

func Test_isConnectionError_WithNetErrorWrappingOpError(t *testing.T) {
	t.Parallel()
	// Simula net.Error che wrappa *net.OpError per coprire branch type assertion
	op := &net.OpError{Err: syscall.ECONNRESET}
	var ne net.Error = mockNetErr{wrapped: op} // non verrà colto da isConnectionError perché non è *net.OpError
	if isConnectionError(ne) {
		t.Fatalf("wrapped net.Error non *net.OpError non dovrebbe essere riconosciuto")
	}

	// Caso positivo: passare direttamente *net.OpError che implementa net.Error
	var ne2 net.Error = op
	if !isConnectionError(ne2) {
		t.Fatalf("opError diretto dovrebbe essere riconosciuto")
	}
}
