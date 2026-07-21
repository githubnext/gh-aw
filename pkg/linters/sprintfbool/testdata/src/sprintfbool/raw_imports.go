package sprintfbool

import (
	`fmt`
	`strconv`
)

func badRawImports(b bool) string {
	return fmt.Sprintf("%t", b) // want `use strconv\.FormatBool\(b\) instead of fmt\.Sprintf\("%t", b\)`
}

func goodRawImports(b bool) string {
	return strconv.FormatBool(b)
}
