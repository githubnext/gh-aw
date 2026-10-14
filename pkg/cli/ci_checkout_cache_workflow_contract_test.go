//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

type ciCheckoutStep struct {
	Name            string            `yaml:"name"`
	ID              string            `yaml:"id"`
	Uses            string            `yaml:"uses"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
}

type ciCheckoutJob struct {
	Permissions map[string]string `yaml:"permissions"`
	Steps       []ciCheckoutStep  `yaml:"steps"`
}

func TestCICheckoutCacheWorkflowContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	for _, name := range []string{"cjs", "cgo", "CWI"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", name+".yml"))
			require.NoError(t, err)
			var workflow struct {
				Jobs map[string]ciCheckoutJob `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(content, &workflow))
			producer := workflow.Jobs["checkout-cache"]
			require.NotEmpty(t, producer.Steps)
			checkout := producer.Steps[0]
			require.True(t, strings.HasPrefix(checkout.Uses, "actions/checkout@"))
			require.Equal(t, "false", checkout.With["persist-credentials"])

			key := "checkout-" + strings.ToLower(name) + "-${{ github.run_id }}-${{ github.sha }}"
			saves, restores, cleanups := 0, 0, 0
			for jobName, job := range workflow.Jobs {
				for index, step := range job.Steps {
					if strings.HasPrefix(step.Uses, "actions/cache/save@") {
						saves++
						require.Equal(t, "checkout-cache", jobName)
						require.Equal(t, key, step.With["key"])
						require.Equal(t, ".", step.With["path"])
					}
					if strings.HasPrefix(step.Uses, "actions/cache/restore@") {
						restores++
						verifyCICheckoutFallback(t, job, index, checkout, key)
						require.Equal(t, producer.Steps[len(producer.Steps)-1].With["enableCrossOsArchive"],
							step.With["enableCrossOsArchive"], jobName)
					}
					if step.Name == "Delete checkout cache" {
						cleanups++
						require.Equal(t, key, step.Env["CACHE_KEY"])
					}
				}

			}
			require.Equal(t, 1, saves)
			require.Positive(t, restores)
			require.Equal(t, 1, cleanups)
		})
	}
}

func verifyCICheckoutFallback(t *testing.T, job ciCheckoutJob, index int, checkout ciCheckoutStep, key string) {
	t.Helper()
	step := job.Steps[index]
	require.Equal(t, key, step.With["key"])
	require.Equal(t, ".", step.With["path"])
	require.Empty(t, step.With["restore-keys"], "never use a stale revision")
	require.NotEqual(t, "true", step.With["fail-on-cache-miss"])
	require.False(t, step.ContinueOnError, "do not hide restore errors")
	require.Equal(t, "restore-checkout", step.ID)
	require.Less(t, index+1, len(job.Steps))
	fallback := job.Steps[index+1]
	require.Equal(t, "steps.restore-checkout.outputs.cache-hit != 'true'", fallback.If)
	require.Equal(t, checkout.Uses, fallback.Uses)
	require.Equal(t, map[string]string{
		"ref":                 "${{ github.sha }}",
		"fetch-depth":         checkout.With["fetch-depth"],
		"persist-credentials": "false",
	}, fallback.With)
	require.False(t, fallback.ContinueOnError, "checkout failures must fail the job")
	require.Equal(t, "read", job.Permissions["contents"])
}
