package panicinlibrarycode

import (
	"errors"
	"fmt"
	"sync"
)

// bad: panic in a pkg/ package.
func riskyFunction() {
	panic("something went wrong") // want `avoid panic in library code; return an error instead`
}

// bad: panic with a value
func anotherRiskyFunction() {
	panic(errors.New("error")) // want `avoid panic in library code; return an error instead`
}

// bad: panic with fmt.Sprintf that does not start with BUG:
func yetAnotherRiskyFunction(n int) {
	panic(fmt.Sprintf("unexpected value: %d", n)) // want `avoid panic in library code; return an error instead`
}

// ok: function that returns an error instead of panicking.
func safeFunction() error {
	return nil
}

// ok: user-defined panic function (not the builtin)
type myType struct{}

func (m myType) panic(msg string) {
	// This is a custom method, not builtin panic
}

func callCustomPanic() {
	m := myType{}
	m.panic("this is ok") // Should not be flagged
}

// ok: panic in top-level init() — init() cannot return an error.
func init() {
	panic("startup registration failure") // should not be flagged
}

// ok: panic inside a sync.Once.Do callback.
var once sync.Once

func allowedSyncOncePanic() {
	once.Do(func() {
		panic("lazy init failure") // should not be flagged
	})
}

// ok: panic whose message starts with "BUG:" — invariant violation.
func allowedBUGPanic() {
	panic(fmt.Sprintf("BUG: unreachable: %v", errors.New("boom"))) // should not be flagged
}

// ok: panic with plain "BUG:" string literal.
func invariantCheck(x int) {
	if x < 0 {
		panic("BUG: x must be non-negative") // should not be flagged
	}
}

// documentedPreconditionPanics panics if the caller passes an invalid mode.
func documentedPreconditionPanics(mode string) {
	if mode == "" {
		panic("invalid mode") // should not be flagged — documented panic contract
	}
}

// ok: method named init is NOT a top-level init; its panic should be flagged.
type myInitType struct{}

func (m myInitType) init() {
	panic("method init panic") // want `avoid panic in library code; return an error instead`
}

func nolintPreviousLineSuppressed() {
	//nolint:panicinlibrarycode
	panic("intentional panic for compatibility")
}

func nolintSameLineSuppressed() {
	panic("intentional panic for compatibility") //nolint:panicinlibrarycode
}
