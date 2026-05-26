package ossetenvlibrary

import o "os"

// BadAliasSetenv calls alias-imported os.Setenv and should be flagged.
func BadAliasSetenv() {
	o.Setenv("KEY", "val") // want "os.Setenv mutates the process environment"
}

// BadAliasUnsetenv calls alias-imported os.Unsetenv and should be flagged.
func BadAliasUnsetenv() {
	o.Unsetenv("KEY") // want "os.Unsetenv mutates the process environment"
}
