package osexitinlibrary

import "os"

// bad: os.Exit in a pkg/ package.
func stopProcess() {
	os.Exit(1) // want `os.Exit called in library package`
}

// ok: helper that does NOT call os.Exit.
func doWork() error {
	return nil
}

func suppressedPreviousLine() {
	//nolint:osexitinlibrary
	os.Exit(1)
}

func suppressedSameLine() {
	os.Exit(1) //nolint:osexitinlibrary
}

// shadowed: a local variable named "os" with an Exit method should NOT be flagged.
type osLike struct{}

func (osLike) Exit(int) {}

func shadowedOs() {
	os := osLike{}
	os.Exit(1) // should NOT be flagged
}
