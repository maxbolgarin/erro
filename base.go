package erro

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error     // Original error if wrapping external error
	message     string    // Base message
	created     time.Time // Creation timestamp

	// Metadata
	id        string          // Error id
	fields    []any           // Key-value fields
	category  Category        // Error category
	class     Class           // Error class
	severity  Severity        // Error severity
	retryable bool            // Retryable flag
	ctx       context.Context // Associated context

	stack  rawStack // Stack trace (program counters only - resolved on demand)
	frames Stack    // Stack trace frames (for caching)

	span Span // Span
}

// Error implements the error interface
func (e *baseError) Error() (out string) {
	out = buildFieldsMessage(e.message, e.fields)
	// Always show severity label for base errors
	if e.severity != "" {
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

// errorWithoutSeverity returns the error message without severity label
func (e *baseError) errorWithoutSeverity() string {
	out := buildFieldsMessage(e.message, e.fields)

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
func (e *baseError) ID(idRaw ...string) Error {
	var id string
	if len(idRaw) > 0 {
		id = truncateString(idRaw[0], maxCodeLength)
	} else {
		id = newID(e.class, e.category, e.created)
	}
	e.id = id
	return e
}

func (e *baseError) Category(category Category) Error {
	e.category = category
	return e
}

func (e *baseError) Class(class Class) Error {
	e.class = class
	return e
}

func (e *baseError) Severity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
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

func (e *baseError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) Span(span Span) Error {
	span.SetAttributes(e.fields...)
	span.RecordError(e)
	e.span = span
	return e
}

// Getter methods for baseError
func (e *baseError) GetBase() Error              { return e }
func (e *baseError) GetContext() context.Context { return e.ctx }
func (e *baseError) GetID() string {
	if e.id == "" {
		e.id = newID(e.class, e.category, e.created)
	}
	return e.id
}
func (e *baseError) GetCategory() Category { return e.category }
func (e *baseError) GetClass() Class       { return e.class }
func (e *baseError) IsRetryable() bool     { return e.retryable }
func (e *baseError) GetSpan() Span         { return e.span }
func (e *baseError) GetFields() []any      { return e.fields }
func (e *baseError) GetCreated() time.Time { return e.created }
func (e *baseError) GetMessage() string    { return e.message }

// Severity checking methods
func (e *baseError) GetSeverity() Severity {
	if e.severity == "" {
		return SeverityUnknown
	}
	return e.severity
}
func (e *baseError) IsCritical() bool { return e.severity == SeverityCritical }
func (e *baseError) IsHigh() bool     { return e.severity == SeverityHigh }
func (e *baseError) IsMedium() bool   { return e.severity == SeverityMedium }
func (e *baseError) IsLow() bool      { return e.severity == SeverityLow }
func (e *baseError) IsInfo() bool     { return e.severity == SeverityInfo }
func (e *baseError) IsUnknown() bool {
	return e.severity == "" || e.severity == SeverityUnknown
}

func (e *baseError) Stack() Stack {
	if e.frames == nil {
		e.frames = e.stack.toFrames()
	}
	return e.frames
}
func (e *baseError) StackFormat() string {
	if e.frames == nil {
		e.frames = e.stack.toFrames()
	}
	return e.frames.FormatFull()
}
func (e *baseError) StackWithError() string {
	if e.frames == nil {
		e.frames = e.stack.toFrames()
	}
	return e.Error() + "\n" + e.frames.FormatFull()
}

// Is checks if this error matches the target error
func (e *baseError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check direct equality
	if e == target {
		return true
	}

	// Check if target is an erro error
	if targetErro, ok := target.(Error); ok {
		// Compare by id if both have non-empty ids
		if e.id != "" && targetErro.GetID() != "" && e.id == targetErro.GetID() {
			return true
		}
		// Compare base messages (without fields) for erro errors
		return e.message == targetErro.GetMessage()
	}

	// For external errors, check if we wrap it first
	if e.originalErr != nil {
		// Check if the wrapped error matches directly
		if e.originalErr == target {
			return true
		}
		// Check if the wrapped error has an Is method and use it
		if x, ok := e.originalErr.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}
	}

	// Final fallback: compare error strings for external errors
	return e.Error() == target.Error()
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
		if utf8.RuneCountInString(key) > maxFieldKeyLength {
			runes := []rune(key)
			key = string(runes[:maxFieldKeyLength])
		}
		builder.WriteString(key)

		builder.WriteString("=")
		value := valueToString(fields[i+1])
		if utf8.RuneCountInString(value) > maxFieldValueLength {
			runes := []rune(value)
			value = string(runes[:maxFieldValueLength])
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

			config := GetStackTraceConfig()
			if !config.Enabled {
				return // No stack trace in disabled mode
			}
			fmt.Fprint(s, "\nStack trace:\n")
			fmt.Fprint(s, err.Stack().FormatFull())

		} else {
			fmt.Fprint(s, err.Error())
		}
	case 's':
		fmt.Fprint(s, err.Error())
	}
}

func newID(class Class, category Category, created time.Time) string {
	if class == "" || len(class) < 2 {
		class = "XX"
	}
	if category == "" || len(category) < 2 {
		category = "XX"
	}
	if created.IsZero() {
		created = time.Now()
	}
	classStr := strings.ToUpper(string(class[:2]))
	categoryStr := strings.ToUpper(string(category[:2]))

	timestampStr := strconv.FormatInt(created.UnixMicro(), 10)
	return classStr + categoryStr + "-" + timestampStr[len(timestampStr)-4:]
}
