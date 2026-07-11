// Package sprintfint contains a single-use test fixture for the sprintfint
// analyzer: the file imports only "fmt" and uses fmt.Sprintf("%d", n) exactly
// once, exercising the strconv-add and fmt-removal paths in the suggested fix.
package sprintfint

import "fmt"

// singleUseFmt is the only "fmt" reference in this file, so the fix must add
// "strconv" and remove the now-unused "fmt" import.
func singleUseFmt(n int) string {
	return fmt.Sprintf("%d", n) // want `use strconv\.Itoa\(x\) instead of fmt\.Sprintf\("%d", x\)`
}
