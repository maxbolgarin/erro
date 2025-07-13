package erro

import (
	"fmt"
	"reflect"
)

// New creates a new error with optional fields
func New(message string, fields ...any) Error {
	return newBaseError(nil, message, fields...)
}

// Newf creates a new error with formatted message and optional fields
func Newf(message string, args ...any) Error {
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
	if erroErr, ok := err.(Error); ok && erroErr != nil {
		// TODO: handle light
		return newWrapError(erroErr, message, fields...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, fields...)
}

// Wrapf wraps an existing error with formatted message and optional fields
func Wrapf(err error, message string, args ...any) Error {
	if err == nil {
		return Newf(message, args...)
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
	if erroErr, ok := err.(Error); ok && erroErr != nil {
		return newWrapError(erroErr, message, args...)
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, message, args...)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func WrapEmpty(err error) Error {
	if err == nil {
		return nil
	}

	// If it's already an erro error, create a wrap that points to its base
	if erroErr, ok := err.(Error); ok && erroErr != nil {
		return newWrapError(erroErr, "")
	}

	// For external errors, create a new base error that wraps it
	return newBaseError(err, "")
}

// Is reports whether any error in err's chain matches target
func Is(err error, target error) (ok bool) {
	if target == nil {
		return err == target
	}

	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

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
		return erroErr.Context().Is(target)
	}

	return false
}

// As finds the first error in err's chain that matches target
func As(err error, target any) (ok bool) {
	if target == nil {
		return false
	}

	if err == nil {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			// Return false instead of panicking
			ok = false
		}
	}()

	// Check if target is a pointer
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr {
		return false
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

	return false
}

// Unwrap returns the underlying error if this wraps an external error
func Unwrap(err error) error {
	if erroErr, ok := err.(Error); ok {
		base := erroErr.Context().BaseError()
		baseInt, ok := base.(*baseError)
		if ok {
			return baseInt.originalErr
		}
		return erroErr.Unwrap()
	}
	return nil
}

func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &multiError{
		errors: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errors = append(e.errors, err)
		}
	}
	return e
}
