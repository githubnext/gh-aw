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

func TestArcDindSmokeEvidenceSanitizesUploadedDestination(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "smoke.md")
	require.NoError(t, os.WriteFile(file, []byte(`---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
runner:
  topology: arc-dind
post-steps:
  - name: Evidence assertion
    run: echo smoke_evidence_diagnostics.cjs
---
Smoke evidence contract fixture.
`), 0o600))
	require.NoError(t, NewCompiler().CompileWorkflow(file))
	content, err := os.ReadFile(filepath.Join(dir, "smoke.lock.yml"))
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string
				ID   string `yaml:"id"`
				If   string
				Run  string
				With map[string]any
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(content, &workflow))
	var consolidate, sanitize string
	consolidateIndex, sanitizeIndex, uploadIndex := -1, -1, -1
	for i, step := range workflow.Jobs["agent"].Steps {
		switch step.Name {
		case "Consolidate artifacts for ARC/DinD":
			consolidateIndex, consolidate = i, step.Run
		case "Sanitize smoke evidence diagnostics":
			sanitizeIndex, sanitize = i, step.Run
			assert.Equal(t, "smoke_evidence_diagnostics", step.ID)
			assert.Equal(t, "always()", step.If)
		}
		if paths, ok := step.With["path"].(string); ok && strings.Contains(paths, "smoke-evidence.json") {
			uploadIndex = i
			assert.Contains(t, paths, "steps.smoke_evidence_diagnostics.outputs.ready == 'true'")
			assert.Contains(t, paths, "format('{0}/gh-aw/smoke-evidence.json', runner.temp)")
		}
	}
	require.GreaterOrEqual(t, consolidateIndex, 0)
	require.Greater(t, sanitizeIndex, consolidateIndex)
	require.Greater(t, uploadIndex, sanitizeIndex)
	require.Contains(t, sanitize, `--sanitize "$RUNNER_TEMP/gh-aw"`)
	require.Contains(t, consolidate, "cp -a --no-clobber")

	helper, err := filepath.Abs("../../actions/setup/js/smoke_evidence_diagnostics.cjs")
	require.NoError(t, err)
	const valid = `{"kind":"session","failures":["No non-empty agent-stdio.log found in the agent logs"],"files":["stdio"]}`
	for _, test := range []struct {
		name, destination string
		symlink, ready    bool
	}{
		{name: "new destination", ready: true},
		{name: "pre-existing valid destination", destination: valid, ready: true},
		{name: "pre-existing injected extra fields", destination: strings.TrimSuffix(valid, "}") + `,"secret":"untrusted-secret"}`, ready: true},
		{name: "pre-existing invalid schema", destination: `{"kind":"session","failures":["untrusted-secret"],"files":["stdio"]}`},
		{name: "pre-existing malformed JSON", destination: "untrusted-secret"},
		{name: "pre-existing oversized file", destination: strings.Repeat("x", 16385)},
		{name: "pre-existing symlink", destination: valid, symlink: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source")
			runnerTemp := filepath.Join(root, "runner")
			destinationRoot := filepath.Join(runnerTemp, "gh-aw")
			require.NoError(t, os.MkdirAll(source, 0o700))
			require.NoError(t, os.MkdirAll(filepath.Join(destinationRoot, "actions"), 0o700))
			require.NoError(t, os.Symlink(helper, filepath.Join(destinationRoot, "actions", "smoke_evidence_diagnostics.cjs")))
			require.NoError(t, os.WriteFile(filepath.Join(source, "smoke-evidence.json"), []byte(valid), 0o600))
			destination := filepath.Join(destinationRoot, "smoke-evidence.json")
			target := filepath.Join(root, "symlink-target.json")
			if test.destination != "" {
				if test.symlink {
					require.NoError(t, os.WriteFile(target, []byte(test.destination), 0o600))
					require.NoError(t, os.Symlink(target, destination))
				} else {
					require.NoError(t, os.WriteFile(destination, []byte(test.destination), 0o600))
				}
			}
			unrelated := filepath.Join(destinationRoot, "firewall.log")
			require.NoError(t, os.WriteFile(unrelated, []byte("existing firewall log"), 0o600))
			outputFile := filepath.Join(root, "output")
			require.NoError(t, os.WriteFile(outputFile, nil, 0o600))
			// BSD cp uses -n but returns nonzero for skipped files; GNU cp
			// --no-clobber succeeds, so do not take the fallback on a skip.
			consolidation := strings.ReplaceAll(consolidate, "--no-clobber", "-n")
			consolidation = strings.ReplaceAll(consolidation, "2>/dev/null || \\\n", "2>/dev/null; true || \\\n")
			consolidation = strings.ReplaceAll(consolidation, "/tmp/gh-aw", `"${SMOKE_SOURCE}"`)
			env := append(os.Environ(), "RUNNER_TEMP="+runnerTemp, "SMOKE_SOURCE="+source, "GITHUB_OUTPUT="+outputFile)
			cmd := exec.Command("bash", "-c", consolidation)
			cmd.Env = env
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			if test.destination != "" {
				preserved, err := os.ReadFile(destination)
				require.NoError(t, err)
				require.Equal(t, test.destination, string(preserved), "No-clobber consolidation must preserve the pre-existing destination")
			}
			cmd = exec.Command("bash", "-c", sanitize)
			cmd.Env = env
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			published, err := os.ReadFile(outputFile)
			require.NoError(t, err)
			if test.ready {
				assert.Equal(t, "ready=true\n", string(published))
				record, err := os.ReadFile(destination)
				require.NoError(t, err)
				assert.JSONEq(t, valid, string(record))
				assert.NotContains(t, string(record), "untrusted-secret")
			} else {
				assert.Empty(t, published)
				_, err := os.Lstat(destination)
				assert.True(t, os.IsNotExist(err), "Invalid destination must be removed from the upload path")
			}
			sourceBytes, err := os.ReadFile(filepath.Join(source, "smoke-evidence.json"))
			require.NoError(t, err)
			assert.JSONEq(t, valid, string(sourceBytes))
			unrelatedBytes, err := os.ReadFile(unrelated)
			require.NoError(t, err)
			assert.Equal(t, "existing firewall log", string(unrelatedBytes))
			if test.symlink {
				targetBytes, err := os.ReadFile(target)
				require.NoError(t, err)
				assert.JSONEq(t, valid, string(targetBytes))
			}
		})
	}
}
