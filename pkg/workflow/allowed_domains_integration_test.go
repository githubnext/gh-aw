package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllowedDomainsEnvironmentVariable(t *testing.T) {
	// Expected GitHub sanitization domains (now in defaults ecosystem)
	expectedGitHubDomains := []string{
		"github.com",
		"github.io",
		"githubusercontent.com",
		"githubassets.com",
		"github.dev",
		"codespaces.new",
	}

	t.Run("GH_AW_ALLOWED_DOMAINS is set with default domains", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()

		workflowContent := `---
on: issues
engine: copilot
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow content.
`

		// Write workflow file
		markdownPath := filepath.Join(tmpDir, "test-workflow.md")
		if err := os.WriteFile(markdownPath, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile workflow
		compiler := NewCompiler(false, "", "test")
		if err := compiler.CompileWorkflow(markdownPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read compiled YAML
		lockPath := filepath.Join(tmpDir, "test-workflow.lock.yml")
		yamlContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}

		yamlStr := string(yamlContent)

		// Check that GH_AW_ALLOWED_DOMAINS is set
		if !strings.Contains(yamlStr, "GH_AW_ALLOWED_DOMAINS:") {
			t.Error("Expected GH_AW_ALLOWED_DOMAINS to be set in compiled workflow")
		}

		// Check that default GitHub domains are included
		for _, domain := range expectedGitHubDomains {
			if !strings.Contains(yamlStr, domain) {
				t.Errorf("Expected domain '%s' to be in GH_AW_ALLOWED_DOMAINS", domain)
			}
		}
	})

	t.Run("GH_AW_ALLOWED_DOMAINS includes network custom domains", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()

		workflowContent := `---
on: issues
engine: copilot
network:
  allowed:
    - example.com
    - trusted.org
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow content.
`

		// Write workflow file
		markdownPath := filepath.Join(tmpDir, "test-workflow.md")
		if err := os.WriteFile(markdownPath, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile workflow
		compiler := NewCompiler(false, "", "test")
		if err := compiler.CompileWorkflow(markdownPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read compiled YAML
		lockPath := filepath.Join(tmpDir, "test-workflow.lock.yml")
		yamlContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}

		yamlStr := string(yamlContent)

		// Check that GH_AW_ALLOWED_DOMAINS is set
		if !strings.Contains(yamlStr, "GH_AW_ALLOWED_DOMAINS:") {
			t.Error("Expected GH_AW_ALLOWED_DOMAINS to be set in compiled workflow")
		}

		// Check that default GitHub domains are included
		expectedDomains := []string{
			"github.com",
			"example.com",
			"trusted.org",
		}

		for _, domain := range expectedDomains {
			if !strings.Contains(yamlStr, domain) {
				t.Errorf("Expected domain '%s' to be in GH_AW_ALLOWED_DOMAINS", domain)
			}
		}
	})

	t.Run("GH_AW_ALLOWED_DOMAINS includes ecosystem domains", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()

		workflowContent := `---
on: issues
engine: copilot
network:
  allowed:
    - defaults
    - python
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow content.
`

		// Write workflow file
		markdownPath := filepath.Join(tmpDir, "test-workflow.md")
		if err := os.WriteFile(markdownPath, []byte(workflowContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile workflow
		compiler := NewCompiler(false, "", "test")
		if err := compiler.CompileWorkflow(markdownPath); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read compiled YAML
		lockPath := filepath.Join(tmpDir, "test-workflow.lock.yml")
		yamlContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}

		yamlStr := string(yamlContent)

		// Check that GH_AW_ALLOWED_DOMAINS is set
		if !strings.Contains(yamlStr, "GH_AW_ALLOWED_DOMAINS:") {
			t.Error("Expected GH_AW_ALLOWED_DOMAINS to be set in compiled workflow")
		}

		// Check that default GitHub domains are included
		defaultDomains := []string{
			"github.com",
			"github.io",
		}

		for _, domain := range defaultDomains {
			if !strings.Contains(yamlStr, domain) {
				t.Errorf("Expected default domain '%s' to be in GH_AW_ALLOWED_DOMAINS", domain)
			}
		}

		// Check that ecosystem defaults are included
		ecosystemDefaultDomains := getEcosystemDomains("defaults")
		if len(ecosystemDefaultDomains) > 0 {
			// Check for at least one ecosystem default domain
			if !strings.Contains(yamlStr, ecosystemDefaultDomains[0]) {
				t.Errorf("Expected ecosystem default domain '%s' to be in GH_AW_ALLOWED_DOMAINS", ecosystemDefaultDomains[0])
			}
		}

		// Check that python ecosystem domains are included
		pythonDomains := getEcosystemDomains("python")
		if len(pythonDomains) > 0 {
			// Check for at least one python domain
			if !strings.Contains(yamlStr, pythonDomains[0]) {
				t.Errorf("Expected python ecosystem domain '%s' to be in GH_AW_ALLOWED_DOMAINS", pythonDomains[0])
			}
		}
	})
}
