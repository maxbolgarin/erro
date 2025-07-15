package erro

import "fmt"

// ErrorTemplate represents a template for creating errors with predefined metadata
type ErrorTemplate struct {
	messageTemplate string
	opts            []any
}

// NewTemplate creates a new error template
func NewTemplate(messageTemplate string, opts ...any) *ErrorTemplate {
	return &ErrorTemplate{
		messageTemplate: messageTemplate,
		opts:            opts,
	}
}

// Create creates an error using the template
func (t *ErrorTemplate) New(fields ...any) Error {
	numVerbs := countVerbs(t.messageTemplate)
	if len(fields) < numVerbs {
		// Not enough arguments for the format string.
		// This is a programmer error, but we should handle it gracefully.
		return newf(t.messageTemplate, mergeFields(fields, t.opts)...)
	}

	formatArgs := fields[:numVerbs]
	metaFields := fields[numVerbs:]

	message := fmt.Sprintf(t.messageTemplate, formatArgs...)
	return newf(message, mergeFields(metaFields, t.opts)...)
}

func (t *ErrorTemplate) Wrap(originalErr error, fields ...any) Error {
	numVerbs := countVerbs(t.messageTemplate)
	if len(fields) < numVerbs {
		return wrapf(originalErr, t.messageTemplate, mergeFields(fields, t.opts)...)
	}

	formatArgs := fields[:numVerbs]
	metaFields := fields[numVerbs:]

	message := fmt.Sprintf(t.messageTemplate, formatArgs...)
	return wrapf(originalErr, message, mergeFields(metaFields, t.opts)...)
}

// Predefined templates with message templates
var (
	// ValidationError creates validation errors
	ValidationError = NewTemplate("failed validation: %s",
		ClassValidation,
		CategoryUserInput,
		SeverityLow,
	)

	// NotFoundError creates not found errors
	NotFoundError = NewTemplate("%s not found",
		ClassNotFound,
		SeverityMedium,
	)

	// DatabaseError creates database errors
	DatabaseError = NewTemplate("database error: %s",
		CategoryDatabase,
		SeverityHigh,
	)

	// NetworkError creates network errors
	NetworkError = NewTemplate("network error: %s",
		CategoryNetwork,
		SeverityMedium,
		Retryable(),
	)

	// AuthenticationError creates authentication errors
	AuthenticationError = NewTemplate("authentication failed: %s",
		CategoryAuth,
		ClassUnauthenticated,
		SeverityMedium,
	)

	// AuthorizationError creates authorization errors
	AuthorizationError = NewTemplate("permission denied: %s",
		CategoryAuth,
		ClassPermissionDenied,
		SeverityHigh,
	)

	// TimeoutError creates timeout errors
	TimeoutError = NewTemplate("operation timeout: %s",
		ClassTimeout,
		SeverityLow,
		Retryable(),
	)

	// ConflictError creates conflict errors
	ConflictError = NewTemplate("conflict: %s",
		ClassConflict,
		SeverityMedium,
	)

	// RateLimitError creates rate limit errors
	RateLimitError = NewTemplate("rate limit exceeded: %s",
		ClassRateLimited,
		SeverityLow,
		Retryable(),
	)

	// InternalError creates internal errors
	InternalError = NewTemplate("internal error: %s",
		ClassInternal,
		SeverityHigh,
	)

	// SecurityError creates security errors
	SecurityError = NewTemplate("security violation: %s",
		ClassSecurity,
		CategorySecurity,
		SeverityCritical,
	)

	// ExternalError creates external service errors
	ExternalError = NewTemplate("external error: %s",
		ClassExternal,
		CategoryExternal,
		SeverityMedium,
		Retryable(),
	)

	// PaymentError creates payment errors
	PaymentError = NewTemplate("payment error: %s",
		CategoryPayment,
		SeverityCritical,
		Retryable(),
	)

	// CacheError creates cache errors
	CacheError = NewTemplate("cache error: %s",
		CategoryCache,
		ClassTemporary,
		SeverityMedium,
		Retryable(),
	)

	// ConfigError creates configuration errors
	ConfigError = NewTemplate("configuration error: %s",
		CategoryConfig,
		ClassInternal,
		SeverityCritical,
	)

	// APIError creates API errors
	APIError = NewTemplate("API error: %s",
		CategoryAPI,
		SeverityHigh,
	)

	// BusinessLogicError creates business logic errors
	BusinessLogicError = NewTemplate("business logic error: %s",
		CategoryBusinessLogic,
		SeverityHigh,
	)

	// StorageError creates storage errors
	StorageError = NewTemplate("storage error: %s",
		CategoryStorage,
		SeverityHigh,
	)

	// ProcessingError creates processing errors
	ProcessingError = NewTemplate("processing error: %s",
		CategoryProcessing,
		ClassInternal,
		SeverityHigh,
	)

	// MonitoringError creates monitoring errors
	MonitoringError = NewTemplate("monitoring error: %s",
		CategoryMonitoring,
		SeverityMedium,
	)

	// NotificationError creates notification errors
	NotificationError = NewTemplate("notification error: %s",
		CategoryNotifications,
		ClassTemporary,
		SeverityLow,
	)

	// AIError creates AI/ML errors
	AIError = NewTemplate("AI error: %s",
		CategoryAI,
		ClassInternal,
		SeverityHigh,
	)

	// AnalyticsError creates analytics errors
	AnalyticsError = NewTemplate("analytics error: %s",
		CategoryAnalytics,
		SeverityLow,
	)

	// EventsTemplate creates events errors
	EventsTemplate = NewTemplate("events error: %s",
		CategoryEvents,
		SeverityMedium,
	)

	// CriticalError creates critical errors
	CriticalError = NewTemplate("critical error: %s",
		ClassCritical,
		SeverityCritical,
	)

	// TemporaryError creates temporary errors
	TemporaryError = NewTemplate("temporary error: %s",
		ClassTemporary,
		SeverityMedium,
	)

	// DataLossError creates data loss errors
	DataLossError = NewTemplate("data loss: %s",
		ClassDataLoss,
		SeverityCritical,
	)

	// ResourceExhaustedError creates resource exhausted errors
	ResourceExhaustedError = NewTemplate("resource exhausted: %s",
		ClassResourceExhausted,
		SeverityHigh,
	)

	// UnavailableError creates unavailable errors
	UnavailableError = NewTemplate("service unavailable: %s",
		ClassUnavailable,
		SeverityHigh,
	)

	// CancelledError creates cancelled errors
	CancelledError = NewTemplate("operation cancelled: %s",
		ClassCancelled,
		SeverityLow,
	)

	// NotImplementedError creates not implemented errors
	NotImplementedError = NewTemplate("not implemented: %s",
		ClassNotImplemented,
		SeverityMedium,
	)

	// AlreadyExistsError creates already exists errors
	AlreadyExistsError = NewTemplate("already exists: %s",
		ClassAlreadyExists,
		SeverityMedium,
	)
)
