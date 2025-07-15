// Package erro provides a comprehensive error handling library for Go, offering
// features like error wrapping, structured logging, stack traces, and error
// classification. It is designed to be a drop-in replacement for the standard
// `errors` package, providing more context and control over error handling.
package erro

import (
	"context"
	"errors"
	"io"
	"net/http"
)

// New creates a new [Error] with a message and optional structured metadata.
//
// This is the primary function for creating rich, structured errors with comprehensive
// context and metadata. The returned error implements the standard error interface while
// providing extensive additional functionality for structured logging, error classification,
// stack traces, tracing integration, and more.
//
// # Basic Usage
//
//	err := erro.New("user not found")
//	err := erro.New("connection failed", "host", "localhost", "port", 5432)
//
// # Error Classification
//
// Classify errors using predefined classes, categories, and severity levels:
//
//	err := erro.New("invalid email format",
//	    erro.ClassValidation,        // Error class for HTTP status mapping
//	    erro.CategoryUserInput,      // Error category for organization
//	    erro.SeverityLow,           // Severity level for prioritization
//	)
//
// # Structured Logging
//
// Add context through key-value fields that integrate seamlessly with structured loggers:
//
//	err := erro.New("database query failed",
//	    "table", "users",
//	    "operation", "SELECT",
//	    "duration_ms", 1500,
//	    "rows_affected", 0,
//	)
//
//	// Use with slog
//	slog.Error("Operation failed", erro.LogFields(err)...)
//
//	// Use with logrus
//	logrus.WithFields(erro.LogFieldsMap(err)).Error("Operation failed")
//
// # Advanced Configuration
//
// Use functional options for advanced error configuration:
//
//	err := erro.New("payment processing failed",
//	    "amount", 29.99,
//	    "currency", "USD",
//	    "merchant_id", "merchant_123",
//
//	    erro.ID("payment_error_001"),        // Custom error ID for tracking
//	    erro.Retryable(),                    // Mark as retryable for retry logic
//	    erro.StackTrace(),                   // Capture stack trace for debugging
//	    erro.ClassExternal,                  // External service error
//	    erro.CategoryPayment,                // Payment-related error
//	    erro.SeverityCritical,              // Critical severity level
//
//	    erro.RecordSpan(span),              // Record in tracing span
//	    erro.RecordMetrics(metricsCollector), // Send to metrics system
//	    erro.SendEvent(ctx, eventDispatcher), // Dispatch error event
//	    erro.Formatter(customFormatter),     // Custom error message formatting
//	)
//
// # JSON Serialization
//
// Errors can be serialized for API responses or database storage:
//
//	jsonData, _ := json.Marshal(err)
//	// Contains: id, class, category, severity, message, fields, stack_trace, etc.
//
// # Format Verb Support
//
// The message supports format verbs when additional arguments are provided:
//
//	err := erro.New("user %s not found in database %s", "john_doe", "users_db",
//	    "query_time_ms", 150,
//	    erro.ClassNotFound,
//	)
//	// Message becomes: "user john_doe not found in database users_db"
//
// # Some Notes
//
// - Stack traces are captured only when explicitly requested with [StackTrace] or [StackTraceWithSkip]
// - Field values are truncated to prevent memory exhaustion
// - Error IDs are automatically generated for tracking without performance impact
// - Structured fields are validated and limited to prevent DoS attacks
//
// # Thread Safety
//
// Individual errors are immutable after creation and safe for concurrent use.
// For collecting multiple errors concurrently, use [NewSafeList] or [NewSafeSet].
func New(message string, fields ...any) Error {
	return newf(message, fields...)
}

// Wrap wraps an existing error with additional context, message, and optional metadata.
//
// This function preserves the original error while adding a new layer of context,
// making it essential for error propagation across application layers. The wrapped
// error maintains full access to all original metadata while allowing additional
// context to be attached at each layer.
//
// If the error to be wrapped is nil, Wrap returns nil, making it safe to use in
// conditional error handling patterns without explicit nil checks.
//
// # Basic Usage
//
//	if err != nil {
//	    return erro.Wrap(err, "failed to process user request")
//	}
//
//	// With additional context
//	if err != nil {
//	    return erro.Wrap(err, "database query failed",
//	        "table", "users",
//	        "operation", "UPDATE",
//	    )
//	}
//
// # Advanced Wrapping Features
//
// Enhance wrapped errors with comprehensive metadata and integrations:
//
//	err := erro.Wrap(dbErr, "user authentication failed",
//	    "username", username,
//	    "login_attempt", attemptCount,
//	    "ip_address", clientIP,
//	    "user_agent", erro.Redact(userAgent), // Redact potentially sensitive data
//
//	    erro.ClassUnauthenticated,           // Authentication failure
//	    erro.CategoryAuth,                   // Authentication category
//	    erro.SeverityMedium,                // Medium severity
//	    erro.ID("auth_failure_001"),         // Custom error ID for tracking
//
//	    erro.StackTrace(),                   // Capture new stack trace at wrap point
//	    erro.RecordSpan(span),              // Record in distributed tracing
//	    erro.RecordMetrics(authMetrics),     // Send to authentication metrics
//	    erro.SendEvent(ctx, securityEvents), // Dispatch to security monitoring
//	)
//
// # Format Verb Support
//
// Use format verbs in wrap messages for dynamic context:
//
//	err := erro.Wrap(originalErr, "failed to process %s for user %d",
//	    operation, userID,
//	    "retry_count", retryCount,
//	    "processing_time_ms", duration.Milliseconds(),
//	)
//
// # Error Chain Inspection
//
// Access the complete error chain and metadata:
//
//	wrappedErr := erro.Wrap(originalErr, "service layer failure")
//
//	// Access original error
//	originalErr := errors.Unwrap(wrappedErr)
//
//	// Check error types in chain
//	if errors.Is(wrappedErr, sql.ErrNoRows) {
//	    // Handle specific database error
//	}
//
//	// Access all accumulated fields from error chain
//	allFields := wrappedErr.AllFields()
//
//	// Get inherited metadata from original error
//	errorClass := wrappedErr.Class()        // Inherits from original if not overridden
//	errorCategory := wrappedErr.Category()  // Inherits from original if not overridden
//	errorID := wrappedErr.ID()             // Inherits from original error
//
// # Performance and Memory Considerations
//
//   - Error wrapping is lightweight and optimized for high-frequency use
//   - Original error references are preserved efficiently without deep copying
//   - Field values are automatically truncated to prevent memory exhaustion
//   - Stack traces are captured only when explicitly requested with [StackTrace] or [StackTraceWithSkip]
//
// # Thread Safety
//
// Wrapped errors are immutable and safe for concurrent access across goroutines.
// The wrapping operation itself is thread-safe and can be called concurrently.
func Wrap(err error, message string, fields ...any) Error {
	if err == nil {
		return nil
	}
	return wrapf(err, message, fields...)
}

// Close is a utility function that closes an io.Closer and wraps any
// resulting error. It is intended to be used in defer statements.
//
// If the closer is nil, it does nothing. If the close operation returns an
// error, it will be wrapped with the provided message and fields.
//
// Example:
//
//	func ReadData(path string) (data []byte, err error) {
//	    f, err := os.Open(path)
//	    if err != nil {
//	        return nil, erro.Wrap(err, "failed to open file")
//	    }
//	    defer erro.Close(&err, f, "failed to close file")
//
//	    // ... read data from f
//	}
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

// Shutdown is a utility function that executes a shutdown function and wraps
// any resulting error. It is intended for use in cleanup operations.
//
// If the shutdown function is nil, it does nothing. If the shutdown function
// returns an error, it will be wrapped with the provided message and fields.
//
// Example:
//
//	func (s *Server) Stop(ctx context.Context) (err error) {
//	    defer erro.Shutdown(ctx, &err, s.grpcServer.GracefulStop, "failed to stop gRPC server")
//	    // ... other shutdown logic
//	}
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

// Is reports whether any error in err's chain matches target.
// It is a drop-in replacement for the standard `errors.Is` function.
func Is(err error, target error) (ok bool) {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
// It is a drop-in replacement for the standard `errors.As` function.
func As(err error, target any) (ok bool) {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// It is a drop-in replacement for the standard `errors.Unwrap` function.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// It is a drop-in replacement for the standard `errors.Join` function.
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

// HTTPCode returns an appropriate HTTP status code for a given error.
//
// This function provides automatic HTTP status code mapping based on error classification,
// enabling seamless integration between your application's error handling and HTTP responses.
//
// # Error Class Mappings
//
// The following error classes map to specific HTTP status codes:
//
//	ClassValidation        -> 400 Bad Request
//	ClassNotFound          -> 404 Not Found
//	ClassAlreadyExists     -> 409 Conflict
//	ClassPermissionDenied  -> 403 Forbidden
//	ClassUnauthenticated   -> 401 Unauthorized
//	ClassTimeout           -> 504 Gateway Timeout
//	ClassConflict          -> 409 Conflict
//	ClassRateLimited       -> 429 Too Many Requests
//	ClassTemporary         -> 503 Service Unavailable
//	ClassUnavailable       -> 503 Service Unavailable
//	ClassInternal          -> 500 Internal Server Error
//	ClassCancelled         -> 499 Client Closed Request (non-standard)
//	ClassNotImplemented    -> 501 Not Implemented
//	ClassSecurity          -> 403 Forbidden
//	ClassCritical          -> 500 Internal Server Error
//	ClassExternal          -> 502 Bad Gateway
//	ClassDataLoss          -> 500 Internal Server Error
//	ClassResourceExhausted -> 429 Too Many Requests
//
// # Error Category Fallback Mappings
//
// When no error class is specified, categories provide fallback mappings:
//
//	CategoryUserInput      -> 400 Bad Request
//	CategoryAuth           -> 401 Unauthorized
//	CategoryDatabase       -> 500 Internal Server Error
//	CategoryNetwork        -> 502 Bad Gateway
//	CategoryAPI            -> 502 Bad Gateway
//	CategoryBusinessLogic  -> 422 Unprocessable Entity
//	CategoryCache          -> 503 Service Unavailable
//	CategoryConfig         -> 500 Internal Server Error
//	CategoryExternal       -> 502 Bad Gateway
//	CategorySecurity       -> 403 Forbidden
//	CategoryPayment        -> 402 Payment Required
//	CategoryStorage        -> 507 Insufficient Storage
//	CategoryProcessing     -> 422 Unprocessable Entity
//	CategoryAnalytics      -> 500 Internal Server Error
//	CategoryAI             -> 500 Internal Server Error
//	CategoryMonitoring     -> 500 Internal Server Error
//	CategoryNotifications  -> 500 Internal Server Error
//	CategoryEvents         -> 500 Internal Server Error
//
// # Basic Usage
//
//	// Using error classes
//	err := erro.New("user not found", erro.ClassNotFound)
//	statusCode := erro.HTTPCode(err) // Returns 404
//
//	err := erro.New("invalid email format", erro.ClassValidation)
//	statusCode := erro.HTTPCode(err) // Returns 400
//
//	// Using error categories as fallback
//	err := erro.New("database connection failed", erro.CategoryDatabase)
//	statusCode := erro.HTTPCode(err) // Returns 500
//
// # REST API Error Handling
//
// Build consistent REST API error responses:
//
//	// Validation errors -> 400 Bad Request
//	err := erro.New("validation failed",
//	    "field", "email",
//	    "reason", "invalid format",
//	    erro.ClassValidation,
//	)
//	statusCode := erro.HTTPCode(err) // 400
//
//	// Authentication errors -> 401 Unauthorized
//	err := erro.New("invalid credentials",
//	    "login_attempt", 3,
//	    erro.ClassUnauthenticated,
//	)
//	statusCode := erro.HTTPCode(err) // 401
//
//	// Authorization errors -> 403 Forbidden
//	err := erro.New("insufficient permissions",
//	    "required_role", "admin",
//	    "user_role", "user",
//	    erro.ClassPermissionDenied,
//	)
//	statusCode := erro.HTTPCode(err) // 403
//
//	// Resource not found -> 404 Not Found
//	err := erro.New("user not found",
//	    "user_id", 12345,
//	    erro.ClassNotFound,
//	)
//	statusCode := erro.HTTPCode(err) // 404
//
//	// Rate limiting -> 429 Too Many Requests
//	err := erro.New("rate limit exceeded",
//	    "limit", 100,
//	    "window", "1h",
//	    erro.ClassRateLimited,
//	)
//	statusCode := erro.HTTPCode(err) // 429
//
// # Middleware Integration
//
// Use with HTTP middleware for centralized error handling:
//
//	func ErrorMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        defer func() {
//	            if err := recover(); err != nil {
//	                var httpErr error
//	                if e, ok := err.(error); ok {
//	                    httpErr = e
//	                } else {
//	                    httpErr = fmt.Errorf("panic: %v", err)
//	                }
//
//	                statusCode := erro.HTTPCode(httpErr)
//	                http.Error(w, httpErr.Error(), statusCode)
//	            }
//	        }()
//	        next.ServeHTTP(w, r)
//	    })
//	}
//
// # Some Notes
//
// - If the error is nil, it returns 200 OK
// - If the error is not an [Error], it returns 500 Internal Server Error
// - If the error is an [Error], it returns the appropriate HTTP status code based on the error class and category
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
