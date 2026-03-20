# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`erro` is a Go error handling library that extends the standard `errors` package with structured context, stack traces, error classification (class/category/severity), HTTP status code mapping, logging integration, and observability hooks. It is a single-package library (`package erro`) with no external dependencies, targeting Go 1.18+.

## Commands

```bash
# Build
go build -v ./...

# Run all tests with race detection
go test -v -race .

# Run a single test
go test -v -run TestName .

# Run benchmarks
go test -bench . -benchmem .

# Run tests with coverage
go test -v -race -coverprofile=coverage.out .
```

Note: tests are in the root package only (`.`), not in subpackages. The `examples/` directory contains standalone example programs, not tests.

## Architecture

All source lives in the root directory as `package erro`. There are no subpackages.

### Core types and flow

- **`entities.go`** — Public interfaces and types: `Error` interface, `ErrorSchema`, `RedactedValue`, `TraceSpan`, `ErrorMetrics`, `EventDispatcher`, security constants (max lengths, max depth), and `KeyGetterFunc` for deduplication.
- **`base.go`** — `baseError` struct (the concrete implementation of `Error`). Contains all interface method implementations, `applyMeta()` which processes the variadic `fields ...any` argument, and the `newBaseError`/`newWrapError` constructors.
- **`meta.go`** — Functional options (`errorOpt`, `errorWork`, `errorFields` types) and the `ErrorClass`, `ErrorCategory`, `ErrorSeverity` typed string constants. Options like `ID()`, `Retryable()`, `StackTrace()`, `RecordSpan()`, `RecordMetrics()`, `SendEvent()` are defined here.
- **`erro.go`** — Public API entry points: `New()`, `Wrap()`, `Close()`, `Shutdown()`, `Is()`, `As()`, `Unwrap()`, `Join()`, `HTTPCode()`. The `HTTPCode` function maps `ErrorClass` and `ErrorCategory` to HTTP status codes.

### Metadata processing

The variadic `fields ...any` parameter in `New()`/`Wrap()` accepts a mix of key-value pairs, `ErrorClass`, `ErrorCategory`, `ErrorSeverity` values, and functional options (`errorOpt`/`errorWork`/`errorFields`). `applyMeta()` in `base.go` dispatches each value by type using a type switch. Options (`errorOpt`) execute immediately, work functions (`errorWork`) execute after fields are set (e.g., for span recording), and remaining values become key-value field pairs.

### Supporting files

- **`stack.go`** — Stack trace capture (`rawStack`, `captureStack`), frame resolution (`StackFrame`, `Stack`), `StackTraceConfig` presets (Development/Production/Strict), frame classification (user/stdlib/runtime/internal).
- **`strings.go`** — String formatting: `FormatErrorWithFields`, `FormatErrorMessage`, `ApplyFormatVerbs` (custom format verb processor), `appendValue` (type-switched value formatter), ID generation (`newID`), `atomicValue` generic wrapper, UTF-8 safe truncation.
- **`context.go`** — Public helper functions: `ExtractError()`, `LogFields()`, `LogFieldsMap()`, `LogError()`, `ErrorToJSON()`, `LogOptions` struct with functional option pattern (`WithID()`, `WithStack()`, etc.), and the `getLogFields`/`getLogFieldsMap` implementations.
- **`template.go`** — `ErrorTemplate` for reusable error patterns with predefined metadata. Includes ~25 predefined templates (e.g., `ValidationError`, `NotFoundError`, `DatabaseError`).
- **`list.go`** — Error collection types: `List` (not thread-safe), `Set` (deduplicating, not thread-safe), `SafeList` and `SafeSet` (thread-safe wrappers using `sync.RWMutex`). Also contains `multiError` and `multiErrorSet` types.

### Key design patterns

- **Immutability**: errors are immutable after creation; safe for concurrent reads.
- **Metadata inheritance**: wrapped errors inherit class, category, severity, ID, span, and created time from inner errors when not overridden at the wrapping layer.
- **Cached `Error()` string**: `fullMessage` uses `atomicValue` for lazy, thread-safe caching.
- **DoS protection**: message, key, and value lengths are truncated; field count is capped at `MaxFieldsCount` (100); wrap depth is limited to `MaxWrapDepth` (50).
- **Two-phase meta processing**: `applyMeta` runs `errorOpt` first (to set struct fields), then `errorWork` second (so span recording has access to fully-constructed error).

## CI

Tests run against Go 1.18 through 1.24 and `1.x` (latest) with race detection. Coverage is uploaded to Codecov.
