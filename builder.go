package erro

import "sync"

var builderPool = sync.Pool{
	New: func() any {
		return &Builder{
			fields: make([]any, 0, 10),
		}
	},
}

func getBuilder() *Builder {
	return builderPool.Get().(*Builder)
}

func putBuilder(b *Builder) {
	b.fields = b.fields[:0]
	b.cause = nil
	b.message = ""
	b.id = ""
	b.class = ""
	b.category = ""
	b.severity = ""
	b.retryable = false
	builderPool.Put(b)
}

// Builder is a mutable, chainable builder for creating any type of immutable error.
// It provides a performance-optimized path by creating the final error in a single allocation.
type Builder struct {
	// The underlying error to be wrapped. If nil, a new error is created.
	cause error

	// Core error information
	message string
	fields  []any

	// Contextual metadata
	id        string
	class     Class
	category  Category
	severity  Severity
	retryable bool
	span      Span

	// Tracks if retryable was explicitly set.
	isRetryableSet bool
	isEncludeStack bool
}

// NewBuilder creates a new builder, optionally starting with a message and fields.
// This is the entry point for creating a new error from scratch.
func NewBuilder(message string, fields ...any) *Builder {
	b := getBuilder()
	b.message = message
	b.fields = fields
	return b
}

// NewBuilderWithError creates a new builder to wrap an existing error.
// This is the entry point for wrapping an error.
func NewBuilderWithError(err error, message string, fields ...any) *Builder {
	b := getBuilder()
	b.cause = err
	b.message = message
	b.fields = fields
	return b
}

// Message sets the message for the new error layer.
func (b *Builder) WithMessage(message string) *Builder {
	b.message = message
	return b
}

// Fields adds key-value pairs to the error context.
// It appends to any existing fields in the builder.
func (b *Builder) WithFields(fields ...any) *Builder {
	b.fields = append(b.fields, fields...)
	return b
}

// Category sets the error's category.
func (b *Builder) WithCategory(c Category) *Builder {
	b.category = c
	return b
}

// Class sets the error's class.
func (b *Builder) WithClass(c Class) *Builder {
	b.class = c
	return b
}

// Severity sets the error's severity.
func (b *Builder) WithSeverity(s Severity) *Builder {
	b.severity = s
	return b
}

// ID sets the error's unique identifier.
func (b *Builder) WithID(id string) *Builder {
	b.id = id
	return b
}

func (b *Builder) GenerateID() *Builder {
	b.id = newID(b.class, b.category)
	return b
}

// Retryable sets the retryable flag for the error.
func (b *Builder) WithRetryable(r bool) *Builder {
	b.retryable = r
	b.isRetryableSet = true
	return b
}

// Span associates an observability span with the error.
func (b *Builder) WithSpan(s Span) *Builder {
	b.span = s
	return b
}

// Stack configures the builder to create a lightweight error with a stack trace.
func (b *Builder) WithStack() *Builder {
	b.isEncludeStack = true
	return b
}

// Build creates the final, immutable error based on the builder's configuration.
// This method performs a single allocation for the new error.
func (b *Builder) Build() Error {
	defer putBuilder(b)

	// --- Case 1: Build a lightweight error ---
	if !b.isEncludeStack {
		return &lightError{
			message:   b.message,
			cause:     b.cause,
			id:        b.id,
			class:     b.class,
			category:  b.category,
			severity:  b.severity,
			retryable: b.retryable,
			fields:    b.fields,
			span:      b.span,
		}
	}

	// --- Case 2: Build a new `baseError` (no existing error to wrap, or wrapping a non-`erro` error) ---
	if b.cause == nil || !isErroError(b.cause) {
		// Call the internal constructor which captures the stack trace and can wrap a standard error.
		err := newBaseError(b.cause, b.message, b.fields...)

		// Directly populate the struct with the builder's context.
		err.id = b.id
		err.class = b.class
		err.category = b.category
		err.severity = b.severity
		err.retryable = b.retryable
		err.span = b.span
		return err
	}

	// --- Case 3: Build a `wrapError` to wrap an existing `erro.Error` ---
	// The builder itself represents the new layer of context.
	var retryablePtr *bool
	if b.isRetryableSet {
		retryablePtr = &b.retryable
	}

	return &wrapError{
		wrapped:     b.cause.(Error), // We know `cause` is an `erro.Error` here.
		wrapMessage: b.message,
		fields:      b.fields,
		id:          b.id,
		class:       b.class,
		category:    b.category,
		severity:    b.severity,
		retryable:   retryablePtr,
		span:        b.span,
	}
}

// isErroError is a helper to check if an error is one of our library's types.
func isErroError(err error) bool {
	_, ok := err.(Error)
	return ok
}
