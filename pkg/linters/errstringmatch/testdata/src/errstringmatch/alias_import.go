package errstringmatch

import str "strings"

// aliased import: str.Contains(err.Error(), ...) should still be flagged.
func checkErrorAlias(err error) bool {
	return str.Contains(err.Error(), "not found") // want `avoid strings\.Contains\(err\.Error\(\)`
}
