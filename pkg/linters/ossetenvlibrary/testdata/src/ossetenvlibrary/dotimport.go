package ossetenvlibrary

import . "os"

// BadDotImportSetenv calls dot-imported os.Setenv and should be flagged.
func BadDotImportSetenv() {
	Setenv("KEY", "val") // want "os.Setenv mutates the process environment"
}

// BadDotImportUnsetenv calls dot-imported os.Unsetenv and should be flagged.
func BadDotImportUnsetenv() {
	Unsetenv("KEY") // want "os.Unsetenv mutates the process environment"
}
