# Log Fields Configuration Guide

The erro package integrates seamlessly with structured logging libraries through flexible field configuration, allowing you to control exactly what error context appears in your logs.

## Basic Log Field Usage

```go
err := erro.New("database query failed",
    "table", "users",
    "query_time_ms", 150,
    "rows_affected", 0,
)

// With slog
slog.Error("Database operation failed", err.LogFields()...)

// With logrus (using LogFieldsMap)
logrus.WithFields(err.LogFieldsMap()).Error("Database operation failed")

// Custom logger integration
erro.LogError(err, func(message string, fields ...any) {
    myLogger.Error(message, fields...)
})
```

## Predefined Log Configurations

### Default Configuration

```go
opts := erro.DefaultLogOptions
// Includes: user fields, category, severity, tracing, retryable, function
// Excludes: ID, created time, package, file, line, stack

fields := erro.LogFields(err) // Uses DefaultLogOptions
```

### Verbose Configuration

```go
fields := erro.LogFields(err, erro.VerboseLogOpts...)
// Includes everything for debugging:
// - All user fields
// - Error ID, category, severity
// - Tracing information
// - Creation timestamp
// - Function, package, file, line
// - Full stack trace
```

### Minimal Configuration

```go
fields := erro.LogFields(err, erro.MinimalLogOpts...)
// Includes only:
// - User fields
// - Severity level
```

## LogOptions Configuration

Control which fields are included in logs using the `LogOptions` struct:

```go
type LogOptions struct {
    // User Fields
    IncludeUserFields  bool // Include custom key-value pairs

    // Error Metadata
    IncludeID          bool // Include error ID
    IncludeCategory    bool // Include error category
    IncludeSeverity    bool // Include error severity
    IncludeRetryable   bool // Include retryable flag
    IncludeTracing     bool // Include trace/span IDs
    IncludeCreatedTime bool // Include creation timestamp

    // Stack Information
    IncludeFunction    bool // Include function name
    IncludePackage     bool // Include package name
    IncludeFile        bool // Include file name
    IncludeLine        bool // Include line number
    IncludeStack       bool // Include full stack trace

    // Configuration
    StackFormat        StackFormat // How to format stack traces
    FieldNamePrefix    string      // Prefix for field names (default: "error_")
}
```

## Option Functions

Fine-tune logging behavior with individual option functions:

### Field Control Options

```go
// Enable/disable specific fields
erro.WithID(true)              // Include error ID
erro.WithCategory(false)       // Exclude category
erro.WithSeverity(true)        // Include severity
erro.WithTracing(true)         // Include trace/span IDs
erro.WithRetryable(true)       // Include retryable flag
erro.WithCreatedTime(false)    // Exclude creation timestamp

// User fields control
erro.WithUserFields(true)      // Include custom key-value pairs
```

### Stack Information Options

```go
erro.WithFunction(true)        // Include function name
erro.WithPackage(false)        // Exclude package name
erro.WithFile(false)           // Exclude file name
erro.WithLine(true)            // Include line number
erro.WithStack(true)           // Include full stack trace
```

### Configuration Options

```go
erro.WithFieldNamePrefix("svc_error_")           // Custom prefix
erro.WithStackFormat(erro.StackFormatJSON)      // Stack format
```

## Stack Format Options

Configure how stack traces appear in logs:

### String Format (Default)

```go
erro.WithStackFormat(erro.StackFormatString)
// Output: "main.processPayment -> payment.validateCard -> payment.checkBalance"
```

### List Format

```go
erro.WithStackFormat(erro.StackFormatList)
// Output: ["main.processPayment", "payment.validateCard", "payment.checkBalance"]
```

### Full Format

```go
erro.WithStackFormat(erro.StackFormatFull)
// Output: Multi-line detailed stack trace with file paths
//   main.processPayment
//     /app/main.go:42
//   payment.validateCard
//     /app/payment/card.go:15
```

### JSON Format

```go
erro.WithStackFormat(erro.StackFormatJSON)
// Output: [
//   {"function": "main.processPayment", "file": "main.go", "line": "42", "type": "user"},
//   {"function": "payment.validateCard", "file": "card.go", "line": "15", "type": "user"}
// ]
```

## Sensitive Data Protection

The package automatically handles sensitive data with redaction:

```go
err := erro.New("authentication failed",
    "username", "john",
    "password", erro.Redact("secret123"), // Automatically redacted
    "api_key", erro.Redact("key_abc123"),
    "ip_address", clientIP,
)

// In logs: password=[REDACTED], api_key=[REDACTED]
fields := erro.LogFields(err)
```


## Integration Examples

### With slog (Standard Library)

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

err := erro.New("payment processing failed",
    "order_id", 12345,
    "amount", 99.99,
    "currency", "USD",
    erro.CategoryPayment,
    erro.SeverityCritical,
)

// Basic integration
logger.Error("Payment failed", erro.LogFields(err)...)

// With custom options
logger.Error("Payment failed", 
    erro.LogFields(err, 
        erro.WithSeverity(true),
        erro.WithCategory(true),
        erro.WithTracing(true),
    )...)
```

### With logrus

```go
logger := logrus.New()
logger.SetFormatter(&logrus.JSONFormatter{})

err := erro.New("database connection lost", 
    "host", "db.example.com",
    "port", 5432,
    "database", "users",
    erro.CategoryDatabase,
    erro.SeverityHigh,
)

// Basic integration
logger.WithFields(erro.LogFieldsMap(err)).Error("Database error")

// With custom options
customFields := erro.LogFieldsMap(err,
    erro.WithUserFields(true),
    erro.WithSeverity(true),
    erro.WithFunction(true),
    erro.WithFieldNamePrefix("db_"),
)
logger.WithFields(customFields).Error("Database connection failed")
```

### With Zerolog

```go
logger := zerolog.New(os.Stdout)

err := erro.New("service unavailable",
    "service", "payment-gateway",
    "timeout_ms", 5000,
    erro.CategoryExternal,
    erro.SeverityMedium,
)

// Convert to map for zerolog
fields := erro.LogFieldsMap(err)
event := logger.Error()
for k, v := range fields {
    event = event.Interface(k, v)
}
event.Msg("Service call failed")
```

### With Custom Logger

```go
type CustomLogger struct {
    // Your logger implementation
}

func (l *CustomLogger) ErrorWithFields(message string, fields map[string]interface{}) {
    // Your logging implementation
}

err := erro.New("custom logging example", 
    "component", "auth",
    erro.SeverityHigh,
)

// Use LogError for custom integration
erro.LogError(err, func(message string, fields ...any) {
    // Convert fields to your format
    fieldMap := make(map[string]interface{})
    for i := 0; i < len(fields); i += 2 {
        if i+1 < len(fields) {
            key := fmt.Sprintf("%v", fields[i])
            fieldMap[key] = fields[i+1]
        }
    }
    
    customLogger.ErrorWithFields(message, fieldMap)
}, erro.VerboseLogOpts...)
```

## Environment-Specific Configurations

### Development Environment

```go
devLogOpts := []erro.LogOption{
    erro.WithUserFields(true),
    erro.WithID(true),
    erro.WithSeverity(true),
    erro.WithCategory(true),
    erro.WithTracing(true),
    erro.WithFunction(true),
    erro.WithFile(true),
    erro.WithLine(true),
    erro.WithStack(true),
    erro.WithStackFormat(erro.StackFormatFull),
    erro.WithFieldNamePrefix("dev_"),
}

// Rich debugging information
logger.Error("Development error", erro.LogFields(err, devLogOpts...)...)
```

### Production Environment

```go
prodLogOpts := []erro.LogOption{
    erro.WithUserFields(true),
    erro.WithSeverity(true),
    erro.WithCategory(true),
    erro.WithTracing(true),
    erro.WithFunction(true),
    erro.WithFile(false), // Hide file paths
    erro.WithLine(false), // Hide line numbers
    erro.WithStack(false), // No stack traces
    erro.WithFieldNamePrefix("prod_"),
}

// Privacy-aware logging
logger.Error("Production error", erro.LogFields(err, prodLogOpts...)...)
```

### Monitoring/Alerting Environment

```go
alertLogOpts := []erro.LogOption{
    erro.WithUserFields(false), // Exclude potentially noisy user fields
    erro.WithSeverity(true),
    erro.WithCategory(true),
    erro.WithFunction(true),
    erro.WithFieldNamePrefix("alert_"),
}

// Clean, focused alerts
alertLogger.Error("Alert condition", erro.LogFields(err, alertLogOpts...)...)
```

## Advanced Configuration Patterns

### Conditional Field Inclusion

```go
func getLogOptions(environment string, includeDebugInfo bool) []erro.LogOption {
    opts := []erro.LogOption{
        erro.WithUserFields(true),
        erro.WithSeverity(true),
        erro.WithCategory(true),
    }
    
    if environment == "development" || includeDebugInfo {
        opts = append(opts,
            erro.WithFunction(true),
            erro.WithFile(true),
            erro.WithLine(true),
            erro.WithStack(true),
        )
    }
    
    if environment == "production" {
        opts = append(opts, erro.WithTracing(true))
    }
    
    return opts
}

// Usage
opts := getLogOptions(os.Getenv("ENV"), shouldIncludeDebug())
logger.Error("Conditional logging", erro.LogFields(err, opts...)...)
```

### Service-Specific Configuration

```go
type ServiceLogger struct {
    logger   *slog.Logger
    logOpts  []erro.LogOption
    service  string
}

func NewServiceLogger(service string, isDev bool) *ServiceLogger {
    opts := []erro.LogOption{
        erro.WithUserFields(true),
        erro.WithSeverity(true),
        erro.WithCategory(true),
        erro.WithFieldNamePrefix(service + "_"),
    }
    
    if isDev {
        opts = append(opts, 
            erro.WithFunction(true),
            erro.WithStack(true),
        )
    }
    
    return &ServiceLogger{
        logger:  slog.Default(),
        logOpts: opts,
        service: service,
    }
}

func (sl *ServiceLogger) Error(msg string, err erro.Error) {
    fields := erro.LogFields(err, sl.logOpts...)
    fields = append(fields, "service", sl.service)
    sl.logger.Error(msg, fields...)
}
```

### Multi-Destination Logging

```go
func logErrorToMultipleDestinations(err erro.Error) {
    // Console (development)
    consoleFields := erro.LogFields(err, erro.VerboseLogOpts...)
    consoleLogger.Error("Error occurred", consoleFields...)
    
    // Application logs (production)
    appFields := erro.LogFields(err,
        erro.WithUserFields(true),
        erro.WithSeverity(true),
        erro.WithCategory(true),
        erro.WithFunction(true),
    )
    appLogger.Error("Application error", appFields...)
    
    // Monitoring (alerts)
    monitoringFields := erro.LogFields(err,
        erro.WithSeverity(true),
        erro.WithCategory(true),
        erro.WithFieldNamePrefix("monitor_"),
    )
    monitoringLogger.Error("Monitoring alert", monitoringFields...)
}
```

## Field Name Customization

### Custom Prefixes

```go
// Different prefixes for different services
userServiceOpts := []erro.LogOption{
    erro.WithFieldNamePrefix("user_svc_"),
    erro.WithSeverity(true),
    erro.WithCategory(true),
}

paymentServiceOpts := []erro.LogOption{
    erro.WithFieldNamePrefix("payment_svc_"),
    erro.WithSeverity(true),
    erro.WithCategory(true),
}

// Results in fields like:
// user_svc_severity, user_svc_category
// payment_svc_severity, payment_svc_category
```

### No Prefix

```go
// Clean field names without prefix
cleanOpts := []erro.LogOption{
    erro.WithFieldNamePrefix(""), // Empty prefix
    erro.WithSeverity(true),
    erro.WithCategory(true),
}

// Results in fields like: severity, category (instead of error_severity, error_category)
```

## Performance Considerations

### Lazy Field Generation

```go
// Fields are only generated when requested
err := erro.New("expensive operation failed",
    "large_data", someExpensiveToStringData,
)

// Fields generated only here
fields := erro.LogFields(err) // Computation happens now
```

### Field Caching

```go
// Cache expensive field computations
type CachedFieldsError struct {
    erro.Error
    cachedFields []any
}

func (e *CachedFieldsError) LogFields(opts ...erro.LogOptions) []any {
    if e.cachedFields == nil {
        e.cachedFields = e.Error.LogFields(opts...)
    }
    return e.cachedFields
}
```

## Best Practices

### 1. Environment-Specific Configuration

```go
var logOpts []erro.LogOption

switch os.Getenv("LOG_LEVEL") {
case "debug":
    logOpts = erro.VerboseLogOpts
case "info":
    logOpts = erro.DefaultLogOptions.ApplyOptions()
case "warn", "error":
    logOpts = erro.MinimalLogOpts
default:
    logOpts = []erro.LogOption{erro.WithUserFields(false)}
}
```

### 2. Consistent Field Naming

```go
// Use consistent prefixes across your application
const ErrorFieldPrefix = "app_error_"

var standardLogOpts = []erro.LogOption{
    erro.WithFieldNamePrefix(ErrorFieldPrefix),
    erro.WithUserFields(true),
    erro.WithSeverity(true),
    erro.WithCategory(true),
}
```

### 3. Structured Error Context

```go
// Build rich context systematically
err := erro.New("payment processing failed",
    // Business context
    "order_id", orderID,
    "customer_id", customerID,
    "payment_method", "credit_card",
    
    // Technical context
    "gateway", "stripe",
    "timeout_ms", 30000,
    "attempt", retryCount,
    
    // Error classification
    erro.CategoryPayment,
    erro.SeverityHigh,
    erro.Retryable(),
)
```

### 4. Security-Conscious Logging

```go
// Always redact sensitive information
err := erro.New("authentication failed",
    "username", username,                    // OK to log
    "password", erro.Redact(password),      // Redacted
    "session_id", erro.Redact(sessionID),   // Redacted
    "ip_address", clientIP,                 // OK to log (usually)
    "user_agent", erro.Redact(userAgent),  // Redacted (potential PII)
)
```
