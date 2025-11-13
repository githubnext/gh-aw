package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompilerGoldenFiles validates compiler output against golden files
func TestCompilerGoldenFiles(t *testing.T) {
	tests := []struct {
		name         string
		workflowFile string
		goldenFile   string
	}{
		{
			name:         "issue trigger workflow",
			workflowFile: "issue_trigger.md",
			goldenFile:   "issue_trigger.lock.yml",
		},
		{
			name:         "pull request trigger workflow",
			workflowFile: "pr_trigger.md",
			goldenFile:   "pr_trigger.lock.yml",
		},
		{
			name:         "scheduled workflow",
			workflowFile: "scheduled.md",
			goldenFile:   "scheduled.lock.yml",
		},
		{
			name:         "command trigger workflow",
			workflowFile: "command_trigger.md",
			goldenFile:   "command_trigger.lock.yml",
		},
		{
			name:         "multi-job workflow",
			workflowFile: "multi_job.md",
			goldenFile:   "multi_job.lock.yml",
		},
		{
			name:         "workflow with MCP servers",
			workflowFile: "mcp_servers.md",
			goldenFile:   "mcp_servers.lock.yml",
		},
		{
			name:         "workflow with safe outputs",
			workflowFile: "safe_outputs.md",
			goldenFile:   "safe_outputs.lock.yml",
		},
		{
			name:         "workflow with network permissions",
			workflowFile: "network_permissions.md",
			goldenFile:   "network_permissions.lock.yml",
		},
		{
			name:         "workflow with imports",
			workflowFile: "with_imports.md",
			goldenFile:   "with_imports.lock.yml",
		},
		{
			name:         "custom engine workflow",
			workflowFile: "custom_engine.md",
			goldenFile:   "custom_engine.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup paths
			workflowPath := filepath.Join("testdata", "workflows", tt.workflowFile)
			goldenPath := filepath.Join("testdata", "golden", tt.goldenFile)

			// Create a temporary directory for compilation output
			tmpDir, err := os.MkdirTemp("", "golden-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Copy the workflow file to temp directory
			workflowContent, err := os.ReadFile(workflowPath)
			if err != nil {
				t.Fatalf("failed to read workflow file %s: %v", workflowPath, err)
			}

			tmpWorkflowPath := filepath.Join(tmpDir, tt.workflowFile)
			if err := os.WriteFile(tmpWorkflowPath, workflowContent, 0644); err != nil {
				t.Fatalf("failed to write temp workflow file: %v", err)
			}

			// Copy shared files if they exist (for imports test)
			sharedSrcDir := filepath.Join("testdata", "workflows", "shared")
			if _, err := os.Stat(sharedSrcDir); err == nil {
				sharedDstDir := filepath.Join(tmpDir, "shared")
				if err := os.MkdirAll(sharedDstDir, 0755); err != nil {
					t.Fatalf("failed to create shared directory: %v", err)
				}

				entries, err := os.ReadDir(sharedSrcDir)
				if err != nil {
					t.Fatalf("failed to read shared directory: %v", err)
				}

				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					content, err := os.ReadFile(filepath.Join(sharedSrcDir, entry.Name()))
					if err != nil {
						t.Fatalf("failed to read shared file %s: %v", entry.Name(), err)
					}
					if err := os.WriteFile(filepath.Join(sharedDstDir, entry.Name()), content, 0644); err != nil {
						t.Fatalf("failed to write shared file %s: %v", entry.Name(), err)
					}
				}
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test-version")
			err = compiler.CompileWorkflow(tmpWorkflowPath)
			if err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			// Read the generated lock file
			lockFilePath := filepath.Join(tmpDir, tt.workflowFile[:len(tt.workflowFile)-3]+".lock.yml")
			got, err := os.ReadFile(lockFilePath)
			if err != nil {
				t.Fatalf("failed to read generated lock file: %v", err)
			}

			// Compare with golden file
			CompareGoldenFile(t, got, goldenPath)
		})
	}
}
