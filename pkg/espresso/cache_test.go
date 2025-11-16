package espresso

import (
	"testing"
	"time"
)

func TestInMemoryCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryCache[string]()
	defer cache.Close()

	key := "test-key"
	value := "test-value"

	err := cache.Set(key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Key not found in cache")
	}

	if *retrieved != value {
		t.Fatalf("Expected %s, got %s", value, *retrieved)
	}
}

func TestInMemoryCache_Expiration(t *testing.T) {
	cache := NewInMemoryCache[string]()
	defer cache.Close()

	key := "expiring-key"
	value := "expiring-value"

	err := cache.Set(key, value, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be present
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Key should be present")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get(key)
	if found {
		t.Fatal("Key should be expired")
	}
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache[string]()
	defer cache.Close()

	key := "delete-key"
	value := "delete-value"

	cache.Set(key, value, 1*time.Minute)

	err := cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, found := cache.Get(key)
	if found {
		t.Fatal("Key should be deleted")
	}
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache[string]()
	defer cache.Close()

	cache.Set("key1", "value1", 1*time.Minute)
	cache.Set("key2", "value2", 1*time.Minute)
	cache.Set("key3", "value3", 1*time.Minute)

	if cache.Size() != 3 {
		t.Fatalf("Expected size 3, got %d", cache.Size())
	}

	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if cache.Size() != 0 {
		t.Fatalf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	capacity := 3
	cache := NewLRUCache[string](capacity)

	cache.Set("key1", "value1", 1*time.Minute)
	cache.Set("key2", "value2", 1*time.Minute)
	cache.Set("key3", "value3", 1*time.Minute)

	// All keys should be present
	for i := 1; i <= 3; i++ {
		key := "key" + string(rune('0'+i))
		_, found := cache.Get(key)
		if !found {
			t.Fatalf("Key %s should be present", key)
		}
	}

	// Add fourth item, should evict key1 (least recently used)
	cache.Set("key4", "value4", 1*time.Minute)

	// key1 should be evicted
	_, found := cache.Get("key1")
	if found {
		t.Fatal("key1 should be evicted")
	}

	// key4 should be present
	_, found = cache.Get("key4")
	if !found {
		t.Fatal("key4 should be present")
	}
}

func TestLRUCache_UpdateMakesRecent(t *testing.T) {
	cache := NewLRUCache[string](2)

	cache.Set("key1", "value1", 1*time.Minute)
	cache.Set("key2", "value2", 1*time.Minute)

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add key3, should evict key2 (not key1)
	cache.Set("key3", "value3", 1*time.Minute)

	_, found := cache.Get("key1")
	if !found {
		t.Fatal("key1 should still be present")
	}

	_, found = cache.Get("key2")
	if found {
		t.Fatal("key2 should be evicted")
	}
}
