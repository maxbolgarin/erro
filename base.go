package erro

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error  // Original error if wrapping external error
	message     string // Base message
	fullMessage string // Full message with fields (caching)

	// Metadata
	id        string    // Error id
	class     Class     // Error class
	category  Category  // Error category
	severity  Severity  // Error severity
	retryable bool      // Retryable flag
	fields    []any     // Key-value fields
	span      Span      // Span
	created   time.Time // Creation timestamp

	stackOnce sync.Once
	stack     rawStack // Stack trace (program counters only - resolved on demand)
	frames    Stack    // Stack trace frames (for caching)
}

// Error implements the error interface
func (e *baseError) Error() string {
	return e.errorString()
}

// errorWithoutSeverity returns the error message without severity label
func (e *baseError) errorString(ignoreSeverity ...bool) (out string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		// Cache the full message
		e.fullMessage = out
	}()

	out = buildFieldsMessage(e.message, e.fields)
	if (len(ignoreSeverity) == 0 || !ignoreSeverity[0]) && e.severity != SeverityUnknown {
		out = e.severity.Label() + " " + out
	}

	if e.originalErr != nil {
		if out == "" {
			return safeErrorString(e.originalErr)
		}
		return out + ": " + safeErrorString(e.originalErr)
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
func (e *baseError) WithID(idRaw ...string) Error {
	var id string
	if len(idRaw) > 0 {
		id = truncateString(idRaw[0], maxCodeLength)
	} else {
		id = newID(e.class, e.category, e.created)
	}
	e.id = id
	return e
}

func (e *baseError) WithCategory(category Category) Error {
	e.category = category
	return e
}

func (e *baseError) WithClass(class Class) Error {
	e.class = class
	return e
}

func (e *baseError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.severity = severity
	e.fullMessage = ""
	return e
}

func (e *baseError) WithFields(fields ...any) Error {
	e.fields = safeAppendFields(e.fields, prepareFields(fields))
	e.fullMessage = ""
	return e
}

func (e *baseError) WithRetryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) WithSpan(span Span) Error {
	if span == nil {
		return e
	}
	span.SetAttributes(e.fields...)
	span.RecordError(e)
	e.span = span
	return e
}

func (e *baseError) RecordMetrics(metrics Metrics) Error {
	if metrics == nil {
		return e
	}
	metrics.RecordError(e)
	return e
}

func (e *baseError) SendEvent(ctx context.Context, dispatcher Dispatcher) Error {
	if dispatcher == nil {
		return e
	}
	dispatcher.SendEvent(ctx, e)
	return e
}

// Getter methods for baseError
func (e *baseError) Context() ErrorContext { return e }
func (e *baseError) ID() string {
	if e.id == "" {
		e.id = newID(e.class, e.category, e.created)
	}
	return e.id
}
func (e *baseError) Class() Class       { return e.class }
func (e *baseError) Category() Category { return e.category }
func (e *baseError) IsRetryable() bool  { return e.retryable }
func (e *baseError) Span() Span         { return e.span }
func (e *baseError) Fields() []any      { return e.fields }
func (e *baseError) Created() time.Time { return e.created }
func (e *baseError) Message() string    { return e.message }
func (e *baseError) Severity() Severity { return e.severity }

func (e *baseError) Stack() Stack {
	e.stackOnce.Do(func() {
		e.frames = e.stack.toFrames()
	})
	return e.frames
}
func (e *baseError) BaseError() ErrorContext { return e }

// Is checks if this error matches the target error
func (e *baseError) Is(target error) (ok bool) {
	if target == nil {
		return false
	}

	// Check direct equality (fastest path)
	if e == target {
		return true
	}

	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	// Fast path for erro errors - compare by metadata first
	if targetErro, ok := target.(ErrorContext); ok {
		// Compare by id if both have non-empty ids (very fast)
		if e.id != "" && targetErro.ID() != "" {
			return e.id == targetErro.ID()
		}

		// Compare base messages without fields (fast)
		if e.message == targetErro.Message() {
			return true
		}
	}

	// For external errors, check if we wrap it directly
	if e.originalErr != nil {
		// Direct reference comparison (very fast)
		if e.originalErr == target {
			return true
		}

		// If the wrapped error has an Is method, use it
		if x, ok := e.originalErr.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}

		// For external errors, compare the original error's string representation
		// This avoids building our full error string with fields
		return e.originalErr.Error() == target.Error()
	}

	// Last resort: only for comparison with external errors without originalErr
	// Try to be smart about when to do expensive string comparison
	if _, isErro := target.(Error); !isErro {
		// Target is external error, we are baseError - only compare if we have no fields
		if len(e.fields) == 0 && e.severity == "" {
			// No fields, safe to compare messages
			return e.message == target.Error()
		}
		// If we have fields, this comparison is likely wrong anyway since
		// external errors won't match our formatted string with fields
		return false
	}

	// Both are erro errors but didn't match on any fast path
	// This should be rare with good error design
	return false
}

// newBaseError creates a new base error with security validation
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return newBaseErrorWithStackSkip(3, originalErr, message, fields...)
}

func newBaseErrorWithStackSkip(skip int, originalErr error, message string, fields ...any) *baseError {
	e := &baseError{
		originalErr: originalErr,
		message:     truncateString(message, maxMessageLength),
		created:     time.Now(),
		fields:      prepareFields(fields),
		stack:       captureStack(skip),
	}
	return e
}
