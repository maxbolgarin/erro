package erro

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// baseError holds the root error with all context and metadata.
type baseError struct {
	// Core error info
	originalErr error               // Original error if wrapping external error
	wrappedErr  *baseError          // Wrapped error if wrapping erro error
	message     string              // Base message
	fullMessage atomicValue[string] // Full message with fields (caching)

	// Metadata
	id        string        // Error id
	class     ErrorClass    // Error class
	category  ErrorCategory // Error category
	severity  ErrorSeverity // Error severity
	retryable bool          // Retryable flag
	fields    []any         // Key-value fields
	span      TraceSpan     // Span
	created   time.Time     // Creation timestamp

	stack  rawStack           // Stack trace (program counters only - resolved on demand)
	frames atomicValue[Stack] // Stack trace frames (for caching)

	formatter        FormatErrorFunc
	stackTraceConfig *StackTraceConfig
}

// Error implements the error interface.
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

	if formatter := e.Formatter(); formatter != nil {
		out = formatter(e)
		if unwrapped := e.Unwrap(); unwrapped != nil {
			unwrappedMsg := unwrapped.Error()
			if unwrappedMsg == "" {
				return out
			}
			if out == "" {
				return unwrappedMsg
			}
			return out + ": " + unwrappedMsg
		}
	}
	if out == "" {
		out = FormatErrorMessage(e)
	}
	return out
}

// Format implements [fmt.Formatter] for stack trace printing.
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the [errors.Unwrap] interface.
func (e *baseError) Unwrap() error {
	if e.wrappedErr != nil {
		return e.wrappedErr
	}
	return e.originalErr
}

// Is reports whether this [Error] can be considered a match for the target.
//
// It is designed for use by the standard [errors.Is] function. The primary
// matching mechanism for [Error] types is the unique error ID.
//
// It checks if the target is also an [Error] type and compares their IDs.
// If both have a non-empty ID and they match, it returns true.
//
// In all other cases, it returns false, delegating the decision to the
// [errors.Is] function, which will then check the wrapped standard error
// by calling [Unwrap].
func (e *baseError) Is(target error) bool {
	if e == nil {
		return false
	}
	targetErr, ok := target.(Error)
	if !ok {
		return false
	}

	// 1. Compare by ID if both are specified. This is the strongest link.
	eID := e.ID()
	targetID := targetErr.ID()
	if eID != "" && targetID != "" {
		return eID == targetID
	}

	// 2. Compare by Class if the target's class is specified.
	// This allows for checking against error "types" or templates.
	targetClass := targetErr.Class()
	targetCategory := targetErr.Category()
	targetSeverity := targetErr.Severity()
	targetRetryable := targetErr.IsRetryable()

	if targetID == "" && (targetClass != ClassUnknown || targetCategory != CategoryUnknown) {
		return e.Class() == targetClass && e.Category() == targetCategory &&
			e.Severity() == targetSeverity && e.IsRetryable() == targetRetryable
	}

	return false
}

// As implements the [errors.As] interface.
func (e *baseError) As(target any) bool {
	targetPtr, ok := target.(**baseError)
	if ok && targetPtr != nil {
		*targetPtr = e
		return true
	}
	if e.wrappedErr != nil {
		return e.wrappedErr.As(target)
	}
	if e.originalErr != nil {
		return errors.As(e.originalErr, target)
	}
	return false
}

// MarshalJSON implements the [json.Marshaler] interface.
func (e *baseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
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

// ID returns the error's identifier.
func (e *baseError) ID() string {
	if e.id == "" && e.wrappedErr != nil {
		return e.wrappedErr.ID()
	}
	return e.id
}

// Class returns the error's class.
func (e *baseError) Class() ErrorClass {
	if e.class == "" && e.wrappedErr != nil {
		return e.wrappedErr.Class()
	}
	return e.class
}

// Category returns the error's category.
func (e *baseError) Category() ErrorCategory {
	if e.category == "" && e.wrappedErr != nil {
		return e.wrappedErr.Category()
	}
	return e.category
}

// Severity returns the error's severity.
func (e *baseError) Severity() ErrorSeverity {
	if e.severity == "" && e.wrappedErr != nil {
		return e.wrappedErr.Severity()
	}
	return e.severity
}

// IsRetryable returns true if the error is marked as retryable.
func (e *baseError) IsRetryable() bool {
	if !e.retryable && e.wrappedErr != nil {
		return e.wrappedErr.IsRetryable()
	}
	return e.retryable
}

// Message returns the error's message.
func (e *baseError) Message() string {
	if e.message != "" {
		return e.message
	}
	if e.wrappedErr != nil {
		return e.wrappedErr.Message()
	}
	if e.originalErr != nil {
		return e.originalErr.Error()
	}
	return ""
}

// Fields returns the error's fields.
func (e *baseError) Fields() []any {
	if len(e.fields) == 0 && e.wrappedErr != nil {
		return e.wrappedErr.Fields()
	}
	fields := make([]any, len(e.fields))
	copy(fields, e.fields)
	return fields
}

// Created returns the time the error was created.
func (e *baseError) Created() time.Time {
	if e.created.IsZero() && e.wrappedErr != nil {
		return e.wrappedErr.Created()
	}
	return e.created
}

// Span returns the error's trace span.
func (e *baseError) Span() TraceSpan {
	if e.span == nil && e.wrappedErr != nil {
		return e.wrappedErr.Span()
	}
	return e.span
}

// AllFields returns all fields from the error and its wrapped errors.
func (e *baseError) AllFields() []any {
	var wrappedFields []any
	if e.wrappedErr != nil {
		wrappedFields = e.wrappedErr.AllFields()
	}

	fields := make([]any, 0, len(e.fields)+len(wrappedFields))
	fields = append(fields, e.fields...)
	fields = append(fields, wrappedFields...)

	return fields
}

// BaseError returns the lowest-level error in the wrap chain.
func (e *baseError) BaseError() Error {
	if e.wrappedErr != nil {
		return e.wrappedErr.BaseError()
	}
	return e
}

// LogFields returns the error's fields for logging.
func (e *baseError) LogFields(opts ...LogOptions) []any {
	return getLogFields(e, opts...)
}

// LogFieldsMap returns the error's fields for logging as a map.
func (e *baseError) LogFieldsMap(opts ...LogOptions) map[string]any {
	return getLogFieldsMap(e, opts...)
}

// Stack returns the error's stack trace.
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

// StackTraceConfig returns the configuration for the stack trace.
func (e *baseError) StackTraceConfig() *StackTraceConfig {
	if e.stackTraceConfig == nil && e.wrappedErr != nil {
		return e.wrappedErr.StackTraceConfig()
	}
	return e.stackTraceConfig
}

// Formatter returns the error's message formatter.
func (e *baseError) Formatter() FormatErrorFunc {
	if e.formatter == nil && e.wrappedErr != nil {
		return e.wrappedErr.Formatter()
	}
	return e.formatter
}

func newBaseError(message string, meta ...any) *baseError {
	e := &baseError{
		message:   truncateString(message, MaxMessageLength),
		formatter: FormatErrorWithFields,
		created:   time.Now(),
	}
	return applyMeta(e, meta...)
}

func newWrapError(errorToWrap error, message string, meta ...any) *baseError {
	e := &baseError{
		message:   truncateString(message, MaxMessageLength),
		formatter: FormatErrorWithFields,
	}
	switch err := errorToWrap.(type) {
	case *baseError:
		e.wrappedErr = err
	default:
		e.originalErr = err
	}
	return applyMeta(e, meta...)
}

func applyMeta(e *baseError, meta ...any) *baseError {
	if len(meta) == 0 {
		return e
	}

	preparedFields := make([]any, 0, getFieldsCapFromMeta(meta))
	for _, f := range meta {
		if f == nil {
			continue
		}
		switch f := f.(type) {
		case errorOpt:
			f(e)
		case errorFields:
			if fields := f(); len(fields) > 0 {
				preparedFields = append(preparedFields, fields...)
			}
		case ErrorClass:
			e.class = f
		case ErrorCategory:
			e.category = f
		case ErrorSeverity:
			e.severity = f
		case errorWork:
			continue
		default:
			preparedFields = append(preparedFields, f)
		}
	}
	if len(preparedFields)%2 != 0 {
		preparedFields = append(preparedFields, MissingFieldPlaceholder)
	}
	if len(preparedFields) > maxPairsCount {
		newPreparedFields := make([]any, maxPairsCount)
		copy(newPreparedFields, preparedFields)
		preparedFields = newPreparedFields
	}
	e.fields = preparedFields
	if e.id == "" && e.wrappedErr == nil {
		e.id = newID()
	}
	runWorkers(e, meta)
	return e
}

func getFieldsCapFromMeta(meta []any) int {
	cap := 0
	for _, f := range meta {
		switch f := f.(type) {
		case errorFields:
			cap += len(f())
		case errorOpt, errorWork, ErrorClass, ErrorCategory, ErrorSeverity:
			continue
		default:
			cap++
		}
	}
	if cap%2 != 0 {
		cap++
	}
	return cap
}

func runWorkers(e *baseError, meta []any) {
	for _, f := range meta {
		switch f := f.(type) {
		case errorWork:
			f(e)
		}
	}
}
