package erro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var (
	ErrMaxWrapDepthExceeded = New("maximum wrap depth exceeded")
)

// Security configuration constants
const (
	// Maximum string lengths to prevent memory exhaustion
	MaxMessageLength = 1000 // Maximum length for error messages
	MaxKeyLength     = 128  // Maximum length for field keys
	MaxValueLength   = 1024 // Maximum length for field values (when converted to string)

	// Maximum array/slice lengths to prevent array bombing
	MaxFieldsCount = 100 // Maximum number of fields (key-value pairs)

	// Wrapping depth limits to prevent stack overflow
	MaxWrapDepth = 50 // Maximum depth of error wrapping

	// Stack trace limits
	MaxStackDepth = 50 // Maximum stack depth

	// Redacted placeholder
	RedactedPlaceholder = "[REDACTED]"
)

type (
	FormatErrorFunc func(err Error) string
	KeyGetterFunc   func(err error) string
)

// Error represents the common interface for all erro errors
type Error interface {
	// Error interface
	error
	fmt.Formatter
	json.Marshaler
	json.Unmarshaler
	Is(target error) bool
	As(target any) bool
	Unwrap() error

	// Metadata
	ID() string
	Class() Class
	Category() Category
	Severity() Severity
	IsRetryable() bool
	Message() string
	Fields() []any
	Span() Span
	Created() time.Time

	// Wrapping
	BaseError() Error
	AllFields() []any

	// Stack trace
	Stack() Stack
}

type Metrics interface {
	RecordError(err Error)
}

type Dispatcher interface {
	SendEvent(ctx context.Context, err Error)
}

type Span interface {
	RecordError(err Error)
	SetAttributes(attributes ...any)
	TraceID() string
	SpanID() string
	ParentSpanID() string
}

type ErrorSchema struct {
	ID           string         `json:"id" bson:"_id" db:"id"`
	Class        Class          `json:"class,omitempty" bson:"class,omitempty" db:"class,omitempty"`
	Category     Category       `json:"category,omitempty" bson:"category,omitempty" db:"category,omitempty"`
	Severity     Severity       `json:"severity,omitempty" bson:"severity,omitempty" db:"severity,omitempty"`
	Created      time.Time      `json:"created,omitempty" bson:"created,omitempty" db:"created,omitempty"`
	Message      string         `json:"message,omitempty" bson:"message,omitempty" db:"message,omitempty"`
	Fields       []any          `json:"fields,omitempty" bson:"fields,omitempty" db:"fields,omitempty"`
	Retryable    bool           `json:"retryable,omitempty" bson:"retryable,omitempty" db:"retryable,omitempty"`
	StackTrace   []StackContext `json:"stack_trace,omitempty" bson:"stack_trace,omitempty" db:"stack_trace,omitempty"`
	TraceID      string         `json:"trace_id,omitempty" bson:"trace_id,omitempty" db:"trace_id,omitempty"`
	SpanID       string         `json:"span_id,omitempty" bson:"span_id,omitempty" db:"span_id,omitempty"`
	ParentSpanID string         `json:"parent_span_id,omitempty" bson:"parent_span_id,omitempty" db:"parent_span_id,omitempty"`
}

// RedactedValue is a wrapper for a value that should be redacted in logs.
type RedactedValue struct {
	Value any
}

// Redact wraps a value to mark it as sensitive. Its content will be replaced
// with RedactedPlaceholder when the error is formatted as a string or JSON.
func Redact(value any) RedactedValue {
	return RedactedValue{Value: value}
}

// Key getter functions for deduplication
var (
	// MessageKeyGetter generates a key based on the error's message.
	MessageKeyGetter KeyGetterFunc = func(err error) string {
		if e, ok := err.(Error); ok {
			return e.Message()
		}
		return err.Error()
	}
	// IDKeyGetter generates a key based on the error's ID.
	IDKeyGetter KeyGetterFunc = func(err error) string {
		if e, ok := err.(Error); ok {
			return e.ID()
		}
		return err.Error()
	}
	// ErrorKeyGetter generates a key based on the error's class.
	ErrorKeyGetter KeyGetterFunc = func(err error) string {
		return err.Error()
	}
)

type Class string

const (
	ClassValidation        Class = "validation"
	ClassNotFound          Class = "not_found"
	ClassAlreadyExists     Class = "already_exists"
	ClassPermissionDenied  Class = "permission_denied"
	ClassUnauthenticated   Class = "unauthenticated"
	ClassTimeout           Class = "timeout"
	ClassConflict          Class = "conflict"
	ClassRateLimited       Class = "rate_limited"
	ClassTemporary         Class = "temporary"
	ClassUnavailable       Class = "unavailable"
	ClassInternal          Class = "internal"
	ClassCancelled         Class = "cancelled"
	ClassNotImplemented    Class = "not_implemented"
	ClassSecurity          Class = "security"
	ClassCritical          Class = "critical"
	ClassExternal          Class = "external"
	ClassDataLoss          Class = "data_loss"
	ClassResourceExhausted Class = "resource_exhausted"
	ClassUnknown           Class = ""
)

type Category string

const (
	CategoryDatabase      Category = "database"
	CategoryNetwork       Category = "network"
	CategoryOS            Category = "os"
	CategoryAuth          Category = "auth"
	CategorySecurity      Category = "security"
	CategoryPayment       Category = "payment"
	CategoryAPI           Category = "api"
	CategoryBusinessLogic Category = "business_logic"
	CategoryCache         Category = "cache"
	CategoryConfig        Category = "config"
	CategoryExternal      Category = "external"
	CategoryUserInput     Category = "user_input"
	CategoryEvents        Category = "events"
	CategoryMonitoring    Category = "monitoring"
	CategoryNotifications Category = "notifications"
	CategoryStorage       Category = "storage"
	CategoryProcessing    Category = "processing"
	CategoryAnalytics     Category = "analytics"
	CategoryAI            Category = "ai"
	CategoryUnknown       Category = ""
)

// Severity represents the severity level of an error
type Severity string

// Predefined error severity levels
const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
	SeverityUnknown  Severity = ""
)

// String returns the string representation of ErrorSeverity
func (s Severity) String() string {
	return string(s)
}

// IsValid checks if the severity is one of the predefined values
func (s Severity) IsValid() bool {
	switch s {
	case SeverityUnknown, SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

func (s Severity) Label() string {
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

func (s Severity) IsCritical() bool {
	return s == SeverityCritical
}

func (s Severity) IsHigh() bool {
	return s == SeverityHigh
}

func (s Severity) IsMedium() bool {
	return s == SeverityMedium
}

func (s Severity) IsLow() bool {
	return s == SeverityLow
}

func (s Severity) IsInfo() bool {
	return s == SeverityInfo
}

func (s Severity) IsUnknown() bool {
	return s == SeverityUnknown
}

func (c Class) String() string {
	return string(c)
}

func (c Category) String() string {
	return string(c)
}
