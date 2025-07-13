package erro

import (
	"context"
	"strconv"
	"strings"
	"sync"
)

// List collects multiple errors and provides the same chaining API as Error.
// It doesn't implement the error interface itself, but provides Err() to get a combined error.
type List struct {
	errors []Error
	// Metadata that will be applied to errors added to this list
	class     Class
	category  Category
	severity  Severity
	fields    []any
	ctx       context.Context
	retryable bool
}

// Newlist creates a new error list
func NewList(capacityRaw ...int) *List {
	var capacity int
	if len(capacityRaw) > 0 {
		capacity = capacityRaw[0]
	}
	return &List{
		errors: make([]Error, 0, capacity),
	}
}

// Add adds an error to the list, applying accumulated metadata
func (g *List) Add(err error) *List {
	if err == nil {
		return g
	}

	var erroErr Error
	if e, ok := err.(Error); ok {
		erroErr = e
	} else {
		erroErr = WrapEmpty(err)
	}

	return g.add(erroErr)
}

// New creates a new error with message and fields and adds it to the list
func (g *List) New(message string, fields ...any) *List {
	return g.add(New(message, fields...))
}

// Errorf creates a new error with formatted message and adds it to the list
func (g *List) Errorf(message string, args ...any) *List {
	return g.add(Errorf(message, args...))
}

// Wrap wraps an error with additional context and adds it to the list
func (g *List) Wrap(err error, message string, fields ...any) *List {
	if err == nil {
		return g.New(message, fields...)
	}
	return g.add(Wrap(err, message, fields...))
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (g *List) WrapEmpty(err error) *List {
	if err == nil {
		return g
	}
	return g.add(WrapEmpty(err))
}

// Wrapf wraps an error with formatted message and adds it to the list
func (g *List) Wrapf(err error, message string, args ...any) *List {
	if err == nil {
		return g.Errorf(message, args...)
	}
	return g.add(Wrapf(err, message, args...))
}

// Err returns a combined error from all errors in the list, or nil if empty.
// This prevents returning a non-nil error that represents an empty list.
func (g *List) Err() error {
	if len(g.errors) == 0 {
		return nil
	}
	if len(g.errors) == 1 {
		return g.errors[0]
	}
	return &multiError{errors: g.errors}
}

// Remove removes error at index i from the list.
func (g *List) Remove(i int) bool {
	if i < 0 || i >= len(g.errors) {
		return false
	}
	g.errors = append(g.errors[:i], g.errors[i+1:]...)
	return true
}

// RemoveError removes the first error that matches the given error.
func (g *List) RemoveError(err Error) bool {
	for i, e := range g.errors {
		if e.Error() == err.Error() {
			g.Remove(i)
			return true
		}
	}
	return false
}

// RemoveAll removes all errors from the list.
func (g *List) Clear() *List {
	g.errors = make([]Error, 0, cap(g.errors))
	return g
}

// Copy returns a copy of the list.
func (g *List) Copy() *List {
	clone := NewList(cap(g.errors))
	clone.errors = append(make([]Error, 0, len(g.errors)), g.errors...)
	clone.class = g.class
	clone.category = g.category
	clone.severity = g.severity
	clone.fields = append(make([]any, 0, len(g.fields)), g.fields...)
	clone.ctx = g.ctx
	clone.retryable = g.retryable
	return clone
}

// Errors returns a copy of the errors slice
func (g *List) Errors() []error {
	result := make([]error, len(g.errors))
	for i, err := range g.errors {
		result[i] = err
	}
	return result
}

// Errs returns a copy of the errors slice
func (g *List) Errs() []Error {
	result := make([]Error, len(g.errors))
	copy(result, g.errors)
	return result
}

// Len returns the number of errors in the list
func (g *List) Len() int {
	return len(g.errors)
}

// Empty returns true if the list is empty
func (g *List) Empty() bool {
	return len(g.errors) == 0
}

// NotEmpty returns true if the list is not empty
func (g *List) NotEmpty() bool {
	return len(g.errors) > 0
}

// First returns the first error in the list, or nil if empty.
func (g *List) First() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Last returns the last error in the list, or nil if empty.
func (g *List) Last() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

func (g *List) Class(class Class) *List {
	g.class = class
	return g
}

func (g *List) Category(category Category) *List {
	g.category = category
	return g
}

func (g *List) Severity(severity Severity) *List {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	g.severity = severity
	return g
}

func (g *List) GetClass() Class       { return g.class }
func (g *List) GetCategory() Category { return g.category }
func (g *List) GetFields() []any      { return g.fields }
func (g *List) GetContext() context.Context {
	return g.ctx
}
func (g *List) IsRetryable() bool { return g.retryable }

// Severity checking methods for List
func (g *List) IsCritical() bool { return g.severity == SeverityCritical }
func (g *List) IsHigh() bool     { return g.severity == SeverityHigh }
func (g *List) IsMedium() bool   { return g.severity == SeverityMedium }
func (g *List) IsLow() bool      { return g.severity == SeverityLow }
func (g *List) IsWarning() bool  { return g.severity == SeverityInfo }
func (g *List) IsUnknown() bool  { return g.severity == "" || g.severity == SeverityUnknown }
func (g *List) GetSeverity() Severity {
	if g.severity == "" {
		return SeverityUnknown
	}
	return g.severity
}

func (g *List) Fields(fields ...any) *List {
	g.fields = append(g.fields, prepareFields(fields)...)
	return g
}

func (g *List) Context(ctx context.Context) *List {
	g.ctx = ctx
	return g
}

func (g *List) Retryable(retryable bool) *List {
	g.retryable = retryable
	return g
}

func (g *List) add(err Error) *List {
	g.applyMetadata(err)
	g.errors = append(g.errors, err)
	return g
}

// applyMetadata applies accumulated metadata to an error
func (g *List) applyMetadata(err Error) {
	if g.class != ClassUnknown && err.GetClass() == ClassUnknown {
		err.Class(g.class)
	}
	if g.category != CategoryUnknown && err.GetCategory() == CategoryUnknown {
		err.Category(g.category)
	}
	if g.severity != SeverityUnknown && err.GetSeverity() == SeverityUnknown {
		err.Severity(g.severity)
	}
	if g.retryable {
		err.Retryable(g.retryable)
	}
	if len(g.fields) > 0 {
		err.Fields(g.fields...)
	}
	if g.ctx != nil && err.GetContext() == nil {
		err.Context(g.ctx)
	}
}

// Set collects unique errors and provides the same chaining API as Error.
// It deduplicates errors based on their message and code.
type Set struct {
	*List
	seen      map[string]int
	keyGetter func(error) string
}

// NewSet creates a new error set that stores only unique errors
func NewSet(capacityRaw ...int) *Set {
	var capacity int
	if len(capacityRaw) > 0 {
		capacity = capacityRaw[0]
	}
	return &Set{
		List:      NewList(capacity),
		seen:      make(map[string]int, capacity),
		keyGetter: MessageKeyGetter,
	}
}
func (s *Set) WithKeyGetter(keyGetter func(error) string) *Set {
	s.keyGetter = keyGetter
	return s
}

// Add adds an error to the set only if it's unique
func (s *Set) Add(err error) *Set {
	if err == nil {
		return s
	}

	var erroErr Error
	if e, ok := err.(Error); ok {
		erroErr = e
	} else {
		erroErr = WrapEmpty(err)
	}

	return s.add(erroErr)
}

// New creates a new error with message and fields and adds it to the set if unique
func (s *Set) New(message string, fields ...any) *Set {
	return s.add(New(message, fields...))
}

// Errorf creates a new error with formatted message and adds it to the set if unique
func (s *Set) Errorf(message string, args ...any) *Set {
	return s.add(Errorf(message, args...))
}

// Wrap wraps an error with additional context and adds it to the set if unique
func (s *Set) Wrap(err error, message string, fields ...any) *Set {
	if err == nil {
		return s.New(message, fields...)
	}
	return s.add(Wrap(err, message, fields...))
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (s *Set) WrapEmpty(err error) *Set {
	if err == nil {
		return s
	}
	return s.add(WrapEmpty(err))
}

// Wrapf wraps an error with formatted message and adds it to the set if unique
func (s *Set) Wrapf(err error, message string, args ...any) *Set {
	if err == nil {
		return s.Errorf(message, args...)
	}
	return s.add(Wrapf(err, message, args...))
}

// Err returns a combined error from all errors in the list, or nil if empty.
// This prevents returning a non-nil error that represents an empty list.
func (s *Set) Err() error {
	if len(s.errors) == 0 {
		return nil
	}

	// Create a copy of the errors for the multiError
	errorsCopy := make([]Error, len(s.errors))
	copy(errorsCopy, s.errors)
	return &multiErrorSet{errors: errorsCopy, counter: s.seen, keyGetter: s.keyGetter}
}

// Clear removes all errors from the set.
func (s *Set) Clear() *Set {
	s.errors = make([]Error, 0, cap(s.errors))
	s.seen = make(map[string]int, cap(s.errors))
	return s
}

// Copy returns a copy of the set.
func (s *Set) Copy() *Set {
	newSeen := make(map[string]int, len(s.seen))
	for k := range s.seen {
		newSeen[k] = s.seen[k]
	}
	return &Set{
		List:      s.List.Copy(),
		seen:      newSeen,
		keyGetter: s.keyGetter,
	}
}

func (s *Set) Class(class Class) *Set {
	s.List.Class(class)
	return s
}

func (s *Set) Category(category Category) *Set {
	s.List.Category(category)
	return s
}

func (s *Set) Severity(severity Severity) *Set {
	s.List.Severity(severity)
	return s
}

func (s *Set) GetClass() Class       { return s.List.GetClass() }
func (s *Set) GetCategory() Category { return s.List.GetCategory() }
func (s *Set) GetFields() []any      { return s.List.GetFields() }
func (s *Set) GetContext() context.Context {
	return s.List.GetContext()
}
func (s *Set) IsRetryable() bool { return s.List.IsRetryable() }

// Severity checking methods for Set
func (s *Set) IsCritical() bool      { return s.List.IsCritical() }
func (s *Set) IsHigh() bool          { return s.List.IsHigh() }
func (s *Set) IsMedium() bool        { return s.List.IsMedium() }
func (s *Set) IsLow() bool           { return s.List.IsLow() }
func (s *Set) IsWarning() bool       { return s.List.IsWarning() }
func (s *Set) IsUnknown() bool       { return s.List.IsUnknown() }
func (s *Set) GetSeverity() Severity { return s.List.GetSeverity() }

func (s *Set) Fields(fields ...any) *Set {
	s.List.Fields(fields...)
	return s
}

func (s *Set) Context(ctx context.Context) *Set {
	s.List.Context(ctx)
	return s
}

func (s *Set) Retryable(retryable bool) *Set {
	s.List.Retryable(retryable)
	return s
}

func (s *Set) add(err Error) *Set {
	s.applyMetadata(err)
	key := s.keyGetter(err)
	if _, ok := s.seen[key]; !ok {
		s.seen[key] = 1
		s.errors = append(s.errors, err)
	} else {
		s.seen[key]++
	}
	return s
}

func MessageKeyGetter(err error) string {
	if e, ok := err.(Error); ok {
		return e.GetMessage()
	}
	return err.Error()
}

func IDKeyGetter(err error) string {
	if e, ok := err.(Error); ok {
		return e.GetID()
	}
	return err.Error()
}

// SafeList is a thread-safe version of List that can be used safely across multiple goroutines
type SafeList struct {
	errors []Error
	// Metadata that will be applied to errors added to this list
	class     Class
	category  Category
	severity  Severity
	fields    []any
	ctx       context.Context
	retryable bool

	// Thread safety
	mu sync.RWMutex // Protects all fields
}

// NewSafeList creates a new thread-safe error list
func NewSafeList(capacityRaw ...int) *SafeList {
	var capacity int
	if len(capacityRaw) > 0 {
		capacity = capacityRaw[0]
	}
	return &SafeList{
		errors: make([]Error, 0, capacity),
	}
}

// Add adds an error to the list, applying accumulated metadata
func (g *SafeList) Add(err error) *SafeList {
	if err == nil {
		return g
	}

	var erroErr Error
	if e, ok := err.(Error); ok {
		erroErr = e
	} else {
		erroErr = WrapEmpty(err)
	}

	return g.add(erroErr)
}

// New creates a new error with message and fields and adds it to the list
func (g *SafeList) New(message string, fields ...any) *SafeList {
	return g.add(New(message, fields...))
}

// Errorf creates a new error with formatted message and adds it to the list
func (g *SafeList) Errorf(message string, args ...any) *SafeList {
	return g.add(Errorf(message, args...))
}

// Wrap wraps an error with additional context and adds it to the list
func (g *SafeList) Wrap(err error, message string, fields ...any) *SafeList {
	if err == nil {
		return g.New(message, fields...)
	}
	return g.add(Wrap(err, message, fields...))
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (g *SafeList) WrapEmpty(err error) *SafeList {
	if err == nil {
		return g
	}
	return g.add(WrapEmpty(err))
}

// Wrapf wraps an error with formatted message and adds it to the list
func (g *SafeList) Wrapf(err error, message string, args ...any) *SafeList {
	if err == nil {
		return g.Errorf(message, args...)
	}
	return g.add(Wrapf(err, message, args...))
}

// Err returns a combined error from all errors in the list, or nil if empty.
// This prevents returning a non-nil error that represents an empty list.
func (g *SafeList) Err() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	if len(g.errors) == 1 {
		return g.errors[0]
	}

	// Create a copy of the errors for the multiError
	errorsCopy := make([]Error, len(g.errors))
	copy(errorsCopy, g.errors)
	return &multiError{errors: errorsCopy}
}

// Remove removes error at index i from the list.
func (g *SafeList) Remove(i int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if i < 0 || i >= len(g.errors) {
		return false
	}
	g.errors = append(g.errors[:i], g.errors[i+1:]...)
	return true
}

// RemoveError removes the first error that matches the given error.
func (g *SafeList) RemoveError(err Error) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, e := range g.errors {
		if e.Error() == err.Error() {
			g.errors = append(g.errors[:i], g.errors[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all errors from the list.
func (g *SafeList) Clear() *SafeList {
	g.mu.Lock()
	g.errors = make([]Error, 0, cap(g.errors))
	g.mu.Unlock()
	return g
}

// Copy returns a copy of the list.
func (g *SafeList) Copy() *SafeList {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewSafeList(cap(g.errors))
	clone.errors = append(make([]Error, 0, len(g.errors)), g.errors...)
	clone.class = g.class
	clone.category = g.category
	clone.severity = g.severity
	clone.fields = append(make([]any, 0, len(g.fields)), g.fields...)
	clone.ctx = g.ctx
	clone.retryable = g.retryable
	return clone
}

// Errors returns a copy of the errors slice
func (g *SafeList) Errors() []error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]error, len(g.errors))
	for i, err := range g.errors {
		result[i] = err
	}
	return result
}

// Errs returns a copy of the errors slice
func (g *SafeList) Errs() []Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]Error, len(g.errors))
	copy(result, g.errors)
	return result
}

// Len returns the number of errors in the list
func (g *SafeList) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.errors)
}

// Empty returns true if the list is empty
func (g *SafeList) Empty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.errors) == 0
}

// NotEmpty returns true if the list is not empty
func (g *SafeList) NotEmpty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.errors) > 0
}

// First returns the first error in the list, or nil if empty.
func (g *SafeList) First() Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Last returns the last error in the list, or nil if empty.
func (g *SafeList) Last() Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

func (g *SafeList) Class(class Class) *SafeList {
	g.mu.Lock()
	g.class = class
	g.mu.Unlock()
	return g
}

func (g *SafeList) Category(category Category) *SafeList {
	g.mu.Lock()
	g.category = category
	g.mu.Unlock()
	return g
}

func (g *SafeList) Severity(severity Severity) *SafeList {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	g.mu.Lock()
	g.severity = severity
	g.mu.Unlock()
	return g
}

func (g *SafeList) GetClass() Class {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.class
}
func (g *SafeList) GetCategory() Category {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.category
}

func (g *SafeList) GetFields() []any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.fields
}
func (g *SafeList) GetContext() context.Context {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ctx
}
func (g *SafeList) IsRetryable() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.retryable
}

// Severity checking methods for List
func (g *SafeList) IsCritical() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == SeverityCritical
}
func (g *SafeList) IsHigh() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == SeverityHigh
}
func (g *SafeList) IsMedium() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == SeverityMedium
}
func (g *SafeList) IsLow() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == SeverityLow
}
func (g *SafeList) IsWarning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == SeverityInfo
}
func (g *SafeList) IsUnknown() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity == "" || g.severity == SeverityUnknown
}
func (g *SafeList) GetSeverity() Severity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.severity == "" {
		return SeverityUnknown
	}
	return g.severity
}

func (g *SafeList) Fields(fields ...any) *SafeList {
	g.mu.Lock()
	g.fields = append(g.fields, prepareFields(fields)...)
	g.mu.Unlock()
	return g
}

func (g *SafeList) Context(ctx context.Context) *SafeList {
	g.mu.Lock()
	g.ctx = ctx
	g.mu.Unlock()
	return g
}

func (g *SafeList) Retryable(retryable bool) *SafeList {
	g.mu.Lock()
	g.retryable = retryable
	g.mu.Unlock()
	return g
}

func (g *SafeList) add(err Error) *SafeList {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.applyMetadata(err)
	g.errors = append(g.errors, err)
	return g
}

// applyMetadata applies accumulated metadata to an error
func (g *SafeList) applyMetadata(err Error) {
	if g.class != ClassUnknown && err.GetClass() == ClassUnknown {
		err.Class(g.class)
	}
	if g.category != CategoryUnknown && err.GetCategory() == CategoryUnknown {
		err.Category(g.category)
	}
	if g.severity != SeverityUnknown && err.GetSeverity() == SeverityUnknown {
		err.Severity(g.severity)
	}
	if len(g.fields) > 0 {
		err.Fields(g.fields...)
	}
	if g.ctx != nil && err.GetContext() == nil {
		err.Context(g.ctx)
	}
	if g.retryable {
		err.Retryable(g.retryable)
	}
}

// SafeSet is a thread-safe version of Set that collects unique errors
type SafeSet struct {
	*SafeList
	seen      map[string]int
	keyGetter func(error) string
}

// NewSafeSet creates a new thread-safe error set that stores only unique errors
func NewSafeSet(capacityRaw ...int) *SafeSet {
	var capacity int
	if len(capacityRaw) > 0 {
		capacity = capacityRaw[0]
	}
	return &SafeSet{
		SafeList:  NewSafeList(capacity),
		seen:      make(map[string]int, capacity),
		keyGetter: MessageKeyGetter,
	}
}

func (s *SafeSet) WithKeyGetter(keyGetter func(error) string) *SafeSet {
	s.keyGetter = keyGetter
	return s
}

// Add adds an error to the set only if it's unique
func (s *SafeSet) Add(err error) *SafeSet {
	if err == nil {
		return s
	}

	var erroErr Error
	if e, ok := err.(Error); ok {
		erroErr = e
	} else {
		erroErr = WrapEmpty(err)
	}

	return s.add(erroErr)
}

// New creates a new error with message and fields and adds it to the set if unique
func (s *SafeSet) New(message string, fields ...any) *SafeSet {
	return s.add(New(message, fields...))
}

// Errorf creates a new error with formatted message and adds it to the set if unique
func (s *SafeSet) Errorf(message string, args ...any) *SafeSet {
	return s.add(Errorf(message, args...))
}

// Wrap wraps an error with additional context and adds it to the set if unique
func (s *SafeSet) Wrap(err error, message string, fields ...any) *SafeSet {
	if err == nil {
		return s.New(message, fields...)
	}
	return s.add(Wrap(err, message, fields...))
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (s *SafeSet) WrapEmpty(err error) *SafeSet {
	if err == nil {
		return s
	}
	return s.add(WrapEmpty(err))
}

// Wrapf wraps an error with formatted message and adds it to the set if unique
func (s *SafeSet) Wrapf(err error, message string, args ...any) *SafeSet {
	if err == nil {
		return s.Errorf(message, args...)
	}
	return s.add(Wrapf(err, message, args...))
}

// Err returns a combined error from all errors in the list, or nil if empty.
// This prevents returning a non-nil error that represents an empty list.
func (s *SafeSet) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.errors) == 0 {
		return nil
	}

	// Create a copy of the errors for the multiError
	errorsCopy := make([]Error, len(s.errors))
	copy(errorsCopy, s.errors)
	return &multiErrorSet{errors: errorsCopy, counter: s.seen, keyGetter: s.keyGetter}
}

// Clear removes all errors from the set.
func (s *SafeSet) Clear() *SafeSet {
	s.mu.Lock()
	s.errors = make([]Error, 0, cap(s.errors))
	s.seen = make(map[string]int, cap(s.errors))
	s.mu.Unlock()
	return s
}

// Copy returns a copy of the set.
func (s *SafeSet) Copy() *SafeSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	newSeen := make(map[string]int, len(s.seen))
	for k := range s.seen {
		newSeen[k] = s.seen[k]
	}
	return &SafeSet{
		SafeList:  s.SafeList.Copy(),
		seen:      newSeen,
		keyGetter: s.keyGetter,
	}
}

func (s *SafeSet) Class(class Class) *SafeSet {
	s.SafeList.Class(class)
	return s
}

func (s *SafeSet) Category(category Category) *SafeSet {
	s.SafeList.Category(category)
	return s
}

func (s *SafeSet) Severity(severity Severity) *SafeSet {
	s.SafeList.Severity(severity)
	return s
}

func (s *SafeSet) GetClass() Class             { return s.SafeList.GetClass() }
func (s *SafeSet) GetCategory() Category       { return s.SafeList.GetCategory() }
func (s *SafeSet) GetFields() []any            { return s.SafeList.GetFields() }
func (s *SafeSet) GetContext() context.Context { return s.SafeList.GetContext() }
func (s *SafeSet) IsRetryable() bool           { return s.SafeList.IsRetryable() }

// Severity checking methods for SafeSet
func (s *SafeSet) IsCritical() bool      { return s.SafeList.IsCritical() }
func (s *SafeSet) IsHigh() bool          { return s.SafeList.IsHigh() }
func (s *SafeSet) IsMedium() bool        { return s.SafeList.IsMedium() }
func (s *SafeSet) IsLow() bool           { return s.SafeList.IsLow() }
func (s *SafeSet) IsWarning() bool       { return s.SafeList.IsWarning() }
func (s *SafeSet) IsUnknown() bool       { return s.SafeList.IsUnknown() }
func (s *SafeSet) GetSeverity() Severity { return s.SafeList.GetSeverity() }

func (s *SafeSet) Fields(fields ...any) *SafeSet {
	s.SafeList.Fields(fields...)
	return s
}

func (s *SafeSet) Context(ctx context.Context) *SafeSet {
	s.SafeList.Context(ctx)
	return s
}

func (s *SafeSet) Retryable(retryable bool) *SafeSet {
	s.SafeList.Retryable(retryable)
	return s
}

func (s *SafeSet) add(err Error) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.applyMetadata(err)
	key := s.keyGetter(err)
	if _, ok := s.seen[key]; !ok {
		s.seen[key] = 1
		s.errors = append(s.errors, err)
	} else {
		s.seen[key]++
	}
	return s
}

// multiError represents multiple errors combined into one
type multiError struct {
	errors []Error
}

// Error implements the error interface for multiError
func (m *multiError) Error() string {
	if len(m.errors) == 0 {
		return ""
	}
	if len(m.errors) == 1 {
		return m.errors[0].Error()
	}

	var builder strings.Builder
	builder.WriteString("multiple errors (")
	builder.WriteString(strconv.Itoa(len(m.errors)))
	builder.WriteString("): ")
	for i, err := range m.errors {
		builder.WriteString("(")
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(") ")
		builder.WriteString(err.Error())
		if i < len(m.errors)-1 {
			builder.WriteString("; ")
		}
	}
	return builder.String()
}

// Unwrap returns the underlying errors for error chain traversal
func (m *multiError) Unwrap() []error {
	result := make([]error, len(m.errors))
	for i, err := range m.errors {
		result[i] = err
	}
	return result
}

// multiError represents multiple errors combined into one
type multiErrorSet struct {
	errors    []Error
	counter   map[string]int
	keyGetter func(error) string
}

// Error implements the error interface for multiError
func (m *multiErrorSet) Error() string {
	if len(m.errors) == 0 {
		return ""
	}
	if len(m.errors) == 1 {
		return m.errors[0].Error()
	}

	var builder strings.Builder
	builder.WriteString("multiple errors (")
	builder.WriteString(strconv.Itoa(len(m.errors)))
	builder.WriteString("): ")
	for i, err := range m.errors {
		builder.WriteString("(")
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString("): ")
		builder.WriteString(err.Error())
		builder.WriteString(" [")
		builder.WriteString(strconv.Itoa(m.counter[m.keyGetter(err)]))
		builder.WriteString("]")

		if i < len(m.errors)-1 {
			builder.WriteString("; ")
		}
	}
	return builder.String()
}

// Unwrap returns the underlying errors for error chain traversal
func (m *multiErrorSet) Unwrap() []error {
	result := make([]error, len(m.errors))
	for i, err := range m.errors {
		result[i] = err
	}
	return result
}
