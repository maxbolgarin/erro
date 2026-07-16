package erro

import (
	"encoding/json"
	"fmt"
	"time"
)

// baseError holds the root error with all context and metadata.
//
// Wrapping state is one of:
//  1. wrappedErr set, originalErr nil — wrapping an erro error directly;
//  2. both set — wrapping a foreign error that contains an erro error
//     deeper in its chain: originalErr preserves the full chain for
//     Unwrap/Error, wrappedErr is the metadata-inheritance source;
//  3. originalErr set, wrappedErr nil — wrapping a foreign error.
type baseError struct {
	// Core error info
	originalErr error               // Original error if wrapping external error
	wrappedErr  *baseError          // Wrapped erro error (metadata inheritance source)
	message     string              // Base message
	fullMessage atomicValue[string] // Full message with fields (caching)
	depth       int                 // Number of erro layers below this one

	// Metadata
	id        string        // Error id
	class     ErrorClass    // Error class
	category  ErrorCategory // Error category
	severity  ErrorSeverity // Error severity
	retryable bool          // Retryable flag
	fields    []any         // Key-value fields
	span      TraceSpan     // Span
	created   time.Time     // Creation timestamp

	stack     rawStack                 // Stack trace (program counters only - resolved on demand)
	stackSkip int                      // Extra caller frames to hide (StackTraceWithSkip)
	frames    atomicValue[cachedStack] // Stack trace frames (cached per config)

	formatter        FormatErrorFunc
	stackTraceConfig *StackTraceConfig
}

// Error implements the error interface.
func (e *baseError) Error() (out string) {
	if fullMessage := e.fullMessage.Load(); fullMessage != "" {
		return fullMessage
	}
	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe error message
			out = fmt.Sprintf("error formatting failed: %v", r)
		}
		e.fullMessage.Store(out)
	}()

	if formatter := e.Formatter(); formatter != nil {
		out = formatter(e)
	}
	if out == "" {
		out = FormatErrorMessage(e)
	}
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
	return out
}

// Format implements [fmt.Formatter] for stack trace printing.
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the [errors.Unwrap] interface.
// When a foreign chain is preserved (state 2), it takes precedence so no
// intermediate layer is skipped.
func (e *baseError) Unwrap() error {
	if e.originalErr != nil {
		return e.originalErr
	}
	if e.wrappedErr != nil {
		return e.wrappedErr
	}
	return nil
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
	if target == nil {
		return false
	}
	targetErr, ok := target.(Error)
	if !ok {
		if e.originalErr != nil {
			return Is(e.originalErr, target)
		}
		if e.wrappedErr != nil {
			return e.wrappedErr.Is(target)
		}
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
	if e == nil || target == nil {
		return false
	}
	targetPtr, ok := target.(*Error)
	if ok && targetPtr != nil {
		*targetPtr = e
		return true
	}
	if e.originalErr != nil {
		// The full original chain contains any wrapped erro error too,
		// so foreign wrapper types stay findable.
		return As(e.originalErr, target)
	}
	if e.wrappedErr != nil {
		return e.wrappedErr.As(target)
	}
	return false
}

// MarshalJSON implements the [json.Marshaler] interface.
func (e *baseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrorToJSON(e))
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
// The same limits that apply at construction time (message length,
// field count) are enforced on deserialized data.
func (e *baseError) UnmarshalJSON(data []byte) error {
	var schema ErrorSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}
	fields := schema.Fields
	if len(fields) > maxPairsCount {
		fields = fields[:maxPairsCount]
	}
	if len(fields)%2 != 0 {
		fields = append(fields, MissingFieldPlaceholder)
	}
	e.originalErr = nil
	e.wrappedErr = nil
	e.depth = 0
	e.fullMessage = atomicValue[string]{}
	e.message = truncateString(schema.Message, MaxMessageLength)
	e.created = schema.Created
	e.span = nil
	e.stack = nil
	e.stackSkip = 0
	e.frames = atomicValue[cachedStack]{}
	e.formatter = FormatErrorWithFields
	e.id = truncateString(schema.ID, MaxKeyLength)
	e.class = schema.Class
	e.category = schema.Category
	e.severity = schema.Severity
	e.retryable = schema.Retryable
	e.fields = fields
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

// Message returns the error's message without fields. When the chain
// contains foreign (non-erro) layers, their full Error() text is included,
// since foreign errors expose no fields-free message.
func (e *baseError) Message() string {
	out := FormatErrorMessage(e)
	if unwrapped := e.Unwrap(); unwrapped != nil {
		var unwrappedMsg string
		if unwrappedErr, ok := unwrapped.(*baseError); ok {
			unwrappedMsg = unwrappedErr.Message()
		} else {
			unwrappedMsg = unwrapped.Error()
		}
		if unwrappedMsg == "" {
			return out
		}
		if out == "" {
			return unwrappedMsg
		}
		return out + ": " + unwrappedMsg
	}
	return out
}

// Fields returns the error's fields.
func (e *baseError) Fields() []any {
	if len(e.fields) == 0 {
		return nil // Get fields only from current level
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
	total := 0
	for p := e; p != nil; p = p.wrappedErr {
		total += len(p.fields)
	}
	fields := make([]any, 0, total)
	for p := e; p != nil; p = p.wrappedErr {
		fields = append(fields, p.fields...)
	}
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
	if e.stack == nil {
		return nil
	}
	if cached := e.frames.Load(); cached.frames != nil && cached.cfg == cfg {
		return cached.frames
	}
	frames := e.stack.toFrames(cfg, e.stackSkip)
	e.frames.Store(cachedStack{cfg: cfg, frames: frames})
	return frames
}

// cachedStack pairs resolved stack frames with the config they were
// resolved under, so a caller with a different config never receives
// frames rendered for another one.
type cachedStack struct {
	cfg    *StackTraceConfig
	frames Stack
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

	if wrapped, ok := errorToWrap.(*baseError); ok {
		e.wrappedErr = wrapped
	} else if As(errorToWrap, &e.wrappedErr) {
		// A foreign error contains an erro error deeper in its chain:
		// preserve the full chain and inherit metadata from the inner one.
		e.originalErr = errorToWrap
	} else {
		e.originalErr = errorToWrap
		e.created = time.Now()
	}

	if e.wrappedErr != nil {
		if e.wrappedErr.depth >= MaxWrapDepth {
			// Depth cap reached: drop this layer (including its metadata
			// and side-effect options) to keep traversal costs bounded.
			return e.wrappedErr
		}
		e.depth = e.wrappedErr.depth + 1
	}

	return applyMeta(e, meta...)
}

func applyMeta(e *baseError, meta ...any) *baseError {
	if len(meta) == 0 {
		if e.wrappedErr == nil {
			e.id = newID(e.created.UnixNano())
		}
		return e
	}

	preparedFields := make([]any, 0, getFieldsCapFromMeta(meta))
	for _, f := range meta {
		if f == nil {
			continue
		}
		switch val := f.(type) {
		case errorOpt:
			val(e)
		case errorFields:
			if fields := val(); len(fields) > 0 {
				preparedFields = append(preparedFields, fields...)
			}
		case ErrorClass:
			e.class = val
		case ErrorCategory:
			e.category = val
		case ErrorSeverity:
			e.severity = val
		case errorWork:
			continue
		default:
			preparedFields = append(preparedFields, val)
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
		e.id = newID(e.created.UnixNano())
	}

	for _, f := range meta {
		switch f := f.(type) {
		case errorWork:
			f(e)
		}
	}

	return e
}

func getFieldsCapFromMeta(meta []any) int {
	resultedCap := 0
	for _, f := range meta {
		switch f := f.(type) {
		case errorFields:
			resultedCap += len(f())
		case errorOpt, errorWork, ErrorClass, ErrorCategory, ErrorSeverity:
			continue
		default:
			resultedCap++
		}
	}
	if resultedCap%2 != 0 {
		resultedCap++
	}
	return resultedCap
}
