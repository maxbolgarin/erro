package erro

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGetFormatErrorWithFullContextBase(t *testing.T) {
	formatter := GetFormatErrorWithFullContextBase()
	err := New("test")
	if formatter(err) == "" {
		t.Error("expected a formatted string")
	}
}

func TestGetFormatErrorWithFullContext(t *testing.T) {
	formatter := GetFormatErrorWithFullContext()
	err := New("test")
	if formatter(err) == "" {
		t.Error("expected a formatted string")
	}
}

func TestFormatErrorMessage(t *testing.T) {
	err := New("test")
	if FormatErrorMessage(err) != "test" {
		t.Errorf("unexpected message: %s", FormatErrorMessage(err))
	}
}

func TestIsASCII(t *testing.T) {
	if !isASCII("ascii") {
		t.Error("expected true for ascii string")
	}
	if isASCII("你好") {
		t.Error("expected false for non-ascii string")
	}
}

func TestTruncateUTF8(t *testing.T) {
	if truncateUTF8("12345", 3) != "123" {
		t.Error("unexpected truncation")
	}
	if truncateUTF8("你好世界", 7) != "你好" {
		t.Error("unexpected utf8 truncation")
	}
	if truncateUTF8("abc", 5) != "abc" {
		t.Error("unexpected truncation")
	}
	if truncateUTF8("abc", 0) != "" {
		t.Error("unexpected truncation")
	}
}

func TestTruncateString(t *testing.T) {
	if truncateString("12345", 3) != "123" {
		t.Error("unexpected truncation")
	}
	if truncateString("你好世界", 7) != "你好" {
		t.Error("unexpected utf8 truncation")
	}
	if truncateString("abc", 5) != "abc" {
		t.Error("unexpected truncation")
	}
}

func TestValueToString(t *testing.T) {
	if valueToString("string") != "string" {
		t.Error("unexpected string conversion")
	}
	if valueToString(123) != "123" {
		t.Error("unexpected int conversion")
	}
	if valueToString(int8(123)) != "123" {
		t.Error("unexpected int8 conversion")
	}
	if valueToString(int16(123)) != "123" {
		t.Error("unexpected int16 conversion")
	}
	if valueToString(int32(123)) != "123" {
		t.Error("unexpected int32 conversion")
	}
	if valueToString(int64(123)) != "123" {
		t.Error("unexpected int64 conversion")
	}
	if valueToString(uint(123)) != "123" {
		t.Error("unexpected uint conversion")
	}
	if valueToString(uint8(123)) != "123" {
		t.Error("unexpected uint8 conversion")
	}
	if valueToString(uint16(123)) != "123" {
		t.Error("unexpected uint16 conversion")
	}
	if valueToString(uint32(123)) != "123" {
		t.Error("unexpected uint32 conversion")
	}
	if valueToString(uint64(123)) != "123" {
		t.Error("unexpected uint64 conversion")
	}
	if valueToString(float32(123.4)) != "123.4" {
		t.Error("unexpected float32 conversion")
	}
	if valueToString(float64(123.4)) != "123.4" {
		t.Error("unexpected float64 conversion")
	}
	if valueToString(true) != "true" {
		t.Error("unexpected bool conversion")
	}
	if valueToString(nil) != "" {
		t.Error("unexpected nil conversion")
	}
	if valueToString(Redact("secret")) != RedactedPlaceholder {
		t.Error("unexpected redacted value conversion")
	}
	now := time.Now()
	if valueToString(now) != now.Format(time.RFC3339) {
		t.Error("unexpected time conversion")
	}
	if valueToString([]byte("bytes")) != "bytes" {
		t.Error("unexpected bytes conversion")
	}
	if valueToString(fmt.Errorf("error")) != "error" {
		t.Error("unexpected error conversion")
	}
	if valueToString([]string{"a", "b"}) != "a,b" {
		t.Error("unexpected string slice conversion")
	}
	if valueToString(struct{}{}) != "{}" {
		t.Error("unexpected struct conversion")
	}
	var nilErr error
	if valueToString(nilErr) != "" {
		t.Error("unexpected nil error conversion")
	}
	var nilStringer fmt.Stringer
	if valueToString(nilStringer) != "" {
		t.Error("unexpected nil stringer conversion")
	}
	stringerObj := stringer{}
	if valueToString(stringerObj) != "stringer" {
		t.Error("unexpected stringer conversion")
	}
}

type stringer struct{}

func (s stringer) String() string {
	return "stringer"
}

func TestAppendValue(t *testing.T) {
	var b strings.Builder
	appendValue(&b, "string", 10)
	if b.String() != "string" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, Redact("secret"), 10)
	if b.String() != RedactedPlaceholder {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, []byte("bytes"), 10)
	if b.String() != "bytes" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	now := time.Now()
	appendValue(&b, now, 100)
	if b.String() != now.Format(time.RFC3339) {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, fmt.Errorf("error"), 10)
	if b.String() != "error" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, 123, 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, true, 10)
	if b.String() != "true" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, []string{"a", "b"}, 10)
	if b.String() != "a,b" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, nil, 10)
	if b.String() != "" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, struct{}{}, 10)
	if b.String() != "{}" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, int8(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, int16(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, int32(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, int64(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, uint(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, uint8(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, uint16(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, uint32(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, uint64(123), 10)
	if b.String() != "123" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, float32(123.4), 10)
	if b.String() != "123.4" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	appendValue(&b, float64(123.4), 10)
	if b.String() != "123.4" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	var nilErr error
	appendValue(&b, nilErr, 10)
	if b.String() != "" {
		t.Error("unexpected appendValue result")
	}
	b.Reset()
	var nilStringer fmt.Stringer
	appendValue(&b, nilStringer, 10)
	if b.String() != "" {
		t.Error("unexpected appendValue result")
	}
}

func TestBuildMessage(t *testing.T) {
	err := New("test", CategoryDatabase, ClassInternal, SeverityCritical)
	msg := buildMessage(err)
	if !strings.Contains(msg, "test") {
		t.Errorf("message should contain base message, got %s", msg)
	}

	errNoMsg := New("", CategoryDatabase, ClassInternal, SeverityCritical)
	msg = buildMessage(errNoMsg)
	if !strings.Contains(msg, "database") {
		t.Errorf("message should contain category, got %s", msg)
	}

	errNoCat := New("", ClassInternal, SeverityCritical)
	msg = buildMessage(errNoCat)
	if !strings.Contains(msg, "internal") {
		t.Errorf("message should contain class, got %s", msg)
	}

	errNoMeta := New("")
	msg = buildMessage(errNoMeta)
	if msg != "" {
		t.Errorf("expected empty message, got %s", msg)
	}
}

func TestBuildFieldsMessage(t *testing.T) {
	msg := buildFieldsMessage("base", []any{"key", "value"})
	if !strings.Contains(msg, "key=value") {
		t.Errorf("expected fields in message, got %s", msg)
	}
	msg = buildFieldsMessage("", []any{"key", "value"})
	if !strings.Contains(msg, "error") {
		t.Errorf("expected default message, got %s", msg)
	}
}

func TestApplyFormatVerbs(t *testing.T) {
	msg, args := ApplyFormatVerbs("hello %s", "world")
	if msg != "hello world" || len(args) != 0 {
		t.Errorf("unexpected format: %s, %v", msg, args)
	}
	msg, args = ApplyFormatVerbs("hello %s %s", "world")
	if msg != "hello world %s" || len(args) != 0 {
		t.Errorf("unexpected format: %s, %v", msg, args)
	}
}

func TestGetFormatErrorWithFullContextBase_NonBaseError(t *testing.T) {
	formatter := GetFormatErrorWithFullContextBase()
	// Create a standard error that's not a baseError
	err := fmt.Errorf("standard error")
	// Create a wrapper that implements Error interface
	wrapper := &errorWrapper{err: err}
	result := formatter(wrapper)
	if result == "" {
		t.Error("expected a formatted string for non-base error")
	}
}

type errorWrapper struct {
	err error
}

func (e *errorWrapper) Error() string                                  { return e.err.Error() }
func (e *errorWrapper) Message() string                                { return e.err.Error() }
func (e *errorWrapper) Fields() []any                                  { return nil }
func (e *errorWrapper) AllFields() []any                               { return nil }
func (e *errorWrapper) ID() string                                     { return "" }
func (e *errorWrapper) Class() ErrorClass                              { return "" }
func (e *errorWrapper) Category() ErrorCategory                        { return "" }
func (e *errorWrapper) Severity() ErrorSeverity                        { return "" }
func (e *errorWrapper) IsRetryable() bool                              { return false }
func (e *errorWrapper) Stack() Stack                                   { return nil }
func (e *errorWrapper) Created() time.Time                             { return time.Time{} }
func (e *errorWrapper) Span() TraceSpan                                { return nil }
func (e *errorWrapper) LogFields(opts ...LogOptions) []any             { return nil }
func (e *errorWrapper) LogFieldsMap(opts ...LogOptions) map[string]any { return nil }
func (e *errorWrapper) BaseError() Error                               { return nil }
func (e *errorWrapper) Is(target error) bool                           { return false }
func (e *errorWrapper) As(target any) bool                             { return false }
func (e *errorWrapper) Unwrap() error                                  { return e.err }
func (e *errorWrapper) Format(s fmt.State, verb rune)                  { formatError(e, s, verb) }
func (e *errorWrapper) MarshalJSON() ([]byte, error)                   { return nil, nil }
func (e *errorWrapper) UnmarshalJSON(data []byte) error                { return nil }

func TestBuildMessage_EdgeCases(t *testing.T) {
	// Test case: error with empty message but category and class
	err := &baseError{
		message:  "",
		category: CategoryUserInput,
		class:    ClassValidation,
	}
	msg := buildMessage(err)
	if msg != "user_input validation" {
		t.Errorf("expected 'user_input validation', got '%s'", msg)
	}

	// Test case: error with only category
	err = &baseError{
		message:  "",
		category: CategoryUserInput,
		class:    "",
	}
	msg = buildMessage(err)
	if msg != "user_input" {
		t.Errorf("expected 'user_input', got '%s'", msg)
	}

	// Test case: error with only class
	err = &baseError{
		message:  "",
		category: "",
		class:    ClassValidation,
	}
	msg = buildMessage(err)
	if msg != "validation" {
		t.Errorf("expected 'validation', got '%s'", msg)
	}

	// Test case: error with only severity
	err = &baseError{
		message:  "",
		category: "",
		class:    "",
		severity: SeverityHigh,
	}
	msg = buildMessage(err)
	if msg != "[HIGH]" {
		t.Errorf("expected '[HIGH]', got '%s'", msg)
	}

	// Test case: error with empty everything
	err = &baseError{
		message:  "",
		category: "",
		class:    "",
		severity: "",
	}
	msg = buildMessage(err)
	if msg != "" {
		t.Errorf("expected empty string, got '%s'", msg)
	}
}

func TestBuildFieldsMessage_EdgeCases(t *testing.T) {
	// Test case: empty fields
	msg := buildFieldsMessage("test message", []any{})
	if msg != "test message" {
		t.Errorf("expected 'test message', got '%s'", msg)
	}

	// Test case: empty message with fields
	msg = buildFieldsMessage("", []any{"key", "value"})
	if msg != "error key=value" {
		t.Errorf("expected 'error key=value', got '%s'", msg)
	}

	// Test case: odd number of fields
	msg = buildFieldsMessage("test", []any{"key1", "value1", "key2"})
	if !strings.Contains(msg, "key1=value1") {
		t.Errorf("expected to contain 'key1=value1', got '%s'", msg)
	}

	// Test case: panic recovery
	msg = buildFieldsMessage("test", []any{"key", func() { panic("test panic") }})
	if !strings.Contains(msg, "test") {
		t.Errorf("expected 'test' after panic recovery, got '%s'", msg)
	}
}

func TestTruncateUTF8_EdgeCases(t *testing.T) {
	// Test case: maxBytes <= 0
	if truncateUTF8("test", 0) != "" {
		t.Error("expected empty string for maxBytes <= 0")
	}
	if truncateUTF8("test", -1) != "" {
		t.Error("expected empty string for maxBytes <= 0")
	}

	// Test case: string shorter than maxBytes
	if truncateUTF8("test", 10) != "test" {
		t.Error("expected unchanged string when shorter than maxBytes")
	}

	// Test case: invalid UTF-8 sequence
	invalidUTF8 := string([]byte{0xFF, 0xFE, 0xFD})
	result := truncateUTF8(invalidUTF8, 2)
	if result != "" {
		t.Errorf("expected empty string for invalid UTF-8, got '%s'", result)
	}

	// Test case: UTF-8 boundary
	utf8String := "你好世界"
	result = truncateUTF8(utf8String, 6) // 6 bytes = 2 UTF-8 characters
	if result != "你好" {
		t.Errorf("expected '你好', got '%s'", result)
	}
}

func TestApplyFormatVerbs_EdgeCases(t *testing.T) {
	// Test case: empty format string
	format, args := ApplyFormatVerbs("", "arg1", "arg2")
	if format != "" || len(args) != 2 {
		t.Error("expected empty format and all args returned")
	}

	// Test case: format with no verbs
	format, args = ApplyFormatVerbs("no verbs here", "arg1")
	if format != "no verbs here" || len(args) != 1 {
		t.Error("expected unchanged format and all args returned")
	}

	// Test case: format with %% (escaped percent)
	format, args = ApplyFormatVerbs("test %% percent", "arg1")
	if format != "test % percent" || len(args) != 1 {
		t.Errorf("expected 'test %% percent', got '%s'", format)
	}

	// Test case: format ending with %
	format, args = ApplyFormatVerbs("test %", "arg1")
	if !strings.Contains(format, "test %") {
		t.Errorf("expected format to contain 'test %%', got '%s'", format)
	}

	// Test case: more verbs than args
	format, args = ApplyFormatVerbs("test %s %s %s", "arg1")
	if !strings.Contains(format, "test arg1 %s %s") {
		t.Errorf("expected format to contain 'test arg1 %%s %%s', got '%s'", format)
	}

	// Test case: more args than verbs
	format, args = ApplyFormatVerbs("test %s", "arg1", "arg2", "arg3")
	if format != "test arg1" || len(args) != 2 {
		t.Errorf("expected 'test arg1' and 2 remaining args, got '%s' and %d args", format, len(args))
	}
}

func TestAtomicValue_EdgeCases(t *testing.T) {
	var av atomicValue[string]

	// Test case: Load when value is nil
	result := av.Load()
	if result != "" {
		t.Errorf("expected empty string for nil value, got '%s'", result)
	}

	// Test case: Load with wrong type stored
	av.value.Store(123)
	result = av.Load()
	if result != "" {
		t.Errorf("expected empty string for wrong type, got '%s'", result)
	}
}

func TestAppendValue_EdgeCases(t *testing.T) {
	var b strings.Builder

	// Test case: nil value
	appendValue(&b, nil, 10)
	if b.String() != "" {
		t.Error("expected empty string for nil value")
	}

	// Test case: nil stringer
	var nilStringer fmt.Stringer
	appendValue(&b, nilStringer, 10)
	if b.String() != "" {
		t.Error("expected empty string for nil stringer")
	}

	// Test case: nil error
	var nilErr error
	appendValue(&b, nilErr, 10)
	if b.String() != "" {
		t.Error("expected empty string for nil error")
	}

	// Test case: truncation of string
	b.Reset()
	appendValue(&b, "very long string that should be truncated", 5)
	if b.String() != "very " {
		t.Errorf("expected truncated string, got '%s'", b.String())
	}

	// Test case: truncation of bytes
	b.Reset()
	appendValue(&b, []byte("very long bytes that should be truncated"), 5)
	if len(b.String()) != 5 {
		t.Errorf("expected 5 bytes, got %d", len(b.String()))
	}

	// Test case: all numeric types
	b.Reset()
	appendValue(&b, int8(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, int16(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, int32(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, uint8(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, uint16(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, uint32(123), 10)
	if b.String() != "123" {
		t.Errorf("expected '123', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, float32(123.4), 10)
	if b.String() != "123.4" {
		t.Errorf("expected '123.4', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, float64(123.4), 10)
	if b.String() != "123.4" {
		t.Errorf("expected '123.4', got '%s'", b.String())
	}

	b.Reset()
	appendValue(&b, true, 10)
	if b.String() != "true" {
		t.Errorf("expected 'true', got '%s'", b.String())
	}

	// Test case: default case (struct)
	b.Reset()
	appendValue(&b, struct{ Name string }{"test"}, 50)
	if !strings.Contains(b.String(), "test") {
		t.Errorf("expected struct to contain 'test', got '%s'", b.String())
	}
}

func TestValueToString_EdgeCases(t *testing.T) {
	// Test case: nil value
	if valueToString(nil) != "" {
		t.Error("expected empty string for nil value")
	}

	// Test case: nil stringer
	var nilStringer fmt.Stringer
	if valueToString(nilStringer) != "" {
		t.Error("expected empty string for nil stringer")
	}

	// Test case: nil error
	var nilErr error
	if valueToString(nilErr) != "" {
		t.Error("expected empty string for nil error")
	}

	// Test case: redacted value
	if valueToString(Redact("secret")) != RedactedPlaceholder {
		t.Error("expected redacted placeholder")
	}

	// Test case: time
	now := time.Now()
	if valueToString(now) != now.Format(time.RFC3339) {
		t.Error("expected RFC3339 formatted time")
	}

	// Test case: stringer
	stringerObj := stringer{}
	if valueToString(stringerObj) != "stringer" {
		t.Error("expected stringer result")
	}

	// Test case: error
	if valueToString(fmt.Errorf("test error")) != "test error" {
		t.Error("expected error message")
	}

	// Test case: all numeric types
	if valueToString(int8(123)) != "123" {
		t.Error("expected '123' for int8")
	}
	if valueToString(int16(123)) != "123" {
		t.Error("expected '123' for int16")
	}
	if valueToString(int32(123)) != "123" {
		t.Error("expected '123' for int32")
	}
	if valueToString(uint8(123)) != "123" {
		t.Error("expected '123' for uint8")
	}
	if valueToString(uint16(123)) != "123" {
		t.Error("expected '123' for uint16")
	}
	if valueToString(uint32(123)) != "123" {
		t.Error("expected '123' for uint32")
	}
	if valueToString(float32(123.4)) != "123.4" {
		t.Error("expected '123.4' for float32")
	}
	if valueToString(float64(123.4)) != "123.4" {
		t.Error("expected '123.4' for float64")
	}
	if valueToString(true) != "true" {
		t.Error("expected 'true' for bool")
	}

	// Test case: string slice
	if valueToString([]string{"a", "b", "c"}) != "a,b,c" {
		t.Error("expected 'a,b,c' for string slice")
	}

	// Test case: default case (struct)
	result := valueToString(struct{ Name string }{"test"})
	if result != "{test}" {
		t.Errorf("expected '{test}', got '%s'", result)
	}
}

func TestNewID_Uniqueness(t *testing.T) {
	// Test that IDs are unique
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := newID()
		if ids[id] {
			t.Errorf("duplicate ID found: %s", id)
		}
		ids[id] = true
	}
}

func TestMergeFields(t *testing.T) {
	fields := []any{"key1", "value1", "key2", "value2"}
	opts := []any{"opt1", "optval1", "opt2", "optval2"}

	result := mergeFields(fields, opts)
	expected := []any{"opt1", "optval1", "opt2", "optval2", "key1", "value1", "key2", "value2"}

	if len(result) != len(expected) {
		t.Errorf("expected %d fields, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("expected %v at index %d, got %v", v, i, result[i])
		}
	}
}

func TestCountVerbs(t *testing.T) {
	// Test case: no verbs
	if countVerbs("no verbs here") != 0 {
		t.Error("expected 0 verbs")
	}

	// Test case: single verb
	if countVerbs("test %s") != 1 {
		t.Error("expected 1 verb")
	}

	// Test case: multiple verbs
	if countVerbs("test %s %d %v") != 3 {
		t.Error("expected 3 verbs")
	}

	// Test case: escaped percent
	if countVerbs("test %% percent") != 0 {
		t.Error("expected 0 verbs for escaped percent")
	}

	// Test case: percent at end
	if countVerbs("test %") != 0 {
		t.Error("expected 0 verbs for percent at end")
	}
}

func TestFormatError_EdgeCases(t *testing.T) {
	err := New("test error")

	// Test case: %s verb
	state := &testState{}
	formatError(err, state, 's')
	if state.String() != "test error" {
		t.Errorf("expected 'test error', got '%s'", state.String())
	}

	// Test case: %v verb (without + flag)
	state.Reset()
	formatError(err, state, 'v')
	if state.String() != "test error" {
		t.Errorf("expected 'test error', got '%s'", state.String())
	}

	// Test case: %v verb (with + flag)
	state.Reset()
	state.flags = 1 // fmt.Flag('+') = 1
	formatError(err, state, 'v')
	if !strings.Contains(state.String(), "test error") {
		t.Errorf("expected to contain 'test error', got '%s'", state.String())
	}
}

type testState struct {
	strings.Builder
	flags int
}

func (t *testState) Flag(c int) bool {
	return t.flags&c != 0
}

func (t *testState) Width() (int, bool) {
	return 0, false
}

func (t *testState) Precision() (int, bool) {
	return 0, false
}
