package erro_test

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

// TestHighVolumeErrorCreation tests performance under high error creation load
func TestHighVolumeErrorCreation(t *testing.T) {
	t.Run("Sequential error creation performance", func(t *testing.T) {
		const numErrors = 100000
		start := time.Now()

		for i := 0; i < numErrors; i++ {
			err := erro.New("performance test",
				"iteration", i,
				"type", "sequential",
				erro.ClassValidation,
			)
			_ = err.Error() // Force string creation
		}

		duration := time.Since(start)
		avgPerError := duration / numErrors

		// Should be able to create errors quickly (increased threshold for CI)
		if avgPerError > 50*time.Microsecond {
			t.Errorf("Sequential error creation too slow: %v per error", avgPerError)
		}

		t.Logf("Sequential: %d errors in %v (avg: %v per error)", numErrors, duration, avgPerError)
	})

	t.Run("Concurrent error creation performance", func(t *testing.T) {
		const numGoroutines = 100
		const errorsPerGoroutine = 1000

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < errorsPerGoroutine; j++ {
					err := erro.New("concurrent test",
						"goroutine", goroutineID,
						"iteration", j,
						erro.ClassValidation,
					)
					_ = err.Error()
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		totalErrors := numGoroutines * errorsPerGoroutine
		avgPerError := duration / time.Duration(totalErrors)

		// Concurrent creation should still be performant (increased threshold for CI)
		if avgPerError > 100*time.Microsecond {
			t.Errorf("Concurrent error creation too slow: %v per error", avgPerError)
		}

		t.Logf("Concurrent: %d errors in %v (avg: %v per error)", totalErrors, duration, avgPerError)
	})
}

// TestMemoryEfficiency tests memory efficiency under load
func TestMemoryEfficiency(t *testing.T) {
	t.Run("Memory usage with large error volumes", func(t *testing.T) {
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		const numErrors = 10000
		errors := make([]erro.Error, numErrors)

		// Create many errors
		for i := 0; i < numErrors; i++ {
			errors[i] = erro.New("memory test",
				"index", i,
				"data", strings.Repeat("x", 100), // 100 bytes per error
				erro.ClassValidation,
			)
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		allocatedBytes := m2.Alloc - m1.Alloc
		bytesPerError := allocatedBytes / numErrors

		// Should not use excessive memory per error
		if bytesPerError > 2000 { // 2KB per error seems reasonable
			t.Errorf("Memory usage too high: %d bytes per error", bytesPerError)
		}

		t.Logf("Memory: %d errors used %d bytes (avg: %d bytes per error)",
			numErrors, allocatedBytes, bytesPerError)

		// Clear references to allow GC
		for i := range errors {
			errors[i] = nil
		}
		errors = nil
		runtime.GC()
	})

	t.Run("Memory efficiency with string caching", func(t *testing.T) {
		// Test that error string caching doesn't cause memory leaks
		const numErrors = 1000
		errors := make([]erro.Error, numErrors)

		for i := 0; i < numErrors; i++ {
			errors[i] = erro.New("caching test", "index", i)
		}

		// Call Error() multiple times to test caching
		for round := 0; round < 3; round++ {
			start := time.Now()
			for _, err := range errors {
				_ = err.Error()
			}
			duration := time.Since(start)

			// Subsequent calls should be faster due to caching
			if round > 0 {
				maxDuration := 50 * time.Millisecond
				if duration > maxDuration {
					t.Errorf("Round %d: Cached error strings took too long: %v", round, duration)
				}
			}
		}
	})
}

// TestStackTracePerformance tests stack trace performance impact
func TestStackTracePerformance(t *testing.T) {
	t.Run("Stack trace vs no stack trace performance", func(t *testing.T) {
		const numErrors = 1000

		// Test without stack traces
		start := time.Now()
		for i := 0; i < numErrors; i++ {
			err := erro.New("no stack test", "index", i)
			_ = err.Error()
		}
		noStackDuration := time.Since(start)

		// Test with stack traces
		start = time.Now()
		for i := 0; i < numErrors; i++ {
			err := erro.New("stack test", "index", i, erro.StackTrace())
			_ = err.Error()
		}
		stackDuration := time.Since(start)

		// Stack traces should add overhead but not be excessive
		overhead := float64(stackDuration) / float64(noStackDuration)
		if overhead > 10.0 { // 10x seems like a reasonable limit
			t.Errorf("Stack trace overhead too high: %.2fx slower", overhead)
		}

		t.Logf("Stack trace overhead: %.2fx (no stack: %v, with stack: %v)",
			overhead, noStackDuration, stackDuration)
	})

	t.Run("Stack trace depth performance", func(t *testing.T) {
		// Test performance with deep stack traces
		deepErr := createVeryDeepStackTrace(100)

		start := time.Now()
		for i := 0; i < 100; i++ {
			_ = deepErr.Stack().String()
		}
		duration := time.Since(start)

		avgPerCall := duration / 100
		if avgPerCall > 10*time.Millisecond {
			t.Errorf("Deep stack trace formatting too slow: %v per call", avgPerCall)
		}
	})
}

// TestConcurrencyPerformance tests performance under concurrent access
func TestConcurrencyPerformance(t *testing.T) {
	t.Run("Concurrent error access performance", func(t *testing.T) {
		// Create a single error accessed by multiple goroutines
		err := erro.New("concurrent access test",
			"data", strings.Repeat("x", 1000),
			erro.StackTrace(),
		)

		const numGoroutines = 100
		const accessesPerGoroutine = 1000

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < accessesPerGoroutine; j++ {
					_ = err.Error()
					_ = err.ID()
					_ = err.Class()
					_ = err.Fields()
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		totalAccesses := numGoroutines * accessesPerGoroutine * 4 // 4 method calls per iteration
		avgPerAccess := duration / time.Duration(totalAccesses)

		// Go 1.18 has different performance characteristics for atomic operations with generics
		// compared to newer versions, so we use a more tolerant threshold
		maxAllowedPerAccess := 5 * time.Microsecond
		if avgPerAccess > maxAllowedPerAccess {
			t.Errorf("Concurrent access too slow: %v per access (max allowed: %v)", avgPerAccess, maxAllowedPerAccess)
		}
	})

	t.Run("Error collection performance under load", func(t *testing.T) {
		const numGoroutines = 50
		const errorsPerGoroutine = 1000

		// Test List performance
		list := erro.NewSafeList()
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < errorsPerGoroutine; j++ {
					list.New("list test", "goroutine", id, "iteration", j)
				}
			}(i)
		}

		wg.Wait()
		listDuration := time.Since(start)

		// Test Set performance
		set := erro.NewSafeSet()
		start = time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < errorsPerGoroutine; j++ {
					set.New("set test", "goroutine", id, "iteration", j)
				}
			}(i)
		}

		wg.Wait()
		setDuration := time.Since(start)

		t.Logf("Collection performance - List: %v, Set: %v", listDuration, setDuration)

		// Verify final counts
		expectedCount := numGoroutines * errorsPerGoroutine
		if list.Len() != expectedCount {
			t.Errorf("List count mismatch: expected %d, got %d", expectedCount, list.Len())
		}

		// Set might have fewer due to deduplication, but should have some
		if set.Len() == 0 {
			t.Error("Set should have some errors")
		}
	})
}

// TestScalabilityLimits tests behavior at scalability limits
func TestScalabilityLimits(t *testing.T) {
	t.Run("Maximum field handling performance", func(t *testing.T) {
		// Create error with maximum allowed fields
		fields := make([]any, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount*2; i += 2 {
			fields[i] = "key"
			fields[i+1] = "value"
		}

		start := time.Now()
		for i := 0; i < 100; i++ {
			err := erro.New("max fields test", fields...)
			_ = err.Error()
		}
		duration := time.Since(start)

		avgPerError := duration / 100
		if avgPerError > 50*time.Millisecond {
			t.Errorf("Max fields handling too slow: %v per error", avgPerError)
		}
	})

	t.Run("Deep error chain performance", func(t *testing.T) {
		// Create deep error chain
		var err error = erro.New("base error")
		for i := 0; i < 100; i++ {
			err = erro.Wrap(err, "layer", "depth", i)
		}

		start := time.Now()
		for i := 0; i < 100; i++ {
			_ = err.Error()
		}
		duration := time.Since(start)

		avgPerCall := duration / 100
		if avgPerCall > 10*time.Millisecond {
			t.Errorf("Deep chain handling too slow: %v per call", avgPerCall)
		}

		// Test unwrapping performance
		start = time.Now()
		current := err
		depth := 0
		for current != nil && depth < 1000 {
			current = erro.Unwrap(current)
			depth++
		}
		unwrapDuration := time.Since(start)

		if unwrapDuration > 10*time.Millisecond {
			t.Errorf("Deep chain unwrapping too slow: %v for depth %d", unwrapDuration, depth)
		}
	})
}

// TestResourceLeakPrevention tests that no resources are leaked
func TestResourceLeakPrevention(t *testing.T) {
	t.Run("Goroutine leak prevention", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		// Create many errors with various features that might spawn goroutines
		for i := 0; i < 1000; i++ {
			err := erro.New("leak test",
				"iteration", i,
				erro.StackTrace(),
				erro.RecordMetrics(nil),
				erro.SendEvent(context.Background(), nil),
			)
			_ = err.Error()
		}

		// Give time for any background goroutines to finish
		time.Sleep(100 * time.Millisecond)
		runtime.GC()

		finalGoroutines := runtime.NumGoroutine()

		// Should not have created persistent goroutines
		if finalGoroutines > initialGoroutines+5 { // Allow some tolerance
			t.Errorf("Potential goroutine leak: %d -> %d goroutines",
				initialGoroutines, finalGoroutines)
		}
	})

	t.Run("Memory leak prevention with error chains", func(t *testing.T) {
		// Force multiple GC cycles to establish baseline
		for i := 0; i < 3; i++ {
			runtime.GC()
			runtime.GC()
		}
		time.Sleep(100 * time.Millisecond) // Allow GC to complete

		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Create many error chains and let them go out of scope
		func() {
			for i := 0; i < 1000; i++ {
				var err error = erro.New("base", "id", i)
				for j := 0; j < 10; j++ {
					err = erro.Wrap(err, "wrap", "level", j)
				}
				_ = err.Error() // Use the error
			}
		}() // Ensure errors go out of scope

		// Force garbage collection and wait
		for i := 0; i < 3; i++ {
			runtime.GC()
			runtime.GC()
		}
		time.Sleep(100 * time.Millisecond) // Allow GC to complete

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		// Check for memory growth - handle potential underflow
		var allocGrowth uint64
		if m2.Alloc > m1.Alloc {
			allocGrowth = m2.Alloc - m1.Alloc
		} else {
			// Memory decreased, which is good
			allocGrowth = 0
		}

		// Memory should be bounded (not growing linearly with iterations)
		// Each error with 10 wraps might use ~1KB, so 1000 errors = ~1MB base + overhead
		maxExpectedGrowth := uint64(5 * 1024 * 1024) // 5MB seems more reasonable
		if allocGrowth > maxExpectedGrowth {
			t.Errorf("Potential memory leak: %d bytes allocated (max expected: %d)",
				allocGrowth, maxExpectedGrowth)
		}

		// Also check total allocations to detect excessive allocation/deallocation
		totalAllocGrowth := m2.TotalAlloc - m1.TotalAlloc
		maxExpectedTotalAlloc := uint64(50 * 1024 * 1024) // 50MB total seems reasonable
		if totalAllocGrowth > maxExpectedTotalAlloc {
			t.Errorf("Excessive total allocations: %d bytes (max expected: %d)",
				totalAllocGrowth, maxExpectedTotalAlloc)
		}
	})
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	t.Run("Basic error creation benchmark", func(t *testing.T) {
		// This test sets performance expectations for basic operations
		const numIterations = 10000

		start := time.Now()
		for i := 0; i < numIterations; i++ {
			err := erro.New("benchmark test", "iteration", i)
			_ = err.Error()
		}
		duration := time.Since(start)

		avgPerOp := duration / numIterations

		// Set reasonable performance expectations (increased for CI)
		if avgPerOp > 25*time.Microsecond {
			t.Errorf("Performance regression: %v per operation (expected < 25µs)", avgPerOp)
		}

		t.Logf("Basic error creation: %v per operation", avgPerOp)
	})

	t.Run("Error wrapping benchmark", func(t *testing.T) {
		baseErr := erro.New("base error")
		const numIterations = 10000

		start := time.Now()
		for i := 0; i < numIterations; i++ {
			err := erro.Wrap(baseErr, "wrapped", "iteration", i)
			_ = err.Error()
		}
		duration := time.Since(start)

		avgPerOp := duration / numIterations

		if avgPerOp > 30*time.Microsecond {
			t.Errorf("Wrapping performance regression: %v per operation (expected < 30µs)", avgPerOp)
		}

		t.Logf("Error wrapping: %v per operation", avgPerOp)
	})
}

// Helper function to create very deep stack trace
func createVeryDeepStackTrace(depth int) erro.Error {
	if depth <= 0 {
		return erro.New("very deep stack trace", erro.StackTrace())
	}
	return createVeryDeepStackTrace(depth - 1)
}
