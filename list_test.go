package erro

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGroup_Basic(t *testing.T) {
	g := NewList()

	// Empty group should return nil
	if err := g.Err(); err != nil {
		t.Errorf("Expected nil error from empty group, got: %v", err)
	}

	// Add one error
	g.New("first error")
	if err := g.Err(); err == nil {
		t.Error("Expected error from group with one error")
	} else if err.Error() != "first error" {
		t.Errorf("Expected 'first error', got: %v", err.Error())
	}

	// Add second error
	g.New("second error")
	err := g.Err()
	if err == nil {
		t.Error("Expected error from group with two errors")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "multiple errors (2)") {
		t.Errorf("Expected multi-error format, got: %v", errStr)
	}
	if !strings.Contains(errStr, "first error") || !strings.Contains(errStr, "second error") {
		t.Errorf("Expected both errors in output, got: %v", errStr)
	}
}

func TestGroup_Chaining(t *testing.T) {
	g := NewList().
		Class("TEST_CLASS").
		Category("validation").
		Severity("high").
		Fields("user", "test-user")

	g.New("validation failed").New("invalid input")

	// Check that metadata was applied
	errors := g.Errs()
	if len(errors) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(errors))
	}

	for i, err := range errors {
		if err.GetID() != "TEST_CLASS" {
			t.Errorf("Error %d: expected code 'TEST_CLASS', got '%s'", i, err.GetID())
		}
		if err.GetCategory() != "validation" {
			t.Errorf("Error %d: expected category 'validation', got '%s'", i, err.GetCategory())
		}
		if err.GetSeverity() != "high" {
			t.Errorf("Error %d: expected severity 'high', got '%s'", i, err.GetSeverity())
		}
	}
}

func TestGroup_AddMethods(t *testing.T) {
	g := NewList()

	// Test Add with external error
	g.Add(errors.New("external error"))

	// Test Addf
	g.Errorf("formatted error: %s", "value")

	// Test AddWrap
	g.Wrap(errors.New("original"), "wrapped")

	// Test AddWrapf
	g.Wrapf(errors.New("original2"), "wrapped: %s", "context")

	if g.Len() != 4 {
		t.Errorf("Expected 4 errors, got %d", g.Len())
	}

	err := g.Err()
	if err == nil {
		t.Error("Expected combined error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "multiple errors (4)") {
		t.Errorf("Expected 4 errors in output, got: %v", errStr)
	}
}

func TestSet_Deduplication(t *testing.T) {
	s := NewSet()

	// Add same error multiple times
	for i := 0; i < 5; i++ {
		s.New("duplicate error")
	}

	if s.Len() != 1 {
		t.Errorf("Expected 1 unique error, got %d", s.Len())
	}

	// Add different error
	s.New("different error")

	if s.Len() != 2 {
		t.Errorf("Expected 2 unique errors, got %d", s.Len())
	}

	err := s.Err()
	if err == nil {
		t.Error("Expected combined error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "multiple errors (2)") {
		t.Errorf("Expected 2 errors in output, got: %v", errStr)
	}
}

func TestSet_CodeBasedDeduplication(t *testing.T) {
	s := NewSet().Class("SAME_CLASS")

	// Add errors with same code and message
	s.New("error message")
	s.New("error message") // Should be deduplicated

	if s.Len() != 1 {
		t.Errorf("Expected 1 unique error, got %d", s.Len())
	}

	// Change code and add same message
	s.Class("DIFFERENT_CLASS").New("error message")

	if s.Len() != 2 {
		t.Errorf("Expected 2 unique errors (different codes), got %d", s.Len())
	}
}

func TestSet_ChainingMethods(t *testing.T) {
	s := NewSet().
		Class("SET_CLASS").
		Category("test").
		Severity("low")

	// Verify chaining returns *Set
	s2 := s.Fields("key", "value")
	if s2 != s {
		t.Error("Chaining should return the same Set instance")
	}

	s.New("test error")

	errors := s.Errs()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	err := errors[0]
	if err.GetID() != "SET_CLASS" {
		t.Errorf("Expected code 'SET_CLASS', got '%s'", err.GetID())
	}
	if err.GetCategory() != "test" {
		t.Errorf("Expected category 'test', got '%s'", err.GetCategory())
	}
}

func TestMultiError_Unwrap(t *testing.T) {
	g := NewList()
	g.New("error 1")
	g.New("error 2")

	err := g.Err()
	multi, ok := err.(*multiError)
	if !ok {
		t.Fatalf("Expected *multiError, got %T", err)
	}

	unwrapped := multi.Unwrap()
	if len(unwrapped) != 2 {
		t.Errorf("Expected 2 unwrapped errors, got %d", len(unwrapped))
	}
}

func ExampleList() {
	g := NewList().
		Class("VALIDATION").
		Category("input").
		Severity("high")

	// Simulate validation errors
	g.New("name is required").
		New("email is invalid").
		Wrap(errors.New("age out of range"), "validation failed")

	if err := g.Err(); err != nil {
		fmt.Println("Validation failed:")
		fmt.Println(err.Error())
	}

	// Output:
	// Validation failed:
	// multiple errors (3): (1) [HIGH] name is required; (2) [HIGH] email is invalid; (3) [HIGH] validation failed: age out of range
}

func ExampleSet() {
	s := NewSet().
		Class("RETRY").
		Category("network").
		Severity("medium")

	// Simulate retry scenario with duplicate errors
	for i := 0; i < 100; i++ {
		// This error will only be stored once
		s.New("connection timeout")

		// These will be stored as separate errors (different messages)
		if i%10 == 0 {
			s.Errorf("retry attempt %d failed", i)
		}
	}

	fmt.Printf("Stored %d unique errors out of 110 attempts\n", s.Len())

	// Output:
	// Stored 11 unique errors out of 110 attempts
}
