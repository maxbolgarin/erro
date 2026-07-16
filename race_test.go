//go:build race

package erro_test

// timingMultiplier relaxes wall-clock assertions in performance-sensitive
// tests when the race detector is enabled, which slows execution by an
// order of magnitude.
const timingMultiplier = 20
