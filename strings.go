package erro

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

func GetFormatErrorWithFullContext(optFuncs ...LogOption) FormatErrorFunc {
	return func(err Error) string {
		fields := getLogFields(err, DefaultLogOptions.ApplyOptions(optFuncs...))
		return buildFieldsMessage(err.Message(), fields)
	}
}

func GetFormatErrorWithFullContextBase(optFuncs ...LogOption) FormatErrorFunc {
	return func(err Error) string {
		if _, ok := err.(*baseError); ok {
			return GetFormatErrorWithFullContext(optFuncs...)(err)
		}
		return FormatErrorWithFields(err)
	}
}

func FormatErrorWithFieldsAndSeverity(err Error) string {
	severity := err.Severity()
	if severity.IsUnknown() {
		return buildFieldsMessage(err.Message(), err.Fields())
	}
	return severity.Label() + " " + buildFieldsMessage(err.Message(), err.Fields())
}

func FormatErrorWithFields(err Error) string {
	return buildFieldsMessage(err.Message(), err.Fields())
}

func FormatErrorWithSeverity(err Error) string {
	severity := err.Severity()
	if severity.IsUnknown() {
		return err.Message()
	}
	return severity.Label() + " " + err.Message()
}

func FormatErrorMessage(err Error) string {
	return err.Message()
}

// buildFieldsMessage creates message with fields appended
func buildFieldsMessage(message string, fields []any) (out string) {
	if len(fields) == 0 {
		return message
	}

	defer func() {
		if r := recover(); r != nil {
			// Fallback to safe string conversion
			out = message
		}
	}()

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
		msg = append(msg, truncateString(key, MaxKeyLength)...)
		msg = append(msg, '=')
		value, ok := fields[i+1].(string)
		if !ok {
			value = valueToString(fields[i+1])
		}
		msg = append(msg, truncateString(value, MaxValueLength)...)
	}

	return string(msg)
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
	case RedactedValue:
		return RedactedPlaceholder
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

			stack := err.Stack()
			if len(stack) > 0 {
				fmt.Fprint(s, "\nStack trace:\n")
				fmt.Fprint(s, stack.FormatFull())
			}

		} else {
			fmt.Fprint(s, err.Error())
		}
	case 's':
		fmt.Fprint(s, err.Error())
	}
}

func newID() string {
	var buf [8]byte

	rnd := rand.Int63()
	for i := 0; i < len(buf); i++ {
		n := int((rnd >> (6 * i)) & 0x3F)
		n = n % 36
		if n < 10 {
			buf[i] = '0' + byte(n)
		} else {
			buf[i] = 'a' + byte(n-10)
		}
	}
	return string(buf[:])
}

type atomicValue[T any] struct {
	value atomic.Value
}

func (a *atomicValue[T]) Load() T {
	if a.value.Load() == nil {
		return *new(T)
	}
	out, ok := a.value.Load().(T)
	if !ok {
		return *new(T)
	}
	return out
}

func (a *atomicValue[T]) Store(value T) {
	a.value.Store(value)
}
