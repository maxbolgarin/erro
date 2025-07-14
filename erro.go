package erro

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func NewSentinel(message string) Error {
	err := &lightError{
		message:   truncateString(message, maxMessageLength),
		formatter: FormatErrorMessage,
		id:        fmt.Sprintf("%x", md5.Sum([]byte(message)))[:12],
	}
	return err
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
	_, ok := err.(*lightError)
	return ok
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

	if erroErr != nil {
		class := erroErr.Context().Class()
		category := erroErr.Context().Category()
		message := erroErr.Context().Message()
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
			// Fallback to message if still unknown
			if status == http.StatusInternalServerError && message != "" {
				status = statusFromMessage(message)
			}
		}
	} else {
		// Not an erro error, use message string matching
		status = statusFromMessage(err.Error())
	}

	return status
}

// statusFromMessage tries to guess HTTP status code from error message
func statusFromMessage(msg string) int {
	msg = strings.ToLower(msg)
	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "already exists"):
		return http.StatusConflict
	case strings.Contains(msg, "permission denied"):
		return http.StatusForbidden
	case strings.Contains(msg, "unauthenticated"), strings.Contains(msg, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(msg, "timeout"):
		return http.StatusGatewayTimeout
	case strings.Contains(msg, "conflict"):
		return http.StatusConflict
	case strings.Contains(msg, "rate limit"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "temporary"), strings.Contains(msg, "unavailable"):
		return http.StatusServiceUnavailable
	case strings.Contains(msg, "validation"), strings.Contains(msg, "invalid"), strings.Contains(msg, "bad request"):
		return http.StatusBadRequest
	case strings.Contains(msg, "cancelled"), strings.Contains(msg, "canceled"):
		return 499 // Client Closed Request (non-standard)
	case strings.Contains(msg, "not implemented"):
		return http.StatusNotImplemented
	case strings.Contains(msg, "security") || strings.Contains(msg, "forbidden"):
		return http.StatusForbidden
	case strings.Contains(msg, "critical"), strings.Contains(msg, "internal"):
		return http.StatusInternalServerError
	case strings.Contains(msg, "external") || strings.Contains(msg, "bad gateway"):
		return http.StatusBadGateway
	case strings.Contains(msg, "data loss"):
		return http.StatusInternalServerError
	case strings.Contains(msg, "resource exhausted"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "payment"):
		return http.StatusPaymentRequired
	case strings.Contains(msg, "storage"):
		return http.StatusInsufficientStorage
	case strings.Contains(msg, "processing"):
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}
