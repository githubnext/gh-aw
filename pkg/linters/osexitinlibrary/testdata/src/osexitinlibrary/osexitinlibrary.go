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
