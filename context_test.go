package erro

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type mockTraceSpan struct {
	traceID      string
	spanID       string
	parentSpanID string
}

func (m *mockTraceSpan) RecordError(err Error) {}

func (m *mockTraceSpan) SetAttributes(attributes ...any) {}

func (m *mockTraceSpan) TraceID() string {
	return m.traceID
}

func (m *mockTraceSpan) SpanID() string {
	return m.spanID
}

func (m *mockTraceSpan) ParentSpanID() string {
	return m.parentSpanID
}

func newMockSpan(traceID, spanID, parentSpanID string) TraceSpan {
	return &mockTraceSpan{
		traceID:      traceID,
		spanID:       spanID,
		parentSpanID: parentSpanID,
	}
}

func TestMergeLogOpts(t *testing.T) {
	opts1 := []LogOption{WithID(true)}
	opts2 := []LogOption{WithCategory(true)}
	merged := MergeLogOpts(opts1, opts2...)
	if len(merged) != 2 {
		t.Errorf("expected 2 merged options, got %d", len(merged))
	}
}

func TestWithStackFormat(t *testing.T) {
	opts := &LogOptions{}
	WithStackFormat(StackFormatFull)(opts)
	if opts.StackFormat != StackFormatFull {
		t.Errorf("expected StackFormatFull, got %v", opts.StackFormat)
	}
	if !opts.IncludeStack {
		t.Error("expected IncludeStack to be true")
	}
}

func TestGetStackTrace(t *testing.T) {
	stack := Stack{
		{
			Name: "main.main",
			File: "main.go",
			Line: 10,
		},
	}
	opts := LogOptions{StackFormat: StackFormatString}
	result := getStackTrace(stack, opts)
	if _, ok := result.(string); !ok {
		t.Errorf("expected string result, got %T", result)
	}

	opts.StackFormat = StackFormatJSON
	result = getStackTrace(stack, opts)
	if _, ok := result.([]map[string]any); !ok {
		t.Errorf("expected []map[string]any result, got %T", result)
	}

	opts.StackFormat = StackFormatFull
	result = getStackTrace(stack, opts)
	if _, ok := result.(string); !ok {
		t.Errorf("expected string result, got %T", result)
	}

	opts.StackFormat = StackFormatList
	result = getStackTrace(stack, opts)
	if _, ok := result.([]string); !ok {
		t.Errorf("expected []string result, got %T", result)
	}
}

func TestGetLogFields(t *testing.T) {
	err := New("test error", "key", "value")
	opts := LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeRetryable:   true,
		IncludeTracing:     true,
		IncludeCreatedTime: true,
		IncludeFunction:    true,
		IncludePackage:     true,
		IncludeFile:        true,
		IncludeLine:        true,
		IncludeStack:       true,
		FieldNamePrefix:    "err_",
	}
	fields := getLogFields(err, opts)
	if len(fields) == 0 {
		t.Error("expected some fields, got none")
	}
}

func TestGetLogFieldsMap(t *testing.T) {
	err := New("test error", "key", "value")
	opts := LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeRetryable:   true,
		IncludeTracing:     true,
		IncludeCreatedTime: true,
		IncludeFunction:    true,
		IncludePackage:     true,
		IncludeFile:        true,
		IncludeLine:        true,
		IncludeStack:       true,
		FieldNamePrefix:    "err_",
	}
	fieldsMap := getLogFieldsMap(err, opts)
	if len(fieldsMap) == 0 {
		t.Error("expected some fields, got none")
	}
}

func TestBaseError(t *testing.T) {
	err1 := New("error1")
	err2 := Wrap(err1, "error2")
	if err2.BaseError() != err1 {
		t.Error("BaseError should return the root error")
	}
}

func TestLogError(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	err := New("test error", "key", "value")
	LogError(err, logFunc, WithUserFields(true))

	if loggedMessage != "test error" {
		t.Errorf("expected message 'test error', got '%s'", loggedMessage)
	}
	if len(loggedFields) == 0 {
		t.Error("expected some fields, got none")
	}
}

func TestLogError_NilError(t *testing.T) {
	var called bool
	logFunc := func(message string, fields ...any) {
		called = true
	}
	LogError(nil, logFunc)
	if called {
		t.Error("logFunc should not be called for nil error")
	}
}

func TestLogError_NilLogFunc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogError panicked with nil logFunc: %v", r)
		}
	}()
	err := New("test error")
	LogError(err, nil)
}

func TestLogError_StandardError(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	err := errors.New("standard error")
	LogError(err, logFunc)

	if loggedMessage != "standard error" {
		t.Errorf("expected message 'standard error', got '%s'", loggedMessage)
	}
	if len(loggedFields) != 0 {
		t.Errorf("expected nil fields, got %v", loggedFields)
	}
}

func TestLogFields_NilError(t *testing.T) {
	fields := LogFields(nil)
	if fields != nil {
		t.Errorf("expected nil fields, got %v", fields)
	}
}

func TestLogFieldsMap_NilError(t *testing.T) {
	fieldsMap := LogFieldsMap(nil)
	if fieldsMap != nil {
		t.Errorf("expected nil fields, got %v", fieldsMap)
	}
}

func TestWithUserFields(t *testing.T) {
	opts := &LogOptions{}
	WithUserFields(false)(opts)
	if opts.IncludeUserFields {
		t.Error("expected IncludeUserFields to be false")
	}
	WithUserFields(true)(opts)
	if !opts.IncludeUserFields {
		t.Error("expected IncludeUserFields to be true")
	}
	WithUserFields()(opts)
	if !opts.IncludeUserFields {
		t.Error("expected IncludeUserFields to be true")
	}
}

func TestWithID(t *testing.T) {
	opts := &LogOptions{}
	WithID(false)(opts)
	if opts.IncludeID {
		t.Error("expected IncludeID to be false")
	}
	WithID(true)(opts)
	if !opts.IncludeID {
		t.Error("expected IncludeID to be true")
	}
	WithID()(opts)
	if !opts.IncludeID {
		t.Error("expected IncludeID to be true")
	}
}

func TestWithCategory(t *testing.T) {
	opts := &LogOptions{}
	WithCategory(false)(opts)
	if opts.IncludeCategory {
		t.Error("expected IncludeCategory to be false")
	}
	WithCategory(true)(opts)
	if !opts.IncludeCategory {
		t.Error("expected IncludeCategory to be true")
	}
	WithCategory()(opts)
	if !opts.IncludeCategory {
		t.Error("expected IncludeCategory to be true")
	}
}

func TestWithSeverity(t *testing.T) {
	opts := &LogOptions{}
	WithSeverity(false)(opts)
	if opts.IncludeSeverity {
		t.Error("expected IncludeSeverity to be false")
	}
	WithSeverity(true)(opts)
	if !opts.IncludeSeverity {
		t.Error("expected IncludeSeverity to be true")
	}
	WithSeverity()(opts)
	if !opts.IncludeSeverity {
		t.Error("expected IncludeSeverity to be true")
	}
}

func TestWithTracing(t *testing.T) {
	opts := &LogOptions{}
	WithTracing(false)(opts)
	if opts.IncludeTracing {
		t.Error("expected IncludeTracing to be false")
	}
	WithTracing(true)(opts)
	if !opts.IncludeTracing {
		t.Error("expected IncludeTracing to be true")
	}
	WithTracing()(opts)
	if !opts.IncludeTracing {
		t.Error("expected IncludeTracing to be true")
	}
}

func TestWithRetryable(t *testing.T) {
	opts := &LogOptions{}
	WithRetryable(false)(opts)
	if opts.IncludeRetryable {
		t.Error("expected IncludeRetryable to be false")
	}
	WithRetryable(true)(opts)
	if !opts.IncludeRetryable {
		t.Error("expected IncludeRetryable to be true")
	}
	WithRetryable()(opts)
	if !opts.IncludeRetryable {
		t.Error("expected IncludeRetryable to be true")
	}
}

func TestWithCreatedTime(t *testing.T) {
	opts := &LogOptions{}
	WithCreatedTime(false)(opts)
	if opts.IncludeCreatedTime {
		t.Error("expected IncludeCreatedTime to be false")
	}
	WithCreatedTime(true)(opts)
	if !opts.IncludeCreatedTime {
		t.Error("expected IncludeCreatedTime to be true")
	}
	WithCreatedTime()(opts)
	if !opts.IncludeCreatedTime {
		t.Error("expected IncludeCreatedTime to be true")
	}
}

func TestWithFunction(t *testing.T) {
	opts := &LogOptions{}
	WithFunction(false)(opts)
	if opts.IncludeFunction {
		t.Error("expected IncludeFunction to be false")
	}
	WithFunction(true)(opts)
	if !opts.IncludeFunction {
		t.Error("expected IncludeFunction to be true")
	}
	WithFunction()(opts)
	if !opts.IncludeFunction {
		t.Error("expected IncludeFunction to be true")
	}
}

func TestWithPackage(t *testing.T) {
	opts := &LogOptions{}
	WithPackage(false)(opts)
	if opts.IncludePackage {
		t.Error("expected IncludePackage to be false")
	}
	WithPackage(true)(opts)
	if !opts.IncludePackage {
		t.Error("expected IncludePackage to be true")
	}
	WithPackage()(opts)
	if !opts.IncludePackage {
		t.Error("expected IncludePackage to be true")
	}
}

func TestWithFile(t *testing.T) {
	opts := &LogOptions{}
	WithFile(false)(opts)
	if opts.IncludeFile {
		t.Error("expected IncludeFile to be false")
	}
	WithFile(true)(opts)
	if !opts.IncludeFile {
		t.Error("expected IncludeFile to be true")
	}
	WithFile()(opts)
	if !opts.IncludeFile {
		t.Error("expected IncludeFile to be true")
	}
}

func TestWithLine(t *testing.T) {
	opts := &LogOptions{}
	WithLine(false)(opts)
	if opts.IncludeLine {
		t.Error("expected IncludeLine to be false")
	}
	WithLine(true)(opts)
	if !opts.IncludeLine {
		t.Error("expected IncludeLine to be true")
	}
	WithLine()(opts)
	if !opts.IncludeLine {
		t.Error("expected IncludeLine to be true")
	}
}

func TestWithStack(t *testing.T) {
	opts := &LogOptions{}
	WithStack(false)(opts)
	if opts.IncludeStack {
		t.Error("expected IncludeStack to be false")
	}
	WithStack(true)(opts)
	if !opts.IncludeStack {
		t.Error("expected IncludeStack to be true")
	}
	WithStack()(opts)
	if !opts.IncludeStack {
		t.Error("expected IncludeStack to be true")
	}
}

func TestWithFieldNamePrefix(t *testing.T) {
	opts := &LogOptions{}
	WithFieldNamePrefix("test_")(opts)
	if opts.FieldNamePrefix != "test_" {
		t.Errorf("expected FieldNamePrefix 'test_', got '%s'", opts.FieldNamePrefix)
	}
}

func TestApplyOptions(t *testing.T) {
	opts := &LogOptions{}
	opts.ApplyOptions(WithID(true), WithCategory(true))
	if !opts.IncludeID || !opts.IncludeCategory {
		t.Error("expected IncludeID and IncludeCategory to be true")
	}
}

func TestDefaultLogOptions(t *testing.T) {
	if !DefaultLogOptions.IncludeUserFields {
		t.Error("expected IncludeUserFields to be true")
	}
	if DefaultLogOptions.IncludeID {
		t.Error("expected IncludeID to be false")
	}
}

func TestVerboseLogOpts(t *testing.T) {
	opts := &LogOptions{}
	for _, opt := range VerboseLogOpts {
		opt(opts)
	}
	if !opts.IncludeUserFields || !opts.IncludeID || !opts.IncludeCategory || !opts.IncludeSeverity || !opts.IncludeTracing || !opts.IncludeRetryable || !opts.IncludeCreatedTime || !opts.IncludeFunction || !opts.IncludePackage || !opts.IncludeFile || !opts.IncludeLine || !opts.IncludeStack {
		t.Error("mismatch in verbose log options")
	}
}

func TestMinimalLogOpts(t *testing.T) {
	opts := &LogOptions{}
	for _, opt := range MinimalLogOpts {
		opt(opts)
	}
	if !opts.IncludeUserFields || opts.IncludeID || !opts.IncludeSeverity || opts.IncludeCategory || opts.IncludeTracing || opts.IncludeRetryable || opts.IncludeCreatedTime || opts.IncludeFunction || opts.IncludePackage || opts.IncludeFile || opts.IncludeLine || opts.IncludeStack {
		t.Error("mismatch in minimal log options")
	}
}

func TestExtractError_Nil(t *testing.T) {
	if err := ExtractError(nil); err != nil {
		t.Error("expected nil for nil error")
	}
}

func TestExtractError_ErroError(t *testing.T) {
	err := New("test")
	if extracted := ExtractError(err); extracted != err {
		t.Error("should return the same erro error")
	}
}

func TestExtractError_StandardError(t *testing.T) {
	err := errors.New("std err")
	extracted := ExtractError(err)
	if extracted.Unwrap() != err {
		t.Error("should wrap a standard error")
	}
}

func TestErrorToJSON(t *testing.T) {
	err := New("test", "key", "value", RecordSpan(newMockSpan("trace1", "span1", "parent1")))
	schema := ErrorToJSON(err)
	if schema.Message != "test" {
		t.Error("incorrect message")
	}
	if len(schema.Fields) != 2 {
		t.Error("incorrect fields count")
	}
	if schema.TraceID != "trace1" {
		t.Error("incorrect trace id")
	}
}

func TestErrorToJSON_Redacted(t *testing.T) {
	err := New("test", "key", Redact("secret"))
	schema := ErrorToJSON(err)
	if schema.Fields[1] != RedactedPlaceholder {
		t.Errorf("expected redacted placeholder, got %v", schema.Fields[1])
	}
}

func TestGetLogFieldsMap_OddFields(t *testing.T) {
	fields := []any{"key1", "val1", "key2"}
	fieldsMap := getLogFieldsMap(&baseError{fields: fields})
	if len(fieldsMap) != 1 {
		t.Errorf("expected 1 field, got %d", len(fieldsMap))
	}
	if _, ok := fieldsMap["key1"]; !ok {
		t.Error("key1 not found in map")
	}
}

func TestGetLogFields_Redacted(t *testing.T) {
	err := New("test", "key", Redact("secret"))
	opts := LogOptions{IncludeUserFields: true}
	fields := getLogFields(err, opts)
	found := false
	for i := 0; i < len(fields); i++ {
		if reflect.DeepEqual(fields[i], RedactedPlaceholder) {
			found = true
			break
		}
	}
	if !found {
		t.Error("redacted value not found in log fields")
	}
}

func TestGetLogFields_NoTopFrame(t *testing.T) {
	err := &baseError{stack: nil}
	opts := LogOptions{IncludeFunction: true}
	fields := getLogFields(err, opts)
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "err_function" {
			t.Error("function field should not be present")
		}
	}
}

func TestGetStackTrace_NilStack(t *testing.T) {
	opts := LogOptions{}
	if res := getStackTrace(nil, opts); res != nil {
		t.Error("expected nil for nil stack")
	}
}

func TestGetLogFields_NoSpan(t *testing.T) {
	err := New("test")
	opts := LogOptions{IncludeTracing: true}
	fields := getLogFields(err, opts)
	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		if key == "trace_id" || key == "span_id" || key == "parent_span_id" {
			t.Errorf("tracing field %s should not be present", key)
		}
	}
}

func TestGetLogFields_WithSpan(t *testing.T) {
	err := New("test", RecordSpan(newMockSpan("t1", "s1", "p1")))
	opts := LogOptions{IncludeTracing: true}
	fields := getLogFields(err, opts)
	found := 0
	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		if key == "trace_id" || key == "span_id" || key == "parent_span_id" {
			found++
		}
	}
	if found != 3 {
		t.Error("tracing fields are missing")
	}
}

func TestGetLogFields_NoRetryable(t *testing.T) {
	err := New("test")
	opts := DefaultLogOptions
	opts.IncludeRetryable = true
	fields := getLogFields(err, opts)
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_retryable" {
			t.Error("retryable field should not be present if false")
		}
	}
}

func TestGetLogFields_WithRetryable(t *testing.T) {
	err := New("test", Retryable())
	opts := DefaultLogOptions
	opts.IncludeRetryable = true
	fields := getLogFields(err, opts)
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_retryable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("retryable field is missing")
	}
}

func TestGetLogFields_NoCreatedTime(t *testing.T) {
	err := New("test")
	opts := LogOptions{IncludeCreatedTime: false}
	fields := getLogFields(err, opts)
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_created" {
			t.Error("created field should not be present")
		}
	}
}

func TestGetLogFields_WithCreatedTime(t *testing.T) {
	err := New("test")
	opts := DefaultLogOptions
	opts.IncludeCreatedTime = true
	fields := getLogFields(err, opts)
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_created" {
			found = true
			break
		}
	}
	if !found {
		t.Error("created field is missing")
	}
}

func TestGetLogFields_NoStack(t *testing.T) {
	err := New("test")
	opts := LogOptions{IncludeStack: false}
	fields := getLogFields(err, opts)
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_stack" {
			t.Error("stack field should not be present")
		}
	}
}

func TestGetLogFields_WithStack(t *testing.T) {
	err := New("test", StackTrace())
	opts := DefaultLogOptions
	opts.IncludeStack = true
	fields := getLogFields(err, opts)
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "error_stack" {
			found = true
			break
		}
	}
	if !found {
		t.Error("stack field is missing")
	}
}

func TestGetLogFields_NilContext(t *testing.T) {
	fields := getLogFields(nil)
	if fields != nil {
		t.Error("expected nil fields for nil context")
	}
}

func TestGetLogFieldsMap_NonStringKey(t *testing.T) {
	fields := []any{123, "value"}
	fieldsMap := getLogFieldsMap(&baseError{fields: fields})
	if _, ok := fieldsMap["123"]; !ok {
		t.Error("key '123' not found in map")
	}
}

func TestLogFieldsWithOptions(t *testing.T) {
	err := New("test error", "key", "value", ErrorCategory("test_cat"), ErrorSeverity("test_sev"), Retryable(), StackTrace())
	span := newMockSpan("trace1", "span1", "parent1")
	err = Wrap(err, "wrapped", RecordSpan(span))

	opts := []LogOption{
		WithUserFields(true),
		WithID(true),
		WithCategory(true),
		WithSeverity(true),
		WithRetryable(true),
		WithTracing(true),
		WithCreatedTime(true),
		WithFunction(true),
		WithPackage(true),
		WithFile(true),
		WithLine(true),
		WithStack(true),
		WithStackFormat(StackFormatList),
		WithFieldNamePrefix("err_"),
	}

	fields := LogFields(err, opts...)
	if len(fields) == 0 {
		t.Fatal("expected fields, got none")
	}

	fieldsMap := LogFieldsMap(err, opts...)
	if len(fieldsMap) == 0 {
		t.Fatal("expected fields map, got none")
	}

	// Check for a few specific fields
	foundCategory := false
	foundStack := false
	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		if key == "err_category" && fields[i+1] == ErrorCategory("test_cat") {
			foundCategory = true
		}
		if key == "err_stack" {
			foundStack = true
		}
	}
	if !foundCategory {
		t.Error("category field not found or incorrect")
	}
	if !foundStack {
		t.Error("stack field not found")
	}
}

func TestLogFieldsWithOptionsStruct(t *testing.T) {
	err := New("test error", "key", "value", ErrorCategory("test_cat"), ErrorSeverity("test_sev"), Retryable(), StackTrace())
	span := newMockSpan("trace1", "span1", "parent1")
	err = Wrap(err, "wrapped", RecordSpan(span))

	opts := LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeRetryable:   true,
		IncludeTracing:     true,
		IncludeCreatedTime: true,
		IncludeFunction:    true,
		IncludePackage:     true,
		IncludeFile:        true,
		IncludeLine:        true,
		IncludeStack:       true,
		StackFormat:        StackFormatList,
		FieldNamePrefix:    "err_",
	}

	fields := LogFieldsWithOptions(err, opts)
	if len(fields) == 0 {
		t.Fatal("expected fields, got none")
	}

	// Check for specific fields
	foundCategory := false
	foundStack := false
	foundUserField := false
	foundTraceID := false
	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		if key == "err_category" && fields[i+1] == ErrorCategory("test_cat") {
			foundCategory = true
		}
		if key == "err_stack" {
			foundStack = true
		}
		if key == "key" && fields[i+1] == "value" {
			foundUserField = true
		}
		if key == "trace_id" && fields[i+1] == "trace1" {
			foundTraceID = true
		}
	}
	if !foundCategory {
		t.Error("category field not found or incorrect")
	}
	if !foundStack {
		t.Error("stack field not found")
	}
	if !foundUserField {
		t.Error("user field not found")
	}
	if !foundTraceID {
		t.Error("trace_id field not found")
	}
}

func TestLogFieldsWithOptionsStruct_NilError(t *testing.T) {
	opts := LogOptions{IncludeUserFields: true}
	fields := LogFieldsWithOptions(nil, opts)
	if fields != nil {
		t.Errorf("expected nil fields for nil error, got %v", fields)
	}
}

func TestLogFieldsWithOptionsStruct_MinimalOptions(t *testing.T) {
	err := New("test error", "key", "value")
	opts := LogOptions{IncludeUserFields: true}
	fields := LogFieldsWithOptions(err, opts)

	if len(fields) < 2 {
		t.Error("expected at least user fields")
	}

	// Should only have user fields
	foundNonUserField := false
	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		if key != "key" {
			foundNonUserField = true
			break
		}
	}
	if foundNonUserField {
		t.Error("found non-user field when only user fields should be included")
	}
}

func TestLogFieldsWithOptionsStruct_StandardError(t *testing.T) {
	err := errors.New("standard error")
	opts := LogOptions{IncludeUserFields: true, IncludeCategory: true, IncludeSeverity: true, IncludeFunction: true}
	fields := LogFieldsWithOptions(err, opts)

	if len(fields) > 0 {
		t.Errorf("expected no fields for standard error, got %v", fields)
	}
}

func TestLogFieldsMapWithOptionsStruct(t *testing.T) {
	err := New("test error", "key", "value", ErrorCategory("test_cat"), ErrorSeverity("test_sev"), Retryable(), StackTrace())
	span := newMockSpan("trace1", "span1", "parent1")
	err = Wrap(err, "wrapped", RecordSpan(span))

	opts := LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeRetryable:   true,
		IncludeTracing:     true,
		IncludeCreatedTime: true,
		IncludeFunction:    true,
		IncludePackage:     true,
		IncludeFile:        true,
		IncludeLine:        true,
		IncludeStack:       true,
		StackFormat:        StackFormatJSON,
		FieldNamePrefix:    "err_",
	}

	fieldsMap := LogFieldsMapWithOptions(err, opts)
	if len(fieldsMap) == 0 {
		t.Fatal("expected fields map, got none")
	}

	// Check for specific fields
	if category, ok := fieldsMap["err_category"]; !ok || category != ErrorCategory("test_cat") {
		t.Error("category field not found or incorrect in map")
	}
	if _, ok := fieldsMap["err_stack"]; !ok {
		t.Error("stack field not found in map")
	}
	if userVal, ok := fieldsMap["key"]; !ok || userVal != "value" {
		t.Error("user field not found or incorrect in map")
	}
	if traceID, ok := fieldsMap["trace_id"]; !ok || traceID != "trace1" {
		t.Error("trace_id field not found or incorrect in map")
	}
}

func TestLogFieldsMapWithOptionsStruct_NilError(t *testing.T) {
	opts := LogOptions{IncludeUserFields: true}
	fieldsMap := LogFieldsMapWithOptions(nil, opts)
	if fieldsMap != nil {
		t.Errorf("expected nil fields map for nil error, got %v", fieldsMap)
	}
}

func TestLogFieldsMapWithOptionsStruct_EmptyOptions(t *testing.T) {
	err := New("test error", "key", "value")
	opts := LogOptions{} // All fields disabled
	fieldsMap := LogFieldsMapWithOptions(err, opts)

	if len(fieldsMap) > 0 {
		t.Errorf("expected empty fields map with disabled options, got %v", fieldsMap)
	}
}

func TestLogFieldsMapWithOptionsStruct_StandardError(t *testing.T) {
	err := errors.New("standard error")
	opts := LogOptions{IncludeUserFields: true, IncludeCategory: true, IncludeSeverity: true, IncludeFunction: true}
	fieldsMap := LogFieldsMapWithOptions(err, opts)

	if len(fieldsMap) > 0 {
		t.Errorf("expected no fields for standard error, got %v", fieldsMap)
	}
}

func TestLogErrorWithOptionsStruct(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	err := New("test error", "key", "value", ErrorCategory("test_cat"), ErrorSeverity("critical"))
	opts := LogOptions{
		IncludeUserFields: true,
		IncludeCategory:   true,
		IncludeSeverity:   true,
		FieldNamePrefix:   "err_",
	}

	LogErrorWithOptions(err, logFunc, opts)

	if loggedMessage != "test error" {
		t.Errorf("expected message 'test error', got '%s'", loggedMessage)
	}
	if len(loggedFields) == 0 {
		t.Error("expected some fields, got none")
	}

	// Check for specific fields in logged output
	foundCategory := false
	foundSeverity := false
	foundUserField := false
	for i := 0; i < len(loggedFields); i += 2 {
		key := loggedFields[i].(string)
		if key == "err_category" && loggedFields[i+1] == ErrorCategory("test_cat") {
			foundCategory = true
		}
		if key == "err_severity" && loggedFields[i+1] == ErrorSeverity("critical") {
			foundSeverity = true
		}
		if key == "key" && loggedFields[i+1] == "value" {
			foundUserField = true
		}
	}
	if !foundCategory {
		t.Error("category field not found in logged output")
	}
	if !foundSeverity {
		t.Error("severity field not found in logged output")
	}
	if !foundUserField {
		t.Error("user field not found in logged output")
	}
}

func TestLogErrorWithOptionsStruct_NilError(t *testing.T) {
	var called bool
	logFunc := func(message string, fields ...any) {
		called = true
	}
	opts := LogOptions{IncludeUserFields: true}
	LogErrorWithOptions(nil, logFunc, opts)
	if called {
		t.Error("logFunc should not be called for nil error")
	}
}

func TestLogErrorWithOptionsStruct_NilLogFunc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogErrorWithOptions panicked with nil logFunc: %v", r)
		}
	}()
	err := New("test error")
	opts := LogOptions{IncludeUserFields: true}
	LogErrorWithOptions(err, nil, opts)
}

func TestLogErrorWithOptionsStruct_StandardError(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	err := errors.New("standard error")
	opts := LogOptions{IncludeUserFields: true, IncludeCategory: true}
	LogErrorWithOptions(err, logFunc, opts)

	if loggedMessage != "standard error" {
		t.Errorf("expected message 'standard error', got '%s'", loggedMessage)
	}
	if len(loggedFields) != 0 {
		t.Errorf("expected nil fields for standard error, got %v", loggedFields)
	}
}

func TestLogErrorWithOptionsStruct_MinimalOptions(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	err := New("test error", "key", "value")
	opts := LogOptions{} // All fields disabled
	LogErrorWithOptions(err, logFunc, opts)

	if loggedMessage != "test error" {
		t.Errorf("expected message 'test error', got '%s'", loggedMessage)
	}
	if len(loggedFields) > 0 {
		t.Errorf("expected no fields with disabled options, got %v", loggedFields)
	}
}

func TestLogErrorWithOptionsStruct_WithTracing(t *testing.T) {
	var loggedMessage string
	var loggedFields []any
	logFunc := func(message string, fields ...any) {
		loggedMessage = message
		loggedFields = fields
	}

	span := newMockSpan("trace1", "span1", "parent1")
	err := New("test error", RecordSpan(span))
	opts := LogOptions{IncludeTracing: true}
	LogErrorWithOptions(err, logFunc, opts)

	if loggedMessage != "test error" {
		t.Errorf("expected message 'test error', got '%s'", loggedMessage)
	}

	// Check for tracing fields
	foundTraceID := false
	foundSpanID := false
	foundParentSpanID := false
	for i := 0; i < len(loggedFields); i += 2 {
		key := loggedFields[i].(string)
		if key == "trace_id" && loggedFields[i+1] == "trace1" {
			foundTraceID = true
		}
		if key == "span_id" && loggedFields[i+1] == "span1" {
			foundSpanID = true
		}
		if key == "parent_span_id" && loggedFields[i+1] == "parent1" {
			foundParentSpanID = true
		}
	}
	if !foundTraceID {
		t.Error("trace_id field not found in logged output")
	}
	if !foundSpanID {
		t.Error("span_id field not found in logged output")
	}
	if !foundParentSpanID {
		t.Error("parent_span_id field not found in logged output")
	}
}

func TestMarshalJSON(t *testing.T) {
	err := New("test message",
		ID("test_id"),
		ClassValidation,
		CategoryDatabase,
		SeverityHigh,
		Retryable(),
		"key", "value",
		"sensitive", Redact("secret"),
	)

	b, jErr := json.Marshal(err)
	if jErr != nil {
		t.Fatalf("json.Marshal failed: %v", jErr)
	}

	var data map[string]interface{}
	if jErr = json.Unmarshal(b, &data); jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if data["id"] != "test_id" {
		t.Errorf("Expected id 'test_id', got '%s'", data["id"])
	}
	if data["class"] != "validation" {
		t.Errorf("Expected class 'validation', got '%s'", data["class"])
	}
	if data["category"] != "database" {
		t.Errorf("Expected category 'database', got '%s'", data["category"])
	}
	if data["severity"] != "high" {
		t.Errorf("Expected severity 'high', got '%s'", data["severity"])
	}
	if data["message"] != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", data["message"])
	}
	if data["retryable"] != true {
		t.Errorf("Expected retryable to be true, got %v", data["retryable"])
	}
	if _, ok := data["created"]; !ok {
		t.Errorf("Expected 'created' field to exist")
	}

	fields, ok := data["fields"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'fields' to be a slice, got %T", data["fields"])
	}
	if len(fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(fields))
	}
	if fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
	if fields[2] != "sensitive" || fields[3] != RedactedPlaceholder {
		t.Errorf("Expected redacted field, got '%s' and '%s'", fields[2], fields[3])
	}
}

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test_id",
		"class": "validation",
		"category": "database",
		"severity": "high",
		"message": "test message",
		"retryable": true,
		"fields": ["key", "value"]
	}`

	err := New("")
	jErr := json.Unmarshal([]byte(jsonData), err)
	if jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if err.ID() != "test_id" {
		t.Errorf("Expected ID 'test_id', got '%s'", err.ID())
	}
	if err.Class() != ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", err.Class())
	}
	if err.Category() != CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", err.Category())
	}
	if err.Severity() != SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", err.Severity())
	}
	if err.Message() != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message())
	}
	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	fields := err.Fields()
	if len(fields) != 2 || fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
}

func TestMarshalJSONWithStack(t *testing.T) {
	err := New("test error", StackTrace())

	b, jErr := json.Marshal(err)
	if jErr != nil {
		t.Fatalf("json.Marshal failed: %v", jErr)
	}

	var data map[string]interface{}
	if jErr = json.Unmarshal(b, &data); jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if _, ok := data["stack_trace"]; !ok {
		t.Errorf("Expected 'stack_trace' field to exist")
	}
}
