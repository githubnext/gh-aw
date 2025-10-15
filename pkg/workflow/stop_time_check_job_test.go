package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreActivationJob tests that pre_activation job is created correctly with stop-time checks
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

		// Verify stop-time checks are in pre_activation job
		preActivationStart := strings.Index(lockContentStr, "pre_activation:")
		agentStart := strings.Index(lockContentStr, "agent:")
		stopTimeCheckPos := strings.Index(lockContentStr, "Checking stop-time limit")

		if stopTimeCheckPos == -1 {
			t.Error("Expected stop-time checks to be present")
		}

		// Stop-time checks should be in pre_activation job (before agent job)
		if stopTimeCheckPos > agentStart {
			t.Error("Stop-time checks should be in pre_activation job, not in agent job")
		}

		// Stop-time checks should be after pre_activation job start
		if stopTimeCheckPos < preActivationStart {
			t.Error("Stop-time checks should be in pre_activation job")
		}
	})

	t.Run("no_pre_activation_job_without_stop_after_or_roles", func(t *testing.T) {
		workflowContent := `---
on:
  workflow_dispatch:
engine: claude
roles: all
---

# Normal Workflow

This workflow has no stop-after configuration and roles: all.
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

		// Verify pre_activation job does not exist
		if strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected NO pre_activation job without stop-after or required roles")
		}
	})

	t.Run("pre_activation_job_with_activation", func(t *testing.T) {
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

		// Verify activation job exists
		if !strings.Contains(lockContentStr, "activation:") {
			t.Error("Expected activation job for unsafe events")
		}

		// Verify pre_activation job exists
		if !strings.Contains(lockContentStr, "pre_activation:") {
			t.Error("Expected pre_activation job")
		}

		// Verify the job dependency chain
		// Extract the jobs section to analyze dependencies
		jobsSection := lockContentStr[strings.Index(lockContentStr, "jobs:"):]

		// Find activation job section and verify it depends on pre_activation
		activationIdx := strings.Index(jobsSection, "activation:")
		if activationIdx == -1 {
			t.Error("Expected activation job")
		}

		activationSection := jobsSection[activationIdx:]
		// Find the next job (starts with "\n  " followed by a non-whitespace character at the beginning of a line)
		nextJobIdx := strings.Index(activationSection[20:], "\nagent:")
		if nextJobIdx != -1 {
			activationSection = activationSection[:nextJobIdx+20]
		}

		if !strings.Contains(activationSection, "needs: pre_activation") {
			t.Errorf("Expected activation job to depend on pre_activation job. Got activation section:\n%s", activationSection)
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
		nextJobIdx = strings.Index(agentSection[20:], "\n  ")
		if nextJobIdx != -1 {
			agentSection = agentSection[:nextJobIdx+20]
		}

		if !strings.Contains(agentSection, "needs: activation") {
			t.Error("Expected agent job to depend on activation job")
		}
	})
}
