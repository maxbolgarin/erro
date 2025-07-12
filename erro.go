package erro

import (
	"fmt"
	"reflect"
)

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
		return newWrapError(erroErr, message, fields...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, fields...)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func WrapEmpty(err error) Error {
	if err == nil {
		return nil
	}

	// If it's already an erro error, create a wrap that points to its base
	if erroErr, ok := err.(Error); ok {
		return newWrapError(erroErr, "")
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, "")
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
		return newWrapError(erroErr, message, args...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, args...)
}

// Is reports whether any error in err's chain matches target
func Is(err error, target error) bool {
	if target == nil {
		return err == target
	}

	// Check current error
	if isComparable(err, target) {
		return true
	}

	// If error has Is method, use it
	if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}

	// Check wrapped errors
	if x, ok := err.(interface{ Unwrap() error }); ok {
		return Is(x.Unwrap(), target)
	}

	// For erro errors, also check the error chain via GetBase
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		if base != err && Is(base, target) {
			return true
		}
	}

	return false
}

// isComparable checks if two errors are directly comparable
func isComparable(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	// Check direct equality
	if err == target {
		return true
	}

	// For erro errors, delegate to their Is method
	if erroErr, ok := err.(Error); ok {
		return erroErr.Is(target)
	}

	return false
}

// As finds the first error in err's chain that matches target
func As(err error, target any) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}

	if err == nil {
		return false
	}

	// Check if target is a pointer
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr {
		panic("errors: target must be a non-nil pointer")
	}

	targetType := typ.Elem()

	// Check current error
	if reflect.TypeOf(err).AssignableTo(targetType) {
		val.Elem().Set(reflect.ValueOf(err))
		return true
	}

	// If error has As method, use it
	if x, ok := err.(interface{ As(any) bool }); ok && x.As(target) {
		return true
	}

	// Check wrapped errors
	if x, ok := err.(interface{ Unwrap() error }); ok {
		return As(x.Unwrap(), target)
	}

	// For erro errors, also check the error chain via GetBase
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		if base != err && As(base, target) {
			return true
		}
	}

	return false
}

// Unwrap returns the underlying error if this wraps an external error
func Unwrap(err error) error {
	if erroErr, ok := err.(Error); ok {
		base := erroErr.GetBase()
		baseInt, ok := base.(*baseError)
		if ok {
			return baseInt.originalErr
		}
	}
	return nil
}
