// Package b tests the user-defined-recover false-negative guard in isolation.
// By declaring a package-level function named recover() we shadow the built-in;
// the linter must not accept that as a valid panic guard.
package b

// recover shadows the built-in recover. //nolint:predeclared
func recover() {}

// userDefinedRecoverGoroutine uses the local recover() (not the built-in) inside
// the defer literal — the goroutine is still unprotected and must be flagged.
func userDefinedRecoverGoroutine() {
	go func() { // want `goroutine launched via a function literal without a top-level defer/recover`
		defer func() {
			recover() // calls the local func, not the built-in
		}()
		panic("oops")
	}()
}
