package erro

import (
	"context"
	"time"
)

// Error represents the common interface for all erro errors
type Error interface {
	error

	// Chaining methods for building errors
	Code(code string) Error
	Category(category string) Error
	Severity(severity ErrorSeverity) Error
	Fields(fields ...any) Error
	Context(ctx context.Context) Error
	Tags(tags ...string) Error
	Retryable(retryable bool) Error
	Span(span Span) Error

	// Extraction methods
	GetBase() Error
	GetContext() context.Context
	GetCode() string
	GetCategory() string
	GetTags() []string
	IsRetryable() bool
	GetSpan() Span
	GetFields() []any
	GetCreated() time.Time

	// Severity checking methods
	GetSeverity() ErrorSeverity
	IsCritical() bool
	IsHigh() bool
	IsMedium() bool
	IsLow() bool
	IsInfo() bool
	IsUnknown() bool

	// Stack trace access
	Stack() Stack
	StackFormat() string
	StackWithError() string

	// Error comparison
	Is(target error) bool
}

type Span interface {
	RecordError(err error)
	SetAttributes(attributes ...any)
	TraceID() string
	SpanID() string
	ParentSpanID() string
}

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

// Predefined error severity levels
const (
	Critical ErrorSeverity = "critical"
	High     ErrorSeverity = "high"
	Medium   ErrorSeverity = "medium"
	Low      ErrorSeverity = "low"
	Info     ErrorSeverity = "info"
	Unknown  ErrorSeverity = ""
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	return string(s)
}

// IsValid checks if the severity is one of the predefined values
func (s ErrorSeverity) IsValid() bool {
	switch s {
	case Unknown, Critical, High, Medium, Low, Info:
		return true
	default:
		return false
	}
}

func (s ErrorSeverity) Label() string {
	switch s {
	case Critical:
		return "[CRIT]"
	case High:
		return "[HIGH]"
	case Medium:
		return "[MED]"
	case Low:
		return "[LOW]"
	case Info:
		return "[INFO]"
	case Unknown:
		fallthrough
	default:
		return ""
	}
}
