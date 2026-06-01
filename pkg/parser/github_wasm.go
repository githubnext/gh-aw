//go:build js || wasm

package parser

import (
	"errors"
	"os"
)

func GetGitHubToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token, nil
	}
	return "", errors.New("GitHub token not available in Wasm (set GITHUB_TOKEN or GH_TOKEN environment variable)")
}
