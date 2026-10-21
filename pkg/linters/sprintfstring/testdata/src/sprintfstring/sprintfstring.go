// Package sprintfstring is the test fixture for the sprintfstring analyzer.
package sprintfstring

import "fmt"

type myString string

func bad(s string) string {
	return fmt.Sprintf("%s", s) // want `use s directly instead of fmt\.Sprintf\("%s", s\)`
}

func badRaw(s string) string {
	return fmt.Sprintf(`%s`, s) // want `use s directly instead of fmt\.Sprintf`
}

func goodNamedString(s myString) string {
	return fmt.Sprintf("%s", s)
}

func goodError(err error) string {
	return fmt.Sprintf("%s", err)
}

func goodBytes(b []byte) string {
	return fmt.Sprintf("%s", b)
}

func goodExtraText(s string) string {
	return fmt.Sprintf("name: %s", s)
}

func goodMultiple(a, b string) string {
	return fmt.Sprintf("%s%s", a, b)
}

func suppressed(s string) string {
	return fmt.Sprintf("%s", s) //nolint:sprintfstring
}
