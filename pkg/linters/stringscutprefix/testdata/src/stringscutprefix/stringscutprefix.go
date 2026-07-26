package stringscutprefix

import "strings"

// flagged: HasPrefix check then TrimPrefix in the body
func processPrefix(s string) string {
	if strings.HasPrefix(s, "foo") { // want `strings\.HasPrefix \+ strings\.TrimPrefix can be replaced with strings\.CutPrefix`
		return strings.TrimPrefix(s, "foo")
	}
	return s
}

// not flagged: HasPrefix but no TrimPrefix in body
func checkOnly(s string) bool {
	if strings.HasPrefix(s, "bar") {
		return true
	}
	return false
}

// not flagged: TrimPrefix called with different prefix
func mismatchedPrefix(s string) string {
	if strings.HasPrefix(s, "foo") {
		return strings.TrimPrefix(s, "bar")
	}
	return s
}

// flagged: field selector expressions also detected
type Obj struct{ Name string }

func processObj(o Obj) string {
	if strings.HasPrefix(o.Name, "v") { // want `strings\.HasPrefix \+ strings\.TrimPrefix can be replaced with strings\.CutPrefix`
		return strings.TrimPrefix(o.Name, "v")
	}
	return o.Name
}

// not flagged: uses CutPrefix already (control case)
func usesCutPrefix(s string) (string, bool) {
	after, found := strings.CutPrefix(s, "foo")
	return after, found
}

// not flagged: TrimPrefix inside a nested closure is not guaranteed to run here
func closureCase(s string) func() string {
	if strings.HasPrefix(s, "foo") {
		return func() string {
			return strings.TrimPrefix(s, "foo")
		}
	}
	return nil
}

// not flagged: TrimPrefix in a nested branch is not a direct equivalent shape
func nestedBranch(s string, debug bool) string {
	if strings.HasPrefix(s, "foo") {
		if debug {
			return strings.TrimPrefix(s, "foo")
		}
	}
	return s
}

// not flagged: s is reassigned before TrimPrefix runs
func reassignedValue(s string) string {
	if strings.HasPrefix(s, "foo") {
		s = strings.TrimSpace(s)
		return strings.TrimPrefix(s, "foo")
	}
	return s
}

// not flagged: prefix is reassigned before TrimPrefix runs
func reassignedPrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		prefix = strings.TrimSpace(prefix)
		return strings.TrimPrefix(s, prefix)
	}
	return s
}
