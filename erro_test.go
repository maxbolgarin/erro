package erro_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

func TestNew(t *testing.T) {
	err := erro.New("test error")
	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}

	if err.Message() != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", err.Message())
	}
}

func TestNewWithFields(t *testing.T) {
	err := erro.New("test error", "key1", "value1", "key2", 123, nil, nil)
	if !strings.HasPrefix(err.Error(), "test error") {
		t.Errorf("Expected prefix 'test error', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "key1=value1") {
		t.Errorf("Expected to contain 'key1=value1', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "key2=123") {
		t.Errorf("Expected to contain 'key2=123', got '%s'", err.Error())
	}
}

func TestNewWithFieldsOdd(t *testing.T) {
	err := erro.New("test error", "key1", "value1", "key2", 123, "key3")
	if !strings.HasPrefix(err.Error(), "test error") {
		t.Errorf("Expected prefix 'test error', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "key1=value1") {
		t.Errorf("Expected to contain 'key1=value1', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "key2=123") {
		t.Errorf("Expected to contain 'key2=123', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "key3="+erro.MissingFieldPlaceholder) {
		t.Errorf("Expected to contain 'key3="+erro.MissingFieldPlaceholder+"', got '%s'", err.Error())
	}
}

func TestNewWithFieldsMany(t *testing.T) {
	targetLength := 2*erro.MaxFieldsCount + 100

	fields := make([]any, 0, targetLength)
	for i := 0; i < targetLength; i++ {
		fields = append(fields, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	if len(fields) != 2*targetLength {
		t.Errorf("Expected %d fields, got %d", targetLength, len(fields))
	}

	err := erro.New("test error", fields...)
	if len(err.Fields()) != 2*erro.MaxFieldsCount {
		t.Errorf("Expected %d fields, got %d", erro.MaxFieldsCount, len(err.Fields()))
	}
}

func TestWrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := erro.Wrap(baseErr, "wrapped error")

	if !strings.HasSuffix(err.Error(), "base error") {
		t.Errorf("Expected suffix 'base error', got '%s'", err.Error())
	}

	if !strings.HasPrefix(err.Error(), "wrapped error") {
		t.Errorf("Expected prefix 'wrapped error', got '%s'", err.Error())
	}

	unwrapped := erro.Unwrap(err)
	if unwrapped != baseErr {
		t.Errorf("Expected unwrapped error to be baseErr, got %v", unwrapped)
	}
}

func TestWrapNil(t *testing.T) {
	err := erro.Wrap(nil, "wrapped nil")
	if err != nil {
		t.Errorf("Expected nil, got %T %v", err, err)
	}
}

func TestIs(t *testing.T) {
	baseErr := erro.New("test error", erro.ID("test_id"))
	wrappedErr := erro.Wrap(baseErr, "wrapped")

	if !erro.Is(wrappedErr, baseErr) {
		t.Errorf("Expected Is to be true for the same error ID")
	}

	otherErr := erro.New("other error", erro.ID("other_id"))
	if erro.Is(wrappedErr, otherErr) {
		t.Errorf("Expected Is to be false for different error IDs")
	}

	stdErr := errors.New("std error")
	wrappedStdErr := erro.Wrap(stdErr, "wrapped std")
	if !erro.Is(wrappedStdErr, stdErr) {
		t.Errorf("Expected Is to be true for wrapped standard error")
	}

	template := &templateError{
		class:     erro.ClassValidation,
		category:  erro.CategoryDatabase,
		severity:  erro.SeverityHigh,
		retryable: true,
	}
	errWithMeta := erro.New("some error", erro.ClassValidation, erro.CategoryDatabase, erro.SeverityHigh, erro.Retryable())
	if !erro.Is(errWithMeta, template) {
		t.Error("expected Is to be true for template error")
	}
}

func TestAs(t *testing.T) {
	baseErr := erro.New("test error")
	var target erro.Error
	if !erro.As(baseErr, &target) {
		t.Errorf("Expected As to be true")
	}
	if target != baseErr {
		t.Errorf("Expected target to be the base error")
	}

	// Test that As can extract a wrapped standard error
	stdErr := &customError{msg: "custom"}
	wrappedErr := erro.Wrap(stdErr, "wrapped")
	var customTarget *customError
	if !erro.As(wrappedErr, &customTarget) {
		t.Errorf("Expected As to be true for wrapped custom error")
	}
	if customTarget.msg != "custom" {
		t.Errorf("Expected custom error message to be 'custom', got '%s'", customTarget.msg)
	}

	var notAPointer customError
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	erro.As(baseErr, notAPointer)

	var asBaseErr *customError
	wrappedBaseErr := erro.Wrap(erro.New("base"), "wrapped")
	if erro.As(wrappedBaseErr, &asBaseErr) {
		t.Error("expected As to be false for wrapped base error")
	}

	var asBaseErrTarget *erro.Error
	if erro.As(wrappedBaseErr, &asBaseErrTarget) {
		t.Error("expected As to be false for wrapped base error")
	}

	var asBaseErrTarget2 **erro.Error
	if erro.As(wrappedBaseErr, &asBaseErrTarget2) {
		t.Error("expected As to be false for wrapped base error")
	}

	var asBaseErrTarget3 **customError
	if erro.As(wrappedBaseErr, &asBaseErrTarget3) {
		t.Error("expected As to be false for wrapped base error")
	}
}

type templateError struct {
	class     erro.ErrorClass
	category  erro.ErrorCategory
	severity  erro.ErrorSeverity
	retryable bool
}

func (e *templateError) Error() string                { return "" }
func (e *templateError) Class() erro.ErrorClass       { return e.class }
func (e *templateError) Category() erro.ErrorCategory { return e.category }
func (e *templateError) Severity() erro.ErrorSeverity { return e.severity }
func (e *templateError) IsRetryable() bool            { return e.retryable }
func (e *templateError) ID() string                   { return "" }
func (e *templateError) Message() string              { return "" }
func (e *templateError) Fields() []any                { return nil }
func (e *templateError) AllFields() []any             { return nil }
func (e *templateError) Created() time.Time           { return time.Time{} }
func (e *templateError) Span() erro.TraceSpan         { return nil }
func (e *templateError) Stack() erro.Stack            { return nil }
func (e *templateError) LogFields(...erro.LogOptions) []any {
	return nil
}
func (e *templateError) LogFieldsMap(...erro.LogOptions) map[string]any {
	return nil
}
func (e *templateError) BaseError() erro.Error                    { return e }
func (e *templateError) StackTraceConfig() *erro.StackTraceConfig { return nil }
func (e *templateError) Formatter() erro.FormatErrorFunc          { return nil }
func (e *templateError) Unwrap() error                            { return nil }
func (e *templateError) Is(target error) bool                     { return false }
func (e *templateError) As(target any) bool                       { return false }
func (e *templateError) Format(s fmt.State, verb rune)            {}
func (e *templateError) MarshalJSON() ([]byte, error)             { return nil, nil }
func (e *templateError) UnmarshalJSON(data []byte) error          { return nil }

func TestAsBaseError(t *testing.T) {
	err := erro.New("test")
	var target any
	if !erro.As(err, &target) {
		t.Error("expected As to be true")
	}
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestGetters(t *testing.T) {
	err := erro.New("test message",
		erro.ID("test_id"),
		erro.ClassValidation,
		erro.CategoryDatabase,
		erro.SeverityHigh,
		erro.Retryable(),
		"key", "value",
	)

	if err.ID() != "test_id" {
		t.Errorf("Expected ID 'test_id', got '%s'", err.ID())
	}
	if err.Class() != erro.ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", err.Category())
	}
	if err.Severity() != erro.SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", err.Severity())
	}
	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	if err.Message() != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message())
	}
	fields := err.Fields()
	if len(fields) != 2 || fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
}

func TestWrappedGetters(t *testing.T) {
	baseErr := erro.New("base message",
		erro.ID("base_id"),
		erro.ClassValidation,
		erro.CategoryDatabase,
		erro.SeverityHigh,
		erro.Retryable(),
		"base_key", "base_value",
	)

	wrappedErr := erro.Wrap(baseErr, "wrapped message", "wrapped_key", "wrapped_value")

	if wrappedErr.ID() != "base_id" {
		t.Errorf("Expected ID 'base_id', got '%s'", wrappedErr.ID())
	}
	if wrappedErr.Class() != erro.ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", wrappedErr.Class())
	}
	if wrappedErr.Category() != erro.CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", wrappedErr.Category())
	}
	if wrappedErr.Severity() != erro.SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", wrappedErr.Severity())
	}
	if !wrappedErr.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	if wrappedErr.Message() != "wrapped message" {
		t.Errorf("Expected message 'wrapped message', got '%s'", wrappedErr.Message())
	}

	allFields := wrappedErr.AllFields()
	if len(allFields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(allFields))
	}
}

func TestErrorFormatting(t *testing.T) {
	baseErr := errors.New("root cause")
	err := erro.Wrap(baseErr, "layer 1", "key1", "val1")
	err = erro.Wrap(err, "layer 2", "key2", "val2")

	expected := "layer 2 key2=val2: layer 1 key1=val1: root cause"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestPlusVVerb(t *testing.T) {
	err := erro.New("test error", erro.StackTrace())
	formatted := fmt.Sprintf("%+v", err)

	if !strings.Contains(formatted, "test error") {
		t.Errorf("Expected formatted string to contain 'test error', got '%s'", formatted)
	}
	if !strings.Contains(formatted, "Stack trace:") {
		t.Errorf("Expected formatted string to contain 'Stack trace:', got '%s'", formatted)
	}
	if !strings.Contains(formatted, "erro_test.TestPlusVVerb") {
		t.Errorf("Expected formatted string to contain the test function name, got '%s'", formatted)
	}
}

type mockCloser struct {
	err error
}

func (m *mockCloser) Close() error {
	return m.err
}

func TestClose(t *testing.T) {
	var err error
	erro.Close(&err, &mockCloser{err: errors.New("close error")}, "failed to close")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to close") {
		t.Errorf("Expected error to contain 'failed to close'")
	}
	if !strings.Contains(err.Error(), "close error") {
		t.Errorf("Expected error to contain 'close error'")
	}

	var err2 error
	erro.Close(&err2, &mockCloser{err: nil}, "failed to close")
	if err2 != nil {
		t.Errorf("Expected error to be nil, got '%s'", err2.Error())
	}
	erro.Close(&err2, nil, "failed to close")
	if err2 != nil {
		t.Errorf("Expected error to be nil, got '%s'", err2.Error())
	}
}

func TestShutdown(t *testing.T) {
	var err error
	shutdownFunc := func(ctx context.Context) error {
		return errors.New("shutdown error")
	}
	erro.Shutdown(context.Background(), &err, shutdownFunc, "failed to shutdown")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to shutdown") {
		t.Errorf("Expected error to contain 'failed to shutdown'")
	}
	if !strings.Contains(err.Error(), "shutdown error") {
		t.Errorf("Expected error to contain 'shutdown error'")
	}

	var err3 error
	erro.Shutdown(context.Background(), &err3, nil, "failed to close")
	if err3 != nil {
		t.Errorf("Expected error to be nil, got '%s'", err3.Error())
	}

	var err4 error
	erro.Shutdown(context.Background(), &err4, func(ctx context.Context) error {
		return nil
	}, "failed to close")
	if err4 != nil {
		t.Errorf("Expected error to be nil, got '%s'", err4.Error())
	}
}

func TestJoin(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	err := erro.Join(err1, err2)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "err1") {
		t.Errorf("Expected error to contain 'err1'")
	}
	if !strings.Contains(err.Error(), "err2") {
		t.Errorf("Expected error to contain 'err2'")
	}
	err = erro.Join(nil)
	if err != nil {
		t.Errorf("Expected error to be nil, got '%s'", err.Error())
	}
}

func TestLogFields(t *testing.T) {
	err := erro.New("test error", "key", "value")
	fields := err.LogFields()
	if len(fields) != 2 {
		t.Errorf("unexpected number of fields: %d", len(fields))
	}
	fieldsMap := err.LogFieldsMap()
	if len(fieldsMap) != 1 {
		t.Errorf("unexpected number of fields in map: %d", len(fieldsMap))
	}
}
