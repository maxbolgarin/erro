package erro

import (
	"encoding/json"
	"fmt"
	"time"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error               // Original error if wrapping external error
	wrappedErr  *baseError          // Wrapped error if wrapping erro error
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
		e.fullMessage.Store(out)

		if r := recover(); r != nil {
			// Fallback to safe error message
			out = fmt.Sprintf("error formatting failed: %v", r)
		}
	}()
	out = e.message
	if formatter := e.Formatter(); formatter != nil {
		out = formatter(e)
		if unwrapped := e.Unwrap(); unwrapped != nil {
			if out == "" {
				return unwrapped.Error()
			}
			return out + ": " + unwrapped.Error()
		}
	}
	return out
}

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *baseError) Unwrap() error {
	if e.wrappedErr != nil {
		return e.wrappedErr
	}
	return e.originalErr
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
	targetCtx := ExtractError(target)
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

	if e.wrappedErr != nil {
		return e.wrappedErr.Is(target)
	}

	if e.originalErr != nil {
		if isErr, ok := e.originalErr.(interface{ Is(error) bool }); ok {
			return isErr.Is(target)
		}
	}

	return false
}

func (e *baseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

func (e *baseError) UnmarshalJSON(data []byte) error {
	var schema ErrorSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}
	e.originalErr = nil
	e.message = schema.Message
	e.created = schema.Created
	e.span = nil
	e.stack = nil
	e.frames = atomicValue[Stack]{}
	e.formatter = FormatErrorWithFields
	e.id = schema.ID
	e.class = schema.Class
	e.category = schema.Category
	e.severity = schema.Severity
	e.retryable = schema.Retryable
	e.fields = schema.Fields
	return nil
}

// Getter methods for baseError
func (e *baseError) ID() string {
	if e.id == "" && e.wrappedErr != nil {
		return e.wrappedErr.ID()
	}
	return e.id
}

func (e *baseError) Class() Class {
	if e.class == "" && e.wrappedErr != nil {
		return e.wrappedErr.Class()
	}
	return e.class
}

func (e *baseError) Category() Category {
	if e.category == "" && e.wrappedErr != nil {
		return e.wrappedErr.Category()
	}
	return e.category
}

func (e *baseError) Severity() Severity {
	if e.severity == "" && e.wrappedErr != nil {
		return e.wrappedErr.Severity()
	}
	return e.severity
}

func (e *baseError) IsRetryable() bool {
	if !e.retryable && e.wrappedErr != nil {
		return e.wrappedErr.IsRetryable()
	}
	return e.retryable
}

func (e *baseError) Message() string {
	if e.message == "" && e.wrappedErr != nil {
		return e.wrappedErr.Message()
	}
	return e.message
}

func (e *baseError) Fields() []any {
	if len(e.fields) == 0 && e.wrappedErr != nil {
		return e.wrappedErr.Fields()
	}
	fields := make([]any, len(e.fields))
	copy(fields, e.fields)
	return fields
}

func (e *baseError) Created() time.Time {
	if e.created.IsZero() && e.wrappedErr != nil {
		return e.wrappedErr.Created()
	}
	return e.created
}

func (e *baseError) Span() Span {
	if e.span == nil && e.wrappedErr != nil {
		return e.wrappedErr.Span()
	}
	return e.span
}

func (e *baseError) AllFields() []any {
	if len(e.fields) == 0 && e.wrappedErr != nil {
		return e.wrappedErr.AllFields()
	}
	return e.Fields()
}

func (e *baseError) BaseError() Error {
	if e.wrappedErr != nil {
		return e.wrappedErr.BaseError()
	}
	return e
}

func (e *baseError) Stack() Stack {
	return e.getStack(e.StackTraceConfig())
}

func (e *baseError) getStack(cfg *StackTraceConfig) Stack {
	if e.stack == nil && e.wrappedErr != nil {
		return e.wrappedErr.getStack(cfg)
	}
	frames := e.frames.Load()
	if frames == nil && e.stack != nil {
		frames = e.stack.toFrames(cfg)
		e.frames.Store(frames)
	}
	return frames
}

func (e *baseError) StackTraceConfig() *StackTraceConfig {
	if e.stackTraceConfig == nil && e.wrappedErr != nil {
		return e.wrappedErr.StackTraceConfig()
	}
	return e.stackTraceConfig
}

func (e *baseError) Formatter() FormatErrorFunc {
	if e.formatter == nil && e.wrappedErr != nil {
		return e.wrappedErr.Formatter()
	}
	return e.formatter
}

// newBaseError creates a new base error with security validation
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return newBaseErrorWithStackSkip(3, originalErr, message, fields...)
}

func newBaseErrorLight(originalErr error, message string, fields ...any) *baseError {
	return newBaseErrorWithStackSkip(0, originalErr, message, fields...)
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

func newWrapError(errorToWrap *baseError, message string, fields ...any) *baseError {
	e := &baseError{
		id:         newID(),
		wrappedErr: errorToWrap,
		message:    truncateString(message, MaxMessageLength),
		created:    time.Now(),
		fields:     prepareFields(fields),
		formatter:  FormatErrorWithFields,
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
