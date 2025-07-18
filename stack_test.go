package erro

import (
	"strings"
	"testing"
)

func TestStackTraceConfig(t *testing.T) {
	dev := DevelopmentStackTraceConfig()
	if !dev.ShowFullPaths {
		t.Error("expected full paths in dev config")
	}
	prod := ProductionStackTraceConfig()
	if prod.ShowFullPaths {
		t.Error("expected no full paths in prod config")
	}
	strict := StrictStackTraceConfig()
	if strict.ShowFunctionNames {
		t.Error("expected no function names in strict config")
	}
}

func TestStackFrame_String(t *testing.T) {
	frame := StackFrame{
		Name:     "main",
		FileName: "main.go",
		Line:     10,
	}
	if frame.String() != "main (main.go:10)" {
		t.Errorf("unexpected frame string: %s", frame.String())
	}
}

func TestStackFrame_FormatFull(t *testing.T) {
	frame := StackFrame{
		FullName: "main.main",
		File:     "/app/main.go",
		Line:     10,
	}
	expected := "\tmain.main\n\t\t/app/main.go:10"
	if frame.FormatFull() != expected {
		t.Errorf("expected '%s', got '%s'", expected, frame.FormatFull())
	}
}

func TestStack_ToJSON(t *testing.T) {
	stack := Stack{
		{Name: "main", FullName: "main.main", FileName: "main.go", Line: 10, StackTraceConfig: DevelopmentStackTraceConfig()},
	}
	json := stack.ToJSON()
	if len(json) != 1 {
		t.Fatal("expected one frame")
	}
	if json[0]["function"] != "main.main" {
		t.Errorf("unexpected function name in json: %v", json[0]["function"])
	}
}

func TestStack_ToJSONUserFrames(t *testing.T) {
	stack := Stack{
		{Name: "main", FullName: "main.main"},
		{Name: "goexit", FullName: "runtime.goexit"},
	}
	json := stack.ToJSONUserFrames()
	if len(json) != 1 {
		t.Fatal("expected one user frame")
	}
}

func TestStack_UserFrames(t *testing.T) {
	stack := Stack{
		{Name: "main", FullName: "main.main"},
		{Name: "goexit", FullName: "runtime.goexit"},
	}
	userFrames := stack.UserFrames()
	if len(userFrames) != 1 {
		t.Fatal("expected one user frame")
	}
}

func TestStack_TopUserFrame(t *testing.T) {
	stack := Stack{
		{Name: "goexit", FullName: "runtime.goexit"},
		{Name: "main", FullName: "main.main"},
	}
	top := stack.TopUserFrame()
	if top == nil || top.Name != "main" {
		t.Error("unexpected top user frame")
	}
}

func TestStack_GetOriginContext(t *testing.T) {
	stack := Stack{
		{Name: "main", FullName: "main.main"},
	}
	ctx := stack.GetOriginContext()
	if ctx == nil || ctx.Function != "main" {
		t.Error("unexpected origin context")
	}
}

func TestStack_GetCallChain(t *testing.T) {
	stack := Stack{
		{Name: "a", FullName: "pkg.a", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "b", FullName: "pkg.b", StackTraceConfig: DevelopmentStackTraceConfig()},
	}
	chain := stack.GetCallChain()
	if len(chain) != 2 || chain[0] != "pkg.a" || chain[1] != "pkg.b" {
		t.Errorf("unexpected call chain: %v", chain)
	}
}

func TestStack_ExtractPackages(t *testing.T) {
	stack := Stack{
		{Package: "pkg1"},
		{Package: "pkg2"},
		{Package: "pkg1"},
	}
	pkgs := stack.ExtractPackages()
	if len(pkgs) != 2 {
		t.Errorf("unexpected packages: %v", pkgs)
	}
}

func TestStack_ToLogFields(t *testing.T) {
	stack := Stack{
		{Name: "main", Package: "main", File: "main.go", Line: 10},
	}
	fields := stack.ToLogFields()
	if fields["error_function"] != "main" {
		t.Error("unexpected function in log fields")
	}
}

func TestStack_IsGlobalError(t *testing.T) {
	stack := Stack{{Name: "init"}, {Name: "main"}}
	if !stack.IsGlobalError() {
		t.Error("expected global error")
	}
}

func TestStack_ContainsFunction(t *testing.T) {
	stack := Stack{{Name: "myFunc"}}
	if !stack.ContainsFunction("myFunc") {
		t.Error("expected to find function")
	}
}

func TestStack_FilterByPackage(t *testing.T) {
	stack := Stack{
		{Package: "pkg1"},
		{Package: "pkg2"},
	}
	filtered := stack.FilterByPackage("pkg1")
	if len(filtered) != 1 {
		t.Error("expected one frame")
	}
}

func TestExtractFunctions(t *testing.T) {
	if extractShortName("a/b/c.d") != "d" {
		t.Error("unexpected short name")
	}
	if extractPackageFromFunction("a/b/c.d") != "c" {
		t.Error("unexpected package name")
	}
	if extractPackageFromType("a/b/c.d") != "c" {
		t.Error("unexpected package from type")
	}
	if extractModule("github.com/user/repo/pkg.fn") != "github.com/user/repo" {
		t.Error("unexpected module name")
	}
}

func TestIsUselessRuntimeFrame(t *testing.T) {
	if !isUselessRuntimeFrame("runtime.main", "") {
		t.Error("expected useless frame")
	}
	if isUselessRuntimeFrame("main.main", "") {
		t.Error("expected non-useless frame")
	}
}

func TestExtractPathElements(t *testing.T) {
	path := "/a/b/c/d.go"
	if extractPathElements(path, 0) != "d.go" {
		t.Error("unexpected path elements")
	}
	if extractPathElements(path, 1) != "c/d.go" {
		t.Error("unexpected path elements")
	}
	if extractPathElements(path, -1) != path {
		t.Error("unexpected path elements")
	}
}

func TestStackFrame_IsFunctions(t *testing.T) {
	userFrame := StackFrame{FullName: "github.com/user/repo.fn"}
	if !userFrame.IsUser() {
		t.Error("expected user frame")
	}
	runtimeFrame := StackFrame{FullName: "runtime.goexit"}
	if runtimeFrame.IsUser() {
		t.Error("expected runtime frame")
	}
	stdLibFrame := StackFrame{FullName: "fmt.Println"}
	if stdLibFrame.IsUser() {
		t.Error("expected stdlib frame")
	}
	testFrame := StackFrame{FileName: "main_test.go"}
	if !testFrame.IsTest() {
		t.Error("expected test frame")
	}
}

func TestStack_String(t *testing.T) {
	stack := Stack{
		{Name: "a", FullName: "pkg.a", FileName: "a.go", Line: 1},
		{Name: "b", FullName: "pkg.b", FileName: "b.go", Line: 2},
	}
	expected := "a (a.go:1) -> b (b.go:2)"
	if stack.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, stack.String())
	}
}

func TestStack_FormatFull(t *testing.T) {
	stack := Stack{
		{FullName: "pkg.a", File: "/app/a.go", Line: 1, StackTraceConfig: DevelopmentStackTraceConfig()},
		{FullName: "pkg.b", File: "/app/b.go", Line: 2, StackTraceConfig: DevelopmentStackTraceConfig()},
	}
	if !strings.Contains(stack.FormatFull(), "pkg.a") {
		t.Error("expected full format to contain function name")
	}
}

func TestStackFrame_Printing(t *testing.T) {
	frame := StackFrame{
		Name:     "myFunc",
		FullName: "github.com/user/repo.myFunc",
		Package:  "repo",
		File:     "/home/user/project/repo/main.go",
		FileName: "main.go",
		Line:     42,
	}

	// Test case 1: Default (nil) config
	if !strings.Contains(frame.String(), "myFunc (main.go:42)") {
		t.Errorf("Default string format is incorrect: %s", frame.String())
	}

	// Test case 2: Production config
	frame.StackTraceConfig = ProductionStackTraceConfig()
	if !strings.Contains(frame.String(), "myFunc (main.go:42)") {
		t.Errorf("Production string format is incorrect: %s", frame.String())
	}

	// Test case 3: Strict config
	frame.StackTraceConfig = StrictStackTraceConfig()
	if !strings.Contains(frame.String(), "[some_function] (main.go:42)") {
		t.Errorf("Strict string format is incorrect: %s", frame.String())
	}

	// Test case 4: Custom config
	frame.StackTraceConfig = &StackTraceConfig{
		ShowFunctionNames: false,
		ShowFileNames:     true,
		ShowLineNumbers:   false,
		FunctionRedacted:  "[REDACTED_FUNC]",
		FileNameRedacted:  "[REDACTED_FILE]",
	}
	if !strings.Contains(frame.String(), "[REDACTED_FUNC] ([REDACTED_FILE])") {
		t.Errorf("Custom string format is incorrect: %s", frame.String())
	}
}

func TestStackFrame_IsErroInternal_EdgeCases(t *testing.T) {
	// Test case: buildInfo is nil
	frame := StackFrame{Name: "someFunction", FullName: "github.com/some/module.someFunction"}
	if frame.IsErroInternal() {
		t.Error("expected non-internal frame when buildInfo is nil")
	}

	// Test case: buildInfo.Path is empty
	frame = StackFrame{Name: "someFunction", FullName: "github.com/some/module.someFunction"}
	if frame.IsErroInternal() {
		t.Error("expected non-internal frame when buildInfo.Path is empty")
	}

	// Test case: test frame should never be internal
	testFrame := StackFrame{Name: "TestFunction", FileName: "test_file_test.go"}
	if testFrame.IsErroInternal() {
		t.Error("test frames should never be considered internal")
	}

	// Test case: internal function name
	internalFrame := StackFrame{Name: "New", FullName: "github.com/maxbolgarin/erro.New"}
	if !internalFrame.IsErroInternal() {
		t.Error("expected internal frame for internal function name")
	}
}

func TestStackFrame_getFrameType_AllTypes(t *testing.T) {
	// Test runtime frame
	runtimeFrame := StackFrame{FullName: "runtime.goexit"}
	if runtimeFrame.getFrameType() != "runtime" {
		t.Errorf("expected runtime type, got %s", runtimeFrame.getFrameType())
	}

	// Test stdlib frame
	stdlibFrame := StackFrame{FullName: "fmt.Println"}
	if stdlibFrame.getFrameType() != "stdlib" {
		t.Errorf("expected stdlib type, got %s", stdlibFrame.getFrameType())
	}

	// Test test frame
	testFrame := StackFrame{FileName: "test_file_test.go"}
	if testFrame.getFrameType() != "test" {
		t.Errorf("expected test type, got %s", testFrame.getFrameType())
	}

	// Test user frame
	userFrame := StackFrame{FullName: "github.com/user/repo.function"}
	if userFrame.getFrameType() != "user" {
		t.Errorf("expected user type, got %s", userFrame.getFrameType())
	}
}

func TestStackFrame_getFileName_EdgeCases(t *testing.T) {
	// Test case: ShowFileNames is false
	frame := StackFrame{
		File:             "/path/to/file.go",
		FileName:         "file.go",
		Line:             42,
		StackTraceConfig: &StackTraceConfig{ShowFileNames: false},
	}
	if frame.getFileName() != defaultFileNameRedacted {
		t.Errorf("expected redacted filename, got %s", frame.getFileName())
	}

	// Test case: custom FileNameRedacted
	frame.StackTraceConfig.FileNameRedacted = "[CUSTOM_REDACTED]"
	if frame.getFileName() != "[CUSTOM_REDACTED]" {
		t.Errorf("expected custom redacted filename, got %s", frame.getFileName())
	}

	// Test case: ShowLineNumbers is false
	frame.StackTraceConfig.ShowFileNames = true
	frame.StackTraceConfig.ShowLineNumbers = false
	if frame.getFileName() != "[CUSTOM_REDACTED]" {
		t.Errorf("expected custom redacted filename when ShowLineNumbers is false, got %s", frame.getFileName())
	}

	// Test case: ShowFileNames is false
	frame.StackTraceConfig.ShowFileNames = false
	if frame.getFileName() != "[CUSTOM_REDACTED]" {
		t.Errorf("expected custom redacted filename when ShowFileNames is false, got %s", frame.getFileName())
	}

	// Test case: ShowFullPaths is true
	frame.StackTraceConfig.ShowFileNames = true
	frame.StackTraceConfig.ShowFullPaths = true
	frame.StackTraceConfig.ShowLineNumbers = true
	if frame.getFileName() != "/path/to/file.go:42" {
		t.Errorf("expected full path with line number, got %s", frame.getFileName())
	}
}

func TestStack_GetOriginContext_NilTopFrame(t *testing.T) {
	// Test case: no user frames
	stack := Stack{
		{Name: "runtime.goexit", FullName: "runtime.goexit"},
		{Name: "fmt.Println", FullName: "fmt.Println"},
	}
	ctx := stack.GetOriginContext()
	if ctx != nil {
		t.Error("expected nil context when no user frames exist")
	}
}

func TestStack_GetCallChain_MoreThan5Frames(t *testing.T) {
	// Test case: more than 5 user frames
	stack := Stack{
		{Name: "a", FullName: "pkg.a", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "b", FullName: "pkg.b", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "c", FullName: "pkg.c", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "d", FullName: "pkg.d", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "e", FullName: "pkg.e", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "f", FullName: "pkg.f", StackTraceConfig: DevelopmentStackTraceConfig()},
		{Name: "g", FullName: "pkg.g", StackTraceConfig: DevelopmentStackTraceConfig()},
	}
	chain := stack.GetCallChain()
	if len(chain) != 5 {
		t.Errorf("expected 5 frames, got %d", len(chain))
	}
}

func TestStack_ToLogFields_WithModule(t *testing.T) {
	// Test case: origin context with module
	stack := Stack{
		{
			Name:     "main",
			FullName: "github.com/user/repo/pkg.main",
			Package:  "pkg",
			File:     "main.go",
			Line:     10,
		},
	}
	fields := stack.ToLogFields()
	if fields["error_module"] != "github.com/user/repo" {
		t.Errorf("expected module in log fields, got %v", fields["error_module"])
	}
}

func TestStack_ToLogFields_WithoutModule(t *testing.T) {
	// Test case: origin context without module
	stack := Stack{
		{
			Name:     "main",
			FullName: "main.main",
			Package:  "main",
			File:     "main.go",
			Line:     10,
		},
	}
	fields := stack.ToLogFields()
	if _, exists := fields["error_module"]; exists {
		t.Error("expected no module field when module is empty")
	}
}

func TestStack_IsGlobalError_EdgeCases(t *testing.T) {
	// Test case: no init functions
	stack := Stack{
		{Name: "main", FullName: "main.main"},
		{Name: "process", FullName: "pkg.process"},
	}
	if stack.IsGlobalError() {
		t.Error("expected no global error when no init functions")
	}

	// Test case: init function in name
	stack = Stack{
		{Name: "init", FullName: "pkg.init"},
	}
	if !stack.IsGlobalError() {
		t.Error("expected global error when init function present")
	}

	// Test case: runtime.doinit
	stack = Stack{
		{Name: "doinit", FullName: "runtime.doinit"},
	}
	if !stack.IsGlobalError() {
		t.Error("expected global error when runtime.doinit present")
	}
}

func TestStack_ContainsFunction_EdgeCases(t *testing.T) {
	// Test case: function not found
	stack := Stack{
		{Name: "main", FullName: "main.main"},
	}
	if stack.ContainsFunction("nonexistent") {
		t.Error("expected function not found")
	}

	// Test case: function with suffix
	stack = Stack{
		{Name: "process", FullName: "pkg.process"},
	}
	if !stack.ContainsFunction("process") {
		t.Error("expected function found by suffix")
	}
}

func TestStackFrame_GetContext_AllTypes(t *testing.T) {
	// Test runtime frame
	runtimeFrame := StackFrame{
		Name:     "goexit",
		FullName: "runtime.goexit",
		Package:  "runtime",
		File:     "/usr/local/go/src/runtime/proc.go",
		FileName: "proc.go",
		Line:     100,
	}
	ctx := runtimeFrame.GetContext()
	if ctx.Metadata["type"] != "runtime" {
		t.Errorf("expected runtime type, got %s", ctx.Metadata["type"])
	}

	// Test stdlib frame
	stdlibFrame := StackFrame{
		Name:     "Println",
		FullName: "fmt.Println",
		Package:  "fmt",
		File:     "/usr/local/go/src/fmt/print.go",
		FileName: "print.go",
		Line:     200,
	}
	ctx = stdlibFrame.GetContext()
	if ctx.Metadata["type"] != "stdlib" {
		t.Errorf("expected stdlib type, got %s", ctx.Metadata["type"])
	}

	// Test test frame
	testFrame := StackFrame{
		Name:     "TestFunction",
		FullName: "pkg.TestFunction",
		Package:  "pkg",
		File:     "/path/to/test_file_test.go",
		FileName: "test_file_test.go",
		Line:     50,
	}
	ctx = testFrame.GetContext()
	if ctx.Metadata["type"] != "test" {
		t.Errorf("expected test type, got %s", ctx.Metadata["type"])
	}

	// Test user frame
	userFrame := StackFrame{
		Name:     "main",
		FullName: "github.com/user/repo.main",
		Package:  "repo",
		File:     "/path/to/main.go",
		FileName: "main.go",
		Line:     10,
	}
	ctx = userFrame.GetContext()
	if ctx.Metadata["type"] != "user" {
		t.Errorf("expected user type, got %s", ctx.Metadata["type"])
	}
}

func TestExtractShortName_EdgeCases(t *testing.T) {
	// Test case: empty string
	if extractShortName("") != "" {
		t.Error("expected empty string for empty input")
	}

	// Test case: no dots
	if extractShortName("simple") != "simple" {
		t.Error("expected same string when no dots")
	}

	// Test case: with parentheses
	if extractShortName("(*Type).method") != "method" {
		t.Error("expected method name after parentheses")
	}

	// Test case: multiple dots
	if extractShortName("pkg.subpkg.function") != "function" {
		t.Error("expected last part after dots")
	}
}

func TestExtractPackageFromFunction_EdgeCases(t *testing.T) {
	// Test case: empty string
	if extractPackageFromFunction("") != "" {
		t.Error("expected empty string for empty input")
	}

	// Test case: no slashes or dots
	if extractPackageFromFunction("simple") != "" {
		t.Error("expected empty string for simple function name")
	}

	// Test case: no slashes, has dots
	if extractPackageFromFunction("pkg.function") != "pkg" {
		t.Error("expected package name before dot")
	}

	// Test case: with parentheses
	if extractPackageFromFunction("(*github.com/user/repo.Type).method") != "repo" {
		t.Error("expected package name from type in parentheses")
	}

	// Test case: after slash, no dots
	if extractPackageFromFunction("github.com/user/repo/function") != "function" {
		t.Error("expected function name when no dots after slash")
	}
}

func TestExtractPackageFromType_EdgeCases(t *testing.T) {
	// Test case: no slashes
	if extractPackageFromType("simple") != "" {
		t.Error("expected empty string when no slashes")
	}

	// Test case: after slash, no dots
	if extractPackageFromType("github.com/user/repo/type") != "type" {
		t.Error("expected type name when no dots after slash")
	}
}

func TestExtractModule_EdgeCases(t *testing.T) {
	// Test case: empty string
	if extractModule("") != "" {
		t.Error("expected empty string for empty input")
	}

	// Test case: no slashes
	if extractModule("simple") != "" {
		t.Error("expected empty string when no slashes")
	}

	// Test case: less than 3 parts
	if extractModule("a/b") != "a" {
		t.Error("expected first part when less than 3 parts")
	}

	// Test case: first part contains dots
	if extractModule("github.com/user/repo/pkg.function") != "github.com/user/repo" {
		t.Error("expected first 3 parts when first part contains dots")
	}
}

func TestCaptureStack_EdgeCases(t *testing.T) {
	// Test case: skip is 0
	stack := captureStack(0)
	if stack != nil {
		t.Error("expected nil stack when skip is 0")
	}
}

func TestRawStack_ToFrames_EdgeCases(t *testing.T) {
	// Test case: empty raw stack
	var rs rawStack
	frames := rs.toFrames(nil)
	if frames != nil {
		t.Error("expected nil frames for empty raw stack")
	}

	// Test case: nil config
	rs = rawStack{1, 2, 3} // Some dummy values
	frames = rs.toFrames(nil)
	if len(frames) == 0 {
		t.Error("expected frames even with nil config")
	}
}

func TestIsUselessRuntimeFrame_EdgeCases(t *testing.T) {
	// Test case: internal function suffix
	if !isUselessRuntimeFrame("pkg.New", "") {
		t.Error("expected useless frame for internal function suffix")
	}

	// Test case: runtime/proc.go
	if !isUselessRuntimeFrame("some.function", "/usr/local/go/src/runtime/proc.go") {
		t.Error("expected useless frame for runtime/proc.go")
	}

	// Test case: runtime/asm_
	if !isUselessRuntimeFrame("some.function", "/usr/local/go/src/runtime/asm_amd64.s") {
		t.Error("expected useless frame for runtime/asm_")
	}

	// Test case: /goexit suffix
	if !isUselessRuntimeFrame("some.function", "/path/to/goexit") {
		t.Error("expected useless frame for /goexit suffix")
	}
}

func TestStackFrame_String_WithConfig(t *testing.T) {
	// Test case: ShowAllCodeFrames false, not user frame
	frame := StackFrame{
		Name:             "fmt.Println",
		FullName:         "fmt.Println",
		StackTraceConfig: &StackTraceConfig{ShowAllCodeFrames: false},
	}
	if frame.String() != defaultHiddenFrame {
		t.Errorf("expected hidden frame, got %s", frame.String())
	}

	// Test case: ShowFileNames false
	frame = StackFrame{
		Name:             "main",
		FullName:         "main.main",
		StackTraceConfig: &StackTraceConfig{ShowFileNames: false, ShowFunctionNames: true},
	}
	result := frame.String()
	if !strings.Contains(result, "main") {
		t.Errorf("expected function name in string, got '%s'", result)
	}
}

func TestStackFrame_FormatFull_WithConfig(t *testing.T) {
	// Test case: ShowAllCodeFrames false, not user frame
	frame := StackFrame{
		Name:             "fmt.Println",
		FullName:         "fmt.Println",
		StackTraceConfig: &StackTraceConfig{ShowAllCodeFrames: false},
	}
	expected := "\t" + defaultHiddenFrame
	if frame.FormatFull() != expected {
		t.Errorf("expected '%s', got '%s'", expected, frame.FormatFull())
	}

	// Test case: ShowFileNames false
	frame = StackFrame{
		Name:             "main",
		FullName:         "main.main",
		StackTraceConfig: &StackTraceConfig{ShowFileNames: false, ShowFunctionNames: true},
	}
	result := frame.FormatFull()
	if !strings.Contains(result, "main") {
		t.Errorf("expected function name in full format, got '%s'", result)
	}
}

func TestStack_String_EmptyStack(t *testing.T) {
	stack := Stack{}
	if stack.String() != "" {
		t.Errorf("expected empty string for empty stack, got '%s'", stack.String())
	}
}

func TestStack_FormatFull_EmptyStack(t *testing.T) {
	stack := Stack{}
	if stack.FormatFull() != "" {
		t.Errorf("expected empty string for empty stack, got '%s'", stack.FormatFull())
	}
}

func TestStack_UserFrames_EmptyStack(t *testing.T) {
	stack := Stack{}
	userFrames := stack.UserFrames()
	if len(userFrames) != 0 {
		t.Error("expected empty user frames for empty stack")
	}
}

func TestStack_TopUserFrame_NoUserFrames(t *testing.T) {
	stack := Stack{
		{Name: "runtime.goexit", FullName: "runtime.goexit"},
		{Name: "fmt.Println", FullName: "fmt.Println"},
	}
	top := stack.TopUserFrame()
	if top != nil {
		t.Error("expected nil when no user frames")
	}
}

func TestStack_ExtractPackages_EmptyStack(t *testing.T) {
	stack := Stack{}
	packages := stack.ExtractPackages()
	if len(packages) != 0 {
		t.Error("expected empty packages for empty stack")
	}
}

func TestStack_FilterByPackage_EmptyStack(t *testing.T) {
	stack := Stack{}
	filtered := stack.FilterByPackage("somepackage")
	if len(filtered) != 0 {
		t.Error("expected empty filtered stack for empty stack")
	}
}

func TestStackFrame_IsStandardLibrary_EdgeCases(t *testing.T) {
	// Test case: empty FullName
	frame := StackFrame{FullName: ""}
	if frame.IsStandardLibrary() {
		t.Error("expected not stdlib for empty FullName")
	}

	// Test case: contains dot before slash
	frame = StackFrame{FullName: "github.com/user/repo.function"}
	if frame.IsStandardLibrary() {
		t.Error("expected not stdlib when contains dot before slash")
	}

	// Test case: stdlib prefix
	frame = StackFrame{FullName: "fmt.Println"}
	if !frame.IsStandardLibrary() {
		t.Error("expected stdlib for fmt prefix")
	}
}

func TestStackFrame_IsRuntime_EdgeCases(t *testing.T) {
	// Test case: runtime package
	frame := StackFrame{Package: "runtime"}
	if !frame.IsRuntime() {
		t.Error("expected runtime frame for runtime package")
	}

	// Test case: runtime file
	frame = StackFrame{File: "/usr/local/go/src/runtime/proc.go"}
	if !frame.IsRuntime() {
		t.Error("expected runtime frame for runtime file")
	}
}

func TestStackFrame_IsTest_EdgeCases(t *testing.T) {
	// Test case: test file
	frame := StackFrame{FileName: "main_test.go"}
	if !frame.IsTest() {
		t.Error("expected test frame for test file")
	}

	// Test case: test function
	frame = StackFrame{Name: "TestFunction"}
	if !frame.IsTest() {
		t.Error("expected test frame for test function")
	}

	// Test case: testing file
	frame = StackFrame{File: "/usr/local/go/src/testing/testing.go"}
	if !frame.IsTest() {
		t.Error("expected test frame for testing file")
	}
}

func TestStackFrame_GetFunctionName_EdgeCases(t *testing.T) {
	// Test case: ShowFunctionNames false, custom redacted
	frame := StackFrame{
		Name:     "function",
		FullName: "pkg.function",
		StackTraceConfig: &StackTraceConfig{
			ShowFunctionNames: false,
			FunctionRedacted:  "[CUSTOM_REDACTED]",
		},
	}
	if frame.getFunctionName() != "[CUSTOM_REDACTED]" {
		t.Errorf("expected custom redacted function name, got %s", frame.getFunctionName())
	}

	// Test case: ShowFunctionNames false, no custom redacted
	frame.StackTraceConfig.FunctionRedacted = ""
	if frame.getFunctionName() != defaultFunctionRedacted {
		t.Errorf("expected default redacted function name, got %s", frame.getFunctionName())
	}

	// Test case: ShowFunctionNames true, ShowPackageNames false
	frame.StackTraceConfig.ShowFunctionNames = true
	frame.StackTraceConfig.ShowPackageNames = false
	if frame.getFunctionName() != "function" {
		t.Errorf("expected function name only, got %s", frame.getFunctionName())
	}
}

func TestExtractPathElements_EdgeCases(t *testing.T) {
	// Test case: pathElements is 0
	path := "/a/b/c/d.go"
	if extractPathElements(path, 0) != "d.go" {
		t.Errorf("expected filename only, got %s", extractPathElements(path, 0))
	}

	// Test case: pathElements is -1
	if extractPathElements(path, -1) != path {
		t.Errorf("expected full path, got %s", extractPathElements(path, -1))
	}

	// Test case: pathElements greater than available parts
	if extractPathElements(path, 10) != "a/b/c/d.go" {
		t.Errorf("expected all parts when elements > available, got %s", extractPathElements(path, 10))
	}

	// Test case: path with empty parts
	path = "/a//b/c/d.go"
	if extractPathElements(path, 2) != "b/c/d.go" {
		t.Errorf("expected cleaned path, got %s", extractPathElements(path, 2))
	}
}
