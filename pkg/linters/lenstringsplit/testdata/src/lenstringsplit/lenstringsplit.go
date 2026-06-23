package lenstringsplit

import "strings"

// flagged: len(strings.Split(...)) inside a return statement.
func countLines(content string) int {
	return len(strings.Split(content, "\n")) // want `len\(strings\.Split\(\.\.\.`
}

// flagged: len(strings.Split(...)) assigned to a variable.
func countFields(s string) int {
	n := len(strings.Split(s, ",")) // want `len\(strings\.Split\(\.\.\.`
	return n
}

// not flagged: the split result is actually iterated or indexed.
func splitAndUse(s string) []string {
	return strings.Split(s, "/")
}

// not flagged: len applied to a pre-split slice, not a strings.Split call.
func countFromSlice(parts []string) int {
	return len(parts)
}

// not flagged: strings.Fields is a different function.
func countWords(s string) int {
	return len(strings.Fields(s))
}
