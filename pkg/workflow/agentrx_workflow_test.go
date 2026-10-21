//go:build !integration

package workflow

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAgentRxSetupUsesHashedDependencyLock(t *testing.T) {
	source, err := os.ReadFile("../../.github/workflows/daily-agentrx-trace-optimizer.md")
	require.NoError(t, err)
	parts := strings.SplitN(string(source), "---", 3)
	require.Len(t, parts, 3)
	var frontmatter struct {
		Steps []struct {
			Name string `yaml:"name"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &frontmatter))
	require.NotEmpty(t, frontmatter.Steps)
	setup := frontmatter.Steps[0].Run
	require.Contains(t, setup, `uv pip sync --require-hashes --only-binary=:all: --python "$AGENTRX_HOME/.venv/bin/python" "${GITHUB_WORKSPACE}/.github/workflows/requirements-agentrx.txt"`)
	require.NotContains(t, setup, "uv pip install", "installing the source project dynamically resolves build and runtime dependencies")
	require.Contains(t, setup, `"$AGENTRX_HOME/.venv/bin/python" "$AGENTRX_HOME/src/run.py" --help`)

	compiled, err := os.ReadFile("../../.github/workflows/daily-agentrx-trace-optimizer.lock.yml")
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(compiled, &workflow))
	var compiledSetup string
	for _, step := range workflow.Jobs["agent"].Steps {
		if step.Name == frontmatter.Steps[0].Name {
			compiledSetup = step.Run
		}
	}
	require.Equal(t, strings.TrimSpace(setup), compiledSetup, "the emitted workflow must preserve the hash-enforced setup")

	lock, err := os.ReadFile("../../.github/workflows/requirements-agentrx.txt")
	require.NoError(t, err)
	revision := regexp.MustCompile(`fetch --depth 1 origin ([0-9a-f]{40})`).FindStringSubmatch(setup)
	require.Len(t, revision, 2, "AgentRx source must remain pinned to an immutable commit")
	require.Contains(t, string(lock), "# AgentRx "+revision[1]+" runtime dependencies.")

	entry := regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*==[0-9][a-zA-Z0-9.]* \\\n    --hash=sha256:[0-9a-f]{64}$`)
	var requirement strings.Builder
	count := 0
	for line := range strings.SplitSeq(string(lock), "\n") {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if requirement.Len() > 0 {
			requirement.WriteByte('\n')
		}
		requirement.WriteString(line)
		if strings.HasSuffix(line, `\`) {
			continue
		}
		require.Regexp(t, entry, requirement.String(), "every dependency must have an exact version and a SHA-256 wheel hash")
		requirement.Reset()
		count++
	}
	require.Zero(t, requirement.Len(), "dependency entries must not be truncated")
	require.Positive(t, count)
}
