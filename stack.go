package erro

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	stdlibPrefixes = []string{
		"runtime.", "testing.", "fmt.", "strings.", "strconv.", "time.",
		"context.", "sync.", "os.", "io.", "net.", "crypto.", "encoding.",
		"reflect.", "sort.", "math.", "unicode.", "errors.", "slog.", "http.",
	}
	internalFuncs = []string{
		"newBaseError", "captureStack", "CaptureStack", "newError",
		"buildErrorMessage", "buildFieldsMessage", "validateFields", "prepareFields",
		"New", "Wrap", "Wrapf", "Errorf", "extractPackage",
	}
	uselessFrames = []string{
		"runtime.main",        // Runtime's main function (not user's main)
		"runtime.goexit",      // Goroutine exit function
		"runtime.deferreturn", // Defer cleanup
	}
)

const (
	defaultFunctionRedacted = "[some_function]"
	defaultFileNameRedacted = "[some_file]"
	defaultStackRedacted    = "[disabled]"
	defaultHiddenFrame      = "[hidden]"
)

// SetStackSamplingRate sets the rate at which stack traces are captured (0.0 - 1.0)
func SetStackSamplingRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	cfgMutex.Lock()
	globalStackTraceConfig.SamplingRate = rate
	cfgMutex.Unlock()
}

// StackTraceConfig controls what information is included in stack traces
type StackTraceConfig struct {
	Enabled       bool // Whether to show stack traces
	ShowFileNames bool // Whether to show file names
	ShowFullPaths bool // Whether to show file paths
	PathElements  int  // Number of path elements to include (0 = filename only, -1 = full path)

	ShowFunctionNames bool // Whether to show function names
	ShowPackageNames  bool // Whether to show package names
	ShowLineNumbers   bool // Whether to show line numbers
	ShowAllCodeFrames bool // Whether to show all types of frames (user, stdlib, etc.)

	FunctionRedacted string // Placeholder for redacted function names (default: "[some_function]")
	FileNameRedacted string // Placeholder for redacted file names (default: "[some_file]")

	MaxFrames int // Maximum number of frames to show

	SamplingRate    float64 // Sampling rate for stack traces (0.0 - 1.0)
	samplingCounter uint64
}

// DevelopmentStackTraceConfig returns the development-safe stack trace configuration
func DevelopmentStackTraceConfig() *StackTraceConfig {
	return &StackTraceConfig{
		Enabled:           true,
		ShowFileNames:     true,
		ShowFullPaths:     true,
		ShowFunctionNames: true,
		ShowPackageNames:  true,
		ShowLineNumbers:   true,
		ShowAllCodeFrames: true,
		PathElements:      -1, // Show full path
		SamplingRate:      1.0,
	}
}

// ProductionStackTraceConfig returns a production-safe stack trace configuration
func ProductionStackTraceConfig() *StackTraceConfig {
	return &StackTraceConfig{
		Enabled:       true,
		ShowFileNames: true,
		ShowFullPaths: false, // Hide full paths, show only filenames
		PathElements:  2,     // Show 2 path elements from project root (e.g., "examples/privacy/main.go")

		ShowFunctionNames: true,  // Hide function names for security
		ShowPackageNames:  false, // Hide package names for security
		ShowLineNumbers:   true,  // Show line numbers for debugging
		ShowAllCodeFrames: false,

		MaxFrames:    10,
		SamplingRate: 1.0,
	}
}

// StrictStackTraceConfig returns a strict privacy stack trace configuration
func StrictStackTraceConfig() *StackTraceConfig {
	return &StackTraceConfig{
		Enabled:       true,
		ShowFileNames: true,  // Hide all file names
		ShowFullPaths: false, // Hide all path information
		PathElements:  0,     // Show only filename

		ShowFunctionNames: false, // Hide all function names
		ShowPackageNames:  false, // Hide all package names
		ShowLineNumbers:   true,  // Show line numbers for debugging
		ShowAllCodeFrames: true,

		MaxFrames: 3, // Very limited frames for strict mode

		SamplingRate: 1.0,
	}
}

// NoStackTraceConfig returns a configuration that completely disables stack traces
func NoStackTraceConfig() *StackTraceConfig {
	return &StackTraceConfig{}
}

// Global stack trace configuration
var (
	globalStackTraceConfig = DevelopmentStackTraceConfig()
	cfgMutex               sync.Mutex
)

// SetStackTraceConfig sets the global stack trace configuration
func SetStackTraceConfig(config *StackTraceConfig) {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	globalStackTraceConfig = config
}

// GetStackTraceConfig returns the current global stack trace configuration
func GetStackTraceConfig() *StackTraceConfig {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	return globalStackTraceConfig
}

// SetDevelopmentStackTrace enables development-safe stack trace configuration
func SetDevelopmentStackTrace() {
	SetStackTraceConfig(DevelopmentStackTraceConfig())
}

// SetProductionStackTrace enables production-safe stack trace configuration
func SetProductionStackTrace() {
	SetStackTraceConfig(ProductionStackTraceConfig())
}

// SetStrictStackTrace enables strict privacy stack trace configuration
func SetStrictStackTrace() {
	SetStackTraceConfig(StrictStackTraceConfig())
}

// DisableStackTrace completely disables stack traces
func DisableStackTrace() {
	SetStackTraceConfig(NoStackTraceConfig())
}

// StackFrame stores a frame's runtime information in a human readable format
// Enhanced with additional context for better error diagnostics
type StackFrame struct {
	Name     string // Function name (e.g., "processPayment")
	FullName string // Full function name (e.g., "github.com/app/payment.processPayment")
	Package  string // Package name (e.g., "payment")
	File     string // Full file path
	FileName string // Just the filename (e.g., "payment.go")
	Line     int    // Line number

	StackTraceConfig *StackTraceConfig
}

// String returns a formatted representation of the stack frame
func (f StackFrame) String() string {
	if f.StackTraceConfig == nil {
		return f.Name + " (" + f.FileName + ":" + strconv.Itoa(f.Line) + ")"
	}
	if !f.StackTraceConfig.ShowAllCodeFrames && !f.IsUser() {
		return defaultHiddenFrame
	}

	var line strings.Builder
	line.Grow(len(f.FileName) + len(f.Name) + 10)

	line.WriteString(f.getFunctionName())
	if f.StackTraceConfig.ShowFileNames {
		line.WriteString(" (" + f.getFileName() + ")")
	}

	return line.String()
}

// FormatFull returns a detailed formatted stack frame
func (f StackFrame) FormatFull() string {
	if f.StackTraceConfig == nil {
		return fmt.Sprintf("\t%s\n\t\t%s:%d", f.FullName, f.File, f.Line)
	}
	if !f.StackTraceConfig.ShowAllCodeFrames && !f.IsUser() {
		return "\t" + defaultHiddenFrame
	}

	var line strings.Builder
	line.Grow(len(f.FileName) + len(f.Name) + 10)

	line.WriteString("\t" + f.getFunctionName())
	if f.StackTraceConfig.ShowFileNames {
		line.WriteString("\n\t\t" + f.getFileName())
	}

	return line.String()
}

// ToJSON returns a JSON-friendly representation of the stack frame
func (f StackFrame) ToJSON() map[string]any {
	return map[string]any{
		"function": f.getFunctionName(),
		"file":     f.getFileName(),
		"line":     strconv.Itoa(f.Line),
		"type":     f.getFrameType(),
	}
}

// IsUser returns true if this frame represents user code (not runtime/stdlib/erro internal)
func (f StackFrame) IsUser() bool {
	return !f.IsRuntime() && !f.IsStandardLibrary() && !f.IsErroInternal()
}

// IsRuntime returns true if this frame is from Go runtime
func (f StackFrame) IsRuntime() bool {
	return strings.HasPrefix(f.FullName, "runtime.") ||
		strings.HasPrefix(f.Package, "runtime") ||
		strings.Contains(f.File, "runtime/")
}

// IsStandardLibrary returns true if this frame is from Go standard library
func (f StackFrame) IsStandardLibrary() bool {
	// Check if the full function name indicates standard library
	if f.FullName == "" {
		return false
	}

	// Standard library packages don't have domain-like paths (no dots before slashes)
	// Examples: fmt.Printf, strings.Contains, testing.tRunner
	// Non-stdlib: github.com/user/repo.Function, example.com/pkg.Function

	// If it contains a domain (has dot before first slash), it's not stdlib
	if idx := strings.Index(f.FullName, "/"); idx > 0 {
		beforeSlash := f.FullName[:idx]
		if strings.Contains(beforeSlash, ".") {
			return false // Has domain, not stdlib
		}
	}

	// If it starts with known stdlib prefixes
	for _, prefix := range stdlibPrefixes {
		if strings.HasPrefix(f.FullName, prefix) {
			return true
		}
	}

	return false
}

// IsTest returns true if this frame is from test code
func (f StackFrame) IsTest() bool {
	return strings.HasSuffix(f.FileName, "_test.go") ||
		strings.Contains(f.Name, "Test") ||
		strings.Contains(f.File, "testing/")
}

// IsErroInternal returns true if this frame is from erro internal functions
func (f StackFrame) IsErroInternal() bool {
	// Filter out erro internal functions
	for _, internal := range internalFuncs {
		if f.Name == internal {
			return true
		}
	}

	// Also filter by package - if it's in github.com/maxbolgarin/erro and not test code
	return strings.Contains(f.FullName, "github.com/maxbolgarin/erro") && !f.IsTest()
}

// getFrameType returns the type of this stack frame
func (f StackFrame) getFrameType() string {
	if f.IsRuntime() {
		return "runtime"
	}
	if f.IsStandardLibrary() {
		return "stdlib"
	}
	if f.IsTest() {
		return "test"
	}
	return "user"
}

func (f StackFrame) getFunctionName() string {
	if f.StackTraceConfig == nil || !f.StackTraceConfig.Enabled {
		return f.FullName
	}
	if f.StackTraceConfig.ShowFunctionNames {
		if f.StackTraceConfig.ShowPackageNames {
			return f.FullName
		}
		return f.Name
	}
	if f.StackTraceConfig.FunctionRedacted != "" {
		return f.StackTraceConfig.FunctionRedacted
	}
	return defaultFunctionRedacted
}

func (f StackFrame) getFileName() string {
	if f.StackTraceConfig == nil || !f.StackTraceConfig.Enabled {
		return f.File
	}

	fileName := f.File

	if f.StackTraceConfig.ShowFileNames {
		if !f.StackTraceConfig.ShowFullPaths {
			fileName = extractPathElements(f.FileName, f.StackTraceConfig.PathElements)
		}
		if f.StackTraceConfig.ShowLineNumbers {
			return fileName + ":" + strconv.Itoa(f.Line)
		}
	}
	if f.StackTraceConfig.FileNameRedacted != "" {
		return f.StackTraceConfig.FileNameRedacted
	}
	return defaultFileNameRedacted
}

// Stack represents a collection of stack frames with enhanced analysis capabilities
type Stack []StackFrame

// String returns a formatted string representation of the entire stack
func (s Stack) String() string {
	var builder strings.Builder
	// Estimate capacity: approximate 50 chars per frame + separators
	estimatedSize := len(s) * 50
	if estimatedSize > 0 {
		builder.Grow(estimatedSize)
	}

	for i, frame := range s {
		if i > 0 {
			builder.WriteString(" -> ")
		}
		builder.WriteString(frame.String())
	}
	return builder.String()
}

// FormatFull returns detailed formatted stack trace
func (s Stack) FormatFull() string {
	var builder strings.Builder
	// Estimate capacity: approximate 100 chars per frame for detailed formatting
	estimatedSize := len(s) * 100
	if estimatedSize > 0 {
		builder.Grow(estimatedSize)
	}

	for i, frame := range s {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(frame.FormatFull())
	}
	return builder.String()
}

// ToJSON returns a JSON-friendly representation of the stack
func (s Stack) ToJSON() []map[string]any {
	frames := make([]map[string]any, len(s))
	for i, frame := range s {
		frames[i] = frame.ToJSON()
	}
	return frames
}

// ToJSONUserFrames returns a JSON-friendly representation of user frames only
func (s Stack) ToJSONUserFrames() []map[string]any {
	userFrames := s.UserFrames()
	frames := make([]map[string]any, len(userFrames))
	for i, frame := range userFrames {
		frames[i] = frame.ToJSON()
	}
	return frames
}

// UserFrames returns only the user code frames, filtering out runtime and stdlib
func (s Stack) UserFrames() Stack {
	// Pre-allocate with reasonable capacity (usually most frames are user code)
	userFrames := make(Stack, 0, len(s))
	for _, frame := range s {
		if frame.IsUser() {
			userFrames = append(userFrames, frame)
		}
	}
	return userFrames
}

// TopUserFrame returns the topmost user code frame (where the error likely originated)
func (s Stack) TopUserFrame() *StackFrame {
	for _, frame := range s {
		if frame.IsUser() {
			return &frame
		}
	}
	return nil
}

// GetOriginContext returns context information about where the error originated
func (s Stack) GetOriginContext() *StackContext {
	topFrame := s.TopUserFrame()
	if topFrame == nil {
		return nil
	}

	info := topFrame.GetContext()
	return &info
}

// GetCallChain returns the call chain of user functions leading to the error
func (s Stack) GetCallChain() []string {
	userFrames := s.UserFrames()
	// Pre-allocate with reasonable capacity, limited to 5 items max
	capacity := len(userFrames)
	if capacity > 5 {
		capacity = 5
	}
	chain := make([]string, 0, capacity)

	for _, frame := range userFrames {
		if len(chain) < 5 { // Limit to prevent too much noise
			chain = append(chain, frame.getFunctionName())
		}
	}

	return chain
}

// ExtractPackages returns unique packages involved in the error
func (s Stack) ExtractPackages() []string {
	userFrames := s.UserFrames()
	packageMap := make(map[string]bool, len(userFrames))
	// Pre-allocate with reasonable capacity (usually fewer packages than frames)
	packages := make([]string, 0, len(userFrames)/2+1)

	for _, frame := range userFrames {
		if frame.Package != "" && !packageMap[frame.Package] {
			packageMap[frame.Package] = true
			packages = append(packages, frame.Package)
		}
	}

	return packages
}

// ToLogFields converts stack context to logging fields
func (s Stack) ToLogFields() map[string]any {
	fields := make(map[string]any)

	if origin := s.GetOriginContext(); origin != nil {
		fields["error_function"] = origin.Function
		fields["error_package"] = origin.Package
		fields["error_file"] = origin.File
		fields["error_line"] = origin.Line

		if origin.Module != "" {
			fields["error_module"] = origin.Module
		}
	}

	if chain := s.GetCallChain(); len(chain) > 0 {
		fields["call_chain"] = strings.Join(chain, " -> ")
	}

	if packages := s.ExtractPackages(); len(packages) > 0 {
		fields["involved_packages"] = strings.Join(packages, ", ")
	}

	return fields
}

// IsGlobalError determines if the stack trace represents a global/init error
func (s Stack) IsGlobalError() bool {
	for _, frame := range s {
		if strings.Contains(strings.ToLower(frame.Name), "init") ||
			strings.Contains(strings.ToLower(frame.FullName), "runtime.doinit") {
			return true
		}
	}
	return false
}

// ContainsFunction returns true if the stack contains a frame with the given function name
func (s Stack) ContainsFunction(functionName string) bool {
	for _, frame := range s {
		if frame.Name == functionName ||
			strings.HasSuffix(frame.FullName, "."+functionName) {
			return true
		}
	}
	return false
}

// FilterByPackage returns frames that belong to the specified package
func (s Stack) FilterByPackage(packageName string) Stack {
	// Pre-allocate with reasonable capacity (often fewer matches than total)
	filtered := make(Stack, 0, len(s)/4+1)
	for _, frame := range s {
		if frame.Package == packageName {
			filtered = append(filtered, frame)
		}
	}
	return filtered
}

// StackContext extracts contextual information from the stack frame
type StackContext struct {
	Function   string            // Function name
	Package    string            // Package name
	Module     string            // Module name (extracted from full path)
	File       string            // File name
	Line       int               // Line number
	IsUserCode bool              // Whether this is user code
	Metadata   map[string]string // Additional extracted metadata
}

// GetContext extracts rich context information from the stack frame
func (f StackFrame) GetContext() StackContext {
	info := StackContext{
		Function:   f.Name,
		Package:    f.Package,
		Module:     extractModule(f.FullName),
		File:       f.FileName,
		Line:       f.Line,
		IsUserCode: f.IsUser(),
		Metadata:   make(map[string]string),
	}

	// Add additional metadata
	info.Metadata["full_function"] = f.FullName
	info.Metadata["file_path"] = f.File

	if f.IsTest() {
		info.Metadata["type"] = "test"
	} else if f.IsRuntime() {
		info.Metadata["type"] = "runtime"
	} else if f.IsStandardLibrary() {
		info.Metadata["type"] = "stdlib"
	} else {
		info.Metadata["type"] = "user"
	}

	return info
}

// extractShortName extracts the short function name from full name
func extractShortName(fullName string) string {
	if fullName == "" {
		return ""
	}
	// Find last dot using byte-based search
	lastDot := -1
	for i := len(fullName) - 1; i >= 0; i-- {
		if fullName[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 {
		return fullName
	}

	// Handle methods like "(*Type).Method"
	if lastDot > 0 && fullName[lastDot-1] == ')' {
		return fullName[lastDot+1:]
	}

	return fullName[lastDot+1:]
}

// extractPackageFromFunction extracts package name from full function name
func extractPackageFromFunction(fullName string) string {
	if fullName == "" {
		return ""
	}

	// For methods like "(*github.com/user/repo/pkg.Type).Method"
	if strings.HasPrefix(fullName, "(*") {
		end := strings.Index(fullName, ")")
		if end > 0 {
			typeName := fullName[2:end]
			return extractPackageFromType(typeName)
		}
	}

	// For regular functions like "github.com/user/repo/pkg.function"
	lastSlash := strings.LastIndex(fullName, "/")
	if lastSlash == -1 {
		// No slash, might be stdlib
		lastDot := strings.LastIndex(fullName, ".")
		if lastDot == -1 {
			return ""
		}
		return fullName[:lastDot]
	}

	afterSlash := fullName[lastSlash+1:]
	dot := strings.Index(afterSlash, ".")
	if dot == -1 {
		return afterSlash
	}

	return afterSlash[:dot]
}

// extractPackageFromType extracts package from type name
func extractPackageFromType(typeName string) string {
	lastSlash := strings.LastIndex(typeName, "/")
	if lastSlash == -1 {
		return ""
	}

	afterSlash := typeName[lastSlash+1:]
	dot := strings.Index(afterSlash, ".")
	if dot == -1 {
		return afterSlash
	}

	return afterSlash[:dot]
}

// extractModule extracts module name from full function name
func extractModule(fullName string) string {
	if fullName == "" {
		return ""
	}

	// Look for domain-like patterns (e.g., github.com, gitlab.com)
	parts := strings.Split(fullName, "/")
	if len(parts) >= 3 {
		// Check if first part looks like a domain
		if strings.Contains(parts[0], ".") {
			return strings.Join(parts[:3], "/")
		}
	}

	// Fallback: return first part before slash
	if idx := strings.Index(fullName, "/"); idx != -1 {
		return fullName[:idx]
	}

	return ""
}

// rawStack stores just the program counters for efficient storage
type rawStack []uintptr

// captureStack captures just the program counters for maximum performance
func captureStack(skip int) rawStack {
	cfg := GetStackTraceConfig()
	rate := cfg.SamplingRate

	if rate <= 0.0 {
		return nil
	}

	if rate < 1.0 {
		// Deterministic sampling using atomic counter
		counter := atomic.AddUint64(&cfg.samplingCounter, 1)
		threshold := uint64(1.0 / rate)
		if counter%threshold == 0 {
			return nil
		}
	}

	pcs := make([]uintptr, maxStackDepth)
	n := runtime.Callers(skip+1, pcs)

	// Copy only the used portion to avoid storing unused memory
	rawPcs := make([]uintptr, n)
	copy(rawPcs, pcs[:n])

	return rawPcs
}

// toFrames converts the raw stack to resolved stack frames on demand
func (rs rawStack) toFrames() Stack {
	if len(rs) == 0 {
		return nil
	}

	frames := make(Stack, 0, len(rs))
	runtimeFrames := runtime.CallersFrames(rs)
	cfg := GetStackTraceConfig()

	for {
		runtimeFrame, more := runtimeFrames.Next()

		// Skip useless runtime frames and internal erro functions
		if isUselessRuntimeFrame(runtimeFrame.Function, runtimeFrame.File) {
			if !more {
				break
			}
			continue
		}

		frame := StackFrame{
			FullName:         runtimeFrame.Function,
			File:             runtimeFrame.File,
			Line:             runtimeFrame.Line,
			StackTraceConfig: cfg,
		}

		// Extract short name
		frame.Name = extractShortName(runtimeFrame.Function)

		// Extract package name
		frame.Package = extractPackageFromFunction(runtimeFrame.Function)

		// Extract filename
		frame.FileName = filepath.Base(runtimeFrame.File)

		frames = append(frames, frame)

		if !more {
			break
		}
	}

	return frames
}

// isUselessRuntimeFrame determines if a frame should be filtered from stack traces
func isUselessRuntimeFrame(function, file string) bool {
	// Filter out common useless runtime frames that appear at the bottom of stacks
	for _, useless := range uselessFrames {
		if function == useless {
			return true
		}
	}

	// Filter out erro internal functions
	for _, internal := range internalFuncs {
		if strings.HasSuffix(function, "."+internal) {
			return true
		}
	}

	// Also filter by file patterns - these are always runtime noise
	if strings.Contains(file, "runtime/proc.go") || // runtime.main lives here
		strings.Contains(file, "runtime/asm_") || // assembly runtime code
		strings.HasSuffix(file, "/goexit") { // goexit variants
		return true
	}

	return false
}

// isEmpty returns true if the stack is empty
func (rs rawStack) isEmpty() bool {
	return len(rs) == 0
}

// formatFull returns detailed formatted stack trace (lazy evaluation)
func (rs rawStack) formatFull() string {
	return rs.toFrames().FormatFull()
}

// extractPathElements extracts the desired number of path elements from a file path
func extractPathElements(fullPath string, pathElements int) string {
	if pathElements == -1 {
		// Show full path
		return fullPath
	}

	if pathElements <= 0 {
		// Show only filename
		return filepath.Base(fullPath)
	}

	// Split the path and take the last N elements
	pathParts := strings.Split(filepath.Clean(fullPath), string(filepath.Separator))

	// Remove empty parts (can happen with absolute paths)
	var cleanParts []string
	for _, part := range pathParts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	// Take the last pathElements + 1 parts (including filename)
	elementsToTake := pathElements + 1
	if elementsToTake > len(cleanParts) {
		elementsToTake = len(cleanParts)
	}

	start := len(cleanParts) - elementsToTake
	selectedParts := cleanParts[start:]

	return strings.Join(selectedParts, string(filepath.Separator))
}
