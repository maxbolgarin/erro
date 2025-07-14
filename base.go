package erro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error               // Original error if wrapping external error
	message     string              // Base message
	fullMessage atomicValue[string] // Full message with fields (caching)

	// Metadata
	id        string    // Error id
	class     Class     // Error class
	category  Category  // Error category
	severity  Severity  // Error severity
	retryable bool      // Retryable flag
	fields    []any     // Key-value fields
	span      Span      // Span
	created   time.Time // Creation timestamp

	stack  rawStack           // Stack trace (program counters only - resolved on demand)
	frames atomicValue[Stack] // Stack trace frames (for caching)

	formatter        FormatErrorFunc
	stackTraceConfig *StackTraceConfig
}

// Error implements the error interface
func (e *baseError) Error() (out string) {
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

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *baseError) Unwrap() error {
	return e.originalErr
}

// Chaining methods for baseError
func (e *baseError) WithID(id string) Error {
	if id == "" {
		return e
	}
	return &wrapError{
		wrapped: e,
		id:      truncateString(id, MaxKeyLength),
	}
}

func (e *baseError) WithCategory(category Category) Error {
	if category == CategoryUnknown {
		return e
	}
	return &wrapError{
		wrapped:  e,
		category: truncateString(category, MaxValueLength),
	}
}

func (e *baseError) WithClass(class Class) Error {
	if class == ClassUnknown {
		return e
	}
	return &wrapError{
		wrapped: e,
		class:   truncateString(class, MaxValueLength),
	}
}

func (e *baseError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		return e
	}
	return &wrapError{
		wrapped:  e,
		severity: severity,
	}
}

func (e *baseError) WithFields(fields ...any) Error {
	if len(fields) == 0 {
		return e
	}

	preparedFields := prepareFields(fields)
	if len(preparedFields) == 0 {
		return e
	}

	// Create a shallow copy of the current baseError.
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

func (e *baseError) WithRetryable(retryable bool) Error {
	return &wrapError{
		wrapped:   e,
		retryable: &retryable,
	}
}

func (e *baseError) WithSpan(span Span) Error {
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

func (e *baseError) WithFormatter(formatter FormatErrorFunc) Error {
	if formatter == nil {
		return e
	}
	newE := *e
	newE.formatter = formatter
	newE.fullMessage.Store("")
	return &newE
}

func (e *baseError) WithStackTraceConfig(config *StackTraceConfig) Error {
	if config == nil {
		return e
	}
	newE := *e
	newE.stackTraceConfig = config
	newE.frames.Store(nil) // Invalidate cached frames
	newE.fullMessage.Store("")
	return &newE
}

func (e *baseError) StackTraceConfig() *StackTraceConfig {
	return e.stackTraceConfig
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

func (e *baseError) Formatter() FormatErrorFunc {
	return e.formatter
}

// Getter methods for baseError
func (e *baseError) Context() ErrorContext { return e }
func (e *baseError) ID() string {
	if e.id != "" {
		return e.id
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.ID()
		}
	}
	return ""
}
func (e *baseError) Class() Class {
	if e.class != ClassUnknown {
		return e.class
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Class()
		}
	}
	return ClassUnknown
}
func (e *baseError) Category() Category {
	if e.category != CategoryUnknown {
		return e.category
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Category()
		}
	}
	return CategoryUnknown
}
func (e *baseError) IsRetryable() bool {
	if e.retryable {
		return e.retryable
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.IsRetryable()
		}
	}
	return false
}
func (e *baseError) Span() Span {
	if e.span != nil {
		return e.span
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Span()
		}
	}
	return nil
}
func (e *baseError) Message() string {
	if e.message != "" {
		return e.message
	}
	if e.originalErr != nil {
		return e.originalErr.Error()
	}
	return ""
}
func (e *baseError) Severity() Severity {
	if e.severity != SeverityUnknown {
		return e.severity
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Severity()
		}
	}
	return SeverityUnknown
}

func (e *baseError) Fields() []any {
	if len(e.fields) == 0 {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Fields()
		}
	}
	fields := make([]any, len(e.fields))
	copy(fields, e.fields)
	return fields
}
func (e *baseError) AllFields() []any {
	if len(e.fields) == 0 {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.AllFields()
		}
	}
	fields := make([]any, len(e.fields))
	copy(fields, e.fields)
	return fields
}

func (e *baseError) Created() time.Time {
	if !e.created.IsZero() {
		return e.created
	}
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.Created()
		}
	}
	return time.Time{}
}

func (e *baseError) Stack() Stack {
	if e.frames.Load() == nil {
		e.frames.Store(e.stack.toFrames(e.StackTraceConfig()))
		e.fullMessage.Store("")
	}
	return e.frames.Load()
}
func (e *baseError) BaseError() ErrorContext {
	if e.originalErr != nil {
		if causeErro, ok := e.originalErr.(ErrorContext); ok {
			return causeErro.BaseError()
		}
	}
	return e
}

// Is reports whether this error can be considered a match for the target.
//
// It is designed for use by the standard `errors.Is` function. The primary
// matching mechanism for `erro` types is the unique error ID.
//
// It checks if the target is also an `erro` type and compares their IDs.
// If both have a non-empty ID and they match, it returns true.
//
// In all other cases, it returns false, delegating the decision to the
// `errors.Is` function, which will then check the wrapped standard error
// by calling `Unwrap`.
func (e *baseError) Is(target error) bool {
	targetCtx := ExtractContext(target)
	if targetCtx == nil {
		// Target is not an `erro` type. We cannot compare by ID.
		// Delegate to `errors.Is` to check the wrapped `originalErr`.
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

func (e *baseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

// newBaseError creates a new base error with security validation
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return newBaseErrorWithStackSkip(3, originalErr, message, fields...)
}

func newBaseErrorWithStackSkip(skip int, originalErr error, message string, fields ...any) *baseError {
	e := &baseError{
		id:          newID(),
		originalErr: originalErr,
		message:     truncateString(message, MaxMessageLength),
		created:     time.Now(),
		fields:      prepareFields(fields),
		stack:       captureStack(skip),
		formatter:   FormatErrorWithFields,
	}
	return e
}

// prepareFields prepares fields with validation and safe truncation
func prepareFields(fields []any) []any {
	if len(fields) == 0 {
		return fields
	}

	// Limit the number of fields to prevent DOS
	maxElements := MaxFieldsCount * 2
	if len(fields) > maxElements {
		fields = fields[:maxElements]
	}

	// Ensure even number of elements (key-value pairs)
	if len(fields)%2 != 0 {
		result := make([]any, len(fields)+1)
		copy(result, fields)
		result[len(fields)] = "<missing>"
		return result
	}

	result := make([]any, len(fields))
	copy(result, fields)
	return result
}
