package erro

import (
	"context"
	"fmt"
	"strings"
)

// ErrorTemplate represents a template for creating errors with predefined metadata
type ErrorTemplate struct {
	id        string
	class     Class
	category  Category
	severity  Severity
	fields    []any
	retryable bool
	ctx       context.Context
	span      Span

	metrics    Metrics
	dispatcher Dispatcher

	messageTemplate string
}

// NewTemplate creates a new error template
func NewTemplate(fields ...any) *ErrorTemplate {
	return &ErrorTemplate{
		fields: fields,
	}
}

func (t *ErrorTemplate) WithID(id string) *ErrorTemplate {
	t.id = id
	return t
}

// WithClass sets the error class for the template
func (t *ErrorTemplate) WithClass(class Class) *ErrorTemplate {
	t.class = class
	return t
}

// WithCategory sets the error category for the template
func (t *ErrorTemplate) WithCategory(category Category) *ErrorTemplate {
	t.category = category
	return t
}

// WithSeverity sets the error severity for the template
func (t *ErrorTemplate) WithSeverity(severity Severity) *ErrorTemplate {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	t.severity = severity
	return t
}

// WithRetryable sets the retryable flag for the template
func (t *ErrorTemplate) WithRetryable(retryable bool) *ErrorTemplate {
	t.retryable = retryable
	return t
}

// WithFields adds fields to the template
func (t *ErrorTemplate) WithFields(fields ...any) *ErrorTemplate {
	t.fields = safeAppendFields(t.fields, prepareFields(fields))
	return t
}

// WithMessageTemplate sets a message template with placeholders
func (t *ErrorTemplate) WithMessageTemplate(template string) *ErrorTemplate {
	t.messageTemplate = template
	return t
}

// WithMetrics sets the metrics for the template
func (t *ErrorTemplate) WithMetrics(metrics Metrics) *ErrorTemplate {
	t.metrics = metrics
	return t
}

// WithDispatcher sets the dispatcher for the template
func (t *ErrorTemplate) WithDispatcher(dispatcher Dispatcher) *ErrorTemplate {
	t.dispatcher = dispatcher
	return t
}

// WithGoContext sets the context for the template
func (t *ErrorTemplate) WithGoContext(ctx context.Context) *ErrorTemplate {
	t.ctx = ctx
	return t
}

// WithSpan sets the span for the template
func (t *ErrorTemplate) WithSpan(span Span) *ErrorTemplate {
	t.span = span
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
		return t.Newf(message)
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

	return t.Newf(message, fields...)
}

// Newf creates an error with a custom message, overriding the template
func (t *ErrorTemplate) Newf(message string, fields ...any) Error {
	if message == "" && t.messageTemplate != "" {
		if len(fields) > 0 {
			message = t.formatTemplate(t.messageTemplate, fields...)
		} else {
			message = strings.TrimSuffix(t.messageTemplate, ": %s")
		}
	}

	// It behaves like New if there is no format verbs in the message
	err := Newf(message, fields...)
	err = t.applyMetadata(err)

	if t.metrics != nil {
		t.metrics.RecordError(err.Context())
	}
	if t.dispatcher != nil {
		t.dispatcher.SendEvent(t.ctx, err.Context())
	}

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

// Wrapf creates an error with a custom message, overriding the template
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
	err = t.applyMetadata(err)
	if t.metrics != nil {
		t.metrics.RecordError(err.Context())
	}
	if t.dispatcher != nil {
		t.dispatcher.SendEvent(t.ctx, err.Context())
	}

	return err
}

func (t *ErrorTemplate) applyMetadata(err Error) Error {
	err.WithClass(t.class)
	err.WithCategory(t.category)
	err.WithSeverity(t.severity)
	err.WithFields(t.fields...)
	err.WithRetryable(t.retryable)
	err.WithSpan(t.span)
	err.WithID(t.id)

	return err
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
			WithClass(ClassValidation).
			WithCategory(CategoryUserInput).
			WithSeverity(SeverityLow).
			WithMessageTemplate("failed validation: %s")

	// NotFoundError creates not found errors
	NotFoundError = NewTemplate().
			WithClass(ClassNotFound).
			WithSeverity(SeverityMedium).
			WithMessageTemplate("%s not found")

	// DatabaseError creates database errors
	DatabaseError = NewTemplate().
			WithCategory(CategoryDatabase).
			WithSeverity(SeverityHigh).
			WithMessageTemplate("database error: %s")

	// NetworkError creates network errors
	NetworkError = NewTemplate().
			WithCategory(CategoryNetwork).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithMessageTemplate("network error: %s")

	// AuthenticationError creates authentication errors
	AuthenticationError = NewTemplate().
				WithCategory(CategoryAuth).
				WithClass(ClassUnauthenticated).
				WithSeverity(SeverityMedium).
				WithMessageTemplate("authentication failed: %s")

	// AuthorizationError creates authorization errors
	AuthorizationError = NewTemplate().
				WithCategory(CategoryAuth).
				WithClass(ClassPermissionDenied).
				WithSeverity(SeverityHigh).
				WithMessageTemplate("permission denied: %s")

	// TimeoutError creates timeout errors
	TimeoutError = NewTemplate().
			WithClass(ClassTimeout).
			WithSeverity(SeverityLow).
			WithRetryable(true).
			WithMessageTemplate("operation timeout: %s")

	// ConflictError creates conflict errors
	ConflictError = NewTemplate().
			WithClass(ClassConflict).
			WithSeverity(SeverityMedium).
			WithMessageTemplate("conflict: %s")

	// RateLimitError creates rate limit errors
	RateLimitError = NewTemplate().
			WithClass(ClassRateLimited).
			WithSeverity(SeverityLow).
			WithRetryable(true).
			WithMessageTemplate("rate limit exceeded: %s")

	// InternalError creates internal errors
	InternalError = NewTemplate().
			WithClass(ClassInternal).
			WithSeverity(SeverityHigh).
			WithMessageTemplate("internal error: %s")

	// SecurityError creates security errors
	SecurityError = NewTemplate().
			WithClass(ClassSecurity).
			WithCategory(CategorySecurity).
			WithSeverity(SeverityCritical).
			WithMessageTemplate("security violation: %s")

	// ExternalError creates external service errors
	ExternalError = NewTemplate().
			WithClass(ClassExternal).
			WithCategory(CategoryExternal).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithMessageTemplate("external error: %s")

	// PaymentError creates payment errors
	PaymentError = NewTemplate().
			WithCategory(CategoryPayment).
			WithSeverity(SeverityCritical).
			WithRetryable(true).
			WithMessageTemplate("payment error: %s")

	// CacheError creates cache errors
	CacheError = NewTemplate().
			WithCategory(CategoryCache).
			WithClass(ClassTemporary).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithMessageTemplate("cache error: %s")

	// ConfigError creates configuration errors
	ConfigError = NewTemplate().
			WithCategory(CategoryConfig).
			WithClass(ClassInternal).
			WithSeverity(SeverityCritical).
			WithMessageTemplate("configuration error: %s")

	// APIError creates API errors
	APIError = NewTemplate().
			WithCategory(CategoryAPI).
			WithSeverity(SeverityHigh).
			WithMessageTemplate("API error: %s")

	// BusinessLogicError creates business logic errors
	BusinessLogicError = NewTemplate().
				WithCategory(CategoryBusinessLogic).
				WithSeverity(SeverityHigh).
				WithMessageTemplate("business logic error: %s")

	// StorageError creates storage errors
	StorageError = NewTemplate().
			WithCategory(CategoryStorage).
			WithSeverity(SeverityHigh).
			WithMessageTemplate("storage error: %s")

	// ProcessingError creates processing errors
	ProcessingError = NewTemplate().
			WithCategory(CategoryProcessing).
			WithClass(ClassInternal).
			WithSeverity(SeverityHigh).
			WithMessageTemplate("processing error: %s")

	// MonitoringError creates monitoring errors
	MonitoringError = NewTemplate().
			WithCategory(CategoryMonitoring).
			WithSeverity(SeverityMedium).
			WithMessageTemplate("monitoring error: %s")

	// NotificationError creates notification errors
	NotificationError = NewTemplate().
				WithCategory(CategoryNotifications).
				WithClass(ClassTemporary).
				WithSeverity(SeverityLow).
				WithMessageTemplate("notification error: %s")

	// AIError creates AI/ML errors
	AIError = NewTemplate().
		WithCategory(CategoryAI).
		WithClass(ClassInternal).
		WithSeverity(SeverityHigh).
		WithMessageTemplate("AI error: %s")

		// AnalyticsError creates analytics errors
	AnalyticsError = NewTemplate().
			WithCategory(CategoryAnalytics).
			WithSeverity(SeverityLow).
			WithMessageTemplate("analytics error: %s")

	// EventsTemplate creates events errors
	EventsTemplate = NewTemplate().
			WithCategory(CategoryEvents).
			WithSeverity(SeverityMedium).
			WithMessageTemplate("events error: %s")

	// CriticalError creates critical errors
	CriticalError = NewTemplate().
			WithClass(ClassCritical).
			WithSeverity(SeverityCritical).
			WithMessageTemplate("critical error: %s")

	// TemporaryError creates temporary errors
	TemporaryError = NewTemplate().
			WithClass(ClassTemporary).
			WithSeverity(SeverityMedium).
			WithMessageTemplate("temporary error: %s")

	// DataLossError creates data loss errors
	DataLossError = NewTemplate().
			WithClass(ClassDataLoss).
			WithSeverity(SeverityCritical).
			WithMessageTemplate("data loss: %s")

	// ResourceExhaustedError creates resource exhausted errors
	ResourceExhaustedError = NewTemplate().
				WithClass(ClassResourceExhausted).
				WithSeverity(SeverityHigh).
				WithMessageTemplate("resource exhausted: %s")

	// UnavailableError creates unavailable errors
	UnavailableError = NewTemplate().
				WithClass(ClassUnavailable).
				WithSeverity(SeverityHigh).
				WithMessageTemplate("service unavailable: %s")

	// CancelledError creates cancelled errors
	CancelledError = NewTemplate().
			WithClass(ClassCancelled).
			WithSeverity(SeverityLow).
			WithMessageTemplate("operation cancelled: %s")

	// NotImplementedError creates not implemented errors
	NotImplementedError = NewTemplate().
				WithClass(ClassNotImplemented).
				WithSeverity(SeverityMedium).
				WithMessageTemplate("not implemented: %s")

	// AlreadyExistsError creates already exists errors
	AlreadyExistsError = NewTemplate().
				WithClass(ClassAlreadyExists).
				WithSeverity(SeverityMedium).
				WithMessageTemplate("already exists: %s")
)
