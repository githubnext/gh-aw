package parser

// SetReadFileFuncForTest overrides the file reading function for testing.
// This enables testing virtual filesystem behavior in native (non-wasm) builds.
// Returns a cleanup function that restores the original.
func SetReadFileFuncForTest(fn func(string) ([]byte, error)) func() {
	original := readFileFunc
	readFileFunc = fn
	return func() {
		readFileFunc = original
	}
}
