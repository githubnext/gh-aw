//go:build !integration

package parser

import (
	"testing"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestGithubTokenFromEnv(t *testing.T) {
	t.Run("prefers GITHUB_TOKEN over GH_TOKEN", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "github-token")
		t.Setenv("GH_TOKEN", "gh-token")

		assert.Equal(t, "github-token", githubTokenFromEnv(logger.New("parser:test")))
	})

	t.Run("falls back to GH_TOKEN when GITHUB_TOKEN is empty", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "gh-token")

		assert.Equal(t, "gh-token", githubTokenFromEnv(nil))
	})

	t.Run("returns empty when neither token is set", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_TOKEN", "")

		assert.Empty(t, githubTokenFromEnv(nil))
	})
}
