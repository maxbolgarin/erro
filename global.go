package erro

import (
	"runtime/debug"
	"sync/atomic"
)

// Global stack trace configuration
var (
	globalStackTraceConfig atomic.Value
	globalFormatter        atomic.Value

	buildInfo *debug.BuildInfo
)

func init() {
	globalStackTraceConfig.Store(DevelopmentStackTraceConfig())
	globalFormatter.Store(&formatterObject{formatter: FormatErrorWithFields})
	buildInfo, _ = debug.ReadBuildInfo()
}

// SetGlobalStackTraceConfig sets the global stack trace configuration
func SetGlobalStackTraceConfig(config *StackTraceConfig) {
	if config == nil {
		config = NoStackTraceConfig()
	}
	globalStackTraceConfig.Store(config)
}

// GetGlobalStackTraceConfig returns the current global stack trace configuration
func GetGlobalStackTraceConfig() *StackTraceConfig {
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

// SetGlobalStackSamplingRate sets the rate at which stack traces are captured (0.0 - 1.0)
func SetGlobalStackSamplingRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	for {
		oldCfgPtr := GetGlobalStackTraceConfig()
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

// SetGlobalFormatter sets the global error formatter.
func SetGlobalFormatter(formatter FormatErrorFunc) {
	globalFormatter.Store(&formatterObject{formatter: formatter})
}

// GetGlobalFormatter returns the global error formatter.
func GetGlobalFormatter() FormatErrorFunc {
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
