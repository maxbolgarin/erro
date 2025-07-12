package erro_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestNewErroPackage(t *testing.T) {
	// Test basic error creation with chaining
	err := erro.New("database connection failed", "host", "localhost", "port", "5432").
		Code("DB_001").
		Category("infrastructure").
		Severity("high").
		Retryable(true).
		Tags("database", "connection")

	// Test error message
	expectedMsg := "[HIGH] database connection failed host=localhost port=5432"
	if err.Error() != expectedMsg {
		t.Errorf("Expected message: %s, got: %s", expectedMsg, err.Error())
	}

	// Test metadata
	if err.GetCode() != "DB_001" {
		t.Errorf("Expected code: DB_001, got: %s", err.GetCode())
	}

	if err.GetCategory() != "infrastructure" {
		t.Errorf("Expected category: infrastructure, got: %s", err.GetCategory())
	}

	if err.GetSeverity() != "high" {
		t.Errorf("Expected severity: high, got: %s", err.GetSeverity())
	}

	if !err.IsRetryable() {
		t.Error("Expected error to be retryable")
	}

	tags := err.GetTags()
	if len(tags) != 2 || tags[0] != "database" || tags[1] != "connection" {
		t.Errorf("Expected tags: [database, connection], got: %v", tags)
	}
}

func TestErrorWrapping(t *testing.T) {
	// Create base error
	baseErr := erro.New("connection timeout", "timeout", "30s").
		Code("TIMEOUT").
		Category("network")

	// Wrap it
	wrappedErr := erro.Wrap(baseErr, "operation failed", "operation", "user_login").
		Severity("critical")

	// Test that base error context is preserved
	if wrappedErr.GetCode() != "TIMEOUT" {
		t.Errorf("Expected code: TIMEOUT, got: %s", wrappedErr.GetCode())
	}

	if wrappedErr.GetCategory() != "network" {
		t.Errorf("Expected category: network, got: %s", wrappedErr.GetCategory())
	}

	if wrappedErr.GetSeverity() != "critical" {
		t.Errorf("Expected severity: critical, got: %s", wrappedErr.GetSeverity())
	}

	// Test that both errors point to the same base
	if wrappedErr.GetBase() != baseErr.GetBase() {
		t.Error("Wrapped error should point to the same base error")
	}

	// Test error message
	expectedMsg := "[CRIT] operation failed operation=user_login: connection timeout timeout=30s"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("Expected: %s, got: %s", expectedMsg, wrappedErr.Error())
	}
}

func TestExternalErrorWrapping(t *testing.T) {
	// Create external error
	externalErr := fmt.Errorf("external error")

	// Wrap it with erro
	wrappedErr := erro.Wrap(externalErr, "failed to process", "step", "validation").
		Code("VALIDATION_ERROR").
		Category("business")

	// Test that it created a new base error
	if wrappedErr.GetCode() != "VALIDATION_ERROR" {
		t.Errorf("Expected code: VALIDATION_ERROR, got: %s", wrappedErr.GetCode())
	}

	// Test error message includes external error
	expectedMsg := "failed to process step=validation: external error"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("Expected: %s, got: %s", expectedMsg, wrappedErr.Error())
	}
}

func TestContextExtraction(t *testing.T) {
	ctx := context.WithValue(context.Background(), "requestID", "12345")

	err := erro.New("operation failed", "user", "alice", "action", "transfer").
		Context(ctx).
		Code("OP_001").
		Category("business").
		Severity("medium").
		Retryable(true)

	// Extract context
	errorCtx := erro.ExtractContext(err)
	if errorCtx == nil {
		t.Fatal("Expected error context, got nil")
	}

	// Test basic fields
	if errorCtx.Message != "operation failed" {
		t.Errorf("Expected message: operation failed, got: %s", errorCtx.Message)
	}

	if errorCtx.Code != "OP_001" {
		t.Errorf("Expected code: OP_001, got: %s", errorCtx.Code)
	}

	if errorCtx.Category != "business" {
		t.Errorf("Expected category: business, got: %s", errorCtx.Category)
	}

	if errorCtx.Severity != "medium" {
		t.Errorf("Expected severity: medium, got: %s", errorCtx.Severity)
	}

	if !errorCtx.Retryable {
		t.Error("Expected error to be retryable")
	}

	// Test fields
	if len(errorCtx.Fields) != 2 {
		t.Errorf("Expected 2 fields, got: %d", len(errorCtx.Fields))
	}

	if errorCtx.Fields["user"] != "alice" {
		t.Errorf("Expected user: alice, got: %v", errorCtx.Fields["user"])
	}

	if errorCtx.Fields["action"] != "transfer" {
		t.Errorf("Expected action: transfer, got: %v", errorCtx.Fields["action"])
	}

	// Test function context
	if errorCtx.Function == "" {
		t.Error("Expected function name to be extracted")
	}

	// Test context
	if errorCtx.Context == nil {
		t.Error("Expected context to be preserved")
	}

	if requestID := errorCtx.Context.Value("requestID"); requestID != "12345" {
		t.Errorf("Expected requestID: 12345, got: %v", requestID)
	}
}

func TestLoggingIntegration(t *testing.T) {
	err := erro.New("database query failed", "table", "users", "query_time", "250ms").
		Code("DB_SLOW").
		Category("performance").
		Severity("warning")

	// Test field extraction for logging
	fields := erro.LogFieldsMap(err)
	if fields == nil {
		t.Fatal("Expected fields map, got nil")
	}

	// Should contain user fields
	if fields["table"] != "users" {
		t.Errorf("Expected table: users, got: %v", fields["table"])
	}

	if fields["query_time"] != "250ms" {
		t.Errorf("Expected query_time: 250ms, got: %v", fields["query_time"])
	}

	// Should contain metadata with error_ prefix (this is how LogFieldsMap actually works)
	if fields["error_code"] != "DB_SLOW" {
		t.Errorf("Expected code: DB_SLOW, got: %v", fields["error_code"])
	}

	if fields["error_category"] != erro.Category("performance") {
		t.Errorf("Expected category: performance, got: %v", fields["error_category"])
	}

	// Should contain stack context
	if fields["error_function"] == nil {
		t.Error("Expected error_function to be present")
	}

	// Test callback logging
	var loggedMessage string
	var loggedFields []any

	erro.LogError(err, func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	})

	if loggedMessage != "database query failed" {
		t.Errorf("Expected message: database query failed, got: %s", loggedMessage)
	}

	if loggedFields[1] != "users" {
		t.Errorf("Expected table: users, got: %v", loggedFields[1])
	}
}

func TestStackTracing(t *testing.T) {
	err := processUser("alice")

	// Test that we can get stack information
	stack := err.Stack()
	if len(stack) == 0 {
		t.Fatal("Expected stack trace, got empty")
	}

	// Test stack context
	topFrame := stack[0]
	if topFrame.Name == "" {
		t.Error("Expected function name in stack frame")
	}

	if topFrame.File == "" {
		t.Error("Expected file name in stack frame")
	}

	if topFrame.Line == 0 {
		t.Error("Expected line number in stack frame")
	}

	// Test user code detection
	stackType := erro.Stack(stack)
	userFrames := stackType.UserFrames()
	if len(userFrames) == 0 {
		t.Error("Expected user code frames")
	}

	// Test context extraction from stack
	if origin := stackType.GetOriginContext(); origin != nil {
		if origin.Function == "" {
			t.Error("Expected function name from origin context")
		}
	}
}

// Helper function for testing stack traces
func processUser(userID string) erro.Error {
	return validateUser(userID)
}

func validateUser(userID string) erro.Error {
	return erro.New("user validation failed", "user_id", userID, "reason", "invalid_format").
		Code("USER_INVALID").
		Category("validation")
}

func TestComplexErrorScenario(t *testing.T) {
	// Simulate a complex error scenario
	baseErr := erro.New("connection refused", "host", "db.example.com", "port", "5432").
		Code("CONN_REFUSED").
		Category("infrastructure").
		Severity("high").
		Retryable(true)

	// Add trace ID and context
	ctx := context.WithValue(context.Background(), "requestID", "req-123")
	baseErr = baseErr.Category("trace-456").Context(ctx)

	// Wrap with business context
	businessErr := erro.Wrap(baseErr, "failed to save user", "user_id", "user-789", "operation", "create").
		Tags("user-management", "database")

	// Final wrap with request context
	finalErr := erro.Wrap(businessErr, "API request failed", "endpoint", "/api/users", "method", "POST")

	// Test that all context is preserved
	if finalErr.GetCode() != "CONN_REFUSED" {
		t.Errorf("Expected code: CONN_REFUSED, got: %s", finalErr.GetCode())
	}

	if finalErr.GetCategory() != "trace-456" {
		t.Errorf("Expected trace ID: trace-456, got: %s", finalErr.GetCategory())
	}

	if len(finalErr.GetTags()) != 2 {
		t.Errorf("Expected 2 tags, got: %d", len(finalErr.GetTags()))
	}

	// Test all errors point to same base
	if finalErr.GetBase() != baseErr.GetBase() {
		t.Error("All wrapped errors should point to the same base")
	}

	// Test comprehensive logging context
	ctx2 := erro.ExtractContext(finalErr)
	if ctx2 == nil {
		t.Fatal("Expected error context")
	}

	// Should have rich fields from all layers
	if len(ctx2.Fields) < 6 { // Should have fields from all wrapping layers
		t.Errorf("Expected at least 6 fields, got: %d", len(ctx2.Fields))
	}

	// Test logging integration
	fields := erro.LogFieldsMap(finalErr)
	if len(fields) < 10 { // Should have user fields + metadata + stack context
		t.Errorf("Expected rich logging context, got only %d fields", len(fields))
	}

	// Verify specific context
	if fields["host"] != "db.example.com" {
		t.Errorf("Expected host from base error: db.example.com, got: %v", fields["host"])
	}

	if fields["endpoint"] != "/api/users" {
		t.Errorf("Expected endpoint from final wrap: /api/users, got: %v", fields["endpoint"])
	}

	if fields["user_id"] != "user-789" {
		t.Errorf("Expected user_id from business wrap: user-789, got: %v", fields["user_id"])
	}

	fmt.Printf("Final error: %s\n", finalErr.Error())
	fmt.Printf("Stack trace:\n%+v\n", finalErr)
	fmt.Printf("Logging fields: %+v\n", fields)
}

func TestSeverityRefactoring(t *testing.T) {
	// Test ErrorSeverity type and predefined constants
	t.Run("ErrorSeverity type", func(t *testing.T) {
		// Test that predefined severities are valid
		severities := []erro.Severity{
			erro.SeverityUnknown,
			erro.SeverityCritical,
			erro.SeverityHigh,
			erro.SeverityMedium,
			erro.SeverityLow,
			erro.SeverityInfo,
		}

		for _, severity := range severities {
			if !severity.IsValid() {
				t.Errorf("Expected severity %s to be valid", severity)
			}
		}

		// Test invalid severity
		invalid := erro.Severity("invalid")
		if invalid.IsValid() {
			t.Error("Expected invalid severity to be invalid")
		}
	})

	t.Run("Severity checking methods", func(t *testing.T) {
		// Test with critical severity
		criticalErr := erro.New("critical error").Severity("critical")
		if !criticalErr.IsCritical() {
			t.Error("Expected error to be critical")
		}
		if criticalErr.IsHigh() || criticalErr.IsLow() || criticalErr.IsInfo() {
			t.Error("Expected error to only be critical")
		}

		// Test with high severity
		highErr := erro.New("high error").Severity("high")
		if !highErr.IsHigh() {
			t.Error("Expected error to be high")
		}
		if highErr.IsCritical() || highErr.IsLow() || highErr.IsInfo() {
			t.Error("Expected error to only be high")
		}

		// Test with unknown severity (empty)
		unknownErr := erro.New("unknown error")
		if !unknownErr.IsUnknown() {
			t.Error("Expected error with no severity to be unknown")
		}
		if unknownErr.IsCritical() || unknownErr.IsHigh() || unknownErr.IsLow() {
			t.Error("Expected error to only be unknown")
		}
	})

	t.Run("GetSeverityLevel method", func(t *testing.T) {
		// Test with predefined severity
		err := erro.New("test error").Severity("high")
		if err.GetSeverity() != erro.SeverityHigh {
			t.Errorf("Expected severity level to be SeverityHigh, got %s", err.GetSeverity())
		}

		// Test with unknown severity
		unknownErr := erro.New("unknown error")
		if unknownErr.GetSeverity() != erro.SeverityUnknown {
			t.Errorf("Expected severity level to be SeverityUnknown, got %s", unknownErr.GetSeverity())
		}
	})

	t.Run("List severity methods", func(t *testing.T) {
		list := erro.NewList().Severity(erro.SeverityCritical)
		if !list.IsCritical() {
			t.Error("Expected list to be critical")
		}
		if list.GetSeverity() != erro.SeverityCritical {
			t.Errorf("Expected list severity level to be SeverityCritical, got %s", list.GetSeverity())
		}

		// Add an error and verify it inherits the severity
		list.New("test error")
		if list.Len() != 1 {
			t.Error("Expected list to have 1 error")
		}

		err := list.First()
		if err == nil {
			t.Fatal("Expected first error to exist")
		}
		if !err.IsCritical() {
			t.Error("Expected error to inherit critical severity from list")
		}
	})

	t.Run("ErrorContext severity methods", func(t *testing.T) {
		err := erro.New("test error").Severity("info")
		ctx := erro.ExtractContext(err)
		if ctx == nil {
			t.Fatal("Expected context to exist")
		}

		if !ctx.IsInfo() {
			t.Error("Expected context to be info")
		}
		if ctx.GetSeverity() != erro.SeverityInfo {
			t.Errorf("Expected context severity level to be SeverityInfo, got %s", ctx.GetSeverity())
		}
	})
}

func TestErrorfAndWrapf(t *testing.T) {
	// Test Errorf with format string and fields
	err1 := erro.Errorf("failed to connect to %s:%d", "database", 5432, "timeout", "30s", "retries", 3)

	expectedMsg := "failed to connect to database:5432 timeout=30s retries=3"
	if err1.Error() != expectedMsg {
		t.Errorf("Expected: %s, got: %s", expectedMsg, err1.Error())
	}

	// Test Errorf with only format args
	err2 := erro.Errorf("user %s not found", "alice")
	expectedMsg2 := "user alice not found"
	if err2.Error() != expectedMsg2 {
		t.Errorf("Expected: %s, got: %s", expectedMsg2, err2.Error())
	}

	// Test Errorf with no args
	err3 := erro.Errorf("simple error")
	if err3.Error() != "simple error" {
		t.Errorf("Expected: simple error, got: %s", err3.Error())
	}

	// Test Wrapf with external error
	baseErr := fmt.Errorf("connection failed")
	err4 := erro.Wrapf(baseErr, "retry %d of %d failed", 2, 3, "service", "auth", "user_id", 123)

	expectedMsg4 := "retry 2 of 3 failed service=auth user_id=123: connection failed"
	if err4.Error() != expectedMsg4 {
		t.Errorf("Expected: %s, got: %s", expectedMsg4, err4.Error())
	}

	// Test Wrapf with erro error
	baseErroErr := erro.New("database error", "table", "users").Code("DB_001")
	err5 := erro.Wrapf(baseErroErr, "operation %s failed", "insert", "operation_id", "op-456")

	// Should preserve base error code
	if err5.GetCode() != "DB_001" {
		t.Errorf("Expected code: DB_001, got: %s", err5.GetCode())
	}

	// Test Wrapf with nil error (should act like Errorf)
	err6 := erro.Wrapf(nil, "created from nil: %s", "test", "field", "value")
	expectedMsg6 := "created from nil: test field=value"
	if err6.Error() != expectedMsg6 {
		t.Errorf("Expected: %s, got: %s", expectedMsg6, err6.Error())
	}

	// Test chaining after Errorf/Wrapf
	err7 := erro.Errorf("api call failed: %s", "timeout", "duration", "5s").
		Code("API_TIMEOUT").
		Category("external").
		Severity("high")

	if err7.GetCode() != "API_TIMEOUT" {
		t.Errorf("Expected code: API_TIMEOUT, got: %s", err7.GetCode())
	}

	if err7.GetCategory() != "external" {
		t.Errorf("Expected category: external, got: %s", err7.GetCategory())
	}

	// Verify fields are preserved
	fields := err7.GetFields()
	if len(fields) != 2 || fields[0] != "duration" || fields[1] != "5s" {
		t.Errorf("Expected fields [duration, 5s], got: %v", fields)
	}

	fmt.Printf("Errorf example: %s\n", err1.Error())
	fmt.Printf("Wrapf example: %s\n", err4.Error())
}
