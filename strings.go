package erro

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

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
	return truncateUTF8(s, maxLen)
}

func truncateUTF8[T ~string](s T, maxBytes int) T {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	// Walk back at most 3 bytes to find a valid UTF-8 character boundary.
	// Continuation bytes have the pattern 10xxxxxx (0x80-0xBF).
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

// ApplyFormatVerbs replaces format verbs in a string with arguments.
//
// It supports a subset of package fmt syntax: optional flags '+', '-', '#',
// '0', an optional width and precision (each up to 4 digits), and the verbs
// b c d e E f F g G o O p q s t T U v x X w ('%w' is formatted like '%v').
// '%%' emits a literal '%'. Anything else — the space flag, '*' width,
// argument indexes like '%[1]d', unknown verbs — is emitted literally and
// consumes no argument, so messages like "90% done" stay intact. Once
// arguments run out, remaining verbs are emitted literally. Unconsumed
// arguments are returned for use as key-value fields.
func ApplyFormatVerbs(format string, args ...any) (result string, rest []any) {
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

	// A panicking Stringer/Formatter argument must not crash error
	// construction; fall back to the raw format string.
	defer func() {
		if r := recover(); r != nil {
			result = format
			rest = args
		}
	}()

	var sb strings.Builder
	sb.Grow(len(format) + len(args)*8)
	argIdx := 0
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			sb.WriteByte(format[i])
			i++
			continue
		}

		if i+1 >= len(format) {
			sb.WriteByte(format[i])
			i++
			continue
		}

		if format[i+1] == '%' {
			sb.WriteByte('%')
			i += 2
			continue
		}

		end, ok := scanVerb(format, i)
		if !ok || argIdx >= len(args) || isOptionValue(args[argIdx]) {
			// Options and classifiers are never format arguments: once one
			// is reached, everything that follows is metadata.
			sb.WriteString(format[i:end])
			i = end
			continue
		}

		writeFormatted(&sb, format[i:end], args[argIdx])
		argIdx++
		i = end
	}

	return sb.String(), args[argIdx:]
}

// isOptionValue reports whether v is an erro option or classifier, which
// must never be consumed as a format argument.
func isOptionValue(v any) bool {
	_, ok := v.(erroOptionMarker)
	return ok
}

// scanVerb parses a fmt-style verb specification starting at format[i],
// which must be '%' not followed by another '%'. It returns the exclusive
// end index of the spec and whether it is a supported argument-consuming
// verb. On unsupported input it returns i+1 so the caller emits just the
// '%' literally and continues.
func scanVerb(format string, i int) (end int, ok bool) {
	j := i + 1
	for j < len(format) {
		if c := format[j]; c == '+' || c == '-' || c == '#' || c == '0' {
			j++
			continue
		}
		break
	}
	digits := 0
	for j < len(format) && format[j] >= '0' && format[j] <= '9' {
		j++
		digits++
	}
	// Bound width/precision so a verb cannot force huge fmt allocations.
	if digits > 4 {
		return i + 1, false
	}
	if j < len(format) && format[j] == '.' {
		j++
		digits = 0
		for j < len(format) && format[j] >= '0' && format[j] <= '9' {
			j++
			digits++
		}
		if digits > 4 {
			return i + 1, false
		}
	}
	if j >= len(format) {
		return i + 1, false
	}
	switch format[j] {
	case 'b', 'c', 'd', 'e', 'E', 'f', 'F', 'g', 'G', 'o', 'O',
		'p', 'q', 's', 't', 'T', 'U', 'v', 'x', 'X', 'w':
		return j + 1, true
	}
	return i + 1, false
}

// writeFormatted renders a single argument according to the verb spec.
// Plain verbs whose output matches appendValue take the fast path;
// everything else goes through fmt.
func writeFormatted(b *strings.Builder, spec string, arg any) {
	// Redacted values never reach fmt, whatever the verb.
	if _, ok := arg.(RedactedValue); ok {
		b.WriteString(RedactedPlaceholder)
		return
	}
	if len(spec) == 2 { // plain verb, no flags/width/precision
		switch spec[1] {
		case 's', 'v', 'w':
			appendValue(b, arg, MaxValueLength)
			return
		case 'd':
			switch arg.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				appendValue(b, arg, MaxValueLength)
				return
			}
		case 't':
			if _, ok := arg.(bool); ok {
				appendValue(b, arg, MaxValueLength)
				return
			}
		case 'f':
			// fmt's plain %f is FormatFloat(v, 'f', 6, bits).
			var tmp [32]byte
			switch v := arg.(type) {
			case float64:
				b.Write(strconv.AppendFloat(tmp[:0], v, 'f', 6, 64))
				return
			case float32:
				b.Write(strconv.AppendFloat(tmp[:0], float64(v), 'f', 6, 32))
				return
			}
		}
	}
	if spec[len(spec)-1] == 'w' {
		spec = spec[:len(spec)-1] + "v"
	}
	b.WriteString(truncateString(fmt.Sprintf(spec, arg), MaxValueLength))
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
	case 'q':
		fmt.Fprintf(s, "%q", err.Error())
	default:
		fmt.Fprintf(s, "%%!%c(%s)", verb, err.Error())
	}
}

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	globalSequence int64
	idPrefix       string
)

func init() {
	var buf [8]byte
	var prefix uint64
	if _, err := rand.Read(buf[:]); err == nil {
		prefix = binary.LittleEndian.Uint64(buf[:])
	} else {
		prefix = uint64(time.Now().UnixNano())
	}
	if prefix == 0 {
		prefix = 1
	}
	idPrefix = encodeCompact(int64(prefix))
}

// newID generates a unique error ID: a random per-process prefix followed by
// a monotonically increasing sequence. IDs are guaranteed unique within a
// process and collide across processes with ~2^-64 probability, which
// matters because ID equality is the primary match for [errors.Is].
// The seed parameter is retained for signature compatibility.
func newID(_ int64) string {
	seq := atomic.AddInt64(&globalSequence, 1)
	var buf [24]byte
	b := append(buf[:0], idPrefix...)
	b = appendCompact(b, seq)
	return string(b)
}

func encodeCompact(n int64) string {
	var buf [11]byte
	return string(appendCompact(buf[:0], n))
}

func appendCompact(dst []byte, n int64) []byte {
	// Use unsigned so negative inputs are encoded from their bit pattern.
	u := uint64(n)
	if u == 0 {
		return append(dst, alphabet[0])
	}
	var tmp [11]byte // Enough for 64-bit in base62
	i := 10
	for u > 0 {
		tmp[i] = alphabet[u%62]
		u /= 62
		i--
	}
	return append(dst, tmp[i+1:]...)
}

type atomicValue[T any] struct {
	value atomic.Value
}

func (a *atomicValue[T]) Load() T {
	out, ok := a.value.Load().(T)
	if !ok {
		return *new(T)
	}
	return out
}

func (a *atomicValue[T]) Store(value T) {
	a.value.Store(value)
}

// isNilValue reports whether v holds a typed-nil whose methods would panic
// when called through an interface.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

func mergeFields(fields []any, opts []any) []any {
	newFields := make([]any, 0, len(fields)+len(opts))
	newFields = append(newFields, opts...)
	newFields = append(newFields, fields...)
	return newFields
}

// countVerbs counts argument-consuming verbs using the same rules as
// [ApplyFormatVerbs], so template argument splitting stays consistent.
func countVerbs(s string) int {
	count := 0
	i := 0
	for i < len(s) {
		if s[i] != '%' {
			i++
			continue
		}
		if i+1 >= len(s) {
			break
		}
		if s[i+1] == '%' {
			i += 2
			continue
		}
		end, ok := scanVerb(s, i)
		if ok {
			count++
		}
		i = end
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
			cut := maxLen
			for cut > 0 && !utf8.RuneStart(v[cut]) {
				cut--
			}
			v = v[:cut]
		}
		b.Write(v)
	case time.Time:
		b.WriteString(truncateString(v.Format(time.RFC3339), maxLen))
	case fmt.Stringer:
		if isNilValue(v) {
			b.WriteString("<nil>")
		} else {
			b.WriteString(truncateString(v.String(), maxLen))
		}
	case error:
		if isNilValue(v) {
			b.WriteString("<nil>")
		} else {
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
		if isNilValue(v) {
			return "<nil>"
		}
		str = v.String()
	case error:
		if isNilValue(v) {
			return "<nil>"
		}
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
