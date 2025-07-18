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
