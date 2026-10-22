//go:build !integration && !windows

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func aiderDiscoveryCommand(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile("../../.github/workflows/" + name + ".md")
	require.NoError(t, err)
	_, body, found := strings.Cut(string(content), "```bash\n")
	require.True(t, found)
	command, _, found := strings.Cut(body, "\n```")
	require.True(t, found)
	return command
}

func TestAiderDiscoveryUsesGoPackagesAndActualNames(t *testing.T) {
	dir := t.TempDir()
	dir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	for name, content := range map[string]string{
		"go.mod":                    "module example.test/aider\n\ngo 1.20\n",
		"cmd/hyphenated/main.go":    "package main\n",
		"untested/source.go":        "package actualname\n",
		"tested/source.go":          "package tested\n",
		"tested/source_test.go":     "package tested\n",
		"external/source.go":        "package external\n",
		"external/source_test.go":   "package external_test\n",
		"testdata/fixture/input.go": "not a Go package\n",
		".hidden/source.go":         "not a Go package\n",
		"nested/go.mod":             "module example.test/nested\n\ngo 1.20\n",
		"nested/source.go":          "not a Go package\n",
	} {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	command := aiderDiscoveryCommand(t, "daily-go-test-stubs-aider")
	// Keep the workflow's scratch file confined to this fixture.
	command = strings.ReplaceAll(command, "/tmp/gh-aw/agent/aider-untested-packages", filepath.Join(dir, "packages"))
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	assert.Equal(t, filepath.Join(dir, "cmd/hyphenated")+" main\n"+filepath.Join(dir, "untested")+" actualname\n", string(output))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "untested/broken.go"), []byte("not a Go package\n"), 0o600))
	cmd = exec.Command("bash", "-c", command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	output, err = cmd.CombinedOutput()
	require.Error(t, err, "failed discovery must not look like an empty package list: %s", output)
}

func TestAiderDebtDiscoveryOnlySelectsSourceMarkers(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"source.go":         "// TODO: implement this\n\t// FIXME handle error\nfunc printConfigTODO() {}\n// printConfigTODO prints a checklist\nvar text = \"TODO: customize\"\n",
		"source_test.go":    "// TODO: test fixture\n",
		"testdata/input.go": "// FIXME: fixture\n",
		"vendor/example.go": "// TODO: vendored\n",
	} {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	cmd := exec.Command("bash", "-c", aiderDiscoveryCommand(t, "daily-code-debt-aider"))
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	assert.Equal(t, "./source.go:1:// TODO: implement this\n./source.go:2:\t// FIXME handle error\n", string(output))
}

func TestAiderDailyGoValidationContracts(t *testing.T) {
	for _, name := range []string{"daily-code-debt-aider", "daily-go-test-stubs-aider"} {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile("../../.github/workflows/" + name + ".md")
			require.NoError(t, err)
			var frontmatter struct {
				Runtimes map[string]struct {
					Version string `yaml:"version"`
				} `yaml:"runtimes"`
			}
			parts := strings.SplitN(string(content), "---", 3)
			require.Len(t, parts, 3)
			require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &frontmatter))
			assert.Equal(t, "1.27.3", frontmatter.Runtimes["go"].Version)
			assert.Contains(t, parts[2], "make fmt-go && go ")
			assert.Contains(t, parts[2], "|| safeoutputs report_incomplete")
			assert.NotContains(t, parts[2], "|| safeoutputs noop")
			assert.NotContains(t, parts[2], "|| true")
			assert.NotContains(t, parts[2], "GOMODCACHE=/tmp/")
			if name == "daily-go-test-stubs-aider" {
				assert.Contains(t, parts[2], "go test ./... -run '^TestPlaceholder_TODO$'")
			}
		})
	}
}
