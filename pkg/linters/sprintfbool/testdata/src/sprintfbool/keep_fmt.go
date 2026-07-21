package sprintfbool

import "fmt"

func badKeepFmt(b bool) string {
	return fmt.Sprintf("%t", b) // want `use strconv\.FormatBool\(b\) instead of fmt\.Sprintf\("%t", b\)`
}

func stillUsesFmt(b bool) string {
	return fmt.Sprintf("%v", b)
}
