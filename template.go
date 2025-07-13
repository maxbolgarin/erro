package erro

import (
	"fmt"
	"strings"
)

// ErrorTemplate represents a template for creating errors with predefined metadata
type ErrorTemplate struct {
	class     Class
	category  Category
	severity  Severity
	fields    []any
	retryable bool

	messageTemplate string
}

// NewTemplate creates a new error template
func NewTemplate(fields ...any) *ErrorTemplate {
	return &ErrorTemplate{
		fields: fields,
	}
}

// Class sets the error class for the template
func (t *ErrorTemplate) Class(class Class) *ErrorTemplate {
	t.class = class
	return t
}

// Category sets the error category for the template
func (t *ErrorTemplate) Category(category Category) *ErrorTemplate {
	t.category = category
	return t
}

// Severity sets the error severity for the template
func (t *ErrorTemplate) Severity(severity Severity) *ErrorTemplate {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	t.severity = severity
	return t
}

// Retryable sets the retryable flag for the template
func (t *ErrorTemplate) Retryable(retryable bool) *ErrorTemplate {
	t.retryable = retryable
	return t
}

// Fields adds fields to the template
func (t *ErrorTemplate) Fields(fields ...any) *ErrorTemplate {
	t.fields = safeAppendFields(t.fields, prepareFields(fields))
	return t
}

// Message sets a message template with placeholders
func (t *ErrorTemplate) Message(template string) *ErrorTemplate {
	t.messageTemplate = template
	return t
}

// Create creates an error using the template
func (t *ErrorTemplate) New(fields ...any) Error {
	var message string

	// Use message template if provided
	if t.messageTemplate != "" {
		if len(fields) > 0 {
			message = t.formatTemplate(t.messageTemplate, fields...)
		} else {
			message = strings.TrimSuffix(t.messageTemplate, ": %s")
		}
		return t.Errorf(message)
	}

	// Fallback to first field as message
	if len(fields) > 0 {
		message = valueToString(fields[0])
		if len(fields) > 1 {
			fields = fields[1:]
		} else {
			fields = nil
		}
	}

	return t.Errorf(message, fields...)
}

// CreateWithMessage creates an error with a custom message, overriding the template
func (t *ErrorTemplate) Errorf(message string, fields ...any) Error {
	if message == "" && t.messageTemplate != "" {
		if len(fields) > 0 {
			message = t.formatTemplate(t.messageTemplate, fields...)
		} else {
			message = strings.TrimSuffix(t.messageTemplate, ": %s")
		}
	}

	// It behaves like New if there is no format verbs in the message
	err := Errorf(message, fields...)

	t.setErrorMetadata(err)

	return err
}

func (t *ErrorTemplate) Wrap(originalErr error, fields ...any) Error {
	var message string

	// Use message template if provided
	if t.messageTemplate != "" {
		if len(fields) > 0 {
			message = t.formatTemplate(t.messageTemplate, fields...)
		} else {
			message = strings.TrimSuffix(t.messageTemplate, ": %s")
		}
		return t.Wrapf(originalErr, message)
	}

	// Fallback to first field as message
	if len(fields) > 0 {
		message = valueToString(fields[0])
		if len(fields) > 1 {
			fields = fields[1:]
		} else {
			fields = nil
		}
	}

	return t.Wrapf(originalErr, message, fields...)
}

// CreateWithMessage creates an error with a custom message, overriding the template
func (t *ErrorTemplate) Wrapf(originalErr error, message string, fields ...any) Error {
	if message == "" && t.messageTemplate != "" {
		if len(fields) > 0 {
			message = t.formatTemplate(t.messageTemplate, fields...)
		} else {
			message = strings.TrimSuffix(t.messageTemplate, ": %s")
		}
	}

	// It behaves like Wrap if there is no format verbs in the message
	err := Wrapf(originalErr, message, fields...)

	t.setErrorMetadata(err)

	return err
}

func (t *ErrorTemplate) setErrorMetadata(err Error) {
	err.Class(t.class)
	err.Category(t.category)
	err.Severity(t.severity)
	err.Retryable(t.retryable)
	err.Fields(t.fields...)
}

// formatTemplate formats a template string using fields
func (t *ErrorTemplate) formatTemplate(template string, fields ...any) string {
	if len(fields) == 0 {
		return template
	}
	return fmt.Sprintf(template, fields...)
}

// Predefined templates with message templates
var (
	// ValidationError creates validation errors
	ValidationError = NewTemplate().
			Class(ClassValidation).
			Category(CategoryUserInput).
			Severity(SeverityLow).
			Message("failed validation: %s")

	// NotFoundError creates not found errors
	NotFoundError = NewTemplate().
			Class(ClassNotFound).
			Severity(SeverityMedium).
			Message("%s not found")

	// DatabaseError creates database errors
	DatabaseError = NewTemplate().
			Category(CategoryDatabase).
			Severity(SeverityHigh).
			Message("database error: %s")

	// NetworkError creates network errors
	NetworkError = NewTemplate().
			Category(CategoryNetwork).
			Severity(SeverityMedium).
			Retryable(true).
			Message("network error: %s")

	// AuthenticationError creates authentication errors
	AuthenticationError = NewTemplate().
				Category(CategoryAuth).
				Class(ClassUnauthenticated).
				Severity(SeverityMedium).
				Message("authentication failed: %s")

	// AuthorizationError creates authorization errors
	AuthorizationError = NewTemplate().
				Category(CategoryAuth).
				Class(ClassPermissionDenied).
				Severity(SeverityHigh).
				Message("permission denied: %s")

	// TimeoutError creates timeout errors
	TimeoutError = NewTemplate().
			Class(ClassTimeout).
			Severity(SeverityLow).
			Retryable(true).
			Message("operation timeout: %s")

	// ConflictError creates conflict errors
	ConflictError = NewTemplate().
			Class(ClassConflict).
			Severity(SeverityMedium).
			Message("conflict: %s")

	// RateLimitError creates rate limit errors
	RateLimitError = NewTemplate().
			Class(ClassRateLimited).
			Severity(SeverityLow).
			Retryable(true).
			Message("rate limit exceeded: %s")

	// InternalError creates internal errors
	InternalError = NewTemplate().
			Class(ClassInternal).
			Severity(SeverityHigh).
			Message("internal error: %s")

	// SecurityError creates security errors
	SecurityError = NewTemplate().
			Class(ClassSecurity).
			Category(CategorySecurity).
			Severity(SeverityCritical).
			Message("security violation: %s")

	// ExternalError creates external service errors
	ExternalError = NewTemplate().
			Class(ClassExternal).
			Category(CategoryExternal).
			Severity(SeverityMedium).
			Retryable(true).
			Message("external error: %s")

	// PaymentError creates payment errors
	PaymentError = NewTemplate().
			Category(CategoryPayment).
			Severity(SeverityCritical).
			Retryable(true).
			Message("payment error: %s")

	// CacheError creates cache errors
	CacheError = NewTemplate().
			Category(CategoryCache).
			Class(ClassTemporary).
			Severity(SeverityMedium).
			Retryable(true).
			Message("cache error: %s")

	// ConfigError creates configuration errors
	ConfigError = NewTemplate().
			Category(CategoryConfig).
			Class(ClassInternal).
			Severity(SeverityCritical).
			Message("configuration error: %s")

	// APIError creates API errors
	APIError = NewTemplate().
			Category(CategoryAPI).
			Severity(SeverityHigh).
			Message("API error: %s")

	// BusinessLogicError creates business logic errors
	BusinessLogicError = NewTemplate().
				Category(CategoryBusinessLogic).
				Severity(SeverityHigh).
				Message("business logic error: %s")

	// StorageError creates storage errors
	StorageError = NewTemplate().
			Category(CategoryStorage).
			Severity(SeverityHigh).
			Message("storage error: %s")

	// ProcessingError creates processing errors
	ProcessingError = NewTemplate().
			Category(CategoryProcessing).
			Class(ClassInternal).
			Severity(SeverityHigh).
			Message("processing error: %s")

	// MonitoringError creates monitoring errors
	MonitoringError = NewTemplate().
			Category(CategoryMonitoring).
			Severity(SeverityMedium).
			Message("monitoring error: %s")

	// NotificationError creates notification errors
	NotificationError = NewTemplate().
				Category(CategoryNotifications).
				Class(ClassTemporary).
				Severity(SeverityLow).
				Message("notification error: %s")

	// AIError creates AI/ML errors
	AIError = NewTemplate().
		Category(CategoryAI).
		Class(ClassInternal).
		Severity(SeverityHigh).
		Message("AI error: %s")

		// AnalyticsError creates analytics errors
	AnalyticsError = NewTemplate().
			Category(CategoryAnalytics).
			Severity(SeverityLow).
			Message("analytics error: %s")

	// EventsTemplate creates events errors
	EventsTemplate = NewTemplate().
			Category(CategoryEvents).
			Severity(SeverityMedium).
			Message("events error: %s")

	// CriticalError creates critical errors
	CriticalError = NewTemplate().
			Class(ClassCritical).
			Severity(SeverityCritical).
			Message("critical error: %s")

	// TemporaryError creates temporary errors
	TemporaryError = NewTemplate().
			Class(ClassTemporary).
			Severity(SeverityMedium).
			Message("temporary error: %s")

	// DataLossError creates data loss errors
	DataLossError = NewTemplate().
			Class(ClassDataLoss).
			Severity(SeverityCritical).
			Message("data loss: %s")

	// ResourceExhaustedError creates resource exhausted errors
	ResourceExhaustedError = NewTemplate().
				Class(ClassResourceExhausted).
				Severity(SeverityHigh).
				Message("resource exhausted: %s")

	// UnavailableError creates unavailable errors
	UnavailableError = NewTemplate().
				Class(ClassUnavailable).
				Severity(SeverityHigh).
				Message("service unavailable: %s")

	// CancelledError creates cancelled errors
	CancelledError = NewTemplate().
			Class(ClassCancelled).
			Severity(SeverityLow).
			Message("operation cancelled: %s")

	// NotImplementedError creates not implemented errors
	NotImplementedError = NewTemplate().
				Class(ClassNotImplemented).
				Severity(SeverityMedium).
				Message("not implemented: %s")

	// AlreadyExistsError creates already exists errors
	AlreadyExistsError = NewTemplate().
				Class(ClassAlreadyExists).
				Severity(SeverityMedium).
				Message("already exists: %s")
)
