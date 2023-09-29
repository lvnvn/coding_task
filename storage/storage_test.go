// Unit tests for storage package
package storage

import (
	"sync"
	"testing"
	"time"
)

func assertEqualInt(t *testing.T, expected int, got int) {
	if got != expected {
		t.Errorf("Expected result: %d, actual result: %d", expected, got)
	}
}

// Check Add() for concurrent write safety
func TestAdd(t *testing.T) {
	var wg sync.WaitGroup
	c := PersistentCounter{}
	timestamp := time.Now().Unix()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			c.Add(timestamp)
			wg.Done()
		}()
	}
	wg.Wait()

	assertEqualInt(t, 50, len(c.counter.ts))
}

// Check Count() without reading from backup file
func TestCount(t *testing.T) {
	now := time.Now()
	timestamps := []int64{
		// More than a minute ago
		now.Add(-100 * time.Second).Unix(),
		now.Add(-80 * time.Second).Unix(),
		// Less than a minute ago
		now.Add(-55 * time.Second).Unix(),
		now.Add(-20 * time.Second).Unix(),
		now.Add(-20 * time.Second).Unix(),
	}
	c := PersistentCounter{
		counter: Counter{ts: timestamps},
	}
	assertEqualInt(t, 3, c.Count())
}
