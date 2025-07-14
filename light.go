package erro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type lightError struct {
	message     string
	cause       error // For wrapping external errors
	fullMessage atomicValue[string]

	id        string
	class     Class
	category  Category
	severity  Severity
	retryable bool
	fields    []any
	span      Span

	formatter FormatErrorFunc
}

func NewLight(message string, fields ...any) Error {
	return newLightError(nil, message, fields...)
}

func WrapLight(err error, message string, fields ...any) Error {
	return newLightError(err, message, fields...)
}

func (e *lightError) Error() (out string) {
	if fullMessage := e.fullMessage.Load(); fullMessage != "" {
		return fullMessage
	}
	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe error message
			out = fmt.Sprintf("error formatting failed: %v", r)
		}
	}()
	out = e.message
	if formatter := e.Formatter(); formatter != nil {
		out = unwrapErrorMessage(e, formatter(e))
	}
	e.fullMessage.Store(out)
	return out
}

// Unwrap implements the Unwrap interface
func (e *lightError) Unwrap() error {
	return e.cause
}

// Implement Error interface methods (lightweight versions)
func (e *lightError) WithID(id string) Error {
	if id == "" {
		return e
	}
	newE := *e // Create a copy
	newE.id = truncateString(id, MaxKeyLength)
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithClass(class Class) Error {
	if class == ClassUnknown {
		return e
	}
	newE := *e // Create a copy
	newE.class = truncateString(class, MaxValueLength)
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithCategory(category Category) Error {
	if category == CategoryUnknown {
		return e
	}
	newE := *e // Create a copy
	newE.category = truncateString(category, MaxValueLength)
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		return e
	}
	newE := *e // Create a copy
	newE.severity = severity
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithRetryable(retryable bool) Error {
	newE := *e
	newE.retryable = retryable
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithFields(fields ...any) Error {
	if len(fields) == 0 {
		return e
	}
	preparedFields := prepareFields(fields)
	if len(preparedFields) == 0 {
		return e
	}

	newE := *e
	newE.fields = make([]any, 0, len(e.fields)+len(preparedFields))
	newE.fields = append(newE.fields, e.fields...)
	newE.fields = append(newE.fields, preparedFields...)

	newE.fullMessage.Store("") // Invalidate cache on the copy

	if newE.span != nil {
		newE.span.SetAttributes(preparedFields...)
	}
	return &newE
}

func (e *lightError) WithSpan(span Span) Error {
	if span == nil {
		return e
	}
	newE := *e // Create a copy
	span.SetAttributes(newE.fields...)
	span.RecordError(&newE)
	newE.span = span
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithFormatter(formatter FormatErrorFunc) Error {
	if formatter == nil {
		return e
	}
	newE := *e
	newE.formatter = formatter
	newE.fullMessage.Store("")
	return &newE
}

func (e *lightError) WithStackTraceConfig(config *StackTraceConfig) Error {
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
	if e.id != "" {
		return e.id
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.ID()
		}
	}
	return ""
}
func (e *lightError) Class() Class {
	if e.class != ClassUnknown {
		return e.class
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Class()
		}
	}
	return ClassUnknown
}
func (e *lightError) Category() Category {
	if e.category != CategoryUnknown {
		return e.category
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Category()
		}
	}
	return CategoryUnknown
}
func (e *lightError) IsRetryable() bool {
	if e.retryable {
		return e.retryable
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.IsRetryable()
		}
	}
	return false
}
func (e *lightError) Span() Span {
	if e.span != nil {
		return e.span
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Span()
		}
	}
	return nil
}
func (e *lightError) Created() time.Time {
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Created()
		}
	}
	return time.Time{}
}
func (e *lightError) Message() string {
	if e.message != "" {
		return e.message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return ""
}
func (e *lightError) Severity() Severity {
	if e.severity != SeverityUnknown {
		return e.severity
	}
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Severity()
		}
	}
	return SeverityUnknown
}
func (e *lightError) Fields() []any {
	if len(e.fields) == 0 {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			return causeErro.Fields()
		}
	}
	out := make([]any, len(e.fields))
	copy(out, e.fields)
	return out
}
func (e *lightError) AllFields() []any {
	var allFields []any
	// Recursively get the fields from the earlier parts of the chain.
	if e.cause != nil {
		if causeErro, ok := e.cause.(ErrorContext); ok {
			allFields = causeErro.Fields()
		}
	}

	// Append the fields from the current level.
	// We must return a new slice to preserve immutability.
	combined := make([]any, 0, len(allFields)+len(e.fields))
	combined = append(combined, allFields...)
	combined = append(combined, e.fields...)
	return combined
}

func (e *lightError) Formatter() FormatErrorFunc {
	if e.formatter != nil {
		return e.formatter
	}
	if e.cause != nil {
		if f, ok := e.cause.(interface{ Formatter() FormatErrorFunc }); ok {
			return f.Formatter()
		}
	}
	return nil
}

func (e *lightError) StackTraceConfig() *StackTraceConfig {
	return nil
}

func (e *lightError) BaseError() ErrorContext {
	if e.cause != nil {
		if causeErro, ok := e.cause.(Error); ok {
			return causeErro.Context().BaseError()
		}
	}
	return e
}

// Stack methods - lightweight errors have no stack traces
func (e *lightError) Stack() Stack {
	return Stack{}
}

func (e *lightError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "%s", e.Error())
}

// Is reports whether this error can be considered a match for the target.
//
// It is designed for use by the standard `errors.Is` function. The primary
// matching mechanism for `erro` types is the unique error ID.
//
// It checks if the target is also an `erro` type and compares their IDs.
// If both have a non-empty ID and they match, it returns true.
//
// In all other cases, it returns false, allowing `errors.Is` to proceed
// by calling `Unwrap` to check the wrapped `cause` error.
func (e *lightError) Is(target error) bool {
	targetCtx := ExtractContext(target)
	if targetCtx == nil {
		// Target is not an `erro` type. We cannot compare by ID.
		// Delegate to `errors.Is` to check the wrapped `cause`.
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

func (e *lightError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

// newLightError creates a new lightweight error
func newLightError(cause error, message string, fields ...any) *lightError {
	e := &lightError{
		id:        newID(),
		message:   truncateString(message, MaxMessageLength),
		cause:     cause,
		fields:    prepareFields(fields),
		formatter: FormatErrorWithFields,
	}
	return e
}
