# Template Creation Guide

Error templates provide a way to create consistent, reusable error patterns with predefined metadata and formatting. Templates ensure consistency across your application and make error handling more maintainable.

## Creating Templates

### Basic Template Creation

```go
// Basic template with message formatting
var ValidationError = erro.NewTemplate("validation failed: %s",
    erro.ClassValidation,
    erro.CategoryUserInput,
    erro.SeverityLow,
)

// Template with multiple format verbs
var DatabaseError = erro.NewTemplate("database %s failed on table %s",
    erro.CategoryDatabase,
    erro.SeverityHigh,
    erro.Retryable(),
)

// Template with rich metadata
var PaymentError = erro.NewTemplate("payment processing failed: %s",
    erro.CategoryPayment,
    erro.ClassExternal,
    erro.SeverityCritical,
    erro.Retryable(),
    erro.ID("PAYMENT_ERROR"),
)
```

### Template Structure

```go
template := erro.NewTemplate(messageTemplate, options...)
```

**Parameters:**
- `messageTemplate`: A string with optional format verbs (`%s`, `%d`, etc.)
- `options`: Any combination of erro metadata options (class, category, severity, etc.)

## Using Templates

### Creating New Errors

```go
// Format arguments fill the template's format verbs first
err := ValidationError.New("invalid email format", 
    "email", "not-an-email",
    "field", "user.email",
)

// Multiple format arguments
err := DatabaseError.New("connection", "users",
    "connection_string", "hidden",
    "timeout_ms", 5000,
)

// Template without format verbs
var SimpleError = erro.NewTemplate("operation failed",
    erro.ClassInternal,
    erro.SeverityMedium,
)

err := SimpleError.New(
    "operation_id", "op_123",
    "duration_ms", 2500,
)
```

### Wrapping Existing Errors

```go
originalErr := sql.ErrNoRows
err := DatabaseError.Wrap(originalErr, "SELECT", "products",
    "query_id", "q123",
    "user_id", 456,
)

// Wrapping with additional context
err := PaymentError.Wrap(originalErr, "stripe gateway timeout",
    "order_id", orderID,
    "amount", amount,
    "currency", "USD",
)
```

### Format Verb Handling

Templates intelligently handle format verbs:

```go
template := erro.NewTemplate("failed to process %s for user %d")

// First arguments are used for formatting
err := template.New("payment", 12345,
    "amount", 99.99,        // Additional fields
    "currency", "USD",      // Additional fields
)
// Message: "failed to process payment for user 12345"
// Fields: amount=99.99, currency=USD
```

## Predefined Templates

The package includes many predefined templates for common scenarios:

### Validation & Input Templates

```go
// Basic validation error
erro.ValidationError.New("email format invalid")

// Not found error
erro.NotFoundError.New("user")

// Conflict error
erro.ConflictError.New("email already exists")

// Example usage:
userErr := erro.ValidationError.New("invalid user data",
    "field", "email",
    "value", userEmail,
    "pattern", emailPattern,
)
```

### Authentication & Authorization Templates

```go
// Authentication failure
erro.AuthenticationError.New("invalid credentials")

// Authorization failure  
erro.AuthorizationError.New("insufficient privileges")

// Security violation
erro.SecurityError.New("suspicious activity detected")

// Example usage:
authErr := erro.AuthenticationError.New("login failed",
    "username", username,
    "attempt_count", attempts,
    "ip_address", clientIP,
)
```

### External Services Templates

```go
// External service error
erro.ExternalError.New("payment gateway timeout")

// Network error
erro.NetworkError.New("connection refused")

// Timeout error
erro.TimeoutError.New("service response timeout")

// Example usage:
extErr := erro.ExternalError.New("API call failed",
    "service", "stripe",
    "endpoint", "/charges",
    "status_code", 503,
    "response_time_ms", 30000,
)
```

### Database & Storage Templates

```go
// Database error
erro.DatabaseError.New("connection pool exhausted")

// Storage error
erro.StorageError.New("disk space insufficient")

// Cache error
erro.CacheError.New("redis connection lost")

// Example usage:
dbErr := erro.DatabaseError.New("query timeout",
    "table", "users",
    "operation", "SELECT",
    "timeout_ms", 10000,
    "rows_scanned", 1000000,
)
```

### System & Processing Templates

```go
// Internal error
erro.InternalError.New("unexpected nil pointer")

// Processing error
erro.ProcessingError.New("data transformation failed")

// Configuration error
erro.ConfigError.New("invalid configuration file")

// Example usage:
sysErr := erro.ProcessingError.New("image processing failed",
    "image_id", imageID,
    "format", "JPEG",
    "size_mb", 25.6,
    "step", "resize",
)
```

### Business Logic Templates

```go
// Business logic error
erro.BusinessLogicError.New("insufficient account balance")

// Payment error
erro.PaymentError.New("card declined")

// Rate limit error
erro.RateLimitError.New("API quota exceeded")

// Example usage:
bizErr := erro.BusinessLogicError.New("order validation failed",
    "order_id", orderID,
    "customer_id", customerID,
    "total_amount", total,
    "available_credit", credit,
)
```

### Resource & State Templates

```go
// Critical error
erro.CriticalError.New("memory leak detected")

// Temporary error
erro.TemporaryError.New("service temporarily unavailable")

// Data loss error
erro.DataLossError.New("backup corruption detected")

// Resource exhausted
erro.ResourceExhaustedError.New("memory limit exceeded")

// Service unavailable
erro.UnavailableError.New("maintenance mode")

// Operation cancelled
erro.CancelledError.New("user cancelled operation")

// Not implemented
erro.NotImplementedError.New("feature not available")

// Already exists
erro.AlreadyExistsError.New("user account exists")
```

## Advanced Template Features

### Templates with Stack Traces

```go
var CriticalTemplate = erro.NewTemplate("critical system error: %s",
    erro.ClassCritical,
    erro.SeverityCritical,
    erro.StackTrace(erro.ProductionStackTraceConfig()),
)

err := CriticalTemplate.New("memory leak detected",
    "component", "payment_processor",
    "memory_mb", 2048,
)
```

### Templates with Observability

```go
var TracedTemplate = erro.NewTemplate("traced operation failed: %s",
    erro.CategoryAPI,
    erro.SeverityMedium,
    erro.RecordMetrics(metricsCollector),
    erro.SendEvent(ctx, eventDispatcher),
)

err := TracedTemplate.New("API request failed",
    "endpoint", "/api/users",
    "method", "POST",
    "user_id", userID,
)
```

### Templates with Custom Formatting

```go
var FormattedTemplate = erro.NewTemplate("custom formatted error: %s",
    erro.Formatter(erro.GetFormatErrorWithFullContext(
        erro.WithSeverity(true),
        erro.WithCategory(true),
        erro.WithFunction(true),
    )),
)

err := FormattedTemplate.New("detailed context",
    "request_id", requestID,
)
```

### Templates with Multiple Options

```go
var ComprehensiveTemplate = erro.NewTemplate("operation %s failed for %s",
    // Classification
    erro.ClassInternal,
    erro.CategoryAPI,
    erro.SeverityHigh,
    
    // Behavior
    erro.Retryable(),
    erro.ID("API_OPERATION_FAILED"),
    
    // Observability
    erro.StackTrace(erro.ProductionStackTraceConfig()),
    erro.RecordMetrics(operationMetrics),
    
    // Formatting
    erro.Formatter(erro.FormatErrorWithFields),
)
```

## Template Best Practices

### Naming Conventions

```go
// Use descriptive names ending in Error or Template
var UserNotFoundError = erro.NewTemplate("user %d not found", erro.ClassNotFound)
var PaymentTemplate = erro.NewTemplate("payment %s failed", erro.CategoryPayment)

// Group related templates
var AuthErrors = struct {
    InvalidCredentials *erro.ErrorTemplate
    SessionExpired     *erro.ErrorTemplate
    PermissionDenied   *erro.ErrorTemplate
}{
    InvalidCredentials: erro.NewTemplate("authentication failed: %s", 
        erro.ClassUnauthenticated, erro.CategoryAuth),
    SessionExpired: erro.NewTemplate("session expired: %s",
        erro.ClassUnauthenticated, erro.CategoryAuth),
    PermissionDenied: erro.NewTemplate("permission denied: %s",
        erro.ClassPermissionDenied, erro.CategoryAuth),
}
```

### Consistent Metadata

```go
// Group related templates with consistent metadata
var DatabaseTemplates = struct {
    Connection *erro.ErrorTemplate
    Query      *erro.ErrorTemplate
    Migration  *erro.ErrorTemplate
}{
    Connection: erro.NewTemplate("database connection failed: %s",
        erro.CategoryDatabase, erro.SeverityHigh, erro.Retryable()),
    
    Query: erro.NewTemplate("database query failed: %s",
        erro.CategoryDatabase, erro.SeverityMedium),
    
    Migration: erro.NewTemplate("database migration failed: %s",
        erro.CategoryDatabase, erro.SeverityCritical),
}
```

### Context-Specific Templates

```go
// Create templates for specific domains
var PaymentTemplates = struct {
    GatewayTimeout    *erro.ErrorTemplate
    InsufficientFunds *erro.ErrorTemplate
    CardDeclined      *erro.ErrorTemplate
    FraudDetected     *erro.ErrorTemplate
}{
    GatewayTimeout: erro.NewTemplate("payment gateway timeout: %s",
        erro.ClassTimeout, erro.CategoryPayment, erro.Retryable()),
        
    InsufficientFunds: erro.NewTemplate("insufficient funds: %s",
        erro.ClassValidation, erro.CategoryPayment),
        
    CardDeclined: erro.NewTemplate("card declined: %s",
        erro.ClassValidation, erro.CategoryPayment),
        
    FraudDetected: erro.NewTemplate("fraud detected: %s",
        erro.ClassSecurity, erro.CategoryPayment, erro.SeverityCritical),
}

// Usage
err := PaymentTemplates.CardDeclined.New("expired card",
    "card_last_four", "1234",
    "expiry_date", "01/20",
    "order_id", orderID,
)
```

## Template Pattern Examples

### Service Layer Template

```go
type UserService struct {
    NotFoundError    *erro.ErrorTemplate
    ValidationError  *erro.ErrorTemplate
    DatabaseError    *erro.ErrorTemplate
}

func NewUserService() *UserService {
    return &UserService{
        NotFoundError: erro.NewTemplate("user %s not found",
            erro.ClassNotFound, erro.CategoryBusinessLogic),
        ValidationError: erro.NewTemplate("user validation failed: %s",
            erro.ClassValidation, erro.CategoryUserInput),
        DatabaseError: erro.NewTemplate("user database operation failed: %s",
            erro.CategoryDatabase, erro.SeverityHigh),
    }
}

func (s *UserService) GetUser(id string) (*User, error) {
    user, err := s.db.FindUser(id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, s.NotFoundError.New(id, "user_id", id)
        }
        return nil, s.DatabaseError.Wrap(err, "SELECT",
            "user_id", id,
            "table", "users",
        )
    }
    return user, nil
}

func (s *UserService) CreateUser(req *CreateUserRequest) (*User, error) {
    if err := s.validateUser(req); err != nil {
        return nil, s.ValidationError.Wrap(err, "create user validation",
            "email", req.Email,
            "username", req.Username,
        )
    }
    
    user, err := s.db.CreateUser(req)
    if err != nil {
        return nil, s.DatabaseError.Wrap(err, "INSERT",
            "email", req.Email,
            "table", "users",
        )
    }
    
    return user, nil
}
```

### HTTP Handler Template

```go
var HTTPTemplates = struct {
    BadRequest   *erro.ErrorTemplate
    Unauthorized *erro.ErrorTemplate
    NotFound     *erro.ErrorTemplate
    ServerError  *erro.ErrorTemplate
}{
    BadRequest: erro.NewTemplate("bad request: %s",
        erro.ClassValidation, erro.CategoryAPI),
    Unauthorized: erro.NewTemplate("unauthorized: %s",
        erro.ClassUnauthenticated, erro.CategoryAuth),
    NotFound: erro.NewTemplate("not found: %s",
        erro.ClassNotFound, erro.CategoryAPI),
    ServerError: erro.NewTemplate("server error: %s",
        erro.ClassInternal, erro.CategoryAPI, erro.SeverityHigh),
}

func handleUserRequest(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        err := HTTPTemplates.BadRequest.New("missing user ID",
            "parameter", "id",
            "method", r.Method,
            "path", r.URL.Path,
        )
        http.Error(w, err.Error(), erro.HTTPCode(err))
        return
    }
    
    user, err := userService.GetUser(userID)
    if err != nil {
        var httpErr erro.Error
        
        // Convert service errors to HTTP errors
        if errors.Is(err, userService.NotFoundError) {
            httpErr = HTTPTemplates.NotFound.Wrap(err, "user lookup failed",
                "user_id", userID,
            )
        } else {
            httpErr = HTTPTemplates.ServerError.Wrap(err, "user service error",
                "user_id", userID,
            )
        }
        
        http.Error(w, httpErr.Error(), erro.HTTPCode(httpErr))
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

### Repository Layer Template

```go
type UserRepository struct {
    ConnectionError *erro.ErrorTemplate
    QueryError      *erro.ErrorTemplate
    NotFoundError   *erro.ErrorTemplate
}

func NewUserRepository() *UserRepository {
    return &UserRepository{
        ConnectionError: erro.NewTemplate("database connection failed: %s",
            erro.CategoryDatabase, erro.SeverityHigh, erro.Retryable()),
        QueryError: erro.NewTemplate("query failed: %s",
            erro.CategoryDatabase, erro.SeverityMedium),
        NotFoundError: erro.NewTemplate("user %s not found",
            erro.ClassNotFound, erro.CategoryDatabase),
    }
}

func (r *UserRepository) FindUser(id string) (*User, error) {
    query := "SELECT * FROM users WHERE id = ?"
    
    var user User
    err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.Name)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, r.NotFoundError.New(id,
                "user_id", id,
                "table", "users",
            )
        }
        
        return nil, r.QueryError.Wrap(err, "user lookup",
            "user_id", id,
            "query", query,
            "table", "users",
        )
    }
    
    return &user, nil
}
```

### Domain-Specific Template Collections

```go
// E-commerce templates
var EcommerceTemplates = struct {
    // Order templates
    OrderNotFound     *erro.ErrorTemplate
    OrderCancelled    *erro.ErrorTemplate
    OrderAlreadyPaid  *erro.ErrorTemplate
    
    // Inventory templates  
    OutOfStock        *erro.ErrorTemplate
    InsufficientStock *erro.ErrorTemplate
    
    // Payment templates
    PaymentDeclined   *erro.ErrorTemplate
    PaymentTimeout    *erro.ErrorTemplate
}{
    // Order templates
    OrderNotFound: erro.NewTemplate("order %s not found",
        erro.ClassNotFound, erro.CategoryBusinessLogic),
    OrderCancelled: erro.NewTemplate("order %s is cancelled",
        erro.ClassValidation, erro.CategoryBusinessLogic),
    OrderAlreadyPaid: erro.NewTemplate("order %s already paid",
        erro.ClassValidation, erro.CategoryBusinessLogic),
    
    // Inventory templates
    OutOfStock: erro.NewTemplate("product %s out of stock",
        erro.ClassValidation, erro.CategoryBusinessLogic),
    InsufficientStock: erro.NewTemplate("insufficient stock for %s",
        erro.ClassValidation, erro.CategoryBusinessLogic),
    
    // Payment templates
    PaymentDeclined: erro.NewTemplate("payment declined: %s",
        erro.ClassValidation, erro.CategoryPayment),
    PaymentTimeout: erro.NewTemplate("payment timeout: %s",
        erro.ClassTimeout, erro.CategoryPayment, erro.Retryable()),
}

// Usage in business logic
func (s *OrderService) ProcessOrder(orderID string) error {
    order, err := s.getOrder(orderID)
    if err != nil {
        return EcommerceTemplates.OrderNotFound.Wrap(err, orderID,
            "order_id", orderID,
        )
    }
    
    if order.Status == "cancelled" {
        return EcommerceTemplates.OrderCancelled.New(orderID,
            "order_id", orderID,
            "cancelled_at", order.CancelledAt,
        )
    }
    
    // Check inventory
    for _, item := range order.Items {
        stock, err := s.inventory.GetStock(item.ProductID)
        if err != nil {
            return err
        }
        
        if stock < item.Quantity {
            if stock == 0 {
                return EcommerceTemplates.OutOfStock.New(item.ProductID,
                    "product_id", item.ProductID,
                    "requested", item.Quantity,
                )
            } else {
                return EcommerceTemplates.InsufficientStock.New(item.ProductID,
                    "product_id", item.ProductID,
                    "requested", item.Quantity,
                    "available", stock,
                )
            }
        }
    }
    
    // Process payment
    if err := s.processPayment(order); err != nil {
        return EcommerceTemplates.PaymentDeclined.Wrap(err, "card processing failed",
            "order_id", orderID,
            "amount", order.Total,
            "payment_method", order.PaymentMethod,
        )
    }
    
    return nil
}
```

## Template Organization Strategies

### Single File Organization

```go
// errors/templates.go
package errors

import "github.com/maxbolgarin/erro"

var (
    // User operations
    UserNotFound = erro.NewTemplate("user %s not found", erro.ClassNotFound)
    UserExists   = erro.NewTemplate("user %s already exists", erro.ClassConflict)
    
    // Database operations  
    DBConnection = erro.NewTemplate("database connection failed: %s", 
        erro.CategoryDatabase, erro.SeverityHigh)
    DBQuery = erro.NewTemplate("database query failed: %s",
        erro.CategoryDatabase, erro.SeverityMedium)
    
    // External services
    APITimeout = erro.NewTemplate("API timeout: %s",
        erro.ClassTimeout, erro.CategoryExternal, erro.Retryable())
    APIError = erro.NewTemplate("API error: %s", 
        erro.CategoryExternal, erro.SeverityMedium)
)
```

### Package-Based Organization

```go
// errors/user/templates.go
package user

var Templates = struct {
    NotFound   *erro.ErrorTemplate
    Validation *erro.ErrorTemplate
    Database   *erro.ErrorTemplate
}{
    NotFound: erro.NewTemplate("user %s not found", erro.ClassNotFound),
    Validation: erro.NewTemplate("user validation failed: %s", erro.ClassValidation),
    Database: erro.NewTemplate("user database error: %s", erro.CategoryDatabase),
}

// errors/payment/templates.go  
package payment

var Templates = struct {
    Declined *erro.ErrorTemplate
    Timeout  *erro.ErrorTemplate
    Fraud    *erro.ErrorTemplate
}{
    Declined: erro.NewTemplate("payment declined: %s", erro.ClassValidation),
    Timeout: erro.NewTemplate("payment timeout: %s", erro.ClassTimeout),
    Fraud: erro.NewTemplate("fraud detected: %s", erro.ClassSecurity),
}
```

### Interface-Based Organization

```go
type ErrorTemplates interface {
    NotFound(resource string, fields ...any) erro.Error
    Validation(reason string, fields ...any) erro.Error
    Internal(operation string, fields ...any) erro.Error
}

type userErrorTemplates struct {
    notFound   *erro.ErrorTemplate
    validation *erro.ErrorTemplate
    internal   *erro.ErrorTemplate
}

func NewUserErrorTemplates() ErrorTemplates {
    return &userErrorTemplates{
        notFound: erro.NewTemplate("user %s not found", erro.ClassNotFound),
        validation: erro.NewTemplate("user validation failed: %s", erro.ClassValidation),
        internal: erro.NewTemplate("user operation failed: %s", erro.ClassInternal),
    }
}

func (e *userErrorTemplates) NotFound(resource string, fields ...any) erro.Error {
    return e.notFound.New(resource, fields...)
}

func (e *userErrorTemplates) Validation(reason string, fields ...any) erro.Error {
    return e.validation.New(reason, fields...)
}

func (e *userErrorTemplates) Internal(operation string, fields ...any) erro.Error {
    return e.internal.New(operation, fields...)
}
```

## Template Testing

### Testing Template Creation

```go
func TestTemplates(t *testing.T) {
    // Test template creates error with correct metadata
    err := UserNotFoundError.New("12345", "source", "database")
    
    assert.Equal(t, erro.ClassNotFound, err.Class())
    assert.Contains(t, err.Error(), "user 12345 not found")
    
    fields := err.Fields()
    assert.Equal(t, "source", fields[0])
    assert.Equal(t, "database", fields[1])
}

func TestTemplateWrapping(t *testing.T) {
    originalErr := errors.New("connection refused")
    err := DatabaseError.Wrap(originalErr, "SELECT", "users",
        "timeout", 5000,
    )
    
    assert.Equal(t, erro.CategoryDatabase, err.Category())
    assert.Contains(t, err.Error(), "database SELECT failed on table users")
    assert.True(t, errors.Is(err, originalErr))
}
```

### Integration Testing

```go
func TestServiceErrorHandling(t *testing.T) {
    service := NewUserService()
    
    // Test not found scenario
    _, err := service.GetUser("nonexistent")
    
    var userErr erro.Error
    require.True(t, errors.As(err, &userErr))
    assert.Equal(t, erro.ClassNotFound, userErr.Class())
    assert.Contains(t, userErr.Error(), "user nonexistent not found")
}
```
