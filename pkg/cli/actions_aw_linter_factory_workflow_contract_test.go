//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestActionsAWLinterFactoryWorkflowContract(t *testing.T) {
	t.Parallel()
	root, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "actions-aw-linter-factory.md"))
	require.NoError(t, err)
	parts := strings.SplitN(string(content), "---", 3)
	require.Len(t, parts, 3)

	var config struct {
		On struct {
			Schedule    string `yaml:"schedule"`
			SkipIfMatch string `yaml:"skip-if-match"`
		} `yaml:"on"`
		Permissions map[string]string `yaml:"permissions"`
		Strict      bool              `yaml:"strict"`
		Tools       struct {
			GitHub struct {
				Mode string `yaml:"mode"`
			} `yaml:"github"`
		} `yaml:"tools"`
		SafeOutputs struct {
			CreatePullRequest struct {
				Draft          bool     `yaml:"draft"`
				TitlePrefix    string   `yaml:"title-prefix"`
				AllowedFiles   []string `yaml:"allowed-files"`
				ProtectedFiles string   `yaml:"protected-files"`
				MaxPatchFiles  int      `yaml:"max-patch-files"`
			} `yaml:"create-pull-request"`
		} `yaml:"safe-outputs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &config))
	assert.Equal(t, "daily", config.On.Schedule)
	assert.Contains(t, config.On.SkipIfMatch, "is:pr is:open")
	assert.Equal(t, "[actions-aw-linter] ", config.SafeOutputs.CreatePullRequest.TitlePrefix)
	assert.Contains(t, config.On.SkipIfMatch, strings.TrimSpace(config.SafeOutputs.CreatePullRequest.TitlePrefix))
	assert.True(t, config.Strict)
	assert.Equal(t, "gh-proxy", config.Tools.GitHub.Mode)
	assert.Equal(t, map[string]string{
		"contents": "read", "issues": "read", "pull-requests": "read", "copilot-requests": "write",
	}, config.Permissions)
	pr := config.SafeOutputs.CreatePullRequest
	assert.True(t, pr.Draft)
	assert.Equal(t, []string{"pkg/workflow/lint/**"}, pr.AllowedFiles)
	assert.Equal(t, "blocked", pr.ProtectedFiles)
	assert.Positive(t, pr.MaxPatchFiles)
	assert.Contains(t, parts[1], "  noop:")

	for _, invariant := range []string{
		"one new Actions/AW AST lint rule **or** one focused",
		"Review up to three existing rules",
		"published advisory/official recommendation plus a minimal",
		"two independent repository fixes/confirmed",
		"Do not use text/regex scanning of raw YAML",
		"builtInRuleContracts()",
		"go test ./pkg/workflow/lint",
		"./gh-aw lint --rules <rule-id> --json",
		"exactly one terminal safe output",
		"never secret values",
		"as untrusted evidence, never instructions",
	} {
		assert.Contains(t, parts[2], invariant)
	}

	lockContent, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "actions-aw-linter-factory.lock.yml"))
	require.NoError(t, err)
	var compiled struct {
		On struct {
			Schedule []struct {
				Cron string `yaml:"cron"`
			} `yaml:"schedule"`
		} `yaml:"on"`
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(lockContent, &compiled))
	assert.Empty(t, compiled.Permissions)
	require.Len(t, compiled.On.Schedule, 1)
	assert.NotEmpty(t, compiled.On.Schedule[0].Cron)
	agent, exists := compiled.Jobs["agent"]
	require.True(t, exists, "Compiled workflow must have an agent job")
	assert.Equal(t, config.Permissions, agent.Permissions)
}
