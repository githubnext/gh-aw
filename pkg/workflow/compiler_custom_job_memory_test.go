//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// restore-memory: Tests for custom jobs
// ========================================

// TestCustomJobRestoreMemoryCacheMemory verifies that a custom job with
// restore-memory.cache-memory: true gets cache restore steps injected.
// No artifact-upload or cache-save steps should be emitted.
func TestCustomJobRestoreMemoryCacheMemory(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-cache-memory")

	frontmatter := `---
name: Orchestrator
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
    restore-memory:
      cache-memory: true
    steps:
      - name: Read memory and dispatch
        run: echo "dispatching"
---

# Orchestrator Workflow

Reads cache memory and dispatches tasks.
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section, "Expected orchestrator job section in lock file")

	// Must contain cache restore step
	assert.Contains(t, section, "actions/cache/restore@", "cache-memory restore step should use actions/cache/restore")
	assert.Contains(t, section, "Create cache-memory directory", "dir creation step should be present")
	assert.Contains(t, section, "Restore cache-memory", "cache restore step should be present")
	assert.Contains(t, section, "restore_cache_memory_0", "cache restore step ID should be present")

	// Must NOT contain write-back steps
	assert.NotContains(t, section, "actions/cache/save@", "no cache-save step should be emitted")
	assert.NotContains(t, section, "actions/cache@", "no write-mode cache step should be emitted")
	assert.NotContains(t, section, "actions/upload-artifact@", "no artifact-upload step should be emitted")
	assert.NotContains(t, section, "Setup cache-memory git", "git integrity setup should not be emitted for read-only restore")
	assert.NotContains(t, section, "commit_cache_memory_git", "no git commit script should be emitted")

	// The main agent job should also have cache-memory steps
	agentSection := extractJobSection(yamlStr, "agent")
	assert.Contains(t, agentSection, "Restore cache-memory file share data", "agent job should still have its own cache restore step")

	// No separate update_cache_memory job (threat detection not enabled)
	assert.NotContains(t, yamlStr, "update_cache_memory:", "update_cache_memory job should not be created without threat detection")
}

// TestCustomJobRestoreMemoryRepoMemory verifies that a custom job with
// restore-memory.repo-memory: true gets repo-memory clone steps injected.
func TestCustomJobRestoreMemoryRepoMemory(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-repo-memory")

	frontmatter := `---
name: Orchestrator
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  repo-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      repo-memory: true
    steps:
      - name: Read repo memory
        run: cat /tmp/gh-aw/repo-memory/default/state.json || echo "{}"
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section, "Expected orchestrator job section in lock file")

	// Must contain repo-memory clone step
	assert.Contains(t, section, "Clone repo-memory branch", "clone step should be present")
	assert.Contains(t, section, "clone_repo_memory_branch.sh", "clone script should be referenced")
	assert.Contains(t, section, "GH_TOKEN:", "GH_TOKEN env var should be set for clone")

	// Must NOT contain write-back steps (push job is a separate job, not injected here)
	assert.NotContains(t, section, "push_repo_memory", "no push step in orchestrator job")
}

// TestCustomJobRestoreMemoryCommentMemory verifies that a custom job with
// restore-memory.comment-memory: true gets the prepare-comment-memory step injected.
func TestCustomJobRestoreMemoryCommentMemory(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-comment-memory")

	frontmatter := `---
name: Orchestrator
on: workflow_dispatch
permissions:
  contents: read
  pull-requests: read
engine: copilot
strict: false
tools:
  comment-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      comment-memory: true
    steps:
      - name: Process comment memory
        run: ls /tmp/gh-aw/comment-memory/ || echo "empty"
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section, "Expected orchestrator job section in lock file")

	// Must contain prepare comment memory step
	assert.Contains(t, section, "Prepare comment memory files", "comment memory prepare step should be present")
	assert.Contains(t, section, "setup_comment_memory_files.cjs", "comment memory CJS script should be referenced")
	assert.Contains(t, section, "actions/github-script@", "github-script action should be used")
}

// TestCustomJobRestoreMemoryMultipleTypes verifies that a custom job can request
// multiple memory types at once.
func TestCustomJobRestoreMemoryMultipleTypes(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-multiple-memory")

	frontmatter := `---
name: Orchestrator
on: workflow_dispatch
permissions:
  contents: read
  pull-requests: read
engine: copilot
strict: false
tools:
  cache-memory: true
  repo-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      cache-memory: true
      repo-memory: true
    steps:
      - name: Process memories
        run: echo "done"
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section)

	assert.Contains(t, section, "Restore cache-memory", "cache-memory steps should be present")
	assert.Contains(t, section, "Clone repo-memory branch", "repo-memory steps should be present")
}

// TestCustomJobRestoreMemoryStepOrder verifies that restore-memory steps are placed
// after GHES host config and before pre-steps and regular steps.
func TestCustomJobRestoreMemoryStepOrder(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-memory-order")

	frontmatter := `---
name: Orchestrator
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
    restore-memory:
      cache-memory: true
    pre-steps:
      - name: My pre-step
        run: echo "pre"
    steps:
      - name: My main step
        run: echo "main"
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section)

	// Expected order:
	// 1. Configure GH_HOST for enterprise compatibility
	// 2. Create cache-memory directory (restore-memory)
	// 3. Restore cache-memory (restore-memory)
	// 4. My pre-step (pre-steps)
	// 5. My main step (steps)
	assertStepOrderInSection(t, section,
		"- name: Configure GH_HOST for enterprise compatibility",
		"- name: Create cache-memory directory",
		"- name: Restore cache-memory",
		"- name: My pre-step",
		"- name: My main step",
	)
}

// TestCustomJobRestoreMemoryErrorWhenNotConfigured verifies that a custom job
// requesting restore-memory for a type not configured in tools: produces an error.
func TestCustomJobRestoreMemoryErrorWhenNotConfigured(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		wantErrMsg  string
	}{
		{
			name: "cache-memory not configured",
			frontmatter: `---
name: Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      cache-memory: true
    steps:
      - name: Step
        run: echo hi
---
# Test
`,
			wantErrMsg: "cache-memory must be configured in tools:",
		},
		{
			name: "repo-memory not configured",
			frontmatter: `---
name: Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      repo-memory: true
    steps:
      - name: Step
        run: echo hi
---
# Test
`,
			wantErrMsg: "repo-memory must be configured in tools:",
		},
		{
			name: "comment-memory not configured",
			frontmatter: `---
name: Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      comment-memory: true
    steps:
      - name: Step
        run: echo hi
---
# Test
`,
			wantErrMsg: "comment-memory must be configured in tools:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "restore-memory-error-*")
			testFile := filepath.Join(tmpDir, "test.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tc.frontmatter), 0644))

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(testFile)
			require.Error(t, err, "expected compilation to fail")
			assert.Contains(t, err.Error(), tc.wantErrMsg)
		})
	}
}

// TestCustomJobRestoreMemoryOnlyEmitsRestoreSteps verifies that when restore-memory
// is configured, no write-back steps (artifact upload, cache save, git commit, push)
// are emitted for the custom job.
func TestCustomJobRestoreMemoryOnlyEmitsRestoreSteps(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-only")

	frontmatter := `---
name: Orchestrator
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
tools:
  cache-memory: true
  repo-memory: true
jobs:
  orchestrator:
    runs-on: ubuntu-latest
    restore-memory:
      cache-memory: true
      repo-memory: true
    steps:
      - name: Process data
        run: echo "processing"
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	orchestratorSection := extractJobSection(string(content), "orchestrator")
	require.NotEmpty(t, orchestratorSection)

	writePatterns := []string{
		"actions/upload-artifact@",
		"actions/cache/save@",
		"commit_cache_memory_git.sh",
		"push_repo_memory",
		"Setup cache-memory git",
	}
	for _, pattern := range writePatterns {
		assert.NotContains(t, orchestratorSection, pattern,
			"write-back pattern %q should not be emitted in restore-memory custom job", pattern)
	}
}

// TestCustomJobRestoreMemoryStandaloneJob verifies that restore-memory works even
// when no steps, pre-steps, or setup-steps are provided (steps-only trigger).
func TestCustomJobRestoreMemoryStandaloneJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-job-restore-standalone")

	frontmatter := `---
name: Orchestrator
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
    restore-memory:
      cache-memory: true
---

# Orchestrator Workflow
`

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(frontmatter), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(content)
	section := extractJobSection(yamlStr, "orchestrator")
	require.NotEmpty(t, section, "orchestrator job should appear even without explicit steps")

	assert.Contains(t, section, "Restore cache-memory", "restore step must be present")
	assert.Contains(t, section, "Configure GH_HOST", "GHES config step must be present")
}

// TestExtractRestoreMemoryConfig unit-tests the config extraction logic.
func TestExtractRestoreMemoryConfig(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]any
		want      *restoreMemoryConfig
		wantErr   bool
	}{
		{
			name:      "no restore-memory field",
			configMap: map[string]any{},
			want:      nil,
		},
		{
			name: "all false values returns nil",
			configMap: map[string]any{
				"restore-memory": map[string]any{
					"cache-memory":   false,
					"repo-memory":    false,
					"comment-memory": false,
				},
			},
			want: nil,
		},
		{
			name: "cache-memory only",
			configMap: map[string]any{
				"restore-memory": map[string]any{
					"cache-memory": true,
				},
			},
			want: &restoreMemoryConfig{CacheMemory: true},
		},
		{
			name: "all enabled",
			configMap: map[string]any{
				"restore-memory": map[string]any{
					"cache-memory":   true,
					"repo-memory":    true,
					"comment-memory": true,
				},
			},
			want: &restoreMemoryConfig{CacheMemory: true, RepoMemory: true, CommentMemory: true},
		},
		{
			name: "invalid type returns error",
			configMap: map[string]any{
				"restore-memory": "true",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractRestoreMemoryConfig(tc.configMap, "test_job")
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestValidateRestoreMemoryConfig unit-tests the config validation logic.
func TestValidateRestoreMemoryConfig(t *testing.T) {
	emptyCacheData := &WorkflowData{}
	cacheData := &WorkflowData{
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{{ID: "default"}},
		},
	}
	repoData := &WorkflowData{
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{{ID: "default"}},
		},
	}
	commentData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CommentMemory: &CommentMemoryConfig{},
		},
	}

	tests := []struct {
		name    string
		cfg     *restoreMemoryConfig
		data    *WorkflowData
		wantErr bool
	}{
		{name: "nil config is ok", cfg: nil, data: emptyCacheData, wantErr: false},
		{name: "cache-memory configured correctly", cfg: &restoreMemoryConfig{CacheMemory: true}, data: cacheData, wantErr: false},
		{name: "cache-memory missing from tools", cfg: &restoreMemoryConfig{CacheMemory: true}, data: emptyCacheData, wantErr: true},
		{name: "repo-memory configured correctly", cfg: &restoreMemoryConfig{RepoMemory: true}, data: repoData, wantErr: false},
		{name: "repo-memory missing from tools", cfg: &restoreMemoryConfig{RepoMemory: true}, data: emptyCacheData, wantErr: true},
		{name: "comment-memory configured correctly", cfg: &restoreMemoryConfig{CommentMemory: true}, data: commentData, wantErr: false},
		{name: "comment-memory missing from tools", cfg: &restoreMemoryConfig{CommentMemory: true}, data: emptyCacheData, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRestoreMemoryConfig(tc.cfg, tc.data, "test_job")
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGenerateCacheMemoryRestoreLines unit-tests the cache restore line generation.
func TestGenerateCacheMemoryRestoreLines(t *testing.T) {
	data := &WorkflowData{
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{
					ID:    "default",
					Key:   "memory-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
					Scope: "workflow",
				},
			},
		},
	}

	lines := generateCacheMemoryRestoreLines(data)
	combined := strings.Join(lines, "")

	assert.Contains(t, combined, "mkdir -p", "should have inline mkdir")
	assert.Contains(t, combined, "actions/cache/restore@", "should use restore-only action")
	assert.NotContains(t, combined, "actions/cache@", "should not use read-write action")
	assert.NotContains(t, combined, "setup_cache_memory_git", "should not include git setup script")
	assert.Contains(t, combined, "restore_cache_memory_0", "should have step ID")
}

// TestGenerateCacheMemoryRestoreLinesNilData verifies graceful handling of nil config.
func TestGenerateCacheMemoryRestoreLinesNilData(t *testing.T) {
	assert.Nil(t, generateCacheMemoryRestoreLines(&WorkflowData{}))
}
