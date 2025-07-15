package erro_test

import (
	"strings"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestStackTraceConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *erro.StackTraceConfig
		assert func(*testing.T, erro.Stack)
	}{
		{
			name:   "Development",
			config: erro.DevelopmentStackTraceConfig(),
			assert: func(t *testing.T, stack erro.Stack) {
				if len(stack) == 0 {
					t.Fatal("stack is empty")
				}
				frame := stack[0]
				if !strings.Contains(frame.FormatFull(), "stack_test.go") {
					t.Errorf("Expected full path in dev config, got: %s", frame.FormatFull())
				}
			},
		},
		{
			name:   "Production",
			config: erro.ProductionStackTraceConfig(),
			assert: func(t *testing.T, stack erro.Stack) {
				if len(stack) == 0 {
					t.Fatal("stack is empty")
				}
				frame := stack[0]
				if strings.Contains(frame.String(), "/") {
					t.Errorf("Expected no full path in prod config, got: %s", frame.String())
				}
			},
		},
		{
			name:   "Strict",
			config: erro.StrictStackTraceConfig(),
			assert: func(t *testing.T, stack erro.Stack) {
				if len(stack) == 0 {
					t.Fatal("stack is empty")
				}
				frame := stack[0]
				if !strings.Contains(frame.String(), "[some_function]") {
					t.Errorf("Expected redacted function name in strict config, got: %s", frame.String())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := erro.New("test error", erro.StackTraceWithSkip(5, tc.config))
			stack := err.Stack()
			tc.assert(t, stack)
		})
	}
}
