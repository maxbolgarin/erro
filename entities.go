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
	maxPairsCount  = MaxFieldsCount * 2

	// Wrapping depth limits to prevent stack overflow
	MaxWrapDepth = 50 // Maximum depth of error wrapping

	// Stack trace limits
	MaxStackDepth = 50 // Maximum stack depth

	// Redacted placeholder
	RedactedPlaceholder     = "[REDACTED]"
	MissingFieldPlaceholder = "<missing>"
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
	Class() ErrorClass
	Category() ErrorCategory
	Severity() ErrorSeverity
	IsRetryable() bool
	Message() string
	Fields() []any
	Span() TraceSpan
	Created() time.Time
	LogFields(opts ...LogOptions) []any
	LogFieldsMap(opts ...LogOptions) map[string]any

	// Wrapping
	BaseError() Error
	AllFields() []any

	// Stack trace
	Stack() Stack
}

type ErrorMetrics interface {
	RecordError(err Error)
}

type EventDispatcher interface {
	SendEvent(ctx context.Context, err Error)
}

type TraceSpan interface {
	RecordError(err Error)
	SetAttributes(attributes ...any)
	TraceID() string
	SpanID() string
	ParentSpanID() string
}

type ErrorSchema struct {
	ID           string         `json:"id" bson:"_id" db:"id"`
	Class        ErrorClass     `json:"class,omitempty" bson:"class,omitempty" db:"class,omitempty"`
	Category     ErrorCategory  `json:"category,omitempty" bson:"category,omitempty" db:"category,omitempty"`
	Severity     ErrorSeverity  `json:"severity,omitempty" bson:"severity,omitempty" db:"severity,omitempty"`
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
