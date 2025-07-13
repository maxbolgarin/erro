package erro

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrorContext contains all extractable context from an error
type ErrorContext struct {
	Message      string          // Base error message
	Function     string          // Function where error was created
	Package      string          // Package where error was created
	File         string          // File where error was created
	Line         int             // Line where error was created
	Fields       map[string]any  // All key-value pairs
	ID           string          // Error id
	Category     Category        // Error category
	Class        Class           // Error class
	Severity     Severity        // Error severity
	Retryable    bool            // Whether error is retryable
	Created      time.Time       // When error was created
	TraceID      string          // Trace ID if available
	SpanID       string          // Span ID if available
	ParentSpanID string          // Parent Span ID if available
	Context      context.Context // Associated context
	Stack        Stack           // Stack trace frames

	fields []any
}

// Severity checking methods for ErrorContext
func (ec *ErrorContext) IsCritical() bool { return ec.Severity == SeverityCritical }
func (ec *ErrorContext) IsHigh() bool     { return ec.Severity == SeverityHigh }
func (ec *ErrorContext) IsMedium() bool   { return ec.Severity == SeverityMedium }
func (ec *ErrorContext) IsLow() bool      { return ec.Severity == SeverityLow }
func (ec *ErrorContext) IsInfo() bool     { return ec.Severity == SeverityInfo }
func (ec *ErrorContext) IsUnknown() bool {
	return ec.Severity == "" || ec.Severity == SeverityUnknown
}
func (ec *ErrorContext) GetSeverity() Severity {
	if ec.Severity == "" {
		return SeverityUnknown
	}
	return ec.Severity
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

	baseInt := erroErr.GetBase()
	base, ok := baseInt.(*baseError)

	if ok && !base.stack.isEmpty() {
		// Find the first user code frame for function context
		stackFrames := base.stack.toFrames()
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

	ec := &ErrorContext{
		Message:   extractFullMessageWithoutFields(erroErr), // Use full message chain instead of just base.message
		Function:  function,
		Package:   pkg,
		File:      file,
		Line:      line,
		Fields:    fields,
		ID:        baseInt.GetID(),
		Category:  baseInt.GetCategory(),
		Class:     baseInt.GetClass(),
		Severity:  baseInt.GetSeverity(),
		Retryable: baseInt.IsRetryable(),
		Created:   baseInt.GetCreated(),
		Context:   baseInt.GetContext(),
		Stack:     baseInt.Stack(),
		fields:    allFields,
	}

	if span := baseInt.GetSpan(); span != nil {
		ec.TraceID = span.TraceID()
		ec.SpanID = span.SpanID()
		ec.ParentSpanID = span.ParentSpanID()
	}
	return ec
}

// LogFields returns a slice of alternating key-value pairs for structured loggers
func LogFields(err error, optFuncs ...func(*LogOptions)) []any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions().ApplyOptions(optFuncs...)
	return ctx.LogFields(opts)
}

// LogFieldsMap returns a map of field key-value pairs for map-based loggers
func LogFieldsMap(err error, optFuncs ...func(*LogOptions)) map[string]any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions().ApplyOptions(optFuncs...)
	return ctx.LogFieldsMap(opts)
}

// WithLogger executes a callback with extracted error context for any logging library
func LogError(err error, logFunc func(message string, fields ...any), optFuncs ...func(*LogOptions)) {
	if err == nil || logFunc == nil {
		return
	}

	ctx := ExtractContext(err)
	if ctx == nil {
		logFunc(err.Error(), nil)
		return
	}

	opts := DefaultLogOptions().ApplyOptions(optFuncs...)
	logFunc(ctx.Message, ctx.LogFields(opts)...)
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

// LogOptions controls which fields are included in logging output
type LogOptions struct {
	// User-defined fields
	IncludeUserFields bool

	// Error metadata
	IncludeID        bool
	IncludeCategory  bool
	IncludeSeverity  bool
	IncludeRetryable bool
	IncludeTracing   bool
	ContextFields    []string

	// Timing information
	IncludeCreatedTime bool

	// Stack context
	IncludeFunction bool
	IncludePackage  bool
	IncludeFile     bool
	IncludeLine     bool
	IncludeStack    bool

	// Stack formatting options
	StackFormat StackFormat

	// Field name prefix. Default is "error_"
	FieldNamePrefix string
}

// StackFormat defines how stack traces should be formatted
type StackFormat int

const (
	StackFormatString StackFormat = iota // Default string format
	StackFormatList                      // List of call chain
	StackFormatFull                      // Full stack trace format
	StackFormatJSON                      // JSON format for structured logging
)

// Default options that include commonly needed fields
func DefaultLogOptions() *LogOptions {
	return &LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeTracing:     true,
		IncludeCreatedTime: false, // Often too verbose
		IncludeRetryable:   true,
		IncludeFunction:    true,
		IncludePackage:     false,
		IncludeFile:        false,
		IncludeLine:        false,
		IncludeStack:       false, // Often too verbose
		StackFormat:        StackFormatJSON,
		FieldNamePrefix:    "error_",
	}
}

// Minimal options that include only essential fields
func MinimalLogOptions() *LogOptions {
	return &LogOptions{
		IncludeUserFields: true,
		IncludeID:         true,
		IncludeSeverity:   true,
		StackFormat:       StackFormatJSON,
		FieldNamePrefix:   "error_",
	}
}

func MinimalLogOpts() []func(*LogOptions) {
	return []func(*LogOptions){
		WithUserFields(true),
		WithID(true),
		WithSeverity(true),
		WithCategory(false),
		WithTracing(false),
		WithRetryable(false),
		WithCreatedTime(false),
		WithFunction(false),
		WithPackage(false),
		WithFile(false),
		WithLine(false),
		WithStack(false),
	}
}

// Verbose options that include all available fields
func VerboseLogOptions() *LogOptions {
	return &LogOptions{
		IncludeUserFields:  true,
		IncludeID:          true,
		IncludeCategory:    true,
		IncludeSeverity:    true,
		IncludeTracing:     true,
		IncludeRetryable:   true,
		IncludeCreatedTime: true,
		IncludeFunction:    true,
		IncludePackage:     true,
		IncludeFile:        true,
		IncludeLine:        true,
		IncludeStack:       true,
		StackFormat:        StackFormatJSON,
		FieldNamePrefix:    "error_",
	}
}

func VerboseLogOpts() []func(*LogOptions) {
	return []func(*LogOptions){
		WithUserFields(true),
		WithID(true),
		WithSeverity(true),
		WithCategory(true),
		WithTracing(true),
		WithRetryable(true),
		WithCreatedTime(true),
		WithFunction(true),
		WithPackage(true),
		WithFile(true),
		WithLine(true),
		WithStack(true),
	}
}

func EmptyLogOptions() *LogOptions {
	return &LogOptions{}
}

func EmptyLogOpts() []func(*LogOptions) {
	return []func(*LogOptions){
		WithUserFields(false),
		WithID(false),
		WithSeverity(false),
		WithCategory(false),
		WithTracing(false),
		WithRetryable(false),
		WithCreatedTime(false),
		WithFunction(false),
		WithPackage(false),
		WithFile(false),
		WithLine(false),
		WithStack(false),
		WithFieldNamePrefix(""),
	}
}

func MergeLogOpts(opts []func(*LogOptions), addsOpts ...func(*LogOptions)) []func(*LogOptions) {
	return append(opts, addsOpts...)
}

// WithUserFields enables/disables user-defined fields
func WithUserFields(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeUserFields = true
		if len(include) > 0 {
			opts.IncludeUserFields = include[0]
		}
	}
}

// WithContextFields sets the context fields to include in the log output
func WithContextFields(fields ...string) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.ContextFields = fields
	}
}

// WithID enables/disables error id field
func WithID(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeID = true
		if len(include) > 0 {
			opts.IncludeID = include[0]
		}
	}
}

// WithCategory enables/disables error category field
func WithCategory(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeCategory = true
		if len(include) > 0 {
			opts.IncludeCategory = include[0]
		}
	}
}

// WithSeverity enables/disables error severity field
func WithSeverity(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeSeverity = true
		if len(include) > 0 {
			opts.IncludeSeverity = include[0]
		}
	}
}

// WithTracing enables/disables tracing field
func WithTracing(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeTracing = true
		if len(include) > 0 {
			opts.IncludeTracing = include[0]
		}
	}
}

// WithRetryable enables/disables retryable flag field
func WithRetryable(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeRetryable = true
		if len(include) > 0 {
			opts.IncludeRetryable = include[0]
		}
	}
}

// WithCreatedTime enables/disables creation timestamp field
func WithCreatedTime(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeCreatedTime = true
		if len(include) > 0 {
			opts.IncludeCreatedTime = include[0]
		}
	}
}

// WithFunction enables/disables function name field
func WithFunction(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeFunction = true
		if len(include) > 0 {
			opts.IncludeFunction = include[0]
		}
	}
}

// WithPackage enables/disables package name field
func WithPackage(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludePackage = true
		if len(include) > 0 {
			opts.IncludePackage = include[0]
		}
	}
}

// WithFile enables/disables file name field
func WithFile(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeFile = true
		if len(include) > 0 {
			opts.IncludeFile = include[0]
		}
	}
}

// WithLine enables/disables line number field
func WithLine(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeLine = true
		if len(include) > 0 {
			opts.IncludeLine = include[0]
		}
	}
}

// WithStack enables/disables full stack trace field
func WithStack(include ...bool) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeStack = true
		if len(include) > 0 {
			opts.IncludeStack = include[0]
		}
	}
}

// WithStackFormat sets the stack trace format
func WithStackFormat(format StackFormat) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.IncludeStack = true
		opts.StackFormat = format
	}
}

// WithFieldNamePrefix sets the field name prefix
func WithFieldNamePrefix(prefix string) func(*LogOptions) {
	return func(opts *LogOptions) {
		opts.FieldNamePrefix = prefix
	}
}

// ApplyOptions applies a set of option functions to LogOptions
func (opts *LogOptions) ApplyOptions(optFuncs ...func(*LogOptions)) *LogOptions {
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return opts
}

// logFields converts ErrorContext to slog-compatible fields with options
func (ec *ErrorContext) LogFields(optsRaw ...*LogOptions) []any {
	if ec == nil {
		return nil
	}

	opts := DefaultLogOptions()
	if len(optsRaw) > 0 {
		opts = optsRaw[0]
	}

	fields := make([]any, 0, len(ec.fields)+12)

	// Add user fields
	if opts.IncludeUserFields {
		for i := 0; i < len(ec.fields); i += 2 {
			if i+1 >= len(ec.fields) {
				break
			}
			key := valueToString(ec.fields[i])
			if utf8.RuneCountInString(key) > maxFieldKeyLength {
				runes := []rune(key)
				key = string(runes[:maxFieldKeyLength])
			}
			value := valueToString(ec.fields[i+1])
			if utf8.RuneCountInString(value) > maxFieldValueLength {
				runes := []rune(value)
				value = string(runes[:maxFieldValueLength])
			}
			fields = append(fields, key, value)
		}
	}

	// Add error metadata
	if opts.IncludeID && ec.ID != "" {
		fields = append(fields, opts.FieldNamePrefix+"id", ec.ID)
	}
	if opts.IncludeCategory && ec.Category != "" {
		fields = append(fields, opts.FieldNamePrefix+"category", ec.Category)
	}
	if opts.IncludeSeverity && ec.Severity != "" {
		fields = append(fields, opts.FieldNamePrefix+"severity", ec.Severity)
	}
	if opts.IncludeTracing {
		if ec.TraceID != "" {
			fields = append(fields, "trace_id", ec.TraceID)
		}
		if ec.SpanID != "" {
			fields = append(fields, "span_id", ec.SpanID)
		}
		if ec.ParentSpanID != "" {
			fields = append(fields, "parent_span_id", ec.ParentSpanID)
		}
	}
	if opts.IncludeRetryable && ec.Retryable {
		fields = append(fields, opts.FieldNamePrefix+"retryable", ec.Retryable)
	}

	if ec.Context != nil && len(opts.ContextFields) > 0 {
		for _, field := range opts.ContextFields {
			value := ec.Context.Value(field)
			if value != nil {
				fields = append(fields, opts.FieldNamePrefix+field, value)
			}
		}
	}

	// Add timing information
	if opts.IncludeCreatedTime && !ec.Created.IsZero() {
		fields = append(fields, opts.FieldNamePrefix+"created_at", ec.Created)
	}

	// Add function context
	if opts.IncludeFunction && ec.Function != "" {
		fields = append(fields, opts.FieldNamePrefix+"function", ec.Function)
	}
	if opts.IncludePackage && ec.Package != "" {
		fields = append(fields, opts.FieldNamePrefix+"package", ec.Package)
	}
	if opts.IncludeFile && ec.File != "" {
		fields = append(fields, opts.FieldNamePrefix+"file", ec.File)
	}
	if opts.IncludeLine && ec.Line > 0 {
		fields = append(fields, opts.FieldNamePrefix+"line", ec.Line)
	}

	// Add stack trace if requested
	if opts.IncludeStack {
		stack := ec.getStackTrace(opts)
		if stack != nil {
			fields = append(fields, opts.FieldNamePrefix+"stack", stack)
		}
	}

	return fields
}

// logFieldsMap converts ErrorContext to logrus-compatible fields with options
func (ec *ErrorContext) LogFieldsMap(optsRaw ...*LogOptions) map[string]any {
	if ec == nil {
		return nil
	}

	opts := DefaultLogOptions()
	if len(optsRaw) > 0 {
		opts = optsRaw[0]
	}

	fields := make(map[string]any, len(ec.Fields)+12)

	// Add user fields
	if opts.IncludeUserFields {
		for i := 0; i < len(ec.fields); i += 2 {
			if i+1 >= len(ec.fields) {
				break
			}
			key := valueToString(ec.fields[i])
			if utf8.RuneCountInString(key) > maxFieldKeyLength {
				runes := []rune(key)
				key = string(runes[:maxFieldKeyLength])
			}
			value := valueToString(ec.fields[i+1])
			if utf8.RuneCountInString(value) > maxFieldValueLength {
				runes := []rune(value)
				value = string(runes[:maxFieldValueLength])
			}
			fields[key] = value
		}
	}

	// Add error metadata
	if opts.IncludeID && ec.ID != "" {
		fields[opts.FieldNamePrefix+"id"] = ec.ID
	}
	if opts.IncludeCategory && ec.Category != "" {
		fields[opts.FieldNamePrefix+"category"] = ec.Category
	}
	if opts.IncludeSeverity && ec.Severity != "" {
		fields[opts.FieldNamePrefix+"severity"] = ec.Severity
	}

	if opts.IncludeTracing {
		if ec.TraceID != "" {
			fields["trace_id"] = ec.TraceID
		}
		if ec.SpanID != "" {
			fields["span_id"] = ec.SpanID
		}
		if ec.ParentSpanID != "" {
			fields["parent_span_id"] = ec.ParentSpanID
		}
	}
	if opts.IncludeRetryable && ec.Retryable {
		fields[opts.FieldNamePrefix+"retryable"] = ec.Retryable
	}

	if ec.Context != nil && len(opts.ContextFields) > 0 {
		for _, field := range opts.ContextFields {
			value := ec.Context.Value(field)
			if value != nil {
				fields[opts.FieldNamePrefix+field] = value
			}
		}
	}

	// Add timing information
	if opts.IncludeCreatedTime && !ec.Created.IsZero() {
		fields[opts.FieldNamePrefix+"created"] = ec.Created
	}

	// Add function context
	if opts.IncludeFunction && ec.Function != "" {
		fields[opts.FieldNamePrefix+"function"] = ec.Function
	}
	if opts.IncludePackage && ec.Package != "" {
		fields[opts.FieldNamePrefix+"package"] = ec.Package
	}
	if opts.IncludeFile && ec.File != "" {
		fields[opts.FieldNamePrefix+"file"] = ec.File
	}
	if opts.IncludeLine && ec.Line > 0 {
		fields[opts.FieldNamePrefix+"line"] = ec.Line
	}

	// Add stack trace if requested
	if opts.IncludeStack {
		stack := ec.getStackTrace(opts)
		if stack != nil {
			fields[opts.FieldNamePrefix+"stack"] = stack
		}
	}

	return fields
}

// getStackTrace returns the stack trace in the requested format
func (ec *ErrorContext) getStackTrace(opts *LogOptions) any {
	if ec == nil || len(ec.Stack) == 0 {
		return nil
	}

	switch opts.StackFormat {
	case StackFormatJSON:
		// Return JSON representation of stack frames
		return ec.Stack.ToJSON()
	case StackFormatFull:
		// Return full stack trace
		return ec.Stack.FormatFull()
	case StackFormatList:
		// Return list of call chain
		return ec.Stack.GetCallChain()
	case StackFormatString:
		fallthrough
	default:
		// Return string representation of stack frames
		return ec.Stack.String()
	}
}

// extractFullMessageWithoutFields builds the complete error message chain without field values
func extractFullMessageWithoutFields(err Error) string {
	switch e := err.GetBase().(type) {
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
