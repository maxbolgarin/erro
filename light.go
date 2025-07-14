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
	fullMessage string

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
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe error message
			out = fmt.Sprintf("error formatting failed: %v", r)
		}
	}()
	e.fullMessage = e.message
	if formatter := e.Formatter(); formatter != nil {
		e.fullMessage = unwrapErrorMessage(e, formatter(e))
	}
	return e.fullMessage
}

// Unwrap implements the Unwrap interface
func (e *lightError) Unwrap() error {
	return e.cause
}

// Implement Error interface methods (lightweight versions)
func (e *lightError) WithID(idRaw ...string) Error {
	newE := *e // Create a copy
	if len(idRaw) > 0 {
		newE.id = truncateString(idRaw[0], maxCodeLength)
	} else if newE.id == "" {
		newE.id = newID(newE.class, newE.category)
	}
	return &newE
}

func (e *lightError) WithClass(class Class) Error {
	newE := *e // Create a copy
	newE.class = class
	return &newE
}

func (e *lightError) WithCategory(category Category) Error {
	newE := *e // Create a copy
	newE.category = category
	return &newE
}

func (e *lightError) WithSeverity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	newE := *e // Create a copy
	newE.severity = severity
	newE.fullMessage = ""
	return &newE
}

func (e *lightError) WithRetryable(retryable bool) Error {
	newE := *e // Create a copy
	newE.retryable = retryable
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

	newE := *e // Create a copy
	newE.fields = make([]any, 0, len(e.fields)+len(preparedFields))
	newE.fields = append(newE.fields, e.fields...)
	newE.fields = append(newE.fields, preparedFields...)

	newE.fullMessage = "" // Invalidate cache on the copy

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
	return &newE
}

func (e *lightError) WithFormatter(formatter FormatErrorFunc) Error {
	newE := *e
	newE.formatter = formatter
	newE.fullMessage = ""
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
	if e.id == "" {
		e.id = newID(e.class, e.category)
	}
	return e.id
}
func (e *lightError) Class() Class       { return e.class }
func (e *lightError) Category() Category { return e.category }
func (e *lightError) IsRetryable() bool  { return e.retryable }
func (e *lightError) Span() Span         { return e.span }
func (e *lightError) Created() time.Time { return time.Time{} }
func (e *lightError) Message() string    { return e.message }

// Severity checking methods
func (e *lightError) Severity() Severity {
	if e.severity == "" {
		return SeverityUnknown
	}
	return e.severity
}

func (e *lightError) Fields() []any {
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

// Is method optimized for lightweight comparison
func (e *lightError) Is(target error) (ok bool) {
	if target == nil {
		return false
	}

	// Direct equality check (fastest)
	if e == target {
		return true
	}

	// Early exit for different types
	if _, isErro := target.(Error); !isErro {
		// For external errors, compare messages directly
		return e.message == target.Error()
	}

	// For erro errors, use optimized comparison
	if targetErro, ok := target.(ErrorContext); ok {
		// Compare by ID first (fastest)
		if e.id != "" {
			if targetID := targetErro.ID(); targetID != "" {
				return e.id == targetID
			}
		}

		// Compare messages (second fastest)
		if e.message == targetErro.Message() {
			return true
		}
	}

	// Handle wrapped errors
	if e.cause != nil {
		if e.cause == target {
			return true
		}
		if x, ok := e.cause.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}
	}

	return false
}

func (e *lightError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

// newLightError creates a new lightweight error
func newLightError(cause error, message string, fields ...any) *lightError {
	e := &lightError{
		message:   truncateString(message, maxMessageLength),
		cause:     cause,
		fields:    prepareFields(fields),
		formatter: GetGlobalFormatter(),
	}
	AddToGatherer(e)
	return e
}
