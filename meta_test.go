package erro_test

import (
	"strings"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestMetaOptions(t *testing.T) {
	formatter := func(err erro.Error) string {
		return "custom format"
	}

	err := erro.New("test message",
		erro.ID("custom_id"),
		erro.Retryable(),
		erro.Fields("key1", "value1"),
		erro.Formatter(formatter),
		erro.StackTrace(),
	)

	if err.ID() != "custom_id" {
		t.Errorf("Expected ID 'custom_id', got '%s'", err.ID())
	}

	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}

	fields := err.Fields()
	if len(fields) != 2 || fields[0] != "key1" || fields[1] != "value1" {
		t.Errorf("Unexpected fields: %v", fields)
	}

	if err.Error() != "custom format" {
		t.Errorf("Expected custom format, got '%s'", err.Error())
	}

	stack := err.Stack()
	if len(stack) == 0 {
		t.Errorf("Expected stack to be captured")
	}
}

func TestStackTraceWithSkip(t *testing.T) {
	err := erro.New("test error", erro.StackTraceWithSkip(1))
	stack := err.Stack()
	if len(stack) == 0 {
		t.Fatal("Expected stack to be captured")
	}
	// The top frame should be the testing framework, not the erro package
	topFrame := stack[0]
	if strings.HasPrefix(topFrame.FullName, "github.com/maxbolgarin/erro.") {
		t.Errorf("Expected top frame to be outside the erro package, but got %s", topFrame.FullName)
	}
}
