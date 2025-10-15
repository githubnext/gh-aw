package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreActivationJob tests that pre-activation job is created correctly
func TestPreActivationJob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pre-activation-job-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	t.Run("pre_activation_job_created_with_stop_after", func(t *testing.T) {
		workflowContent := `---
on:
  workflow_dispatch:
  stop-after: "+48h"
engine: claude
---

# Stop-Time Workflow

This workflow has a stop-after configuration.
`
		workflowFile := filepath.Join(tmpDir, "stop-time-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify pre_activation job exists
		if !strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected pre_activation job to be created")
		}

		// Verify pre_activation job has actions: write permission
		if !strings.Contains(lockContentStr, "actions: write  # Required for gh workflow disable") {
			t.Error("Expected pre_activation job to have actions: write permission")
		}

		// Verify stop-time check is in pre_activation job
		preActivationStart := strings.Index(lockContentStr, "pre_activation:")
		agentStart := strings.Index(lockContentStr, "agent:")
		stopTimeCheckPos := strings.Index(lockContentStr, "Check stop-time limit")

		if stopTimeCheckPos == -1 {
			t.Error("Expected stop-time check to be present")
		}

		// Stop-time check should be in pre_activation job (before agent job)
		if stopTimeCheckPos > agentStart {
			t.Error("Stop-time check should be in pre_activation job, not in agent job")
		}

		// Stop-time check should be after pre_activation job start
		if stopTimeCheckPos < preActivationStart {
			t.Error("Stop-time check should be in pre_activation job")
		}

		// Verify stop_time_expired output exists
		if !strings.Contains(lockContentStr, "stop_time_expired:") {
			t.Error("Expected stop_time_expired output in pre_activation job")
		}
	})

	t.Run("no_pre_activation_job_without_stop_after_or_unsafe_events", func(t *testing.T) {
		workflowContent := `---
on:
  schedule:
    - cron: "0 9 * * 1"
engine: claude
---

# Normal Workflow

This workflow has no stop-after configuration and only safe events.
`
		workflowFile := filepath.Join(tmpDir, "normal-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify pre_activation job does not exist (schedule is a safe event)
		if strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected NO pre_activation job without stop-after or unsafe events")
		}
	})

	t.Run("pre_activation_with_stop_after_and_activation_checks", func(t *testing.T) {
		workflowContent := `---
on:
  issues:
    types: [opened]
  stop-after: "+24h"
engine: claude
---

# Workflow with Activation

This workflow has activation job and stop-after.
`
		workflowFile := filepath.Join(tmpDir, "activation-workflow.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Compilation failed: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify pre_activation job exists
		if !strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected pre_activation job")
		}

		// Verify activation job exists
		if !strings.Contains(lockContentStr, "activation:") {
			t.Error("Expected activation job for unsafe events")
		}

		// Verify the job dependency chain
		// Extract the jobs section to analyze dependencies
		jobsSection := lockContentStr[strings.Index(lockContentStr, "jobs:"):]

		// Find activation job section (not pre_activation)
		activationIdx := strings.Index(jobsSection, "\n  activation:")
		if activationIdx == -1 {
			// Try without leading newline (in case it's the first job)
			activationIdx = strings.Index(jobsSection, "activation:")
			if activationIdx == 0 {
				activationIdx = 0
			} else {
				t.Error("Expected activation job")
				return
			}
		} else {
			activationIdx++ // Skip the newline
		}

		// Extract activation job section (find next job at same indent level)
		activationSection := jobsSection[activationIdx:]
		// Look for the next job (starts with "  <name>:" at the beginning of a line after the first line)
		lines := strings.Split(activationSection, "\n")
		endIdx := len(activationSection)
		for i := 1; i < len(lines); i++ {
			// Check if this line starts a new job (two spaces followed by non-space, ending with colon)
			if len(lines[i]) > 2 && lines[i][0:2] == "  " && lines[i][2] != ' ' && strings.Contains(lines[i], ":") {
				// This is the start of a new job, calculate position
				endIdx = 0
				for j := 0; j < i; j++ {
					endIdx += len(lines[j]) + 1 // +1 for newline
				}
				break
			}
		}
		activationSection = activationSection[:endIdx]

		// Verify activation depends on pre_activation
		if !strings.Contains(activationSection, "needs: pre_activation") {
			t.Errorf("Expected activation job to depend on pre_activation job, got:\n%s", activationSection)
		}

		// Verify activation checks stop_time_expired in its if condition
		if !strings.Contains(lockContentStr, "stop_time_expired") {
			t.Errorf("Expected activation job to check stop_time_expired in if condition")
		}

		// Verify agent job exists and depends on activation
		if !strings.Contains(lockContentStr, "agent:") {
			t.Error("Expected agent job")
		}

		agentIdx := strings.Index(jobsSection, "agent:")
		if agentIdx == -1 {
			t.Error("Expected agent job")
		}

		agentSection := jobsSection[agentIdx:]
		// Extract agent section the same way
		agentLines := strings.Split(agentSection, "\n")
		agentEndIdx := len(agentSection)
		for i := 1; i < len(agentLines); i++ {
			if len(agentLines[i]) > 2 && agentLines[i][0:2] == "  " && agentLines[i][2] != ' ' && strings.Contains(agentLines[i], ":") {
				agentEndIdx = 0
				for j := 0; j < i; j++ {
					agentEndIdx += len(agentLines[j]) + 1
				}
				break
			}
		}
		agentSection = agentSection[:agentEndIdx]

		if !strings.Contains(agentSection, "needs: activation") {
			t.Error("Expected agent job to depend on activation job")
		}
	})
}
