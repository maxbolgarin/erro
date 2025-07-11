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
	wrapPoint   uintptr    // Single PC for where this wrap occurred (not full stack!)
	createdAt   time.Time  // When this wrap was created
}

func (e *wrapError) Error() (out string) {
	out = buildFieldsMessage(e.wrapMessage, e.wrapFields)

	// Build the complete chain by getting the wrapped error's message
	var wrappedMsg string
	if e.wrapped != nil {
		wrappedMsg = e.wrapped.Error()
	} else {
		wrappedMsg = e.base.Error()
	}

	return out + ": " + wrappedMsg
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

// Unwrap implements the Unwrap interface
func (e *wrapError) Unwrap() error {
	return e.wrapped
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

func newWrapError(base *baseError, wrapped Error, message string, fields ...any) *wrapError {
	return &wrapError{
		base:        base,
		wrapped:     wrapped,
		wrapMessage: message,
		wrapFields:  prepareFields(fields),
		wrapPoint:   CaptureWrapPoint(3), // Skip Wrap, newWrapError and capture caller
		createdAt:   time.Now(),
	}
}
