package erro

import (
	"strconv"
	"strings"
	"sync"
)

// --- Base Implementation: List ---

// List collects multiple errors and provides a fluent API for adding and configuring them.
// It is not thread-safe.
type List struct {
	errors []Error
}

// NewList creates a new error list.
func NewList(capacity ...int) *List {
	var c int
	if len(capacity) > 0 {
		c = capacity[0]
	}
	return &List{
		errors: make([]Error, 0, c),
	}
}

// add is the internal method for appending an error.
func (g *List) add(err Error) {
	g.errors = append(g.errors, err)
}

// Add adds an error to the list, converting it to an erro.Error if necessary.
func (g *List) Add(err error) *List {
	if err == nil {
		return g
	}
	g.add(ExtractError(err))
	return g
}

// New creates a new error and adds it to the list.
func (g *List) New(message string, meta ...any) *List {
	return addNew(g, message, meta...)
}

// Wrap wraps an existing error and adds it to the list.
func (g *List) Wrap(err error, message string, meta ...any) *List {
	return addWrap(g, err, message, meta...)
}

// Err returns a combined error from all errors in the list, or nil if empty.
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

// Remove removes an error at index i from the list.
func (g *List) Remove(i int) bool {
	if i < 0 || i >= len(g.errors) {
		return false
	}
	g.errors = append(g.errors[:i], g.errors[i+1:]...)
	return true
}

// RemoveError removes the first error that matches the given error by ID.
func (g *List) RemoveError(err Error) bool {
	if err == nil {
		return false
	}
	id := err.ID()
	if id == "" {
		return false
	}
	for i, e := range g.errors {
		if e.ID() == id {
			return g.Remove(i)
		}
	}
	return false
}

// Clear removes all errors from the list.
func (g *List) Clear() *List {
	g.errors = make([]Error, 0, cap(g.errors))
	return g
}

// Copy returns a shallow copy of the list.
func (g *List) Copy() *List {
	clone := NewList(cap(g.errors))
	clone.errors = append(make([]Error, 0, len(g.errors)), g.errors...)
	return clone
}

// --- List Accessors ---
func (g *List) Errors() []error {
	result := make([]error, len(g.errors))
	for i, err := range g.errors {
		result[i] = err
	}
	return result
}
func (g *List) Errs() []Error  { return g.errors }
func (g *List) Len() int       { return len(g.errors) }
func (g *List) Empty() bool    { return len(g.errors) == 0 }
func (g *List) NotEmpty() bool { return len(g.errors) > 0 }
func (g *List) First() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}
func (g *List) Last() Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

// --- Deduplicating Implementation: Set ---

// Set collects unique errors, deduplicating them based on a configurable key.
// It is not thread-safe.
type Set struct {
	*List
	seen      map[string]int
	keyGetter KeyGetterFunc
}

// NewSet creates a new error set that stores only unique errors.
func NewSet(capacity ...int) *Set {
	return &Set{
		List:      NewList(capacity...),
		seen:      make(map[string]int),
		keyGetter: MessageKeyGetter,
	}
}

// add overrides the embedded List's add method to provide deduplication.
func (s *Set) add(err Error) {
	key := s.keyGetter(err)
	if key == "" {
		// Do not add errors that produce an empty key.
		return
	}
	if count, ok := s.seen[key]; ok {
		s.seen[key] = count + 1
	} else {
		s.seen[key] = 1
		s.List.errors = append(s.List.errors, err)
	}
}

// --- Set Creator Methods (for fluent API) ---
func (s *Set) Add(err error) *Set {
	if err == nil {
		return s
	}
	s.add(ExtractError(err))
	return s
}
func (s *Set) New(message string, meta ...any) *Set {
	return addNew(s, message, meta...)
}
func (s *Set) Wrap(err error, message string, meta ...any) *Set {
	return addWrap(s, err, message, meta...)
}

// --- Set Overridden Methods ---
func (s *Set) WithKeyGetter(keyGetter KeyGetterFunc) *Set {
	if keyGetter != nil {
		s.keyGetter = keyGetter
	}
	return s
}

// Err returns a combined error that includes deduplication counts.
func (s *Set) Err() error {
	if s.Len() == 0 {
		return nil
	}
	if s.Len() == 1 {
		return s.First()
	}
	errorsCopy := make([]error, s.Len())
	copy(errorsCopy, s.Errors())
	return &multiErrorSet{errors: errorsCopy, counter: s.seen, keyGetter: s.keyGetter}
}

// Clear removes all errors and resets the deduplication map.
func (s *Set) Clear() *Set {
	s.List.Clear()
	s.seen = make(map[string]int, cap(s.List.errors))
	return s
}

// Copy returns a shallow copy of the set.
func (s *Set) Copy() *Set {
	clone := NewSet(cap(s.List.errors))
	clone.List = s.List.Copy()
	clone.keyGetter = s.keyGetter
	for k, v := range s.seen {
		clone.seen[k] = v
	}
	return clone
}

// Remove removes an error and its key from the seen map.
func (s *Set) Remove(i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	err := s.Errs()[i]
	if s.List.Remove(i) {
		key := s.keyGetter(err)
		if key != "" {
			delete(s.seen, key)
		}
		return true
	}
	return false
}

// RemoveError removes an error by its instance and its key from the seen map.
func (s *Set) RemoveError(err Error) bool {
	if err == nil {
		return false
	}
	id := err.ID()
	if id == "" {
		return false
	}
	for i, e := range s.Errs() {
		if e.ID() == id {
			return s.Remove(i)
		}
	}
	return false
}

// --- Thread-Safe Wrapper: SafeList ---

// SafeList is a thread-safe version of List.
type SafeList struct {
	mu   sync.RWMutex
	list *List
}

// NewSafeList creates a new thread-safe error list.
func NewSafeList(capacity ...int) *SafeList {
	return &SafeList{list: NewList(capacity...)}
}

func (sl *SafeList) Add(err error) *SafeList {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.list.Add(err)
	return sl
}
func (sl *SafeList) New(message string, meta ...any) *SafeList {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.list.New(message, meta...)
	return sl
}
func (sl *SafeList) Wrap(err error, message string, meta ...any) *SafeList {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.list.Wrap(err, message, meta...)
	return sl
}
func (sl *SafeList) Err() error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Err()
}
func (sl *SafeList) Remove(i int) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.list.Remove(i)
}
func (sl *SafeList) RemoveError(err Error) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.list.RemoveError(err)
}
func (sl *SafeList) Clear() *SafeList {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.list.Clear()
	return sl
}
func (sl *SafeList) Copy() *SafeList {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return &SafeList{list: sl.list.Copy()}
}
func (sl *SafeList) Errors() []error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Errors()
}
func (sl *SafeList) Errs() []Error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Errs()
}
func (sl *SafeList) Len() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Len()
}
func (sl *SafeList) Empty() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Empty()
}
func (sl *SafeList) NotEmpty() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.NotEmpty()
}
func (sl *SafeList) First() Error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.First()
}
func (sl *SafeList) Last() Error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.list.Last()
}

// --- Thread-Safe Wrapper: SafeSet ---

// SafeSet is a thread-safe version of Set.
type SafeSet struct {
	mu  sync.RWMutex
	set *Set
}

// NewSafeSet creates a new thread-safe error set.
func NewSafeSet(capacity ...int) *SafeSet {
	return &SafeSet{set: NewSet(capacity...)}
}

func (ss *SafeSet) Add(err error) *SafeSet {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.set.Add(err)
	return ss
}
func (ss *SafeSet) New(message string, meta ...any) *SafeSet {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.set.New(message, meta...)
	return ss
}
func (ss *SafeSet) Wrap(err error, message string, meta ...any) *SafeSet {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.set.Wrap(err, message, meta...)
	return ss
}
func (ss *SafeSet) Err() error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Err()
}
func (ss *SafeSet) Remove(i int) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.set.Remove(i)
}
func (ss *SafeSet) RemoveError(err Error) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.set.RemoveError(err)
}
func (ss *SafeSet) Clear() *SafeSet {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.set.Clear()
	return ss
}
func (ss *SafeSet) Copy() *SafeSet {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return &SafeSet{set: ss.set.Copy()}
}
func (ss *SafeSet) Errors() []error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Errors()
}
func (ss *SafeSet) Errs() []Error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Errs()
}
func (ss *SafeSet) Len() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Len()
}
func (ss *SafeSet) Empty() bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Empty()
}
func (ss *SafeSet) NotEmpty() bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.NotEmpty()
}
func (ss *SafeSet) First() Error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.First()
}
func (ss *SafeSet) Last() Error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.set.Last()
}
func (ss *SafeSet) WithKeyGetter(keyGetter KeyGetterFunc) *SafeSet {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.set.WithKeyGetter(keyGetter)
	return ss
}

// --- Multi-Error Types ---

// multiError represents multiple errors combined into one.
// It is compatible with Go 1.20's multi-error unwrapping.
type multiError struct {
	errors []error
}

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
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString("(")
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(") ")
		builder.WriteString(err.Error())
	}
	return builder.String()
}

// Unwrap returns the underlying errors for error chain traversal.
func (m *multiError) Unwrap() []error {
	return m.errors
}

// multiErrorSet is the error type returned by a Set, including deduplication counts.
type multiErrorSet struct {
	errors    []error
	counter   map[string]int
	keyGetter func(error) string
}

func (m *multiErrorSet) Error() string {
	if len(m.errors) == 0 {
		return ""
	}
	if len(m.errors) == 1 {
		return m.errors[0].Error()
	}

	var builder strings.Builder
	builder.WriteString("multiple unique errors (")
	builder.WriteString(strconv.Itoa(len(m.errors)))
	builder.WriteString("): ")
	for i, err := range m.errors {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString("(")
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteString(") ")
		builder.WriteString(err.Error())
		if count, ok := m.counter[m.keyGetter(err)]; ok && count > 1 {
			builder.WriteString(" [")
			builder.WriteString(strconv.Itoa(count))
			builder.WriteString(" times]")
		}
	}
	return builder.String()
}

// Unwrap returns the underlying errors for error chain traversal.
func (m *multiErrorSet) Unwrap() []error {
	return m.errors
}

// Errorf creates a new formatted error and adds it to the list.
func addNew[T interface{ add(Error) }](g T, message string, meta ...any) T {
	g.add(newf(message, meta...))
	return g
}

// Wrap wraps an existing error and adds it to the list.
func addWrap[T interface{ add(Error) }](g T, err error, message string, meta ...any) T {
	g.add(wrapf(err, message, meta...))
	return g
}
