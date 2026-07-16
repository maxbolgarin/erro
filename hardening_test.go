package erro_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/maxbolgarin/erro"
)

// --- Redaction defense in depth ---

type recordingSpan struct {
	attrs []any
}

func (s *recordingSpan) RecordError(err erro.Error)      {}
func (s *recordingSpan) SetAttributes(attributes ...any) { s.attrs = attributes }
func (s *recordingSpan) TraceID() string                 { return "trace-1" }
func (s *recordingSpan) SpanID() string                  { return "span-1" }
func (s *recordingSpan) ParentSpanID() string            { return "" }

func TestRedactedValue_NeverLeaks(t *testing.T) {
	secret := "s3cr3t-value"
	redacted := erro.Redact(secret)

	t.Run("fmt verbs", func(t *testing.T) {
		for _, out := range []string{
			fmt.Sprintf("value: %v", redacted),
			fmt.Sprintf("value: %s", redacted),
			fmt.Sprintf("value: %+v", redacted),
			fmt.Sprintf("value: %#v", redacted),
			fmt.Sprint(redacted),
		} {
			if strings.Contains(out, secret) {
				t.Errorf("secret leaked through fmt: %q", out)
			}
			if !strings.Contains(out, erro.RedactedPlaceholder) {
				t.Errorf("expected placeholder in fmt output, got %q", out)
			}
		}
	})

	t.Run("json.Marshal of the value", func(t *testing.T) {
		data, err := json.Marshal(redacted)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		if strings.Contains(string(data), secret) {
			t.Errorf("secret leaked through JSON: %s", data)
		}
	})

	t.Run("fields slice printed with %v", func(t *testing.T) {
		err := erro.New("op failed", "password", redacted)
		out := fmt.Sprintf("%v", err.AllFields())
		if strings.Contains(out, secret) {
			t.Errorf("secret leaked through fields printing: %q", out)
		}
	})

	t.Run("span attributes", func(t *testing.T) {
		span := &recordingSpan{}
		erro.New("op failed", "password", redacted, erro.RecordSpan(span))
		for _, attr := range span.attrs {
			if s, ok := attr.(string); ok && strings.Contains(s, secret) {
				t.Errorf("secret leaked to span attributes: %v", span.attrs)
			}
			if _, ok := attr.(erro.RedactedValue); ok {
				t.Errorf("raw RedactedValue passed to span attributes: %v", span.attrs)
			}
		}
	})

	t.Run("redacted key position in ErrorToJSON", func(t *testing.T) {
		err := erro.New("op failed", redacted, "value-for-redacted-key")
		schema := erro.ErrorToJSON(err)
		data, jsonErr := json.Marshal(schema)
		if jsonErr != nil {
			t.Fatalf("marshal failed: %v", jsonErr)
		}
		if strings.Contains(string(data), secret) {
			t.Errorf("secret in key position leaked: %s", data)
		}
	})
}

// --- JSON robustness ---

func TestErrorToJSON_HostileValues(t *testing.T) {
	ch := make(chan int)
	err := erro.New("bad values",
		"nan", math.NaN(),
		"inf", math.Inf(1),
		"neg_inf", math.Inf(-1),
		"channel", ch,
		"func", func() {},
		"ok", 42,
	)

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("json.Marshal must not fail on hostile field values: %v", jsonErr)
	}
	if !strings.Contains(string(data), "bad values") {
		t.Errorf("expected message in JSON output: %s", data)
	}
	if !strings.Contains(string(data), "42") {
		t.Errorf("expected normal values preserved: %s", data)
	}
}

func TestUnmarshalJSON_EnforcesLimits(t *testing.T) {
	fields := make([]any, 0, erro.MaxFieldsCount*4)
	for i := 0; i < erro.MaxFieldsCount*2; i++ {
		fields = append(fields, fmt.Sprintf("key_%d", i), "value")
	}
	payload := map[string]any{
		"id":      "test_id",
		"message": strings.Repeat("m", erro.MaxMessageLength*3),
		"fields":  fields,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal setup failed: %v", err)
	}

	target := erro.New("placeholder")
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got := len(target.AllFields()); got > erro.MaxFieldsCount*2 {
		t.Errorf("fields not capped on unmarshal: %d entries", got)
	}
	if got := len(target.Message()); got > erro.MaxMessageLength {
		t.Errorf("message not truncated on unmarshal: %d bytes", got)
	}
}

func TestUnmarshalJSON_OddFieldsNormalized(t *testing.T) {
	data := []byte(`{"id":"x","message":"m","fields":["lonely_key"]}`)
	target := erro.New("placeholder")
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got := len(target.AllFields()); got%2 != 0 {
		t.Errorf("odd field count survived unmarshal: %d", got)
	}
}

// --- Collection safety ---

func TestListErrs_ReturnsCopy(t *testing.T) {
	list := erro.NewList()
	list.New("first").New("second")

	errs := list.Errs()
	errs[0] = nil

	if list.First() == nil {
		t.Error("mutating the Errs() result must not affect the list")
	}
}

func TestSafeSet_ErrThenAdd_NoRace(t *testing.T) {
	set := erro.NewSafeSet()
	set.New("initial one").New("initial two")

	combined := set.Err()
	if combined == nil {
		t.Fatal("expected non-nil combined error")
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				set.New(fmt.Sprintf("err-%d-%d", n, j))
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = combined.Error()
			}
		}()
	}
	wg.Wait()
}

func TestSafeList_ErrsThenAdd_NoRace(t *testing.T) {
	list := erro.NewSafeList()
	list.New("initial")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				list.New(fmt.Sprintf("err-%d-%d", n, j))
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				for _, e := range list.Errs() {
					_ = e.Error()
				}
			}
		}()
	}
	wg.Wait()
}

// --- Nil-wrap consistency ---

func TestWrapNil_Consistency(t *testing.T) {
	if err := erro.NotFoundError.Wrap(nil, "user"); err != nil {
		t.Errorf("template.Wrap(nil) must return nil, got %v", err)
	}

	list := erro.NewList()
	list.Wrap(nil, "context")
	if !list.Empty() {
		t.Error("List.Wrap(nil) must not add an error")
	}

	set := erro.NewSet()
	set.Wrap(nil, "context")
	if !set.Empty() {
		t.Error("Set.Wrap(nil) must not add an error")
	}

	safeList := erro.NewSafeList()
	safeList.Wrap(nil, "context")
	if !safeList.Empty() {
		t.Error("SafeList.Wrap(nil) must not add an error")
	}

	safeSet := erro.NewSafeSet()
	safeSet.Wrap(nil, "context")
	if !safeSet.Empty() {
		t.Error("SafeSet.Wrap(nil) must not add an error")
	}
}

// --- Wrap chain preservation ---

func TestWrap_PreservesForeignLayers(t *testing.T) {
	base := erro.New("base failure", erro.ClassNotFound, "base_key", "base_value")
	foreign := fmt.Errorf("foreign context: %w", base)
	wrapped := erro.Wrap(foreign, "top level")

	if msg := wrapped.Error(); !strings.Contains(msg, "foreign context") {
		t.Errorf("foreign layer message dropped from chain: %q", msg)
	}
	if !errors.Is(wrapped, base) {
		t.Error("errors.Is must reach the erro error through the foreign layer")
	}
	if !errors.Is(wrapped, foreign) {
		t.Error("errors.Is must match the preserved foreign layer")
	}
	if unwrapped := errors.Unwrap(wrapped); unwrapped != foreign {
		t.Errorf("Unwrap must return the foreign layer, got %T", unwrapped)
	}
	if wrapped.Class() != erro.ClassNotFound {
		t.Error("metadata must still be inherited through the foreign layer")
	}
	if wrapped.ID() != base.ID() {
		t.Error("ID must still be inherited through the foreign layer")
	}
	if fields := wrapped.AllFields(); len(fields) != 2 {
		t.Errorf("expected inherited base fields, got %v", fields)
	}
}

// --- Wrap depth enforcement ---

func TestWrap_DepthCap(t *testing.T) {
	base := erro.New("root cause")
	err := error(base)
	for i := 0; i < erro.MaxWrapDepth*3; i++ {
		err = erro.Wrap(err, fmt.Sprintf("layer %d", i))
	}

	depth := 0
	for current := err; current != nil; current = errors.Unwrap(current) {
		depth++
		if depth > erro.MaxWrapDepth+2 {
			t.Fatalf("wrap depth not enforced: chain longer than %d", erro.MaxWrapDepth+2)
		}
	}
	if !errors.Is(err, base) {
		t.Error("depth-capped chain must still match the root error")
	}
}

// --- ID uniqueness under load ---

func TestErrorID_ParallelUniqueness(t *testing.T) {
	const goroutines = 8
	const perGoroutine = 5000

	ids := make([][]string, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			local := make([]string, 0, perGoroutine)
			for i := 0; i < perGoroutine; i++ {
				local = append(local, erro.New("burst").ID())
			}
			ids[g] = local
		}(g)
	}
	wg.Wait()

	seen := make(map[string]struct{}, goroutines*perGoroutine)
	for _, batch := range ids {
		for _, id := range batch {
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate error ID generated: %s", id)
			}
			seen[id] = struct{}{}
		}
	}
}

// --- Stack classification and capture ---

func TestStackFrameClassification_RealWorldNames(t *testing.T) {
	stdlib := []string{
		"net/http.(*Client).Do",
		"encoding/json.Marshal",
		"log/slog.(*Logger).Error",
		"database/sql.(*DB).QueryContext",
		"fmt.Println",
		"runtime.gopanic",
	}
	for _, name := range stdlib {
		frame := erro.StackFrame{FullName: name}
		if frame.IsUser() {
			t.Errorf("%s must not be classified as user code", name)
		}
	}

	user := []string{
		"main.main",
		"github.com/some/app/internal/service.(*Service).Process",
		"github.com/some/app/cmd/server.run",
		"gitlab.example.com/team/project.Handler",
	}
	for _, name := range user {
		frame := erro.StackFrame{FullName: name}
		if !frame.IsUser() {
			t.Errorf("%s must be classified as user code", name)
		}
		if frame.IsErroInternal() {
			t.Errorf("%s must not be classified as erro-internal", name)
		}
	}

	if !(erro.StackFrame{FullName: "github.com/maxbolgarin/erro.New"}).IsErroInternal() {
		t.Error("erro's own frames must classify as internal")
	}
}

func TestStackTrace_MaxFramesEnforced(t *testing.T) {
	err := deepStackError(8, erro.StrictStackTraceConfig())
	if got := len(err.Stack()); got > 3 {
		t.Errorf("Strict config allows 3 frames, got %d", got)
	}

	err = deepStackError(15, erro.ProductionStackTraceConfig())
	if got := len(err.Stack()); got > 10 {
		t.Errorf("Production config allows 10 frames, got %d", got)
	}
}

func deepStackError(depth int, cfg *erro.StackTraceConfig) erro.Error {
	if depth <= 0 {
		return erro.New("deep", erro.StackTrace(cfg))
	}
	return deepStackError(depth-1, cfg)
}

func TestStackTopFrame_AllEntryPoints(t *testing.T) {
	const wantFunc = "TestStackTopFrame_AllEntryPoints"

	assertTop := func(label string, err erro.Error) {
		t.Helper()
		stack := err.Stack()
		if len(stack) == 0 {
			t.Errorf("%s: no stack captured", label)
			return
		}
		if !strings.Contains(stack[0].FullName, wantFunc) {
			t.Errorf("%s: top frame is %q, want the caller %q", label, stack[0].FullName, wantFunc)
		}
	}

	assertTop("New", erro.New("a", erro.StackTrace()))
	assertTop("Wrap", erro.Wrap(errors.New("inner"), "b", erro.StackTrace()))

	tpl := erro.NewTemplate("tpl: %s", erro.StackTrace())
	assertTop("Template.New with verb args", tpl.New("x"))
	assertTop("Template.New without verb args", tpl.New())
	assertTop("Template.Wrap", tpl.Wrap(errors.New("inner"), "x"))

	list := erro.NewList()
	list.New("c", erro.StackTrace())
	assertTop("List.New", list.First())
}

func stackSkipHelper() erro.Error {
	return erro.New("from helper", erro.StackTraceWithSkip(1))
}

func TestStackTraceWithSkip_SkipsCallerFrames(t *testing.T) {
	err := stackSkipHelper()
	stack := err.Stack()
	if len(stack) == 0 {
		t.Fatal("no stack captured")
	}
	if strings.Contains(stack[0].FullName, "stackSkipHelper") {
		t.Errorf("skip=1 must hide the helper frame, top is %q", stack[0].FullName)
	}
	if !strings.Contains(stack[0].FullName, "TestStackTraceWithSkip_SkipsCallerFrames") {
		t.Errorf("top frame should be the helper's caller, got %q", stack[0].FullName)
	}
}

// --- Formatting robustness ---

func TestFormatVerb_QuotedAndUnknown(t *testing.T) {
	err := erro.New("some error")

	quoted := fmt.Sprintf("%q", err)
	if quoted != `"some error"` {
		t.Errorf("expected quoted error string, got %s", quoted)
	}

	unknown := fmt.Sprintf("%d", err)
	if !strings.Contains(unknown, "some error") {
		t.Errorf("unsupported verb should still surface the message, got %s", unknown)
	}
}

type typedNilStringer struct{ name string }

func (s *typedNilStringer) String() string { return s.name }

func TestFormatVerbs_Correctness(t *testing.T) {
	tests := []struct {
		name    string
		message string
		args    []any
		want    string
	}{
		{"hex verb", "code: %x", []any{255}, "code: ff"},
		{"upper hex", "code: %X", []any{255}, "code: FF"},
		{"precision float", "value: %.2f", []any{3.14159}, "value: 3.14"},
		{"width int", "id: %05d", []any{42}, "id: 00042"},
		{"quoted", "input: %q", []any{"raw"}, `input: "raw"`},
		{"type verb", "got: %T", []any{42}, "got: int"},
		{"bool verb", "flag: %t", []any{true}, "flag: true"},
		{"wrap verb acts as value", "cause: %w", []any{errors.New("inner")}, "cause: inner"},
		{"percent literal with fields", "90% failure", []any{"key", "val"}, "90% failure key=val"},
		{"escaped percent", "load: 100%%", []any{"key", "val"}, "load: 100% key=val"},
		{"star width not consumed", "pad: %*d", []any{"key", "val"}, "pad: %*d key=val"},
		{"indexed verb not consumed", "n: %[1]d", []any{"key", "val"}, "n: %[1]d key=val"},
		{"oversized width rejected", "big: %99999d", []any{"key", "val"}, "big: %99999d key=val"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := erro.New(tc.message, tc.args...)
			if got := err.Error(); got != tc.want {
				t.Errorf("New(%q, %v) = %q, want %q", tc.message, tc.args, got, tc.want)
			}
		})
	}
}

func TestFormatVerbs_TypedNilDoesNotPanic(t *testing.T) {
	var nilStringer *typedNilStringer

	err := erro.New("value: %s", nilStringer)
	if err == nil {
		t.Fatal("expected an error value")
	}
	if msg := err.Error(); !strings.Contains(msg, "value:") {
		t.Errorf("unexpected message: %q", msg)
	}

	err = erro.New("msg", "key", nilStringer)
	if !strings.Contains(err.Error(), "key=") {
		t.Errorf("typed-nil field value should render, got %q", err.Error())
	}
}

func TestFormatVerbs_OptionsNeverConsumed(t *testing.T) {
	err := erro.New("op failed: %s", erro.ClassNotFound, erro.SeverityHigh)
	if err.Class() != erro.ClassNotFound {
		t.Errorf("class consumed as a format argument, got class %q and message %q",
			err.Class(), err.Error())
	}
	if strings.Contains(err.Error(), "not_found") {
		t.Errorf("classifier leaked into the message: %q", err.Error())
	}
}

func TestTemplate_NoDoubleFormatting(t *testing.T) {
	// A user-supplied value containing a verb must not trigger a second
	// formatting pass that eats the template's metadata.
	err := erro.NotFoundError.New("row %s")
	if err.Class() != erro.ClassNotFound {
		t.Errorf("template class lost to injected verb, got %q (message %q)",
			err.Class(), err.Error())
	}
	if !strings.Contains(err.Error(), "row %s not found") {
		t.Errorf("injected verb must stay literal, got %q", err.Error())
	}
}

func TestTemplate_MatchesNewFormatting(t *testing.T) {
	tpl := erro.NewTemplate("code: %x")
	fromTemplate := tpl.New(255).Error()
	fromNew := erro.New("code: %x", 255).Error()
	if fromTemplate != fromNew {
		t.Errorf("template and New disagree: %q vs %q", fromTemplate, fromNew)
	}

	plain := erro.NewTemplate("90% done", erro.ClassInternal).New("key", "val")
	if got := plain.Error(); !strings.Contains(got, "90% done") {
		t.Errorf("verb-less template message corrupted: %q", got)
	}
}

func TestAppendValue_BytesUTF8Boundary(t *testing.T) {
	prefix := strings.Repeat("a", erro.MaxValueLength-1)
	value := []byte(prefix + "éé") // 2-byte runes straddling the cut point

	err := erro.New("msg", "data", value)
	if !utf8.ValidString(err.Error()) {
		t.Error("truncated []byte value produced invalid UTF-8 in error message")
	}
}
