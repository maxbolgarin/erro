package erro

import "fmt"

// ErrorTemplate represents a template for creating errors with predefined metadata.
type ErrorTemplate struct {
	messageTemplate string
	opts            []any
}

// NewTemplate creates a new error template with a message and predefined options.
func NewTemplate(messageTemplate string, opts ...any) *ErrorTemplate {
	return &ErrorTemplate{
		messageTemplate: messageTemplate,
		opts:            opts,
	}
}

// New creates an error from the template.
func (t *ErrorTemplate) New(fields ...any) Error {
	numVerbs := countVerbs(t.messageTemplate)
	if len(fields) < numVerbs {
		return newBaseError(t.messageTemplate, mergeFields(fields, t.opts)...)
	}

	formatArgs := fields[:numVerbs]
	metaFields := fields[numVerbs:]

	message := fmt.Sprintf(t.messageTemplate, formatArgs...)
	return newf(message, mergeFields(metaFields, t.opts)...)
}

// Wrap wraps an existing error with the template's message and options.
func (t *ErrorTemplate) Wrap(originalErr error, fields ...any) Error {
	numVerbs := countVerbs(t.messageTemplate)
	if len(fields) < numVerbs {
		return newWrapError(originalErr, t.messageTemplate, mergeFields(fields, t.opts)...)
	}

	formatArgs := fields[:numVerbs]
	metaFields := fields[numVerbs:]

	message := fmt.Sprintf(t.messageTemplate, formatArgs...)
	return wrapf(originalErr, message, mergeFields(metaFields, t.opts)...)
}

// Predefined error templates.
var (
	// ValidationError creates a new validation error.
	ValidationError = NewTemplate("failed validation: %s",
		ClassValidation,
		CategoryUserInput,
		SeverityLow,
	)

	// NotFoundError creates a new not found error.
	NotFoundError = NewTemplate("%s not found",
		ClassNotFound,
		SeverityMedium,
	)

	// DatabaseError creates a new database error.
	DatabaseError = NewTemplate("database error: %s",
		CategoryDatabase,
		SeverityHigh,
	)

	// NetworkError creates a new network error.
	NetworkError = NewTemplate("network error: %s",
		CategoryNetwork,
		SeverityMedium,
		Retryable(),
	)

	// AuthenticationError creates a new authentication error.
	AuthenticationError = NewTemplate("authentication failed: %s",
		CategoryAuth,
		ClassUnauthenticated,
		SeverityMedium,
	)

	// AuthorizationError creates a new authorization error.
	AuthorizationError = NewTemplate("permission denied: %s",
		CategoryAuth,
		ClassPermissionDenied,
		SeverityHigh,
	)

	// TimeoutError creates a new timeout error.
	TimeoutError = NewTemplate("operation timeout: %s",
		ClassTimeout,
		SeverityLow,
		Retryable(),
	)

	// ConflictError creates a new conflict error.
	ConflictError = NewTemplate("conflict: %s",
		ClassConflict,
		SeverityMedium,
	)

	// RateLimitError creates a new rate limit error.
	RateLimitError = NewTemplate("rate limit exceeded: %s",
		ClassRateLimited,
		SeverityLow,
		Retryable(),
	)

	// InternalError creates a new internal error.
	InternalError = NewTemplate("internal error: %s",
		ClassInternal,
		SeverityHigh,
	)

	// SecurityError creates a new security error.
	SecurityError = NewTemplate("security violation: %s",
		ClassSecurity,
		CategorySecurity,
		SeverityCritical,
	)

	// ExternalError creates a new external service error.
	ExternalError = NewTemplate("external error: %s",
		ClassExternal,
		CategoryExternal,
		SeverityMedium,
		Retryable(),
	)

	// PaymentError creates a new payment error.
	PaymentError = NewTemplate("payment error: %s",
		CategoryPayment,
		SeverityCritical,
		Retryable(),
	)

	// CacheError creates a new cache error.
	CacheError = NewTemplate("cache error: %s",
		CategoryCache,
		ClassTemporary,
		SeverityMedium,
		Retryable(),
	)

	// ConfigError creates a new configuration error.
	ConfigError = NewTemplate("configuration error: %s",
		CategoryConfig,
		ClassInternal,
		SeverityCritical,
	)

	// APIError creates a new API error.
	APIError = NewTemplate("API error: %s",
		CategoryAPI,
		SeverityHigh,
	)

	// BusinessLogicError creates a new business logic error.
	BusinessLogicError = NewTemplate("business logic error: %s",
		CategoryBusinessLogic,
		SeverityHigh,
	)

	// StorageError creates a new storage error.
	StorageError = NewTemplate("storage error: %s",
		CategoryStorage,
		SeverityHigh,
	)

	// ProcessingError creates a new processing error.
	ProcessingError = NewTemplate("processing error: %s",
		CategoryProcessing,
		ClassInternal,
		SeverityHigh,
	)

	// MonitoringError creates a new monitoring error.
	MonitoringError = NewTemplate("monitoring error: %s",
		CategoryMonitoring,
		SeverityMedium,
	)

	// NotificationError creates a new notification error.
	NotificationError = NewTemplate("notification error: %s",
		CategoryNotifications,
		ClassTemporary,
		SeverityLow,
	)

	// AIError creates a new AI/ML error.
	AIError = NewTemplate("AI error: %s",
		CategoryAI,
		ClassInternal,
		SeverityHigh,
	)

	// AnalyticsError creates a new analytics error.
	AnalyticsError = NewTemplate("analytics error: %s",
		CategoryAnalytics,
		SeverityLow,
	)

	// EventsTemplate creates a new events error.
	EventsTemplate = NewTemplate("events error: %s",
		CategoryEvents,
		SeverityMedium,
	)

	// CriticalError creates a new critical error.
	CriticalError = NewTemplate("critical error: %s",
		ClassCritical,
		SeverityCritical,
	)

	// TemporaryError creates a new temporary error.
	TemporaryError = NewTemplate("temporary error: %s",
		ClassTemporary,
		SeverityMedium,
	)

	// DataLossError creates a new data loss error.
	DataLossError = NewTemplate("data loss: %s",
		ClassDataLoss,
		SeverityCritical,
	)

	// ResourceExhaustedError creates a new resource exhausted error.
	ResourceExhaustedError = NewTemplate("resource exhausted: %s",
		ClassResourceExhausted,
		SeverityHigh,
	)

	// UnavailableError creates a new unavailable error.
	UnavailableError = NewTemplate("service unavailable: %s",
		ClassUnavailable,
		SeverityHigh,
	)

	// CancelledError creates a new cancelled error.
	CancelledError = NewTemplate("operation cancelled: %s",
		ClassCancelled,
		SeverityLow,
	)

	// NotImplementedError creates a new not implemented error.
	NotImplementedError = NewTemplate("not implemented: %s",
		ClassNotImplemented,
		SeverityMedium,
	)

	// AlreadyExistsError creates a new already exists error.
	AlreadyExistsError = NewTemplate("already exists: %s",
		ClassAlreadyExists,
		SeverityMedium,
	)
)
