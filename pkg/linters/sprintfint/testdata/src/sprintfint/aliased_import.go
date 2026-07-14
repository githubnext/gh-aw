// Package sprintfint contains a test fixture for the sprintfint analyzer
// verifying that when "strconv" is already imported under an alias, the
// suggested fix uses the alias as the qualifier instead of the default name.
package sprintfint

import (
	sc "strconv"
	"fmt"
)

// badAliased has strconv imported as "sc"; the fix must emit sc.Itoa, not strconv.Itoa.
func badAliased(n int) string {
	return fmt.Sprintf("%d", n) // want `use strconv\.Itoa\(x\) instead of fmt\.Sprintf\("%d", x\)`
}

// goodAliasedAlready uses the alias directly — not flagged.
func goodAliasedAlready(n int) string {
	return sc.Itoa(n)
}
