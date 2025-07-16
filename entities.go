package erro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var (
	// ErrMaxWrapDepthExceeded is returned when the maximum error wrapping depth is exceeded.
	ErrMaxWrapDepthExceeded = New("maximum wrap depth exceeded")
)

// Security configuration constants
const (
	// MaxMessageLength is the maximum length for error messages.
	MaxMessageLength = 1000
	// MaxKeyLength is the maximum length for field keys.
	MaxKeyLength = 128
	// MaxValueLength is the maximum length for field values when converted to a string.
	MaxValueLength = 1024

	// MaxFieldsCount is the maximum number of fields (key-value pairs).
	MaxFieldsCount = 100
	maxPairsCount  = MaxFieldsCount * 2

	// MaxWrapDepth is the maximum depth of error wrapping.
	MaxWrapDepth = 50

	// MaxStackDepth is the maximum stack depth.
	MaxStackDepth = 50

	// RedactedPlaceholder is the placeholder for redacted values.
	RedactedPlaceholder = "[REDACTED]"
	// MissingFieldPlaceholder is the placeholder for missing field values.
	MissingFieldPlaceholder = "<missing>"
)

type (
	// FormatErrorFunc is a function that formats an error into a string.
	FormatErrorFunc func(err Error) string
	// KeyGetterFunc is a function that generates a key for an error, used for deduplication.
	KeyGetterFunc func(err error) string
)

// Error represents the common interface for all erro errors.
type Error interface {
	// error interface
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

// ErrorMetrics is an interface for recording error metrics.
type ErrorMetrics interface {
	RecordError(err Error)
}

// EventDispatcher is an interface for sending error events.
type EventDispatcher interface {
	SendEvent(ctx context.Context, err Error)
}

// TraceSpan is an interface for recording errors in a trace span.
type TraceSpan interface {
	RecordError(err Error)
	SetAttributes(attributes ...any)
	TraceID() string
	SpanID() string
	ParentSpanID() string
}

// ErrorSchema is a serializable representation of an error.
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
	// MessageKeyGetter generates a key based on the error's message without fields.
	MessageKeyGetter KeyGetterFunc = func(err error) string {
		if erroErr, ok := err.(Error); ok {
			return erroErr.Message()
		}
		var erroErr Error
		if As(err, &erroErr) {
			return erroErr.Message()
		}
		return err.Error()
	}
	// IDKeyGetter generates a key based on the error's ID.
	IDKeyGetter KeyGetterFunc = func(err error) string {
		if erroErr, ok := err.(Error); ok {
			return erroErr.ID()
		}
		var erroErr Error
		if As(err, &erroErr) {
			return erroErr.ID()
		}
		return err.Error()
	}
	// ErrorKeyGetter generates a key based on the error's string representation (err.Error()).
	ErrorKeyGetter KeyGetterFunc = func(err error) string {
		return err.Error()
	}
)
