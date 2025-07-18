package erro_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

// TestIsMethod tests the Is method with all possible cases
func TestIsMethod(t *testing.T) {
	t.Run("Nil error cases", func(t *testing.T) {
		// Test with nil target
		err := erro.New("test error")
		if err.Is(nil) {
			t.Error("Is should return false for nil target")
		}
	})

	t.Run("ID-based comparison", func(t *testing.T) {
		// Test with matching IDs
		err1 := erro.New("error 1", erro.ID("test_id_123"))
		err2 := erro.New("error 2", erro.ID("test_id_123"))
		if !err1.Is(err2) {
			t.Error("Is should return true for errors with matching IDs")
		}

		// Test with different IDs
		err3 := erro.New("error 3", erro.ID("different_id"))
		if err1.Is(err3) {
			t.Error("Is should return false for errors with different IDs")
		}

		// Test with one error having ID, other not having ID
		err4 := erro.New("error 4") // no ID
		if err1.Is(err4) {
			t.Error("Is should return false when only one error has ID")
		}
		if err4.Is(err1) {
			t.Error("Is should return false when only one error has ID")
		}

		// Test with both errors having empty IDs
		err5 := erro.New("error 5", erro.ID(""))
		err6 := erro.New("error 6", erro.ID(""))
		if err5.Is(err6) {
			t.Error("Is should return false for errors with empty IDs")
		}
	})

	t.Run("Class-based comparison", func(t *testing.T) {
		// Test with matching class, category, severity, and retryable
		template := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}
		err := erro.New("test error",
			erro.ClassValidation,
			erro.CategoryDatabase,
			erro.SeverityHigh,
			erro.Retryable(),
		)
		if !err.Is(template) {
			t.Error("Is should return true for matching class, category, severity, and retryable")
		}

		// Test with different class
		template2 := &templateError{
			class:     erro.ClassNotFound,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}
		if err.Is(template2) {
			t.Error("Is should return false for different class")
		}

		// Test with different category
		template3 := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryNetwork,
			severity:  erro.SeverityHigh,
			retryable: true,
		}
		if err.Is(template3) {
			t.Error("Is should return false for different category")
		}

		// Test with different severity
		template4 := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityLow,
			retryable: true,
		}
		if err.Is(template4) {
			t.Error("Is should return false for different severity")
		}

		// Test with different retryable
		template5 := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: false,
		}
		if err.Is(template5) {
			t.Error("Is should return false for different retryable")
		}
	})

	t.Run("Class comparison with ID present", func(t *testing.T) {
		// When target has ID, class comparison should be ignored
		// Create a template with ID using the testTemplateError type
		template := &testTemplateError{
			id:        "test_id",
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}

		err := erro.New("test error",
			erro.ID("different_id"),
			erro.ClassValidation,
			erro.CategoryDatabase,
			erro.SeverityHigh,
			erro.Retryable(),
		)
		if err.Is(template) {
			t.Error("Is should return false when IDs don't match, even if class matches")
		}
	})

	t.Run("Unknown class and category", func(t *testing.T) {
		// Test with unknown class and category
		template := &templateError{
			class:     erro.ClassUnknown,
			category:  erro.CategoryUnknown,
			severity:  erro.SeverityHigh,
			retryable: true,
		}
		err := erro.New("test error", erro.SeverityHigh, erro.Retryable())
		if err.Is(template) {
			t.Error("Is should return false when target has unknown class and category")
		}
	})

	t.Run("Non-Error target", func(t *testing.T) {
		// Test with standard error as target
		stdErr := errors.New("standard error")
		err := erro.New("test error")
		if err.Is(stdErr) {
			t.Error("Is should return false for non-Error target")
		}
	})

	t.Run("Wrapped errors", func(t *testing.T) {
		// Test with wrapped errors
		baseErr := erro.New("base error", erro.ID("base_id"))
		wrappedErr := erro.Wrap(baseErr, "wrapped error")

		// Should match the base error
		if !wrappedErr.Is(baseErr) {
			t.Error("Is should return true for wrapped error matching base error")
		}

		// Should not match different error
		otherErr := erro.New("other error", erro.ID("other_id"))
		if wrappedErr.Is(otherErr) {
			t.Error("Is should return false for wrapped error not matching other error")
		}
	})

	t.Run("Deep wrapped errors", func(t *testing.T) {
		// Test with deeply wrapped errors
		baseErr := erro.New("base error", erro.ID("deep_base_id"))
		wrapped1 := erro.Wrap(baseErr, "first wrap")
		wrapped2 := erro.Wrap(wrapped1, "second wrap")
		wrapped3 := erro.Wrap(wrapped2, "third wrap")

		// Should match the base error
		if !wrapped3.Is(baseErr) {
			t.Error("Is should return true for deeply wrapped error matching base error")
		}

		// Should match intermediate wrapped error
		if !wrapped3.Is(wrapped1) {
			t.Error("Is should return true for deeply wrapped error matching intermediate error")
		}
	})

	t.Run("Mixed error types", func(t *testing.T) {
		// Test with mixed standard and erro errors
		stdErr := errors.New("standard error")
		wrappedStdErr := erro.Wrap(stdErr, "wrapped standard")

		if !errors.Is(wrappedStdErr, stdErr) {
			t.Error("Is should return true for wrapped standard error")
		}
		if !wrappedStdErr.Is(stdErr) {
			t.Error("Is should return true for wrapped standard error")
		}

		// Test with custom error type
		customErr := &customError{msg: "custom error"}
		wrappedCustomErr := erro.Wrap(customErr, "wrapped custom")

		if !wrappedCustomErr.Is(customErr) {
			t.Error("Is should return true for wrapped custom error")
		}
	})
}

// TestAsMethod tests the As method with all possible cases
func TestAsMethod(t *testing.T) {
	t.Run("Nil error cases", func(t *testing.T) {
		// Test with nil target
		err := erro.New("test error")
		if err.As(nil) {
			t.Error("As should return false for nil target")
		}
	})

	t.Run("BaseError pointer target", func(t *testing.T) {
		// Test with *Error target
		err := erro.New("test error")
		var target erro.Error
		if !err.As(&target) {
			t.Error("As should return true for *Error target")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		if target != err {
			t.Error("Target should contain the original error")
		}

		// Test with nil pointer
		var nilTarget erro.Error
		if !err.As(&nilTarget) {
			t.Error("As should return true for nil *Error target")
		}
		if nilTarget == nil {
			t.Error("Target should not be nil after successful As")
		}
	})

	t.Run("Wrong pointer type", func(t *testing.T) {
		// Test with wrong pointer type
		err := erro.New("test error")
		var target *customError
		if err.As(&target) {
			t.Error("As should return false for wrong pointer type")
		}
		if target != nil {
			t.Error("Target should remain nil for failed As")
		}
	})

	t.Run("Wrapped standard error", func(t *testing.T) {
		// Test with wrapped standard error
		stdErr := &customError{msg: "custom error"}
		wrappedErr := erro.Wrap(stdErr, "wrapped")

		var target *customError
		if !wrappedErr.As(&target) {
			t.Error("As should return true for wrapped standard error")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		if target.msg != "custom error" {
			t.Errorf("Target should contain original error, got: %s", target.msg)
		}
	})

	t.Run("Wrapped erro error", func(t *testing.T) {
		// Test with wrapped erro error
		baseErr := erro.New("base error")
		wrappedErr := erro.Wrap(baseErr, "wrapped")

		var target erro.Error
		if !wrappedErr.As(&target) {
			t.Error("As should return true for wrapped erro error")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		if target.ID() != baseErr.ID() {
			t.Error("Target should contain the base error")
		}
	})

	t.Run("Deep wrapped errors", func(t *testing.T) {
		// Test with deeply wrapped errors
		baseErr := erro.New("base error")
		wrapped1 := erro.Wrap(baseErr, "first wrap")
		wrapped2 := erro.Wrap(wrapped1, "second wrap")
		wrapped3 := erro.Wrap(wrapped2, "third wrap")

		var target erro.Error
		if !wrapped3.As(&target) {
			t.Error("As should return true for deeply wrapped error")
		}
		if target.ID() != baseErr.ID() {
			t.Error("Target should contain the base error")
		}
	})

	t.Run("Mixed error types", func(t *testing.T) {
		// Test with mixed error types
		customErr := &customError{msg: "custom"}
		wrappedCustom := erro.Wrap(customErr, "wrapped custom")
		mixedErr := erro.Wrap(wrappedCustom, "mixed")
		mixedErr = erro.Wrap(mixedErr, "more mixed")

		// Should extract custom error
		var customTarget *customError
		if !mixedErr.As(&customTarget) {
			t.Error("As should return true for mixed error types")
		}
		if customTarget.msg != "custom" {
			t.Errorf("Target should contain original custom error, got: %s", customTarget.msg)
		}

		// Should not extract erro error when wrapped around custom
		var erroTarget erro.Error
		if !mixedErr.As(&erroTarget) {
			t.Error("As should return true for erro error when custom error is present")
		}
	})

	t.Run("Multiple As calls", func(t *testing.T) {
		// Test multiple As calls on the same error
		err := erro.New("test error")

		var target1 erro.Error
		if !err.As(&target1) {
			t.Error("First As call should succeed")
		}

		var target2 erro.Error
		if !err.As(&target2) {
			t.Error("Second As call should succeed")
		}

		if target1.ID() != target2.ID() {
			t.Error("Multiple As calls should return the same error")
		}
	})

	t.Run("Interface{} target", func(t *testing.T) {
		// Test with interface{} target
		err := erro.New("test error")
		var target interface{}

		if err.As(&target) {
			t.Error("As should return false for interface{} target")
		}
	})

	t.Run("Nil interface target", func(t *testing.T) {
		// Test with nil interface target
		err := erro.New("test error")
		var target interface{} = nil

		if err.As(&target) {
			t.Error("As should return false for nil interface target")
		}
	})

	t.Run("String target", func(t *testing.T) {
		// Test with string target (should fail)
		err := erro.New("test error")
		var target string

		if err.As(&target) {
			t.Error("As should return false for string target")
		}
	})

	t.Run("Int target", func(t *testing.T) {
		// Test with int target (should fail)
		err := erro.New("test error")
		var target int

		if err.As(&target) {
			t.Error("As should return false for int target")
		}
	})

	t.Run("Slice target", func(t *testing.T) {
		// Test with slice target (should fail)
		err := erro.New("test error")
		var target []string

		if err.As(&target) {
			t.Error("As should return false for slice target")
		}
	})

	t.Run("Map target", func(t *testing.T) {
		// Test with map target (should fail)
		err := erro.New("test error")
		var target map[string]string

		if err.As(&target) {
			t.Error("As should return false for map target")
		}
	})

	t.Run("Struct target", func(t *testing.T) {
		// Test with struct target (should fail)
		err := erro.New("test error")
		var target struct{ Field string }

		if err.As(&target) {
			t.Error("As should return false for struct target")
		}
	})

	t.Run("Channel target", func(t *testing.T) {
		// Test with channel target (should fail)
		err := erro.New("test error")
		var target chan int

		if err.As(&target) {
			t.Error("As should return false for channel target")
		}
	})

	t.Run("Function target", func(t *testing.T) {
		// Test with function target (should fail)
		err := erro.New("test error")
		var target func()

		if err.As(&target) {
			t.Error("As should return false for function target")
		}
	})
}

// TestIsAsEdgeCases tests edge cases and boundary conditions
func TestIsAsEdgeCases(t *testing.T) {
	t.Run("Empty error messages", func(t *testing.T) {
		err1 := erro.New("")
		err2 := erro.New("")

		if err1.Is(err2) {
			t.Error("Is should return false for errors with empty messages and no IDs")
		}
	})

	t.Run("Very long error messages", func(t *testing.T) {
		longMsg := string(make([]byte, 1000))
		err1 := erro.New(longMsg)
		err2 := erro.New(longMsg)

		if err1.Is(err2) {
			t.Error("Is should return false for errors with long messages and no IDs")
		}
	})

	t.Run("Special characters in messages", func(t *testing.T) {
		specialMsg := "error with special chars: 测试 🚀 émojis 🔥"
		err1 := erro.New(specialMsg)
		err2 := erro.New(specialMsg)

		if err1.Is(err2) {
			t.Error("Is should return false for errors with special characters and no IDs")
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		err := erro.New("concurrent test error")

		// Test concurrent Is calls
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				_ = err.Is(errors.New("test"))
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("Memory pressure", func(t *testing.T) {
		// Create many errors to test memory pressure
		errorList := make([]erro.Error, 1000)
		for i := 0; i < 1000; i++ {
			errorList[i] = erro.New(fmt.Sprintf("error %d", i))
		}

		// Test Is and As on all errors
		for _, err := range errorList {
			_ = err.Is(errors.New("test"))

			var target erro.Error
			_ = err.As(&target)
		}
	})

	t.Run("Recursive errors", func(t *testing.T) {
		// Test with errors that might cause recursion
		err := erro.New("recursive test")
		wrapped := erro.Wrap(err, "wrapped")

		// This should not cause infinite recursion
		_ = wrapped.Is(err)

		var target erro.Error
		_ = wrapped.As(&target)
	})

	t.Run("Error with all metadata", func(t *testing.T) {
		// Test with error that has all possible metadata
		err := erro.New("full metadata test",
			erro.ID("test_id"),
			erro.ClassValidation,
			erro.CategoryDatabase,
			erro.SeverityHigh,
			erro.Retryable(),
			"field1", "value1",
			"field2", 123,
		)

		// Test Is
		template := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}
		if !err.Is(template) {
			t.Error("Is should work with full metadata")
		}

		// Test As
		var target erro.Error
		if !err.As(&target) {
			t.Error("As should work with full metadata")
		}
	})
}

// TestIsAsPerformance tests performance characteristics
func TestIsAsPerformance(t *testing.T) {
	t.Run("Is performance", func(t *testing.T) {
		err := erro.New("performance test")
		template := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}

		// Benchmark Is calls
		start := time.Now()
		for i := 0; i < 10000; i++ {
			_ = err.Is(template)
		}
		duration := time.Since(start)

		avgPerCall := duration / 10000
		if avgPerCall > 10*time.Microsecond {
			t.Errorf("Is performance too slow: %v per call", avgPerCall)
		}
	})

	t.Run("As performance", func(t *testing.T) {
		err := erro.New("performance test")

		// Benchmark As calls
		start := time.Now()
		for i := 0; i < 10000; i++ {
			var target erro.Error
			_ = err.As(&target)
		}
		duration := time.Since(start)

		avgPerCall := duration / 10000
		if avgPerCall > 10*time.Microsecond {
			t.Errorf("As performance too slow: %v per call", avgPerCall)
		}
	})

	t.Run("Deep chain performance", func(t *testing.T) {
		// Create deep error chain
		var err erro.Error = erro.New("base error")
		for i := 0; i < 100; i++ {
			err = erro.Wrap(err, fmt.Sprintf("layer %d", i))
		}

		// Test Is performance on deep chain
		template := &templateError{
			class:     erro.ClassValidation,
			category:  erro.CategoryDatabase,
			severity:  erro.SeverityHigh,
			retryable: true,
		}

		start := time.Now()
		for i := 0; i < 1000; i++ {
			_ = err.Is(template)
		}
		duration := time.Since(start)

		avgPerCall := duration / 1000
		if avgPerCall > 100*time.Microsecond {
			t.Errorf("Deep chain Is performance too slow: %v per call", avgPerCall)
		}

		// Test As performance on deep chain
		start = time.Now()
		for i := 0; i < 1000; i++ {
			var target erro.Error
			_ = err.As(&target)
		}
		duration = time.Since(start)

		avgPerCall = duration / 1000
		if avgPerCall > 100*time.Microsecond {
			t.Errorf("Deep chain As performance too slow: %v per call", avgPerCall)
		}
	})
}

// TestMultiErrorIsAs tests the Is and As methods for multiError type
func TestMultiErrorIsAs(t *testing.T) {
	t.Run("Is method - matching errors", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))
		err3 := erro.New("error 3", erro.ID("id3"))

		// Create multi-error
		list := erro.NewList()
		list.Add(err1)
		list.Add(err2)
		list.Add(err3)
		multiErr := list.Err()

		// Test Is with each individual error
		if !errors.Is(multiErr, err1) {
			t.Error("Is should return true for first error in multi-error")
		}
		if !errors.Is(multiErr, err2) {
			t.Error("Is should return true for second error in multi-error")
		}
		if !errors.Is(multiErr, err3) {
			t.Error("Is should return true for third error in multi-error")
		}
	})

	t.Run("Is method - non-matching errors", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))

		// Create multi-error with only err1
		list := erro.NewList()
		list.Add(err1)
		multiErr := list.Err()

		// Test Is with non-matching error
		if errors.Is(multiErr, err2) {
			t.Error("Is should return false for non-matching error")
		}
	})

	t.Run("Is method - standard errors", func(t *testing.T) {
		// Create standard errors
		stdErr1 := errors.New("standard error 1")
		stdErr2 := errors.New("standard error 2")

		// Create multi-error with standard errors
		list := erro.NewList()
		list.Add(stdErr1)
		list.Add(stdErr2)
		multiErr := list.Err()

		// Test Is with standard errors
		if !errors.Is(multiErr, stdErr1) {
			t.Error("Is should return true for first standard error in multi-error")
		}
		if !errors.Is(multiErr, stdErr2) {
			t.Error("Is should return true for second standard error in multi-error")
		}
	})

	t.Run("Is method - mixed error types", func(t *testing.T) {
		// Create mixed error types
		erroErr := erro.New("erro error", erro.ID("erro_id"))
		stdErr := errors.New("standard error")
		customErr := &customError{msg: "custom error"}

		// Create multi-error with mixed types
		list := erro.NewList()
		list.Add(erroErr)
		list.Add(stdErr)
		list.Add(customErr)
		multiErr := list.Err()

		// Test Is with each type
		if !errors.Is(multiErr, erroErr) {
			t.Error("Is should return true for erro error in multi-error")
		}
		if !errors.Is(multiErr, stdErr) {
			t.Error("Is should return true for standard error in multi-error")
		}
		if !errors.Is(multiErr, customErr) {
			t.Error("Is should return true for custom error in multi-error")
		}
	})

	t.Run("As method - erro.Error target", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))

		// Create multi-error
		list := erro.NewList()
		list.Add(err1)
		list.Add(err2)
		multiErr := list.Err()

		// Test As with erro.Error target
		var target erro.Error
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for erro.Error target")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		// Should return the first erro.Error in the list
		if target.ID() != err1.ID() {
			t.Error("Target should contain the first erro.Error from multi-error")
		}
	})

	t.Run("As method - custom error target", func(t *testing.T) {
		// Create custom error
		customErr := &customError{msg: "custom error"}
		stdErr := errors.New("standard error")

		// Create multi-error with custom error
		list := erro.NewList()
		list.Add(customErr)
		list.Add(stdErr)
		multiErr := list.Err()

		// Test As with custom error target
		var target *customError
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for custom error target")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		if target.msg != "custom error" {
			t.Errorf("Target should contain original custom error, got: %s", target.msg)
		}
	})

	t.Run("As method - custom error target", func(t *testing.T) {
		// Create custom errors
		customErr1 := &customError{msg: "custom error 1"}
		customErr2 := &customError{msg: "custom error 2"}

		// Create multi-error with custom errors
		list := erro.NewList()
		list.Add(customErr1)
		list.Add(customErr2)
		multiErr := list.Err()

		// Test As with custom error target
		var target *customError
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for custom error target")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		// Should return the first error in the list
		if target.msg != "custom error 1" {
			t.Errorf("Target should contain the first error from multi-error, got: %s", target.msg)
		}
	})

	t.Run("As method - wrong target type", func(t *testing.T) {
		// Create multi-error
		list := erro.NewList()
		list.Add(erro.New("error 1"))
		list.Add(erro.New("error 2"))
		multiErr := list.Err()

		// Test As with wrong target type
		var target *customError
		if errors.As(multiErr, &target) {
			t.Error("As should return false for wrong target type")
		}
		if target != nil {
			t.Error("Target should remain nil for failed As")
		}
	})

	t.Run("Empty multi-error", func(t *testing.T) {
		// Create empty list
		list := erro.NewList()
		multiErr := list.Err()

		// Test Is with empty multi-error
		if multiErr != nil {
			t.Error("Empty list should return nil error")
		}

		// Test As with empty multi-error
		var target erro.Error
		if errors.As(multiErr, &target) {
			t.Error("As should return false for nil multi-error")
		}
	})

	t.Run("Single error in multi-error", func(t *testing.T) {
		// Create single error
		err := erro.New("single error", erro.ID("single_id"))

		// Create list with single error
		list := erro.NewList()
		list.Add(err)
		multiErr := list.Err()

		// Should return the single error directly, not a multi-error
		if !errors.Is(multiErr, err) {
			t.Error("Single error should be returned directly and match with Is")
		}
		// The Err() method should return the single error directly
		if multiErr != err {
			t.Error("Single error should be returned directly from Err()")
		}
	})
}

// TestMultiErrorSetIsAs tests the Is and As methods for multiErrorSet type
func TestMultiErrorSetIsAs(t *testing.T) {
	t.Run("Is method - matching errors", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))
		err3 := erro.New("error 3", erro.ID("id3"))

		// Create set with unique errors
		set := erro.NewSet()
		set.Add(err1)
		set.Add(err2)
		set.Add(err3)
		multiErr := set.Err()

		// Test Is with each individual error
		if !errors.Is(multiErr, err1) {
			t.Error("Is should return true for first error in multi-error set")
		}
		if !errors.Is(multiErr, err2) {
			t.Error("Is should return true for second error in multi-error set")
		}
		if !errors.Is(multiErr, err3) {
			t.Error("Is should return true for third error in multi-error set")
		}
	})

	t.Run("Is method - non-matching errors", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))

		// Create set with only err1
		set := erro.NewSet()
		set.Add(err1)
		multiErr := set.Err()

		// Test Is with non-matching error
		if errors.Is(multiErr, err2) {
			t.Error("Is should return false for non-matching error in set")
		}
	})

	t.Run("Is method - deduplicated errors", func(t *testing.T) {
		// Create same error multiple times
		err := erro.New("duplicate error", erro.ID("dup_id"))

		// Create set with duplicate errors
		set := erro.NewSet()
		set.Add(err)
		set.Add(err) // This should be deduplicated
		set.Add(err) // This should be deduplicated
		multiErr := set.Err()

		// Test Is with the deduplicated error
		if !errors.Is(multiErr, err) {
			t.Error("Is should return true for deduplicated error in set")
		}
	})

	t.Run("Is method - standard errors", func(t *testing.T) {
		// Create standard errors
		stdErr1 := errors.New("standard error 1")
		stdErr2 := errors.New("standard error 2")

		// Create set with standard errors
		set := erro.NewSet()
		set.Add(stdErr1)
		set.Add(stdErr2)
		multiErr := set.Err()

		// Test Is with standard errors
		if !errors.Is(multiErr, stdErr1) {
			t.Error("Is should return true for first standard error in set")
		}
		if !errors.Is(multiErr, stdErr2) {
			t.Error("Is should return true for second standard error in set")
		}
	})

	t.Run("Is method - mixed error types", func(t *testing.T) {
		// Create mixed error types
		erroErr := erro.New("erro error", erro.ID("erro_id"))
		stdErr := errors.New("standard error")
		customErr := &customError{msg: "custom error"}

		// Create set with mixed types
		set := erro.NewSet()
		set.Add(erroErr)
		set.Add(stdErr)
		set.Add(customErr)
		multiErr := set.Err()

		// Test Is with each type
		if !errors.Is(multiErr, erroErr) {
			t.Error("Is should return true for erro error in set")
		}
		if !errors.Is(multiErr, stdErr) {
			t.Error("Is should return true for standard error in set")
		}
		if !errors.Is(multiErr, customErr) {
			t.Error("Is should return true for custom error in set")
		}
	})

	t.Run("As method - erro.Error target", func(t *testing.T) {
		// Create individual errors
		err1 := erro.New("error 1", erro.ID("id1"))
		err2 := erro.New("error 2", erro.ID("id2"))

		// Create set
		set := erro.NewSet()
		set.Add(err1)
		set.Add(err2)
		multiErr := set.Err()

		// Test As with erro.Error target
		var target erro.Error
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for erro.Error target in set")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		// Should return the first erro.Error in the set
		if target.ID() != err1.ID() {
			t.Error("Target should contain the first erro.Error from set")
		}
	})

	t.Run("As method - custom error target", func(t *testing.T) {
		// Create custom error
		customErr := &customError{msg: "custom error"}
		stdErr := errors.New("standard error")

		// Create set with custom error
		set := erro.NewSet()
		set.Add(customErr)
		set.Add(stdErr)
		multiErr := set.Err()

		// Test As with custom error target
		var target *customError
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for custom error target in set")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		if target.msg != "custom error" {
			t.Errorf("Target should contain original custom error, got: %s", target.msg)
		}
	})

	t.Run("As method - custom error target", func(t *testing.T) {
		// Create custom errors
		customErr1 := &customError{msg: "custom error 1"}
		customErr2 := &customError{msg: "custom error 2"}

		// Create set with custom errors
		set := erro.NewSet()
		set.Add(customErr1)
		set.Add(customErr2)
		multiErr := set.Err()

		// Test As with custom error target
		var target *customError
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for custom error target in set")
		}
		if target == nil {
			t.Error("Target should not be nil after successful As")
		}
		// Should return the first error in the set
		if target.msg != "custom error 1" {
			t.Errorf("Target should contain the first error from set, got: %s", target.msg)
		}
	})

	t.Run("As method - wrong target type", func(t *testing.T) {
		// Create set
		set := erro.NewSet()
		set.Add(erro.New("error 1"))
		set.Add(erro.New("error 2"))
		multiErr := set.Err()

		// Test As with wrong target type
		var target *customError
		if errors.As(multiErr, &target) {
			t.Error("As should return false for wrong target type in set")
		}
		if target != nil {
			t.Error("Target should remain nil for failed As")
		}
	})

	t.Run("Empty set", func(t *testing.T) {
		// Create empty set
		set := erro.NewSet()
		multiErr := set.Err()

		// Test Is with empty set
		if multiErr != nil {
			t.Error("Empty set should return nil error")
		}

		// Test As with empty set
		var target erro.Error
		if errors.As(multiErr, &target) {
			t.Error("As should return false for nil set error")
		}
	})

	t.Run("Single error in set", func(t *testing.T) {
		// Create single error
		err := erro.New("single error", erro.ID("single_id"))

		// Create set with single error
		set := erro.NewSet()
		set.Add(err)
		multiErr := set.Err()

		// Should return the single error directly, not a multi-error
		if !errors.Is(multiErr, err) {
			t.Error("Single error should be returned directly and match with Is")
		}
		// The Err() method should return the single error directly
		if multiErr != err {
			t.Error("Single error should be returned directly from Err()")
		}
	})

	t.Run("Deduplication with custom key getter", func(t *testing.T) {
		// Create errors with same message but different IDs
		err1 := erro.New("same message", erro.ID("id1"))
		err2 := erro.New("same message", erro.ID("id2"))

		// Create set with custom key getter that uses message
		set := erro.NewSet()
		set.WithKeyGetter(func(err error) string {
			if e, ok := err.(erro.Error); ok {
				return e.Message()
			}
			return err.Error()
		})
		set.Add(err1)
		set.Add(err2)
		multiErr := set.Err()

		// Should only contain one error due to deduplication
		if !errors.Is(multiErr, err1) {
			t.Error("Is should return true for first error after deduplication")
		}
		if errors.Is(multiErr, err2) {
			t.Error("Is should return false for second error after deduplication")
		}
	})
}

// TestMultiErrorEdgeCases tests edge cases for multi-error types
func TestMultiErrorEdgeCases(t *testing.T) {
	t.Run("Very large multi-error", func(t *testing.T) {
		// Create many errors
		list := erro.NewList()
		for i := 0; i < 1000; i++ {
			list.Add(erro.New(fmt.Sprintf("error %d", i), erro.ID(fmt.Sprintf("id_%d", i))))
		}
		multiErr := list.Err()

		// Test Is with first and last error
		firstErr := list.Errs()[0]
		lastErr := list.Errs()[len(list.Errs())-1]

		if !errors.Is(multiErr, firstErr) {
			t.Error("Is should return true for first error in large multi-error")
		}
		if !errors.Is(multiErr, lastErr) {
			t.Error("Is should return true for last error in large multi-error")
		}

		// Test As
		var target erro.Error
		if !errors.As(multiErr, &target) {
			t.Error("As should return true for large multi-error")
		}
	})

	t.Run("Concurrent access to multi-error", func(t *testing.T) {
		// Create multi-error
		list := erro.NewList()
		list.Add(erro.New("error 1"))
		list.Add(erro.New("error 2"))
		list.Add(erro.New("error 3"))
		multiErr := list.Err()

		// Test concurrent Is calls
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				_ = errors.Is(multiErr, errors.New("test"))
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("Nested multi-errors", func(t *testing.T) {
		// Create nested multi-errors
		innerList := erro.NewList()
		innerList.Add(erro.New("inner error 1"))
		innerList.Add(erro.New("inner error 2"))
		innerMultiErr := innerList.Err()

		outerList := erro.NewList()
		outerList.Add(erro.New("outer error"))
		outerList.Add(innerMultiErr)
		outerMultiErr := outerList.Err()

		// Test Is with inner errors
		innerErr1 := innerList.Errs()[0]
		if !errors.Is(outerMultiErr, innerErr1) {
			t.Error("Is should return true for nested inner error")
		}
	})

	t.Run("Multi-error with nil errors", func(t *testing.T) {
		// Create list with nil errors (should be filtered out)
		list := erro.NewList()
		list.Add(nil)
		list.Add(erro.New("valid error"))
		list.Add(nil)
		multiErr := list.Err()

		// Should only contain the valid error
		validErr := list.Errs()[0]
		if !errors.Is(multiErr, validErr) {
			t.Error("Is should return true for valid error when nil errors are present")
		}
	})

	t.Run("Multi-error with empty key errors", func(t *testing.T) {
		// Create set with errors that produce empty keys
		set := erro.NewSet()
		set.WithKeyGetter(func(err error) string {
			return "" // Always return empty key
		})
		set.Add(erro.New("error 1"))
		set.Add(erro.New("error 2"))
		multiErr := set.Err()

		// Should be empty due to empty keys
		if multiErr != nil {
			t.Error("Set should return nil when all errors produce empty keys")
		}
	})
}

// testTemplateError is a template error with an ID field for testing
type testTemplateError struct {
	id        string
	class     erro.ErrorClass
	category  erro.ErrorCategory
	severity  erro.ErrorSeverity
	retryable bool
}

func (e *testTemplateError) Error() string                { return "" }
func (e *testTemplateError) Class() erro.ErrorClass       { return e.class }
func (e *testTemplateError) Category() erro.ErrorCategory { return e.category }
func (e *testTemplateError) Severity() erro.ErrorSeverity { return e.severity }
func (e *testTemplateError) IsRetryable() bool            { return e.retryable }
func (e *testTemplateError) ID() string                   { return e.id }
func (e *testTemplateError) Message() string              { return "" }
func (e *testTemplateError) Fields() []any                { return nil }
func (e *testTemplateError) AllFields() []any             { return nil }
func (e *testTemplateError) Created() time.Time           { return time.Time{} }
func (e *testTemplateError) Span() erro.TraceSpan         { return nil }
func (e *testTemplateError) Stack() erro.Stack            { return nil }
func (e *testTemplateError) LogFields(...erro.LogOptions) []any {
	return nil
}
func (e *testTemplateError) LogFieldsMap(...erro.LogOptions) map[string]any {
	return nil
}
func (e *testTemplateError) BaseError() erro.Error                    { return e }
func (e *testTemplateError) StackTraceConfig() *erro.StackTraceConfig { return nil }
func (e *testTemplateError) Formatter() erro.FormatErrorFunc          { return nil }
func (e *testTemplateError) Unwrap() error                            { return nil }
func (e *testTemplateError) Is(target error) bool                     { return false }
func (e *testTemplateError) As(target any) bool                       { return false }
func (e *testTemplateError) Format(s fmt.State, verb rune)            {}
func (e *testTemplateError) MarshalJSON() ([]byte, error)             { return nil, nil }
func (e *testTemplateError) UnmarshalJSON(data []byte) error          { return nil }
