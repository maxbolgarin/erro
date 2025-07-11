package erro

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

type StackType string

const (
	StackTypeUser            StackType = "user"
	StackTypeRuntime         StackType = "runtime"
	StackTypeStandardLibrary StackType = "stdlib"
	StackTypeTest            StackType = "test"
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
)

const maxDepth = 50

// RawStack stores just the program counters for efficient storage
type RawStack []uintptr

// CaptureWrapPoint captures just the program counter of the immediate caller
// This is much faster than capturing a full stack for wrap operations
func CaptureWrapPoint(skip int) uintptr {
	var pcs [1]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	if n > 0 {
		return pcs[0]
	}
	return 0
}

// ResolveWrapPoint converts a single program counter to a StackFrame
func ResolveWrapPoint(pc uintptr) StackFrame {
	if pc == 0 {
		return StackFrame{}
	}

	runtimeFrames := runtime.CallersFrames([]uintptr{pc})
	runtimeFrame, _ := runtimeFrames.Next()

	return StackFrame{
		FullName: runtimeFrame.Function,
		File:     runtimeFrame.File,
		Line:     runtimeFrame.Line,
		Name:     extractShortName(runtimeFrame.Function),
		Package:  extractPackageFromFunction(runtimeFrame.Function),
		FileName: filepath.Base(runtimeFrame.File),
	}
}

// CaptureStack captures just the program counters for maximum performance
func CaptureStack(skip int) RawStack {
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+1, pcs[:])

	// Copy only the used portion to avoid storing unused memory
	rawPcs := make([]uintptr, n)
	copy(rawPcs, pcs[:n])

	return rawPcs
}

// ToFrames converts the raw stack to resolved stack frames on demand
func (rs RawStack) ToFrames() Stack {
	if len(rs) == 0 {
		return nil
	}

	frames := make(Stack, 0, len(rs))
	runtimeFrames := runtime.CallersFrames(rs)

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
			FullName: runtimeFrame.Function,
			File:     runtimeFrame.File,
			Line:     runtimeFrame.Line,
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
	uselessFrames := []string{
		"runtime.main",        // Runtime's main function (not user's main)
		"runtime.goexit",      // Goroutine exit function
		"runtime.deferreturn", // Defer cleanup
	}

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

// Len returns the number of stack frames
func (rs RawStack) Len() int {
	return len(rs)
}

// IsEmpty returns true if the stack is empty
func (rs RawStack) IsEmpty() bool {
	return len(rs) == 0
}

// Convenience methods that delegate to resolved frames when needed

// TopUserFrame returns the topmost user code frame (lazy evaluation)
func (rs RawStack) TopUserFrame() *StackFrame {
	return rs.ToFrames().TopUserFrame()
}

// UserFrames returns only the user code frames (lazy evaluation)
func (rs RawStack) UserFrames() Stack {
	return rs.ToFrames().UserFrames()
}

// GetOriginContext returns context information about where the error originated
func (rs RawStack) GetOriginContext() *ContextInfo {
	return rs.ToFrames().GetOriginContext()
}

// ToLogFields converts stack context to logging fields (lazy evaluation)
func (rs RawStack) ToLogFields() map[string]any {
	return rs.ToFrames().ToLogFields()
}

// String returns a formatted string representation (lazy evaluation)
func (rs RawStack) String() string {
	return rs.ToFrames().String()
}

// FormatFull returns detailed formatted stack trace (lazy evaluation)
func (rs RawStack) FormatFull() string {
	return rs.ToFrames().FormatFull()
}

// GetCaller returns information about a specific caller in the stack
func GetCaller(skip int) *StackFrame {
	rawStack := CaptureStack(skip + 1)
	stack := rawStack.ToFrames()
	if len(stack) > 0 {
		return &stack[0]
	}
	return nil
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
}

// String returns a formatted representation of the stack frame
func (f StackFrame) String() string {
	return fmt.Sprintf("%s (%s:%d)", f.Name, f.FileName, f.Line)
}

// Format returns a formatted stack frame with custom separator
func (f StackFrame) Format(sep string) string {
	return fmt.Sprintf("%s%s%s%s%d", f.Name, sep, f.FileName, sep, f.Line)
}

// FormatFull returns a detailed formatted stack frame
func (f StackFrame) FormatFull() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.FullName, f.File, f.Line)
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

	// If it has no slashes and contains a dot, it's likely stdlib (e.g., "fmt.Printf")
	if !strings.Contains(f.FullName, "/") && strings.Contains(f.FullName, ".") {
		return true
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

// ContextInfo extracts contextual information from the stack frame
type ContextInfo struct {
	Function   string            // Function name
	Package    string            // Package name
	Module     string            // Module name (extracted from full path)
	File       string            // File name
	Line       int               // Line number
	IsUserCode bool              // Whether this is user code
	Metadata   map[string]string // Additional extracted metadata
}

// GetContextInfo extracts rich context information from the stack frame
func (f StackFrame) GetContextInfo() ContextInfo {
	info := ContextInfo{
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
		info.Metadata["type"] = string(StackTypeTest)
	} else if f.IsRuntime() {
		info.Metadata["type"] = string(StackTypeRuntime)
	} else if f.IsStandardLibrary() {
		info.Metadata["type"] = string(StackTypeStandardLibrary)
	} else {
		info.Metadata["type"] = string(StackTypeUser)
	}

	return info
}

// Stack represents a collection of stack frames with enhanced analysis capabilities
type Stack []StackFrame

// String returns a formatted string representation of the entire stack
func (s Stack) String() string {
	var builder strings.Builder
	for i, frame := range s {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(frame.String())
	}
	return builder.String()
}

// Format returns formatted stack frames with custom options
func (s Stack) Format(sep string, invert bool) []string {
	var str []string
	for _, f := range s {
		formatted := f.Format(sep)
		if invert {
			str = append(str, formatted)
		} else {
			str = append([]string{formatted}, str...)
		}
	}
	return str
}

// FormatFull returns detailed formatted stack trace
func (s Stack) FormatFull() string {
	var builder strings.Builder
	for i, frame := range s {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(frame.FormatFull())
	}
	return builder.String()
}

// UserFrames returns only the user code frames, filtering out runtime and stdlib
func (s Stack) UserFrames() Stack {
	var userFrames Stack
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
func (s Stack) GetOriginContext() *ContextInfo {
	topFrame := s.TopUserFrame()
	if topFrame == nil {
		return nil
	}

	info := topFrame.GetContextInfo()
	return &info
}

// GetCallChain returns the call chain of user functions leading to the error
func (s Stack) GetCallChain() []string {
	var chain []string
	userFrames := s.UserFrames()

	for _, frame := range userFrames {
		if len(chain) < 5 { // Limit to prevent too much noise
			chain = append(chain, frame.Name)
		}
	}

	return chain
}

// ExtractPackages returns unique packages involved in the error
func (s Stack) ExtractPackages() []string {
	packageMap := make(map[string]bool)
	var packages []string

	for _, frame := range s.UserFrames() {
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

// extractShortName extracts the short function name from full name
func extractShortName(fullName string) string {
	if fullName == "" {
		return ""
	}

	// Handle methods (e.g., "(*Type).Method" or "Type.Method")
	if idx := strings.LastIndex(fullName, ")."); idx != -1 {
		return fullName[idx+2:]
	}

	// Handle regular functions
	if idx := strings.LastIndex(fullName, "."); idx != -1 {
		return fullName[idx+1:]
	}

	return fullName
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
	var filtered Stack
	for _, frame := range s {
		if frame.Package == packageName {
			filtered = append(filtered, frame)
		}
	}
	return filtered
}

func (s StackType) String() string {
	return string(s)
}
