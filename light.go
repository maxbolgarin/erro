package erro

import (
	"context"
	"fmt"
	"time"
	"unsafe"
)

type lightError struct {
	message     string
	cause       error // For wrapping external errors
	fullMessage string

	id        string
	class     Class
	category  Category
	severity  Severity
	retryable bool
	fields    []any

	span Span
}

func NewLight(message string, fields ...any) Error {
	return newLightError(nil, message, fields...)
}

func WrapLight(err error, message string, fields ...any) Error {
	return newLightError(err, message, fields...)
}

func (e *lightError) Error() (res string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe error message
			res = fmt.Sprintf("error formatting failed: %v", r)
		}
		e.fullMessage = res
	}()
	if e.cause == nil && e.severity == "" && len(e.fields) == 0 {
		return e.message
	}

	capacity := len(e.message)

	var errMsg, label string
	if e.cause != nil {
		errMsg = safeErrorString(e.cause)
		capacity += len(errMsg) + 2
	}
	if e.severity != "" {
		label = e.severity.Label()
		capacity += len(label) + 2
	}
	capacity += len(e.fields) * 20

	out := make([]byte, 0, capacity)
	if label != "" {
		out = append(out, label...)
		out = append(out, ' ')
	}
	out = append(out, e.message...)
	for i := 0; i < len(e.fields); i += 2 {
		if i+1 >= len(e.fields) {
			break
		}

		out = append(out, ' ')
		key, ok := e.fields[i].(string)
		if !ok {
			key = valueToString(e.fields[i])
		}
		out = append(out, truncateString(key, maxFieldKeyLength)...)
		out = append(out, '=')
		value, ok := e.fields[i+1].(string)
		if !ok {
			value = valueToString(e.fields[i+1])
		}
		out = append(out, truncateString(value, maxFieldValueLength)...)
	}

	if errMsg != "" {
		out = append(out, ':')
		out = append(out, ' ')
		out = append(out, errMsg...)
	}
	return unsafe.String(unsafe.SliceData(out), len(out))
}

// Unwrap implements the Unwrap interface
func (e *lightError) Unwrap() error {
	return e.cause
}

// Implement Error interface methods (lightweight versions)
func (e *lightError) WithID(idRaw ...string) Error {
	if len(idRaw) > 0 {
		e.id = truncateString(idRaw[0], maxCodeLength)
	} else if e.id == "" {
		e.id = newID(e.class, e.category)
	}
	return e
}

func (e *lightError) WithClass(class Class) Error {
	e.class = class
	return e
}

func (e *lightError) WithCategory(category Category) Error {
	e.category = category
	return e
}

func (e *lightError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.severity = severity
	e.fullMessage = ""
	return e
}

func (e *lightError) WithRetryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *lightError) WithFields(fields ...any) Error {
	e.fields = fields
	e.fullMessage = ""
	return e
}

func (e *lightError) WithSpan(span Span) Error {
	if span == nil {
		return e
	}
	span.SetAttributes(e.fields...)
	span.RecordError(e)
	e.span = span
	return e
}

// Lightweight implementations - these don't do expensive operations
func (e *lightError) RecordMetrics(metrics Metrics) Error {
	if metrics == nil {
		return e
	}
	metrics.RecordError(e)
	return e
}

func (e *lightError) SendEvent(ctx context.Context, dispatcher Dispatcher) Error {
	if dispatcher == nil {
		return e
	}
	dispatcher.SendEvent(ctx, e)
	return e
}

// Getter methods
func (e *lightError) Context() ErrorContext { return e }
func (e *lightError) ID() string {
	if e.id == "" {
		e.id = newID(e.class, e.category)
	}
	return e.id
}
func (e *lightError) Class() Class       { return e.class }
func (e *lightError) Category() Category { return e.category }
func (e *lightError) IsRetryable() bool  { return e.retryable }
func (e *lightError) Span() Span         { return e.span }
func (e *lightError) Fields() []any      { return e.fields }
func (e *lightError) Created() time.Time { return time.Time{} }
func (e *lightError) Message() string    { return e.message }

// Severity checking methods
func (e *lightError) Severity() Severity {
	if e.severity == "" {
		return SeverityUnknown
	}
	return e.severity
}

func (e *lightError) BaseError() ErrorContext {
	if e.cause == nil {
		if causeErro, ok := e.cause.(Error); ok {
			return causeErro.Context().BaseError()
		}
	}
	return e
}

// Stack methods - lightweight errors have no stack traces
func (e *lightError) Stack() Stack {
	return e.toFullError().Stack()
}

func (e *lightError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "%s", e.Error())
}

// Is method optimized for lightweight comparison
func (e *lightError) Is(target error) (ok bool) {
	if target == nil {
		return false
	}

	// Direct equality check
	if e == target {
		return true
	}

	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	// Fast comparison with other erro errors
	if targetErro, ok := target.(ErrorContext); ok {
		// Compare by ID first (fastest)
		if e.id != "" && targetErro.ID() != "" {
			return e.id == targetErro.ID()
		}

		// Compare messages
		if e.message == targetErro.Message() {
			return true
		}

		// Compare class/category
		if e.class != "" && targetErro.Class() != "" && e.class == targetErro.Class() {
			if e.category != "" && targetErro.Category() != "" && e.category == targetErro.Category() {
				return true
			}
		}
	}

	// For external errors
	if e.cause != nil {
		if e.cause == target {
			return true
		}
		if x, ok := e.cause.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}
		return e.cause.Error() == target.Error()
	}

	// Final comparison
	if _, isErro := target.(Error); !isErro {
		return e.message == target.Error()
	}

	return false
}

// toFullError converts a lightError to a full baseError when needed
func (e *lightError) toFullError() ErrorContext {
	fullErr := newBaseErrorWithStackSkip(4, e.cause, e.message)
	fullErr.id = e.id
	fullErr.class = e.class
	fullErr.category = e.category
	fullErr.severity = e.severity
	fullErr.retryable = e.retryable
	fullErr.fields = e.fields
	fullErr.span = e.span
	return fullErr
}

// newLightError creates a new lightweight error
func newLightError(cause error, message string, fields ...any) *lightError {
	return &lightError{
		message: truncateString(message, maxMessageLength),
		cause:   cause,
		fields:  fields,
	}
}

// IsLight checks if any error is a lightweight error
func IsLight(err error) bool {
	_, ok := err.(*lightError)
	return ok
}
