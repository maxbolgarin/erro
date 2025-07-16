package erro_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestTemplate(t *testing.T) {
	tmpl := erro.NewTemplate("template error: %s",
		erro.ClassValidation,
		erro.CategoryUserInput,
		erro.SeverityHigh,
		erro.Retryable(),
		erro.ID("TEMPLATE_ID"),
	)

	err := tmpl.New("test message", "key", "value")

	if err.ID() != "TEMPLATE_ID" {
		t.Errorf("Expected ID 'TEMPLATE_ID', got '%s'", err.ID())
	}
	if err.Class() != erro.ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryUserInput {
		t.Errorf("Expected category 'user_input', got '%s'", err.Category())
	}
	if err.Severity() != erro.SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", err.Severity())
	}
	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	if !strings.Contains(err.Error(), "template error: test message") {
		t.Errorf("Expected message 'template error: test message', got '%s'", err.Error())
	}
	fields := err.Fields()
	if len(fields) != 2 || fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
}

func TestTemplate_NoArgs(t *testing.T) {
	tmpl := erro.NewTemplate("template error: %s",
		erro.ClassValidation,
		erro.CategoryUserInput,
		erro.SeverityHigh,
		erro.Retryable(),
		erro.ID("TEMPLATE_ID"),
	)

	err := tmpl.New()

	if err.ID() != "TEMPLATE_ID" {
		t.Errorf("Expected ID 'TEMPLATE_ID', got '%s'", err.ID())
	}
	if err.Class() != erro.ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryUserInput {
		t.Errorf("Expected category 'user_input', got '%s'", err.Category())
	}
	if err.Severity() != erro.SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", err.Severity())
	}
	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	if !strings.Contains(err.Error(), "template error") {
		t.Errorf("Expected message 'template error', got '%s'", err.Error())
	}
}

func TestTemplateWrap(t *testing.T) {
	tmpl := erro.NewTemplate("template wrap: %s",
		erro.ClassInternal,
		erro.CategoryDatabase,
	)

	baseErr := errors.New("base error")
	err := tmpl.Wrap(baseErr, "wrapped message", "key", "value")

	if err.Class() != erro.ClassInternal {
		t.Errorf("Expected class 'internal', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", err.Category())
	}
	if !strings.Contains(err.Error(), "template wrap: wrapped message") {
		t.Errorf("Expected message 'template wrap: wrapped message', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "base error") {
		t.Errorf("Expected message to contain 'base error', got '%s'", err.Error())
	}
}

func TestTemplateWrap_NoArgs(t *testing.T) {
	tmpl := erro.NewTemplate("template wrap: %s",
		erro.ClassInternal,
		erro.CategoryDatabase,
	)

	baseErr := errors.New("base error")
	err := tmpl.Wrap(baseErr)

	if err.Class() != erro.ClassInternal {
		t.Errorf("Expected class 'internal', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", err.Category())
	}
	if !strings.Contains(err.Error(), "template wrap") {
		t.Errorf("Expected message 'template wrap', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "base error") {
		t.Errorf("Expected message to contain 'base error', got '%s'", err.Error())
	}
}

func TestPredefinedTemplates(t *testing.T) {
	testCases := []struct {
		name     string
		template *erro.ErrorTemplate
		class    erro.ErrorClass
		category erro.ErrorCategory
		severity erro.ErrorSeverity
	}{
		{"ValidationError", erro.ValidationError, erro.ClassValidation, erro.CategoryUserInput, erro.SeverityLow},
		{"NotFoundError", erro.NotFoundError, erro.ClassNotFound, "", erro.SeverityMedium},
		{"DatabaseError", erro.DatabaseError, "", erro.CategoryDatabase, erro.SeverityHigh},
		{"NetworkError", erro.NetworkError, "", erro.CategoryNetwork, erro.SeverityMedium},
		{"AuthenticationError", erro.AuthenticationError, erro.ClassUnauthenticated, erro.CategoryAuth, erro.SeverityMedium},
		{"AuthorizationError", erro.AuthorizationError, erro.ClassPermissionDenied, erro.CategoryAuth, erro.SeverityHigh},
		{"TimeoutError", erro.TimeoutError, erro.ClassTimeout, "", erro.SeverityLow},
		{"ConflictError", erro.ConflictError, erro.ClassConflict, "", erro.SeverityMedium},
		{"RateLimitError", erro.RateLimitError, erro.ClassRateLimited, "", erro.SeverityLow},
		{"InternalError", erro.InternalError, erro.ClassInternal, "", erro.SeverityHigh},
		{"SecurityError", erro.SecurityError, erro.ClassSecurity, erro.CategorySecurity, erro.SeverityCritical},
		{"ExternalError", erro.ExternalError, erro.ClassExternal, erro.CategoryExternal, erro.SeverityMedium},
		{"PaymentError", erro.PaymentError, "", erro.CategoryPayment, erro.SeverityCritical},
		{"CacheError", erro.CacheError, erro.ClassTemporary, erro.CategoryCache, erro.SeverityMedium},
		{"ConfigError", erro.ConfigError, erro.ClassInternal, erro.CategoryConfig, erro.SeverityCritical},
		{"APIError", erro.APIError, "", erro.CategoryAPI, erro.SeverityHigh},
		{"BusinessLogicError", erro.BusinessLogicError, "", erro.CategoryBusinessLogic, erro.SeverityHigh},
		{"StorageError", erro.StorageError, "", erro.CategoryStorage, erro.SeverityHigh},
		{"ProcessingError", erro.ProcessingError, erro.ClassInternal, erro.CategoryProcessing, erro.SeverityHigh},
		{"MonitoringError", erro.MonitoringError, "", erro.CategoryMonitoring, erro.SeverityMedium},
		{"NotificationError", erro.NotificationError, erro.ClassTemporary, erro.CategoryNotifications, erro.SeverityLow},
		{"AIError", erro.AIError, erro.ClassInternal, erro.CategoryAI, erro.SeverityHigh},
		{"AnalyticsError", erro.AnalyticsError, "", erro.CategoryAnalytics, erro.SeverityLow},
		{"EventsTemplate", erro.EventsTemplate, "", erro.CategoryEvents, erro.SeverityMedium},
		{"CriticalError", erro.CriticalError, erro.ClassCritical, "", erro.SeverityCritical},
		{"TemporaryError", erro.TemporaryError, erro.ClassTemporary, "", erro.SeverityMedium},
		{"DataLossError", erro.DataLossError, erro.ClassDataLoss, "", erro.SeverityCritical},
		{"ResourceExhaustedError", erro.ResourceExhaustedError, erro.ClassResourceExhausted, "", erro.SeverityHigh},
		{"UnavailableError", erro.UnavailableError, erro.ClassUnavailable, "", erro.SeverityHigh},
		{"CancelledError", erro.CancelledError, erro.ClassCancelled, "", erro.SeverityLow},
		{"NotImplementedError", erro.NotImplementedError, erro.ClassNotImplemented, "", erro.SeverityMedium},
		{"AlreadyExistsError", erro.AlreadyExistsError, erro.ClassAlreadyExists, "", erro.SeverityMedium},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.template.New("test")
			if err.Class() != tc.class {
				t.Errorf("Expected class '%s', got '%s'", tc.class, err.Class())
			}
			if err.Category() != tc.category {
				t.Errorf("Expected category '%s', got '%s'", tc.category, err.Category())
			}
			if err.Severity() != tc.severity {
				t.Errorf("Expected severity '%s', got '%s'", tc.severity, err.Severity())
			}
		})
	}
}
