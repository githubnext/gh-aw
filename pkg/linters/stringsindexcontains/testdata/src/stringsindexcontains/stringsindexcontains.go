package stringsindexcontains

import "strings"

func badContains(s, sub string) bool {
	return strings.Index(s, sub) != -1 // want `use strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func badContainsGEQ(s, sub string) bool {
	return strings.Index(s, sub) >= 0 // want `use strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func badNotContains(s, sub string) bool {
	return strings.Index(s, sub) == -1 // want `use !strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func badNotContainsLT(s, sub string) bool {
	return strings.Index(s, sub) < 0 // want `use !strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func badYodaContains(s, sub string) bool {
	return -1 != strings.Index(s, sub) // want `use strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func badYodaNotContains(s, sub string) bool {
	return -1 == strings.Index(s, sub) // want `use !strings\.Contains\(s, sub\) instead of strings\.Index comparison`
}

func goodContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func goodNotContains(s, sub string) bool {
	return !strings.Contains(s, sub)
}

func goodIndexUsedForPosition(s, sub string) int {
	// Using the index value itself (not just for containment check) is fine.
	return strings.Index(s, sub)
}

func goodIndexComparesNonMinusOne(s, sub string) bool {
	// Comparing against a value other than -1/0 is fine.
	return strings.Index(s, sub) > 3
}
