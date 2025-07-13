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
	createdAt   time.Time  // When this wrap was created (caching)
	wrapDepth   int        // Depth of this specific wrap (without mutating base) (caching)

	fullMessage string // Full message with fields (caching)
}

func (e *wrapError) Error() (out string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		e.fullMessage = out
	}()

	out = buildFieldsMessage(e.wrapMessage, e.wrapFields)

	// Build the complete chain by getting the wrapped error's message
	var wrappedMsg string
	if e.wrapped != nil {
		// For wrapped errors, get the error message without severity label to avoid duplication
		if baseErr, ok := e.wrapped.(*baseError); ok {
			wrappedMsg = baseErr.errorWithoutSeverity()
		} else if wrapErr, ok := e.wrapped.(*wrapError); ok {
			wrappedMsg = wrapErr.errorWithoutSeverity()
		} else {
			wrappedMsg = e.wrapped.Error()
		}
	} else {
		wrappedMsg = e.base.errorWithoutSeverity()
	}

	if out == "" {
		return wrappedMsg
	}

	if e.base.severity != "" {
		out = e.base.severity.Label() + " " + out
	}

	return out + ": " + wrappedMsg
}

// errorWithoutSeverity returns the error message without severity label
func (e *wrapError) errorWithoutSeverity() (out string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		e.fullMessage = out
	}()

	out = buildFieldsMessage(e.wrapMessage, e.wrapFields)

	// Build the complete chain by getting the wrapped error's message without severity
	var wrappedMsg string
	if e.wrapped != nil {
		if baseErr, ok := e.wrapped.(*baseError); ok {
			wrappedMsg = baseErr.errorWithoutSeverity()
		} else if wrapErr, ok := e.wrapped.(*wrapError); ok {
			wrappedMsg = wrapErr.errorWithoutSeverity()
		} else {
			wrappedMsg = e.wrapped.Error()
		}
	} else {
		wrappedMsg = e.base.errorWithoutSeverity()
	}

	if out == "" {
		return wrappedMsg
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
func (e *wrapError) ID(idRaw ...string) Error {
	var id string
	if len(idRaw) > 0 {
		id = truncateString(idRaw[0], maxCodeLength)
	} else {
		id = e.base.GetID()
	}
	e.base.id = id
	return e
}

func (e *wrapError) Category(category Category) Error {
	e.base.category = category
	return e
}

func (e *wrapError) Class(class Class) Error {
	e.base.class = class
	return e
}

func (e *wrapError) Severity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.base.severity = severity
	e.fullMessage = ""
	return e
}

func (e *wrapError) Fields(fields ...any) Error {
	e.wrapFields = safeAppendFields(e.wrapFields, prepareFields(fields))
	if e.base.span != nil {
		e.base.span.SetAttributes(e.wrapFields...)
	}
	e.fullMessage = ""
	return e
}

func (e *wrapError) Context(ctx context.Context) Error {
	e.base.ctx = ctx
	return e
}

func (e *wrapError) Retryable(retryable bool) Error {
	e.base.retryable = retryable
	return e
}

func (e *wrapError) Span(span Span) Error {
	span.SetAttributes(e.base.fields...)
	span.SetAttributes(e.wrapFields...)
	span.RecordError(e)
	e.base.span = span
	return e
}

func (e *wrapError) RecordMetrics(metrics Metrics) Error {
	metrics.RecordError(e)
	return e
}

// Getter methods for wrapError
func (e *wrapError) GetBase() Error              { return e.base }
func (e *wrapError) GetContext() context.Context { return e.base.ctx }
func (e *wrapError) GetID() string               { return e.base.GetID() }
func (e *wrapError) GetCategory() Category       { return e.base.category }
func (e *wrapError) GetClass() Class             { return e.base.class }
func (e *wrapError) IsRetryable() bool           { return e.base.retryable }
func (e *wrapError) GetSpan() Span               { return e.base.span }
func (e *wrapError) GetCreated() time.Time       { return e.base.created }
func (e *wrapError) GetMessage() string          { return e.base.message }

// Severity checking methods
func (e *wrapError) GetSeverity() Severity { return e.base.GetSeverity() }
func (e *wrapError) IsCritical() bool      { return e.base.IsCritical() }
func (e *wrapError) IsHigh() bool          { return e.base.IsHigh() }
func (e *wrapError) IsMedium() bool        { return e.base.IsMedium() }
func (e *wrapError) IsLow() bool           { return e.base.IsLow() }
func (e *wrapError) IsInfo() bool          { return e.base.IsInfo() }
func (e *wrapError) IsUnknown() bool {
	return e.base.IsUnknown()
}
func (e *wrapError) Stack() Stack {
	if e.base.initStack() {
		e.fullMessage = ""
	}
	return e.base.frames
}
func (e *wrapError) StackFormat() string {
	if e.base.initStack() {
		e.fullMessage = ""
	}
	return e.base.frames.FormatFull()
}
func (e *wrapError) StackWithError() string {
	if e.base.initStack() {
		e.fullMessage = ""
	}
	return e.Error() + "\n" + e.base.frames.FormatFull()
}

// Is checks if this error or any wrapped error matches the target
func (e *wrapError) Is(target error) bool {
	return e.isWithVisited(target, make(map[*baseError]bool))
}

// isWithVisited implements Is with cycle detection
func (e *wrapError) isWithVisited(target error, visited map[*baseError]bool) bool {
	if target == nil {
		return false
	}

	// Check direct equality first (fastest path)
	if e == target {
		return true
	}

	// Cycle detection: if we've already visited this base error in this path,
	// skip this branch to avoid infinite recursion
	if e.base != nil && visited[e.base] {
		return false // Skip this branch, not an error condition
	}

	// Mark this base as visited for cycle detection
	if e.base != nil {
		visited[e.base] = true
		// Clean up after this branch to allow other paths to visit the same base
		defer func() { delete(visited, e.base) }()
	}

	// Fast path: Check if target is an erro error and use optimized comparison
	if targetErro, ok := target.(Error); ok {
		// Compare by id if both have non-empty ids (very fast)
		if e.base != nil && e.base.id != "" && targetErro.GetID() != "" {
			if e.base.id == targetErro.GetID() {
				return true
			}
		}

		// Compare base messages (fast)
		if e.base != nil && e.base.message == targetErro.GetMessage() {
			return true
		}
	}

	// Check if the wrapped error matches directly (most common case)
	if e.wrapped != nil {
		if e.wrapped == target {
			return true
		}

		// Recursive check with cycle detection
		if wrapErr, ok := e.wrapped.(*wrapError); ok {
			if wrapErr.isWithVisited(target, visited) {
				return true
			}
		} else if e.wrapped.Is(target) {
			return true
		}
	}

	// Check if the base error matches (use our optimized baseError.Is)
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
	fields = prepareFields(fields)
	if wrapped == nil {
		wrapped = New(message, fields...)
	}

	// Calculate the depth without mutating the original base error
	depth := calculateWrapDepth(wrapped)

	if depth > maxWrapDepth {
		return Wrap(ErrMaxWrapDepthExceeded, message, fields...)
	}

	baseInt := wrapped.GetBase()
	base, ok := baseInt.(*baseError)
	if !ok {
		// This shouldn't happen, but handle gracefully
		return New(message, fields...)
	}

	if base.span != nil {
		base.span.SetAttributes(fields...)
	}

	return &wrapError{
		base:        base,
		wrapped:     wrapped,
		wrapMessage: truncateString(message, maxMessageLength),
		wrapFields:  fields,
		createdAt:   time.Now(),
		wrapDepth:   depth,
	}
}

// calculateWrapDepth calculates the wrap depth without mutating any errors
func calculateWrapDepth(err Error) int {
	depth := 0
	current := err
	visited := make(map[Error]bool) // Cycle detection

	for current != nil {
		// Cycle detection
		if visited[current] {
			break
		}
		visited[current] = true

		if wrapErr, ok := current.(*wrapError); ok {
			depth++
			current = wrapErr.wrapped
		} else {
			// This is a base error, stop counting
			break
		}
	}

	return depth
}
