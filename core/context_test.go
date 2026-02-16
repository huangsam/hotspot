package core

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContextConcurrentAccess tests that context values can be safely accessed concurrently.
func TestContextConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	// Test concurrent reads of context values
	const numGoroutines = 50
	done := make(chan bool, numGoroutines)

	// Set up context with various values
	ctx = withSuppressHeader(ctx)
	ctx = withUseFollow(ctx, true)
	ctx = withAnalysisID(ctx, 12345)

	for i := range numGoroutines {
		go func(id int) {
			defer func() { done <- true }()

			// Concurrent reads should be safe
			suppress := shouldSuppressHeader(ctx)
			useFollow := shouldUseFollow(ctx)
			analysisID, ok := getAnalysisID(ctx)

			// Verify values are correct
			assert.True(t, suppress, "Goroutine %d: shouldSuppressHeader should be true", id)
			assert.True(t, useFollow, "Goroutine %d: shouldUseFollow should be true", id)
			assert.True(t, ok, "Goroutine %d: getAnalysisID should return true", id)
			assert.Equal(t, int64(12345), analysisID, "Goroutine %d: analysisID should be 12345", id)
		}(i)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}
}

// TestContextIsolation tests that different contexts maintain isolation.
func TestContextIsolation(t *testing.T) {
	baseCtx := context.Background()

	// Create multiple contexts with different values
	ctx1 := withAnalysisID(baseCtx, 1)
	ctx2 := withAnalysisID(baseCtx, 2)
	ctx3 := withSuppressHeader(baseCtx)

	// Test concurrent access to different contexts
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		id1, ok1 := getAnalysisID(ctx1)
		assert.True(t, ok1)
		assert.Equal(t, int64(1), id1)
		assert.False(t, shouldSuppressHeader(ctx1))
	}()

	go func() {
		defer wg.Done()
		id2, ok2 := getAnalysisID(ctx2)
		assert.True(t, ok2)
		assert.Equal(t, int64(2), id2)
		assert.False(t, shouldSuppressHeader(ctx2))
	}()

	go func() {
		defer wg.Done()
		id3, ok3 := getAnalysisID(ctx3)
		assert.False(t, ok3)
		assert.Equal(t, int64(0), id3)
		assert.True(t, shouldSuppressHeader(ctx3))
	}()

	wg.Wait()
}
