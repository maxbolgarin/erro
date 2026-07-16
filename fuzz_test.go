package erro

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzApplyFormatVerbs(f *testing.F) {
	f.Add("hello %s", "world", 5)
	f.Add("90% failure", "key", 1)
	f.Add("value: %.2f", "x", 3)
	f.Add("%%%s%d%", "a", 2)
	f.Add("%[1]d %*d % f", "b", 0)
	f.Add("big: %99999999d", "c", 7)
	f.Add("", "", 0)
	f.Add(strings.Repeat("%s", 50), "arg", -1)

	f.Fuzz(func(t *testing.T, format string, s string, n int) {
		out, rest := ApplyFormatVerbs(format, s, n)

		if len(rest) > 2 {
			t.Fatalf("more args returned than passed: %d", len(rest))
		}
		if !strings.ContainsRune(format, '%') && out != format {
			t.Fatalf("format without verbs must pass through unchanged: %q -> %q", format, out)
		}
		if utf8.ValidString(format) && utf8.ValidString(s) && !utf8.ValidString(out) {
			t.Fatalf("valid inputs produced invalid UTF-8: %q", out)
		}
	})
}

func FuzzTruncateUTF8(f *testing.F) {
	f.Add("hello", 3)
	f.Add("你好世界", 7)
	f.Add("", 10)
	f.Add("héllo wörld", 0)
	f.Add(strings.Repeat("é", 100), 33)
	f.Add("abc", -5)

	f.Fuzz(func(t *testing.T, s string, maxBytes int) {
		out := truncateUTF8(s, maxBytes)

		if maxBytes >= 0 && len(out) > maxBytes {
			t.Fatalf("truncated string longer than limit: %d > %d", len(out), maxBytes)
		}
		if !strings.HasPrefix(s, out) {
			t.Fatalf("truncation must produce a prefix: %q from %q", out, s)
		}
		if utf8.ValidString(s) && !utf8.ValidString(out) {
			t.Fatalf("truncation broke UTF-8: %q -> %q", s, out)
		}
	})
}
