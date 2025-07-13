package erro

import (
	"context"
	"time"
	"unsafe"
)

type lightError struct {
	message     string
	id          string
	class       Class
	category    Category
	severity    Severity
	retryable   bool
	ctx         context.Context
	span        Span
	fields      []any
	fullMessage string
	cause       error // For wrapping external errors
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
		e.fullMessage = res
	}()
	if e.cause == nil && e.severity == "" && len(e.fields) == 0 {
		return e.message
	}

	capacity := len(e.message)

	var errMsg, label string
	if e.cause != nil {
		errMsg = e.cause.Error()
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
func (e *lightError) ID(idRaw ...string) Error {
	if len(idRaw) > 0 {
		e.id = truncateString(idRaw[0], maxCodeLength)
	} else if e.id == "" {
		e.id = newID(e.class, e.category)
	}
	return e
}

func (e *lightError) Class(class Class) Error {
	e.class = class
	return e
}

func (e *lightError) Category(category Category) Error {
	e.category = category
	return e
}

func (e *lightError) Severity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.severity = severity
	e.fullMessage = ""
	return e
}

func (e *lightError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *lightError) Context(ctx context.Context) Error {
	e.ctx = ctx
	return e
}

func (e *lightError) Span(span Span) Error {
	e.span = span
	return e
}

// Lightweight implementations - these don't do expensive operations
func (e *lightError) Fields(fields ...any) Error {
	e.fields = fields
	e.fullMessage = ""
	return e
}

// Getter methods
func (e *lightError) GetBase() Error              { return e }
func (e *lightError) GetContext() context.Context { return nil }
func (e *lightError) GetID() string {
	if e.id == "" {
		e.id = newID(e.class, e.category)
	}
	return e.id
}
func (e *lightError) GetCategory() Category { return e.category }
func (e *lightError) GetClass() Class       { return e.class }
func (e *lightError) IsRetryable() bool     { return e.retryable }
func (e *lightError) GetSpan() Span         { return nil }
func (e *lightError) GetFields() []any      { return nil }
func (e *lightError) GetCreated() time.Time { return time.Time{} }
func (e *lightError) GetMessage() string    { return e.message }

// Severity checking methods
func (e *lightError) GetSeverity() Severity {
	if e.severity == "" {
		return SeverityUnknown
	}
	return e.severity
}
func (e *lightError) IsCritical() bool { return e.severity == SeverityCritical }
func (e *lightError) IsHigh() bool     { return e.severity == SeverityHigh }
func (e *lightError) IsMedium() bool   { return e.severity == SeverityMedium }
func (e *lightError) IsLow() bool      { return e.severity == SeverityLow }
func (e *lightError) IsInfo() bool     { return e.severity == SeverityInfo }
func (e *lightError) IsUnknown() bool {
	return e.severity == "" || e.severity == SeverityUnknown
}

// Stack methods - lightweight errors have no stack traces
func (e *lightError) Stack() Stack {
	return e.toFullError().Stack()
}
func (e *lightError) StackFormat() string {
	return e.toFullError().StackFormat()
}
func (e *lightError) StackWithError() string {
	return e.toFullError().StackWithError()
}

// Is method optimized for lightweight comparison
func (e *lightError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Direct equality check
	if e == target {
		return true
	}

	// Fast comparison with other erro errors
	if targetErro, ok := target.(Error); ok {
		// Compare by ID first (fastest)
		if e.id != "" && targetErro.GetID() != "" {
			return e.id == targetErro.GetID()
		}

		// Compare messages
		if e.message == targetErro.GetMessage() {
			return true
		}

		// Compare class/category
		if e.class != "" && targetErro.GetClass() != "" && e.class == targetErro.GetClass() {
			if e.category != "" && targetErro.GetCategory() != "" && e.category == targetErro.GetCategory() {
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
func (e *lightError) toFullError() Error {
	fullErr := newBaseErrorWithStackSkip(4, e.cause, e.message)
	fullErr.id = e.id
	fullErr.class = e.class
	fullErr.category = e.category
	fullErr.severity = e.severity
	fullErr.retryable = e.retryable
	fullErr.fields = e.fields
	fullErr.ctx = e.ctx
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
