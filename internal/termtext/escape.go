// Package termtext separates untrusted text from terminal control sequences.
package termtext

import (
	"strconv"
	"strings"
	"unicode"
)

// Escape renders control characters visibly and replaces invalid UTF-8.
// Call it before adding application-owned terminal styling.
func Escape(s string) string {
	var out strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			quoted := strconv.QuoteRune(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
