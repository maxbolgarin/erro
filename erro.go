package erro

import (
	"context"
	"errors"
	"io"
	"net/http"
)

// New creates a new error with optional fields
func New(message string, fields ...any) Error {
	return newf(message, fields...)
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string, fields ...any) Error {
	if err == nil {
		return nil
	}
	return wrapf(err, message, fields...)
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

func newf(message string, meta ...any) *baseError {
	if len(meta) == 0 {
		return newBaseError(message)
	}
	message, meta = ApplyFormatVerbs(message, meta...)
	return newBaseError(message, meta...)
}

func wrapf(err error, message string, meta ...any) *baseError {
	message, meta = ApplyFormatVerbs(message, meta...)
	return newWrapError(err, message, meta...)
}
