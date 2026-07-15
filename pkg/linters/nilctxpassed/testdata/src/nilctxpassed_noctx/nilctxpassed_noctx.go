// Package nilctxpassed_noctx tests the nilctxpassed analyzer for the case
// where the analyzed package does NOT directly import "context" but still
// passes nil to a context.Context parameter via an external function.
package nilctxpassed_noctx

import "ctxhelper"

// BadNoDirectImport passes nil to a context.Context param without importing context.
func BadNoDirectImport() {
	ctxhelper.TakesCtx(nil) // want `nil passed as context\.Context; use context\.Background\(\) or context\.TODO\(\) instead`
}

// GoodNilNonContext passes nil to a non-context pointer param — not flagged.
func GoodNilNonContext() {
	ctxhelper.TakesOtherPointer(nil)
}
