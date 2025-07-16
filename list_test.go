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
	if safeList.Len() != 99 {
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
