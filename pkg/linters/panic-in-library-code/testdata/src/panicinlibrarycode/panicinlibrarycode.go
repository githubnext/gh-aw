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

// ok: panic in init() — init() cannot return an error.
func init() {
	panic("init panic is allowed") // should not be flagged
}

// ok: panic inside a sync.Once.Do callback.
var once sync.Once

func setupOnce() {
	once.Do(func() {
		panic("once.Do panic is allowed") // should not be flagged
	})
}

// ok: panic whose message starts with "BUG:" — invariant violation.
func invariantCheck(x int) {
	if x < 0 {
		panic("BUG: x must be non-negative") // should not be flagged
	}
}

// ok: fmt.Sprintf whose format string starts with "BUG:".
func invariantCheckFormatted(x int) {
	if x < 0 {
		panic(fmt.Sprintf("BUG: x must be non-negative, got %d", x)) // should not be flagged
	}
}

// documentedPanic panics if the argument is negative.
// Panics if n < 0 — callers must ensure n is non-negative.
func documentedPanic(n int) {
	if n < 0 {
		panic("n must be non-negative") // should not be flagged — documented panic contract
	}
}
