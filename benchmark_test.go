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

func Benchmark_New_Light_Empty(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.NewLight("connection failed")
	}
}

func Benchmark_New_Light_WithFields(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.NewLight("connection failed", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_New_WithFields(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_Newf_WithFields(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Newf("connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
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

func Benchmark_Wrap_Light_Empty(b *testing.B) {
	baseErr := erro.NewLight("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.WrapLight(baseErr, "connection failed")
	}
}

func Benchmark_Wrap_Light_WithFields(b *testing.B) {
	baseErr := erro.NewLight("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.WrapLight(baseErr, "connection failed", "host", "localhost", "port", 5432)
	}
}

func Benchmark_Wrap_WithFields_NoStack(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func Benchmark_Wrap_WithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed", "host", "localhost", "port", 5432)
	}
}

func Benchmark_Wrapf_WithFields(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrapf(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

// Error

func Benchmark_New_ErrorString_Empty(b *testing.B) {
	baseErr := erro.New("base error message")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func Benchmark_New_ErrorString_WithFields(b *testing.B) {
	baseErr := erro.New("base error message", "foo", 123, "bar", true)
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

func Benchmark_NewWrapper_NoStack_Optimized(b *testing.B) {
	baseErr := erro.New("context build")
	fields := []any{"key3", "val3", "key4", 43}
	id := "ID_123"
	var span erro.Span
	var metrics erro.Metrics
	var dispatcher erro.Dispatcher
	ctx := context.Background()
	cfg := erro.DevelopmentStackTraceConfig()
	formatter := erro.FormatErrorWithFields
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.NewWrapper(baseErr, "context build", fields...).
			WithCategory(erro.CategoryDatabase).
			WithClass(erro.ClassValidation).
			WithSeverity(erro.SeverityHigh).
			WithID(id).
			WithRetryable(true).
			WithSpan(span).
			WithMetrics(metrics).
			WithEvent(ctx, dispatcher).
			WithStackTraceConfig(cfg).
			WithFormatter(formatter).
			Build()
	}
}

func Benchmark_NewWrapper_WithStack_NotOptimized(b *testing.B) {
	baseErr := errors.New("context build")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.NewWrapper(baseErr, "context build", "key1", "val1", "key2", 42).
			WithCategory("test").
			WithClass("test").
			WithSeverity("test").
			WithRetryable(true).
			WithSpan(nil).
			WithStackTraceConfig(erro.DevelopmentStackTraceConfig()).
			WithFormatter(erro.FormatErrorWithFields).
			WithEvent(context.Background(), nil).
			WithMetrics(nil).
			WithStack().
			Build()
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

func Benchmark_LogFields(b *testing.B) {
	err := erro.NewError("context build", "key1", "val1", "key2", 42).
		WithCategory("test").
		WithClass("test").
		WithSeverity("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key3", "val3", "key4", 43).
		WithStack().
		Build()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		erro.LogFields(err)
	}
}

func Benchmark_LogFieldsMap(b *testing.B) {
	err := erro.NewError("context build", "key1", "val1", "key2", 42).
		WithCategory("test").
		WithClass("test").
		WithSeverity("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key3", "val3", "key4", 43).
		WithStack().
		Build()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.LogFieldsMap(err)
	}
}

func Benchmark_LogError(b *testing.B) {
	err := erro.NewError("context build", "key1", "val1", "key2", 42).
		WithCategory("test").
		WithClass("test").
		WithSeverity("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key3", "val3", "key4", 43).
		WithStack().
		Build()

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
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.NewTemplate("key1", "value1", "key2", 123).
			WithClass(erro.ClassValidation).
			WithCategory(erro.CategoryUserInput).
			WithSeverity(erro.SeverityHigh).
			WithRetryable(true).
			WithID("TEMPLATE_ID").
			WithStack().
			WithEvent(ctx, nil).
			WithSpan(nil).
			WithMetrics(nil).
			WithMessageTemplate("template error: %s")
	}
}

func Benchmark_New_Error_FromTemplate(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithStackTraceConfig(erro.DevelopmentStackTraceConfig()).
		WithFormatter(erro.FormatErrorWithFields).
		WithStack().
		WithEvent(context.Background(), nil).
		WithMetrics(nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong")
	}
}

func Benchmark_New_Error_FromTemplate_WithMessageAndFields(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithMessageTemplate("template error: %s")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong", "key1", "value1", "key2", 123)
	}
}

func Benchmark_Wrap_Error_FromTemplate_Full_WithStack(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithStack().
		WithSpan(nil).
		WithMetrics(nil).
		WithEvent(context.Background(), nil).
		WithMessageTemplate("template error: %s")

	baseErr := errors.New("base error")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}

// HTTPCode benchmarks

func Benchmark_HTTPCode_WithClass(b *testing.B) {
	err := erro.NewError("not found").WithClass(erro.ClassNotFound).Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.HTTPCode(err)
	}
}

func Benchmark_HTTPCode_StandardError(b *testing.B) {
	err := errors.New("not found")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.HTTPCode(err)
	}
}

func Benchmark_HTTPCode_UnknownClassCategory(b *testing.B) {
	err := erro.NewError("some error").WithClass("").WithCategory("").Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = erro.HTTPCode(err)
	}
}
