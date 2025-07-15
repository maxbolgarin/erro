package erro

import "context"

type (
	errorOpt    func(err *baseError)
	errorWork   func(err Error)
	errorFields func() []any
)

func ID(id string) errorOpt {
	return func(err *baseError) {
		err.id = id
	}
}

func Retryable() errorOpt {
	return func(err *baseError) {
		err.retryable = true
	}
}

func Fields(fields ...any) errorFields {
	return func() []any {
		return fields
	}
}

func Formatter(f FormatErrorFunc) errorOpt {
	return func(err *baseError) {
		if f == nil {
			return
		}
		err.formatter = f
	}
}

func StackTrace(c ...*StackTraceConfig) errorOpt {
	return func(err *baseError) {
		err.stack = captureStack(3)
		if len(c) > 0 {
			err.stackTraceConfig = c[0]
		} else {
			err.stackTraceConfig = nil
		}
	}
}

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

func RecordMetrics(m ErrorMetrics) errorWork {
	return func(err Error) {
		if m == nil {
			return
		}
		m.RecordError(err)
	}
}

func SendEvent(ctx context.Context, d EventDispatcher) errorWork {
	return func(err Error) {
		if d == nil {
			return
		}
		d.SendEvent(ctx, err)
	}
}

type ErrorClass string

const (
	ClassValidation        ErrorClass = "validation"
	ClassNotFound          ErrorClass = "not_found"
	ClassAlreadyExists     ErrorClass = "already_exists"
	ClassPermissionDenied  ErrorClass = "permission_denied"
	ClassUnauthenticated   ErrorClass = "unauthenticated"
	ClassTimeout           ErrorClass = "timeout"
	ClassConflict          ErrorClass = "conflict"
	ClassRateLimited       ErrorClass = "rate_limited"
	ClassTemporary         ErrorClass = "temporary"
	ClassUnavailable       ErrorClass = "unavailable"
	ClassInternal          ErrorClass = "internal"
	ClassCancelled         ErrorClass = "cancelled"
	ClassNotImplemented    ErrorClass = "not_implemented"
	ClassSecurity          ErrorClass = "security"
	ClassCritical          ErrorClass = "critical"
	ClassExternal          ErrorClass = "external"
	ClassDataLoss          ErrorClass = "data_loss"
	ClassResourceExhausted ErrorClass = "resource_exhausted"
	ClassUnknown           ErrorClass = ""
)

type ErrorCategory string

const (
	CategoryDatabase      ErrorCategory = "database"
	CategoryNetwork       ErrorCategory = "network"
	CategoryOS            ErrorCategory = "os"
	CategoryAuth          ErrorCategory = "auth"
	CategorySecurity      ErrorCategory = "security"
	CategoryPayment       ErrorCategory = "payment"
	CategoryAPI           ErrorCategory = "api"
	CategoryBusinessLogic ErrorCategory = "business_logic"
	CategoryCache         ErrorCategory = "cache"
	CategoryConfig        ErrorCategory = "config"
	CategoryExternal      ErrorCategory = "external"
	CategoryUserInput     ErrorCategory = "user_input"
	CategoryEvents        ErrorCategory = "events"
	CategoryMonitoring    ErrorCategory = "monitoring"
	CategoryNotifications ErrorCategory = "notifications"
	CategoryStorage       ErrorCategory = "storage"
	CategoryProcessing    ErrorCategory = "processing"
	CategoryAnalytics     ErrorCategory = "analytics"
	CategoryAI            ErrorCategory = "ai"
	CategoryUnknown       ErrorCategory = ""
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

// Predefined error severity levels
const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
	SeverityUnknown  ErrorSeverity = ""
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	return string(s)
}

// IsValid checks if the severity is one of the predefined values
func (s ErrorSeverity) IsValid() bool {
	switch s {
	case SeverityUnknown, SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

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

func (s ErrorSeverity) IsCritical() bool {
	return s == SeverityCritical
}

func (s ErrorSeverity) IsHigh() bool {
	return s == SeverityHigh
}

func (s ErrorSeverity) IsMedium() bool {
	return s == SeverityMedium
}

func (s ErrorSeverity) IsLow() bool {
	return s == SeverityLow
}

func (s ErrorSeverity) IsInfo() bool {
	return s == SeverityInfo
}

func (s ErrorSeverity) IsUnknown() bool {
	return s == SeverityUnknown
}

func (c ErrorClass) String() string {
	return string(c)
}

func (c ErrorCategory) String() string {
	return string(c)
}
