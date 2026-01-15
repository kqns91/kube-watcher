package dedup

import (
	"testing"
	"time"
)

func TestNewDeduplicator(t *testing.T) {
	ttl := time.Minute
	maxSize := 100

	d := NewDeduplicator(ttl, maxSize)
	// NewDeduplicator always returns non-nil, but verify
	if d == nil {
		t.Fatal("NewDeduplicator returned nil")
		return // unreachable, but helps staticcheck
	}
	defer d.Stop()

	if d.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, d.ttl)
	}

	if d.maxSize != maxSize {
		t.Errorf("Expected maxSize %d, got %d", maxSize, d.maxSize)
	}
}

func TestDeduplicator_ShouldProcess_NewEvent(t *testing.T) {
	d := NewDeduplicator(time.Minute, 100)
	defer d.Stop()

	key := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod",
		EventType: "UPDATED",
	}

	data := map[string]string{"status": "Running"}

	// First time should process
	if !d.ShouldProcess(key, data) {
		t.Error("First event should be processed")
	}

	// Second time with same data should not process
	if d.ShouldProcess(key, data) {
		t.Error("Duplicate event should not be processed")
	}
}

func TestDeduplicator_ShouldProcess_DifferentData(t *testing.T) {
	d := NewDeduplicator(time.Minute, 100)
	defer d.Stop()

	key := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod",
		EventType: "UPDATED",
	}

	data1 := map[string]string{"status": "Pending"}
	data2 := map[string]string{"status": "Running"}

	// First event
	if !d.ShouldProcess(key, data1) {
		t.Error("First event should be processed")
	}

	// Different data should process
	if !d.ShouldProcess(key, data2) {
		t.Error("Event with different data should be processed")
	}
}

func TestDeduplicator_ShouldProcess_DifferentKeys(t *testing.T) {
	d := NewDeduplicator(time.Minute, 100)
	defer d.Stop()

	key1 := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod-1",
		EventType: "UPDATED",
	}

	key2 := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod-2",
		EventType: "UPDATED",
	}

	data := map[string]string{"status": "Running"}

	// First pod
	if !d.ShouldProcess(key1, data) {
		t.Error("First event should be processed")
	}

	// Different pod should process even with same data
	if !d.ShouldProcess(key2, data) {
		t.Error("Event for different resource should be processed")
	}
}

func TestDeduplicator_ShouldProcess_TTLExpired(t *testing.T) {
	ttl := 100 * time.Millisecond
	d := NewDeduplicator(ttl, 100)
	defer d.Stop()

	key := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod",
		EventType: "UPDATED",
	}

	data := map[string]string{"status": "Running"}

	// First event
	if !d.ShouldProcess(key, data) {
		t.Error("First event should be processed")
	}

	// Wait for TTL to expire
	time.Sleep(ttl + 50*time.Millisecond)

	// After TTL, same event should process again
	if !d.ShouldProcess(key, data) {
		t.Error("Event should be processed after TTL expires")
	}
}

func TestDeduplicator_CacheEviction(t *testing.T) {
	maxSize := 3
	d := NewDeduplicator(time.Minute, maxSize)
	defer d.Stop()

	// Add more events than max size
	for i := 0; i < maxSize+2; i++ {
		key := EventKey{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "test-pod-" + string(rune('0'+i)),
			EventType: "UPDATED",
		}
		data := map[string]string{"index": string(rune('0' + i))}

		d.ShouldProcess(key, data)
	}

	stats := d.Stats()
	size := stats["size"].(int)

	if size > maxSize {
		t.Errorf("Cache size %d exceeds maxSize %d", size, maxSize)
	}
}

func TestDeduplicator_Cleanup(t *testing.T) {
	ttl := 100 * time.Millisecond
	d := NewDeduplicator(ttl, 100)
	defer d.Stop()

	// Add some events
	for i := 0; i < 5; i++ {
		key := EventKey{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "test-pod-" + string(rune('0'+i)),
			EventType: "UPDATED",
		}
		data := map[string]string{"index": string(rune('0' + i))}
		d.ShouldProcess(key, data)
	}

	// Check initial size
	stats := d.Stats()
	initialSize := stats["size"].(int)

	if initialSize != 5 {
		t.Errorf("Expected initial size 5, got %d", initialSize)
	}

	// Wait for cleanup
	time.Sleep(ttl + 200*time.Millisecond)

	// Trigger cleanup manually
	d.cleanup()

	// Check size after cleanup
	stats = d.Stats()
	finalSize := stats["size"].(int)

	if finalSize != 0 {
		t.Errorf("Expected cache to be empty after cleanup, got size %d", finalSize)
	}
}

func TestDeduplicator_Stats(t *testing.T) {
	ttl := time.Minute
	maxSize := 100
	d := NewDeduplicator(ttl, maxSize)
	defer d.Stop()

	stats := d.Stats()

	if stats["max_size"].(int) != maxSize {
		t.Errorf("Expected max_size %d, got %d", maxSize, stats["max_size"])
	}

	if stats["ttl"].(string) != ttl.String() {
		t.Errorf("Expected ttl %s, got %s", ttl.String(), stats["ttl"])
	}

	if stats["size"].(int) != 0 {
		t.Errorf("Expected initial size 0, got %d", stats["size"])
	}
}

func TestDeduplicator_ConcurrentAccess(t *testing.T) {
	d := NewDeduplicator(time.Minute, 1000)
	defer d.Stop()

	done := make(chan bool)

	// Run multiple goroutines concurrently
	for i := 0; i < 10; i++ {
		go func(index int) {
			for j := 0; j < 100; j++ {
				key := EventKey{
					Kind:      "Pod",
					Namespace: "default",
					Name:      "test-pod",
					EventType: "UPDATED",
				}
				data := map[string]interface{}{
					"index": index,
					"count": j,
				}
				d.ShouldProcess(key, data)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we reach here without panic, concurrent access is safe
	stats := d.Stats()
	if stats["size"].(int) < 0 {
		t.Error("Cache size should not be negative")
	}
}

func BenchmarkDeduplicator_ShouldProcess(b *testing.B) {
	d := NewDeduplicator(time.Minute, 10000)
	defer d.Stop()

	key := EventKey{
		Kind:      "Pod",
		Namespace: "default",
		Name:      "test-pod",
		EventType: "UPDATED",
	}

	data := map[string]string{"status": "Running"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.ShouldProcess(key, data)
	}
}
