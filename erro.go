package erro

import (
	"context"
	"fmt"
)

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

// New creates a new error with optional fields
func New(message string, fields ...any) Error {
	return newBaseError(nil, message, fields...)
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
		return newWrapError(base, erroErr, message, fields...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, fields...)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func WrapEmpty(err error) Error {
	if err == nil {
		return nil
	}
	return newBaseError(err, err.Error(), nil)
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
		return newWrapError(base, erroErr, message, args...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, args...)
}

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
