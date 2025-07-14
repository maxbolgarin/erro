package erro

import (
	"context"
	"time"
)

// Builder is a mutable, chainable builder for creating any type of immutable error.
// It provides a performance-optimized path by creating the final error in a single allocation.
type Builder struct {
	// The underlying error to be wrapped. If nil, a new error is created.
	cause error

	// Core error information
	message string
	fields  []any

	// Contextual metadata
	id               string
	class            Class
	category         Category
	severity         Severity
	retryable        bool
	span             Span
	formatter        FormatErrorFunc
	stackTraceConfig *StackTraceConfig

	// Tracks if retryable was explicitly set.
	isRetryableSet bool
	isEncludeStack bool

	metrics   Metrics
	sendEvent func(err Error)
}

// NewError creates a new builder, optionally starting with a message and fields.
// This is the entry point for creating a new error from scratch.
func NewError(message string, fields ...any) *Builder {
	b := &Builder{
		message:   message,
		fields:    prepareFields(fields),
		formatter: FormatErrorWithFields,
	}
	return b
}

// NewWrapper creates a new builder to wrap an existing error.
// This is the entry point for wrapping an error.
func NewWrapper(err error, message string, fields ...any) *Builder {
	b := &Builder{
		cause:     err,
		message:   message,
		fields:    prepareFields(fields),
		formatter: FormatErrorWithFields,
	}
	return b
}

// ID sets the error's unique identifier.
func (b *Builder) WithID(id string) *Builder {
	b.id = id
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

// Retryable sets the retryable flag for the error.
func (b *Builder) WithRetryable(r bool) *Builder {
	b.retryable = r
	b.isRetryableSet = true
	return b
}

// Fields adds key-value pairs to the error context.
// It appends to any existing fields in the builder.
func (b *Builder) WithFields(fields ...any) *Builder {
	b.fields = append(b.fields, prepareFields(fields)...)
	return b
}

// Span associates an observability span with the error.
func (b *Builder) WithSpan(s Span) *Builder {
	b.span = s
	return b
}

func (b *Builder) WithFormatter(f FormatErrorFunc) *Builder {
	b.formatter = f
	return b
}

func (b *Builder) WithStackTraceConfig(c *StackTraceConfig) *Builder {
	b.stackTraceConfig = c
	b.isEncludeStack = true
	return b
}

// Stack configures the builder to create a lightweight error with a stack trace.
func (b *Builder) WithStack() *Builder {
	b.isEncludeStack = true
	return b
}

func (b *Builder) WithMetrics(m Metrics) *Builder {
	b.metrics = m
	return b
}

func (b *Builder) WithEvent(ctx context.Context, d Dispatcher) *Builder {
	if d == nil {
		return b
	}
	b.sendEvent = func(err Error) {
		d.SendEvent(ctx, err)
	}
	return b
}

func (b *Builder) WithEventFunc(f func(err Error)) *Builder {
	if f == nil {
		return b
	}
	b.sendEvent = f
	return b
}

// Build creates the final, immutable error based on the builder's configuration.
// This method performs a single allocation for the new error.
func (b *Builder) Build() Error {
	if b.id == "" {
		b.id = newID()
	}

	err := &baseError{
		message:          b.message,
		id:               b.id,
		class:            b.class,
		category:         b.category,
		severity:         b.severity,
		retryable:        b.retryable,
		fields:           b.fields,
		created:          time.Now(),
		formatter:        b.formatter,
		stackTraceConfig: b.stackTraceConfig,
	}

	var hasStack bool
	if erroErr, ok := b.cause.(*baseError); ok {
		err.wrappedErr = erroErr
		if errBase, ok := err.wrappedErr.BaseError().(*baseError); ok {
			hasStack = errBase.stack != nil || errBase.wrappedErr != nil
		}
	} else {
		err.originalErr = b.cause
	}

	if b.isEncludeStack && !hasStack {
		err.stackTraceConfig = b.stackTraceConfig
		err.stack = captureStack(3)
	}

	if b.span != nil {
		err.span = b.span
		err.span.RecordError(err)
		err.span.SetAttributes(b.fields...)
	}
	if b.sendEvent != nil {
		b.sendEvent(err)
	}
	if b.metrics != nil {
		b.metrics.RecordError(err)
	}

	return err
}
