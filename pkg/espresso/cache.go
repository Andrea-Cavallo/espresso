package espresso

import (
	"sync"
	"time"
)

// cacheEntry rappresenta un elemento nella cache con TTL
type cacheEntry[T any] struct {
	value      T
	expiration time.Time
}

// InMemoryCache implementa CacheProvider con cache in-memory thread-safe
type InMemoryCache[T any] struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry[T]
	// Cleanup goroutine control
	stopCleanup chan struct{}
	closed      bool
}

// NewInMemoryCache crea una nuova istanza di cache in-memory
func NewInMemoryCache[T any]() *InMemoryCache[T] {
	cache := &InMemoryCache[T]{
		entries:     make(map[string]*cacheEntry[T]),
		stopCleanup: make(chan struct{}),
	}

	// Avvia goroutine per pulizia automatica elementi scaduti
	go cache.cleanupLoop()

	return cache
}

// Get recupera un valore dalla cache
func (c *InMemoryCache[T]) Get(key string) (*T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Verifica se il valore è scaduto
	if time.Now().After(entry.expiration) {
		return nil, false
	}

	return &entry.value, true
}

// Set memorizza un valore nella cache con TTL
func (c *InMemoryCache[T]) Set(key string, value T, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	expiration := time.Now().Add(ttl)
	c.entries[key] = &cacheEntry[T]{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Delete rimuove un valore dalla cache
func (c *InMemoryCache[T]) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

// Clear svuota completamente la cache
func (c *InMemoryCache[T]) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry[T])
	return nil
}

// Size restituisce il numero di elementi nella cache
func (c *InMemoryCache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Close ferma la goroutine di cleanup
func (c *InMemoryCache[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.stopCleanup)
	}

	return nil
}

// cleanupLoop rimuove periodicamente gli elementi scaduti
func (c *InMemoryCache[T]) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup rimuove gli elementi scaduti
func (c *InMemoryCache[T]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiration) {
			delete(c.entries, key)
		}
	}
}

// LRUCache implementa una cache con politica Least Recently Used
type LRUCache[T any] struct {
	mu         sync.RWMutex
	capacity   int
	entries    map[string]*lruEntry[T]
	head, tail *lruEntry[T]
}

// lruEntry rappresenta un nodo nella lista LRU
type lruEntry[T any] struct {
	key        string
	value      T
	expiration time.Time
	prev, next *lruEntry[T]
}

// NewLRUCache crea una nuova cache LRU con capacità massima
func NewLRUCache[T any](capacity int) *LRUCache[T] {
	return &LRUCache[T]{
		capacity: capacity,
		entries:  make(map[string]*lruEntry[T]),
	}
}

// Get recupera un valore dalla cache LRU
func (c *LRUCache[T]) Get(key string) (*T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Verifica scadenza
	if time.Now().After(entry.expiration) {
		c.removeEntry(entry)
		return nil, false
	}

	// Sposta in testa (most recently used)
	c.moveToFront(entry)

	return &entry.value, true
}

// Set memorizza un valore nella cache LRU
func (c *LRUCache[T]) Set(key string, value T, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Se esiste già, aggiorna
	if entry, exists := c.entries[key]; exists {
		entry.value = value
		entry.expiration = time.Now().Add(ttl)
		c.moveToFront(entry)
		return nil
	}

	// Crea nuovo entry
	newEntry := &lruEntry[T]{
		key:        key,
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	// Se la cache è piena, rimuovi l'elemento meno recente
	if len(c.entries) >= c.capacity {
		c.removeLRU()
	}

	c.entries[key] = newEntry
	c.addToFront(newEntry)

	return nil
}

// Delete rimuove un elemento dalla cache
func (c *LRUCache[T]) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.entries[key]; exists {
		c.removeEntry(entry)
	}

	return nil
}

// Clear svuota la cache
func (c *LRUCache[T]) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*lruEntry[T])
	c.head = nil
	c.tail = nil

	return nil
}

// moveToFront sposta un entry in testa alla lista
func (c *LRUCache[T]) moveToFront(entry *lruEntry[T]) {
	if entry == c.head {
		return
	}

	c.removeFromList(entry)
	c.addToFront(entry)
}

// addToFront aggiunge un entry in testa alla lista
func (c *LRUCache[T]) addToFront(entry *lruEntry[T]) {
	entry.next = c.head
	entry.prev = nil

	if c.head != nil {
		c.head.prev = entry
	}

	c.head = entry

	if c.tail == nil {
		c.tail = entry
	}
}

// removeFromList rimuove un entry dalla lista
func (c *LRUCache[T]) removeFromList(entry *lruEntry[T]) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		c.head = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		c.tail = entry.prev
	}
}

// removeEntry rimuove completamente un entry
func (c *LRUCache[T]) removeEntry(entry *lruEntry[T]) {
	c.removeFromList(entry)
	delete(c.entries, entry.key)
}

// removeLRU rimuove l'elemento meno recentemente usato
func (c *LRUCache[T]) removeLRU() {
	if c.tail != nil {
		c.removeEntry(c.tail)
	}
}
