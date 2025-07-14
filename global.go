package erro

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Global stack trace configuration
var (
	globalStackTraceConfig atomic.Value
	globalFormatter        atomic.Value
	globalGatherer         = &errorGatherer{
		seen: make(map[string]time.Time),
	}

	buildInfo *debug.BuildInfo
)

func init() {
	globalStackTraceConfig.Store(DevelopmentStackTraceConfig())
	globalFormatter.Store(&formatterObject{formatter: FormatErrorWithFields})
	buildInfo, _ = debug.ReadBuildInfo()
}

// SetDefaultStackTraceConfig sets the global stack trace configuration
func SetDefaultStackTraceConfig(config *StackTraceConfig) {
	if config == nil {
		config = NoStackTraceConfig()
	}
	globalStackTraceConfig.Store(config)
}

// GetDefaultStackTraceConfig returns the current global stack trace configuration
func GetDefaultStackTraceConfig() *StackTraceConfig {
	cfgRaw := globalStackTraceConfig.Load()
	if cfgRaw == nil {
		return DevelopmentStackTraceConfig()
	}
	cfg, ok := cfgRaw.(*StackTraceConfig)
	if !ok {
		return DevelopmentStackTraceConfig()
	}
	return cfg
}

// SetDefaultStackSamplingRate sets the rate at which stack traces are captured (0.0 - 1.0)
func SetDefaultStackSamplingRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	for {
		oldCfgPtr := GetDefaultStackTraceConfig()
		newCfg := *oldCfgPtr
		newCfg.SamplingRate = rate

		// Atomically swap if the config hasn't changed
		if globalStackTraceConfig.CompareAndSwap(oldCfgPtr, &newCfg) {
			return
		}
	}
}

type formatterObject struct {
	formatter FormatErrorFunc
}

// SetDefaultFormatter sets the global error formatter.
func SetDefaultFormatter(formatter FormatErrorFunc) {
	globalFormatter.Store(&formatterObject{formatter: formatter})
}

// GetDefaultFormatter returns the global error formatter.
func GetDefaultFormatter() FormatErrorFunc {
	res := globalFormatter.Load()
	if res == nil {
		return FormatErrorWithFields
	}
	f, ok := res.(*formatterObject)
	if !ok {
		return FormatErrorWithFields
	}
	return f.formatter
}

// errorGatherer accumulates errors that occur in the application.
type errorGatherer struct {
	seen map[string]time.Time

	metrics    Metrics
	dispatcher Dispatcher

	enabled atomic.Bool
	mu      sync.Mutex
}

// EnableAutoErrorGatherer enables the global error gatherer.
func EnableAutoErrorGatherer(metrics Metrics, dispatcher Dispatcher) {
	globalGatherer.mu.Lock()
	defer globalGatherer.mu.Unlock()

	globalGatherer.enabled.Store(true)
	globalGatherer.metrics = metrics
	globalGatherer.dispatcher = dispatcher
}

func (g *errorGatherer) add(ctx context.Context, err Error) {
	if !g.enabled.Load() || err == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	key := IDKeyGetter(err)
	if _, ok := g.seen[key]; ok {
		g.seen[key] = time.Now()
		return
	}
	g.seen[key] = time.Now()

	g.dispatcher.SendEvent(ctx, err.Context())
	g.metrics.RecordError(err.Context())

	for key, t := range g.seen {
		if time.Since(t) > 10*time.Minute {
			delete(g.seen, key)
		}
	}
}
