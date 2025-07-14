package erro_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maxbolgarin/erro"
)

// New

func BenchmarkNewSTD(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.New("connection failed")
	}
}

func BenchmarkNewLightEmpty(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.NewLight("connection failed")
	}
}

func BenchmarkNewLightWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.NewLight("connection failed", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func BenchmarkNewWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func BenchmarkNewfWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Newf("connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

// Wrapping

func BenchmarkErrorfSTD(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("connection failed to address=%s:%d: %w", "localhost", 5432, baseErr)
	}
}

func BenchmarkWrapLightEmpty(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.WrapLight(baseErr, "connection failed")
	}
}

func BenchmarkWrapLightWithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.WrapLight(baseErr, "connection failed", "host", "localhost", "port", 5432)
	}
}

func BenchmarkWrapWithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed", "host", "localhost", "port", 5432)
	}
}

func BenchmarkWrapWithFieldsNoStack(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func BenchmarkWrapfWithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrapf(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

// Error

func BenchmarkNewErrorStringEmpty(b *testing.B) {
	baseErr := erro.New("base error message")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func BenchmarkNewErrorStringWithFields(b *testing.B) {
	baseErr := erro.New("base error message", "foo", 123, "bar", true)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func BenchmarkWrapErrorDeep(b *testing.B) {
	err := erro.New("root error", "key1", "val1", "key2", 42)
	for i := 0; i < 10; i++ {
		err = erro.Wrap(err, fmt.Sprintf("wrap level %d", i), "key3", 3.14, "key4", true)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// Context building

func BenchmarkNewErrorWithContext(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		erro.New("context build", "key1", "val1", "key2", 42).
			WithCategory("test").
			WithClass("test").
			WithSeverity("test").
			WithID("test").
			WithRetryable(true).
			WithSpan(nil).
			WithFields("key3", "val3", "key4", 43)
	}
}

func BenchmarkNewLightWithContext(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		erro.NewLight("context build", "key1", "val1", "key2", 42).
			WithCategory("test").
			WithClass("test").
			WithSeverity("test").
			WithID("test").
			WithRetryable(true).
			WithSpan(nil).
			WithFields("key3", "val3", "key4", 43)
	}
}

func BenchmarkNewBuilderNoStackOptimized(b *testing.B) {
	baseErr := erro.New("context build")
	fields := []any{"key3", "val3", "key4", 43}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.NewBuilderWithError(baseErr, "context build", fields...).
			WithCategory(erro.CategoryDatabase).
			WithClass(erro.ClassValidation).
			WithSeverity(erro.SeverityHigh).
			WithID("ID_123").
			WithRetryable(true).
			WithSpan(nil).
			Build()
	}
}

func BenchmarkNewBuilderWithStackNotOptimized(b *testing.B) {
	baseErr := errors.New("context build")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		erro.NewBuilderWithError(baseErr, "context build", "key1", "val1", "key2", 42).
			WithCategory("test").
			WithClass("test").
			WithSeverity("test").
			GenerateID().
			WithRetryable(true).
			WithSpan(nil).
			WithFields("key3", "val3", "key4", 43).
			WithStack().
			Build()
	}
}

func BenchmarkErrorContext(b *testing.B) {
	err := erro.New("context retrieve", "key1", "val1", "key2", 42)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx := err.Context()
		_ = ctx.ID()
		_ = ctx.Class()
		_ = ctx.Category()
		_ = ctx.Message()
		_ = ctx.Fields()
		_ = ctx.Severity()
		_ = ctx.IsRetryable()
		_ = ctx.Span()
		_ = ctx.Created()
		_ = ctx.Stack()
	}
}

// Log Fields

func BenchmarkLogFields(b *testing.B) {
	err := erro.New("full error", "key1", "val1", "key2", 42, "key3", 3.14, "key4", true).
		WithClass("test").
		WithCategory("test").
		WithSeverity("test").
		WithID("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key5", "val5", "key6", 44)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		erro.LogFields(err)
	}
}

func BenchmarkLogFieldsMap(b *testing.B) {
	err := erro.New("full error", "key1", "val1", "key2", 42, "key3", 3.14, "key4", true).
		WithClass("test").
		WithCategory("test").
		WithSeverity("test").
		WithID("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key5", "val5", "key6", 44)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		erro.LogFieldsMap(err)
	}
}

func BenchmarkLogError(b *testing.B) {
	err := erro.New("full error", "key1", "val1", "key2", 42, "key3", 3.14, "key4", true).
		WithClass("test").
		WithCategory("test").
		WithSeverity("test").
		WithID("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key5", "val5", "key6", 44)

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

func BenchmarkNewTemplate(b *testing.B) {
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
			WithGoContext(ctx).
			WithSpan(nil).
			WithMetrics(nil).
			WithDispatcher(nil).
			WithMessageTemplate("template error: %s")
	}
}

func BenchmarkNewErrorFromTemplate(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong")
	}
}

func BenchmarkNewErrorFromTemplateWithMessageAndFields(b *testing.B) {
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

func BenchmarkWrapErrorFromTemplateFullWithStack(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithStack().
		WithGoContext(context.Background()).
		WithSpan(nil).
		WithMetrics(nil).
		WithDispatcher(nil).
		WithMessageTemplate("template error: %s")

	baseErr := errors.New("base error")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}
