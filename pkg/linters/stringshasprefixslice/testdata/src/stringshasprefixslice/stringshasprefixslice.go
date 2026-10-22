package stringshasprefixslice

import "strings"

func badLiteral(s string) string {
	if strings.HasPrefix(s, "v") { // want `use strings\.CutPrefix\(s, "v"\) instead of strings\.HasPrefix followed by slicing with len\("v"\)`
		s = s[len("v"):]
	}
	return s
}

func badVar(s, prefix string) string {
	if strings.HasPrefix(s, prefix) { // want `use strings\.CutPrefix\(s, prefix\) instead of strings\.HasPrefix followed by slicing with len\(prefix\)`
		return s[len(prefix):]
	}
	return s
}

func badWithElse(s, prefix string) string {
	if strings.HasPrefix(s, prefix) { // want `use strings\.CutPrefix\(s, prefix\) instead`
		return strings.TrimSpace(s[len(prefix):])
	} else {
		return ""
	}
}

func goodCutPrefix(s, prefix string) string {
	if rest, ok := strings.CutPrefix(s, prefix); ok {
		return rest
	}
	return s
}

func goodDifferentPrefix(s, prefix, other string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(other):]
	}
	return s
}

func goodBoundedSlice(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix) : len(prefix)+1]
	}
	return s
}

func goodNoSlice(s, prefix string) bool {
	if strings.HasPrefix(s, prefix) {
		return true
	}
	return false
}

func suppressed(s string) string {
	//nolint:stringshasprefixslice
	if strings.HasPrefix(s, "x") {
		return s[len("x"):]
	}
	return s
}
