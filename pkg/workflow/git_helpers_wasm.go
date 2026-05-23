//go:build js || wasm

package workflow

import "fmt"

func findGitRoot() string {
	return "."
}

func RunGitCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("git commands not available in Wasm")
}
