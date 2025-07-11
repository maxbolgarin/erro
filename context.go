package erro

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrorContext contains all extractable context from an error
type ErrorContext struct {
	Message   string          // Base error message
	Function  string          // Function where error was created
	Package   string          // Package where error was created
	File      string          // File where error was created
	Line      int             // Line where error was created
	Fields    map[string]any  // All key-value pairs
	Code      string          // Error code
	Category  string          // Error category
	Severity  string          // Error severity
	Tags      []string        // Error tags
	Retryable bool            // Whether error is retryable
	CreatedAt time.Time       // When error was created
	TraceID   string          // Trace ID if available
	Context   context.Context // Associated context
}

// ExtractContext extracts all available context from an error
func ExtractContext(err error) *ErrorContext {
	if err == nil {
		return nil
	}
	erroErr, ok := err.(Error)
	if !ok {
		return &ErrorContext{
			Message: err.Error(),
		}
	}
	base := erroErr.GetBase()

	// Extract fields as map
	fields := make(map[string]any)
	allFields := erroErr.GetFields()
	for i := 0; i < len(allFields); i += 2 {
		if i+1 < len(allFields) {
			key := fmt.Sprintf("%v", allFields[i])
			fields[key] = allFields[i+1]
		}
	}

	// Extract origin context from stack on demand
	var function, pkg, file string
	var line int

	if !base.stack.IsEmpty() {
		// Find the first user code frame for function context
		stackFrames := base.stack.ToFrames()
		stackType := Stack(stackFrames)
		if topUserFrame := stackType.TopUserFrame(); topUserFrame != nil {
			function = topUserFrame.Name
			pkg = topUserFrame.Package
			file = topUserFrame.File
			line = topUserFrame.Line
		} else if len(stackFrames) > 0 {
			// Fallback to first frame if no user code found
			frame := stackFrames[0]
			function = frame.Name
			pkg = frame.Package
			file = frame.File
			line = frame.Line
		}
	}

	return &ErrorContext{
		Message:   base.message,
		Function:  function,
		Package:   pkg,
		File:      file,
		Line:      line,
		Fields:    fields,
		Code:      base.code,
		Category:  base.category,
		Severity:  base.severity,
		Tags:      base.tags,
		Retryable: base.retryable,
		CreatedAt: base.createdAt,
		TraceID:   base.traceID,
		Context:   base.ctx,
	}

}

const (
	maxFieldsInLog   = 100
	maxValueLenInLog = 1000
)

// LoggingContext contains all context information for structured logging
type LoggingContext struct {
	Message     string         // Primary error message
	Fields      map[string]any // Key-value fields
	StackFields map[string]any // Stack-derived fields
	ErrorMeta   map[string]any // Error metadata (code, category, etc.)
	Context     map[string]any // Additional context
}

// BuildLoggingContext creates a comprehensive logging context from an error
func BuildLoggingContext(err error) *LoggingContext {
	if err == nil {
		return nil
	}

	erroErr, ok := err.(Error)
	if !ok {
		return &LoggingContext{
			Message: err.Error(),
		}
	}

	ctx := &LoggingContext{
		Fields:      make(map[string]any),
		StackFields: make(map[string]any),
		ErrorMeta:   make(map[string]any),
		Context:     make(map[string]any),
	}

	base := erroErr.GetBase()
	ctx.Message = base.message

	// Extract fields
	ctx.Fields = extractFromFields(erroErr.GetFields())

	// Extract stack fields
	if !base.stack.IsEmpty() {
		ctx.StackFields = base.stack.ToLogFields()
	}

	// Extract error metadata
	if code := erroErr.GetCode(); code != "" {
		ctx.ErrorMeta["code"] = code
	}
	if category := erroErr.GetCategory(); category != "" {
		ctx.ErrorMeta["category"] = category
	}
	if severity := erroErr.GetSeverity(); severity != "" {
		ctx.ErrorMeta["severity"] = severity
	}
	if tags := erroErr.GetTags(); len(tags) > 0 {
		ctx.ErrorMeta["tags"] = tags
	}
	if traceID := erroErr.GetTraceID(); traceID != "" {
		ctx.ErrorMeta["trace_id"] = traceID
	}
	if erroErr.IsRetryable() {
		ctx.ErrorMeta["retryable"] = true
	}

	// Extract timing information
	ctx.ErrorMeta["created_at"] = base.createdAt

	// Extract function context from stack on demand
	if !base.stack.IsEmpty() {
		if topUserFrame := base.stack.TopUserFrame(); topUserFrame != nil {
			if topUserFrame.Name != "" {
				ctx.Context["function"] = topUserFrame.Name
			}
			if topUserFrame.Package != "" {
				ctx.Context["package"] = topUserFrame.Package
			}
		} else {
			// Fallback to first frame
			stackFrames := base.stack.ToFrames()
			if len(stackFrames) > 0 {
				frame := stackFrames[0]
				if frame.Name != "" {
					ctx.Context["function"] = frame.Name
				}
				if frame.Package != "" {
					ctx.Context["package"] = frame.Package
				}
			}
		}
	}

	return ctx
}

// ToSlogFields converts logging context to slog-compatible fields
func (lc *LoggingContext) ToSlogFields() []any {
	var fields []any

	// Add all field types
	for key, value := range lc.Fields {
		fields = append(fields, key, value)
	}
	for key, value := range lc.StackFields {
		fields = append(fields, key, value)
	}
	for key, value := range lc.ErrorMeta {
		fields = append(fields, key, value)
	}
	for key, value := range lc.Context {
		fields = append(fields, key, value)
	}

	return fields
}

// ToLogrusFields converts logging context to logrus-compatible fields
func (lc *LoggingContext) ToLogrusFields() map[string]any {
	fields := make(map[string]any)

	// Merge all field types
	for key, value := range lc.Fields {
		fields[key] = value
	}
	for key, value := range lc.StackFields {
		fields[key] = value
	}
	for key, value := range lc.ErrorMeta {
		fields[key] = value
	}
	for key, value := range lc.Context {
		fields[key] = value
	}

	return fields
}

// ToZapFields converts logging context to zap-compatible fields
func (lc *LoggingContext) ToZapFields() map[string]any {
	// For zap, we can use the same format as logrus
	return lc.ToLogrusFields()
}

// ToGenericMap converts to a generic map for any logging framework
func (lc *LoggingContext) ToGenericMap() map[string]any {
	return lc.ToLogrusFields()
}

// Enhanced field parsing for error messages
type FieldParser struct {
	quoteChar  rune
	escapeChar rune
	separators []rune
	equalChar  rune
}

// NewFieldParser creates a new field parser with default settings
func NewFieldParser() *FieldParser {
	return &FieldParser{
		quoteChar:  '"',
		escapeChar: '\\',
		separators: []rune{' ', '\t'},
		equalChar:  '=',
	}
}

// ParseFromMessage parses key=value pairs from an error message
func (fp *FieldParser) ParseFromMessage(message string) (baseMessage string, fields map[string]string) {
	return fp.parseFieldsFromString(message)
}

// parseFieldsFromString does the actual parsing work
func (fp *FieldParser) parseFieldsFromString(message string) (string, map[string]string) {
	if message == "" {
		return "", nil
	}

	// For wrapped errors, only parse the part before the first colon
	wrapIndex := fp.findWrapDelimiter(message)
	originalMessage := message
	if wrapIndex != -1 {
		message = message[:wrapIndex]
	}

	fields := make(map[string]string)
	tokens := fp.parseTokens(message)

	if len(tokens) == 0 {
		return originalMessage, nil
	}

	// Find the first field (contains '=')
	firstFieldIndex := -1
	for i, token := range tokens {
		if strings.Contains(token, string(fp.equalChar)) {
			firstFieldIndex = i
			break
		}
	}

	// If no fields found, return the entire message as base
	if firstFieldIndex == -1 {
		return originalMessage, nil
	}

	// Extract base message (everything before the first field)
	baseMessage := strings.Join(tokens[:firstFieldIndex], " ")

	// Extract fields
	for i := firstFieldIndex; i < len(tokens); i++ {
		token := tokens[i]
		if equalIndex := strings.Index(token, string(fp.equalChar)); equalIndex != -1 {
			key := token[:equalIndex]
			value := token[equalIndex+1:]

			if key != "" {
				unescapedValue := fp.unescapeValue(value)
				fields[key] = unescapedValue
			}
		}
	}

	return baseMessage, fields
}

// findWrapDelimiter finds the position of ": " that indicates error wrapping
func (fp *FieldParser) findWrapDelimiter(message string) int {
	var inQuotes bool
	var i int

	for i < len(message)-1 {
		r := rune(message[i])

		if inQuotes {
			if r == fp.escapeChar && i+1 < len(message) {
				i += 2
				continue
			} else if r == fp.quoteChar {
				inQuotes = false
			}
		} else {
			if r == fp.quoteChar {
				inQuotes = true
			} else if r == ':' && i+1 < len(message) && message[i+1] == ' ' {
				return i
			}
		}

		i++
	}

	return -1
}

// parseTokens splits a string into tokens, respecting quoted strings
func (fp *FieldParser) parseTokens(s string) []string {
	var tokens []string
	var current strings.Builder
	var inQuotes bool
	var i int

	for i < len(s) {
		r := rune(s[i])

		if inQuotes {
			if r == fp.escapeChar && i+1 < len(s) {
				current.WriteRune(r)
				i++
				if i < len(s) {
					current.WriteRune(rune(s[i]))
				}
			} else if r == fp.quoteChar {
				current.WriteRune(r)
				inQuotes = false
			} else {
				current.WriteRune(r)
			}
		} else {
			if r == fp.quoteChar {
				current.WriteRune(r)
				inQuotes = true
			} else if fp.isSeparator(r) {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		}

		i++
	}

	// Add the last token
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// isSeparator checks if a rune is a separator
func (fp *FieldParser) isSeparator(r rune) bool {
	for _, sep := range fp.separators {
		if r == sep {
			return true
		}
	}
	return false
}

// unescapeValue removes quotes and unescapes characters in a field value
func (fp *FieldParser) unescapeValue(value string) string {
	if len(value) < 2 {
		return value
	}

	// Check if value is quoted
	if value[0] == byte(fp.quoteChar) && value[len(value)-1] == byte(fp.quoteChar) {
		// Remove quotes and unescape
		inner := value[1 : len(value)-1]
		var result strings.Builder
		var escaped bool

		for _, r := range inner {
			if escaped {
				result.WriteRune(r)
				escaped = false
				continue
			}

			if r == fp.escapeChar {
				escaped = true
				continue
			}

			result.WriteRune(r)
		}

		return result.String()
	}

	return value
}

// Global convenience functions for field extraction

// ExtractFields extracts key-value pairs from an error message
func ExtractFields(err error) (string, map[string]string) {
	if err == nil {
		return "", nil
	}

	parser := NewFieldParser()
	return parser.ParseFromMessage(err.Error())
}

// ExtractFieldsFromString extracts key-value pairs from a message string
func ExtractFieldsFromString(message string) (string, map[string]string) {
	parser := NewFieldParser()
	return parser.ParseFromMessage(message)
}

// LogFields returns a slice of alternating key-value pairs for structured loggers
func LogFields(err error) []any {
	ctx := BuildLoggingContext(err)
	if ctx == nil {
		return nil
	}
	return ctx.ToSlogFields()
}

// LogFieldsMap returns a map of field key-value pairs for map-based loggers
func LogFieldsMap(err error) map[string]any {
	ctx := BuildLoggingContext(err)
	if ctx == nil {
		return nil
	}
	return ctx.ToGenericMap()
}

// WithLogger executes a callback with extracted error context for any logging library
func WithLogger(err error, logFunc func(message string, fields map[string]any)) {
	if err == nil || logFunc == nil {
		return
	}

	ctx := BuildLoggingContext(err)
	if ctx == nil {
		logFunc(err.Error(), nil)
		return
	}

	logFunc(ctx.Message, ctx.ToGenericMap())
}

// LogError extracts both message and fields for convenient logging
func LogError(err error) (message string, fields map[string]any) {
	ctx := BuildLoggingContext(err)
	if ctx == nil {
		return err.Error(), nil
	}
	return ctx.Message, ctx.ToGenericMap()
}

// extractFromFields converts a slice of fields to a map
func extractFromFields(fields []any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	maxFields := len(fields) / 2
	if maxFields > maxFieldsInLog {
		maxFields = maxFieldsInLog
	}

	result := make(map[string]any, maxFields)

	count := 0
	for i := 0; i < len(fields) && count < maxFields; i += 2 {
		if i+1 >= len(fields) {
			break
		}

		key := valueToString(fields[i])
		value := valueToString(fields[i+1])

		if len(value) > maxValueLenInLog {
			value = value[:maxValueLenInLog] + "..."
		}

		result[key] = value
		count++
	}

	return result
}

// valueToString converts any value to string
func valueToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
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
		return strings.Join(v, ",")
	default:
		return fmt.Sprintf("%v", v)
	}
}
