package erro

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error     // Original error if wrapping external error
	message     string    // Base message
	stack       rawStack  // Stack trace (program counters only - resolved on demand)
	createdAt   time.Time // Creation timestamp

	// Metadata
	fields    []any           // Key-value fields
	code      string          // Error code
	category  string          // Error category
	severity  string          // Error severity
	tags      []string        // Tags
	retryable bool            // Retryable flag
	traceID   string          // Trace ID
	ctx       context.Context // Associated context
}

// Error implements the error interface
func (e *baseError) Error() (out string) {
	out = buildFieldsMessage(e.message, e.fields)

	if e.originalErr != nil {
		// If wrapping external error, include it
		return out + ": " + e.originalErr.Error()
	}

	return out
}

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Print with stack trace
			fmt.Fprint(s, e.Error())
			fmt.Fprint(s, "\nStack trace:\n")
			for _, frame := range e.stack.toFrames() {
				fmt.Fprintf(s, "\t%s.%s\n\t\t%s:%d\n", frame.Package, frame.Name, frame.File, frame.Line)
			}
		} else {
			fmt.Fprint(s, e.Error())
		}
	case 's':
		fmt.Fprint(s, e.Error())
	}
}

// Unwrap implements the Unwrap interface
func (e *baseError) Unwrap() error {
	return e.originalErr
}

// Chaining methods for baseError
func (e *baseError) Code(code string) Error {
	e.code = code
	return e
}

func (e *baseError) Category(category string) Error {
	e.category = category
	return e
}

func (e *baseError) Severity(severity string) Error {
	e.severity = severity
	return e
}

func (e *baseError) Fields(fields ...any) Error {
	e.fields = append(e.fields, prepareFields(fields)...)
	return e
}

func (e *baseError) Context(ctx context.Context) Error {
	e.ctx = ctx
	return e
}

func (e *baseError) Tags(tags ...string) Error {
	e.tags = append(e.tags, tags...)
	return e
}

func (e *baseError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) TraceID(traceID string) Error {
	e.traceID = traceID
	return e
}

// Getter methods for baseError
func (e *baseError) GetBase() *baseError         { return e }
func (e *baseError) GetContext() context.Context { return e.ctx }
func (e *baseError) GetCode() string             { return e.code }
func (e *baseError) GetCategory() string         { return e.category }
func (e *baseError) GetSeverity() string         { return e.severity }
func (e *baseError) GetTags() []string           { return e.tags }
func (e *baseError) IsRetryable() bool           { return e.retryable }
func (e *baseError) GetTraceID() string          { return e.traceID }
func (e *baseError) GetFields() []any            { return e.fields }
func (e *baseError) Stack() []StackFrame         { return e.stack.toFrames() }
func (e *baseError) StackFormat() string         { return e.stack.formatFull() }
func (e *baseError) ErrorWithStack() string {
	return e.Error() + "\n" + e.StackFormat()
}

// newBaseError creates a new base error
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return &baseError{
		originalErr: originalErr,
		message:     message,
		stack:       captureStack(3), // Skip New, newBaseError and caller
		createdAt:   time.Now(),
		fields:      prepareFields(fields),
	}
}

// buildFieldsMessage creates message with fields appended
func buildFieldsMessage(message string, fields []any) string {
	if len(fields) == 0 {
		return message
	}

	var builder strings.Builder

	// Estimate capacity: message + fields with reasonable estimates for key=value pairs
	// Each field pair needs: space + key + "=" + value (estimate ~20 chars per pair)
	estimatedSize := len(message) + (len(fields)/2)*20
	builder.Grow(estimatedSize)

	builder.WriteString(message)

	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}
		builder.WriteString(" ")
		builder.WriteString(valueToString(fields[i]))
		builder.WriteString("=")
		builder.WriteString(valueToString(fields[i+1]))
	}

	return builder.String()
}

// prepareFields ensures fields come in key-value pairs
func prepareFields(fields []any) []any {
	if len(fields)%2 != 0 {
		return append(fields, "<missing>")
	}
	return fields
}

// countFormatVerbs counts the number of format verbs in a format string
func countFormatVerbs(format string) int {
	count := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			if i+1 < len(format) && format[i+1] != '%' {
				count++
				// Skip the verb character
				i++
			} else if i+1 < len(format) && format[i+1] == '%' {
				// Skip escaped %
				i++
			}
		}
	}
	return count
}
