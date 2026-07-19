// Package trimleftright is the test fixture for the trimleftright analyzer.
package trimleftright

import "strings"

// bad: TrimLeft with multi-char literal — likely meant TrimPrefix
func badTrimLeft(s string) string {
	return strings.TrimLeft(s, "foo") // want `strings\.TrimLeft with a multi-character cutset`
}

// bad: TrimRight with repeated alphanumeric cutset — likely meant TrimSuffix
func badTrimRight(s string) string {
	return strings.TrimRight(s, "barr") // want `strings\.TrimRight with a multi-character cutset`
}

// good: TrimLeft with single character — valid cutset usage
func goodSingleChar(s string) string {
	return strings.TrimLeft(s, " ")
}

// good: TrimRight with empty string — valid (noop)
func goodEmpty(s string) string {
	return strings.TrimRight(s, "")
}

// good: TrimPrefix — correct function for prefix removal
func goodTrimPrefix(s string) string {
	return strings.TrimPrefix(s, "foo")
}

// good: TrimSuffix — correct function for suffix removal
func goodTrimSuffix(s string) string {
	return strings.TrimSuffix(s, "bar")
}

// good: TrimLeft with single Unicode rune
func goodUnicodeRune(s string) string {
	return strings.TrimLeft(s, "→")
}

// good: intentional multi-character cutset
func goodIntentionalCutset(s string) string {
	return strings.TrimLeft(s, "aeiou")
}

// good: alphanumeric literal with no repeated runes
func goodNoRepeatedAlnum(s string) string {
	return strings.TrimLeft(s, "abc")
}

// suppressed: nolint directive suppresses the diagnostic
func suppressed(s string) string {
	return strings.TrimLeft(s, "foo") //nolint:trimleftright
}
