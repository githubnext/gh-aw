//go:build js || wasm

package workflow

import (
	"fmt"
	"os"
)

func findGitRoot() string {
	return "."
}

func GetCurrentGitTag() string {
	if ref := os.Getenv("GITHUB_REF"); len(ref) > 10 && ref[:10] == "refs/tags/" {
		return ref[10:]
	}
	return ""
}

func RunGit(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("git commands not available in Wasm")
}

func RunGitCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("git commands not available in Wasm")
}
