package erro

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"
)

// baseError holds the root error with all context and metadata
type baseError struct {
	// Core error info
	originalErr error     // Original error if wrapping external error
	message     string    // Base message
	fullMessage string    // Full message with fields (caching)
	created     time.Time // Creation timestamp

	// Metadata
	id        string          // Error id
	fields    []any           // Key-value fields
	category  Category        // Error category
	class     Class           // Error class
	severity  Severity        // Error severity
	retryable bool            // Retryable flag
	ctx       context.Context // Associated context

	stackOnce sync.Once
	stack     rawStack // Stack trace (program counters only - resolved on demand)
	frames    Stack    // Stack trace frames (for caching)

	span Span // Span
}

// Error implements the error interface
func (e *baseError) Error() (out string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		e.fullMessage = out
	}()

	out = buildFieldsMessage(e.message, e.fields)
	// Always show severity label for base errors
	if e.severity != "" {
		out = e.severity.Label() + " " + out
	}

	if e.originalErr != nil {
		if out == "" {
			return e.originalErr.Error()
		}
		return out + ": " + e.originalErr.Error()
	}
	return out
}

// errorWithoutSeverity returns the error message without severity label
func (e *baseError) errorWithoutSeverity() (out string) {
	if e.fullMessage != "" {
		return e.fullMessage
	}
	defer func() {
		e.fullMessage = out
	}()

	out = buildFieldsMessage(e.message, e.fields)

	if e.originalErr != nil {
		if out == "" {
			return e.originalErr.Error()
		}
		return out + ": " + e.originalErr.Error()
	}
	return out
}

// Format implements fmt.Formatter for stack trace printing
func (e *baseError) Format(s fmt.State, verb rune) {
	formatError(e, s, verb)
}

// Unwrap implements the Unwrap interface
func (e *baseError) Unwrap() error {
	return e.originalErr
}

// Chaining methods for baseError
func (e *baseError) ID(idRaw ...string) Error {
	var id string
	if len(idRaw) > 0 {
		id = truncateString(idRaw[0], maxCodeLength)
	} else {
		id = newID(e.class, e.category, e.created)
	}
	e.id = id
	return e
}

func (e *baseError) Category(category Category) Error {
	e.category = category
	return e
}

func (e *baseError) Class(class Class) Error {
	e.class = class
	return e
}

func (e *baseError) Severity(severity Severity) Error {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	e.severity = severity
	e.fullMessage = ""
	return e
}

func (e *baseError) Fields(fields ...any) Error {
	e.fields = safeAppendFields(e.fields, prepareFields(fields))
	e.fullMessage = ""
	return e
}

func (e *baseError) Context(ctx context.Context) Error {
	e.ctx = ctx
	return e
}

func (e *baseError) Retryable(retryable bool) Error {
	e.retryable = retryable
	return e
}

func (e *baseError) Span(span Span) Error {
	span.SetAttributes(e.fields...)
	span.RecordError(e)
	e.span = span
	return e
}

// Getter methods for baseError
func (e *baseError) GetBase() Error              { return e }
func (e *baseError) GetContext() context.Context { return e.ctx }
func (e *baseError) GetID() string {
	if e.id == "" {
		e.id = newID(e.class, e.category, e.created)
	}
	return e.id
}
func (e *baseError) GetCategory() Category { return e.category }
func (e *baseError) GetClass() Class       { return e.class }
func (e *baseError) IsRetryable() bool     { return e.retryable }
func (e *baseError) GetSpan() Span         { return e.span }
func (e *baseError) GetFields() []any      { return e.fields }
func (e *baseError) GetCreated() time.Time { return e.created }
func (e *baseError) GetMessage() string    { return e.message }

// Severity checking methods
func (e *baseError) GetSeverity() Severity {
	if e.severity == "" {
		return SeverityUnknown
	}
	return e.severity
}
func (e *baseError) IsCritical() bool { return e.severity == SeverityCritical }
func (e *baseError) IsHigh() bool     { return e.severity == SeverityHigh }
func (e *baseError) IsMedium() bool   { return e.severity == SeverityMedium }
func (e *baseError) IsLow() bool      { return e.severity == SeverityLow }
func (e *baseError) IsInfo() bool     { return e.severity == SeverityInfo }
func (e *baseError) IsUnknown() bool {
	return e.severity == "" || e.severity == SeverityUnknown
}

func (e *baseError) Stack() Stack {
	if e.initStack() {
		e.fullMessage = ""
	}
	return e.frames
}
func (e *baseError) StackFormat() string {
	if e.initStack() {
		e.fullMessage = ""
	}
	return e.frames.FormatFull()
}
func (e *baseError) StackWithError() string {
	if e.initStack() {
		e.fullMessage = ""
	}
	return e.Error() + "\n" + e.frames.FormatFull()
}

// Is checks if this error matches the target error
func (e *baseError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check direct equality (fastest path)
	if e == target {
		return true
	}

	// Fast path for erro errors - compare by metadata first
	if targetErro, ok := target.(Error); ok {
		// Compare by id if both have non-empty ids (very fast)
		if e.id != "" && targetErro.GetID() != "" {
			return e.id == targetErro.GetID()
		}

		// Compare base messages without fields (fast)
		if e.message == targetErro.GetMessage() {
			return true
		}
	}

	// For external errors, check if we wrap it directly
	if e.originalErr != nil {
		// Direct reference comparison (very fast)
		if e.originalErr == target {
			return true
		}

		// If the wrapped error has an Is method, use it
		if x, ok := e.originalErr.(interface{ Is(error) bool }); ok {
			return x.Is(target)
		}

		// For external errors, compare the original error's string representation
		// This avoids building our full error string with fields
		return e.originalErr.Error() == target.Error()
	}

	// Last resort: only for comparison with external errors without originalErr
	// Try to be smart about when to do expensive string comparison
	if _, isErro := target.(Error); !isErro {
		// Target is external error, we are baseError - only compare if we have no fields
		if len(e.fields) == 0 && e.severity == "" {
			// No fields, safe to compare messages
			return e.message == target.Error()
		}
		// If we have fields, this comparison is likely wrong anyway since
		// external errors won't match our formatted string with fields
		return false
	}

	// Both are erro errors but didn't match on any fast path
	// This should be rare with good error design
	return false
}

func (e *baseError) initStack() (changed bool) {
	e.stackOnce.Do(func() {
		e.frames = e.stack.toFrames()
		changed = true
	})
	return
}

// newBaseError creates a new base error with security validation
func newBaseError(originalErr error, message string, fields ...any) *baseError {
	return newBaseErrorWithStackSkip(3, originalErr, message, fields...)
}

func newBaseErrorWithStackSkip(skip int, originalErr error, message string, fields ...any) *baseError {
	e := &baseError{
		originalErr: originalErr,
		message:     truncateString(message, maxMessageLength),
		created:     time.Now(),
		fields:      prepareFields(fields),
		stack:       captureStack(skip),
	}
	return e
}

// buildFieldsMessage creates message with fields appended
func buildFieldsMessage(message string, fields []any) string {
	if len(fields) == 0 {
		return message
	}

	msg := make([]byte, 0, len(message)+len(fields)*20)
	msg = append(msg, message...)

	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}

		msg = append(msg, ' ')
		key, ok := fields[i].(string)
		if !ok {
			key = valueToString(fields[i])
		}
		msg = append(msg, truncateString(key, maxFieldKeyLength)...)
		msg = append(msg, '=')
		value, ok := fields[i+1].(string)
		if !ok {
			value = valueToString(fields[i+1])
		}
		msg = append(msg, truncateString(value, maxFieldValueLength)...)
	}

	return unsafe.String(unsafe.SliceData(msg), len(msg))
}

// valueToStringTruncated converts any value to string and truncates efficiently using byte-based approach
func valueToString(value any) string {
	if value == nil {
		return ""
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	case time.Time:
		str = v.Format(time.RFC3339)
	case fmt.Stringer:
		if v == nil {
			return ""
		}
		str = v.String()
	case error:
		if v == nil {
			return ""
		}
		str = v.Error()
	case int:
		return strconv.FormatInt(int64(v), 10) // Numbers don't need truncation
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case []string:
		str = strings.Join(v, ",")
	default:
		str = fmt.Sprintf("%v", v)
	}

	return str
}

func truncateString[T ~string](s T, maxLen int) T {
	// Efficient byte-based truncation that preserves UTF-8 boundaries
	if len(s) <= maxLen {
		return s
	}

	// Fast path for ASCII strings (most common case)
	if isASCII(s) {
		return s[:maxLen]
	}

	// Slow path for UTF-8 strings - truncate at safe boundary
	return truncateUTF8(s, maxLen)
}

// isASCII checks if string contains only ASCII characters (fast path)
func isASCII[T ~string](s T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// truncateUTF8 truncates string at UTF-8 boundary without expensive rune conversion
func truncateUTF8[T ~string](s T, maxBytes int) T {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}

	// Find the largest valid UTF-8 prefix within maxBytes
	for i := maxBytes; i > 0; i-- {
		if utf8.ValidString(string(s[:i])) {
			return s[:i]
		}
	}
	return ""
}

// countFormatVerbs counts the number of format verbs in a format string
func countFormatVerbs(format string) int {
	count := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			if i+1 < len(format) && format[i+1] != '%' {
				count++
				// Skip the verb character
				i++
			} else if i+1 < len(format) && format[i+1] == '%' {
				// Skip escaped %
				i++
			}
		}
	}
	return count
}

func formatError(err Error, s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Print with stack trace
			fmt.Fprint(s, err.Error())

			config := GetStackTraceConfig()
			if !config.Enabled {
				return // No stack trace in disabled mode
			}
			fmt.Fprint(s, "\nStack trace:\n")
			fmt.Fprint(s, err.Stack().FormatFull())

		} else {
			fmt.Fprint(s, err.Error())
		}
	case 's':
		fmt.Fprint(s, err.Error())
	}
}

func newID(class Class, category Category, created ...time.Time) string {
	var buf [8]byte

	if len(class) < 2 {
		buf[0] = 'X'
		buf[1] = 'X'
	} else {
		buf[0] = toUpperByte(class[0])
		buf[1] = toUpperByte(class[1])
	}
	if len(category) < 2 {
		buf[2] = 'X'
		buf[3] = 'X'
	} else {
		buf[2] = toUpperByte(category[0])
		buf[3] = toUpperByte(category[1])
	}
	buf[4] = '_'

	// Always use a 4-digit unique value (0-9999), padded to 4 bytes
	var unique int
	if len(created) == 0 || created[0].IsZero() {
		unique = rand.Intn(1000)
	} else {
		unique = int(created[0].UnixMicro() % 1000)
	}
	// Format unique as 3-digit zero-padded string
	buf[5] = '0' + byte(unique/100)
	buf[6] = '0' + byte((unique/10)%10)
	buf[7] = '0' + byte(unique%10)

	return string(buf[:])
}

func toUpperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 'a' + 'A'
	}
	return b
}
