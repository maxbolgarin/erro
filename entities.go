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
	Class(class Class) Error
	Category(category Category) Error
	Severity(severity Severity) Error
	Tags(tags ...Tag) Error
	Retryable(retryable bool) Error
	Fields(fields ...any) Error
	Context(ctx context.Context) Error
	Span(span Span) Error

	// Extraction methods
	GetBase() Error
	GetCreated() time.Time
	GetCode() string
	GetClass() Class
	GetCategory() Category
	GetTags() []Tag
	HasTag(tag Tag) bool
	IsRetryable() bool
	GetFields() []any
	GetContext() context.Context
	GetSpan() Span

	// Severity checking methods
	GetSeverity() Severity
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
	ClassDependencyFailed  Class = "dependency_failed"
	ClassNotImplemented    Class = "not_implemented"
	ClassSecurity          Class = "security"
	ClassCritical          Class = "critical"
	ClassExternal          Class = "external"
	ClassDataLoss          Class = "data_loss"
	ClassResourceExhausted Class = "resource_exhausted"
	ClassExpected          Class = "expected"
	ClassUnknown           Class = ""
)

type Category string

const (
	CategoryDatabase      Category = "database"
	CategoryNetwork       Category = "network"
	CategoryOS            Category = "os"
	CategoryAuth          Category = "auth"
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

type Tag string

const (
	TagUnknown       Tag = "unknown"
	TagTemporary     Tag = "temporary"
	TagRetryable     Tag = "retryable"
	TagCritical      Tag = "critical"
	TagSecurity      Tag = "security"
	TagUserInput     Tag = "user_input"
	TagExternal      Tag = "external"
	TagExpected      Tag = "expected"
	TagDeprecated    Tag = "deprecated"
	TagAudit         Tag = "audit"
	TagFeatureFlag   Tag = "feature_flag"
	TagExperimental  Tag = "experimental"
	TagMigration     Tag = "migration"
	TagThrottled     Tag = "throttled"
	TagAlert         Tag = "alert"
	TagMonitoring    Tag = "monitoring"
	TagCompliance    Tag = "compliance"
	TagPerformance   Tag = "performance"
	TagResource      Tag = "resource"
	TagDataLoss      Tag = "data_loss"
	TagTimeout       Tag = "timeout"
	TagCancelled     Tag = "cancelled"
	TagConflict      Tag = "conflict"
	TagDuplicate     Tag = "duplicate"
	TagNotFound      Tag = "not_found"
	TagAlreadyExists Tag = "already_exists"
	TagHigh          Tag = "high"
	TagMedium        Tag = "medium"
	TagLow           Tag = "low"
	TagInfo          Tag = "info"
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

func (c Class) String() string {
	return string(c)
}

func (c Category) String() string {
	return string(c)
}

func (t Tag) String() string {
	return string(t)
}
