package erro

// ExtractError ensures that an error can be treated as an [Error].
//
// If the given error is already an [Error], it is returned as is.
// If it's a standard error, it's wrapped in a new [Error] so that
// it can be used with the features of this package.
// If the error is nil, it returns nil.
func ExtractError(err error) Error {
	if err == nil {
		return nil
	}
	if erroErr, ok := err.(Error); ok {
		return erroErr
	}
	var errError Error
	if As(err, &errError) {
		return errError
	}
	return newWrapError(err, "")
}

// LogFields returns a slice of alternating key-value pairs for structured
// logging, extracted from the given error.
//
// This is useful for integrating with logging libraries like `slog` that
// accept key-value pairs.
//
// Example:
//
//	slog.Error("something went wrong", erro.LogFields(err)...)
func LogFields(err error, optFuncs ...LogOption) []any {
	ctx := ExtractError(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions
	if len(optFuncs) > 0 {
		opts = (&LogOptions{}).ApplyOptions(optFuncs...)
	}
	return getLogFields(ctx, opts)
}

// LogFieldsWithOptions returns a slice of alternating key-value pairs for structured
// logging, extracted from the given error.
//
// This is useful for integrating with logging libraries like `slog` that
// accept key-value pairs.
//
// Example:
//
//	slog.Error("something went wrong", erro.LogFieldsWithOptions(err, erro.LogOptions{
//	    IncludeUserFields: true,
//	})...)
func LogFieldsWithOptions(err error, opts LogOptions) []any {
	ctx := ExtractError(err)
	if ctx == nil {
		return nil
	}
	return getLogFields(ctx, opts)
}

// LogFieldsMap returns a map of field key-value pairs for structured logging,
// extracted from the given error.
//
// This is useful for integrating with logging libraries like `logrus` that
// accept a map of fields.
//
// Example:
//
//	logrus.WithFields(erro.LogFieldsMap(err)).Error("something went wrong")
func LogFieldsMap(err error, optFuncs ...LogOption) map[string]any {
	ctx := ExtractError(err)
	if ctx == nil {
		return nil
	}
	opts := DefaultLogOptions
	if len(optFuncs) > 0 {
		opts = (&LogOptions{}).ApplyOptions(optFuncs...)
	}
	return getLogFieldsMap(ctx, opts)
}

// LogFieldsMapWithOptions returns a map of field key-value pairs for structured logging,
// extracted from the given error.
//
// This is useful for integrating with logging libraries like `logrus` that
// accept a map of fields.
//
// Example:
//
//	logrus.WithFields(erro.LogFieldsMapWithOptions(err, erro.LogOptions{
//	    IncludeUserFields: true,
//	})).Error("something went wrong")
func LogFieldsMapWithOptions(err error, opts LogOptions) map[string]any {
	ctx := ExtractError(err)
	if ctx == nil {
		return nil
	}
	return getLogFieldsMap(ctx, opts)
}

// LogError executes a callback with the error message and structured fields,
// allowing for integration with any logging library.
//
// If the error is not an [Error], it logs the error message directly.
//
// Example:
//
//	erro.LogError(err, func(message string, fields ...any) {
//	    myLogger.Error(message, fields...)
//	})
func LogError(err error, logFunc func(message string, fields ...any), optFuncs ...LogOption) {
	if err == nil || logFunc == nil {
		return
	}

	errError, ok := err.(Error)
	if !ok {
		if !As(err, &errError) {
			logFunc(err.Error())
			return
		}
	}

	opts := DefaultLogOptions
	if len(optFuncs) > 0 {
		opts = (&LogOptions{}).ApplyOptions(optFuncs...)
	}
	logFunc(errError.Message(), getLogFields(errError, opts)...)
}

// LogErrorWithOptions executes a callback with the error message and structured fields,
// allowing for integration with any logging library.
//
// If the error is not an [Error], it logs the error message directly.
//
// Example:
//
//	erro.LogErrorWithOptions(err, func(message string, fields ...any) {
//	    myLogger.Error(message, fields...)
//	}, erro.LogOptions{
//	    IncludeUserFields: true,
//	})
func LogErrorWithOptions(err error, logFunc func(message string, fields ...any), opts LogOptions) {
	if err == nil || logFunc == nil {
		return
	}

	errError, ok := err.(Error)
	if !ok {
		if !As(err, &errError) {
			logFunc(err.Error())
			return
		}
	}

	logFunc(errError.Message(), getLogFields(errError, opts)...)
}

// ErrorToJSON converts an error to a serializable [ErrorSchema] struct.
//
// This is useful for sending error details over the network or storing them
// in a structured format. Sensitive fields are redacted.
func ErrorToJSON(err Error) ErrorSchema {
	schema := ErrorSchema{
		ID:        err.ID(),
		Class:     err.Class(),
		Category:  err.Category(),
		Severity:  err.Severity(),
		Created:   err.Created(),
		Message:   err.Message(),
		Retryable: err.IsRetryable(),
	}

	// Redact sensitive fields before serialization.
	allFields := err.AllFields()
	if len(allFields) > 0 {
		redactedFields := make([]any, len(allFields))
		copy(redactedFields, allFields)
		for i := 1; i < len(redactedFields); i += 2 {
			if _, ok := redactedFields[i].(RedactedValue); ok {
				redactedFields[i] = RedactedPlaceholder
			}
		}
		schema.Fields = redactedFields
	}

	span := err.Span()
	if span != nil {
		schema.TraceID = span.TraceID()
		schema.SpanID = span.SpanID()
		schema.ParentSpanID = span.ParentSpanID()
	}

	stack := err.Stack()
	if len(stack) > 0 {
		schema.StackTrace = make([]StackContext, len(stack))
		for i, frame := range stack {
			schema.StackTrace[i] = frame.GetContext()
		}
	}

	return schema
}

// LogOption is a function that configures logging options.
type LogOption func(*LogOptions)

// LogOptions controls which fields are included in logging output.
type LogOptions struct {
	// IncludeUserFields determines whether to include user-defined fields.
	IncludeUserFields bool

	// IncludeID determines whether to include the error ID.
	IncludeID bool
	// IncludeCategory determines whether to include the error category.
	IncludeCategory bool
	// IncludeSeverity determines whether to include the error severity.
	IncludeSeverity bool
	// IncludeRetryable determines whether to include the retryable flag.
	IncludeRetryable bool
	// IncludeTracing determines whether to include tracing information (TraceID, SpanID).
	IncludeTracing bool

	// IncludeCreatedTime determines whether to include the error creation timestamp.
	IncludeCreatedTime bool

	// IncludeFunction determines whether to include the function name from the stack trace.
	IncludeFunction bool
	// IncludePackage determines whether to include the package name from the stack trace.
	IncludePackage bool
	// IncludeFile determines whether to include the file name from the stack trace.
	IncludeFile bool
	// IncludeLine determines whether to include the line number from the stack trace.
	IncludeLine bool
	// IncludeStack determines whether to include the full stack trace.
	IncludeStack bool

	// StackFormat defines how stack traces should be formatted.
	StackFormat StackFormat

	// FieldNamePrefix is a prefix added to all field names. Default is "error_".
	FieldNamePrefix string
}

// StackFormat defines how stack traces should be formatted in logs.
type StackFormat int

const (
	// StackFormatString formats the stack trace as a single string.
	StackFormatString StackFormat = iota
	// StackFormatList formats the stack trace as a list of function calls.
	StackFormatList
	// StackFormatFull formats the stack trace with detailed information.
	StackFormatFull
	// StackFormatJSON formats the stack trace as a JSON object.
	StackFormatJSON
)

var (
	// DefaultLogOptions includes a balanced set of fields for typical logging.
	DefaultLogOptions = LogOptions{
		IncludeUserFields:  true,
		IncludeID:          false, // Useless in logs for most cases
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

	// VerboseLogOpts is a slice of [LogOption] functions for verbose logging.
	VerboseLogOpts = []LogOption{
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

	// MinimalLogOpts is a slice of [LogOption] functions for minimal logging.
	MinimalLogOpts = []LogOption{
		WithUserFields(true),
		WithSeverity(true),
	}
)

// MergeLogOpts merges multiple slices of [LogOption] functions into one.
func MergeLogOpts(opts []LogOption, addsOpts ...LogOption) []LogOption {
	return append(opts, addsOpts...)
}

// WithUserFields returns a [LogOption] to enable or disable user-defined fields.
func WithUserFields(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeUserFields = true
		if len(include) > 0 {
			opts.IncludeUserFields = include[0]
		}
	}
}

// WithID returns a [LogOption] to enable or disable the error ID field.
func WithID(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeID = true
		if len(include) > 0 {
			opts.IncludeID = include[0]
		}
	}
}

// WithCategory returns a [LogOption] to enable or disable the error category field.
func WithCategory(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeCategory = true
		if len(include) > 0 {
			opts.IncludeCategory = include[0]
		}
	}
}

// WithSeverity returns a [LogOption] to enable or disable the error severity field.
func WithSeverity(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeSeverity = true
		if len(include) > 0 {
			opts.IncludeSeverity = include[0]
		}
	}
}

// WithTracing returns a [LogOption] to enable or disable tracing fields.
func WithTracing(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeTracing = true
		if len(include) > 0 {
			opts.IncludeTracing = include[0]
		}
	}
}

// WithRetryable returns a [LogOption] to enable or disable the retryable flag field.
func WithRetryable(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeRetryable = true
		if len(include) > 0 {
			opts.IncludeRetryable = include[0]
		}
	}
}

// WithCreatedTime returns a [LogOption] to enable or disable the creation timestamp field.
func WithCreatedTime(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeCreatedTime = true
		if len(include) > 0 {
			opts.IncludeCreatedTime = include[0]
		}
	}
}

// WithFunction returns a [LogOption] to enable or disable the function name field.
func WithFunction(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeFunction = true
		if len(include) > 0 {
			opts.IncludeFunction = include[0]
		}
	}
}

// WithPackage returns a [LogOption] to enable or disable the package name field.
func WithPackage(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludePackage = true
		if len(include) > 0 {
			opts.IncludePackage = include[0]
		}
	}
}

// WithFile returns a [LogOption] to enable or disable the file name field.
func WithFile(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeFile = true
		if len(include) > 0 {
			opts.IncludeFile = include[0]
		}
	}
}

// WithLine returns a [LogOption] to enable or disable the line number field.
func WithLine(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeLine = true
		if len(include) > 0 {
			opts.IncludeLine = include[0]
		}
	}
}

// WithStack returns a [LogOption] to enable or disable the full stack trace field.
func WithStack(include ...bool) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeStack = true
		if len(include) > 0 {
			opts.IncludeStack = include[0]
		}
	}
}

// WithStackFormat returns a [LogOption] to set the stack trace format.
func WithStackFormat(format StackFormat) LogOption {
	return func(opts *LogOptions) {
		opts.IncludeStack = true
		opts.StackFormat = format
	}
}

// WithFieldNamePrefix returns a [LogOption] to set the field name prefix.
func WithFieldNamePrefix(prefix string) LogOption {
	return func(opts *LogOptions) {
		opts.FieldNamePrefix = prefix
	}
}

// ApplyOptions applies a set of option functions to [LogOptions].
func (opts *LogOptions) ApplyOptions(optFuncs ...LogOption) LogOptions {
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	return *opts
}

// getLogFieldsMap converts [Error] to slog-compatible fields with options
func getLogFieldsMap(ec Error, optsRaw ...LogOptions) map[string]any {
	fields := getLogFields(ec, optsRaw...)

	fieldsMap := make(map[string]any, len(fields))
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}
		key, ok := fields[i].(string)
		if !ok {
			key = valueToString(fields[i])
		}
		fieldsMap[truncateString(key, MaxKeyLength)] = fields[i+1]
	}
	return fieldsMap
}

// getLogFields converts [Error] to logrus-compatible fields with options
func getLogFields(ec Error, optsRaw ...LogOptions) []any {
	if ec == nil {
		return nil
	}

	opts := DefaultLogOptions
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
		errorFields       = ec.AllFields()
	)

	span := ec.Span()
	if span != nil {
		errorTraceID = span.TraceID()
		errorSpanID = span.SpanID()
		errorParentSpanID = span.ParentSpanID()
	}

	fields := make([]any, 0, len(errorFields)+30)

	// Add user fields
	if opts.IncludeUserFields && len(errorFields) > 0 {
		for i := 0; i < len(errorFields); i++ {
			if _, ok := errorFields[i].(RedactedValue); ok {
				fields = append(fields, RedactedPlaceholder)
			} else {
				fields = append(fields, errorFields[i])
			}
		}
	}

	// Add error metadata
	if opts.IncludeID && errorID != "" {
		fields = append(fields, opts.FieldNamePrefix+"id", errorID)
	}
	if opts.IncludeCategory && errorCategory != "" {
		fields = append(fields, opts.FieldNamePrefix+"category", errorCategory)
	}
	if opts.IncludeSeverity && errorSeverity != "" {
		fields = append(fields, opts.FieldNamePrefix+"severity", errorSeverity)
	}
	if opts.IncludeTracing {
		if errorTraceID != "" {
			fields = append(fields, "trace_id", errorTraceID)
		}
		if errorSpanID != "" {
			fields = append(fields, "span_id", errorSpanID)
		}
		if errorParentSpanID != "" {
			fields = append(fields, "parent_span_id", errorParentSpanID)
		}
	}
	if opts.IncludeRetryable && errorRetryable {
		fields = append(fields, opts.FieldNamePrefix+"retryable", errorRetryable)
	}

	// Add timing information
	if opts.IncludeCreatedTime && !errorCreated.IsZero() {
		fields = append(fields, opts.FieldNamePrefix+"created", errorCreated)
	}

	if opts.IncludeFunction || opts.IncludePackage || opts.IncludeFile || opts.IncludeLine {
		topFrame := errorStack.TopUserFrame()
		if topFrame == nil && len(errorStack) > 0 {
			topFrame = &errorStack[0]
		}
		// Add function context
		if opts.IncludeFunction && topFrame != nil && topFrame.Name != "" {
			fields = append(fields, opts.FieldNamePrefix+"function", topFrame.Name)
		}
		if opts.IncludePackage && topFrame != nil && topFrame.Package != "" {
			fields = append(fields, opts.FieldNamePrefix+"package", topFrame.Package)
		}
		if opts.IncludeFile && topFrame != nil && topFrame.File != "" {
			fields = append(fields, opts.FieldNamePrefix+"file", topFrame.File)
		}
		if opts.IncludeLine && topFrame != nil && topFrame.Line > 0 {
			fields = append(fields, opts.FieldNamePrefix+"line", topFrame.Line)
		}
	}

	// Add stack trace if requested
	if opts.IncludeStack {
		stack := getStackTrace(errorStack, opts)
		if stack != nil {
			fields = append(fields, opts.FieldNamePrefix+"stack", stack)
		}
	}

	return fields
}

// getStackTrace returns the stack trace in the requested format
func getStackTrace(stack Stack, opts LogOptions) any {
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
