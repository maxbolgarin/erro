package erro_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maxbolgarin/erro"
)

// New

func Benchmark_New_STD(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.New("connection failed")
	}
}

func Benchmark_New(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed")
	}
}

func Benchmark_New_WithFields(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "address", "localhost:5432", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_New_WithFieldsAndFormatVerbs(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_NewWithStack(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", erro.StackTrace())
	}
}

func Benchmark_NewWithStack_WithFields(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "address", "localhost:5432", "key1", "value1", "key2", 123, "key3", 1.23, erro.StackTrace())
	}
}

// Wrapping

func Benchmark_Errorf_STD(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("connection failed to address=%s:%d: %w", "localhost", 5432, baseErr)
	}
}

func Benchmark_Wrap(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed")
	}
}

func Benchmark_Wrap_WithFields(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed", "address", "localhost:5432", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_Wrap_WithFieldsAndFormatVerbs(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_WrapWithStack(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed", erro.StackTrace())
	}
}

func Benchmark_WrapWithStack_WithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed", "address", "localhost:5432", "key1", "value1", "key2", 123, "key3", 1.23, erro.StackTrace())
	}
}

// Error

func Benchmark_New_ErrorString(b *testing.B) {
	baseErr := erro.New("base error message")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func Benchmark_New_ErrorString_WithFields(b *testing.B) {
	baseErr := erro.New("base error message address=%s:%d", "localhost", 5432, "foo", 123, "bar", true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func Benchmark_Wrap_Error_Deep(b *testing.B) {
	err := erro.New("root error", "key1", "val1", "key2", 42)
	for i := 0; i < 10; i++ {
		err = erro.Wrap(err, fmt.Sprintf("wrap level %d", i), "key3", 3.14, "key4", true)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// Context building

func Benchmark_New_AllMeta_WithStack(b *testing.B) {
	var span erro.TraceSpan
	var metrics erro.ErrorMetrics
	var dispatcher erro.EventDispatcher
	ctx := context.Background()
	formatter := erro.FormatErrorWithFields
	cfg := erro.StrictStackTraceConfig()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = erro.New("context build address=%s:%d", "localhost", 5432,
			"key1", "val1", "key2", 42,
			"refacted_key", erro.Redact("redacted_value"),
			erro.Fields("key3", "val3", "key4", 43),

			erro.CategoryDatabase,
			erro.ClassValidation,
			erro.SeverityHigh,

			erro.Retryable(),
			erro.RecordSpan(span),
			erro.RecordMetrics(metrics),
			erro.SendEvent(ctx, dispatcher),
			erro.Formatter(formatter),
			erro.StackTrace(cfg),
		)
	}
}

func Benchmark_New_AllMeta_NoStack(b *testing.B) {
	var span erro.TraceSpan
	var metrics erro.ErrorMetrics
	var dispatcher erro.EventDispatcher
	ctx := context.Background()
	formatter := erro.FormatErrorWithFields
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = erro.New("context build address=%s:%d", "localhost", 5432,
			"key1", "val1", "key2", 42,
			"refacted_key", erro.Redact("redacted_value"),
			erro.Fields("key3", "val3", "key4", 43),

			erro.CategoryDatabase,
			erro.ClassValidation,
			erro.SeverityHigh,

			erro.Retryable(),
			erro.RecordSpan(span),
			erro.RecordMetrics(metrics),
			erro.SendEvent(ctx, dispatcher),
			erro.Formatter(formatter),
		)
	}
}

func Benchmark_New_AllMeta_NoStack_Optimized(b *testing.B) {
	fields := []any{}
	id := "ID_123"
	var span erro.TraceSpan
	var metrics erro.ErrorMetrics
	var dispatcher erro.EventDispatcher
	ctx := context.Background()
	formatter := erro.FormatErrorWithFields

	opts := []any{
		"address", "localhost:5432",
		"key1", "val1", "key2", 42,
		"refacted_key", erro.Redact("redacted_value"),
		erro.Fields(fields...),

		erro.CategoryDatabase,
		erro.ClassValidation,
		erro.SeverityHigh,

		erro.ID(id),
		erro.Retryable(),
		erro.RecordSpan(span),
		erro.RecordMetrics(metrics),
		erro.SendEvent(ctx, dispatcher),
		erro.Formatter(formatter),
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = erro.New("context build", opts...)
	}
}

func Benchmark_Error_Context(b *testing.B) {
	err := erro.New("context retrieve", "key1", "val1", "key2", 42)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.ID()
		_ = err.Class()
		_ = err.Category()
		_ = err.Message()
		_ = err.Fields()
		_ = err.Severity()
		_ = err.IsRetryable()
		_ = err.Span()
		_ = err.Created()
		_ = err.Stack()
	}
}

// Log Fields

func Benchmark_LogFields_Default(b *testing.B) {
	err := newErr()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		erro.LogFields(err)
	}
}

func Benchmark_LogFields_Minimal(b *testing.B) {
	err := newErr()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		erro.LogFields(err, erro.MinimalLogOpts...)
	}
}

func Benchmark_LogFields_Verbose(b *testing.B) {
	err := newErr()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		erro.LogFields(err, erro.VerboseLogOpts...)
	}
}

func Benchmark_LogFieldsMap(b *testing.B) {
	err := newErr()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.LogFieldsMap(err)
	}
}

func Benchmark_LogError(b *testing.B) {
	err := newErr()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.LogError(err, func(message string, fields ...any) {
			_ = message
			_ = fields
		})
	}
}

// Template benchmarks

func Benchmark_New_Template(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newTemplateWithStack()
	}
}

func Benchmark_New_FromTemplate(b *testing.B) {
	tmpl := newTemplate()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New()
	}
}

func Benchmark_New_FromTemplate_WithMessageAndFields(b *testing.B) {
	tmpl := newTemplate()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong", "key1", "value1", "key2", 123)
	}
}

func Benchmark_New_FromTemplate_Full(b *testing.B) {
	tmpl := newTemplateWithStack()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong", "key1", "value1", "key2", 123)
	}
}

func Benchmark_Wrap_FromTemplate(b *testing.B) {
	tmpl := newTemplate()

	baseErr := erro.New("base error")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr)
	}
}

func Benchmark_Wrap_FromTemplate_WithMessageAndFields(b *testing.B) {
	tmpl := newTemplate()

	baseErr := erro.New("base error")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}

func Benchmark_Wrap_FromTemplate_Full(b *testing.B) {
	tmpl := newTemplateWithStack()

	baseErr := erro.New("base error")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}

// HTTPCode benchmarks

func Benchmark_HTTPCode_Class(b *testing.B) {
	err := erro.New("some error", erro.ClassValidation, erro.CategoryDatabase)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.HTTPCode(err)
	}
}
func Benchmark_HTTPCode_Category(b *testing.B) {
	err := erro.New("some error", erro.CategoryDatabase)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.HTTPCode(err)
	}
}

// Other benchmarks

func Benchmark_Sprintf(b *testing.B) {
	format := "connection failed address=%s:%d key1=%s key2=%d key3=%f key4=%t"
	arg1 := "localhost"
	arg2 := 5432
	arg3 := "value1"
	arg4 := 123
	arg5 := 3.14
	arg6 := true

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf(format, arg1, arg2, arg3, arg4, arg5, arg6)
	}
}

func Benchmark_ApplyFormatVerbs(b *testing.B) {
	format := "connection failed address=%s:%d key1=%s key2=%d key3=%f key4=%t"
	arg1 := "localhost"
	arg2 := 5432
	arg3 := "value1"
	arg4 := 123
	arg5 := 3.14
	arg6 := true

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = erro.ApplyFormatVerbs(format, arg1, arg2, arg3, arg4, arg5, arg6)
	}
}

func newErr() erro.Error {
	fields := []any{"key3", "val3", "key4", 43}
	id := "ID_123"
	var span erro.TraceSpan
	var metrics erro.ErrorMetrics
	var dispatcher erro.EventDispatcher
	ctx := context.Background()
	formatter := erro.FormatErrorWithFields

	return erro.New("context build",
		erro.CategoryDatabase,
		erro.ClassValidation,
		erro.SeverityHigh,

		erro.ID(id),
		erro.Retryable(),
		erro.Fields(fields...),
		erro.RecordSpan(span),
		erro.RecordMetrics(metrics),
		erro.SendEvent(ctx, dispatcher),
		erro.Formatter(formatter),
	)
}

func newTemplate() *erro.ErrorTemplate {
	return erro.NewTemplate("template error: %s",
		erro.ClassValidation,
		erro.CategoryUserInput,
		erro.SeverityHigh,
		erro.Retryable(),
		erro.ID("TEMPLATE_ID"),
		erro.RecordSpan(nil),
		erro.RecordMetrics(nil),
		erro.SendEvent(context.Background(), nil),
		erro.Formatter(nil),
	)
}

func newTemplateWithStack() *erro.ErrorTemplate {
	return erro.NewTemplate("template error: %s",
		erro.ClassValidation,
		erro.CategoryUserInput,
		erro.SeverityHigh,
		erro.Retryable(),
		erro.ID("TEMPLATE_ID"),
		erro.StackTrace(erro.StrictStackTraceConfig()),
		erro.RecordSpan(nil),
		erro.RecordMetrics(nil),
		erro.SendEvent(context.Background(), nil),
		erro.Formatter(nil),
	)
}
