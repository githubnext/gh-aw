package osexitinlibrary

import xos "os"

// flagged: aliased os import still resolves to os package
func stopProcessAliased() {
	xos.Exit(1) // want `os.Exit called in library package`
}

// not flagged: shadowing local named "os" — not the stdlib package
type fakeOS struct{}

func (fakeOS) Exit(code int) {}

func notFlaggedShadowedOS() {
	var os fakeOS
	os.Exit(1)
}
