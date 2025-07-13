package erro

import (
	"context"
	"fmt"
	"sync"
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

func (e *wrapError) Error() string {
	return e.errorString()
}

// errorWithoutSeverity returns the error message without severity label
func (e *wrapError) errorString(ignoreSeverity ...bool) (out string) {
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
			wrappedMsg = baseErr.errorString(true)
		} else if wrapErr, ok := e.wrapped.(*wrapError); ok {
			wrappedMsg = wrapErr.errorString(true)
		} else {
			wrappedMsg = safeErrorString(e.wrapped)
		}
	} else {
		wrappedMsg = e.base.errorString(true)
	}

	if out == "" {
		return wrappedMsg
	}

	if (len(ignoreSeverity) == 0 || !ignoreSeverity[0]) && e.base.severity != SeverityUnknown {
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
func (e *wrapError) WithID(idRaw ...string) Error {
	var id string
	if len(idRaw) > 0 {
		id = truncateString(idRaw[0], maxCodeLength)
	} else {
		id = e.base.ID()
	}
	e.base.id = id
	return e
}

func (e *wrapError) WithCategory(category Category) Error {
	e.base.category = category
	return e
}

func (e *wrapError) WithClass(class Class) Error {
	e.base.class = class
	return e
}

func (e *wrapError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.base.severity = severity
	e.fullMessage = ""
	return e
}

func (e *wrapError) WithFields(fields ...any) Error {
	e.wrapFields = safeAppendFields(e.wrapFields, prepareFields(fields))
	if e.base.span != nil {
		e.base.span.SetAttributes(e.wrapFields...)
	}
	e.fullMessage = ""
	return e
}

func (e *wrapError) WithRetryable(retryable bool) Error {
	e.base.retryable = retryable
	return e
}

func (e *wrapError) WithSpan(span Span) Error {
	if span == nil {
		return e
	}
	span.SetAttributes(e.base.fields...)
	span.SetAttributes(e.wrapFields...)
	span.RecordError(e)
	e.base.span = span
	return e
}

func (e *wrapError) RecordMetrics(metrics Metrics) Error {
	if metrics == nil {
		return e
	}
	metrics.RecordError(e)
	return e
}

func (e *wrapError) SendEvent(ctx context.Context, dispatcher Dispatcher) Error {
	if dispatcher == nil {
		return e
	}
	dispatcher.SendEvent(ctx, e)
	return e
}

// Getter methods for wrapError
func (e *wrapError) Context() ErrorContext   { return e }
func (e *wrapError) ID() string              { return e.base.ID() }
func (e *wrapError) Class() Class            { return e.base.Class() }
func (e *wrapError) Category() Category      { return e.base.Category() }
func (e *wrapError) IsRetryable() bool       { return e.base.IsRetryable() }
func (e *wrapError) Span() Span              { return e.base.Span() }
func (e *wrapError) Created() time.Time      { return e.base.Created() }
func (e *wrapError) Severity() Severity      { return e.base.Severity() }
func (e *wrapError) Stack() Stack            { return e.base.Stack() }
func (e *wrapError) BaseError() ErrorContext { return e.base }

func (e *wrapError) Message() string {
	return e.wrapMessage + ": " + e.wrapped.Context().Message()
}

func (e *wrapError) Fields() []any {
	// Add fields from wrapped error (if it exists)
	var wrappedFields []any
	if e.wrapped != nil {
		wrappedFields = e.wrapped.Context().Fields()
	} else {
		wrappedFields = e.base.fields
	}

	// Create with exact capacity to avoid reallocations
	allFields := make([]any, len(e.wrapFields)+len(wrappedFields))
	copy(allFields, e.wrapFields)
	copy(allFields[len(e.wrapFields):], wrappedFields)

	return allFields
}

var visitedMapPool = sync.Pool{
	New: func() any {
		return make(map[*baseError]bool, 16) // Pre-allocate with reasonable capacity
	},
}

// Is checks if this error or any wrapped error matches the target
func (e *wrapError) Is(target error) bool {
	visited, ok := visitedMapPool.Get().(map[*baseError]bool)
	if !ok {
		visited = make(map[*baseError]bool, 16)
	}
	defer func() {
		recover()
		// Clear the map before returning to pool
		for k := range visited {
			delete(visited, k)
		}
		visitedMapPool.Put(visited)
	}()
	return e.isWithVisited(target, visited, 0)
}

const maxCycleDetectionDepth = 50

// isWithVisited implements Is with cycle detection
func (e *wrapError) isWithVisited(target error, visited map[*baseError]bool, depth int) bool {
	if target == nil || depth > maxCycleDetectionDepth {
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
	}

	// Fast path: Check if target is an erro error and use optimized comparison
	if targetErro, ok := target.(ErrorContext); ok {
		// Compare by id if both have non-empty ids (very fast)
		if e.base != nil && e.base.id != "" && targetErro.ID() != "" {
			if e.base.id == targetErro.ID() {
				return true
			}
		}
		// Compare base messages (fast)
		if e.base != nil && e.base.message == targetErro.Message() {
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
			if wrapErr.isWithVisited(target, visited, depth+1) {
				return true
			}
		} else if e.wrapped.Context().Is(target) { // Recursive check with cycle detection
			return true
		}
	}

	// Check if the base error matches (use our optimized baseError.Is)
	if e.base != nil && e.base.Is(target) {
		return true
	}

	return false
}

func newWrapError(wrapped Error, message string, fields ...any) Error {
	fields = prepareFields(fields)
	if wrapped == nil {
		wrapped = newBaseError(nil, message, fields...)
	}

	// Calculate the depth without mutating the original base error
	depth := calculateWrapDepth(wrapped)

	if depth > maxWrapDepth {
		return Wrap(ErrMaxWrapDepthExceeded, message, fields...)
	}

	baseInt := wrapped.Context().BaseError()
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
