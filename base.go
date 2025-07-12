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
	created     time.Time // Creation timestamp

	// Metadata
	fields    []any           // Key-value fields
	code      string          // Error code
	category  string          // Error category
	severity  ErrorSeverity   // Error severity
	tags      []string        // Tags
	retryable bool            // Retryable flag
	traceID   string          // Trace ID
	ctx       context.Context // Associated context

	depth int // Tracks wrapping depth to prevent stack overflow
}

// Error implements the error interface
func (e *baseError) Error() (out string) {
	out = buildFieldsMessage(e.message, e.fields)
	if e.depth == 0 && e.severity != "" {
		out = e.severity.Label() + " " + out
	}

	if e.originalErr != nil {
		if out == "" {
			return e.originalErr.Error()
		}
		return out + ": " + e.originalErr.Error()
	}
	return out
}

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *baseError) Unwrap() error {
	return e.originalErr
}

// Chaining methods for baseError
func (e *baseError) Code(code string) Error {
	e.code = truncateString(code, maxCodeLength)
	return e
}

func (e *baseError) Category(category string) Error {
	e.category = truncateString(category, maxCategoryLength)
	return e
}

func (e *baseError) Severity(severity ErrorSeverity) Error {
	if !severity.IsValid() {
		severity = Unknown
	}
	e.severity = severity
	return e
}

func (e *baseError) Fields(fields ...any) Error {
	e.fields = safeAppendFields(e.fields, prepareFields(fields))
	return e
}

func (e *baseError) Context(ctx context.Context) Error {
	e.ctx = ctx
	return e
}

func (e *baseError) Tags(tags ...string) Error {
	e.tags = safeAppendFields(e.tags, tags)
	return e
}

func (e *baseError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) TraceID(traceID string) Error {
	e.traceID = truncateString(traceID, maxTraceIDLength)
	return e
}

// Getter methods for baseError
func (e *baseError) GetBase() Error              { return e }
func (e *baseError) GetContext() context.Context { return e.ctx }
func (e *baseError) GetCode() string             { return e.code }
func (e *baseError) GetCategory() string         { return e.category }
func (e *baseError) GetTags() []string           { return e.tags }
func (e *baseError) IsRetryable() bool           { return e.retryable }
func (e *baseError) GetTraceID() string          { return e.traceID }
func (e *baseError) GetFields() []any            { return e.fields }
func (e *baseError) GetCreated() time.Time       { return e.created }

// Severity checking methods
func (e *baseError) GetSeverity() ErrorSeverity {
	if e.severity == "" {
		return Unknown
	}
	return e.severity
}
func (e *baseError) IsCritical() bool { return e.severity == Critical }
func (e *baseError) IsHigh() bool     { return e.severity == High }
func (e *baseError) IsMedium() bool   { return e.severity == Medium }
func (e *baseError) IsLow() bool      { return e.severity == Low }
func (e *baseError) IsInfo() bool     { return e.severity == Info }
func (e *baseError) IsUnknown() bool {
	return e.severity == "" || e.severity == Unknown
}

func (e *baseError) Stack() Stack           { return e.stack.toFrames() }
func (e *baseError) StackFormat() string    { return e.stack.formatFull() }
func (e *baseError) StackWithError() string { return e.Error() + "\n" + e.StackFormat() }

// Is checks if this error matches the target error
func (e *baseError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check direct equality
	if e == target {
		return true
	}

	// Check if target is an erro error and compare by code and message
	if targetErro, ok := target.(Error); ok {
		// Compare by code if both have codes
		if e.code != "" && targetErro.GetCode() != "" {
			return e.code == targetErro.GetCode()
		}
		// Compare by message if no codes
		return e.message == targetErro.Error()
	}

	// For external errors, compare by message or check if we wrap it
	if e.originalErr != nil {
		// Check if the wrapped error matches
		if e.originalErr == target {
			return true
		}
		// Check if the wrapped error is also a complex error that might match
		if x, ok := e.originalErr.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}
	}

	// Final fallback: compare by message
	return e.message == target.Error()
}

// newBaseError creates a new base error with security validation
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return &baseError{
		originalErr: originalErr,
		message:     truncateString(message, maxMessageLength),
		stack:       captureStack(3), // Skip New, newBaseError and caller
		created:     time.Now(),
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
		key := valueToString(fields[i])
		if len(key) > maxFieldKeyLength {
			key = key[:maxFieldKeyLength]
		}
		builder.WriteString(key)
		builder.WriteString("=")
		value := valueToString(fields[i+1])
		if len(value) > maxFieldValueLength {
			value = value[:maxFieldValueLength]
		}
		builder.WriteString(value)
	}

	return builder.String()
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

func formatError(err Error, s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Print with stack trace
			fmt.Fprint(s, err.Error())
			fmt.Fprint(s, "\nStack trace:\n")
			for _, frame := range err.Stack() {
				fmt.Fprintf(s, "\t%s.%s\n\t\t%s:%d\n", frame.Package, frame.Name, frame.File, frame.Line)
			}
		} else {
			fmt.Fprint(s, err.Error())
		}
	case 's':
		fmt.Fprint(s, err.Error())
	}
}
