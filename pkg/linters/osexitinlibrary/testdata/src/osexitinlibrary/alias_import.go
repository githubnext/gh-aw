package osexitinlibrary

import xos "os"

// aliased import: xos.Exit should still be flagged.
func stopProcessAlias() {
	xos.Exit(1) // want `os.Exit called in library package`
}
