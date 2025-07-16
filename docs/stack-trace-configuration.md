# Stack Trace Configuration Guide

Stack traces provide valuable debugging information by showing the call chain that led to an error. The erro package offers flexible stack trace configuration for different environments and security requirements.

## Capturing Stack Traces

Stack traces are **opt-in** for performance reasons. You must explicitly request them:

```go
// Basic stack trace capture
err := erro.New("database connection failed", 
    "host", "localhost",
    erro.StackTrace(), // Captures stack trace
)

// Stack trace with custom configuration
err := erro.New("payment failed",
    "amount", 100.50,
    erro.StackTrace(erro.ProductionStackTraceConfig()),
)

// Skip frames when capturing (useful for wrapper functions)
err := erro.New("validation failed",
    erro.StackTraceWithSkip(2), // Skip 2 more frames
)
```

## Predefined Stack Trace Configurations

The package provides several predefined configurations:

### Development Configuration

```go
config := erro.DevelopmentStackTraceConfig()
// Shows:
// - Full file paths
// - Function names with packages
// - Line numbers
// - All code frames (including stdlib)
```

### Production Configuration

```go
config := erro.ProductionStackTraceConfig()
// Shows:
// - Relative file paths (2 elements from project root)
// - Function names without packages
// - Line numbers
// - Only user code frames (filters out stdlib)
// - Limited to 10 frames
```

### Strict Configuration

```go
config := erro.StrictStackTraceConfig()
// Shows:
// - Filenames only (no paths)
// - No function names
// - Line numbers only
// - Limited to 3 frames
```

## Custom Stack Trace Configuration

Create your own configuration for specific needs:

```go
customConfig := &erro.StackTraceConfig{
    ShowFileNames:     true,   // Show file names
    ShowFullPaths:     true,   // Show full paths
    ShowFunctionNames: true,   // Show function names
    ShowPackageNames:  true,   // Show package names
    ShowLineNumbers:   true,   // Show line numbers
    ShowAllCodeFrames: true,   // Show all types of frames (user, stdlib, etc.)
    PathElements:      2,      // Show 2 path elements
    FunctionRedacted:  "[FUNC]", // Placeholder for hidden functions
    FileNameRedacted:  "[FILE]", // Placeholder for hidden files
    MaxFrames:         5,      // Limit to 5 frames
}

err := erro.New("custom error", erro.StackTrace(customConfig))
```

### Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `ShowFileNames` | `bool` | Whether to show file names |
| `ShowFullPaths` | `bool` | Whether to show full file paths |
| `PathElements` | `int` | Number of path elements to include (0 = filename only, -1 = full path) |
| `ShowFunctionNames` | `bool` | Whether to show function names |
| `ShowPackageNames` | `bool` | Whether to show package names |
| `ShowLineNumbers` | `bool` | Whether to show line numbers |
| `ShowAllCodeFrames` | `bool` | Whether to show all types of frames (user, stdlib, etc.) |
| `FunctionRedacted` | `string` | Placeholder for redacted function names |
| `FileNameRedacted` | `string` | Placeholder for redacted file names |
| `MaxFrames` | `int` | Maximum number of frames to show |

## Using Stack Traces

### Displaying Stack Traces

```go
// Print stack trace with %+v
fmt.Printf("Error with stack: %+v\n", err)

// Access stack programmatically
stack := err.Stack()
userFrames := stack.UserFrames()
topFrame := stack.TopUserFrame()

// Get origin context
origin := stack.GetOriginContext()
if origin != nil {
    fmt.Printf("Error originated in %s at line %d\n", 
        origin.Function, origin.Line)
}

// Convert to log fields
logFields := stack.ToLogFields()
```

### Stack Analysis

```go
stack := err.Stack()

// Get only user code frames (filters out stdlib/runtime)
userFrames := stack.UserFrames()

// Get the topmost user frame (where error likely originated)
topFrame := stack.TopUserFrame()
if topFrame != nil {
    fmt.Printf("Error in function: %s\n", topFrame.Name)
    fmt.Printf("File: %s:%d\n", topFrame.FileName, topFrame.Line)
}

// Get call chain
callChain := stack.GetCallChain()
fmt.Printf("Call chain: %s\n", strings.Join(callChain, " -> "))

// Extract involved packages
packages := stack.ExtractPackages()
fmt.Printf("Packages: %s\n", strings.Join(packages, ", "))

// Check for specific functions
if stack.ContainsFunction("processPayment") {
    fmt.Println("Error occurred in payment processing")
}

// Filter by package
paymentFrames := stack.FilterByPackage("payment")
```

### JSON Serialization

```go
// Convert stack to JSON
stackJSON := stack.ToJSON()

// Convert only user frames to JSON
userStackJSON := stack.ToJSONUserFrames()

// Example output:
// [
//   {
//     "function": "main.processPayment",
//     "file": "main.go:42",
//     "line": "42",
//     "type": "user"
//   }
// ]
```

## Environment-Specific Examples

### Development Environment

```go
// Full debugging information
err := erro.New("development error",
    "debug_info", "detailed context",
    erro.StackTrace(erro.DevelopmentStackTraceConfig()),
)

// Output shows:
// - Full file paths: /Users/dev/project/internal/service/payment.go:42
// - Full function names: service.processPayment
// - All frames including stdlib
```

### Production Environment

```go
// Privacy-aware configuration
err := erro.New("production error",
    "request_id", "req_123",
    erro.StackTrace(erro.ProductionStackTraceConfig()),
)

// Output shows:
// - Relative paths: internal/service/payment.go:42
// - Short function names: processPayment
// - Only user code frames
// - Limited number of frames
```

### Strict Security Environment

```go
// Minimal information
err := erro.New("security error",
    "event_type", "auth_failure",
    erro.StackTrace(erro.StrictStackTraceConfig()),
)

// Output shows:
// - Filename only: payment.go:42
// - No function names
// - Very limited frames
```

### Custom Environment Configuration

```go
// API service configuration
apiStackConfig := &erro.StackTraceConfig{
    ShowFileNames:     true,
    ShowFullPaths:     false,
    ShowFunctionNames: true,
    ShowPackageNames:  false,
    ShowLineNumbers:   true,
    ShowAllCodeFrames: false,
    PathElements:      1,  // Show parent dir + filename
    MaxFrames:         8,  // Reasonable limit for APIs
    FunctionRedacted:  "[API_FUNC]",
    FileNameRedacted:  "[API_FILE]",
}

err := erro.New("API error",
    "endpoint", "/api/users",
    "method", "POST",
    erro.StackTrace(apiStackConfig),
)
```

## Security Considerations

### Production Deployment

- **Use `ProductionStackTraceConfig()`** to hide internal paths
- **Avoid full paths** in logs that might be exposed
- **Limit frame count** to prevent information leakage
- **Filter stdlib frames** to focus on application code

### Strict Security Requirements

- **Use `StrictStackTraceConfig()`** for maximum privacy
- **Consider disabling** stack traces entirely in highly sensitive environments
- **Review log aggregation** systems for potential stack trace exposure
- **Implement log filtering** to remove stack traces from user-facing outputs

### Logging Integration

- **Never log stack traces** to user-facing systems
- **Use different configurations** for different log destinations
- **Implement log level controls** (debug vs production)
- **Consider structured logging** to separate stack data

### Best Practices

1. **Environment-specific configuration**:
   ```go
   var stackConfig *erro.StackTraceConfig
   switch os.Getenv("ENVIRONMENT") {
   case "development":
       stackConfig = erro.DevelopmentStackTraceConfig()
   case "production":
       stackConfig = erro.ProductionStackTraceConfig()
   case "strict":
       stackConfig = erro.StrictStackTraceConfig()
   }
   ```

2. **Conditional stack capture**:
   ```go
   var opts []any
   if shouldCaptureStack() {
       opts = append(opts, erro.StackTrace(getStackConfig()))
   }
   
   err := erro.New("conditional stack error", opts...)
   ```

3. **Wrapper function patterns**:
   ```go
   func wrapDatabaseError(err error, operation string) error {
       return erro.Wrap(err, "database operation failed",
           "operation", operation,
           erro.StackTraceWithSkip(1), // Skip this wrapper function
       )
   }
   ```

4. **Stack trace in different contexts**:
   ```go
   // For debugging
   debugErr := erro.New("debug issue", 
       erro.StackTrace(erro.DevelopmentStackTraceConfig()))
   
   // For monitoring
   monitoringErr := erro.New("monitoring issue",
       erro.StackTrace(erro.ProductionStackTraceConfig()))
   
   // For user-facing (no stack)
   userErr := erro.New("user issue") // No stack trace
   ```

## Performance Considerations

- **Stack traces have overhead** - only capture when needed
- **Frame resolution is lazy** - computed on first access and then cached
- **Consider sampling** in high-throughput scenarios
- **Use skip parameters** to avoid unnecessary wrapper frames
- **Cache configurations** rather than creating them repeatedly

## Advanced Usage

### Custom Frame Filtering

```go
// Get stack and filter manually
stack := err.Stack()
filteredFrames := make(erro.Stack, 0)

for _, frame := range stack {
    // Custom filtering logic
    if strings.Contains(frame.Package, "myapp") {
        filteredFrames = append(filteredFrames, frame)
    }
}
```

### Integration with Monitoring

```go
err := erro.New("monitored error",
    "service", "payment",
    erro.StackTrace(erro.ProductionStackTraceConfig()),
)

// Extract context for monitoring
stack := err.Stack()
if origin := stack.GetOriginContext(); origin != nil {
    monitoring.RecordError(map[string]interface{}{
        "error_function": origin.Function,
        "error_file":     origin.File,
        "error_line":     origin.Line,
        "call_chain":     strings.Join(stack.GetCallChain(), "->"),
    })
}
```
