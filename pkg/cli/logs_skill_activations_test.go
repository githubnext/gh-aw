//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestExtractSkillActivationsFromAgentOutput verifies that explicit skill_invocation
// entries in agent_output.json are detected and returned as SkillActivation records.
func TestExtractSkillActivationsFromAgentOutput(t *testing.T) {
	t.Parallel()
	testRun := WorkflowRun{DatabaseID: 123, WorkflowName: "test-workflow"}

	tests := []struct {
		name            string
		content         string
		wantCount       int
		wantFirstName   string
		wantFirstStatus string
		wantFirstSource string
	}{
		{
			name: "single_skill_invocation",
			content: `{
				"items": [
					{
						"type": "skill_invocation",
						"name": "docs-check-style",
						"status": "invoked",
						"timestamp": "2024-01-01T12:00:00Z"
					}
				]
			}`,
			wantCount:       1,
			wantFirstName:   "docs-check-style",
			wantFirstStatus: "invoked",
			wantFirstSource: "agent_output",
		},
		{
			name: "multiple_skill_invocations",
			content: `{
				"items": [
					{
						"type": "skill_invocation",
						"name": "docs-check-style",
						"timestamp": "2024-01-01T10:00:00Z"
					},
					{
						"type": "skill_invocation",
						"name": "frontmatter-audit",
						"status": "invoked",
						"timestamp": "2024-01-01T10:01:00Z"
					},
					{
						"type": "create-issue",
						"title": "This should be ignored"
					}
				]
			}`,
			wantCount:       2,
			wantFirstName:   "docs-check-style",
			wantFirstStatus: "invoked",
			wantFirstSource: "agent_output",
		},
		{
			name: "no_skill_invocations",
			content: `{
				"items": [
					{
						"type": "missing_tool",
						"tool": "terraform",
						"reason": "Not available"
					}
				]
			}`,
			wantCount: 0,
		},
		{
			name: "skill_invocation_with_empty_name_skipped",
			content: `{
				"items": [
					{
						"type": "skill_invocation",
						"name": "",
						"status": "invoked"
					},
					{
						"type": "skill_invocation",
						"name": "valid-skill",
						"status": "invoked"
					}
				]
			}`,
			wantCount:       1,
			wantFirstName:   "valid-skill",
			wantFirstStatus: "invoked",
			wantFirstSource: "agent_output",
		},
		{
			name: "skill_invocation_without_explicit_status_defaults_to_invoked",
			content: `{
				"items": [
					{
						"type": "skill_invocation",
						"name": "docs-frontmatter-audit"
					}
				]
			}`,
			wantCount:       1,
			wantFirstName:   "docs-frontmatter-audit",
			wantFirstStatus: "invoked",
			wantFirstSource: "agent_output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := testutil.TempDir(t, "skill-activations-*")
			// Write agent_output.json
			agentOutputPath := filepath.Join(tmpDir, constants.AgentOutputFilename.String())
			if err := os.WriteFile(agentOutputPath, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("failed to write agent_output.json: %v", err)
			}

			got, err := extractSkillActivationsFromRun(tmpDir, testRun, false, "", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Errorf("got %d activations, want %d", len(got), tc.wantCount)
			}
			if tc.wantCount > 0 && len(got) > 0 {
				first := got[0]
				if first.Name != tc.wantFirstName {
					t.Errorf("first activation name = %q, want %q", first.Name, tc.wantFirstName)
				}
				if first.Status != tc.wantFirstStatus {
					t.Errorf("first activation status = %q, want %q", first.Status, tc.wantFirstStatus)
				}
				if first.Source != tc.wantFirstSource {
					t.Errorf("first activation source = %q, want %q", first.Source, tc.wantFirstSource)
				}
			}
		})
	}
}

// TestExtractSkillActivationsFromLogFiles verifies that skill invocation patterns
// in raw agent log files are detected when no agent_output.json is present.
func TestExtractSkillActivationsFromLogFiles(t *testing.T) {
	t.Parallel()
	testRun := WorkflowRun{DatabaseID: 456, WorkflowName: "log-parse-test"}

	tests := []struct {
		name       string
		logContent string
		wantNames  []string
		wantSource string
	}{
		{
			name: "copilot_skill_invocation_form",
			logContent: `Starting task...
skill(docs-check-style)
Continuing...`,
			wantNames:  []string{"docs-check-style"},
			wantSource: "log_parse",
		},
		{
			name: "structured_log_invoked_form",
			logContent: `[skills] invoked: frontmatter-audit
Running checks...`,
			wantNames:  []string{"frontmatter-audit"},
			wantSource: "log_parse",
		},
		{
			name: "skill_invoked_label_form",
			logContent: `Skill invoked: content-type-checker
Done.`,
			wantNames:  []string{"content-type-checker"},
			wantSource: "log_parse",
		},
		{
			name: "multiple_skills_deduplicated",
			logContent: `skill(docs-check-style)
skill(docs-check-style)
skill(frontmatter-audit)`,
			wantNames:  []string{"docs-check-style", "frontmatter-audit"},
			wantSource: "log_parse",
		},
		{
			name: "no_skill_patterns",
			logContent: `Running agent...
No skills here.`,
			wantNames: nil,
		},
		{
			name:       "case_insensitive_matching",
			logContent: `SKILL(MySkill)`,
			wantNames:  []string{"MySkill"},
			wantSource: "log_parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := testutil.TempDir(t, "skill-log-parse-*")

			logPath := filepath.Join(tmpDir, "agent.log")
			if err := os.WriteFile(logPath, []byte(tc.logContent), 0o600); err != nil {
				t.Fatalf("failed to write log file: %v", err)
			}

			got, err := extractSkillActivationsFromRun(tmpDir, testRun, false, "", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tc.wantNames) {
				names := make([]string, len(got))
				for i, a := range got {
					names[i] = a.Name
				}
				t.Errorf("got %d activations %v, want %d %v", len(got), names, len(tc.wantNames), tc.wantNames)
				return
			}

			gotNames := make(map[string]struct{})
			for _, a := range got {
				gotNames[a.Name] = struct{}{}
				if tc.wantSource != "" && a.Source != tc.wantSource {
					t.Errorf("activation %q: source = %q, want %q", a.Name, a.Source, tc.wantSource)
				}
				if a.Status != "invoked" {
					t.Errorf("activation %q: status = %q, want %q", a.Name, a.Status, "invoked")
				}
			}
			for _, wantName := range tc.wantNames {
				if _, ok := gotNames[wantName]; !ok {
					t.Errorf("expected skill %q not found in activations", wantName)
				}
			}
		})
	}
}

// TestExtractSkillActivationsBothSourcesMerged verifies that skills from both
// agent_output.json and log files are returned (applies to all skills).
func TestExtractSkillActivationsBothSourcesMerged(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "skill-merge-*")
	testRun := WorkflowRun{DatabaseID: 789, WorkflowName: "merge-test"}

	// Write agent_output.json with one explicit entry.
	agentOutputContent := `{
		"items": [
			{
				"type": "skill_invocation",
				"name": "from-agent-output",
				"status": "invoked"
			}
		]
	}`
	agentOutputPath := filepath.Join(tmpDir, constants.AgentOutputFilename.String())
	if err := os.WriteFile(agentOutputPath, []byte(agentOutputContent), 0o600); err != nil {
		t.Fatalf("failed to write agent_output.json: %v", err)
	}

	// Also write a log file that has a different skill not present in agent_output.
	logContent := `skill(from-log-file)`
	logPath := filepath.Join(tmpDir, "agent.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0o600); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	got, err := extractSkillActivationsFromRun(tmpDir, testRun, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both skills should be present since they have different names.
	if len(got) != 2 {
		names := make([]string, len(got))
		for i, a := range got {
			names[i] = a.Name
		}
		t.Fatalf("got %d activations %v, want 2 (both sources merged)", len(got), names)
	}

	byName := make(map[string]SkillActivation, len(got))
	for _, a := range got {
		byName[a.Name] = a
	}

	if act, ok := byName["from-agent-output"]; !ok {
		t.Error("expected skill from-agent-output not found")
	} else if act.Source != "agent_output" {
		t.Errorf("from-agent-output source = %q, want %q", act.Source, "agent_output")
	}

	if act, ok := byName["from-log-file"]; !ok {
		t.Error("expected skill from-log-file not found")
	} else if act.Source != "log_parse" {
		t.Errorf("from-log-file source = %q, want %q", act.Source, "log_parse")
	}
}

// TestExtractSkillActivationsAgentOutputWinsOnDuplicate verifies that when the
// same skill name appears in both agent_output.json and log files, the
// agent_output version is kept (and the log entry is silently dropped).
func TestExtractSkillActivationsAgentOutputWinsOnDuplicate(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "skill-dedup-*")
	testRun := WorkflowRun{DatabaseID: 790, WorkflowName: "dedup-test"}

	agentOutputContent := `{
		"items": [
			{
				"type": "skill_invocation",
				"name": "shared-skill",
				"status": "invoked"
			}
		]
	}`
	agentOutputPath := filepath.Join(tmpDir, constants.AgentOutputFilename.String())
	if err := os.WriteFile(agentOutputPath, []byte(agentOutputContent), 0o600); err != nil {
		t.Fatalf("failed to write agent_output.json: %v", err)
	}

	// Log file mentions the same skill.
	logContent := `skill(shared-skill)`
	logPath := filepath.Join(tmpDir, "agent.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0o600); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	got, err := extractSkillActivationsFromRun(tmpDir, testRun, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d activations, want 1 (duplicate deduplicated)", len(got))
	}
	if got[0].Name != "shared-skill" {
		t.Errorf("got skill name %q, want %q", got[0].Name, "shared-skill")
	}
	if got[0].Source != "agent_output" {
		t.Errorf("got source %q, want %q (agent_output should win on duplicate)", got[0].Source, "agent_output")
	}
}

// TestExtractSkillActivationsProvenanceFields verifies that the ReportProvenance
// fields are populated correctly on each returned record.
func TestExtractSkillActivationsProvenanceFields(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "skill-provenance-*")
	testRun := WorkflowRun{
		DatabaseID:   999,
		WorkflowName: "provenance-workflow",
	}

	content := `{
		"items": [
			{
				"type": "skill_invocation",
				"name": "some-skill",
				"status": "invoked",
				"timestamp": "2024-06-01T09:00:00Z"
			}
		]
	}`
	agentOutputPath := filepath.Join(tmpDir, constants.AgentOutputFilename.String())
	if err := os.WriteFile(agentOutputPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write agent_output.json: %v", err)
	}

	got, err := extractSkillActivationsFromRun(tmpDir, testRun, false, "exp1", "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d activations, want 1", len(got))
	}

	act := got[0]
	if act.RunID != testRun.DatabaseID {
		t.Errorf("RunID = %d, want %d", act.RunID, testRun.DatabaseID)
	}
	if act.WorkflowName != testRun.WorkflowName {
		t.Errorf("WorkflowName = %q, want %q", act.WorkflowName, testRun.WorkflowName)
	}
	if act.ExperimentName != "exp1" {
		t.Errorf("ExperimentName = %q, want %q", act.ExperimentName, "exp1")
	}
	if act.Variant != "v1" {
		t.Errorf("Variant = %q, want %q", act.Variant, "v1")
	}
	if act.Timestamp != "2024-06-01T09:00:00Z" {
		t.Errorf("Timestamp = %q, want %q", act.Timestamp, "2024-06-01T09:00:00Z")
	}
}
