package stringsjoinone

import "strings"

// flagged: single-element []string literal, separator is unused.
func joinOne(name string) string {
	return strings.Join([]string{name}, ", ") // want `strings\.Join called with a single-element slice`
}

// flagged: single-element []string literal with a string literal element.
func joinOneLiteral() string {
	return strings.Join([]string{"hello"}, "-") // want `strings\.Join called with a single-element slice`
}

// flagged: single-element []string literal assigned to a variable.
func joinOneAssigned(s string) string {
	result := strings.Join([]string{s}, "/") // want `strings\.Join called with a single-element slice`
	return result
}

// not flagged: two-element slice literal.
func joinTwo(a, b string) string {
	return strings.Join([]string{a, b}, ", ")
}

// not flagged: zero-element slice literal.
func joinZero() string {
	return strings.Join([]string{}, ", ")
}

// not flagged: variable slice, not a literal.
func joinSlice(parts []string) string {
	return strings.Join(parts, ", ")
}

// not flagged: three-element slice literal.
func joinThree(a, b, c string) string {
	return strings.Join([]string{a, b, c}, ", ")
}

// not flagged: a nolint directive suppresses the diagnostic.
func joinSuppressed(s string) string {
	return strings.Join([]string{s}, ", ") //nolint:stringsjoinone
}

// not flagged: separator is a function call (potential side effects).
func joinFuncCallSep(s string, f func() string) string {
	return strings.Join([]string{s}, f())
}

// not flagged: separator receives from a channel (observable side effect).
func joinChanSep(s string, ch <-chan string) string {
	return strings.Join([]string{s}, <-ch)
}
