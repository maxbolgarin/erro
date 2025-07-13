package erro

func ExtractError(err error) Error {
	if err == nil {
		return nil
	}
	if erroErr, ok := err.(Error); ok {
		return erroErr
	}
	return newLightError(err, "")
}

func ExtractContext(err error) ErrorContext {
	if err == nil {
		return nil
	}
	if erroErr, ok := err.(ErrorContext); ok {
		return erroErr
	}
	return newLightError(err, "")
}

// LogFields returns a slice of alternating key-value pairs for structured loggers
func LogFields(err error, optFuncs ...func(*LogOptions)) []any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions().ApplyOptions(optFuncs...)
	return getLogFields(ctx, opts)
}

// LogFieldsMap returns a map of field key-value pairs for map-based loggers
func LogFieldsMap(err error, optFuncs ...func(*LogOptions)) map[string]any {
	ctx := ExtractContext(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions().ApplyOptions(optFuncs...)
	return getLogFieldsMap(ctx, opts)
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
	logFunc(ctx.Message(), getLogFields(ctx, opts)...)
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
func getLogFields(ec ErrorContext, optsRaw ...*LogOptions) []any {
	fieldsMap := getLogFieldsMap(ec, optsRaw...)

	fields := make([]any, 0, len(fieldsMap)*2)
	for key, value := range fieldsMap {
		fields = append(fields, key, value)
	}
	return fields
}

// logFieldsMap converts ErrorContext to logrus-compatible fields with options
func getLogFieldsMap(ec ErrorContext, optsRaw ...*LogOptions) map[string]any {
	if ec == nil {
		return nil
	}

	opts := DefaultLogOptions()
	if len(optsRaw) > 0 {
		opts = optsRaw[0]
	}

	var (
		errorID           = ec.ID()
		errorCategory     = ec.Category()
		errorSeverity     = ec.Severity()
		errorRetryable    = ec.IsRetryable()
		errorCreated      = ec.Created()
		errorTraceID      = ""
		errorSpanID       = ""
		errorParentSpanID = ""
		errorStack        = ec.Stack()
		errorFields       = ec.Fields()
	)

	span := ec.Span()
	if span != nil {
		errorTraceID = span.TraceID()
		errorSpanID = span.SpanID()
		errorParentSpanID = span.ParentSpanID()
	}

	fields := make(map[string]any, len(errorFields)+14)

	// Add user fields
	if opts.IncludeUserFields {
		for i := 0; i < len(errorFields); i += 2 {
			if i+1 >= len(errorFields) {
				break
			}
			key, ok := errorFields[i].(string)
			if !ok {
				key = valueToString(errorFields[i])
			}
			value, ok := errorFields[i+1].(string)
			if !ok {
				value = valueToString(errorFields[i+1])
			}
			fields[truncateString(key, maxFieldKeyLength)] = truncateString(value, maxFieldValueLength)
		}
	}

	// Add error metadata
	if opts.IncludeID && errorID != "" {
		fields[opts.FieldNamePrefix+"id"] = errorID
	}
	if opts.IncludeCategory && errorCategory != "" {
		fields[opts.FieldNamePrefix+"category"] = errorCategory
	}
	if opts.IncludeSeverity && errorSeverity != "" {
		fields[opts.FieldNamePrefix+"severity"] = errorSeverity
	}
	if opts.IncludeTracing {
		if errorTraceID != "" {
			fields["trace_id"] = errorTraceID
		}
		if errorSpanID != "" {
			fields["span_id"] = errorSpanID
		}
		if errorParentSpanID != "" {
			fields["parent_span_id"] = errorParentSpanID
		}
	}
	if opts.IncludeRetryable && errorRetryable {
		fields[opts.FieldNamePrefix+"retryable"] = errorRetryable
	}

	// Add timing information
	if opts.IncludeCreatedTime && !errorCreated.IsZero() {
		fields[opts.FieldNamePrefix+"created"] = errorCreated
	}

	topFrame := errorStack.TopUserFrame()
	if topFrame == nil {
		topFrame = &errorStack[0]
	}

	// Add function context
	if opts.IncludeFunction && topFrame.Name != "" {
		fields[opts.FieldNamePrefix+"function"] = topFrame.Name
	}
	if opts.IncludePackage && topFrame.Package != "" {
		fields[opts.FieldNamePrefix+"package"] = topFrame.Package
	}
	if opts.IncludeFile && topFrame.File != "" {
		fields[opts.FieldNamePrefix+"file"] = topFrame.File
	}
	if opts.IncludeLine && topFrame.Line > 0 {
		fields[opts.FieldNamePrefix+"line"] = topFrame.Line
	}

	// Add stack trace if requested
	if opts.IncludeStack {
		stack := getStackTrace(errorStack, opts)
		if stack != nil {
			fields[opts.FieldNamePrefix+"stack"] = stack
		}
	}

	return fields
}

// getStackTrace returns the stack trace in the requested format
func getStackTrace(stack Stack, opts *LogOptions) any {
	if len(stack) == 0 {
		return nil
	}

	switch opts.StackFormat {
	case StackFormatJSON:
		// Return JSON representation of stack frames
		return stack.ToJSON()
	case StackFormatFull:
		// Return full stack trace
		return stack.FormatFull()
	case StackFormatList:
		// Return list of call chain
		return stack.GetCallChain()
	case StackFormatString:
		fallthrough
	default:
		// Return string representation of stack frames
		return stack.String()
	}
}
