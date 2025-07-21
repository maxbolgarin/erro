# 🚀 `erro` - Next-Generation Error Handling for Go

[![Go Version][version-img]][doc] [![GoDoc][doc-img]][doc] [![Build][ci-img]][ci] [![Coverage][coverage-img]][coverage] [![GoReport][report-img]][report]


**Move your error handling to the next level.** `erro` is a powerful, production-ready error library that gives you everything the standard `errors` package should have provided: structured context, stack traces, automatic HTTP status codes, seamless logging integration and comprehensive debugging tools.

#### 📦 Installation

```bash
go get -u github.com/maxbolgarin/erro
```

## ✨ Why `erro` > standard `errors`?

**Before: Basic Go errors leave you guessing**
```go
// Standard Go - minimal context, hard to debug
return fmt.Errorf("user operation failed: %w", err)

// What went wrong? Which user? What operation? What's the HTTP status?
// You'll spend time digging through logs to find out.
```

**After: Rich, actionable errors with full context**
```go
// erro - rich context, automatic status codes, structured logging
return erro.Wrap(err, "failed to update user profile",
    "user_id", userID,
    "operation", "profile_update", 
    erro.ClassValidation,        // → HTTP 400
    erro.CategoryUserInput,      // → Organized error tracking
    erro.SeverityMedium,         // → Priority for alerts
    erro.RecordSpan(span),       // → OpenTelemetry tracing
    erro.RecordMetrics(metrics), // → Prometheus metrics
    erro.SendEvent(ctx, dispatcher),  // → Error event tracking (e.g. Sentry, Honeycomb, etc.)
)

// Instant context: exactly what failed, for whom, why, and how to respond
```

## 🎯 Key Benefits

- **🔍 Rich Context**: Error carries structured metadata, stack traces, and debugging information with smart classification
- **📊 Logging-Native**: Integration with `slog`, `logrus` and any structured logger
- **🔍 Monitoring-Native**: Easyly gather metrics, trace spans or send events on error creation
- **🔒 Security-Aware**: Redaction of sensitive data in logs and traces, protection from DoS attacks
- **🔄 Drop-in Replacement**: Fully compatible with standard `errors` package - migrate gradually or just replace `errors` to `erro` in your codebase
- **🎯 Production-Ready**: Comprehensive testing, thread-safe, used in production environments, performance is suitable for 95% of use cases (watch [benchmarks below](#-performance--benchmark))

## 🚀 Quick Start

```go
package main

import (
    "log/slog"
    "net/http"
    "github.com/maxbolgarin/erro"
)

// Create rich errors with context
func GetUser(id int) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        return nil, erro.Wrap(err, "failed to retrieve user",
            "user_id", id,
            "table", "users",
            erro.ClassNotFound,      // → HTTP 404
            erro.CategoryDatabase,   // → Error grouping
        )
    }
    return user, nil
}

// HTTP handler with automatic status codes
func UserHandler(w http.ResponseWriter, r *http.Request) {
    user, err := GetUser(123)
    if err != nil {
        // Automatic HTTP status code mapping
        statusCode := erro.HTTPCode(err)  // 404 from ClassNotFound
        
        // Rich structured logging
        slog.Error("user retrieval failed", erro.LogFields(err)...)
        
        http.Error(w, err.Error(), statusCode)
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

## 🔥 Advanced Features

### 🏷️ Error Templates - Consistency Made Easy

Tutorial: [Error Templates](docs/template-creation.md)

```go
// Define reusable error templates
var (
    ValidationError = erro.NewTemplate("validation failed: %s",
        erro.ClassValidation,
        erro.CategoryUserInput,
        erro.SeverityLow,
    )
)

// Use templates for consistent error creation
func ValidateEmail(email string) error {
    if !isValidEmail(email) {
        return ValidationError.New("invalid email format", 
            "email", email,
            "pattern", emailRegex,
        )
    }
    return nil
}
```

### 📊 Error Collections - Handle Multiple Errors
```go
// Collect multiple validation errors
validator := erro.NewList()
validator.New("email required", "field", "email", erro.ClassValidation)
validator.New("password too short", "field", "password", "min_length", 8)

// Return all errors at once
if err := validator.Err(); err != nil {
    return err  // "multiple errors (2): [1] email required; [2] password too short"
}

// Thread-safe collections for concurrent operations
safeCollector := erro.NewSafeSet()  // Deduplicates identical errors
```

### 🔒 Security & Sensitive Data Protection
```go
func AuthenticateUser(username, password string) error {
    err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
    if err != nil {
        return erro.Wrap(err, "authentication failed",
            "username", username,
            "password", erro.Redact(password),  // Automatically redacted in logs
            "ip_address", clientIP,
            "attempt_count", attemptCount,
            erro.ClassUnauthenticated,
        )
    }
    return nil
}
```

### 🔍 Stack Traces & Debugging

Tutorial: [Stack Trace Configuration](docs/stack-trace-configuration.md)

```go
// Capture stack traces for debugging
func CriticalOperation() error {
    return erro.New("critical system failure",
        "component", "payment_processor",
        "transaction_id", txID,
        erro.StackTrace(),           // Capture stack trace
        erro.ClassCritical,
        erro.SeverityCritical,
    )
}

// Print detailed stack trace
fmt.Printf("%+v\n", err)  // Full stack trace with file:line info
```

### 🌐 HTTP Integration & REST APIs
```go
// Automatic HTTP status code mapping
func APIErrorHandler(err error) (int, any) {
    statusCode := erro.HTTPCode(err)  // Intelligent mapping
    
    response := map[string]any{
        "error": err.Error(),
        "status": statusCode,
        "timestamp": time.Now(),
    }
    
    // Add structured error details
    if erroErr, ok := err.(erro.Error); ok {
        response["error_id"] = erroErr.ID()      // Unique error ID
        response["class"] = erroErr.Class()      // Error classification
        response["severity"] = erroErr.Severity() // Priority level
        response["retryable"] = erroErr.IsRetryable()
        response["fields"] = erroErr.LogFieldsMap()
    }
    
    return statusCode, response
}

// Status code mapping examples:
// erro.ClassValidation      → 400 Bad Request
// erro.ClassNotFound        → 404 Not Found  
// erro.ClassUnauthenticated → 401 Unauthorized
// erro.ClassPermissionDenied → 403 Forbidden
// erro.ClassRateLimited     → 429 Too Many Requests
// erro.ClassExternal        → 502 Bad Gateway
```

### 📈 Observability & Monitoring Integration
```go
// Integrate with tracing, metrics, and events
func ProcessPayment(ctx context.Context, amount float64) error {
    // ...
    
    return erro.New("payment processing failed",
        "amount", amount,
        "currency", "USD",
        "merchant_id", merchantID,
        
        // Observability integrations
        erro.RecordSpan(span),              // OpenTelemetry tracing
        erro.RecordMetrics(metricsCollector), // Prometheus metrics
        erro.SendEvent(ctx, eventDispatcher), // Error event tracking
        
        erro.ClassExternal,
        erro.CategoryPayment,
        erro.SeverityCritical,
    )
}
```

## 🔄 Migration Guide

### Drop-in Replacement
```go
// Before: Standard errors
import "errors"

err := errors.New("something failed")
```

```go
// After: Rich errors
import "github.com/maxbolgarin/erro"

err := erro.New("something failed")

// All standard functions still work
if erro.Is(err, err1) { ... }  // ✅ Works
erro.Unwrap(err1)  // ✅ Works
```

### Gradual Enhancement
```go
// Phase 1: Basic replacement
- return errors.New("user not found")
+ return erro.New("user not found")

// Phase 2: Add context  
+ return erro.New("user not found", "user_id", id)

// Phase 3: Add classification
+ return erro.New("user not found", "user_id", id, erro.ClassNotFound)

// Phase 4: Full features
+ return erro.New("user not found",
+     "user_id", id,
+     "table", "users", 
+     erro.ClassNotFound,
+     erro.CategoryDatabase,
+ )
```

## 📊 Performance & Benchmark

### What AI says about this package after writing edge cases tests

**The erro library now demonstrates:**

- Robust security with DoS protection
- Excellent performance (~416ns/error)
- Memory safety with no leaks
- Thread safety with comprehensive concurrency testing
- Standards compliance with Go error interfaces
- High test coverage with extensive edge case coverage

### Benchmarks

```bash
go test -bench . -benchmem
```

```text
goos: darwin
goarch: arm64
pkg: github.com/maxbolgarin/erro
cpu: Apple M1 Pro
```

#### New

```text
Benchmark_New_STD-8                                   1000000000               0.316 ns/op           0 B/op            0 allocs/op

Benchmark_New-8                                          8659695               132.8 ns/op           272 B/op          2 allocs/op
Benchmark_New_WithFields-8                               6191596               207.1 ns/op           400 B/op          3 allocs/op
Benchmark_New_WithFieldsAndFormatVerbs-8                 4019469               290.0 ns/op           464 B/op          4 allocs/op
Benchmark_NewWithStack-8                                 1917016               602.4 ns/op           336 B/op          4 allocs/op
Benchmark_NewWithStack_WithFields-8                      1644594               736.4 ns/op           464 B/op          5 allocs/op
```

#### Wrap

```text
Benchmark_Errorf_STD-8                                  10615393               110.3 ns/op            80 B/op          2 allocs/op
Benchmark_Errorf_STD_WithFields-8                        3269908               397.5 ns/op           128 B/op          2 allocs/op

Benchmark_Wrap_STD-8                                     5731766               211.0 ns/op           272 B/op          2 allocs/op
Benchmark_Wrap_WithFields-8                              4054627               313.4 ns/op           400 B/op          3 allocs/op
Benchmark_Wrap_WithFieldsAndFormatVerbs-8                3114643               396.7 ns/op           464 B/op          4 allocs/op
Benchmark_WrapWithStack-8                                1617450               799.5 ns/op           336 B/op          4 allocs/op
Benchmark_WrapWithStack_WithFields-8                     1297959               928.0 ns/op           464 B/op          5 allocs/op
Benchmark_Wrap_Erro_NoIDGeneration-8                    14933049               81.17 ns/op           256 B/op          1 allocs/op
```

#### Error() - has cache, only first call is slow

```text
Benchmark_New_ErrorString-8                             297651590                4.021 ns/op           0 B/op          0 allocs/op
Benchmark_New_ErrorString_WithFields-8                  303490167                3.926 ns/op           0 B/op          0 allocs/op
Benchmark_Wrap_Error_Deep-8                             295614921                4.210 ns/op           0 B/op          0 allocs/op
```

#### AllMeta - optimized means preallocation slice of options

```text
Benchmark_New_AllMeta_WithStack-8                         968913               1076 ns/op            896 B/op         14 allocs/op
Benchmark_New_AllMeta_NoStack-8                          2255648               513.6 ns/op           824 B/op         11 allocs/op
Benchmark_New_AllMeta_NoStack_Optimized-8                5139520               229.7 ns/op           384 B/op          2 allocs/op
```

#### LogFields

```text
Benchmark_Error_Context-8                               30605966               37.19 ns/op           64 B/op           1 allocs/op
Benchmark_LogFields_Default-8                            3674788               325.0 ns/op           768 B/op         10 allocs/op
Benchmark_LogFields_Minimal-8                            4873533               257.2 ns/op           720 B/op          5 allocs/op
Benchmark_LogFields_Verbose-8                            2671768               423.3 ns/op           840 B/op         12 allocs/op
Benchmark_LogFieldsMap-8                                 1997305               614.1 ns/op          1432 B/op         14 allocs/op
Benchmark_LogError-8                                     3502598               357.7 ns/op           784 B/op         11 allocs/op
```

#### Template

```text
Benchmark_New_Template-8                                 5434267               221.1 ns/op           448 B/op         10 allocs/op
Benchmark_New_FromTemplate-8                             5488628               220.5 ns/op           400 B/op          2 allocs/op
Benchmark_New_FromTemplate_WithMessageAndFields-8        3428262               354.6 ns/op           576 B/op          4 allocs/op
Benchmark_New_FromTemplate_Full-8                        1395340               860.9 ns/op           624 B/op          5 allocs/op
Benchmark_Wrap_FromTemplate-8                            6116706               196.3 ns/op           400 B/op          2 allocs/op
Benchmark_Wrap_FromTemplate_WithMessageAndFields-8       3667002               326.2 ns/op           576 B/op          4 allocs/op
Benchmark_Wrap_FromTemplate_Full-8                       1215620               996.9 ns/op           624 B/op          5 allocs/op
```

#### HTTPCode

```text
Benchmark_HTTPCode_Class-8                              57952584                20.52 ns/op           16 B/op          1 allocs/op
Benchmark_HTTPCode_Category-8                           56516245                20.91 ns/op           16 B/op          1 allocs/op
```

#### Sprintf vs ApplyFormatVerbs

```text
Benchmark_Sprintf-8                                      3440610               343.2 ns/op           144 B/op          5 allocs/op
Benchmark_ApplyFormatVerbs-8                             4271595               268.5 ns/op           160 B/op          5 allocs/op
```

### 🎯 Performance Insights

**⚡ Operations Insights**
- Standard `errors.New()`: **0.32ns** (baseline, no features)
- `erro.New()` no fields: **133ns** (small overhead in most cases)
- Standard `fmt.Errorf()` with only error wrapped: **110ns**
- Standard `fmt.Errorf()` with error wrapped and many fields: **397ns**
- `erro.Wrap()` no fields with `errors.New` wrapping: **211ns** (overhead like `erro.New` + wrapping logic)
- `erro.Wrap()` with fields: **313ns** (faster than `fmt.Errorf()` with fields)
- `erro.Wrap()` is fast (**81ns**) if you already have an `erro.Error`, so in long chains it wins over many `fmt.Errorf()`
- HTTP status code mapping: **20ns** - virtually zero overhead

**🤔 Consider Alternatives For**
- **Ultra-high Performance** - If you need absolute minimal overhead (so do not use `fmt.Errorf` and `fmt.Sprintf`)
- **Simple Scripts** - Standard `errors` might be sufficient for basic scripts 
- **Legacy Codebases** - If you can't gradually migrate existing error handling


## Why Go's approach to errors is better than exceptions?

Go's approach to errors—as values rather than exceptions—offers significant advantages for **modern microservice and observability-driven architectures**. 

By treating errors as explicit return values, Go encourages developers to handle failures directly at the point where they occur, making error flows transparent and predictable. This explicitness enables rich error wrapping, structured context, and metadata propagation, which are essential for **tracing issues across distributed systems**. Unlike try-catch mechanisms that can obscure the origin and context of failures, Go's model makes it easy to:
* attach contextual information
* correlate errors with logs and traces
* surface actionable insights for monitoring and debugging

As a result, error handling in Go aligns naturally with the needs of production-grade systems, where observability, debuggability, and reliability are paramount. Use the natural strength of Go approach to errors and build your systems with observability in mind.


## 📚 Documentation

- **[Stack Trace Configuration](docs/stack-trace-configuration.md)** - Configure stack traces for different environments
- **[Log Fields Configuration](docs/log-fields-configuration.md)** - Master structured logging integration
- **[Template Creation](docs/template-creation.md)** - Create consistent, reusable error patterns
- **[API Reference](https://pkg.go.dev/github.com/maxbolgarin/erro)** - Complete API documentation
- **[Examples](examples/)** - Usage examples

## 🤝 Contributing

We welcome contributions! Open an issue or submit a pull request.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

[version-img]: https://img.shields.io/badge/Go-%3E%3D%201.18-%23007d9c
[doc-img]: https://pkg.go.dev/badge/github.com/maxbolgarin/erro
[doc]: https://pkg.go.dev/github.com/maxbolgarin/erro
[ci-img]: https://github.com/maxbolgarin/erro/actions/workflows/go.yaml/badge.svg
[ci]: https://github.com/maxbolgarin/erro/actions
[report-img]: https://goreportcard.com/badge/github.com/maxbolgarin/erro
[report]: https://goreportcard.com/report/github.com/maxbolgarin/erro
[coverage-img]: https://codecov.io/gh/maxbolgarin/erro/branch/main/graph/badge.svg
[coverage]: https://codecov.io/gh/maxbolgarin/erro
