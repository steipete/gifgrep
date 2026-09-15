package termtext

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestEscape(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Café 猫 🦞", "Café 猫 🦞"},
		{"\x1b]52;c;payload\a", `\x1b]52;c;payload\a`},
		{"\u009b31m\r\n\t", `\u009b31m\r\n\t`},
		{string([]byte{0x9b}) + "31m", "�31m"},
	} {
		if got := Escape(tc.input); got != tc.want {
			t.Errorf("Escape(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
}

func FuzzEscape(f *testing.F) {
	for _, s := range []string{"Café 猫 🦞", "\x1b]52;c;payload\a", "\u009b31m", string([]byte{0x9b})} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := Escape(s)
		if !utf8.ValidString(got) || strings.ContainsFunc(got, unicode.IsControl) {
			t.Fatalf("unsafe result %q", got)
		}
		if Escape(got) != got {
			t.Fatalf("escaping is not idempotent: %q", got)
		}
	})
}
