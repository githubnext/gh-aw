//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestContributionCheckWorkflowSafeOutputContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read contribution-check workflow")

	text := string(content)
	assert.Contains(t, text, "emit exactly", "Workflow must explicitly limit noop emission")
	assert.Contains(t, text, "one consolidated noop", "Workflow must require a single consolidated noop")

	assert.Contains(t, text, "temporary_id", "Workflow must mention temporary_id for summary issue linkage")
	assert.Contains(t, text, "aw_summary", "Workflow should provide a concrete temporary_id example")
	assert.Contains(t, text, "add_labels.item_number", "Workflow must mention add_labels item_number linkage")
	assert.Contains(t, text, "#<temporary_id>", "Workflow must describe item_number temporary_id reference format")
	assert.Contains(t, text, "Never emit `add_comment` without a numeric target field", "Workflow must forbid targetless add_comment items")
	assert.Contains(t, text, "\"issue_number\":35304", "Workflow should include a concrete add_comment issue_number example")
	assert.Contains(t, text, "model: claude-haiku-4.5", "Workflow should require small model for contribution-checker subagent calls")
}

func TestContributionCheckWorkflowAllowsRequiredShellCommands(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read contribution-check workflow")

	text := string(content)
	assert.Contains(t, text, `"git"`, "Existing shell permissions must not be widened to fix evidence collection")
	assert.Contains(t, text, `"jq *"`, "Workflow must allow jq payload construction for safeoutputs create_issue")

	lockPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Should read compiled contribution-check workflow")

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Env map[string]string
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(lockContent, &workflow))
	for _, step := range workflow.Jobs["agent"].Steps {
		if config := step.Env["GH_AW_COPILOT_SDK_TOOL_CONFIG"]; config != "" {
			var tools struct {
				Permissions struct {
					AllowedTools []string `json:"allowedTools"`
				} `json:"permissions"`
			}
			require.NoError(t, json.Unmarshal([]byte(config), &tools))
			for _, tool := range []string{"shell(git:*)", "shell(jq)", "shell(safeoutputs:*)"} {
				assert.Contains(t, tools.Permissions.AllowedTools, tool)
			}
			for _, tool := range []string{"shell", "shell(curl)", "shell(curl:*)", "shell(cd)", "shell(command:*)"} {
				assert.NotContains(t, tools.Permissions.AllowedTools, tool)
			}
			assert.Equal(t, "5", step.Env["GH_AW_MAX_TOOL_DENIALS"])
			return
		}
	}
	t.Fatal("Agent job must configure Copilot SDK tool permissions")
}

func TestContributionCheckWorkflowLocalEvidenceContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md"))
	require.NoError(t, err)
	text := string(content)
	for _, required := range []string{
		"pr-evidence/<number>-metadata.json",
		"pr-evidence/<number>-files.json",
		"do not dispatch or retry that PR",
		"Do not fetch PR data or referenced issues again",
		"Never post contributor comments for `triage:error` rows",
		"Missing evidence, tool denials, and `triage:error` results are terminal",
		"`quality` does NOT start with `\"triage\"`",
	} {
		assert.Contains(t, text, required)
	}
	assert.NotContains(t, text, "It fetches its own PR data")
	assert.NotContains(t, text, "read-only access via GitHub MCP tools")

	agent, err := os.ReadFile(filepath.Join(repoRoot, ".github", "agents", "contribution-checker.agent.md"))
	require.NoError(t, err)
	for _, required := range []string{
		"read only those files",
		"author_association",
		"On a tool denial, missing/invalid evidence, or a failed supported fetch, stop collecting data",
		"Use `null` for unknown checklist values and line counts",
		"An unavailable `patch` is not an empty diff",
		"leave `comment` empty",
	} {
		assert.Contains(t, string(agent), required)
	}
}

func TestContributionCheckPrefetchesPREvidence(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required to exercise the workflow pre-agent step")
	}
	script := contributionCheckEvidenceScript(t)
	for _, tc := range []struct {
		name       string
		wantError  string
		firstFiles int
	}{
		{name: "paginated-files", firstFiles: 2},
		{name: "empty-diff"},
		{name: "metadata-failure", wantError: "PR metadata could not be collected or validated."},
		{name: "invalid-metadata", wantError: "PR metadata could not be collected or validated."},
		{name: "partial-files-failure", wantError: "PR files and patches could not be collected completely."},
		{name: "invalid-files", wantError: "PR files and patches could not be collected completely."},
		{name: "incomplete-files", wantError: "PR files and patches could not be collected completely."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workspace := t.TempDir()
			bin := filepath.Join(workspace, "bin")
			require.NoError(t, os.Mkdir(bin, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte(contributionCheckGHFixture), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(workspace, "pr-filter-results.json"), []byte(`{"pr_numbers":[71100,71108]}`), 0o600))
			require.NoError(t, os.Mkdir(filepath.Join(workspace, "pr-evidence"), 0o755))
			errorPath := filepath.Join(workspace, "pr-evidence", "71100-error.txt")
			require.NoError(t, os.WriteFile(errorPath, []byte("stale error"), 0o600))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", script)
			cmd.Dir = workspace
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GITHUB_WORKSPACE="+workspace, "TARGET_REPOSITORY=github/gh-aw", "EVIDENCE_CASE="+tc.name)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)

			if tc.wantError != "" {
				message, err := os.ReadFile(errorPath)
				require.NoError(t, err)
				assert.Equal(t, tc.wantError+"\n", string(message))
				assert.Contains(t, string(output), "::warning::PR 71100")
			} else {
				_, err := os.Stat(errorPath)
				assert.True(t, os.IsNotExist(err), "successful collection must clear stale failures")
				files := contributionCheckReadFiles(t, workspace, "71100")
				assert.Len(t, files, tc.firstFiles)
				if tc.firstFiles > 0 {
					assert.Equal(t, "first.go", files[0]["filename"])
					assert.Equal(t, "second_test.go", files[1]["filename"])
					assert.Equal(t, "@@ first patch @@", files[0]["patch"])
				}
			}
			assert.Len(t, contributionCheckReadFiles(t, workspace, "71108"), 2,
				"one PR's failure must not prevent collection for the next PR")
			_, err = os.Stat(filepath.Join(workspace, "should-not-exist"))
			assert.True(t, os.IsNotExist(err), "PR content must not execute as shell instructions")
		})
	}
}

func contributionCheckEvidenceScript(t *testing.T) string {
	t.Helper()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md"))
	require.NoError(t, err)
	parts := strings.SplitN(string(content), "---\n", 3)
	require.Len(t, parts, 3)
	var workflow struct {
		Steps []struct{ Name, Run string }
	}
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &workflow))
	for _, step := range workflow.Steps {
		if step.Name == "Fetch PR evaluation data" {
			return step.Run
		}
	}
	t.Fatal("Workflow must prefetch PR evidence before parallel evaluation")
	return ""
}

func TestContributionCheckPrefetchValidatesPRNumbers(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required to exercise the workflow pre-agent step")
	}
	script := contributionCheckEvidenceScript(t)
	for _, tc := range []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "no-prs", input: `{"pr_numbers":[]}`},
		{name: "malformed-json", input: `{`, wantErr: true},
		{name: "missing-list", input: `{}`, wantErr: true},
		{name: "non-numeric-pr", input: `{"pr_numbers":["../../outside"]}`, wantErr: true},
		{name: "fractional-pr", input: `{"pr_numbers":[1.5]}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workspace := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(workspace, "pr-filter-results.json"), []byte(tc.input), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(workspace, "gh"), []byte("#!/bin/sh\necho 'Unexpected API call' >&2\nexit 97\n"), 0o755))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-c", script)
			cmd.Dir = workspace
			cmd.Env = append(os.Environ(), "PATH="+workspace+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GITHUB_WORKSPACE="+workspace, "TARGET_REPOSITORY=github/gh-aw")
			output, err := cmd.CombinedOutput()
			if tc.wantErr {
				require.Error(t, err, "%s", output)
			} else {
				require.NoError(t, err, "%s", output)
			}
			assert.NotContains(t, string(output), "Unexpected API call")
			files, err := os.ReadDir(filepath.Join(workspace, "pr-evidence"))
			require.NoError(t, err)
			assert.Empty(t, files)
		})
	}
}

func contributionCheckReadFiles(t *testing.T, workspace, number string) []map[string]any {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(workspace, "pr-evidence", number+"-files.json"))
	require.NoError(t, err)
	var files []map[string]any
	require.NoError(t, json.Unmarshal(content, &files))
	return files
}

const contributionCheckGHFixture = `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" != api ]]; then
  echo "Only REST JSON reads are permitted in this fixture" >&2
  exit 97
fi
case "$2" in
  repos/github/gh-aw/pulls/71100|repos/github/gh-aw/pulls/71108)
    number="${2##*/}"
    count=2
    if [[ "$number" == 71100 ]]; then
      case "$EVIDENCE_CASE" in
        metadata-failure) echo "fixture API failure" >&2; exit 1 ;;
        invalid-metadata) printf '{"message":"unexpected response"}'; exit 0 ;;
        empty-diff) count=0 ;;
      esac
    fi
    printf '{"number":%s,"user":{"login":"alice"},"changed_files":%s,"body":"$(touch should-not-exist)"}\n' "$number" "$count"
    ;;
  repos/github/gh-aw/pulls/71100/files?per_page=100|repos/github/gh-aw/pulls/71108/files?per_page=100)
    [[ "$3" == --paginate && "$4" == --jq && "$5" == '.[]' ]] || exit 97
    if [[ "$2" == */71100/* ]]; then
      case "$EVIDENCE_CASE" in
        empty-diff) exit 0 ;;
        invalid-files) printf '"metadata is not a file"\n'; exit 0 ;;
      esac
    fi
    printf '{"filename":"first.go","additions":1,"deletions":0,"patch":"@@ first patch @@"}\n'
    if [[ "$2" == */71100/* ]]; then
      case "$EVIDENCE_CASE" in
        partial-files-failure) echo "fixture pagination failure" >&2; exit 1 ;;
        incomplete-files) exit 0 ;;
      esac
    fi
    printf '{"filename":"second_test.go","additions":2,"deletions":1,"patch":"@@ second patch @@"}\n'
    ;;
  *) echo "Unexpected API endpoint" >&2; exit 97 ;;
esac
`
