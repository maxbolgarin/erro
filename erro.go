package erro

import (
	"errors"
	"fmt"
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
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target any) (ok bool) {
	return errors.As(err, target)
}

// Unwrap returns the underlying error if this wraps an external error
func Unwrap(err error) error {
	return errors.Unwrap(err)
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
