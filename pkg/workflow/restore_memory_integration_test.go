//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestRestoreMemoryWithUsesError verifies that restore-memory: true on a reusable
// workflow call job (one that has a `uses:` key) produces a compile-time error with
// a clear message, rather than being silently ignored.
func TestRestoreMemoryWithUsesError(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-uses-error")
	workflowPath := filepath.Join(tmpDir, "restore-memory-uses.md")

	content := `---
name: Restore Memory Uses Error
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  reusable-caller:
    uses: ./.github/workflows/called.yml
    restore-memory: true
---

# Restore Memory Uses Error
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	err := compiler.CompileWorkflow(workflowPath)
	if err == nil {
		t.Fatal("Expected compile error for restore-memory on reusable workflow job, got nil")
	}
	if !strings.Contains(err.Error(), "restore-memory") {
		t.Errorf("Error should mention restore-memory, got: %v", err)
	}
	if !strings.Contains(err.Error(), "reusable") || !strings.Contains(err.Error(), "reusable-caller") {
		t.Errorf("Error should mention the job name and reusable context, got: %v", err)
	}
}

// the PR: a scheduled job that reads cache-memory to build a dispatch list before
// the agent job runs.  The agent job must still carry its own full cache write-path;
// the orchestrator job must carry only read-only restore steps.
func TestRestoreMemoryScheduledOrchestrator(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-orchestrator")
	workflowPath := filepath.Join(tmpDir, "orchestrator.md")

	content := `---
name: Scheduled Orchestrator
on:
  schedule:
    - cron: "0 * * * *"
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    needs: []
    restore-memory: true
    steps:
      - name: Read state and dispatch
        run: |
          state=$(cat /tmp/gh-aw/cache-memory/state.json 2>/dev/null || echo '{}')
          echo "state=$state"
---

# Scheduled Orchestrator

Reads cache-memory state and dispatches tasks accordingly.
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	orchestratorSection := extractJobSection(lockFile, "orchestrator")
	if orchestratorSection == "" {
		t.Fatal("Expected orchestrator job section in lock file")
	}

	// Orchestrator must have read-only restore steps.
	if !strings.Contains(orchestratorSection, "Create cache-memory directory") {
		t.Error("Expected cache-memory directory creation step in orchestrator job")
	}
	if !strings.Contains(orchestratorSection, "actions/cache/restore@") {
		t.Error("Expected actions/cache/restore step in orchestrator job")
	}
	if !strings.Contains(orchestratorSection, "Restore cache-memory") {
		t.Error("Expected Restore cache-memory step in orchestrator job")
	}

	// Orchestrator must NOT have write-back steps.
	if strings.Contains(orchestratorSection, "actions/cache@") && !strings.Contains(orchestratorSection, "actions/cache/restore@") {
		t.Error("Orchestrator job must not use the read-write actions/cache action")
	}
	if strings.Contains(orchestratorSection, "actions/upload-artifact@") {
		t.Error("Orchestrator job must not upload artifacts")
	}
	if strings.Contains(orchestratorSection, "Setup cache-memory git") {
		t.Error("Orchestrator job must not set up git integrity")
	}

	// Agent job must still have its own full cache path (restore + save).
	agentSection := extractJobSection(lockFile, "agent")
	if agentSection == "" {
		t.Fatal("Expected agent job section in lock file")
	}
	if !strings.Contains(agentSection, "Restore cache-memory file share data") {
		t.Error("Agent job should still have its own cache restore step")
	}
}

// TestRestoreMemoryWorkflowIDSanitizedEnvInjection verifies that when cache-memory
// restore is requested the compiler injects GH_AW_WORKFLOW_ID_SANITIZED into the
// custom job's env block so that the cache key matches what the agent job uses.
func TestRestoreMemoryWorkflowIDSanitizedEnvInjection(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-env-inject")
	// File name: "my-orchestrator.md"
	// GetWorkflowIDFromPath → "my-orchestrator"
	// SanitizeWorkflowIDForCacheKey → "myorchestrator"
	workflowPath := filepath.Join(tmpDir, "my-orchestrator.md")

	content := `---
name: Env Inject Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory: true
    steps:
      - name: Use cache
        run: ls /tmp/gh-aw/cache-memory/
---

# Env Inject Test
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	orchestratorSection := extractJobSection(lockFile, "orchestrator")
	if orchestratorSection == "" {
		t.Fatal("Expected orchestrator job section in lock file")
	}

	// The env block for the orchestrator job should carry the sanitized workflow ID.
	// SanitizeWorkflowIDForCacheKey("my-orchestrator") == "myorchestrator"
	if !strings.Contains(orchestratorSection, "GH_AW_WORKFLOW_ID_SANITIZED") {
		t.Error("Expected GH_AW_WORKFLOW_ID_SANITIZED in orchestrator job env")
	}
	if !strings.Contains(orchestratorSection, "myorchestrator") {
		t.Error("Expected sanitized workflow ID 'myorchestrator' in orchestrator job env")
	}
}

// TestRestoreMemoryFalseIsNoop verifies that restore-memory: false does not inject
// any restore steps into the custom job, even when memory stores are configured.
func TestRestoreMemoryFalseIsNoop(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-noop")
	workflowPath := filepath.Join(tmpDir, "noop.md")

	content := `---
name: No-Op Restore Memory
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  processor:
    runs-on: ubuntu-latest
    restore-memory: false
    steps:
      - name: Do something else
        run: echo "no memory needed"
---

# No-Op Test
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	processorSection := extractJobSection(lockFile, "processor")
	if processorSection == "" {
		t.Fatal("Expected processor job section in lock file")
	}

	if strings.Contains(processorSection, "actions/cache/restore@") {
		t.Error("No cache restore step should be injected when restore-memory is false")
	}
	if strings.Contains(processorSection, "Restore cache-memory") {
		t.Error("No restore step name should appear when restore-memory is false")
	}
	if strings.Contains(processorSection, "Create cache-memory directory") {
		t.Error("No directory creation step should be injected when restore-memory is false")
	}
}

// TestRestoreMemoryMultipleNamedCaches verifies that all named cache entries receive
// their own restore steps when the cache-memory tool uses the array form.
func TestRestoreMemoryMultipleNamedCaches(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-multi-cache")
	workflowPath := filepath.Join(tmpDir, "multi-cache.md")

	content := `---
name: Multi Cache Restore
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session
jobs:
  aggregator:
    runs-on: ubuntu-latest
    restore-memory: true
    steps:
      - name: Aggregate results
        run: |
          cat /tmp/gh-aw/cache-memory/result.json || true
          cat /tmp/gh-aw/cache-memory-session/session.json || true
---

# Multi Cache Restore
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	aggregatorSection := extractJobSection(lockFile, "aggregator")
	if aggregatorSection == "" {
		t.Fatal("Expected aggregator job section in lock file")
	}

	// Both named caches must have their own mkdir and restore steps.
	if !strings.Contains(aggregatorSection, "Create cache-memory directory (default)") {
		t.Error("Expected create step for default cache")
	}
	if !strings.Contains(aggregatorSection, "Restore cache-memory (default)") {
		t.Error("Expected restore step for default cache")
	}
	if !strings.Contains(aggregatorSection, "Create cache-memory directory (session)") {
		t.Error("Expected create step for session cache")
	}
	if !strings.Contains(aggregatorSection, "Restore cache-memory (session)") {
		t.Error("Expected restore step for session cache")
	}

	// Step IDs must be present and distinct.
	if !strings.Contains(aggregatorSection, "restore_cache_memory_0") {
		t.Error("Expected restore_cache_memory_0 step ID for first cache")
	}
	if !strings.Contains(aggregatorSection, "restore_cache_memory_1") {
		t.Error("Expected restore_cache_memory_1 step ID for second cache")
	}

	// No write-back steps.
	if strings.Contains(aggregatorSection, "actions/upload-artifact@") {
		t.Error("No artifact upload should be emitted in restore-memory job")
	}
}

// TestRestoreMemoryJobIsolation verifies that when two custom jobs are defined and
// only one has restore-memory: true, no restore steps bleed into the other job.
func TestRestoreMemoryJobIsolation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-isolation")
	workflowPath := filepath.Join(tmpDir, "isolation.md")

	content := `---
name: Job Isolation Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  with_restore:
    runs-on: ubuntu-latest
    restore-memory: true
    steps:
      - name: Read memory
        run: cat /tmp/gh-aw/cache-memory/data.json || true
  without_restore:
    runs-on: ubuntu-latest
    steps:
      - name: Other task
        run: echo "I do not use memory"
---

# Job Isolation Test
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	withRestoreSection := extractJobSection(lockFile, "with_restore")
	if withRestoreSection == "" {
		t.Fatal("Expected with_restore job section in lock file")
	}
	if !strings.Contains(withRestoreSection, "actions/cache/restore@") {
		t.Error("Expected cache restore step in with_restore job")
	}

	withoutRestoreSection := extractJobSection(lockFile, "without_restore")
	if withoutRestoreSection == "" {
		t.Fatal("Expected without_restore job section in lock file")
	}
	if strings.Contains(withoutRestoreSection, "actions/cache/restore@") {
		t.Error("Cache restore steps must not bleed into without_restore job")
	}
	if strings.Contains(withoutRestoreSection, "Restore cache-memory") {
		t.Error("Restore step name must not appear in without_restore job")
	}
}

// TestRestoreMemoryNonBooleanError verifies that a non-boolean restore-memory value
// (e.g. a quoted string) produces a compile-time error with a clear message.
func TestRestoreMemoryNonBooleanError(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-bad-type")
	workflowPath := filepath.Join(tmpDir, "bad-type.md")

	content := `---
name: Bad Type Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory: "yes"
    steps:
      - name: Step
        run: echo hi
---

# Bad Type Test
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	err := compiler.CompileWorkflow(workflowPath)
	if err == nil {
		t.Fatal("Expected compile error for non-boolean restore-memory, got nil")
	}
	if !strings.Contains(err.Error(), "restore-memory") {
		t.Errorf("Error should mention restore-memory, got: %v", err)
	}
	if !strings.Contains(err.Error(), "boolean") {
		t.Errorf("Error should mention boolean type, got: %v", err)
	}
}

// TestRestoreMemoryRepoMemoryInCustomJob verifies that a custom job with
// restore-memory: true and a repo-memory tool gets the read-only clone steps
// injected, but no push/write-back steps.
func TestRestoreMemoryRepoMemoryInCustomJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-repo")
	workflowPath := filepath.Join(tmpDir, "repo-restore.md")

	content := `---
name: Repo Memory Restore
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  repo-memory: true
jobs:
  reader:
    runs-on: ubuntu-latest
    restore-memory: true
    steps:
      - name: Read state
        run: cat /tmp/gh-aw/repo-memory/default/state.json || echo '{}'
---

# Repo Memory Restore
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	readerSection := extractJobSection(lockFile, "reader")
	if readerSection == "" {
		t.Fatal("Expected reader job section in lock file")
	}

	// Clone step must be present.
	if !strings.Contains(readerSection, "Clone repo-memory branch") {
		t.Error("Expected Clone repo-memory branch step in reader job")
	}
	if !strings.Contains(readerSection, "clone_repo_memory_branch.sh") {
		t.Error("Expected clone_repo_memory_branch.sh script reference in reader job")
	}

	// No push step must be present in the reader custom job (push lives in push_repo_memory job).
	if strings.Contains(readerSection, "push_repo_memory") {
		t.Error("Push step must not appear in restore-memory custom job")
	}

	// Agent job still has its own push (the push_repo_memory job must exist).
	if !strings.Contains(lockFile, "push_repo_memory") {
		t.Error("Expected push_repo_memory job for the agent write-path")
	}
}

// TestRestoreMemoryNeedsAndRestoreMemory verifies that a custom job can declare both
// an explicit needs dependency and restore-memory: true in the same job, and that
// the compiled output reflects both the dependency and the injected restore steps.
func TestRestoreMemoryNeedsAndRestoreMemory(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-needs")
	workflowPath := filepath.Join(tmpDir, "needs-and-restore.md")

	content := `---
name: Needs And Restore
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  setup:
    runs-on: ubuntu-latest
    needs: []
    steps:
      - name: Initialize
        run: echo "setting up"
    outputs:
      done: ${{ steps.init.outputs.done }}
  consumer:
    runs-on: ubuntu-latest
    needs: [setup]
    restore-memory: true
    steps:
      - name: Consume memory
        run: cat /tmp/gh-aw/cache-memory/output.json || true
---

# Needs And Restore
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	consumerSection := extractJobSection(lockFile, "consumer")
	if consumerSection == "" {
		t.Fatal("Expected consumer job section in lock file")
	}

	// consumer must depend on setup.
	if !strings.Contains(consumerSection, "setup") {
		t.Error("Expected consumer job to depend on setup job")
	}

	// consumer must also have cache restore steps.
	if !strings.Contains(consumerSection, "Restore cache-memory") {
		t.Error("Expected Restore cache-memory step in consumer job")
	}
	if !strings.Contains(consumerSection, "actions/cache/restore@") {
		t.Error("Expected actions/cache/restore action in consumer job")
	}
}

// TestRestoreMemoryStepOrderIntegration is an end-to-end check that GHES host config
// comes before restore-memory steps, which in turn come before user-defined steps,
// mirroring the ordering requirements from the unit tests.
func TestRestoreMemoryStepOrderIntegration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "restore-memory-step-order")
	workflowPath := filepath.Join(tmpDir, "step-order.md")

	content := `---
name: Step Order Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
jobs:
  ordered:
    runs-on: ubuntu-latest
    restore-memory: true
    pre-steps:
      - name: My Pre Step
        run: echo "pre"
    steps:
      - name: My Main Step
        run: echo "main"
---

# Step Order Test
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockFile := string(raw)

	orderedSection := extractJobSection(lockFile, "ordered")
	if orderedSection == "" {
		t.Fatal("Expected ordered job section in lock file")
	}

	ghesPos := strings.Index(orderedSection, "Configure GH_HOST")
	restorePos := strings.Index(orderedSection, "Restore cache-memory")
	preStepPos := strings.Index(orderedSection, "My Pre Step")
	mainStepPos := strings.Index(orderedSection, "My Main Step")

	if ghesPos < 0 {
		t.Fatal("GH_HOST configuration step not found in ordered job")
	}
	if restorePos < 0 {
		t.Fatal("Restore cache-memory step not found in ordered job")
	}
	if preStepPos < 0 {
		t.Fatal("My Pre Step not found in ordered job")
	}
	if mainStepPos < 0 {
		t.Fatal("My Main Step not found in ordered job")
	}

	if ghesPos >= restorePos {
		t.Error("GH_HOST configuration must come before restore-memory steps")
	}
	if restorePos >= preStepPos {
		t.Error("restore-memory steps must come before pre-steps")
	}
	if preStepPos >= mainStepPos {
		t.Error("pre-steps must come before main steps")
	}
}
