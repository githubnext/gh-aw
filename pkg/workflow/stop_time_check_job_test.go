package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStopTimeCheckJob tests that stop-time check job is created correctly
func TestStopTimeCheckJob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stop-time-check-job-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	t.Run("stop_time_check_job_created_with_stop_after", func(t *testing.T) {
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

		// Verify stop_time_check job exists
		if !strings.Contains(lockContentStr, "stop_time_check:") {
			t.Error("Expected stop_time_check job to be created")
		}

		// Verify stop_time_check job has actions: write permission
		if !strings.Contains(lockContentStr, "actions: write  # Required for gh workflow disable") {
			t.Error("Expected stop_time_check job to have actions: write permission")
		}

		// Verify safety checks are in stop_time_check job, not main job
		stopTimeCheckStart := strings.Index(lockContentStr, "stop_time_check:")
		agentStart := strings.Index(lockContentStr, "agent:")
		safetyChecksPos := strings.Index(lockContentStr, "Performing safety checks before executing agentic tools")

		if safetyChecksPos == -1 {
			t.Error("Expected safety checks to be present")
		}

		// Safety checks should be in stop_time_check job (before agent job)
		if safetyChecksPos > agentStart {
			t.Error("Safety checks should be in stop_time_check job, not in agent job")
		}

		// Safety checks should be after stop_time_check job start
		if safetyChecksPos < stopTimeCheckStart {
			t.Error("Safety checks should be in stop_time_check job")
		}
	})

	t.Run("no_stop_time_check_job_without_stop_after", func(t *testing.T) {
		workflowContent := `---
on:
  workflow_dispatch:
engine: claude
---

# Normal Workflow

This workflow has no stop-after configuration.
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

		// Verify stop_time_check job does not exist
		if strings.Contains(lockContentStr, "stop_time_check:") {
			t.Error("Expected NO stop_time_check job without stop-after")
		}
	})

	t.Run("stop_time_check_job_depends_on_activation", func(t *testing.T) {
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

		// Verify stop_time_check job exists
		if !strings.Contains(lockContentStr, "stop_time_check:") {
			t.Error("Expected stop_time_check job")
		}

		// Verify the job dependency chain
		// Extract the jobs section to analyze dependencies
		jobsSection := lockContentStr[strings.Index(lockContentStr, "jobs:"):]

		// Find stop_time_check job section
		stopTimeCheckIdx := strings.Index(jobsSection, "stop_time_check:")
		if stopTimeCheckIdx == -1 {
			t.Error("Expected stop_time_check job")
		}

		// Find the needs line in stop_time_check job
		stopTimeCheckSection := jobsSection[stopTimeCheckIdx:]
		nextJobIdx := strings.Index(stopTimeCheckSection[20:], "\n  ")
		if nextJobIdx != -1 {
			stopTimeCheckSection = stopTimeCheckSection[:nextJobIdx+20]
		}

		if !strings.Contains(stopTimeCheckSection, "needs: activation") {
			t.Error("Expected stop_time_check job to depend on activation job")
		}

		// Verify agent job exists and depends on activation (not stop_time_check)
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
