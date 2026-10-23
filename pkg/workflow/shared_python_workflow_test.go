//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type sharedPythonStep struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

func TestSharedPythonDependencyLocks(t *testing.T) {
	locks := make(map[string]map[string]string)
	for _, name := range []string{"scientific", "nlp"} {
		lock, err := os.ReadFile("../../.github/workflows/requirements-python-" + name + ".txt")
		require.NoError(t, err)
		entries := make(map[string]string)
		var entry strings.Builder
		for line := range strings.SplitSeq(string(lock), "\n") {
			if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
				continue
			}
			if entry.Len() > 0 {
				entry.WriteByte('\n')
			}
			entry.WriteString(line)
			if strings.HasSuffix(line, `\`) {
				continue
			}
			require.Regexp(t, hashedPythonRequirement, entry.String())
			pkg, _, ok := strings.Cut(entry.String(), "==")
			require.True(t, ok)
			require.NotContains(t, entries, pkg, "dependencies must not be duplicated")
			entries[pkg] = entry.String()
			entry.Reset()
		}
		require.Zero(t, entry.Len(), "dependency entries must not be truncated")
		input, err := os.ReadFile("../../.github/workflows/requirements-python-" + name + ".in")
		require.NoError(t, err)
		directCount := 0
		for requirement := range strings.SplitSeq(strings.TrimSpace(string(input)), "\n") {
			pkg, _, ok := strings.Cut(requirement, "==")
			require.True(t, ok)
			require.Contains(t, entries[pkg], requirement+" \\\n")
			directCount++
		}
		require.Greater(t, len(entries), directCount, "locks must include transitive dependencies")
		locks[name] = entries
	}
	commonCount := 0
	for pkg, entry := range locks["scientific"] {
		if nlpEntry, ok := locks["nlp"][pkg]; ok {
			require.Equal(t, entry, nlpEntry, "shared dependencies must have identical versions and hashes in either install order: %s", pkg)
			commonCount++
		}
	}
	require.Positive(t, commonCount)
}

func TestSharedPythonGeneratedSetups(t *testing.T) {
	expected := make(map[string]string)
	for _, tt := range []struct {
		source string
		name   string
		lock   string
	}{
		{"python-dataviz", "Install Python scientific libraries", "scientific"},
		{"trending-charts-simple", "Setup Python environment", "scientific"},
		{"python-nlp", "Setup Python NLP environment", "nlp"},
	} {
		source, err := os.ReadFile("../../.github/workflows/shared/" + tt.source + ".md")
		require.NoError(t, err)
		parts := strings.SplitN(string(source), "---", 3)
		require.Len(t, parts, 3)
		var frontmatter struct {
			Steps []sharedPythonStep `yaml:"steps"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &frontmatter))
		var setup string
		for _, step := range frontmatter.Steps {
			if step.Name == tt.name {
				setup = strings.TrimSpace(step.Run)
			}
		}
		require.NotEmpty(t, setup)
		require.Contains(t, setup, `uv venv --python 3.12 --python-preference only-managed /tmp/gh-aw/python/venv`)
		require.Contains(t, setup, `uv pip install --quiet --require-hashes --only-binary=:all: --python /tmp/gh-aw/python/venv/bin/python -r "${GITHUB_WORKSPACE}/.github/workflows/requirements-python-`+tt.lock+`.txt"`)
		require.NotContains(t, setup, "--seed", "do not bootstrap unlocked pip or setuptools")
		require.NotContains(t, setup, "uv pip sync", "must preserve packages from sibling imports")
		require.Contains(t, setup, "if [ ! -d /tmp/gh-aw/python/venv ]; then")
		expected[tt.name] = setup
	}

	paths, err := filepath.Glob("../../.github/workflows/*.lock.yml")
	require.NoError(t, err)
	seen := make(map[string]int)
	for _, path := range paths {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		if !strings.Contains(string(content), "/tmp/gh-aw/python/venv") {
			continue
		}
		var workflow struct {
			Jobs map[string]struct {
				Steps []sharedPythonStep `yaml:"steps"`
			} `yaml:"jobs"`
		}
		require.NoError(t, yaml.Unmarshal(content, &workflow), path)
		for _, job := range workflow.Jobs {
			for _, step := range job.Steps {
				setup, ok := expected[step.Name]
				if !ok || !strings.Contains(step.Run, "/tmp/gh-aw/python/venv") {
					continue
				}
				require.Equal(t, setup, strings.TrimSpace(step.Run), "%s: %s must preserve the hash-enforced source setup", path, step.Name)
				seen[step.Name]++
			}
		}
	}
	for name := range expected {
		require.Positive(t, seen[name], "the generated workflows must exercise %s", name)
	}
}
