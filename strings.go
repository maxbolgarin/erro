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

func init() {
	// Seed the random number generator for better randomness
	rand.Seed(time.Now().UnixNano())
}

// GetFormatErrorWithFullContextBase returns a [FormatErrorFunc] that formats an error with full context.
func GetFormatErrorWithFullContextBase(optFuncs ...LogOption) FormatErrorFunc {
	return func(err Error) string {
		if _, ok := err.(*baseError); ok {
			return GetFormatErrorWithFullContext(optFuncs...)(err)
		}
		return FormatErrorWithFields(err)
	}
}

// GetFormatErrorWithFullContext returns a [FormatErrorFunc] that formats an error with full context, including log fields.
func GetFormatErrorWithFullContext(optFuncs ...LogOption) FormatErrorFunc {
	return func(err Error) string {
		fields := getLogFields(err, DefaultLogOptions.ApplyOptions(optFuncs...))
		return buildFieldsMessage(buildMessage(err), fields)
	}
}

// FormatErrorWithFields formats an [Error] with its message and fields.
func FormatErrorWithFields(err Error) string {
	return buildFieldsMessage(buildMessage(err), err.Fields())
}

// FormatErrorMessage formats an error with its message only.
func FormatErrorMessage(err Error) string {
	return buildMessage(err)
}

func buildMessage(err Error) string {
	e, ok := err.(*baseError)
	if !ok {
		return err.Message()
	}

	if len(e.message) > 0 {
		return e.message
	}

	var msg strings.Builder
	msg.Grow(len(e.category) + len(e.class) + 2)
	if e.category != "" {
		msg.WriteString(e.category.String())
		if e.class != "" {
			msg.WriteRune(' ')
		}
	}
	if e.class != "" {
		msg.WriteString(e.class.String())
	}

	if msg.Len() == 0 {
		if e.severity != "" {
			return e.severity.Label()
		}
		return ""
	}

	return msg.String()
}

func buildFieldsMessage(message string, fields []any) (out string) {
	if len(fields) == 0 {
		return message
	}
	if message == "" {
		message = "error"
	}

	defer func() {
		if r := recover(); r != nil {
			out = message
		}
	}()

	var msg strings.Builder
	msg.Grow(len(message) + len(fields)*20)
	msg.WriteString(message)

	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}

		msg.WriteRune(' ')
		appendValue(&msg, fields[i], MaxKeyLength)
		msg.WriteRune('=')
		appendValue(&msg, fields[i+1], MaxValueLength)
	}

	return msg.String()
}

func truncateString[T ~string](s T, maxLen int) T {
	if len(s) <= maxLen {
		return s
	}
	if isASCII(s) {
		return s[:maxLen]
	}
	return truncateUTF8(s, maxLen)
}

func isASCII[T ~string](s T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func truncateUTF8[T ~string](s T, maxBytes int) T {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	for i := maxBytes; i > 0; i-- {
		if utf8.ValidString(string(s[:i])) {
			return s[:i]
		}
	}
	return ""
}

// ApplyFormatVerbs replaces format verbs in a string with arguments.
func ApplyFormatVerbs(format string, args ...any) (string, []any) {
	if format == "" {
		return "", args
	}

	var hasFormats bool
	for i := 0; i < len(format); i++ {
		if format[i] == '%' {
			hasFormats = true
			break
		}
	}
	if !hasFormats {
		return format, args
	}

	var result strings.Builder
	result.Grow(len(format) + len(args)*8)
	argIdx := 0
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			result.WriteByte(format[i])
			i++
			continue
		}

		if i+1 >= len(format) {
			result.WriteByte(format[i])
			i++
			continue
		}

		if format[i+1] == '%' {
			result.WriteRune('%')
			i += 2
			continue
		}

		if argIdx < len(args) {
			appendValue(&result, args[argIdx], MaxValueLength)
			argIdx++
			i += 2
		} else {
			result.WriteByte(format[i])
			result.WriteByte(format[i+1])
			i += 2
		}
	}

	return result.String(), args[argIdx:]
}

func formatError(err Error, s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
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

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var globalSequence int64

func newID(seed int64) string {
	seq := atomic.AddInt64(&globalSequence, 1) & 0xFFF
	combined := (seed << 12) | seq
	return encodeCompact(combined)
}

func encodeCompact(n int64) string {
	var buf [11]byte // Enough for 64-bit in base62
	i := 10
	for n > 0 {
		buf[i] = alphabet[n%62]
		n /= 62
		i--
	}
	return string(buf[i+1:])
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

func mergeFields(fields []any, opts []any) []any {
	newFields := make([]any, 0, len(fields)+len(opts))
	newFields = append(newFields, opts...)
	newFields = append(newFields, fields...)
	return newFields
}

func countVerbs(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			if i+1 < len(s) && s[i+1] != '%' {
				count++
			} else {
				i++
			}
		}
	}
	return count
}

func appendValue(b *strings.Builder, value any, maxLen int) {
	if value == nil {
		return
	}

	var tmp [64]byte

	switch v := value.(type) {
	case string:
		b.WriteString(truncateString(v, maxLen))
	case RedactedValue:
		b.WriteString(RedactedPlaceholder)
	case []byte:
		if len(v) > maxLen {
			v = v[:maxLen]
		}
		b.Write(v)
	case time.Time:
		b.WriteString(truncateString(v.Format(time.RFC3339), maxLen))
	case fmt.Stringer:
		if v != nil {
			b.WriteString(truncateString(v.String(), maxLen))
		}
	case error:
		if v != nil {
			b.WriteString(truncateString(v.Error(), maxLen))
		}
	case int:
		b.Write(strconv.AppendInt(tmp[:0], int64(v), 10))
	case int8:
		b.Write(strconv.AppendInt(tmp[:0], int64(v), 10))
	case int16:
		b.Write(strconv.AppendInt(tmp[:0], int64(v), 10))
	case int32:
		b.Write(strconv.AppendInt(tmp[:0], int64(v), 10))
	case int64:
		b.Write(strconv.AppendInt(tmp[:0], v, 10))
	case uint:
		b.Write(strconv.AppendUint(tmp[:0], uint64(v), 10))
	case uint8:
		b.Write(strconv.AppendUint(tmp[:0], uint64(v), 10))
	case uint16:
		b.Write(strconv.AppendUint(tmp[:0], uint64(v), 10))
	case uint32:
		b.Write(strconv.AppendUint(tmp[:0], uint64(v), 10))
	case uint64:
		b.Write(strconv.AppendUint(tmp[:0], v, 10))
	case float32:
		b.Write(strconv.AppendFloat(tmp[:0], float64(v), 'g', -1, 32))
	case float64:
		b.Write(strconv.AppendFloat(tmp[:0], v, 'g', -1, 64))
	case bool:
		b.Write(strconv.AppendBool(tmp[:0], v))
	case []string:
		b.WriteString(truncateString(strings.Join(v, ","), maxLen))
	default:
		b.WriteString(truncateString(fmt.Sprintf("%v", v), maxLen))
	}
}

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
		str = v.String()
	case error:
		str = v.Error()
	case int:
		return strconv.FormatInt(int64(v), 10)
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
