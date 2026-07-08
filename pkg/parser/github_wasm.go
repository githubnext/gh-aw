//go:build js || wasm

package parser

import (
	"errors"

	"github.com/github/gh-aw/pkg/envutil"
)

func GetGitHubToken() (string, error) {
	// Wasm callers do not use the package logger, so pass nil to suppress debug
	// logging while still centralizing environment variable access.
	if token := envutil.GetStringFromEnv("GITHUB_TOKEN", "", nil); token != "" {
		return token, nil
	}
	if token := envutil.GetStringFromEnv("GH_TOKEN", "", nil); token != "" {
		return token, nil
	}
	return "", errors.New("GitHub token not available in Wasm (set GITHUB_TOKEN or GH_TOKEN environment variable)")
}
