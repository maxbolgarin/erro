# 🚀 `erro` - Next-Generation Error Handling for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/maxbolgarin/erro.svg)](https://pkg.go.dev/github.com/maxbolgarin/erro)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxbolgarin/erro)](https://goreportcard.com/report/github.com/maxbolgarin/erro)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Transform your Go error handling from basic to brilliant.** `erro` is a powerful, production-ready error library that gives you everything the standard `errors` package should have provided: structured context, automatic HTTP status codes, seamless logging integration and comprehensive debugging tools.

## ✨ Why `erro`?

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
    "fields_updated", []string{"email", "name"},
    erro.ClassValidation,        // → HTTP 400
    erro.CategoryUserInput,      // → Organized error tracking
    erro.SeverityMedium,         // → Priority for alerts
)

// Instant context: exactly what failed, for whom, why, and how to respond
```

## 🎯 Key Benefits

- **🔍 Rich Context**: Error carries structured metadata, stack traces, and debugging information
- **🏷️ Smart Classification**: Organize errors by class, category, and severity for better handling
- **📊 Logging-Native**: Integration with `slog`, `logrus` and any structured logger
- **🔒 Security-Aware**: Redaction of sensitive data in logs and traces, protection from DoS attacks
- **🔄 Drop-in Replacement**: Fully compatible with standard `errors` package - migrate gradually
- **🎯 Production-Ready**: Comprehensive testing, thread-safe, used in production environments

## 🚀 Quick Start

### Installation
```bash
go get -u github.com/maxbolgarin/erro
```

### Basic Usage - Immediate Upgrade
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
```go
// Define reusable error templates
var (
    ValidationError = erro.NewTemplate("validation failed: %s",
        erro.ClassValidation,
        erro.CategoryUserInput,
        erro.SeverityLow,
    )
    
    DatabaseError = erro.NewTemplate("database operation failed: %s",
        erro.CategoryDatabase,
        erro.SeverityHigh,
        erro.Retryable(),  // Mark as retryable for retry logic
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
    return err  // "multiple errors (2): (1) email required; (2) password too short"
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
if errors.Is(err, err) { ... }  // ✅ Works
errors.Unwrap(err1)  // ✅ Works
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

## 📊 Performance & Production Ready

### Benchmarks
```
BenchmarkNew-8                  5000000    250 ns/op    120 B/op    2 allocs/op
BenchmarkWrap-8                 3000000    380 ns/op    180 B/op    3 allocs/op
BenchmarkLogFields-8            2000000    850 ns/op    320 B/op    8 allocs/op
BenchmarkHTTPCode-8            50000000     30 ns/op      0 B/op    0 allocs/op
```

### Production Features
- **Memory Safe**: Automatic field truncation prevents memory exhaustion
- **Thread Safe**: Immutable errors, safe error collections available
- **Zero Allocation**: Fast paths for common operations
- **Configurable**: Stack traces only when needed, adjustable limits
- **Battle Tested**: Comprehensive test suite, used in production

## 🎯 Use Cases

### ✅ Perfect For
- **REST APIs** - Automatic HTTP status codes, structured responses
- **Microservices** - Rich context for distributed debugging
- **Production Systems** - Comprehensive error tracking and monitoring
- **Team Development** - Standardized error handling across teams
- **Debugging** - Stack traces and structured context for faster resolution

### 🤔 Consider Alternatives For
- **Ultra-high Performance** - If you need absolute minimal overhead
- **Simple Scripts** - Standard `errors` might be sufficient for basic scripts
- **Legacy Codebases** - If you can't gradually migrate existing error handling

## 🛠️ Advanced Configuration

### Structured Logging Options
```go
// Configure logging output
err := erro.New("operation failed", "key", "value")

// Minimal logging
slog.Error("failed", erro.LogFields(err, 
    erro.WithUserFields(true),
    erro.WithID(true),
    erro.WithSeverity(true),
)...)

// Verbose logging  
slog.Error("failed", erro.LogFields(err,
    erro.WithUserFields(true),
    erro.WithStack(true),
    erro.WithTracing(true),
    erro.WithFieldNamePrefix("error_"),
)...)
```

### Stack Trace Configuration
```go
// Production config - minimal overhead
err := erro.New("error", erro.StackTrace(erro.ProductionStackTraceConfig()))

// Development config - full details
err := erro.New("error", erro.StackTrace(erro.DevelopmentStackTraceConfig()))

// Custom config
config := &erro.StackTraceConfig{
    MaxDepth:     20,
    SkipPackages: []string{"runtime", "net/http"},
    Format:       erro.StackFormatJSON,
}
err := erro.New("error", erro.StackTrace(config))
```

## 📚 Documentation

- **[API Reference](https://pkg.go.dev/github.com/maxbolgarin/erro)** - Complete API documentation
- **[Examples](examples/)** - Usage examples
- **[Best Practices](docs/best-practices.md)** - Recommended patterns and practices

## 🤝 Contributing

We welcome contributions! Open an issue or submit a pull request.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

