package erro

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add(errors.New("test error"))
	if list.Len() != 1 {
		t.Errorf("expected 1 error, got %d", list.Len())
	}
	list.Add(nil)
	if list.Len() != 1 {
		t.Errorf("expected 1 error, got %d", list.Len())
	}
}

func TestList_New(t *testing.T) {
	list := NewList()
	list.New("test error")
	if list.Len() != 1 {
		t.Errorf("expected 1 error, got %d", list.Len())
	}
}

func TestList_Wrap(t *testing.T) {
	list := NewList()
	list.Wrap(errors.New("test error"), "wrapped")
	if list.Len() != 1 {
		t.Errorf("expected 1 error, got %d", list.Len())
	}
}

func TestList_Err(t *testing.T) {
	list := NewList()
	if list.Err() != nil {
		t.Error("expected multiError for empty list")
	}
	list.Add(errors.New("test error"))
	if list.Err() == nil {
		t.Error("expected an error for non-empty list")
	}
	list.Add(errors.New("test error 2"))
	err := list.Err()
	if err == nil {
		t.Error("expected an error for non-empty list")
	}
	if err == nil {
		t.Error("expected an error for non-empty list")
	}
}

func TestList_Remove(t *testing.T) {
	list := NewList()
	list.Add(errors.New("test error"))
	if !list.Remove(0) {
		t.Error("expected to remove an error")
	}
	if list.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", list.Len())
	}
	if list.Remove(0) {
		t.Error("expected not to remove an error from empty list")
	}
}

func TestList_RemoveError(t *testing.T) {
	list := NewList()
	err := New("test error", ID("1"))
	list.Add(err)
	if list.RemoveError(errors.New("test error")) {
		t.Error("expected not to remove an error")
	}
	if !list.RemoveError(err) {
		t.Error("expected to remove an error")
	}
	if list.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", list.Len())
	}
	if list.RemoveError(err) {
		t.Error("expected not to remove an error")
	}
	if list.RemoveError(nil) {
		t.Error("expected not to remove nil error")
	}
	noIdErr := &baseError{}
	list.Add(noIdErr)
	if list.RemoveError(noIdErr) {
		t.Error("expected not to remove error with no id")
	}
}

func TestList_RemoveInvalid(t *testing.T) {
	list := NewList()
	list.Add(errors.New("test error"))
	if list.Remove(-1) {
		t.Error("expected not to remove with negative index")
	}
	if list.Remove(1) {
		t.Error("expected not to remove with out of bounds index")
	}
	if list.Len() != 1 {
		t.Errorf("expected 1 error, got %d", list.Len())
	}
}

func TestList_Clear(t *testing.T) {
	list := NewList()
	list.Add(errors.New("test error"))
	list.Clear()
	if list.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", list.Len())
	}
}

func TestList_Copy(t *testing.T) {
	list := NewList()
	list.Add(errors.New("test error"))
	clone := list.Copy()
	if clone.Len() != 1 {
		t.Errorf("expected 1 error, got %d", clone.Len())
	}
	list.Add(errors.New("another error"))
	if clone.Len() != 1 {
		t.Errorf("clone should not be affected by original list changes, expected 1 error, got %d", clone.Len())
	}
}

func TestList_Accessors(t *testing.T) {
	list := NewList()
	if !list.Empty() {
		t.Error("expected list to be empty")
	}
	if list.NotEmpty() {
		t.Error("expected list not to be not-empty")
	}
	if list.First() != nil {
		t.Error("expected nil first error")
	}
	if list.Last() != nil {
		t.Error("expected nil last error")
	}
	err1 := New("err1")
	err2 := New("err2")
	list.Add(err1)
	list.Add(err2)
	if list.Len() != 2 {
		t.Errorf("expected 2 errors, got %d", list.Len())
	}
	if list.Empty() {
		t.Error("expected list not to be empty")
	}
	if !list.NotEmpty() {
		t.Error("expected list to be not-empty")
	}
	if list.First() != err1 {
		t.Error("unexpected first error")
	}
	if list.Last() != err2 {
		t.Error("unexpected last error")
	}
	if len(list.Errors()) != 2 {
		t.Error("unexpected number of errors")
	}
	if len(list.Errs()) != 2 {
		t.Error("unexpected number of errors")
	}
}

func TestSet_Add(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error"))
	set.Add(errors.New("test error"))
	set.Add(nil)
	if set.Len() != 1 {
		t.Errorf("expected 1 error, got %d", set.Len())
	}
}

func TestSet_New(t *testing.T) {
	set := NewSet()
	set.New("test error")
	set.New("test error")
	if set.Len() != 1 {
		t.Errorf("expected 1 error, got %d", set.Len())
	}
}

func TestSet_Wrap(t *testing.T) {
	set := NewSet()
	err := errors.New("test error")
	set.Wrap(err, "wrapped")
	set.Wrap(err, "wrapped")
	if set.Len() != 1 {
		t.Errorf("expected 1 error, got %d", set.Len())
	}
}

func TestSet_WithKeyGetter(t *testing.T) {
	set := NewSet()
	set.WithKeyGetter(IDKeyGetter)
	set.New("test error", ID("1"))
	set.New("another error", ID("1"))
	if set.Len() != 1 {
		t.Errorf("expected 1 error, got %d", set.Len())
	}
	set.WithKeyGetter(nil) // should not panic
}

func TestSet_Err(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error"))
	set.Add(errors.New("test error"))
	err := set.Err()
	if err == nil {
		t.Error("expected an error")
	}
	if !errors.Is(err, set.First()) {
		t.Error("expected error to be in the set")
	}
}

func TestSet_Clear(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error"))
	set.Clear()
	if set.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", set.Len())
	}
}

func TestSet_Copy(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error"))
	clone := set.Copy()
	if clone.Len() != 1 {
		t.Errorf("expected 1 error, got %d", clone.Len())
	}
	set.Add(errors.New("another error"))
	if clone.Len() != 1 {
		t.Errorf("clone should not be affected by original set changes, expected 1 error, got %d", clone.Len())
	}
}

func TestSet_Remove(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error"))
	if set.Remove(-1) {
		t.Error("expected not to remove with negative index")
	}
	if set.Remove(10) {
		t.Error("expected not to remove with out of bounds index")
	}
	if set.Len() != 1 {
		t.Errorf("expected 1 error, got %d", set.Len())
	}
	if !set.Remove(0) {
		t.Error("expected to remove an error")
	}
	if set.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", set.Len())
	}
}

func TestSet_RemoveError(t *testing.T) {
	set := NewSet()
	err := New("test error", ID("1"))
	set.Add(err)
	if set.RemoveError(nil) {
		t.Error("expected not to remove nil error")
	}
	if !set.RemoveError(errors.New("test error")) {
		t.Error("expected not to remove error with wrong id")
	}
	if set.RemoveError(err) {
		t.Error("expected to remove an error")
	}
	if set.Len() != 0 {
		t.Errorf("expected 0 errors, got %d", set.Len())
	}
}

func TestSet_AddMultiple(t *testing.T) {
	set := NewSet()
	set.Add(errors.New("test error 1"))
	set.Add(errors.New("test error 2"))
	if set.Len() != 2 {
		t.Errorf("expected 2 errors, got %d", set.Len())
	}
	err := set.Err()
	if err == nil {
		t.Error("expected an error")
	}
	if !strings.Contains(err.Error(), "test error 1") {
		t.Errorf("expected error to contain 'test error 1'")
	}
	if !strings.Contains(err.Error(), "test error 2") {
		t.Errorf("expected error to contain 'test error 2'")
	}
}

func TestSafeList(t *testing.T) {
	safeList := NewSafeList()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safeList.Add(fmt.Errorf("error %d", i))
		}(i)
	}
	wg.Wait()
	if safeList.Len() != 100 {
		t.Errorf("expected 100 errors, got %d", safeList.Len())
	}
	if safeList.Err() == nil {
		t.Error("expected an error")
	}
	if safeList.First() == nil {
		t.Error("expected a first error")
	}
	if safeList.Last() == nil {
		t.Error("expected a last error")
	}
	if safeList.Empty() {
		t.Error("expected not empty")
	}
	if !safeList.NotEmpty() {
		t.Error("expected not empty")
	}
	if len(safeList.Errors()) != 100 {
		t.Error("unexpected number of errors")
	}
	if len(safeList.Errs()) != 100 {
		t.Error("unexpected number of errors")
	}
	clone := safeList.Copy()
	if clone.Len() != 100 {
		t.Error("unexpected clone length")
	}
	safeList.Remove(0)
	if safeList.Len() != 99 {
		t.Error("unexpected length after remove")
	}
	errToRemove := safeList.First()
	safeList.RemoveError(errToRemove)
	if safeList.Len() != 98 {
		t.Error("unexpected length after remove error")
	}
	safeList.Clear()
	if safeList.Len() != 0 {
		t.Error("unexpected length after clear")
	}
	safeList.New("new error")
	if safeList.Len() != 1 {
		t.Error("unexpected length after new")
	}
	safeList.Wrap(errors.New("test"), "wrapped")
	if safeList.Len() != 2 {
		t.Error("unexpected length after wrap")
	}
}

func TestSafeSet(t *testing.T) {
	safeSet := NewSafeSet()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			safeSet.Add(errors.New("test error"))
		}()
	}
	wg.Wait()
	if safeSet.Len() != 1 {
		t.Errorf("expected 1 error, got %d", safeSet.Len())
	}
	safeSet.WithKeyGetter(IDKeyGetter) // should not panic

	safeSet.Clear()
	if safeSet.Len() != 0 {
		t.Error("unexpected length after clear")
	}

	safeSet.New("new error", ID("1"))
	if safeSet.Len() != 1 {
		t.Error("unexpected length after new")
	}

	safeSet.Wrap(errors.New("test"), "wrapped", ID("2"))
	if safeSet.Len() != 2 {
		t.Error("unexpected length after wrap")
	}

	if safeSet.Empty() {
		t.Error("expected not empty")
	}

	if !safeSet.NotEmpty() {
		t.Error("expected not empty")
	}

	if len(safeSet.Errors()) != 2 {
		t.Error("unexpected number of errors")
	}

	if len(safeSet.Errs()) != 2 {
		t.Error("unexpected number of errors")
	}

	clone := safeSet.Copy()
	if clone.Len() != 2 {
		t.Error("unexpected clone length")
	}

	firstErr := safeSet.First()
	if firstErr == nil {
		t.Error("expected first error")
	}

	lastErr := safeSet.Last()
	if lastErr == nil {
		t.Error("expected last error")
	}

	if !safeSet.Remove(0) {
		t.Error("expected to remove an error")
	}

	if !safeSet.RemoveError(lastErr) {
		t.Error("expected to remove an error")
	}
	if safeSet.Err() != nil {
		t.Error("expected an error")
	}

	safeSet.New("new error", ID("1"))
	if safeSet.Err().Error() != "new error" {
		t.Error("unexpected error message")
	}
}

func TestMultiErrorSet_Error(t *testing.T) {
	counter := map[string]int{"err1": 2, "err2": 1}
	m := &multiErrorSet{
		errors:    []error{errors.New("err1"), errors.New("err2")},
		counter:   counter,
		keyGetter: func(err error) string { return err.Error() },
	}
	expected := "multiple unique errors (2): [1] err1 (2 times); [2] err2"
	if m.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, m.Error())
	}
}

func TestSafeList_ConcurrentAddRemove(t *testing.T) {
	safeList := NewSafeList()
	var wg sync.WaitGroup
	// Add 1000 errors
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safeList.New(fmt.Sprintf("error %d", i), ID(fmt.Sprintf("%d", i)))
		}(i)
	}
	wg.Wait()

	// Concurrently remove 500 errors
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safeList.RemoveError(New("", ID(fmt.Sprintf("%d", i))))
		}(i)
	}
	wg.Wait()

	if safeList.Len() != 500 {
		t.Errorf("expected 500 errors, got %d", safeList.Len())
	}
}

func TestMultiError_Unwrap(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	m := &multiError{errors: []error{err1, err2}}
	unwrapped := m.Unwrap()
	if len(unwrapped) != 2 || unwrapped[0] != err1 || unwrapped[1] != err2 {
		t.Errorf("unexpected unwrapped errors: %v", unwrapped)
	}
}

func TestMultiErrorSet_Unwrap(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	counter := map[string]int{"err1": 1, "err2": 1}
	m := &multiErrorSet{
		errors:    []error{err1, err2},
		counter:   counter,
		keyGetter: func(err error) string { return err.Error() },
	}
	unwrapped := m.Unwrap()
	if len(unwrapped) != 2 || unwrapped[0] != err1 || unwrapped[1] != err2 {
		t.Errorf("unexpected unwrapped errors: %v", unwrapped)
	}
}

func TestSafeSet_ConcurrentAdd(t *testing.T) {
	safeSet := NewSafeSet()
	safeSet.WithKeyGetter(IDKeyGetter)
	var wg sync.WaitGroup
	// Add 1000 errors with 100 unique IDs
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			safeSet.New("error", ID(fmt.Sprintf("%d", i%100)))
		}(i)
	}
	wg.Wait()

	if safeSet.Len() != 100 {
		t.Errorf("expected 100 errors, got %d", safeSet.Len())
	}
}

func TestKeyGetters(t *testing.T) {
	t.Run("MessageKeyGetter", func(t *testing.T) {
		// Test with erro.Error
		err1 := New("test message", ID("123"))
		key1 := MessageKeyGetter(err1)
		if key1 != "test message" {
			t.Errorf("expected 'test message', got '%s'", key1)
		}

		// Test with wrapped erro.Error - should get the wrapped message
		err2 := Wrap(err1, "wrapped message")
		key2 := MessageKeyGetter(err2)
		if key2 != "wrapped message" {
			t.Errorf("expected 'wrapped message', got '%s'", key2)
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		key3 := MessageKeyGetter(stdErr)
		if key3 != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key3)
		}

		// Test with nil error
		key4 := MessageKeyGetter(nil)
		if key4 != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key4)
		}
	})

	t.Run("IDKeyGetter", func(t *testing.T) {
		// Test with erro.Error with ID
		err1 := New("test message", ID("123"))
		key1 := IDKeyGetter(err1)
		if key1 != "123" {
			t.Errorf("expected '123', got '%s'", key1)
		}

		// Test with erro.Error without ID - should fallback to Error()
		err2 := New("test message")
		key2 := IDKeyGetter(err2)
		expected2 := err2.ID()
		if key2 != expected2 {
			t.Errorf("expected '%s' (fallback to newID()), got '%s'", expected2, key2)
		}

		// Test with wrapped erro.Error - should get the wrapped error's ID
		err3 := Wrap(err1, "wrapped message")
		key3 := IDKeyGetter(err3)
		if key3 != "123" {
			t.Errorf("expected '123', got '%s'", key3)
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		key4 := IDKeyGetter(stdErr)
		if key4 != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key4)
		}

		// Test with nil error
		key5 := IDKeyGetter(nil)
		if key5 != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key5)
		}
	})

	t.Run("ErrorKeyGetter", func(t *testing.T) {
		// Test with erro.Error
		err1 := New("test message", ID("123"))
		key1 := ErrorKeyGetter(err1)
		expected1 := err1.Error()
		if key1 != expected1 {
			t.Errorf("expected '%s', got '%s'", expected1, key1)
		}

		// Test with standard error
		stdErr := errors.New("standard error")
		key2 := ErrorKeyGetter(stdErr)
		if key2 != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key2)
		}

		// Test with wrapped error
		wrappedErr := Wrap(stdErr, "wrapped")
		key3 := ErrorKeyGetter(wrappedErr)
		expected3 := wrappedErr.Error()
		if key3 != expected3 {
			t.Errorf("expected '%s', got '%s'", expected3, key3)
		}

		// Test with nil error
		key4 := ErrorKeyGetter(nil)
		if key4 != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key4)
		}
	})
}

func TestKeyGetters_AsBehavior(t *testing.T) {
	t.Run("MessageKeyGetter_WithWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("standard error message")

		// Wrap it with erro
		wrappedErr := Wrap(stdErr, "wrapped message")

		// Test MessageKeyGetter with wrapped standard error
		key := MessageKeyGetter(wrappedErr)
		if key != "wrapped message" {
			t.Errorf("expected 'wrapped message', got '%s'", key)
		}
	})

	t.Run("IDKeyGetter_WithWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("standard error message")

		// Wrap it with erro
		wrappedErr := Wrap(stdErr, "wrapped message")

		// Test IDKeyGetter with wrapped standard error
		key := IDKeyGetter(wrappedErr)
		expected := wrappedErr.ID()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("ErrorKeyGetter_WithWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("standard error message")

		// Wrap it with erro
		wrappedErr := Wrap(stdErr, "wrapped message")

		// Test ErrorKeyGetter with wrapped standard error
		key := ErrorKeyGetter(wrappedErr)
		expected := wrappedErr.Error()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("MessageKeyGetter_WithDeepWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("deep standard error")

		// Wrap it multiple times
		wrapped1 := Wrap(stdErr, "first wrap")
		wrapped2 := Wrap(wrapped1, "second wrap")
		wrapped3 := Wrap(wrapped2, "third wrap")

		// Test MessageKeyGetter with deeply wrapped standard error
		key := MessageKeyGetter(wrapped3)
		if key != "third wrap" {
			t.Errorf("expected 'third wrap', got '%s'", key)
		}
	})

	t.Run("IDKeyGetter_WithDeepWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("deep standard error")

		// Wrap it multiple times
		wrapped1 := Wrap(stdErr, "first wrap")
		wrapped2 := Wrap(wrapped1, "second wrap")
		wrapped3 := Wrap(wrapped2, "third wrap")

		// Test IDKeyGetter with deeply wrapped standard error
		key := IDKeyGetter(wrapped3)
		expected := wrapped3.ID()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("ErrorKeyGetter_WithDeepWrappedStandardError", func(t *testing.T) {
		// Create a standard error
		stdErr := errors.New("deep standard error")

		// Wrap it multiple times
		wrapped1 := Wrap(stdErr, "first wrap")
		wrapped2 := Wrap(wrapped1, "second wrap")
		wrapped3 := Wrap(wrapped2, "third wrap")

		// Test ErrorKeyGetter with deeply wrapped standard error
		key := ErrorKeyGetter(wrapped3)
		expected := wrapped3.Error()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("MessageKeyGetter_WithWrappedErroError", func(t *testing.T) {
		// Create an erro error
		erroErr := New("original message", ID("123"))

		// Wrap it with another erro error
		wrappedErr := Wrap(erroErr, "wrapped message")

		// Test MessageKeyGetter with wrapped erro error
		key := MessageKeyGetter(wrappedErr)
		if key != "wrapped message" {
			t.Errorf("expected 'wrapped message', got '%s'", key)
		}
	})

	t.Run("IDKeyGetter_WithWrappedErroError", func(t *testing.T) {
		// Create an erro error with ID
		erroErr := New("original message", ID("123"))

		// Wrap it with another erro error
		wrappedErr := Wrap(erroErr, "wrapped message")

		// Test IDKeyGetter with wrapped erro error
		key := IDKeyGetter(wrappedErr)
		if key != "123" {
			t.Errorf("expected '123', got '%s'", key)
		}
	})

	t.Run("MessageKeyGetter_WithWrappedErroError", func(t *testing.T) {
		// Create an erro error without ID
		erroErr := New("original message")

		// Wrap it with another erro error
		wrappedErr := fmt.Errorf("wrapped message: %w", erroErr)

		// Test IDKeyGetter with wrapped erro error without ID
		key := MessageKeyGetter(wrappedErr)
		expected := erroErr.Message()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("IDKeyGetter_WithWrappedErroError", func(t *testing.T) {
		// Create an erro error without ID
		erroErr := New("original message")

		// Wrap it with another erro error
		wrappedErr := fmt.Errorf("wrapped message: %w", erroErr)

		// Test IDKeyGetter with wrapped erro error without ID
		key := IDKeyGetter(wrappedErr)
		expected := erroErr.ID()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("ErrorKeyGetter_WithWrappedErroError", func(t *testing.T) {
		// Create an erro error
		erroErr := New("original message", ID("123"))

		// Wrap it with another erro error
		wrappedErr := Wrap(erroErr, "wrapped message")

		// Test ErrorKeyGetter with wrapped erro error
		key := ErrorKeyGetter(wrappedErr)
		expected := wrappedErr.Error()
		if key != expected {
			t.Errorf("expected '%s', got '%s'", expected, key)
		}
	})

	t.Run("MessageKeyGetter_WithNilError", func(t *testing.T) {
		// Test MessageKeyGetter with nil error
		key := MessageKeyGetter(nil)
		if key != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key)
		}
	})

	t.Run("IDKeyGetter_WithNilError", func(t *testing.T) {
		// Test IDKeyGetter with nil error
		key := IDKeyGetter(nil)
		if key != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key)
		}
	})

	t.Run("ErrorKeyGetter_WithNilError", func(t *testing.T) {
		// Test ErrorKeyGetter with nil error
		key := ErrorKeyGetter(nil)
		if key != "" {
			t.Errorf("expected empty string for nil error, got '%s'", key)
		}
	})

	t.Run("MessageKeyGetter_WithStandardError", func(t *testing.T) {
		// Test MessageKeyGetter with standard error
		stdErr := errors.New("standard error")
		key := MessageKeyGetter(stdErr)
		if key != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key)
		}
	})

	t.Run("IDKeyGetter_WithStandardError", func(t *testing.T) {
		// Test IDKeyGetter with standard error
		stdErr := errors.New("standard error")
		key := IDKeyGetter(stdErr)
		if key != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key)
		}
	})

	t.Run("ErrorKeyGetter_WithStandardError", func(t *testing.T) {
		// Test ErrorKeyGetter with standard error
		stdErr := errors.New("standard error")
		key := ErrorKeyGetter(stdErr)
		if key != "standard error" {
			t.Errorf("expected 'standard error', got '%s'", key)
		}
	})
}

func TestSet_WithWrappedStandardErrors(t *testing.T) {
	t.Run("SetWithMessageKeyGetter_StandardErrors", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(MessageKeyGetter)

		// Create standard errors and wrap them with same message
		stdErr1 := errors.New("same message")
		stdErr2 := errors.New("same message")
		stdErr3 := errors.New("different message")

		wrapped1 := Wrap(stdErr1, "same wrap message")
		wrapped2 := Wrap(stdErr2, "same wrap message")
		wrapped3 := Wrap(stdErr3, "different wrap message")

		set.Add(wrapped1)
		set.Add(wrapped2) // Should be deduplicated (same message)
		set.Add(wrapped3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique messages
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "same wrap message") {
			t.Error("expected error to contain 'same wrap message'")
		}
		if !strings.Contains(errStr, "different wrap message") {
			t.Error("expected error to contain 'different wrap message'")
		}
	})

	t.Run("SetWithIDKeyGetter_StandardErrors", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(IDKeyGetter)

		// Create standard errors and wrap them with same message
		stdErr1 := errors.New("same underlying message")
		stdErr2 := errors.New("same underlying message")
		stdErr3 := errors.New("different underlying message")

		wrapped1 := Wrap(stdErr1, "same wrap")
		wrapped2 := Wrap(stdErr2, "same wrap")
		wrapped3 := Wrap(stdErr3, "different wrap")

		set.Add(wrapped1)
		set.Add(wrapped2) // Should be deduplicated (same error string)
		set.Add(wrapped3)

		if set.Len() != 3 {
			t.Errorf("expected 3 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique errors
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "same wrap") {
			t.Error("expected error to contain 'same wrap'")
		}
		if !strings.Contains(errStr, "different wrap") {
			t.Error("expected error to contain 'different wrap'")
		}
	})

	t.Run("SetWithErrorKeyGetter_StandardErrors", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(ErrorKeyGetter)

		// Create standard errors and wrap them with same message
		stdErr1 := errors.New("same message")
		stdErr2 := errors.New("same message")
		stdErr3 := errors.New("different message")

		wrapped1 := Wrap(stdErr1, "same wrap")
		wrapped2 := Wrap(stdErr2, "same wrap")
		wrapped3 := Wrap(stdErr3, "different wrap")

		set.Add(wrapped1)
		set.Add(wrapped2) // Should be deduplicated
		set.Add(wrapped3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique errors
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "same wrap") {
			t.Error("expected error to contain 'same wrap'")
		}
		if !strings.Contains(errStr, "different wrap") {
			t.Error("expected error to contain 'different wrap'")
		}
	})
}

func TestSet_WithKeyGetters(t *testing.T) {
	t.Run("SetWithMessageKeyGetter", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(MessageKeyGetter)

		// Add errors with same message but different IDs
		err1 := New("same message", ID("1"))
		err2 := New("same message", ID("2"))
		err3 := New("different message", ID("3"))

		set.Add(err1)
		set.Add(err2) // Should be deduplicated
		set.Add(err3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique messages
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "same message") {
			t.Error("expected error to contain 'same message'")
		}
		if !strings.Contains(errStr, "different message") {
			t.Error("expected error to contain 'different message'")
		}
	})

	t.Run("SetWithIDKeyGetter", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(IDKeyGetter)

		// Add errors with same ID but different messages
		err1 := New("message 1", ID("same-id"))
		err2 := New("message 2", ID("same-id"))
		err3 := New("message 3", ID("different-id"))

		set.Add(err1)
		set.Add(err2) // Should be deduplicated
		set.Add(err3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique error messages
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "message 1") {
			t.Error("expected error to contain 'message 1'")
		}
		if !strings.Contains(errStr, "message 3") {
			t.Error("expected error to contain 'message 3'")
		}
	})

	t.Run("SetWithErrorKeyGetter", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(ErrorKeyGetter)

		// Add errors with same string representation
		err1 := New("same message", ID("1"))
		err2 := New("same message", ID("1")) // Same ID too
		err3 := New("different message", ID("2"))

		set.Add(err1)
		set.Add(err2) // Should be deduplicated
		set.Add(err3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}

		// Verify the error message contains both unique errors
		err := set.Err()
		if err == nil {
			t.Error("expected an error")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "same message") {
			t.Error("expected error to contain 'same message'")
		}
		if !strings.Contains(errStr, "different message") {
			t.Error("expected error to contain 'different message'")
		}
	})

	t.Run("SetWithCustomKeyGetter", func(t *testing.T) {
		set := NewSet()
		customKeyGetter := func(err error) string {
			if erroErr, ok := err.(Error); ok {
				return erroErr.Message() + "_" + erroErr.ID()
			}
			return err.Error()
		}
		set.WithKeyGetter(customKeyGetter)

		// Add errors with same message+ID combination
		err1 := New("test", ID("123"))
		err2 := New("test", ID("123"))
		err3 := New("test", ID("456"))

		set.Add(err1)
		set.Add(err2) // Should be deduplicated
		set.Add(err3)

		if set.Len() != 2 {
			t.Errorf("expected 2 unique errors, got %d", set.Len())
		}
	})

	t.Run("SetWithNilKeyGetter", func(t *testing.T) {
		set := NewSet()
		set.WithKeyGetter(nil) // Should not panic

		err1 := New("test", ID("123"))
		err2 := New("test", ID("123"))

		set.Add(err1)
		set.Add(err2)

		// When nil key getter is set, it doesn't change the key getter,
		// so it keeps the default MessageKeyGetter and deduplicates
		if set.Len() != 1 {
			t.Errorf("expected 1 error with nil key getter (deduplication), got %d", set.Len())
		}
	})
}

func TestSafeSet_WithKeyGetters(t *testing.T) {
	t.Run("SafeSetWithMessageKeyGetter", func(t *testing.T) {
		safeSet := NewSafeSet()
		safeSet.WithKeyGetter(MessageKeyGetter)

		var wg sync.WaitGroup
		// Add 100 errors with 10 unique messages
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				message := fmt.Sprintf("message_%d", i%10)
				safeSet.New(message, ID(fmt.Sprintf("id_%d", i)))
			}(i)
		}
		wg.Wait()

		if safeSet.Len() != 10 {
			t.Errorf("expected 10 unique errors, got %d", safeSet.Len())
		}
	})

	t.Run("SafeSetWithIDKeyGetter", func(t *testing.T) {
		safeSet := NewSafeSet()
		safeSet.WithKeyGetter(IDKeyGetter)

		var wg sync.WaitGroup
		// Add 100 errors with 10 unique IDs
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				id := fmt.Sprintf("id_%d", i%10)
				safeSet.New(fmt.Sprintf("message_%d", i), ID(id))
			}(i)
		}
		wg.Wait()

		if safeSet.Len() != 10 {
			t.Errorf("expected 10 unique errors, got %d", safeSet.Len())
		}
	})

	t.Run("SafeSetWithErrorKeyGetter", func(t *testing.T) {
		safeSet := NewSafeSet()
		safeSet.WithKeyGetter(ErrorKeyGetter)

		var wg sync.WaitGroup
		// Add 100 errors with 10 unique string representations
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				message := fmt.Sprintf("message_%d", i%10)
				id := fmt.Sprintf("id_%d", i%10)
				safeSet.New(message, ID(id))
			}(i)
		}
		wg.Wait()

		if safeSet.Len() != 10 {
			t.Errorf("expected 10 unique errors, got %d", safeSet.Len())
		}
	})
}

func TestDebug_WrappedErrorStrings(t *testing.T) {
	stdErr1 := errors.New("same message")
	stdErr2 := errors.New("same message")

	wrapped1 := Wrap(stdErr1, "wrapped 1")
	wrapped2 := Wrap(stdErr2, "wrapped 2")

	t.Logf("wrapped1.Error(): %s", wrapped1.Error())
	t.Logf("wrapped2.Error(): %s", wrapped2.Error())

	// Test key getters
	t.Logf("MessageKeyGetter(wrapped1): %s", MessageKeyGetter(wrapped1))
	t.Logf("MessageKeyGetter(wrapped2): %s", MessageKeyGetter(wrapped2))
	t.Logf("IDKeyGetter(wrapped1): %s", IDKeyGetter(wrapped1))
	t.Logf("IDKeyGetter(wrapped2): %s", IDKeyGetter(wrapped2))
	t.Logf("ErrorKeyGetter(wrapped1): %s", ErrorKeyGetter(wrapped1))
	t.Logf("ErrorKeyGetter(wrapped2): %s", ErrorKeyGetter(wrapped2))
}

func TestDebug_IDKeyGetter_WrappedErrors(t *testing.T) {
	stdErr1 := errors.New("message 1")
	stdErr2 := errors.New("message 2")
	stdErr3 := errors.New("message 3")

	wrapped1 := Wrap(stdErr1, "same wrap")
	wrapped2 := Wrap(stdErr2, "same wrap")
	wrapped3 := Wrap(stdErr3, "different wrap")

	t.Logf("IDKeyGetter(wrapped1): %s", IDKeyGetter(wrapped1))
	t.Logf("IDKeyGetter(wrapped2): %s", IDKeyGetter(wrapped2))
	t.Logf("IDKeyGetter(wrapped3): %s", IDKeyGetter(wrapped3))
}
