package erro

import (
	"context"
	"fmt"
	"time"
)

// wrapError is a lightweight wrapper that points to baseError
type wrapError struct {
	base        *baseError // Pointer to base error
	wrapped     Error      // The wrapped error (to get all its fields)
	wrapMessage string     // Wrap message
	wrapFields  []any      // Additional fields for this wrap
	createdAt   time.Time  // When this wrap was created
}

func (e *wrapError) Error() (out string) {
	out = buildFieldsMessage(e.wrapMessage, e.wrapFields)

	// Build the complete chain by getting the wrapped error's message
	var wrappedMsg string
	if e.wrapped != nil {
		wrappedMsg = e.wrapped.Error()
	} else {
		wrappedMsg = e.base.Error()
	}

	if out == "" {
		return wrappedMsg
	}

	if e.base.severity != "" {
		out = e.base.severity.Label() + " " + out
	}

	return out + ": " + wrappedMsg
}

func (e *wrapError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *wrapError) Unwrap() error {
	return e.wrapped
}

// Chaining methods for wrapError - these modify the base error
func (e *wrapError) Code(code string) Error {
	e.base.code = truncateString(code, maxCodeLength)
	return e
}

func (e *wrapError) Category(category string) Error {
	e.base.category = truncateString(category, maxCategoryLength)
	return e
}

func (e *wrapError) Severity(severity ErrorSeverity) Error {
	if !severity.IsValid() {
		severity = Unknown
	}
	e.base.severity = severity
	return e
}

func (e *wrapError) Fields(fields ...any) Error {
	e.wrapFields = safeAppendFields(e.wrapFields, prepareFields(fields))
	return e
}

func (e *wrapError) Context(ctx context.Context) Error {
	e.base.ctx = ctx
	return e
}

func (e *wrapError) Tags(tags ...string) Error {
	e.base.tags = safeAppendFields(e.base.tags, tags)
	return e
}

func (e *wrapError) Retryable(retryable bool) Error {
	e.base.retryable = retryable
	return e
}

func (e *wrapError) TraceID(traceID string) Error {
	e.base.traceID = truncateString(traceID, maxTraceIDLength)
	return e
}

// Getter methods for wrapError
func (e *wrapError) GetBase() Error              { return e.base }
func (e *wrapError) GetContext() context.Context { return e.base.ctx }
func (e *wrapError) GetCode() string             { return e.base.code }
func (e *wrapError) GetCategory() string         { return e.base.category }
func (e *wrapError) GetTags() []string           { return e.base.tags }
func (e *wrapError) IsRetryable() bool           { return e.base.retryable }
func (e *wrapError) GetTraceID() string          { return e.base.traceID }
func (e *wrapError) GetCreated() time.Time       { return e.base.created }

// Severity checking methods
func (e *wrapError) GetSeverity() ErrorSeverity { return e.base.GetSeverity() }
func (e *wrapError) IsCritical() bool           { return e.base.IsCritical() }
func (e *wrapError) IsHigh() bool               { return e.base.IsHigh() }
func (e *wrapError) IsMedium() bool             { return e.base.IsMedium() }
func (e *wrapError) IsLow() bool                { return e.base.IsLow() }
func (e *wrapError) IsInfo() bool               { return e.base.IsInfo() }
func (e *wrapError) IsUnknown() bool {
	return e.base.IsUnknown()
}
func (e *wrapError) Stack() Stack           { return e.base.stack.toFrames() }
func (e *wrapError) StackFormat() string    { return e.base.stack.formatFull() }
func (e *wrapError) StackWithError() string { return e.Error() + "\n" + e.StackFormat() }

// Is checks if this error or any wrapped error matches the target
func (e *wrapError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check direct equality
	if e == target {
		return true
	}

	// Check if the wrapped error matches directly (most common case)
	if e.wrapped != nil {
		if e.wrapped == target {
			return true
		}
		// Use the wrapped error's Is method if it has one
		if e.wrapped.Is(target) {
			return true
		}
	}

	// Check if the base error matches
	if e.base != nil && e.base.Is(target) {
		return true
	}

	return false
}

func (e *wrapError) GetFields() []any {
	// Add fields from wrapped error (if it exists)
	var wrappedFields []any
	if e.wrapped != nil {
		wrappedFields = e.wrapped.GetFields()
	} else {
		wrappedFields = e.base.fields
	}

	// Create with exact capacity to avoid reallocations
	allFields := make([]any, len(e.wrapFields)+len(wrappedFields))
	copy(allFields, e.wrapFields)
	copy(allFields[len(e.wrapFields):], wrappedFields)

	return allFields
}

func newWrapError(wrapped Error, message string, fields ...any) Error {
	if wrapped == nil {
		wrapped = New(message, fields...)
	}

	var depth int

	baseInt := wrapped.GetBase()
	base, ok := baseInt.(*baseError)
	if ok {
		depth = base.depth
		base.depth++
	}

	if depth > maxWrapDepth {
		return Wrap(ErrMaxWrapDepthExceeded, message, fields...)
	}

	return &wrapError{
		base:        base,
		wrapped:     wrapped,
		wrapMessage: truncateString(message, maxMessageLength),
		wrapFields:  prepareFields(fields),
		createdAt:   time.Now(),
	}
}
