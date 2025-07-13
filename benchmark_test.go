package erro_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maxbolgarin/erro"
)

// New

func BenchmarkStdErrorsNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.New("connection failed")
	}
}

func BenchmarkLightNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.NewLight("connection failed")
	}
}

func BenchmarkLightNewWithFields(b *testing.B) {
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
		_ = erro.New("connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

// Wrapping

func BenchmarkStdErrorf(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("connection failed to address=%s:%d: %w", "localhost", 5432, baseErr)
	}
}

func BenchmarkWrapLight(b *testing.B) {
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

func BenchmarkWrap(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed")
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

func BenchmarkWrapfWithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrapf(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

func BenchmarkWrapWithoutStackGetting(b *testing.B) {
	baseErr := erro.New("connection refused")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "connection failed address=%s:%d", "localhost", 5432, "key1", "value1", "key2", 123, "key3", 1.23)
	}
}

// Error

func BenchmarkGeneralErrorString(b *testing.B) {
	baseErr := erro.New("base error message")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func BenchmarkGeneralErrorStringWithFields(b *testing.B) {
	baseErr := erro.New("base error message", "foo", 123, "bar", true)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = baseErr.Error()
	}
}

func BenchmarkErrorStringWrapErrorDeep(b *testing.B) {
	err := erro.New("root error", "key1", "val1", "key2", 42)
	for i := 0; i < 10; i++ {
		err = erro.Wrap(err, fmt.Sprintf("wrap level %d", i), "key3", 3.14, "key4", true)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// Log fields

func BenchmarkErrorContextBuilding(b *testing.B) {
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

func BenchmarkErrorLightContextBuilding(b *testing.B) {
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

func BenchmarkErrorContextRetrieving(b *testing.B) {
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

func BenchmarkErrorFieldsGetting(b *testing.B) {
	err := erro.New("full error", "key1", "val1", "key2", 42, "key3", 3.14, "key4", true).
		WithClass("test").
		WithCategory("test").
		WithSeverity("test").
		WithID("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key5", "val5", "key6", 44)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		erro.LogFields(err)
	}
}

func BenchmarkErrorFieldsGettingWithWrap(b *testing.B) {
	baseErr := erro.New("base error", "key1", "val1", "key2", 42)
	wrappedErr := erro.Wrap(baseErr, "wrapped error", "key3", 3.14, "key4", true).
		WithClass("test").
		WithCategory("test").
		WithSeverity("test").
		WithID("test").
		WithRetryable(true).
		WithSpan(nil).
		WithFields("key5", "val5", "key6", 44)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		erro.LogFields(wrappedErr)
	}
}

// Template benchmarks

func BenchmarkTemplateCreationWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.NewTemplate("key1", "value1", "key2", 123).
			WithClass(erro.ClassValidation).
			WithCategory(erro.CategoryUserInput).
			WithSeverity(erro.SeverityHigh).
			WithRetryable(true).
			WithID("TEMPLATE_ID").
			WithMessageTemplate("template error: %s")
	}
}

func BenchmarkTemplateNewError(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong")
	}
}

func BenchmarkTemplateNewErrorWithMessageAndFields(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithMessageTemplate("template error: %s")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tmpl.New("something went wrong", "key1", "value1", "key2", 123)
	}
}

func BenchmarkTemplateWrapErrorNoStack(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithMessageTemplate("template error: %s")
	baseErr := erro.New("base error")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template")
	}
}

func BenchmarkTemplateWrapErrorWithFieldsNoStack(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithMessageTemplate("template error: %s")
	baseErr := erro.New("base error")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}

func BenchmarkTemplateWrapErrorWithFieldsWithStack(b *testing.B) {
	tmpl := erro.NewTemplate().
		WithClass(erro.ClassValidation).
		WithCategory(erro.CategoryUserInput).
		WithSeverity(erro.SeverityHigh).
		WithRetryable(true).
		WithID("TEMPLATE_ID").
		WithMessageTemplate("template error: %s")
	baseErr := errors.New("base error")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Wrap(baseErr, "wrapped by template", "key1", "value1", "key2", 123)
	}
}
