package osgetenvlibrary

import "os"

// BadGetenv calls os.Getenv and should be flagged.
func BadGetenv() string {
	return os.Getenv("CONFIG_KEY") // want "os.Getenv couples the library to the process environment"
}

// BadLookupEnv calls os.LookupEnv and should be flagged.
func BadLookupEnv() (string, bool) {
	return os.LookupEnv("CONFIG_KEY") // want "os.LookupEnv couples the library to the process environment"
}

// OkSetenv calls os.Setenv (not our concern here) and should NOT be flagged.
func OkSetenv() error {
	return os.Setenv("KEY", "val")
}

type fakeOS struct{}

func (fakeOS) Getenv(_ string) string { return "" }

// LocalVarNamedOS should not be flagged just because the variable is named os.
func LocalVarNamedOS() string {
	os := fakeOS{}
	return os.Getenv("KEY")
}

// SuppressedGetenv uses a nolint directive and should not be flagged.
func SuppressedGetenv() string {
	return os.Getenv("CONFIG_KEY") //nolint:osgetenvlibrary
}
