package erro

import (
	"context"
	"testing"
)

func TestID(t *testing.T) {
	err := New("test", ID("123"))
	if err.ID() != "123" {
		t.Errorf("expected ID 123, got %s", err.ID())
	}
}

func TestRetryable(t *testing.T) {
	err := New("test", Retryable())
	if !err.IsRetryable() {
		t.Errorf("expected retryable to be true")
	}
}

func TestFields(t *testing.T) {
	err := New("test", Fields("key", "value"))
	fields := err.Fields()
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}

func TestFormatter(t *testing.T) {
	customFormatter := func(err Error) string {
		return "custom format"
	}
	err := New("test", Formatter(customFormatter))
	if err.Error() != "custom format" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
	err = New("test", Formatter(nil)) // should not panic
}

func TestStackTrace(t *testing.T) {
	err := New("test", StackTrace())
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTrace(nil))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTrace(nil, nil))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTrace(&StackTraceConfig{}))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTrace(DevelopmentStackTraceConfig()))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
}

func TestStackTraceWithSkip(t *testing.T) {
	err := New("test", StackTraceWithSkip(1))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTraceWithSkip(-1, &StackTraceConfig{}))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
	err = New("test", StackTraceWithSkip(1, DevelopmentStackTraceConfig()))
	if err.Stack() == nil {
		t.Error("expected stack trace to be captured")
	}
}

type mockMetrics struct {
	recorded bool
}

func (m *mockMetrics) RecordError(err Error) {
	m.recorded = true
}

func TestRecordMetrics(t *testing.T) {
	metrics := &mockMetrics{}
	_ = New("test", RecordMetrics(metrics))
	if !metrics.recorded {
		t.Error("expected metrics to be recorded")
	}
	_ = New("test", RecordMetrics(nil)) // should not panic
}

type mockDispatcher struct {
	sent bool
}

func (d *mockDispatcher) SendEvent(ctx context.Context, err Error) {
	d.sent = true
}

func TestSendEvent(t *testing.T) {
	dispatcher := &mockDispatcher{}
	_ = New("test", SendEvent(context.Background(), dispatcher))
	if !dispatcher.sent {
		t.Error("expected event to be sent")
	}
	_ = New("test", SendEvent(context.Background(), nil)) // should not panic
}

func TestRecordSpan(t *testing.T) {
	span := &mockTraceSpan{}
	_ = New("test", RecordSpan(span))
	// how to assert this?
	_ = New("test", RecordSpan(nil)) // should not panic
}

func TestErrorSeverity_String(t *testing.T) {
	if SeverityCritical.String() != "critical" {
		t.Errorf("unexpected string for severity")
	}
}

func TestErrorSeverity_IsValid(t *testing.T) {
	if !SeverityCritical.IsValid() {
		t.Error("expected critical severity to be valid")
	}
	if ErrorSeverity("invalid").IsValid() {
		t.Error("expected invalid severity to be invalid")
	}
}

func TestErrorSeverity_Label(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		label    string
	}{
		{SeverityCritical, "[CRIT]"},
		{SeverityHigh, "[HIGH]"},
		{SeverityMedium, "[MED]"},
		{SeverityLow, "[LOW]"},
		{SeverityInfo, "[INFO]"},
		{SeverityUnknown, ""},
		{ErrorSeverity("invalid"), ""},
	}

	for _, tt := range tests {
		if label := tt.severity.Label(); label != tt.label {
			t.Errorf("for severity %s, expected label %s, got %s", tt.severity, tt.label, label)
		}
	}
}

func TestErrorSeverity_Is(t *testing.T) {
	if !SeverityCritical.IsCritical() {
		t.Error("expected critical severity")
	}
	if !SeverityHigh.IsHigh() {
		t.Error("expected high severity")
	}
	if !SeverityMedium.IsMedium() {
		t.Error("expected medium severity")
	}
	if !SeverityLow.IsLow() {
		t.Error("expected low severity")
	}
	if !SeverityInfo.IsInfo() {
		t.Error("expected info severity")
	}
	if !SeverityUnknown.IsUnknown() {
		t.Error("expected unknown severity")
	}
}

func TestErrorClass_String(t *testing.T) {
	if ClassValidation.String() != "validation" {
		t.Errorf("unexpected string for class")
	}
}

func TestErrorCategory_String(t *testing.T) {
	if CategoryDatabase.String() != "database" {
		t.Errorf("unexpected string for category")
	}
}
