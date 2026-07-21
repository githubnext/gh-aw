package sprintfbool

import "fmt"

func badOne(b bool) string {
	return fmt.Sprintf("%t", b) // want `use strconv\.FormatBool\(b\) instead of fmt\.Sprintf\("%t", b\)`
}

func badTwo(b bool) string {
	return fmt.Sprintf("%t", b) // want `use strconv\.FormatBool\(b\) instead of fmt\.Sprintf\("%t", b\)`
}
