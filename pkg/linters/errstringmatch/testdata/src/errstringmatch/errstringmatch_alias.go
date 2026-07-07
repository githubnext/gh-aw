package errstringmatch

import (
	str "strings"
)

// flagged: aliased strings import still resolves to strings package
func checkErrorAliased(err error) bool {
	return str.Contains(err.Error(), "not found") // want `avoid strings\.Contains\(err\.Error\(\)`
}

// not flagged: shadowing local named "strings" — not the stdlib package
type fakeStrings struct{}

func (fakeStrings) Contains(s, substr string) bool { return false }

func checkShadowedStrings(err error) bool {
	var strings fakeStrings
	return strings.Contains(err.Error(), "prefix")
}
