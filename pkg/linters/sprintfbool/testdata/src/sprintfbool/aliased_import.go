package sprintfbool

import (
	"fmt"
	sc "strconv"
)

func badAliased(b bool) string {
	return fmt.Sprintf("%t", b) // want `use strconv\.FormatBool\(b\) instead of fmt\.Sprintf\("%t", b\)`
}

func goodAliasedAlready(b bool) string {
	return sc.FormatBool(b)
}
