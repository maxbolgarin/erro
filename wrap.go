package erro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// wrapError is a lightweight wrapper that points to baseError
type wrapError struct {
	// The previous error in the chain. Can be a *baseError or another *wrapError.
	wrapped Error

	// --- Context for THIS level ONLY ---
	wrapMessage string
	fields      []any

	// We only store the context if it was set at this level.
	// Otherwise, it's left as the zero value (e.g., "" for string).
	id        string
	class     Class
	category  Category
	severity  Severity
	retryable *bool // Use a pointer for a tri-state: nil=not set, true=set true, false=set false
	span      Span

	fullMessage atomicValue[string] // Full message with fields (caching)

	formatter        FormatErrorFunc
	stackTraceConfig *StackTraceConfig
}

// errorWithoutSeverity returns the error message without severity label
func (e *wrapError) Error() (out string) {
	if e.wrapMessage == "" {
		return e.Unwrap().Error()
	}
	if fullMessage := e.fullMessage.Load(); fullMessage != "" {
		return fullMessage
	}
	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe error message
			out = fmt.Sprintf("error formatting failed: %v", r)
		}
	}()
	out = e.wrapMessage
	if formatter := e.Formatter(); formatter != nil {
		out = unwrapErrorMessage(e, formatter(e))
	}
	e.fullMessage.Store(out)
	return out
}

func (e *wrapError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *wrapError) Unwrap() error {
	return e.wrapped
}

// Chaining methods for wrapError - these modify the base error
func (e *wrapError) WithID(id string) Error {
	if id == "" {
		return e
	}
	return &wrapError{
		wrapped: e,
		id:      truncateString(id, MaxKeyLength),
	}
}

func (e *wrapError) WithCategory(category Category) Error {
	if category == CategoryUnknown {
		return e
	}
	return &wrapError{
		wrapped:  e,
		category: truncateString(category, MaxValueLength),
	}
}

func (e *wrapError) WithClass(class Class) Error {
	if class == ClassUnknown {
		return e
	}
	return &wrapError{
		wrapped: e,
		class:   truncateString(class, MaxValueLength),
	}
}

func (e *wrapError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		return e
	}
	return &wrapError{
		wrapped:  e,
		severity: severity,
	}
}

func (e *wrapError) WithFields(fields ...any) Error {
	if len(fields) == 0 {
		return e
	}

	preparedFields := prepareFields(fields)
	if len(preparedFields) == 0 {
		return e
	}

	// Create a shallow copy of the current wrapError.
	newE := *e
	newE.fields = make([]any, 0, len(e.fields)+len(preparedFields))
	newE.fields = append(newE.fields, e.fields...)
	newE.fields = append(newE.fields, preparedFields...)

	// Invalidate the message cache on the new copy.
	newE.fullMessage.Store("")

	// Record fields in the span if it exists.
	if newE.span != nil {
		newE.span.SetAttributes(preparedFields...)
	}

	return &newE
}

func (e *wrapError) WithStackTraceConfig(config *StackTraceConfig) Error {
	if config == nil {
		return e
	}
	newE := *e
	newE.stackTraceConfig = config
	newE.fullMessage.Store("")
	return &newE
}

func (e *wrapError) WithFormatter(formatter FormatErrorFunc) Error {
	if formatter == nil {
		return e
	}
	newE := *e
	newE.formatter = formatter
	newE.fullMessage.Store("")
	return &newE
}

func (e *wrapError) Formatter() FormatErrorFunc {
	if e.formatter != nil {
		return e.formatter
	}
	if e.wrapped != nil {
		if f, ok := e.wrapped.(interface{ Formatter() FormatErrorFunc }); ok {
			return f.Formatter()
		}
	}
	return nil
}

func (e *wrapError) StackTraceConfig() *StackTraceConfig {
	if e.stackTraceConfig != nil {
		return e.stackTraceConfig
	}
	if e.wrapped != nil {
		if f, ok := e.wrapped.(interface{ StackTraceConfig() *StackTraceConfig }); ok {
			return f.StackTraceConfig()
		}
	}
	return nil
}

func (e *wrapError) WithRetryable(retryable bool) Error {
	return &wrapError{
		wrapped:   e,
		retryable: &retryable,
	}
}

func (e *wrapError) WithSpan(span Span) Error {
	if span == nil {
		return e
	}
	span.SetAttributes(e.Fields()...)
	span.RecordError(e)
	return &wrapError{
		wrapped: e,
		span:    span,
	}
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
func (e *wrapError) BaseError() ErrorContext {
	if e.wrapped != nil {
		return e.wrapped.Context().BaseError()
	}
	return e
}
func (e *wrapError) Context() ErrorContext {
	return e
}

func (e *wrapError) ID() string {
	if e.id != "" {
		return e.id
	}
	if e.wrapped != nil {
		return e.wrapped.Context().ID()
	}
	return ""
}
func (e *wrapError) Class() Class {
	if e.class != ClassUnknown {
		return e.class
	}
	if e.wrapped != nil {
		return e.wrapped.Context().Class()
	}
	return ClassUnknown
}
func (e *wrapError) Category() Category {
	if e.category != CategoryUnknown {
		return e.category
	}
	if e.wrapped != nil {
		return e.wrapped.Context().Category()
	}
	return CategoryUnknown
}
func (e *wrapError) IsRetryable() bool {
	if e.retryable != nil {
		return *e.retryable
	}
	if e.wrapped != nil {
		return e.wrapped.Context().IsRetryable()
	}
	return false
}
func (e *wrapError) Span() Span {
	if e.span != nil {
		return e.span
	}
	if e.wrapped != nil {
		return e.wrapped.Context().Span()
	}
	return nil
}
func (e *wrapError) Created() time.Time {
	if e.wrapped != nil {
		return e.wrapped.Context().Created()
	}
	return time.Time{}
}
func (e *wrapError) Severity() Severity {
	if e.severity != SeverityUnknown {
		return e.severity
	}
	if e.wrapped != nil {
		return e.wrapped.Context().Severity()
	}
	return SeverityUnknown
}
func (e *wrapError) Stack() Stack {
	if e.wrapped != nil {
		return e.wrapped.Context().Stack()
	}
	return nil
}
func (e *wrapError) Message() string {
	if e.wrapMessage != "" {
		return e.wrapMessage
	}
	if e.wrapped != nil {
		return e.wrapped.Context().Message()
	}
	return ""
}

func (e *wrapError) Fields() []any {
	if len(e.fields) == 0 {
		return e.wrapped.Context().Fields()
	}
	out := make([]any, len(e.fields))
	copy(out, e.fields)
	return out
}

func (e *wrapError) AllFields() []any {
	var allFields []any
	// Recursively get the fields from the earlier parts of the chain.
	if e.wrapped != nil {
		allFields = e.wrapped.Context().Fields()
	}

	// Append the fields from the current level.
	// We must return a new slice to preserve immutability.
	combined := make([]any, 0, len(allFields)+len(e.fields))
	combined = append(combined, allFields...)
	combined = append(combined, e.fields...)
	return combined
}

// Is reports whether this error can be considered a match for the target.
//
// It is designed for use by the standard `errors.Is` function. The primary
// matching mechanism for `erro` types is the unique error ID.
//
// It checks if the target is also an `erro` type and compares their effective IDs.
// The `ID()` method on `wrapError` correctly traverses the chain to find the
// outermost, most specific ID.
//
// If a match is not found, it returns false, allowing `errors.Is` to proceed
// by calling `Unwrap()` on this error.
func (e *wrapError) Is(target error) bool {
	targetCtx := ExtractContext(target)
	if targetCtx == nil {
		// Target is not an `erro` type. We cannot compare by ID.
		// Delegate to `errors.Is` to check the wrapped error.
		return false
	}

	// Both are `erro` types. Compare by their effective IDs.
	// The ID() method correctly finds the outermost ID in a chain.
	eID := e.ID()
	targetID := targetCtx.ID()

	if eID != "" && targetID != "" {
		return eID == targetID
	}

	return false
}

func (e *wrapError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

func newWrapError(wrapped Error, message string, fields ...any) Error {
	// 1. Perform the check before creating the new wrapper.
	depth := calculateWrapDepth(wrapped)
	if depth >= MaxWrapDepth {
		// Do not create a new wrapper. Instead, return a specific error.
		// We wrap the original 'wrapped' error so it's not lost.
		return Wrap(ErrMaxWrapDepthExceeded, "failed to wrap error", "original_error", wrapped)
	}

	preparedFields := prepareFields(fields)
	if wrapped == nil {
		// This should ideally not happen if called from Wrap/Wrapf, but as a safeguard:
		return newBaseError(nil, message, preparedFields...)
	}

	return &wrapError{
		wrapped:     wrapped,
		wrapMessage: truncateString(message, MaxMessageLength),
		fields:      preparedFields,
		formatter:   FormatErrorWithFields,
		id:          newID(),
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
