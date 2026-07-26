package stringsindexhasprefix

import "strings"

func badHasPrefix(s, sub string) bool {
	return strings.Index(s, sub) == 0 // want `use strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badNotHasPrefix(s, sub string) bool {
	return strings.Index(s, sub) != 0 // want `use !strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badYodaHasPrefix(s, sub string) bool {
	return 0 == strings.Index(s, sub) // want `use strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badYodaNotHasPrefix(s, sub string) bool {
	return 0 != strings.Index(s, sub) // want `use !strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badParenHasPrefix(s, sub string) bool {
	return (strings.Index(s, sub)) == 0 // want `use strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badParenNotHasPrefix(s, sub string) bool {
	return (strings.Index(s, sub)) != 0 // want `use !strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badParenYodaHasPrefix(s, sub string) bool {
	return 0 == (strings.Index(s, sub)) // want `use strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func badParenYodaNotHasPrefix(s, sub string) bool {
	return 0 != (strings.Index(s, sub)) // want `use !strings\.HasPrefix\(s, sub\) instead of strings\.Index comparison`
}

func goodHasPrefix(s, sub string) bool {
	return strings.HasPrefix(s, sub)
}

func goodNotHasPrefix(s, sub string) bool {
	return !strings.HasPrefix(s, sub)
}

func goodContainsStyleCheck(s, sub string) bool {
	return strings.Index(s, sub) >= 0
}

func goodNotContainsStyleCheck(s, sub string) bool {
	return strings.Index(s, sub) == -1
}
