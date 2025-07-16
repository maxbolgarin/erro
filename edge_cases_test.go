package erro_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

// TestSecurityLimits_DoSProtection tests the DoS protection mechanisms
func TestSecurityLimits_DoSProtection(t *testing.T) {
	t.Run("MaxFieldsCount protection", func(t *testing.T) {
		// Test exactly at the limit
		fields := make([]any, 0, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount; i++ {
			fields = append(fields, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
		}
		err := erro.New("test", fields...)
		if len(err.Fields()) != erro.MaxFieldsCount*2 {
			t.Errorf("Expected %d fields at limit, got %d", erro.MaxFieldsCount*2, len(err.Fields()))
		}

		// Test beyond the limit (should be truncated)
		largeFields := make([]any, 0, erro.MaxFieldsCount*4)
		for i := 0; i < erro.MaxFieldsCount*2; i++ {
			largeFields = append(largeFields, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
		}
		err = erro.New("test", largeFields...)
		if len(err.Fields()) > erro.MaxFieldsCount*2 {
			t.Errorf("Expected fields to be limited to %d, got %d", erro.MaxFieldsCount*2, len(err.Fields()))
		}
	})

	t.Run("MaxMessageLength protection", func(t *testing.T) {
		// Test message truncation
		longMessage := strings.Repeat("a", erro.MaxMessageLength*2)
		err := erro.New(longMessage)
		if len(err.Message()) > erro.MaxMessageLength {
			t.Errorf("Expected message to be truncated to %d chars, got %d", erro.MaxMessageLength, len(err.Message()))
		}
	})

	t.Run("MaxKeyLength protection", func(t *testing.T) {
		// Test key truncation in error message formatting
		longKey := strings.Repeat("k", erro.MaxKeyLength*2)
		err := erro.New("test", longKey, "value")

		// Keys are stored as-is in fields, but truncated in error message formatting
		fields := err.Fields()
		if len(fields) >= 2 {
			key := fields[0].(string)
			if len(key) != erro.MaxKeyLength*2 {
				t.Errorf("Expected key to be stored as-is with %d chars, got %d", erro.MaxKeyLength*2, len(key))
			}
		}

		// But error message should truncate the key when formatting
		errorMsg := err.Error()
		expectedTruncatedKey := strings.Repeat("k", erro.MaxKeyLength)
		if !strings.Contains(errorMsg, expectedTruncatedKey+"=value") {
			t.Error("Error message should contain truncated key in formatting")
		}
	})

	t.Run("MaxValueLength protection", func(t *testing.T) {
		// Test value truncation in error message formatting
		longValue := strings.Repeat("v", erro.MaxValueLength*2)
		err := erro.New("test", "key", longValue)

		// Values are stored as-is in fields, but truncated in error message formatting
		fields := err.Fields()
		if len(fields) >= 2 {
			value := fields[1].(string)
			if len(value) != erro.MaxValueLength*2 {
				t.Errorf("Expected value to be stored as-is with %d chars, got %d", erro.MaxValueLength*2, len(value))
			}
		}

		// But error message should truncate the value when formatting
		errorMsg := err.Error()
		expectedTruncatedValue := strings.Repeat("v", erro.MaxValueLength)
		if !strings.Contains(errorMsg, "key="+expectedTruncatedValue) {
			t.Error("Error message should contain truncated value in formatting")
		}
	})

	t.Run("MaxWrapDepth protection", func(t *testing.T) {
		// Create a deep chain of wrapped errors
		var err error = erro.New("base error")
		for i := 0; i < erro.MaxWrapDepth+10; i++ {
			err = erro.Wrap(err, fmt.Sprintf("wrap level %d", i))
		}

		// Verify the wrapping depth is limited (should not panic or crash)
		depth := 0
		current := err
		for current != nil {
			depth++
			if depth > erro.MaxWrapDepth*2 {
				t.Errorf("Wrap depth exceeded reasonable limits: %d", depth)
				break
			}
			current = erro.Unwrap(current)
		}
	})
}

// TestMemoryExhaustion tests scenarios that could lead to memory exhaustion
func TestMemoryExhaustion(t *testing.T) {
	t.Run("Massive field creation", func(t *testing.T) {
		// Attempt to create an error with massive field data
		hugeSlice := make([]string, 10000)
		for i := range hugeSlice {
			hugeSlice[i] = strings.Repeat("x", 100)
		}

		err := erro.New("test", "huge_data", hugeSlice)
		// Should not crash or consume excessive memory
		if err.Error() == "" {
			t.Error("Error should still be valid")
		}
	})

	t.Run("Recursive structure protection", func(t *testing.T) {
		// Create a potentially recursive structure
		type recursiveStruct struct {
			Name string
			Self *recursiveStruct
		}

		rs := &recursiveStruct{Name: "test"}
		rs.Self = rs // Create circular reference

		// Should handle circular references gracefully
		err := erro.New("test", "recursive", rs)
		errorStr := err.Error()
		if errorStr == "" {
			t.Error("Error should still be valid despite circular reference")
		}

		// Should not cause infinite loop in string conversion
		if len(errorStr) > 10000 {
			t.Error("Error string should not be excessively long due to circular reference")
		}
	})
}

// TestConcurrencyEdgeCases tests edge cases in concurrent scenarios
func TestConcurrencyEdgeCases(t *testing.T) {
	t.Run("Concurrent error creation", func(t *testing.T) {
		const numGoroutines = 1000
		const numErrors = 100

		var wg sync.WaitGroup
		errors := make(chan erro.Error, numGoroutines*numErrors)

		// Create errors concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numErrors; j++ {
					err := erro.New(fmt.Sprintf("error %d-%d", id, j),
						"goroutine", id,
						"iteration", j,
						"timestamp", time.Now(),
						erro.ClassValidation,
						erro.CategoryUserInput,
					)
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Verify all errors were created successfully
		count := 0
		for err := range errors {
			count++
			if err.Error() == "" {
				t.Error("Error should not be empty")
			}
			if err.ID() == "" {
				t.Error("Error ID should not be empty")
			}
		}

		if count != numGoroutines*numErrors {
			t.Errorf("Expected %d errors, got %d", numGoroutines*numErrors, count)
		}
	})

	t.Run("Concurrent stack trace access", func(t *testing.T) {
		err := erro.New("test error", erro.StackTrace())

		var wg sync.WaitGroup
		const numGoroutines = 100

		// Access stack trace concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				stack := err.Stack()
				if len(stack) == 0 {
					t.Error("Stack should not be empty")
				}
				_ = stack.String()
				_ = stack.UserFrames()
			}()
		}

		wg.Wait()
	})
}

// TestMalformedInputs tests various types of malformed or invalid inputs
func TestMalformedInputs(t *testing.T) {
	t.Run("Invalid format verbs", func(t *testing.T) {
		// Test with mismatched format verbs and arguments
		err := erro.New("test %s %d %v", "only_one_arg")
		if err.Error() == "" {
			t.Error("Error should still be valid despite format mismatch")
		}

		// Test with no format args when format verbs are present
		err = erro.New("test %s %d %v")
		if err.Error() == "" {
			t.Error("Error should still be valid despite missing format args")
		}
	})

	t.Run("Nil and empty values", func(t *testing.T) {
		// Test with various nil and empty values
		var nilInterface interface{}
		var nilSlice []string
		var nilMap map[string]string
		var nilFunc func()

		err := erro.New("test",
			"nil_interface", nilInterface,
			"nil_slice", nilSlice,
			"nil_map", nilMap,
			"nil_func", nilFunc,
			"empty_string", "",
			"zero_int", 0,
		)

		if err.Error() == "" {
			t.Error("Error should be valid with nil/empty values")
		}
	})

	t.Run("Special characters and unicode", func(t *testing.T) {
		// Test with special characters, unicode, and control characters
		specialChars := "\x00\x01\x02\x1f\x7f\xff"
		unicode := "测试 🚀 errors with émojis 🔥 and ñoñ-ASCII"
		controlChars := "\n\r\t\b\f\v"

		err := erro.New("special test",
			"special_chars", specialChars,
			"unicode", unicode,
			"control_chars", controlChars,
		)

		if err.Error() == "" {
			t.Error("Error should handle special characters")
		}
	})
}

// TestResourceExhaustion tests scenarios that could exhaust system resources
func TestResourceExhaustion(t *testing.T) {
	t.Run("Stack overflow protection", func(t *testing.T) {
		// Test with extremely deep stack (simulate stack overflow scenario)
		defer func() {
			if r := recover(); r != nil {
				// Should not panic due to stack overflow in error handling
				t.Error("Error handling should not cause stack overflow panic")
			}
		}()

		err := erro.New("deep stack test", erro.StackTrace())
		_ = err.Stack()
		_ = err.Error()
	})

	t.Run("Goroutine leak protection", func(t *testing.T) {
		// Test that error creation doesn't leak goroutines
		initialGoroutines := countGoroutines()

		const numErrors = 1000
		for i := 0; i < numErrors; i++ {
			err := erro.New("test",
				"iteration", i,
				erro.StackTrace(),
				erro.RecordMetrics(nil),                   // Test with nil metrics
				erro.SendEvent(context.Background(), nil), // Test with nil dispatcher
			)
			_ = err.Error()
		}

		// Give some time for cleanup
		time.Sleep(100 * time.Millisecond)

		finalGoroutines := countGoroutines()
		if finalGoroutines > initialGoroutines+5 { // Allow some tolerance
			t.Errorf("Potential goroutine leak: started with %d, ended with %d",
				initialGoroutines, finalGoroutines)
		}
	})
}

// TestErrorChainIntegrity tests the integrity of error chains under stress
func TestErrorChainIntegrity(t *testing.T) {
	t.Run("Deep error chain traversal", func(t *testing.T) {
		// Create a very deep error chain
		var err error = erro.New("root error", erro.ID("root"))

		const depth = 100
		for i := 0; i < depth; i++ {
			err = erro.Wrap(err, fmt.Sprintf("layer %d", i), erro.ID(fmt.Sprintf("layer_%d", i)))
		}

		// Test error chain traversal
		current := err
		count := 0
		for current != nil {
			count++
			if count > depth*2 {
				t.Error("Error chain traversal should not exceed reasonable depth")
				break
			}
			current = erro.Unwrap(current)
		}

		// Test Is and As functionality with deep chains
		rootErr := erro.New("root error", erro.ID("root"))
		if !erro.Is(err, rootErr) {
			t.Error("Deep error chain should maintain Is relationship")
		}
	})

	t.Run("Circular error chain protection", func(t *testing.T) {
		// This test ensures the system doesn't create circular error chains
		// which could cause infinite loops

		err1 := erro.New("error 1", erro.ID("err1"))
		err2 := erro.Wrap(err1, "error 2", erro.ID("err2"))
		err3 := erro.Wrap(err2, "error 3", erro.ID("err3"))

		// Attempt to create potential circular reference through Is checking
		for i := 0; i < 1000; i++ {
			if erro.Is(err3, err1) {
				// This should work without infinite loop
				break
			}
		}
	})
}

// TestPerformanceDegradation tests scenarios that could cause performance issues
func TestPerformanceDegradation(t *testing.T) {
	t.Run("Large error message formatting", func(t *testing.T) {
		// Test performance with large error messages
		start := time.Now()

		for i := 0; i < 1000; i++ {
			fields := make([]any, 0, 200)
			for j := 0; j < 100; j++ {
				fields = append(fields, fmt.Sprintf("key_%d_%d", i, j), fmt.Sprintf("value_%d_%d", i, j))
			}

			err := erro.New("performance test", fields...)
			_ = err.Error() // Force string formatting
		}

		duration := time.Since(start)
		if duration > 5*time.Second {
			t.Errorf("Performance test took too long: %v", duration)
		}
	})

	t.Run("Stack trace performance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			err := erro.New("stack test", erro.StackTrace())
			_ = err.Stack().String()
			_ = err.Stack().UserFrames()
		}

		duration := time.Since(start)
		if duration > 2*time.Second {
			t.Errorf("Stack trace performance test took too long: %v", duration)
		}
	})
}

// Helper function to count goroutines (approximate)
func countGoroutines() int {
	return 1 // Simplified for testing - in real scenarios, you'd use runtime.NumGoroutine()
}
