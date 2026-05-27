package ossetenvlibrary

import "os"

// BadSetenv calls os.Setenv and should be flagged.
func BadSetenv() {
	os.Setenv("KEY", "val") // want "os.Setenv mutates the process environment"
}

// BadUnsetenv calls os.Unsetenv and should be flagged.
func BadUnsetenv() {
	os.Unsetenv("KEY") // want "os.Unsetenv mutates the process environment"
}

// OkGetenv calls os.Getenv and should NOT be flagged.
func OkGetenv() string {
	return os.Getenv("KEY")
}

type fakeOS struct{}

func (fakeOS) Setenv(_, _ string) error { return nil }

// LocalVarNamedOS should not be flagged just because the variable is named os.
func LocalVarNamedOS() {
	os := fakeOS{}
	os.Setenv("KEY", "val")
}
