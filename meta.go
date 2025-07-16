package erro

import "context"

type (
	errorOpt    func(err *baseError)
	errorWork   func(err Error)
	errorFields func() []any
)

// ID sets a custom identifier for the error.
func ID(id string) errorOpt {
	return func(err *baseError) {
		err.id = id
	}
}

// Retryable marks the error as retryable.
func Retryable() errorOpt {
	return func(err *baseError) {
		err.retryable = true
	}
}

// Fields adds structured data to the error.
func Fields(fields ...any) errorFields {
	return func() []any {
		return fields
	}
}

// Formatter sets a custom error message formatter.
func Formatter(f FormatErrorFunc) errorOpt {
	return func(err *baseError) {
		if f == nil {
			return
		}
		err.formatter = f
	}
}

// StackTrace captures a stack trace for the error.
func StackTrace(c ...*StackTraceConfig) errorOpt {
	return func(err *baseError) {
		err.stack = captureStack(5)
		if len(c) > 0 {
			err.stackTraceConfig = c[0]
		} else {
			err.stackTraceConfig = nil
		}
	}
}

// StackTraceWithSkip captures a stack trace, skipping a specified number of frames.
func StackTraceWithSkip(skip int, c ...*StackTraceConfig) errorOpt {
	return func(err *baseError) {
		err.stack = captureStack(skip)
		if len(c) > 0 {
			err.stackTraceConfig = c[0]
		} else {
			err.stackTraceConfig = nil
		}
	}
}

// RecordSpan records the error in a tracing span.
func RecordSpan(s TraceSpan) errorWork {
	return func(err Error) {
		if s == nil {
			return
		}
		s.RecordError(err)
		if base, ok := err.(*baseError); ok {
			s.SetAttributes(base.fields...)
			base.span = s
			return
		}
		s.SetAttributes(err.Fields()...)
	}
}

// RecordMetrics records the error with a metrics collector.
func RecordMetrics(m ErrorMetrics) errorWork {
	return func(err Error) {
		if m == nil {
			return
		}
		m.RecordError(err)
	}
}

// SendEvent sends the error to an event dispatcher.
func SendEvent(ctx context.Context, d EventDispatcher) errorWork {
	return func(err Error) {
		if d == nil {
			return
		}
		d.SendEvent(ctx, err)
	}
}

// ErrorClass represents the class of an error.
type ErrorClass string

const (
	// ClassValidation indicates an error due to invalid input.
	ClassValidation ErrorClass = "validation"
	// ClassNotFound indicates that a resource was not found.
	ClassNotFound ErrorClass = "not_found"
	// ClassAlreadyExists indicates that a resource already exists.
	ClassAlreadyExists ErrorClass = "already_exists"
	// ClassPermissionDenied indicates a permission error.
	ClassPermissionDenied ErrorClass = "permission_denied"
	// ClassUnauthenticated indicates an authentication error.
	ClassUnauthenticated ErrorClass = "unauthenticated"
	// ClassTimeout indicates that an operation timed out.
	ClassTimeout ErrorClass = "timeout"
	// ClassConflict indicates a conflict with the current state of a resource.
	ClassConflict ErrorClass = "conflict"
	// ClassRateLimited indicates that a rate limit has been exceeded.
	ClassRateLimited ErrorClass = "rate_limited"
	// ClassTemporary indicates a temporary error that may be resolved on retry.
	ClassTemporary ErrorClass = "temporary"
	// ClassUnavailable indicates that a service is unavailable.
	ClassUnavailable ErrorClass = "unavailable"
	// ClassInternal indicates a generic internal error.
	ClassInternal ErrorClass = "internal"
	// ClassCancelled indicates that an operation was cancelled.
	ClassCancelled ErrorClass = "cancelled"
	// ClassNotImplemented indicates that a feature is not implemented.
	ClassNotImplemented ErrorClass = "not_implemented"
	// ClassSecurity indicates a security-related error.
	ClassSecurity ErrorClass = "security"
	// ClassCritical indicates a critical, unrecoverable error.
	ClassCritical ErrorClass = "critical"
	// ClassExternal indicates an error from an external service.
	ClassExternal ErrorClass = "external"
	// ClassDataLoss indicates a loss of data.
	ClassDataLoss ErrorClass = "data_loss"
	// ClassResourceExhausted indicates that a resource has been exhausted.
	ClassResourceExhausted ErrorClass = "resource_exhausted"
	// ClassUnknown represents an unknown error class.
	ClassUnknown ErrorClass = ""
)

// ErrorCategory represents the category of an error.
type ErrorCategory string

const (
	// CategoryDatabase indicates a database-related error.
	CategoryDatabase ErrorCategory = "database"
	// CategoryNetwork indicates a network-related error.
	CategoryNetwork ErrorCategory = "network"
	// CategoryOS indicates an operating system-related error.
	CategoryOS ErrorCategory = "os"
	// CategoryAuth indicates an authentication or authorization error.
	CategoryAuth ErrorCategory = "auth"
	// CategorySecurity indicates a security-related error.
	CategorySecurity ErrorCategory = "security"
	// CategoryPayment indicates a payment-related error.
	CategoryPayment ErrorCategory = "payment"
	// CategoryAPI indicates an API-related error.
	CategoryAPI ErrorCategory = "api"
	// CategoryBusinessLogic indicates an error in business logic.
	CategoryBusinessLogic ErrorCategory = "business_logic"
	// CategoryCache indicates a cache-related error.
	CategoryCache ErrorCategory = "cache"
	// CategoryConfig indicates a configuration error.
	CategoryConfig ErrorCategory = "config"
	// CategoryExternal indicates an error from an external service.
	CategoryExternal ErrorCategory = "external"
	// CategoryUserInput indicates an error due to invalid user input.
	CategoryUserInput ErrorCategory = "user_input"
	// CategoryEvents indicates an event-related error.
	CategoryEvents ErrorCategory = "events"
	// CategoryMonitoring indicates a monitoring-related error.
	CategoryMonitoring ErrorCategory = "monitoring"
	// CategoryNotifications indicates a notification-related error.
	CategoryNotifications ErrorCategory = "notifications"
	// CategoryStorage indicates a storage-related error.
	CategoryStorage ErrorCategory = "storage"
	// CategoryProcessing indicates a data processing error.
	CategoryProcessing ErrorCategory = "processing"
	// CategoryAnalytics indicates an analytics-related error.
	CategoryAnalytics ErrorCategory = "analytics"
	// CategoryAI indicates an AI/ML-related error.
	CategoryAI ErrorCategory = "ai"
	// CategoryUnknown represents an unknown error category.
	CategoryUnknown ErrorCategory = ""
)

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity string

// Predefined error severity levels.
const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
	SeverityUnknown  ErrorSeverity = ""
)

// String returns the string representation of ErrorSeverity.
func (s ErrorSeverity) String() string {
	return string(s)
}

// IsValid checks if the severity is one of the predefined values.
func (s ErrorSeverity) IsValid() bool {
	switch s {
	case SeverityUnknown, SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

// Label returns a short, human-readable label for the severity level.
func (s ErrorSeverity) Label() string {
	switch s {
	case SeverityCritical:
		return "[CRIT]"
	case SeverityHigh:
		return "[HIGH]"
	case SeverityMedium:
		return "[MED]"
	case SeverityLow:
		return "[LOW]"
	case SeverityInfo:
		return "[INFO]"
	case SeverityUnknown:
		fallthrough
	default:
		return ""
	}
}

// IsCritical returns true if the severity is Critical.
func (s ErrorSeverity) IsCritical() bool {
	return s == SeverityCritical
}

// IsHigh returns true if the severity is High.
func (s ErrorSeverity) IsHigh() bool {
	return s == SeverityHigh
}

// IsMedium returns true if the severity is Medium.
func (s ErrorSeverity) IsMedium() bool {
	return s == SeverityMedium
}

// IsLow returns true if the severity is Low.
func (s ErrorSeverity) IsLow() bool {
	return s == SeverityLow
}

// IsInfo returns true if the severity is Info.
func (s ErrorSeverity) IsInfo() bool {
	return s == SeverityInfo
}

// IsUnknown returns true if the severity is Unknown.
func (s ErrorSeverity) IsUnknown() bool {
	return s == SeverityUnknown
}

// String returns the string representation of ErrorClass.
func (c ErrorClass) String() string {
	return string(c)
}

// String returns the string representation of ErrorCategory.
func (c ErrorCategory) String() string {
	return string(c)
}
