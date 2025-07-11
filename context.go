package erro

import (
	"context"
	"fmt"
	"sort"
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

	allFields := erroErr.GetFields()
	fields := make(map[string]any, len(allFields)/2)

	for i := 0; i < len(allFields); i += 2 {
		if i+1 < len(allFields) {
			key := valueToString(allFields[i])
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
		Message:   extractFullMessageWithoutFields(erroErr), // Use full message chain instead of just base.message
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

// LogFields returns a slice of alternating key-value pairs for structured loggers
func LogFields(err error) []any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	return ctx.LogFields()
}

// LogFieldsMap returns a map of field key-value pairs for map-based loggers
func LogFieldsMap(err error) map[string]any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	return ctx.LogFieldsMap()
}

// WithLogger executes a callback with extracted error context for any logging library
func LogError(err error, logFunc func(message string, fields ...any)) {
	if err == nil || logFunc == nil {
		return
	}

	ctx := ExtractContext(err)
	if ctx == nil {
		logFunc(err.Error(), nil)
		return
	}

	logFunc(ctx.Message, ctx.LogFields()...)
}

// String formats the ErrorContext like a normal error with fields
func (ec *ErrorContext) String() string {
	if ec == nil {
		return ""
	}

	if len(ec.Fields) == 0 {
		return ec.Message
	}

	var builder strings.Builder
	// Estimate capacity: message + fields with reasonable estimates for key=value pairs
	estimatedSize := len(ec.Message) + len(ec.Fields)*20
	builder.Grow(estimatedSize)

	builder.WriteString(ec.Message)

	// Add fields in sorted order for consistent output
	keys := make([]string, 0, len(ec.Fields))
	for key := range ec.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(" ")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(valueToString(ec.Fields[key]))
	}

	return builder.String()
}

// ToSlogFields converts ErrorContext to slog-compatible fields
func (ec *ErrorContext) LogFields() []any {
	if ec == nil {
		return nil
	}

	fields := make([]any, 0, len(ec.Fields)+11)

	// Add user fields
	for key, value := range ec.Fields {
		fields = append(fields, key, value)
	}

	// Add error metadata
	if ec.Code != "" {
		fields = append(fields, "code", ec.Code)
	}
	if ec.Category != "" {
		fields = append(fields, "category", ec.Category)
	}
	if ec.Severity != "" {
		fields = append(fields, "severity", ec.Severity)
	}
	if len(ec.Tags) > 0 {
		fields = append(fields, "tags", ec.Tags)
	}
	if ec.TraceID != "" {
		fields = append(fields, "trace_id", ec.TraceID)
	}
	if ec.Retryable {
		fields = append(fields, "retryable", true)
	}

	// Add timing information
	fields = append(fields, "created_at", ec.CreatedAt)

	// Add function context
	if ec.Function != "" {
		fields = append(fields, "error_function", ec.Function)
	}
	if ec.Package != "" {
		fields = append(fields, "error_package", ec.Package)
	}
	if ec.File != "" {
		fields = append(fields, "error_file", ec.File)
	}
	if ec.Line > 0 {
		fields = append(fields, "error_line", ec.Line)
	}

	return fields
}

// ToLogrusFields converts ErrorContext to logrus-compatible fields
func (ec *ErrorContext) LogFieldsMap() map[string]any {
	if ec == nil {
		return nil
	}

	fields := make(map[string]any, len(ec.Fields)+11)

	// Add user fields
	for key, value := range ec.Fields {
		fields[key] = value
	}

	// Add error metadata
	if ec.Code != "" {
		fields["code"] = ec.Code
	}
	if ec.Category != "" {
		fields["category"] = ec.Category
	}
	if ec.Severity != "" {
		fields["severity"] = ec.Severity
	}
	if len(ec.Tags) > 0 {
		fields["tags"] = ec.Tags
	}
	if ec.TraceID != "" {
		fields["trace_id"] = ec.TraceID
	}
	if ec.Retryable {
		fields["retryable"] = true
	}

	// Add timing information
	fields["created_at"] = ec.CreatedAt

	// Add function context
	if ec.Function != "" {
		fields["error_function"] = ec.Function
	}
	if ec.Package != "" {
		fields["error_package"] = ec.Package
	}
	if ec.File != "" {
		fields["error_file"] = ec.File
	}
	if ec.Line > 0 {
		fields["error_line"] = ec.Line
	}

	return fields
}

// extractFullMessageWithoutFields builds the complete error message chain without field values
func extractFullMessageWithoutFields(err Error) string {
	switch e := err.(type) {
	case *wrapError:
		// Get wrap message without fields + ": " + wrapped error message without fields
		wrapMsg := e.wrapMessage
		if e.wrapped != nil {
			wrappedMsg := extractFullMessageWithoutFields(e.wrapped)
			return wrapMsg + ": " + wrappedMsg
		} else {
			// Fallback to base if no wrapped error
			baseMsg := e.base.message
			if e.base.originalErr != nil {
				return wrapMsg + ": " + baseMsg + ": " + e.base.originalErr.Error()
			}
			return wrapMsg + ": " + baseMsg
		}
	case *baseError:
		// Get base message without fields + optionally original error
		if e.originalErr != nil {
			return e.message + ": " + e.originalErr.Error()
		}
		return e.message
	default:
		// Fallback for unknown types
		return err.Error()
	}
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
