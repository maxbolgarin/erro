package erro_test

import (
	"fmt"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestStackFrameFiltering(t *testing.T) {
	// Create an error to test stack frame filtering
	err := erro.New("test error")

	// Get the stack frames
	frames := err.Context().Stack()

	// Print each frame for manual inspection
	fmt.Println("Stack frames (should NOT contain runtime.main or runtime.goexit):")
	for i, frame := range frames {
		fmt.Printf("%d: %s (%s:%d)\n", i, frame.Name, frame.FileName, frame.Line)
	}

	// Verify no useless runtime frames
	for _, frame := range frames {
		if frame.FullName == "runtime.main" {
			t.Errorf("Found runtime.main frame that should be filtered: %s", frame.FullName)
		}
		if frame.FullName == "runtime.goexit" {
			t.Errorf("Found runtime.goexit frame that should be filtered: %s", frame.FullName)
		}
		if frame.Name == "goexit" {
			t.Errorf("Found goexit frame that should be filtered: %s", frame.FullName)
		}
	}
}

func helperFunction() erro.Error {
	return anotherHelperFunction()
}

func anotherHelperFunction() erro.Error {
	return erro.New("deep stack error", "level", "deep")
}

func TestStackFrameFilteringWithDeepStack(t *testing.T) {
	// Create error through helper functions to get a deeper stack
	err := helperFunction()

	frames := err.Context().Stack()

	fmt.Println("\nDeep stack frames (should show user functions only):")
	for i, frame := range frames {
		fmt.Printf("%d: %s (%s:%d)\n", i, frame.Name, frame.FileName, frame.Line)
	}

	// Should contain our helper functions
	found := false
	for _, frame := range frames {
		if frame.Name == "anotherHelperFunction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should contain anotherHelperFunction in stack trace")
	}

	// Should NOT contain runtime noise
	for _, frame := range frames {
		if frame.FullName == "runtime.main" || frame.FullName == "runtime.goexit" {
			t.Errorf("Found filtered runtime frame that should not be present: %s", frame.FullName)
		}
	}
}
