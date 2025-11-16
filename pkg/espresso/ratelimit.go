package espresso

import (
	"context"
	"sync"
	"time"
)

// TokenBucketRateLimiter implementa rate limiting con algoritmo Token Bucket
type TokenBucketRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	rate    int           // Token per secondo
	burst   int           // Capacità massima del bucket
	cleanup time.Duration // Intervallo di cleanup dei bucket inattivi
}

// tokenBucket rappresenta un bucket per una specifica chiave
type tokenBucket struct {
	tokens         float64
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewTokenBucketRateLimiter crea un nuovo rate limiter con token bucket
// rate: numero di richieste permesse al secondo
// burst: numero massimo di richieste burst
func NewTokenBucketRateLimiter(rate, burst int) *TokenBucketRateLimiter {
	limiter := &TokenBucketRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
		cleanup: 5 * time.Minute,
	}

	// Avvia cleanup periodico dei bucket inattivi
	go limiter.cleanupLoop()

	return limiter
}

// Allow verifica se una richiesta può essere eseguita immediatamente
func (rl *TokenBucketRateLimiter) Allow(key string) bool {
	bucket := rl.getOrCreateBucket(key)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Ricarica tokens
	rl.refillTokens(bucket)

	// Verifica se ci sono token disponibili
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// Wait attende fino a quando una richiesta può essere eseguita
func (rl *TokenBucketRateLimiter) Wait(ctx context.Context, key string) error {
	bucket := rl.getOrCreateBucket(key)

	for {
		bucket.mu.Lock()
		rl.refillTokens(bucket)

		if bucket.tokens >= 1.0 {
			bucket.tokens -= 1.0
			bucket.mu.Unlock()
			return nil
		}

		// Calcola quanto tempo aspettare
		waitTime := time.Duration((1.0-bucket.tokens)/float64(rl.rate)) * time.Second
		bucket.mu.Unlock()

		// Attesa con supporto per context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continua il loop per riprovare
		}
	}
}

// Reset resetta il rate limiter per una specifica chiave
func (rl *TokenBucketRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.buckets, key)
}

// getOrCreateBucket ottiene o crea un bucket per una chiave
func (rl *TokenBucketRateLimiter) getOrCreateBucket(key string) *tokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check dopo aver acquisito il lock
	if bucket, exists := rl.buckets[key]; exists {
		return bucket
	}

	bucket = &tokenBucket{
		tokens:         float64(rl.burst),
		lastRefillTime: time.Now(),
	}

	rl.buckets[key] = bucket
	return bucket
}

// refillTokens ricarica i token nel bucket
func (rl *TokenBucketRateLimiter) refillTokens(bucket *tokenBucket) {
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefillTime).Seconds()

	// Calcola quanti token aggiungere
	tokensToAdd := elapsed * float64(rl.rate)
	bucket.tokens = min(bucket.tokens+tokensToAdd, float64(rl.burst))
	bucket.lastRefillTime = now
}

// cleanupLoop rimuove periodicamente i bucket inattivi
func (rl *TokenBucketRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanupInactiveBuckets()
	}
}

// cleanupInactiveBuckets rimuove i bucket che non sono stati usati recentemente
func (rl *TokenBucketRateLimiter) cleanupInactiveBuckets() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-rl.cleanup)

	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		if bucket.lastRefillTime.Before(threshold) {
			delete(rl.buckets, key)
		}
		bucket.mu.Unlock()
	}
}

// SlidingWindowRateLimiter implementa rate limiting con finestra scorrevole
type SlidingWindowRateLimiter struct {
	mu       sync.RWMutex
	windows  map[string]*slidingWindow
	limit    int           // Numero massimo di richieste per finestra
	window   time.Duration // Dimensione della finestra
	segments int           // Numero di segmenti nella finestra
}

// slidingWindow rappresenta una finestra scorrevole per tracking richieste
type slidingWindow struct {
	mu       sync.Mutex
	counts   []int
	times    []time.Time
	position int
}

// NewSlidingWindowRateLimiter crea un nuovo rate limiter con finestra scorrevole
// limit: numero massimo di richieste permesse nella finestra
// window: dimensione della finestra temporale
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windows:  make(map[string]*slidingWindow),
		limit:    limit,
		window:   window,
		segments: 10,
	}
}

// Allow verifica se una richiesta può essere eseguita
func (rl *SlidingWindowRateLimiter) Allow(key string) bool {
	window := rl.getOrCreateWindow(key)

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)

	// Conta richieste nella finestra
	count := 0
	for i, t := range window.times {
		if t.After(threshold) {
			count += window.counts[i]
		}
	}

	if count >= rl.limit {
		return false
	}

	// Aggiungi nuova richiesta
	window.times = append(window.times, now)
	window.counts = append(window.counts, 1)

	// Rimuovi vecchie entries
	rl.cleanupWindow(window, threshold)

	return true
}

// Wait non è implementato per sliding window (usa Allow invece)
func (rl *SlidingWindowRateLimiter) Wait(ctx context.Context, key string) error {
	// Implementazione semplice: polling con backoff
	backoff := 10 * time.Millisecond

	for {
		if rl.Allow(key) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = min(backoff*2, 1*time.Second)
		}
	}
}

// Reset resetta il rate limiter per una chiave
func (rl *SlidingWindowRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.windows, key)
}

// getOrCreateWindow ottiene o crea una finestra per una chiave
func (rl *SlidingWindowRateLimiter) getOrCreateWindow(key string) *slidingWindow {
	rl.mu.RLock()
	window, exists := rl.windows[key]
	rl.mu.RUnlock()

	if exists {
		return window
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if window, exists := rl.windows[key]; exists {
		return window
	}

	window = &slidingWindow{
		counts: make([]int, 0),
		times:  make([]time.Time, 0),
	}

	rl.windows[key] = window
	return window
}

// cleanupWindow rimuove entries vecchie dalla finestra
func (rl *SlidingWindowRateLimiter) cleanupWindow(window *slidingWindow, threshold time.Time) {
	validIndex := 0
	for i, t := range window.times {
		if t.After(threshold) {
			window.times[validIndex] = window.times[i]
			window.counts[validIndex] = window.counts[i]
			validIndex++
		}
	}

	window.times = window.times[:validIndex]
	window.counts = window.counts[:validIndex]
}

// FixedWindowRateLimiter implementa rate limiting con finestra fissa
type FixedWindowRateLimiter struct {
	mu      sync.RWMutex
	windows map[string]*fixedWindow
	limit   int
	window  time.Duration
}

// fixedWindow rappresenta una finestra fissa
type fixedWindow struct {
	mu        sync.Mutex
	count     int
	resetTime time.Time
}

// NewFixedWindowRateLimiter crea un nuovo rate limiter con finestra fissa
func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		windows: make(map[string]*fixedWindow),
		limit:   limit,
		window:  window,
	}
}

// Allow verifica se una richiesta può essere eseguita
func (rl *FixedWindowRateLimiter) Allow(key string) bool {
	window := rl.getOrCreateWindow(key)

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()

	// Reset se la finestra è scaduta
	if now.After(window.resetTime) {
		window.count = 0
		window.resetTime = now.Add(rl.window)
	}

	if window.count >= rl.limit {
		return false
	}

	window.count++
	return true
}

// Wait attende fino a quando una richiesta può essere eseguita
func (rl *FixedWindowRateLimiter) Wait(ctx context.Context, key string) error {
	for {
		if rl.Allow(key) {
			return nil
		}

		window := rl.getOrCreateWindow(key)
		window.mu.Lock()
		waitTime := time.Until(window.resetTime)
		window.mu.Unlock()

		if waitTime <= 0 {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Riprova dopo il reset
		}
	}
}

// Reset resetta il rate limiter per una chiave
func (rl *FixedWindowRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.windows, key)
}

// getOrCreateWindow ottiene o crea una finestra per una chiave
func (rl *FixedWindowRateLimiter) getOrCreateWindow(key string) *fixedWindow {
	rl.mu.RLock()
	window, exists := rl.windows[key]
	rl.mu.RUnlock()

	if exists {
		return window
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if window, exists := rl.windows[key]; exists {
		return window
	}

	window = &fixedWindow{
		count:     0,
		resetTime: time.Now().Add(rl.window),
	}

	rl.windows[key] = window
	return window
}

// Helper function per min
func min[T int | float64 | time.Duration](a, b T) T {
	if a < b {
		return a
	}
	return b
}
