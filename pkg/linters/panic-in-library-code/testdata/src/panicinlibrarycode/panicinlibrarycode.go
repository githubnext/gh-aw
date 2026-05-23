package panicinlibrarycode

import "errors"

// bad: panic in a pkg/ package.
func riskyFunction() {
	panic("something went wrong") // want `avoid panic in library code; return an error instead`
}

// bad: panic with a value
func anotherRiskyFunction() {
	panic(errors.New("error")) // want `avoid panic in library code; return an error instead`
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
