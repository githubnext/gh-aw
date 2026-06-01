//go:build js || wasm

package workflow

import "errors"

func findGitRoot() string {
	return "."
}

func RunGitCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, errors.New("git commands not available in Wasm")
}
