//go:build !integration

package workflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

type workflowFrontmatter struct {
	Steps []struct {
		Name string `yaml:"name"`
		Run  string `yaml:"run"`
	} `yaml:"steps"`
}

func readCopilotCentralizationOptimizerSource(t *testing.T) string {
	t.Helper()

	sourceContent, err := os.ReadFile("../../.github/workflows/copilot-centralization-optimizer.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}
	return string(sourceContent)
}

func assertContainsAll(t *testing.T, text string, fragments []string) {
	t.Helper()

	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected content to contain %q", fragment)
		}
	}
}

func extractCollectAgentTaskDataRunScript(t *testing.T, sourceText string) string {
	t.Helper()

	parts := strings.SplitN(sourceText, "\n---\n", 2)
	if len(parts) != 2 {
		t.Fatal("expected workflow source to contain frontmatter separator")
	}

	var frontmatter workflowFrontmatter
	if err := yaml.Unmarshal([]byte(parts[0]), &frontmatter); err != nil {
		t.Fatalf("failed to parse workflow frontmatter: %v", err)
	}

	for _, step := range frontmatter.Steps {
		if step.Name == "Collect agent task data" {
			return step.Run
		}
	}

	t.Fatal("failed to find Collect agent task data step")
	return ""
}

func TestCopilotCentralizationOptimizerWorkflowHandlesTaskFetchFailures(t *testing.T) {
	sourceText := readCopilotCentralizationOptimizerSource(t)
	assertContainsAll(t, sourceText, []string{
		`_task_ids=$(mktemp)`,
		`if gh api --paginate \`,
		`done < "$_task_ids" \`,
		`echo "Agent tasks API request failed; proceeding with empty dataset" >&2`,
		`jq -s '.' /tmp/gh-aw/data/task-summaries.jsonl > /tmp/gh-aw/data/task-summaries.json`,
	})
}

func TestCopilotCentralizationOptimizerCompiledWorkflowHandlesTaskFetchFailures(t *testing.T) {
	lockContent, err := os.ReadFile("../../.github/workflows/copilot-centralization-optimizer.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockText := string(lockContent)
	assertContainsAll(t, lockText, []string{
		`_task_ids=$(mktemp)`,
		`if gh api --paginate \\\n`,
		`done < \"$_task_ids\" \\\n`,
		`echo \"Agent tasks API request failed; proceeding with empty dataset\" >&2`,
		`jq -s '.' /tmp/gh-aw/data/task-summaries.jsonl > /tmp/gh-aw/data/task-summaries.json`,
	})
}

func TestCopilotCentralizationOptimizerTaskFetchFailureStillWritesJSON(t *testing.T) {
	sourceText := readCopilotCentralizationOptimizerSource(t)
	script := extractCollectAgentTaskDataRunScript(t, sourceText)

	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	script = strings.ReplaceAll(script, "/tmp/gh-aw/data", dataDir)

	fakeBinDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("failed to create fake bin dir: %v", err)
	}

	fakeGHPath := filepath.Join(fakeBinDir, "gh")
	if err := os.WriteFile(fakeGHPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("failed to write fake gh: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"REPO=github/gh-aw",
		"TASK_LOOKBACK_DAYS=30",
		"TASK_LIMIT=300",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected script to tolerate gh failures, got %v:\n%s", err, output)
	}

	taskSummariesJSON, err := os.ReadFile(filepath.Join(dataDir, "task-summaries.json"))
	if err != nil {
		t.Fatalf("expected task-summaries.json to be written: %v", err)
	}
	if strings.TrimSpace(string(taskSummariesJSON)) != "[]" {
		t.Fatalf("expected task-summaries.json to contain an empty JSON array, got %q", taskSummariesJSON)
	}
}
