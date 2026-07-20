package sprintfbool

import "fmt"

func badCommentOverlap(b bool) string {
	return fmt.Sprintf("%t", /* rationale */ b) // want `use strconv\.FormatBool\(.*\) instead of fmt\.Sprintf\("%t", .*\)`
}
