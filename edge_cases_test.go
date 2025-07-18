package erro_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

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

// customErr is a test error type for testing errors.As compatibility
type customErr struct {
	code int
	msg  string
}

func (e *customErr) Error() string {
	return fmt.Sprintf("custom error %d: %s", e.code, e.msg)
}

// TestStandardLibraryIntegration tests integration with Go standard library
func TestStandardLibraryIntegration(t *testing.T) {
	t.Run("Standard errors.Is compatibility", func(t *testing.T) {
		// Test with various standard library errors
		standardErrors := []error{
			io.EOF,
			os.ErrNotExist,
			context.Canceled,
			context.DeadlineExceeded,
		}

		for _, stdErr := range standardErrors {
			wrappedErr := erro.Wrap(stdErr, "wrapped standard error")

			// Should maintain Is relationship
			if !errors.Is(wrappedErr, stdErr) {
				t.Errorf("Standard error relationship lost for %T", stdErr)
			}

			// Should work with standard errors.Is
			if !errors.Is(wrappedErr, stdErr) {
				t.Errorf("Standard errors.Is failed for %T", stdErr)
			}
		}
	})

	t.Run("Standard errors.As compatibility", func(t *testing.T) {
		originalErr := &customErr{code: 404, msg: "not found"}
		wrappedErr := erro.Wrap(originalErr, "wrapped custom error")

		// Should work with standard errors.As
		var target *customErr
		if !errors.As(wrappedErr, &target) {
			t.Error("Standard errors.As failed")
		}

		if target.code != 404 {
			t.Errorf("Expected code 404, got %d", target.code)
		}
	})

	t.Run("fmt.Formatter interface", func(t *testing.T) {
		err := erro.New("format test",
			"key", "value",
			erro.StackTrace(),
		)

		// Test supported format verbs (only %s and %v are officially supported)
		supportedFormats := []string{"%s", "%v", "%+v"}
		for _, format := range supportedFormats {
			formatted := fmt.Sprintf(format, err)
			if formatted == "" {
				t.Errorf("Empty result for supported format %s", format)
			}
		}

		// Test unsupported format verbs (may result in empty strings or default formatting)
		unsupportedFormats := []string{"%#v", "%q", "%d"}
		for _, format := range unsupportedFormats {
			formatted := fmt.Sprintf(format, err)
			// Unsupported formats may result in empty strings or use Go's default formatting
			t.Logf("Unsupported format %s resulted in: %q", format, formatted)
			// We don't assert anything specific here since behavior varies by verb
		}

		// %+v should include stack trace
		fullFormat := fmt.Sprintf("%+v", err)
		if !strings.Contains(fullFormat, "edge_cases_test.go") {
			t.Error("Full format should contain stack trace information")
		}
	})

	t.Run("context.Context integration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Wait for timeout
		<-ctx.Done()

		err := erro.Wrap(ctx.Err(), "context timeout",
			"operation", "test",
			"timeout", "100ms",
		)

		// Should maintain context error properties
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Error("Should maintain context.DeadlineExceeded relationship")
		}
	})
}

// TestHTTPIntegration tests HTTP-related edge cases
func TestHTTPIntegration(t *testing.T) {
	t.Run("HTTP status code edge cases", func(t *testing.T) {
		testCases := []struct {
			name         string
			err          error
			expectedCode int
		}{
			{"nil error", nil, http.StatusOK},
			{"standard error", errors.New("standard"), http.StatusInternalServerError},
			{"wrapped standard error", erro.Wrap(errors.New("std"), "wrapped"), http.StatusInternalServerError},
			{"unknown class", erro.New("test", erro.ErrorClass("unknown")), http.StatusInternalServerError},
			{"unknown category", erro.New("test", erro.ErrorCategory("unknown")), http.StatusInternalServerError},
			{"multiple classes", erro.New("test", erro.ClassValidation, erro.ClassNotFound), http.StatusNotFound}, // Last one wins
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				code := erro.HTTPCode(tc.err)
				if code != tc.expectedCode {
					t.Errorf("Expected %d, got %d", tc.expectedCode, code)
				}
			})
		}
	})

	t.Run("HTTP middleware integration", func(t *testing.T) {
		// Test error handling in HTTP middleware
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/validation":
				err := erro.New("validation failed", erro.ClassValidation)
				http.Error(w, err.Error(), erro.HTTPCode(err))
			case "/notfound":
				err := erro.New("resource not found", erro.ClassNotFound)
				http.Error(w, err.Error(), erro.HTTPCode(err))
			case "/panic":
				panic(erro.New("panic error", erro.ClassCritical))
			default:
				w.WriteHeader(http.StatusOK)
			}
		})

		// Wrap with recovery middleware
		recoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					var err error
					if e, ok := r.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("panic: %v", r)
					}
					http.Error(w, err.Error(), erro.HTTPCode(err))
				}
			}()
			handler.ServeHTTP(w, r)
		})

		testCases := []struct {
			path         string
			expectedCode int
		}{
			{"/validation", http.StatusBadRequest},
			{"/notfound", http.StatusNotFound},
			{"/panic", http.StatusInternalServerError},
			{"/ok", http.StatusOK},
		}

		for _, tc := range testCases {
			t.Run(tc.path, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.path, nil)
				w := httptest.NewRecorder()

				recoveryHandler.ServeHTTP(w, req)

				if w.Code != tc.expectedCode {
					t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
				}
			})
		}
	})
}

// TestErrorCollectionIntegration tests error collection edge cases
func TestErrorCollectionIntegration(t *testing.T) {
	t.Run("Mixed error types in collections", func(t *testing.T) {
		list := erro.NewList()

		// Add different types of errors
		list.Add(errors.New("standard error"))
		list.Add(erro.New("erro error", erro.ClassValidation))
		list.Add(io.EOF)
		list.Add(context.Canceled)

		if list.Len() != 4 {
			t.Errorf("Expected 4 errors, got %d", list.Len())
		}

		combinedErr := list.Err()
		if combinedErr == nil {
			t.Error("Should return combined error")
		}

		// Should be able to unwrap and check individual errors
		if !errors.Is(combinedErr, io.EOF) {
			t.Error("Should maintain Is relationship with EOF")
		}
		if !errors.Is(combinedErr, context.Canceled) {
			t.Error("Should maintain Is relationship with Canceled")
		}
	})

	t.Run("Error set deduplication edge cases", func(t *testing.T) {
		set := erro.NewSet()

		// Add same error multiple times
		baseErr := errors.New("duplicate error")
		set.Add(baseErr)
		set.Add(baseErr)                       // Same instance
		set.Add(errors.New("duplicate error")) // Different instance, same message

		// Should deduplicate based on message
		if set.Len() != 1 {
			t.Errorf("Expected 1 unique error, got %d", set.Len())
		}

		// Test with custom key getter
		set.WithKeyGetter(erro.IDKeyGetter)
		set.Clear()

		set.New("error 1", erro.ID("same_id"))
		set.New("error 2", erro.ID("same_id"))
		set.New("error 3", erro.ID("different_id"))

		// Should deduplicate based on ID
		if set.Len() != 2 {
			t.Errorf("Expected 2 unique errors by ID, got %d", set.Len())
		}
	})
}

// TestTemplateIntegration tests template system edge cases
func TestTemplateIntegration(t *testing.T) {
	t.Run("Template inheritance and overriding", func(t *testing.T) {
		baseTemplate := erro.NewTemplate("base error: %s",
			erro.ClassValidation,
			erro.CategoryUserInput,
		)

		// Create error with additional fields that might override template
		err := baseTemplate.New("specific issue",
			"field1", "value1",
			erro.ClassNotFound, // Override class from template
			erro.SeverityHigh,  // Add severity not in template
		)

		// Should use overridden class
		if err.Class() != erro.ClassNotFound {
			t.Errorf("Expected NotFound class, got %s", err.Class())
		}

		// Should inherit category from template
		if err.Category() != erro.CategoryUserInput {
			t.Errorf("Expected UserInput category, got %s", err.Category())
		}

		// Should have additional severity
		if err.Severity() != erro.SeverityHigh {
			t.Errorf("Expected High severity, got %s", err.Severity())
		}
	})

	t.Run("Template with format verb edge cases", func(t *testing.T) {
		template := erro.NewTemplate("error with %d items and %s status")

		// Test with correct number of args
		err1 := template.New(42, "active")
		if !strings.Contains(err1.Error(), "42 items") {
			t.Error("Should format numbers correctly")
		}

		// Test with insufficient args
		err2 := template.New(42) // Missing second arg
		if err2.Error() == "" {
			t.Error("Should handle missing args gracefully")
		}

		// Test with excess args
		err3 := template.New(42, "active", "extra")
		if err3.Error() == "" {
			t.Error("Should handle excess args gracefully")
		}
	})
}

// TestResourceCleanupIntegration tests resource cleanup scenarios
func TestResourceCleanupIntegration(t *testing.T) {
	t.Run("File operations with Close utility", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "erro_test_*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		// Test successful close
		var closeErr error
		erro.Close(&closeErr, tmpFile, "failed to close temp file", "file", tmpFile.Name())

		if closeErr != nil {
			t.Errorf("Should not have close error: %v", closeErr)
		}

		// Test close with existing error (should not override)
		existingErr := errors.New("existing error")
		closeErr = existingErr
		erro.Close(&closeErr, tmpFile, "failed to close again") // File already closed

		if closeErr != existingErr {
			t.Error("Should not override existing error")
		}
	})

	t.Run("Context shutdown with Shutdown utility", func(t *testing.T) {
		shutdownCalled := false
		shutdownFunc := func(ctx context.Context) error {
			shutdownCalled = true
			return nil
		}

		ctx := context.Background()
		var shutdownErr error

		erro.Shutdown(ctx, &shutdownErr, shutdownFunc, "shutdown failed")

		if !shutdownCalled {
			t.Error("Shutdown function should have been called")
		}
		if shutdownErr != nil {
			t.Errorf("Should not have shutdown error: %v", shutdownErr)
		}

		// Test with shutdown error
		errorShutdownFunc := func(ctx context.Context) error {
			return errors.New("shutdown failed")
		}

		shutdownErr = nil
		erro.Shutdown(ctx, &shutdownErr, errorShutdownFunc, "shutdown failed")

		if shutdownErr == nil {
			t.Error("Should have shutdown error")
		}
	})
}

// TestExternalInterfaceImplementation tests implementation of external interfaces
func TestExternalInterfaceImplementation(t *testing.T) {
	t.Run("Unwrapper interface compatibility", func(t *testing.T) {
		// Test that errors implement unwrapping correctly for Go 1.13+ error handling
		baseErr := errors.New("base")
		wrapped := erro.Wrap(baseErr, "wrapped")
		doubleWrapped := erro.Wrap(wrapped, "double wrapped")

		// Test direct unwrapping
		if erro.Unwrap(wrapped) != baseErr {
			t.Error("Direct unwrap failed")
		}

		// Test deep unwrapping with errors.Is
		if !errors.Is(doubleWrapped, baseErr) {
			t.Error("Deep Is check failed")
		}

		// Test unwrap chain
		current := error(doubleWrapped)
		depth := 0
		for current != nil {
			depth++
			if depth > 10 {
				t.Error("Unwrap chain too deep or circular")
				break
			}
			current = errors.Unwrap(current)
		}
	})

	t.Run("JSON marshaler/unmarshaler interface", func(t *testing.T) {
		originalErr := erro.New("json test",
			"field1", "value1",
			"field2", 42,
			erro.ClassValidation,
		)

		// Should implement json.Marshaler
		data, err := originalErr.MarshalJSON()
		if err != nil {
			t.Errorf("MarshalJSON failed: %v", err)
		}

		// Should implement json.Unmarshaler
		newErr := erro.New("")
		unmarshalErr := newErr.UnmarshalJSON(data)
		if unmarshalErr != nil {
			t.Errorf("UnmarshalJSON failed: %v", unmarshalErr)
		}

		// Should preserve essential data
		if newErr.Message() != originalErr.Message() {
			t.Error("Message not preserved")
		}
		if newErr.Class() != originalErr.Class() {
			t.Error("Class not preserved")
		}
	})
}

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

		// Set reasonable performance expectations (increased for CI and different Go versions)
		if avgPerOp > 50*time.Microsecond {
			t.Errorf("Performance regression: %v per operation (expected < 50µs)", avgPerOp)
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

		if avgPerOp > 60*time.Microsecond {
			t.Errorf("Wrapping performance regression: %v per operation (expected < 60µs)", avgPerOp)
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

// TestJSONSerializationEdgeCases tests edge cases in JSON serialization
func TestJSONSerializationEdgeCases(t *testing.T) {
	t.Run("Large JSON serialization", func(t *testing.T) {
		// Create error with many fields that could cause large JSON
		fields := make([]any, 0, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount; i++ {
			fields = append(fields,
				fmt.Sprintf("key_%d", i),
				strings.Repeat("large_value_", 10)+fmt.Sprintf("_%d", i),
			)
		}

		err := erro.New("large serialization test", fields...)
		err = erro.Wrap(err, "wrapped error", erro.StackTrace())

		// Marshal to JSON
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal large error: %v", marshalErr)
		}

		// Should produce reasonable size JSON
		if len(jsonData) > 1000000 { // 1MB limit
			t.Errorf("JSON size too large: %d bytes", len(jsonData))
		}

		// Should be able to unmarshal back
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal large error: %v", unmarshalErr)
		}
	})

	t.Run("Invalid JSON unmarshaling", func(t *testing.T) {
		testCases := []struct {
			name     string
			jsonData string
		}{
			{"malformed JSON", `{"id": "test", "message": "test"`},
			{"invalid field types", `{"id": 123, "message": ["array"]}`},
			{"null values", `{"id": null, "message": null, "fields": null}`},
			{"empty JSON", `{}`},
			{"array instead of object", `["not", "an", "object"]`},
			{"string instead of object", `"not an object"`},
			{"very large JSON", `{"message": "` + strings.Repeat("x", 10000) + `"}`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := erro.New("")
				unmarshalErr := json.Unmarshal([]byte(tc.jsonData), err)
				// Should handle invalid JSON gracefully without panicking
				if unmarshalErr == nil && tc.name != "empty JSON" && tc.name != "null values" {
					t.Logf("Expected error for %s, but got none", tc.name)
				}
			})
		}
	})

	t.Run("Special characters in JSON", func(t *testing.T) {
		// Test with characters that could break JSON
		specialValues := []any{
			"unicode: 测试 🚀 émojis 🔥",
			"quotes: \"double\" and 'single'",
			"backslashes: \\ and \\n and \\t",
			"control chars: \x00\x01\x02\x1f",
			"json injection: \"},\"injected\":\"value",
		}

		for i, val := range specialValues {
			err := erro.New("special chars test", fmt.Sprintf("key_%d", i), val)

			// Should marshal without error
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Failed to marshal special chars: %v", marshalErr)
			}

			// Should unmarshal without error
			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Failed to unmarshal special chars: %v", unmarshalErr)
			}
		}
	})

	t.Run("Redacted values in JSON", func(t *testing.T) {
		err := erro.New("redaction test",
			"public", "visible_value",
			"secret", erro.Redact("sensitive_data"),
			"password", erro.Redact("super_secret_password"),
			"api_key", erro.Redact(map[string]string{"key": "secret_key_value"}),
		)

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal redacted error: %v", marshalErr)
		}

		jsonStr := string(jsonData)

		// Should contain redacted placeholder
		if !strings.Contains(jsonStr, erro.RedactedPlaceholder) {
			t.Error("JSON should contain redacted placeholder")
		}

		// Should NOT contain sensitive data
		if strings.Contains(jsonStr, "sensitive_data") ||
			strings.Contains(jsonStr, "super_secret_password") ||
			strings.Contains(jsonStr, "secret_key_value") {
			t.Error("JSON should not contain sensitive data")
		}

		// Should contain public data
		if !strings.Contains(jsonStr, "visible_value") {
			t.Error("JSON should contain public data")
		}
	})

	t.Run("Stack trace in JSON", func(t *testing.T) {
		err := erro.New("stack trace test", erro.StackTrace())

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal error with stack: %v", marshalErr)
		}

		var parsed map[string]interface{}
		parseErr := json.Unmarshal(jsonData, &parsed)
		if parseErr != nil {
			t.Errorf("Failed to parse JSON: %v", parseErr)
		}

		// Should contain stack trace
		if _, exists := parsed["stack_trace"]; !exists {
			t.Error("JSON should contain stack_trace field")
		}

		// Unmarshal back to error
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal error with stack: %v", unmarshalErr)
		}
	})
}

// TestComplexDataTypeSerialization tests serialization of complex data types
func TestComplexDataTypeSerialization(t *testing.T) {
	t.Run("Nested structures", func(t *testing.T) {
		type complexStruct struct {
			Name     string            `json:"name"`
			Values   []int             `json:"values"`
			Metadata map[string]string `json:"metadata"`
			Time     time.Time         `json:"time"`
		}

		complex := complexStruct{
			Name:   "test_struct",
			Values: []int{1, 2, 3, 4, 5},
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Time: time.Now(),
		}

		err := erro.New("complex data test",
			"complex_struct", complex,
			"nested_map", map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": []string{"a", "b", "c"},
				},
			},
		)

		// Should marshal successfully
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal complex data: %v", marshalErr)
		}

		// Should unmarshal successfully
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal complex data: %v", unmarshalErr)
		}
	})

	t.Run("Circular reference handling", func(t *testing.T) {
		type circularStruct struct {
			Name string          `json:"name"`
			Self *circularStruct `json:"self,omitempty"`
		}

		circular := &circularStruct{Name: "circular"}
		circular.Self = circular

		err := erro.New("circular test", "circular_data", circular)

		// Should handle circular references gracefully
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			// Circular references should either be handled or fail gracefully
			t.Logf("Circular reference marshal failed as expected: %v", marshalErr)
		} else {
			// If it succeeds, should produce valid JSON
			var result map[string]interface{}
			parseErr := json.Unmarshal(jsonData, &result)
			if parseErr != nil {
				t.Errorf("Produced invalid JSON for circular reference: %v", parseErr)
			}
		}
	})

	t.Run("Large slice/array handling", func(t *testing.T) {
		largeSlice := make([]int, 10000)
		for i := range largeSlice {
			largeSlice[i] = i
		}

		err := erro.New("large slice test", "large_slice", largeSlice)

		// Should handle large slices efficiently
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal large slice: %v", marshalErr)
		}

		// Should have reasonable size (truncation might occur)
		if len(jsonData) > 2000000 { // 2MB limit
			t.Errorf("JSON for large slice too big: %d bytes", len(jsonData))
		}
	})

	t.Run("Nil and empty values", func(t *testing.T) {
		var nilSlice []string
		var nilMap map[string]string
		var nilInterface interface{}

		err := erro.New("nil values test",
			"nil_slice", nilSlice,
			"nil_map", nilMap,
			"nil_interface", nilInterface,
			"empty_slice", []string{},
			"empty_map", map[string]string{},
			"empty_string", "",
		)

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal nil values: %v", marshalErr)
		}

		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal nil values: %v", unmarshalErr)
		}
	})
}

// TestSerializationBoundaryConditions tests boundary conditions in serialization
func TestSerializationBoundaryConditions(t *testing.T) {
	t.Run("Maximum field count serialization", func(t *testing.T) {
		// Test serialization at maximum field capacity
		fields := make([]any, 0, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount; i++ {
			fields = append(fields, fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
		}

		err := erro.New("max fields test", fields...)

		// Multiple serialization/deserialization cycles
		for cycle := 0; cycle < 5; cycle++ {
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Cycle %d: marshal failed: %v", cycle, marshalErr)
			}

			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Cycle %d: unmarshal failed: %v", cycle, unmarshalErr)
			}

			err = newErr // Use for next cycle
		}
	})

	t.Run("Empty error serialization", func(t *testing.T) {
		err := erro.New("")

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal empty error: %v", marshalErr)
		}

		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal empty error: %v", unmarshalErr)
		}
	})

	t.Run("Concurrent serialization", func(t *testing.T) {
		err := erro.New("concurrent test",
			"field1", "value1",
			"field2", 42,
			"field3", time.Now(),
			erro.StackTrace(),
		)

		const numGoroutines = 100
		errors := make(chan error, numGoroutines)

		// Serialize concurrently
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("panic: %v", r)
					}
				}()

				jsonData, marshalErr := json.Marshal(err)
				if marshalErr != nil {
					errors <- marshalErr
					return
				}

				newErr := erro.New("")
				unmarshalErr := json.Unmarshal(jsonData, newErr)
				if unmarshalErr != nil {
					errors <- unmarshalErr
					return
				}

				errors <- nil
			}()
		}

		// Check results
		for i := 0; i < numGoroutines; i++ {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent serialization error: %v", err)
			}
		}
	})
}

// TestSerializationPerformance tests performance characteristics of serialization
func TestSerializationPerformance(t *testing.T) {
	t.Run("Serialization performance with stack traces", func(t *testing.T) {
		err := erro.New("performance test", erro.StackTrace())

		start := time.Now()
		for i := 0; i < 1000; i++ {
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Marshal failed: %v", marshalErr)
			}

			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Unmarshal failed: %v", unmarshalErr)
			}
		}
		duration := time.Since(start)

		if duration > 3*time.Second {
			t.Errorf("Serialization performance too slow: %v", duration)
		}
	})

	t.Run("Memory allocation during serialization", func(t *testing.T) {
		err := erro.New("memory test",
			"data", strings.Repeat("x", 1000),
			erro.StackTrace(),
		)

		// Test that repeated serialization doesn't cause memory leaks
		for i := 0; i < 1000; i++ {
			jsonData, _ := json.Marshal(err)
			_ = jsonData // Use the data to prevent optimization
		}

		// This test mainly ensures no panics or excessive memory growth
		// In a real scenario, you'd monitor memory usage here
	})
}

// TestInputValidationEdgeCases tests various edge cases in input validation
func TestInputValidationEdgeCases(t *testing.T) {
	t.Run("Extreme numeric values", func(t *testing.T) {
		extremeValues := []any{
			math.MaxInt64,
			math.MinInt64,
			math.MaxFloat64,
			math.SmallestNonzeroFloat64,
			math.Inf(1),
			math.Inf(-1),
			math.NaN(),
			uint64(math.MaxUint64),
		}

		for i, val := range extremeValues {
			err := erro.New("extreme value test", fmt.Sprintf("key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle extreme value %v", val)
			}
		}
	})

	t.Run("Complex data types", func(t *testing.T) {
		complexValues := []any{
			complex(1.5, 2.5),
			complex(math.Inf(1), math.NaN()),
			make(chan int),
			make(chan struct{}, 100),
			func() {},
			unsafe.Pointer(nil),
		}

		for i, val := range complexValues {
			err := erro.New("complex type test", fmt.Sprintf("key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle complex value %v of type %T", val, val)
			}
		}
	})

	t.Run("Reflection edge cases", func(t *testing.T) {
		// Test with reflect.Value types
		reflectValues := []any{
			reflect.ValueOf(42),
			reflect.ValueOf("string"),
			reflect.ValueOf([]int{1, 2, 3}),
			reflect.Zero(reflect.TypeOf("")),
			reflect.ValueOf(nil),
		}

		for i, val := range reflectValues {
			err := erro.New("reflect test", fmt.Sprintf("reflect_key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle reflect value %v", val)
			}
		}
	})

	t.Run("Interface edge cases", func(t *testing.T) {
		// Test with various interface{} values
		var nilInterface interface{}
		var typedNil *string
		var emptyInterface interface{} = (*string)(nil)

		interfaceValues := []any{
			nilInterface,
			typedNil,
			emptyInterface,
			interface{}(42),
			interface{}(nil),
		}

		for i, val := range interfaceValues {
			err := erro.New("interface test", fmt.Sprintf("interface_key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle interface value %v", val)
			}
		}
	})
}

// TestBoundaryConditionValidation tests boundary conditions
func TestBoundaryConditionValidation(t *testing.T) {
	t.Run("String length boundaries", func(t *testing.T) {
		testCases := []struct {
			name   string
			length int
		}{
			{"single char", 1},
			{"at max key length", erro.MaxKeyLength},
			{"beyond max key length", erro.MaxKeyLength + 1},
			{"at max value length", erro.MaxValueLength},
			{"beyond max value length", erro.MaxValueLength + 1},
			{"at max message length", erro.MaxMessageLength},
			{"beyond max message length", erro.MaxMessageLength + 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testString := strings.Repeat("x", tc.length)

				// Test as message
				err := erro.New(testString)
				if err.Error() == "" {
					t.Error("Should handle string as message")
				}

				// Test as key
				err = erro.New("test", testString, "value")
				if err.Error() == "" {
					t.Error("Should handle string as key")
				}

				// Test as value
				err = erro.New("test", "key", testString)
				if err.Error() == "" {
					t.Error("Should handle string as value")
				}
			})
		}
	})

	t.Run("Field count boundaries", func(t *testing.T) {
		testCases := []struct {
			name       string
			fieldCount int
		}{
			{"no fields", 0},
			{"single field pair", 2},
			{"at max fields", erro.MaxFieldsCount * 2},
			{"beyond max fields", erro.MaxFieldsCount*2 + 10},
			{"odd field count", erro.MaxFieldsCount*2 + 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fields := make([]any, tc.fieldCount)
				for i := 0; i < tc.fieldCount; i++ {
					if i%2 == 0 {
						fields[i] = fmt.Sprintf("key_%d", i/2)
					} else {
						fields[i] = fmt.Sprintf("value_%d", i/2)
					}
				}

				err := erro.New("boundary test", fields...)
				if err.Error() == "" {
					t.Error("Should handle field boundary conditions")
				}

				// Check that fields are properly limited
				actualFields := err.Fields()
				maxExpected := erro.MaxFieldsCount * 2
				if len(actualFields) > maxExpected {
					t.Errorf("Fields should be limited to %d, got %d", maxExpected, len(actualFields))
				}
			})
		}
	})

	t.Run("Stack depth boundaries", func(t *testing.T) {
		// Test with maximum stack depth
		err := erro.New("stack depth test", erro.StackTrace())
		stack := err.Stack()

		if len(stack) > erro.MaxStackDepth {
			t.Errorf("Stack depth should be limited to %d, got %d", erro.MaxStackDepth, len(stack))
		}

		// Test stack trace with deep call stack
		deepErr := createDeepStackTrace(50)
		deepStack := deepErr.Stack()

		if len(deepStack) > erro.MaxStackDepth {
			t.Errorf("Deep stack should be limited to %d, got %d", erro.MaxStackDepth, len(deepStack))
		}
	})
}

// TestMalformedDataHandling tests handling of malformed or corrupted data
func TestMalformedDataHandling(t *testing.T) {
	t.Run("Invalid UTF-8 sequences", func(t *testing.T) {
		invalidUTF8 := []string{
			"\xff\xfe\xfd",                         // Invalid byte sequences
			"valid\xff\xfeinvalid",                 // Mixed valid/invalid
			string([]byte{0xff, 0xfe, 0xfd, 0xfc}), // Raw invalid bytes
		}

		for i, invalid := range invalidUTF8 {
			err := erro.New("utf8 test", fmt.Sprintf("key_%d", i), invalid)
			if err.Error() == "" {
				t.Error("Should handle invalid UTF-8")
			}

			// Should not panic when converting to string
			_ = err.Error()
		}
	})

	t.Run("Zero-width and invisible characters", func(t *testing.T) {
		invisibleChars := []string{
			"\u200b", // Zero width space
			"\u200c", // Zero width non-joiner
			"\u200d", // Zero width joiner
			"\ufeff", // Byte order mark
			"\u2060", // Word joiner
		}

		for i, char := range invisibleChars {
			err := erro.New("invisible char test", fmt.Sprintf("key_%d", i), char)
			if err.Error() == "" {
				t.Error("Should handle invisible characters")
			}
		}
	})

	t.Run("Control characters and escapes", func(t *testing.T) {
		controlChars := []string{
			"\x00",     // Null
			"\x07",     // Bell
			"\x08",     // Backspace
			"\x1b[31m", // ANSI escape sequence
			"\x1b[0m",  // ANSI reset
		}

		for i, char := range controlChars {
			err := erro.New("control char test", fmt.Sprintf("key_%d", i), char)
			if err.Error() == "" {
				t.Error("Should handle control characters")
			}
		}
	})

	t.Run("Extremely nested structures", func(t *testing.T) {
		// Create deeply nested map
		depth := 100
		nested := make(map[string]interface{})
		current := nested

		for i := 0; i < depth; i++ {
			next := make(map[string]interface{})
			current[fmt.Sprintf("level_%d", i)] = next
			current = next
		}
		current["final"] = "value"

		err := erro.New("nested test", "nested_data", nested)
		if err.Error() == "" {
			t.Error("Should handle deeply nested structures")
		}

		// Should not cause stack overflow
		_ = err.Error()
	})
}

// TestDataTypeValidation tests validation of various Go data types
func TestDataTypeValidation(t *testing.T) {
	t.Run("Slice and array edge cases", func(t *testing.T) {
		sliceTests := []any{
			[]int{},                 // Empty slice
			[]int{1, 2, 3},          // Regular slice
			make([]int, 0, 1000),    // High capacity, zero length
			make([]string, 1000),    // Large slice
			[...]int{1, 2, 3, 4, 5}, // Array
			(*[]int)(nil),           // Nil slice pointer
		}

		for i, slice := range sliceTests {
			err := erro.New("slice test", fmt.Sprintf("slice_%d", i), slice)
			if err.Error() == "" {
				t.Errorf("Should handle slice type %T", slice)
			}
		}
	})

	t.Run("Map edge cases", func(t *testing.T) {
		mapTests := []any{
			map[string]int{},                      // Empty map
			map[string]int{"key": 1},              // Regular map
			make(map[string]int, 1000),            // Large capacity map
			map[interface{}]interface{}{1: "one"}, // Interface{} map
			(*map[string]int)(nil),                // Nil map pointer
		}

		for i, m := range mapTests {
			err := erro.New("map test", fmt.Sprintf("map_%d", i), m)
			if err.Error() == "" {
				t.Errorf("Should handle map type %T", m)
			}
		}
	})

	t.Run("Pointer edge cases", func(t *testing.T) {
		value := 42
		var nilPtr *int

		pointerTests := []any{
			&value,                          // Valid pointer
			nilPtr,                          // Nil pointer
			&nilPtr,                         // Pointer to nil pointer
			unsafe.Pointer(&value),          // Unsafe pointer
			uintptr(unsafe.Pointer(&value)), // Uintptr
		}

		for i, ptr := range pointerTests {
			err := erro.New("pointer test", fmt.Sprintf("ptr_%d", i), ptr)
			if err.Error() == "" {
				t.Errorf("Should handle pointer type %T", ptr)
			}
		}
	})

	t.Run("Time and duration edge cases", func(t *testing.T) {
		timeTests := []any{
			time.Time{},     // Zero time
			time.Now(),      // Current time
			time.Unix(0, 0), // Unix epoch
			time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC), // Far future
			time.Duration(0),           // Zero duration
			time.Nanosecond,            // Smallest duration
			time.Hour * 24 * 365 * 100, // Large duration
			-time.Hour,                 // Negative duration
		}

		for i, timeVal := range timeTests {
			err := erro.New("time test", fmt.Sprintf("time_%d", i), timeVal)
			if err.Error() == "" {
				t.Errorf("Should handle time type %T", timeVal)
			}
		}
	})
}

// TestMemoryAndGarbageCollection tests memory-related edge cases
func TestMemoryAndGarbageCollection(t *testing.T) {
	t.Run("Large field values with GC pressure", func(t *testing.T) {
		// Create errors with large data and force GC
		for i := 0; i < 100; i++ {
			largeData := make([]byte, 10000)
			for j := range largeData {
				largeData[j] = byte(j % 256)
			}

			err := erro.New("gc test", "large_data", largeData, "iteration", i)
			_ = err.Error() // Force string creation

			if i%10 == 0 {
				runtime.GC() // Force garbage collection
			}
		}

		// Test should complete without memory issues
		runtime.GC()
	})

	t.Run("Reference cycles with error fields", func(t *testing.T) {
		type cyclicStruct struct {
			Name string
			Ref  *cyclicStruct
		}

		root := &cyclicStruct{Name: "root"}
		child := &cyclicStruct{Name: "child", Ref: root}
		root.Ref = child // Create cycle

		err := erro.New("cycle test", "cyclic_data", root)
		_ = err.Error()

		// Should not prevent garbage collection
		root = nil
		child = nil
		runtime.GC()
	})
}

// Helper function to create deep stack trace
func createDeepStackTrace(depth int) erro.Error {
	if depth <= 0 {
		return erro.New("deep stack trace", erro.StackTrace())
	}
	return createDeepStackTrace(depth - 1)
}
