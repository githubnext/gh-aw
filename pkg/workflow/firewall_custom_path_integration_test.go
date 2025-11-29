package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirewallCustomPathCompilation(t *testing.T) {
	tests := []struct {
		name            string
		markdown        string
		wantContains    []string
		dontWantContain []string
	}{
		{
			name: "custom absolute path skips installation",
			markdown: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall:
    path: /usr/local/bin/awf-custom
  allowed:
    - defaults
---

# Test workflow with custom absolute AWF path
`,
			wantContains: []string{
				"Validate custom AWF binary",
				"/usr/local/bin/awf-custom",
				"--version",
				"if [ ! -f",
				"if [ ! -x",
			},
			dontWantContain: []string{
				"Install awf binary",
				"curl -L https://github.com/githubnext/gh-aw-firewall",
			},
		},
		{
			name: "custom relative path resolved to workspace",
			markdown: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall:
    path: bin/awf
  allowed:
    - defaults
---

# Test workflow with relative AWF path
`,
			wantContains: []string{
				"Validate custom AWF binary",
				"${GITHUB_WORKSPACE}/bin/awf",
			},
			dontWantContain: []string{
				"Install awf binary",
			},
		},
		{
			name: "no path triggers default installation",
			markdown: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall: true
  allowed:
    - defaults
---

# Test workflow with default AWF installation
`,
			wantContains: []string{
				"Install awf binary",
				"curl -L https://github.com/githubnext/gh-aw-firewall",
			},
			dontWantContain: []string{
				"Validate custom AWF binary",
			},
		},
		{
			name: "path with version ignores version",
			markdown: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall:
    path: /custom/awf
    version: v999.0.0
  allowed:
    - defaults
---

# Test workflow where path takes precedence over version
`,
			wantContains: []string{
				"Validate custom AWF binary",
				"/custom/awf",
			},
			dontWantContain: []string{
				"Install awf binary",
				"v999.0.0",
			},
		},
		{
			name: "custom path used in execution command",
			markdown: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall:
    path: /opt/awf
  allowed:
    - defaults
---

# Test that custom path is used in execution
`,
			wantContains: []string{
				"sudo -E /opt/awf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir, err := os.MkdirTemp("", "awf-path-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Write the test workflow file
			workflowFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowFile, []byte(tt.markdown), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(workflowFile); err != nil {
				t.Fatalf("CompileWorkflow() error: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			result := string(lockContent)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Compiled workflow missing expected content: %q\n\nGot:\n%s", want, result)
				}
			}

			for _, dontWant := range tt.dontWantContain {
				if strings.Contains(result, dontWant) {
					t.Errorf("Compiled workflow contains unexpected content: %q", dontWant)
				}
			}
		})
	}
}

func TestFirewallCustomPathBackwardCompatibility(t *testing.T) {
	// Ensure existing workflows without path field continue to work
	markdown := `---
on: push
permissions:
  contents: read
engine: copilot
network:
  firewall:
    version: v1.2.3
    log-level: debug
  allowed:
    - defaults
---

# Existing workflow without custom path
`

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "awf-compat-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the test workflow file
	workflowFile := filepath.Join(tmpDir, "backward-compat.md")
	if err := os.WriteFile(workflowFile, []byte(markdown), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	result := string(lockContent)

	// Should use default installation
	if !strings.Contains(result, "Install awf binary") {
		t.Error("Expected default AWF installation for workflow without path field")
	}

	// Should include version
	if !strings.Contains(result, "v1.2.3") {
		t.Error("Expected specified version in installation step")
	}

	// Should not have validation step
	if strings.Contains(result, "Validate custom AWF binary") {
		t.Error("Should not have validation step for workflow without custom path")
	}
}
