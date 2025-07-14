package erro

import (
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
	erroErr := New(message, fields...)
	return g.add(erroErr)
}

// Errorf creates a new error with formatted message and adds it to the list
func (g *List) Errorf(message string, args ...any) *List {
	erroErr := Newf(message, args...)
	return g.add(erroErr)
}

// Wrap wraps an error with additional context and adds it to the list
func (g *List) Wrap(err error, message string, fields ...any) *List {
	if err == nil {
		return g.New(message, fields...)
	}
	erroErr := Wrap(err, message, fields...)
	return g.add(erroErr)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (g *List) WrapEmpty(err error) *List {
	if err == nil {
		return g
	}
	erroErr := WrapEmpty(err)
	return g.add(erroErr)
}

// Wrapf wraps an error with formatted message and adds it to the list
func (g *List) Wrapf(err error, message string, args ...any) *List {
	if err == nil {
		return g.Errorf(message, args...)
	}
	erroErr := Wrapf(err, message, args...)
	return g.add(erroErr)
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
	errorsCopy := make([]error, len(g.errors))
	for i, err := range g.errors {
		errorsCopy[i] = err
	}
	return &multiError{errors: errorsCopy}
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
		if e.Context().ID() == err.Context().ID() {
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

func (g *List) WithClass(class Class) *List {
	g.class = class
	return g
}

func (g *List) WithCategory(category Category) *List {
	g.category = category
	return g
}

func (g *List) WithSeverity(severity Severity) *List {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	g.severity = severity
	return g
}

func (g *List) WithFields(fields ...any) *List {
	g.fields = safeAppendFields(g.fields, prepareFields(fields))
	return g
}

func (g *List) WithRetryable(retryable bool) *List {
	g.retryable = retryable
	return g
}

func (g *List) Class() Class       { return g.class }
func (g *List) Category() Category { return g.category }
func (g *List) Fields() []any      { return g.fields }
func (g *List) IsRetryable() bool  { return g.retryable }
func (g *List) Severity() Severity { return g.severity }

func (g *List) add(err Error) *List {
	g.errors = append(g.errors, g.withMetadata(err))
	return g
}

// applyMetadata applies accumulated metadata to an error
func (g *List) withMetadata(err Error) Error {
	if g.class != ClassUnknown && err.Context().Class() == ClassUnknown {
		err = err.WithClass(g.class)
	}
	if g.category != CategoryUnknown && err.Context().Category() == CategoryUnknown {
		err = err.WithCategory(g.category)
	}
	if g.severity != SeverityUnknown && err.Context().Severity() == SeverityUnknown {
		err = err.WithSeverity(g.severity)
	}
	if g.retryable {
		err = err.WithRetryable(g.retryable)
	}
	if len(g.fields) > 0 {
		err = err.WithFields(g.fields...)
	}
	return err
}

// Set collects unique errors and provides the same chaining API as Error.
// It deduplicates errors based on their message and code.
type Set struct {
	*List
	seen map[string]int
	// It won't add the error if the keyGetter returns an empty string
	keyGetter KeyGetterFunc
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
	erroErr := New(message, fields...)
	return s.add(erroErr)
}

// Errorf creates a new error with formatted message and adds it to the set if unique
func (s *Set) Errorf(message string, args ...any) *Set {
	erroErr := Newf(message, args...)
	return s.add(erroErr)
}

// Wrap wraps an error with additional context and adds it to the set if unique
func (s *Set) Wrap(err error, message string, fields ...any) *Set {
	if err == nil {
		return s.New(message, fields...)
	}
	erroErr := Wrap(err, message, fields...)
	return s.add(erroErr)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (s *Set) WrapEmpty(err error) *Set {
	if err == nil {
		return s
	}
	erroErr := WrapEmpty(err)
	return s.add(erroErr)
}

// Wrapf wraps an error with formatted message and adds it to the set if unique
func (s *Set) Wrapf(err error, message string, args ...any) *Set {
	if err == nil {
		return s.Errorf(message, args...)
	}
	erroErr := Wrapf(err, message, args...)
	return s.add(erroErr)
}

// Err returns a combined error from all errors in the list, or nil if empty.
// This prevents returning a non-nil error that represents an empty list.
func (s *Set) Err() error {
	if len(s.errors) == 0 {
		return nil
	}

	// Create a copy of the errors for the multiError
	errorsCopy := make([]error, len(s.errors))
	for i, err := range s.errors {
		errorsCopy[i] = err
	}
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

// Remove removes error at index i from the list.
func (g *Set) Remove(i int) bool {
	if i < 0 || i >= len(g.errors) {
		return false
	}
	key := g.keyGetter(g.errors[i])
	if key == "" {
		return false
	}
	delete(g.seen, key)
	g.errors = append(g.errors[:i], g.errors[i+1:]...)

	return true
}

// RemoveError removes the first error that matches the given error.
func (g *Set) RemoveError(err Error) bool {
	for i, e := range g.errors {
		if e.Context().ID() == err.Context().ID() {
			g.Remove(i)
			return true
		}
	}
	return false
}

// Errors returns a copy of the errors slice
func (g *Set) Errors() []error {
	result := make([]error, len(g.errors))
	for i, err := range g.errors {
		result[i] = err
	}
	return result
}

// Errs returns a copy of the errors slice
func (g *Set) Errs() []Error {
	result := make([]Error, len(g.errors))
	copy(result, g.errors)
	return result
}

// Len returns the number of errors in the list
func (g *Set) Len() int {
	return len(g.errors)
}

// Empty returns true if the list is empty
func (g *Set) Empty() bool {
	return len(g.errors) == 0
}

// NotEmpty returns true if the list is not empty
func (g *Set) NotEmpty() bool {
	return len(g.errors) > 0
}

// First returns the first error in the list, or nil if empty.
func (g *Set) First() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Last returns the last error in the list, or nil if empty.
func (g *Set) Last() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

func (s *Set) WithClass(class Class) *Set {
	s.List.WithClass(class)
	return s
}

func (s *Set) WithCategory(category Category) *Set {
	s.List.WithCategory(category)
	return s
}

func (s *Set) WithSeverity(severity Severity) *Set {
	s.List.WithSeverity(severity)
	return s
}

func (s *Set) WithFields(fields ...any) *Set {
	s.List.WithFields(fields...)
	return s
}

func (s *Set) WithRetryable(retryable bool) *Set {
	s.List.WithRetryable(retryable)
	return s
}

func (s *Set) Class() Class       { return s.List.Class() }
func (s *Set) Category() Category { return s.List.Category() }
func (s *Set) Fields() []any      { return s.List.Fields() }
func (s *Set) IsRetryable() bool  { return s.List.IsRetryable() }
func (s *Set) Severity() Severity { return s.List.Severity() }

func (s *Set) add(err Error) *Set {
	err = s.withMetadata(err)
	key := s.keyGetter(err)
	if key == "" {
		return s
	}
	if _, ok := s.seen[key]; !ok {
		s.seen[key] = 1
		s.errors = append(s.errors, err)
	} else {
		s.seen[key]++
	}
	return s
}

// SafeList is a thread-safe version of List that can be used safely across multiple goroutines
type SafeList struct {
	errors []Error
	// Metadata that will be applied to errors added to this list
	class     Class
	category  Category
	severity  Severity
	fields    []any
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
	erroErr := New(message, fields...)
	return g.add(erroErr)
}

// Errorf creates a new error with formatted message and adds it to the list
func (g *SafeList) Errorf(message string, args ...any) *SafeList {
	erroErr := Newf(message, args...)
	return g.add(erroErr)
}

// Wrap wraps an error with additional context and adds it to the list
func (g *SafeList) Wrap(err error, message string, fields ...any) *SafeList {
	if err == nil {
		return g.New(message, fields...)
	}
	erroErr := Wrap(err, message, fields...)
	return g.add(erroErr)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (g *SafeList) WrapEmpty(err error) *SafeList {
	if err == nil {
		return g
	}
	erroErr := WrapEmpty(err)
	return g.add(erroErr)
}

// Wrapf wraps an error with formatted message and adds it to the list
func (g *SafeList) Wrapf(err error, message string, args ...any) *SafeList {
	if err == nil {
		return g.Errorf(message, args...)
	}
	erroErr := Wrapf(err, message, args...)
	return g.add(erroErr)
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
	errorsCopy := make([]error, len(g.errors))
	for i, err := range g.errors {
		errorsCopy[i] = err
	}
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
		if e.Context().ID() == err.Context().ID() {
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

func (g *SafeList) WithClass(class Class) *SafeList {
	g.mu.Lock()
	g.class = class
	g.mu.Unlock()
	return g
}

func (g *SafeList) WithCategory(category Category) *SafeList {
	g.mu.Lock()
	g.category = category
	g.mu.Unlock()
	return g
}

func (g *SafeList) WithSeverity(severity Severity) *SafeList {
	if !severity.IsValid() {
		severity = SeverityUnknown
	}
	g.mu.Lock()
	g.severity = severity
	g.mu.Unlock()
	return g
}

func (g *SafeList) WithFields(fields ...any) *SafeList {
	g.mu.Lock()
	g.fields = safeAppendFields(g.fields, prepareFields(fields))
	g.mu.Unlock()
	return g
}

func (g *SafeList) WithRetryable(retryable bool) *SafeList {
	g.mu.Lock()
	g.retryable = retryable
	g.mu.Unlock()
	return g
}

func (g *SafeList) Class() Class {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.class
}
func (g *SafeList) Category() Category {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.category
}
func (g *SafeList) Fields() []any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.fields
}
func (g *SafeList) IsRetryable() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.retryable
}
func (g *SafeList) Severity() Severity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.severity
}

func (g *SafeList) add(err Error) *SafeList {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.errors = append(g.errors, g.withMetadata(err))
	return g
}

// applyMetadata applies accumulated metadata to an error
func (g *SafeList) withMetadata(err Error) Error {
	if g.class != ClassUnknown && err.Context().Class() == ClassUnknown {
		err = err.WithClass(g.class)
	}
	if g.category != CategoryUnknown && err.Context().Category() == CategoryUnknown {
		err = err.WithCategory(g.category)
	}
	if g.severity != SeverityUnknown && err.Context().Severity() == SeverityUnknown {
		err = err.WithSeverity(g.severity)
	}
	if len(g.fields) > 0 {
		err = err.WithFields(g.fields...)
	}
	if g.retryable {
		err = err.WithRetryable(g.retryable)
	}
	return err
}

// SafeSet is a thread-safe version of Set that collects unique errors
type SafeSet struct {
	*List
	seen map[string]int
	// It won't add the error if the keyGetter returns an empty string
	keyGetter KeyGetterFunc
	mu        sync.RWMutex
}

// NewSafeSet creates a new thread-safe error set that stores only unique errors
func NewSafeSet(capacityRaw ...int) *SafeSet {
	var capacity int
	if len(capacityRaw) > 0 {
		capacity = capacityRaw[0]
	}
	return &SafeSet{
		List:      NewList(capacity),
		seen:      make(map[string]int, capacity),
		keyGetter: MessageKeyGetter,
	}
}

func (s *SafeSet) WithKeyGetter(keyGetter func(error) string) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	erroErr := New(message, fields...)
	return s.add(erroErr)
}

// Errorf creates a new error with formatted message and adds it to the set if unique
func (s *SafeSet) Errorf(message string, args ...any) *SafeSet {
	erroErr := Newf(message, args...)
	return s.add(erroErr)
}

// Wrap wraps an error with additional context and adds it to the set if unique
func (s *SafeSet) Wrap(err error, message string, fields ...any) *SafeSet {
	if err == nil {
		return s.New(message, fields...)
	}
	erroErr := Wrap(err, message, fields...)
	return s.add(erroErr)
}

// WrapEmpty wraps an error without a message to create an erro.Error from it.
func (s *SafeSet) WrapEmpty(err error) *SafeSet {
	if err == nil {
		return s
	}
	erroErr := WrapEmpty(err)
	return s.add(erroErr)
}

// Wrapf wraps an error with formatted message and adds it to the set if unique
func (s *SafeSet) Wrapf(err error, message string, args ...any) *SafeSet {
	if err == nil {
		return s.Errorf(message, args...)
	}
	erroErr := Wrapf(err, message, args...)
	return s.add(erroErr)
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
	errorsCopy := make([]error, len(s.errors))
	for i, err := range s.errors {
		errorsCopy[i] = err
	}
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
		List:      s.List.Copy(),
		seen:      newSeen,
		keyGetter: s.keyGetter,
	}
}

// Remove removes error at index i from the list.
func (g *SafeSet) Remove(i int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if i < 0 || i >= len(g.errors) {
		return false
	}
	err := g.errors[i]
	key := g.keyGetter(err)
	if key == "" {
		return false
	}
	delete(g.seen, key)
	g.errors = append(g.errors[:i], g.errors[i+1:]...)

	return true
}

// RemoveError removes the first error that matches the given error.
func (g *SafeSet) RemoveError(err Error) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, e := range g.errors {
		if e.Context().ID() == err.Context().ID() {
			g.Remove(i)
			return true
		}
	}
	return false
}

// Errors returns a copy of the errors slice
func (g *SafeSet) Errors() []error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]error, len(g.errors))
	for i, err := range g.errors {
		result[i] = err
	}
	return result
}

// Errs returns a copy of the errors slice
func (g *SafeSet) Errs() []Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]Error, len(g.errors))
	copy(result, g.errors)
	return result
}

// Len returns the number of errors in the list
func (g *SafeSet) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.errors)
}

// Empty returns true if the list is empty
func (g *SafeSet) Empty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.errors) == 0
}

// NotEmpty returns true if the list is not empty
func (g *SafeSet) NotEmpty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.errors) > 0
}

// First returns the first error in the list, or nil if empty.
func (g *SafeSet) First() Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Last returns the last error in the list, or nil if empty.
func (g *SafeSet) Last() Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

func (s *SafeSet) WithClass(class Class) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.class = class
	return s
}

func (s *SafeSet) WithCategory(category Category) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.category = category
	return s
}

func (s *SafeSet) WithSeverity(severity Severity) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.severity = severity
	return s
}

func (s *SafeSet) WithFields(fields ...any) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fields = safeAppendFields(s.fields, prepareFields(fields))
	return s
}

func (s *SafeSet) WithRetryable(retryable bool) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.retryable = retryable
	return s
}

func (s *SafeSet) Class() Class {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.class
}
func (s *SafeSet) Category() Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.category
}
func (s *SafeSet) Fields() []any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fields
}
func (s *SafeSet) IsRetryable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.retryable
}
func (s *SafeSet) Severity() Severity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.severity
}

func (s *SafeSet) add(err Error) *SafeSet {
	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.withMetadata(err)
	key := s.keyGetter(err)
	if key == "" {
		return s
	}
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
	errors []error
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
	// Estimate capacity: prefix + count + each error (approx 100 chars) + separators
	estimatedSize := 50 + len(m.errors)*100
	builder.Grow(estimatedSize)

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
	return m.errors
}

// multiError represents multiple errors combined into one
type multiErrorSet struct {
	errors    []error
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
	// Estimate capacity: prefix + count + each error (approx 100 chars) + counters + separators
	estimatedSize := 50 + len(m.errors)*120
	builder.Grow(estimatedSize)

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
	return m.errors
}
