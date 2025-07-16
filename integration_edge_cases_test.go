package erro_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

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
		if !strings.Contains(fullFormat, "integration_edge_cases_test.go") {
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

// TestLoggingIntegration tests integration with logging frameworks
func TestLoggingIntegration(t *testing.T) {
	t.Run("slog integration", func(t *testing.T) {
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		err := erro.New("test error",
			"user_id", 123,
			"operation", "test",
			"sensitive", erro.Redact("secret"),
			erro.ClassValidation,
			erro.CategoryUserInput,
		)

		// Test LogFields integration
		logger.Error("Operation failed", erro.LogFields(err)...)

		output := logOutput.String()

		// Should contain error fields
		if !strings.Contains(output, "user_id=123") {
			t.Error("Should contain user_id field")
		}
		if !strings.Contains(output, "operation=test") {
			t.Error("Should contain operation field")
		}

		// Should redact sensitive data
		if strings.Contains(output, "secret") {
			t.Error("Should not contain sensitive data")
		}
		if !strings.Contains(output, erro.RedactedPlaceholder) {
			t.Error("Should contain redacted placeholder")
		}
	})

	t.Run("LogFieldsMap integration", func(t *testing.T) {
		err := erro.New("map test",
			"string_field", "value",
			"int_field", 42,
			"bool_field", true,
			"nil_field", nil,
		)

		fieldsMap := erro.LogFieldsMap(err)

		// Should contain all fields as map
		if fieldsMap["string_field"] != "value" {
			t.Error("Missing string field")
		}
		if fieldsMap["int_field"] != 42 {
			t.Error("Missing int field")
		}
		if fieldsMap["bool_field"] != true {
			t.Error("Missing bool field")
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
