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
	disabled := DisabledStackTraceConfig()
	if disabled.ShowFileNames {
		t.Error("expected no file names in disabled config")
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
		t.Error("expected non-useless frame")	}
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
