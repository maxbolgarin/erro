package erro_test

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

// Benchmark error creation (hot path - should be very fast now)
func BenchmarkErrorsNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.New("connection failed")
	}
}

func BenchmarkErroNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed")
	}
}

func BenchmarkErroNewWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "host", "localhost", "port", 5432)
	}
}

// Benchmark formatted error creation
func BenchmarkFmtErrorf(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("connection failed to %s:%d", "localhost", 5432)
	}
}

func BenchmarkErroErrorf(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Errorf("connection failed to %s:%d", "localhost", 5432)
	}
}

func BenchmarkErroErrorfWithFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Errorf("connection failed to %s:%d", "localhost", 5432, "timeout", "30s", "retries", 3)
	}
}

// Benchmark error wrapping (should be much faster now - only single PC capture)
func BenchmarkFmtErrorfWrap(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("failed to connect to database: %w", baseErr)
	}
}

func BenchmarkErroWrap(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "failed to connect to database")
	}
}

func BenchmarkErroWrapWithFields(b *testing.B) {
	baseErr := errors.New("connection refused")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "failed to connect to database", "host", "localhost", "port", 5432)
	}
}

func BenchmarkErroWrapErroError(b *testing.B) {
	baseErr := erro.New("connection refused", "host", "localhost")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.Wrap(baseErr, "failed to connect to database", "operation", "save_user")
	}
}

// Benchmark deep error wrapping (should show big improvements - no duplicate stack frames)
func BenchmarkDeepWrappingStdLib(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err1 := errors.New("network timeout")
		err2 := fmt.Errorf("connection failed: %w", err1)
		err3 := fmt.Errorf("database operation failed: %w", err2)
		err4 := fmt.Errorf("user save failed: %w", err3)
		_ = fmt.Errorf("API request failed: %w", err4)
	}
}

func BenchmarkDeepWrappingErro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err1 := erro.New("network timeout", "host", "db.example.com")
		err2 := erro.Wrap(err1, "connection failed", "port", 5432)
		err3 := erro.Wrap(err2, "database operation failed", "table", "users")
		err4 := erro.Wrap(err3, "user save failed", "user_id", "123")
		_ = erro.Wrap(err4, "API request failed", "endpoint", "/api/users")
	}
}

// Benchmark error message building (should be unchanged)
func BenchmarkErrorsNewError(b *testing.B) {
	err := errors.New("connection failed")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErroNewError(b *testing.B) {
	err := erro.New("connection failed")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErroNewWithFieldsError(b *testing.B) {
	err := erro.New("connection failed", "host", "localhost", "port", 5432, "timeout", "30s")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// Benchmark stack access (lazy evaluation - only pays cost when used)
func BenchmarkErroGetStack(b *testing.B) {
	err := erro.New("connection failed", "host", "localhost")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = err.Stack() // This should be slower (lazy evaluation)
	}
}

func BenchmarkErroWrappedGetStack(b *testing.B) {
	baseErr := erro.New("connection refused", "host", "localhost")
	wrappedErr := erro.Wrap(baseErr, "operation failed", "table", "users")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wrappedErr.Stack() // Should resolve wrap point + base stack
	}
}

// Benchmark context extraction (rare operations - acceptable to be slower)
func BenchmarkErroExtractContext(b *testing.B) {
	err := erro.New("connection failed", "host", "localhost").
		ID("DB_001").
		Category("infrastructure").
		Severity("high")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.ExtractContext(err) // Should be slower (lazy evaluation)
	}
}

func BenchmarkErroLogFieldsMap(b *testing.B) {
	err := erro.New("connection failed", "host", "localhost").
		ID("DB_001").
		Category("infrastructure").
		Severity("high")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.LogFieldsMap(err) // Should be slower (lazy evaluation)
	}
}

// Benchmark realistic usage patterns
func BenchmarkRealisticStdLibUsage(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		baseErr := errors.New("connection refused")
		wrappedErr := fmt.Errorf("failed to connect to database: %w", baseErr)
		finalErr := fmt.Errorf("user operation failed: %w", wrappedErr)
		_ = finalErr.Error()
	}
}

func BenchmarkRealisticErroUsage(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		baseErr := erro.New("connection refused", "host", "db.example.com", "port", 5432).
			ID("CONN_001").
			Category("infrastructure")
		wrappedErr := erro.Wrap(baseErr, "failed to connect to database", "timeout", "30s")
		finalErr := erro.Wrap(wrappedErr, "user operation failed", "user_id", "123", "operation", "save")
		_ = finalErr.Error()
	}
}

// Benchmark memory usage with different field counts
func BenchmarkErroNewNoFields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("simple error")
	}
}

func BenchmarkErroNew2Fields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("error", "key1", "value1")
	}
}

func BenchmarkErroNew8Fields(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("error", "key1", "value1", "key2", "value2", "key3", "value3", "key4", "value4")
	}
}

// Benchmark chaining performance
func BenchmarkErroNewWithChaining(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = erro.New("connection failed", "host", "localhost").
			ID("DB_001").
			Category("infrastructure").
			Severity("high").
			Retryable(true)
	}
}

// Benchmark the optimization: wrap points vs full stacks
func BenchmarkMultipleWrapsOld(b *testing.B) {
	// Simulate old behavior with full stack capture on each wrap
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err1 := erro.New("base error")
		err2 := erro.Wrap(err1, "wrap 1")
		err3 := erro.Wrap(err2, "wrap 2")
		err4 := erro.Wrap(err3, "wrap 3")
		_ = erro.Wrap(err4, "wrap 4")
	}
}

func BenchmarkMultipleWrapsNew(b *testing.B) {
	// Our optimized approach with single PC capture per wrap
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err1 := erro.New("base error")
		err2 := erro.Wrap(err1, "wrap 1")
		err3 := erro.Wrap(err2, "wrap 2")
		err4 := erro.Wrap(err3, "wrap 3")
		_ = erro.Wrap(err4, "wrap 4")
	}
}

func BenchmarkUnixNano(b *testing.B) {
	a := "1"
	c := "3"

	t := time.Now()
	for i := 0; i < b.N; i++ {
		timestamp := strconv.FormatInt(t.UnixNano(), 10)
		_ = a + c + timestamp[len(timestamp)-4:]
	}
}

func BenchmarkUnix(b *testing.B) {
	a := "1"
	c := "3"

	for i := 0; i < b.N; i++ {
		_ = a + c + strconv.Itoa(rand.Intn(10000))
	}
}
