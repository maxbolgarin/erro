package erro

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ErrorContext contains all extractable context from an error
type ErrorContext struct {
	Message   string          // Base error message
	Function  string          // Function where error was created
	Package   string          // Package where error was created
	File      string          // File where error was created
	Line      int             // Line where error was created
	Fields    map[string]any  // All key-value pairs
	Code      string          // Error code
	Category  string          // Error category
	Severity  string          // Error severity
	Tags      []string        // Error tags
	Retryable bool            // Whether error is retryable
	CreatedAt time.Time       // When error was created
	TraceID   string          // Trace ID if available
	Context   context.Context // Associated context
}

// Error represents the common interface for all erro errors
type Error interface {
	error

	// Chaining methods for building errors
	Code(code string) Error
	Category(category string) Error
	Severity(severity string) Error
	Fields(fields ...any) Error
	Context(ctx context.Context) Error
	Tags(tags ...string) Error
	Retryable(retryable bool) Error
	TraceID(traceID string) Error

	// Extraction methods
	GetBase() *baseError
	GetContext() context.Context
	GetCode() string
	GetCategory() string
	GetSeverity() string
	GetTags() []string
	IsRetryable() bool
	GetTraceID() string
	GetFields() []any

	// Stack trace access
	Stack() []StackFrame
	StackFormat() string
	ErrorWithStack() string
	Format(s fmt.State, verb rune)
}

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error     // Original error if wrapping external error
	message     string    // Base message
	stack       RawStack  // Stack trace (program counters only - resolved on demand)
	createdAt   time.Time // Creation timestamp

	// Metadata
	fields    []any           // Key-value fields
	code      string          // Error code
	category  string          // Error category
	severity  string          // Error severity
	tags      []string        // Tags
	retryable bool            // Retryable flag
	traceID   string          // Trace ID
	ctx       context.Context // Associated context
}

// wrapError is a lightweight wrapper that points to baseError
type wrapError struct {
	base        *baseError // Pointer to base error
	wrapped     Error      // The wrapped error (to get all its fields)
	wrapMessage string     // Wrap message
	wrapFields  []any      // Additional fields for this wrap
	wrapPoint   uintptr    // Single PC for where this wrap occurred (not full stack!)
	createdAt   time.Time  // When this wrap was created
}

// New creates a new error with optional fields
func New(message string, fields ...any) Error {
	return newBaseError(nil, message, fields...)
}

// countFormatVerbs counts the number of format verbs in a format string
func countFormatVerbs(format string) int {
	count := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			if i+1 < len(format) && format[i+1] != '%' {
				count++
				// Skip the verb character
				i++
			} else if i+1 < len(format) && format[i+1] == '%' {
				// Skip escaped %
				i++
			}
		}
	}
	return count
}

// Errorf creates a new error with formatted message and optional fields
func Errorf(message string, args ...any) Error {
	// Count format verbs in the message
	formats := countFormatVerbs(message)

	// If there are no format verbs, all args are fields
	if formats == 0 {
		return newBaseError(nil, message, args...)
	}
	if formats > len(args) {
		formats = len(args)
	}

	message = fmt.Sprintf(message, args[:formats]...)
	args = args[formats:]

	// Create a new error with the formatted message and remaining args as fields
	return newBaseError(nil, message, args...)
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string, fields ...any) Error {
	if err == nil {
		return New(message, fields...)
	}

	// If it's already an erro error, create a wrap that points to its base
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		return &wrapError{
			base:        base,
			wrapped:     erroErr,
			wrapMessage: message,
			wrapFields:  prepareFields(fields),
			wrapPoint:   CaptureWrapPoint(2), // Skip Wrap function and capture caller
			createdAt:   time.Now(),
		}
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, fields...)
}

// Wrapf wraps an existing error with formatted message and optional fields
func Wrapf(err error, message string, args ...any) Error {
	if err == nil {
		return Errorf(message, args...)
	}

	// Find where format args end and fields begin
	formats := countFormatVerbs(message)

	// If there are no format verbs, all args are fields
	if formats > 0 {
		if formats > len(args) {
			formats = len(args)
		}
		message = fmt.Sprintf(message, args[:formats]...)
		args = args[formats:]
	}

	// If it's already an erro error, create a wrap that points to its base
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		return &wrapError{
			base:        base,
			wrapped:     erroErr,
			wrapMessage: message,
			wrapFields:  prepareFields(args),
			wrapPoint:   CaptureWrapPoint(2), // Skip Wrapf function and capture caller
			createdAt:   time.Now(),
		}
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, args...)
}

// newBaseError creates a new base error
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return &baseError{
		originalErr: originalErr,
		message:     message,
		stack:       CaptureStack(3), // Skip New, newBaseError and caller
		createdAt:   time.Now(),
		fields:      prepareFields(fields),
	}
}

// Error implements the error interface
func (e *baseError) Error() string {
	return e.buildErrorMessage()
}

func (e *wrapError) Error() string {
	wrapMsg := buildFieldsMessage(e.wrapMessage, e.wrapFields)

	// Build the complete chain by getting the wrapped error's message
	var wrappedMsg string
	if e.wrapped != nil {
		wrappedMsg = e.wrapped.Error()
	} else {
		wrappedMsg = e.base.buildErrorMessage()
	}

	return wrapMsg + ": " + wrappedMsg
}

// buildErrorMessage constructs the error message with fields
func (e *baseError) buildErrorMessage() (out string) {
	out = buildFieldsMessage(e.message, e.fields)

	if e.originalErr != nil {
		// If wrapping external error, include it
		return out + ": " + e.originalErr.Error()
	}

	return out
}

// buildFieldsMessage creates message with fields appended
func buildFieldsMessage(message string, fields []any) string {
	if len(fields) == 0 {
		return message
	}

	var builder strings.Builder

	// Estimate capacity: message + fields with reasonable estimates for key=value pairs
	// Each field pair needs: space + key + "=" + value (estimate ~20 chars per pair)
	estimatedSize := len(message) + (len(fields)/2)*20
	builder.Grow(estimatedSize)

	builder.WriteString(message)

	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}
		builder.WriteString(" ")
		builder.WriteString(valueToString(fields[i]))
		builder.WriteString("=")
		builder.WriteString(valueToString(fields[i+1]))
	}

	return builder.String()
}

// prepareFields ensures fields come in key-value pairs
func prepareFields(fields []any) []any {
	if len(fields)%2 != 0 {
		return append(fields, "<missing>")
	}
	return fields
}

// Chaining methods for baseError
func (e *baseError) Code(code string) Error {
	e.code = code
	return e
}

func (e *baseError) Category(category string) Error {
	e.category = category
	return e
}

func (e *baseError) Severity(severity string) Error {
	e.severity = severity
	return e
}

func (e *baseError) Fields(fields ...any) Error {
	e.fields = append(e.fields, prepareFields(fields)...)
	return e
}

func (e *baseError) Context(ctx context.Context) Error {
	e.ctx = ctx
	return e
}

func (e *baseError) Tags(tags ...string) Error {
	e.tags = append(e.tags, tags...)
	return e
}

func (e *baseError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) TraceID(traceID string) Error {
	e.traceID = traceID
	return e
}

// Chaining methods for wrapError - these modify the base error
func (e *wrapError) Code(code string) Error {
	e.base.code = code
	return e
}

func (e *wrapError) Category(category string) Error {
	e.base.category = category
	return e
}

func (e *wrapError) Severity(severity string) Error {
	e.base.severity = severity
	return e
}

func (e *wrapError) Fields(fields ...any) Error {
	e.wrapFields = append(e.wrapFields, prepareFields(fields)...)
	return e
}

func (e *wrapError) Context(ctx context.Context) Error {
	e.base.ctx = ctx
	return e
}

func (e *wrapError) Tags(tags ...string) Error {
	e.base.tags = append(e.base.tags, tags...)
	return e
}

func (e *wrapError) Retryable(retryable bool) Error {
	e.base.retryable = retryable
	return e
}

func (e *wrapError) TraceID(traceID string) Error {
	e.base.traceID = traceID
	return e
}

// Getter methods for baseError
func (e *baseError) GetBase() *baseError         { return e }
func (e *baseError) GetContext() context.Context { return e.ctx }
func (e *baseError) GetCode() string             { return e.code }
func (e *baseError) GetCategory() string         { return e.category }
func (e *baseError) GetSeverity() string         { return e.severity }
func (e *baseError) GetTags() []string           { return e.tags }
func (e *baseError) IsRetryable() bool           { return e.retryable }
func (e *baseError) GetTraceID() string          { return e.traceID }
func (e *baseError) GetFields() []any            { return e.fields }
func (e *baseError) Stack() []StackFrame         { return e.stack.ToFrames() }
func (e *baseError) StackFormat() string         { return e.stack.FormatFull() }
func (e *baseError) ErrorWithStack() string {
	return e.Error() + "\n" + e.StackFormat()
}

// Getter methods for wrapError
func (e *wrapError) GetBase() *baseError         { return e.base }
func (e *wrapError) GetContext() context.Context { return e.base.ctx }
func (e *wrapError) GetCode() string             { return e.base.code }
func (e *wrapError) GetCategory() string         { return e.base.category }
func (e *wrapError) GetSeverity() string         { return e.base.severity }
func (e *wrapError) GetTags() []string           { return e.base.tags }
func (e *wrapError) IsRetryable() bool           { return e.base.retryable }
func (e *wrapError) GetTraceID() string          { return e.base.traceID }
func (e *wrapError) ErrorWithStack() string      { return e.Error() + "\n" + e.StackFormat() }

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

func (e *wrapError) Stack() []StackFrame {
	// Add our wrap point first
	var wrapFrame StackFrame
	if e.wrapPoint != 0 {
		wrapFrame = ResolveWrapPoint(e.wrapPoint)
	}

	// If we're wrapping another erro error, get its stack (which may include other wrap points)
	var framesToAdd []StackFrame
	if e.wrapped != nil {
		framesToAdd = e.wrapped.Stack()
	} else {
		// Fallback to base stack if no wrapped error
		framesToAdd = e.base.stack.ToFrames()
	}

	hasWrapFrame := 0
	if wrapFrame.Name != "" {
		hasWrapFrame = 1
	}

	frames := make([]StackFrame, len(framesToAdd)+hasWrapFrame)
	if hasWrapFrame == 1 {
		frames[0] = wrapFrame
	}

	if len(framesToAdd) > 0 {
		copy(frames[1:], framesToAdd)
	}

	return frames
}

func (e *wrapError) StackFormat() string {
	// Add our wrap point first
	var wrapFrame StackFrame
	if e.wrapPoint != 0 {
		wrapFrame = ResolveWrapPoint(e.wrapPoint)
	}

	// If we're wrapping another erro error, get its stack (which may include other wrap points)
	var framesToAdd string
	if e.wrapped != nil {
		framesToAdd = e.wrapped.StackFormat()
	} else {
		// Fallback to base stack if no wrapped error
		framesToAdd = e.base.stack.FormatFull()
	}

	if wrapFrame.Name != "" {
		return wrapFrame.FormatFull() + "\n" + framesToAdd
	}
	return framesToAdd
}

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Print with stack trace
			fmt.Fprint(s, e.Error())
			fmt.Fprint(s, "\nStack trace:\n")
			for _, frame := range e.stack.ToFrames() {
				fmt.Fprintf(s, "\t%s.%s\n\t\t%s:%d\n", frame.Package, frame.Name, frame.File, frame.Line)
			}
		} else {
			fmt.Fprint(s, e.Error())
		}
	case 's':
		fmt.Fprint(s, e.Error())
	}
}

func (e *wrapError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Print with complete stack trace
			fmt.Fprint(s, e.Error())
			fmt.Fprint(s, "\nStack trace:\n")
			for _, frame := range e.Stack() {
				fmt.Fprintf(s, "\t%s.%s\n\t\t%s:%d\n", frame.Package, frame.Name, frame.File, frame.Line)
			}
		} else {
			fmt.Fprint(s, e.Error())
		}
	case 's':
		fmt.Fprint(s, e.Error())
	}
}

// ExtractContext extracts all available context from an error
func ExtractContext(err error) *ErrorContext {
	if err == nil {
		return nil
	}

	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()

		// Extract fields as map
		fields := make(map[string]any)
		allFields := erroErr.GetFields()
		for i := 0; i < len(allFields); i += 2 {
			if i+1 < len(allFields) {
				key := fmt.Sprintf("%v", allFields[i])
				fields[key] = allFields[i+1]
			}
		}

		// Extract origin context from stack on demand
		var function, pkg, file string
		var line int

		if !base.stack.IsEmpty() {
			// Find the first user code frame for function context
			stackFrames := base.stack.ToFrames()
			stackType := Stack(stackFrames)
			if topUserFrame := stackType.TopUserFrame(); topUserFrame != nil {
				function = topUserFrame.Name
				pkg = topUserFrame.Package
				file = topUserFrame.File
				line = topUserFrame.Line
			} else if len(stackFrames) > 0 {
				// Fallback to first frame if no user code found
				frame := stackFrames[0]
				function = frame.Name
				pkg = extractPackage(frame.Name)
				file = frame.File
				line = frame.Line
			}
		}

		return &ErrorContext{
			Message:   base.message,
			Function:  function,
			Package:   pkg,
			File:      file,
			Line:      line,
			Fields:    fields,
			Code:      base.code,
			Category:  base.category,
			Severity:  base.severity,
			Tags:      base.tags,
			Retryable: base.retryable,
			CreatedAt: base.createdAt,
			TraceID:   base.traceID,
			Context:   base.ctx,
		}
	}

	// For non-erro errors, create basic context
	return &ErrorContext{
		Message: err.Error(),
		Fields:  make(map[string]any),
	}
}

// captureStack is now handled by the enhanced CaptureStack function in stack.go
func captureStack(skip int) []StackFrame {
	return CaptureStack(skip).ToFrames()
}

// extractPackage extracts package name from function name
func extractPackage(functionName string) string {
	// Function names are like "github.com/user/repo/package.function"
	lastSlash := strings.LastIndex(functionName, "/")
	if lastSlash == -1 {
		return ""
	}

	afterSlash := functionName[lastSlash+1:]
	dot := strings.Index(afterSlash, ".")
	if dot == -1 {
		return afterSlash
	}

	return afterSlash[:dot]
}

// Helper functions for common error operations

// Is reports whether any error in err's chain matches target
func Is(err error, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	// Check direct equality
	if err.Error() == target.Error() {
		return true
	}

	// Check if target is an erro error
	if targeterro, ok := target.(Error); ok {
		targetBase := targeterro.GetBase()

		// Check against our error
		if erroErr, ok := err.(Error); ok {
			errBase := erroErr.GetBase()
			return errBase.message == targetBase.message && errBase.code == targetBase.code
		}
	}

	return false
}

// As finds the first error in err's chain that matches target
func As(err error, target any) bool {
	if err == nil {
		return false
	}

	// Try direct assignment
	if erroErr, ok := err.(Error); ok {
		if targetPtr, ok := target.(*Error); ok {
			*targetPtr = erroErr
			return true
		}
	}

	return false
}

// Unwrap returns the underlying error if this wraps an external error
func Unwrap(err error) error {
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		return base.originalErr
	}
	return nil
}
