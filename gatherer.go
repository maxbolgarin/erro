package erro

import (
	"sync"
	"sync/atomic"
)

// ErrorGatherer accumulates errors that occur in the application.
type ErrorGatherer struct {
	enabled   atomic.Bool
	errors    []Error
	mu        sync.RWMutex
	seen      map[string]bool
	keyGetter func(Error) string
}

// Key getter functions for deduplication
var (
	// MessageKeyGetter generates a key based on the error's message.
	GathererMessageKeyGetter = func(err Error) string {
		return err.Context().Message()
	}
	// IDKeyGetter generates a key based on the error's ID.
	GathererIDKeyGetter = func(err Error) string {
		return err.Context().ID()
	}
	// ErrorKeyGetter generates a key based on the error's class.
	GathererErrorKeyGetter = func(err Error) string {
		return err.Error()
	}
)

var globalGatherer = &ErrorGatherer{
	seen:      make(map[string]bool),
	keyGetter: GathererMessageKeyGetter, // Default to message key getter
}

// SetGathererKeyGetter sets the key getter function for deduplication.
func SetGathererKeyGetter(getter func(Error) string) {
	globalGatherer.mu.Lock()
	defer globalGatherer.mu.Unlock()
	globalGatherer.keyGetter = getter
}

// EnableGatherer enables the global error gatherer.
func EnableGatherer() {
	globalGatherer.enabled.Store(true)
}

// DisableGatherer disables the global error gatherer.
func DisableGatherer() {
	globalGatherer.enabled.Store(false)
}

// GathererEnabled returns true if the global error gatherer is enabled.
func GathererEnabled() bool {
	return globalGatherer.enabled.Load()
}

// AddToGatherer adds an error to the global gatherer if it is enabled.
func AddToGatherer(err Error) {
	if !globalGatherer.enabled.Load() || err == nil {
		return
	}
	globalGatherer.add(err)
}

// GetGatheredErrors returns a copy of the errors accumulated by the global gatherer.
func GetGatheredErrors() []Error {
	return globalGatherer.getErrors()
}

// ClearGatheredErrors clears the errors accumulated by the global gatherer.
func ClearGatheredErrors() {
	globalGatherer.clear()
}

func (g *ErrorGatherer) add(err Error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.keyGetter != nil {
		key := g.keyGetter(err)
		if _, ok := g.seen[key]; ok {
			return // Deduplicate
		}
		g.seen[key] = true
	}

	g.errors = append(g.errors, err)
}

func (g *ErrorGatherer) getErrors() []Error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// Return a copy to prevent race conditions on the slice.
	errs := make([]Error, len(g.errors))
	copy(errs, g.errors)
	return errs
}

func (g *ErrorGatherer) clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errors = nil
	g.seen = make(map[string]bool)
}
