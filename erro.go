package erro

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// New creates a new error with optional fields
func New(message string, fields ...any) Error {
	return newf(newBaseError, message, fields...)
}

func NewWithStack(message string, fields ...any) Error {
	return newf(newBaseErrorWithStack, message, fields...)
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string, fields ...any) Error {
	return wrap(newBaseError, err, message, fields...)
}

func WrapWithStack(err error, message string, fields ...any) Error {
	return wrap(newBaseErrorWithStack, err, message, fields...)
}

func Close(err *error, cl io.Closer, msg string, fields ...any) {
	if cl == nil {
		return
	}
	errClose := cl.Close()
	if errClose == nil {
		return
	}
	if err != nil && *err == nil {
		*err = Wrap(errClose, msg, fields...)
	}
}

func Shutdown(ctx context.Context, err *error, sd func(ctx context.Context) error, msg string, fields ...any) {
	if sd == nil {
		return
	}
	errClose := sd(ctx)
	if errClose == nil {
		return
	}
	if err != nil && *err == nil {
		*err = Wrap(errClose, msg, fields...)
	}
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

// IsLight checks if any error is a lightweight error
func IsLight(err error) bool {
	errBase, ok := err.(*baseError)
	return ok && errBase.stack == nil
}

func HTTPCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	status := http.StatusInternalServerError
	var erroErr Error
	if e, ok := err.(Error); ok && e != nil {
		erroErr = e
	}

	if erroErr == nil {
		return status
	}

	class := erroErr.Class()
	category := erroErr.Category()
	switch class {
	case ClassValidation:
		status = http.StatusBadRequest
	case ClassNotFound:
		status = http.StatusNotFound
	case ClassAlreadyExists:
		status = http.StatusConflict
	case ClassPermissionDenied:
		status = http.StatusForbidden
	case ClassUnauthenticated:
		status = http.StatusUnauthorized
	case ClassTimeout:
		status = http.StatusGatewayTimeout
	case ClassConflict:
		status = http.StatusConflict
	case ClassRateLimited:
		status = http.StatusTooManyRequests
	case ClassTemporary:
		status = http.StatusServiceUnavailable
	case ClassUnavailable:
		status = http.StatusServiceUnavailable
	case ClassInternal:
		status = http.StatusInternalServerError
	case ClassCancelled:
		status = 499 // Client Closed Request (non-standard)
	case ClassNotImplemented:
		status = http.StatusNotImplemented
	case ClassSecurity:
		status = http.StatusForbidden
	case ClassCritical:
		status = http.StatusInternalServerError
	case ClassExternal:
		status = http.StatusBadGateway
	case ClassDataLoss:
		status = http.StatusInternalServerError
	case ClassResourceExhausted:
		status = http.StatusTooManyRequests
	default:
		// Try category if class is unknown
		switch category {
		case CategoryUserInput:
			status = http.StatusBadRequest
		case CategoryAuth:
			status = http.StatusUnauthorized
		case CategoryDatabase:
			status = http.StatusInternalServerError
		case CategoryNetwork:
			status = http.StatusBadGateway
		case CategoryAPI:
			status = http.StatusBadGateway
		case CategoryBusinessLogic:
			status = http.StatusUnprocessableEntity
		case CategoryCache:
			status = http.StatusServiceUnavailable
		case CategoryConfig:
			status = http.StatusInternalServerError
		case CategoryExternal:
			status = http.StatusBadGateway
		case CategorySecurity:
			status = http.StatusForbidden
		case CategoryPayment:
			status = http.StatusPaymentRequired
		case CategoryStorage:
			status = http.StatusInsufficientStorage
		case CategoryProcessing:
			status = http.StatusUnprocessableEntity
		case CategoryAnalytics:
			status = http.StatusInternalServerError
		case CategoryAI:
			status = http.StatusInternalServerError
		case CategoryMonitoring:
			status = http.StatusInternalServerError
		case CategoryNotifications:
			status = http.StatusInternalServerError
		case CategoryEvents:
			status = http.StatusInternalServerError
		}
	}

	return status
}

func newf(errConstructor func(err error, message string, fields ...any) *baseError, message string, args ...any) *baseError {
	// Count format verbs in the message
	formats := countFormatVerbs(message)

	// If there are no format verbs, all args are fields
	if formats == 0 {
		return errConstructor(nil, message, args...)
	}
	if formats > len(args) {
		formats = len(args)
	}

	message = fmt.Sprintf(message, args[:formats]...)
	args = args[formats:]

	// Create a new error with the formatted message and remaining args as fields
	return errConstructor(nil, message, args...)
}

func wrap(errConstructor func(err error, message string, fields ...any) *baseError, err error, message string, fields ...any) *baseError {
	if err == nil {
		return nil
	}

	// Find where format args end and fields begin
	formats := countFormatVerbs(message)

	// If there are no format verbs, all args are fields
	if formats > 0 {
		if formats > len(fields) {
			formats = len(fields)
		}
		message = fmt.Sprintf(message, fields[:formats]...)
		fields = fields[formats:]
	}

	// If it's already an erro error, create a wrap that points to its base
	if erroErr, ok := err.(*baseError); ok && erroErr != nil {
		return newWrapError(erroErr, message, fields...)
	}

	// For external errors, create a new base error that wraps it
	return errConstructor(err, message, fields...)
}
