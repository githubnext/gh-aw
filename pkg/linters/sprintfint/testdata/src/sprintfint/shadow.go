// Package sprintfint contains a test fixture for the sprintfint analyzer
// verifying that when the default qualifier name is shadowed by a local
// parameter, no SuggestedFix is emitted (the diagnostic is still reported).
package sprintfint

import "fmt"

// shadowStrconv has a parameter named "strconv" that shadows the package name.
// The linter should report the diagnostic but emit no SuggestedFix, because
// adding import "strconv" and emitting strconv.Itoa would fail to compile —
// the parameter `strconv int` takes precedence over the package binding.
func shadowStrconv(strconv int, n int) string {
	return fmt.Sprintf("%d", n) // want `use strconv\.Itoa\(x\) instead of fmt\.Sprintf\("%d", x\)`
}
