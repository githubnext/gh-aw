//go:build !integration

package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunListDomains_NoWorkflows(t *testing.T) {
	// Change to a temp directory with no workflows
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	// Create the .github/workflows directory but with no markdown files
	err = os.MkdirAll(".github/workflows", 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	t.Run("no workflows text output", func(t *testing.T) {
		err := RunListDomains(false)
		assert.NoError(t, err, "RunListDomains should not error with no workflows")
	})

	t.Run("no workflows JSON output", func(t *testing.T) {
		err := RunListDomains(true)
		assert.NoError(t, err, "RunListDomains should not error with no workflows in JSON mode")
	})
}

func TestRunListDomains_WithWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.MkdirAll(".github/workflows", 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Write a workflow file with network config
	workflowContent := `---
engine: copilot
network:
  allowed:
    - github
    - node
---
# Test Workflow
Do something.
`
	workflowPath := filepath.Join(".github", "workflows", "test-workflow.md")
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0600)
	require.NoError(t, err, "Failed to write workflow file")

	t.Run("text output", func(t *testing.T) {
		err := RunListDomains(false)
		assert.NoError(t, err, "RunListDomains should not error")
	})

	t.Run("JSON output", func(t *testing.T) {
		// Capture stdout by redirecting
		err := RunListDomains(true)
		assert.NoError(t, err, "RunListDomains JSON should not error")
	})
}

func TestRunWorkflowDomains_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.MkdirAll(".github/workflows", 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	workflowContent := `---
engine: copilot
network:
  allowed:
    - github
  blocked:
    - malicious.example.com
---
# Test
`
	workflowPath := filepath.Join(".github", "workflows", "my-workflow.md")
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0600)
	require.NoError(t, err, "Failed to write workflow file")

	err = RunWorkflowDomains("my-workflow", true)
	assert.NoError(t, err, "RunWorkflowDomains JSON should not error")
}

func TestRunWorkflowDomains_TextOutput(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.MkdirAll(".github/workflows", 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	workflowContent := `---
engine: copilot
network:
  allowed:
    - github
---
# Test
`
	workflowPath := filepath.Join(".github", "workflows", "my-workflow.md")
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0600)
	require.NoError(t, err, "Failed to write workflow file")

	err = RunWorkflowDomains("my-workflow", false)
	assert.NoError(t, err, "RunWorkflowDomains text should not error")
}

func TestWorkflowDomainsDetail_JSONMarshaling(t *testing.T) {
	detail := WorkflowDomainsDetail{
		Workflow:       "my-workflow",
		Engine:         "copilot",
		AllowedDomains: []string{"api.github.com", "github.com"},
		BlockedDomains: []string{"malicious.example.com"},
	}

	jsonBytes, err := json.MarshalIndent(detail, "", "  ")
	require.NoError(t, err, "Should marshal WorkflowDomainsDetail to JSON")

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"workflow": "my-workflow"`, "JSON should contain workflow name")
	assert.Contains(t, jsonStr, `"engine": "copilot"`, "JSON should contain engine")
	assert.Contains(t, jsonStr, `"allowed_domains"`, "JSON should contain allowed_domains key")
	assert.Contains(t, jsonStr, `"blocked_domains"`, "JSON should contain blocked_domains key")
	assert.Contains(t, jsonStr, `"api.github.com"`, "JSON should contain allowed domain")
	assert.Contains(t, jsonStr, `"malicious.example.com"`, "JSON should contain blocked domain")
}

func TestWorkflowDomainsSummary_JSONMarshaling(t *testing.T) {
	summary := WorkflowDomainsSummary{
		Workflow: "my-workflow",
		Engine:   "copilot",
		Allowed:  10,
		Blocked:  2,
	}

	jsonBytes, err := json.MarshalIndent(summary, "", "  ")
	require.NoError(t, err, "Should marshal WorkflowDomainsSummary to JSON")

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"workflow": "my-workflow"`, "JSON should contain workflow name")
	assert.Contains(t, jsonStr, `"engine": "copilot"`, "JSON should contain engine")
	assert.Contains(t, jsonStr, `"allowed": 10`, "JSON should contain allowed count")
	assert.Contains(t, jsonStr, `"blocked": 2`, "JSON should contain blocked count")
}

func TestBuildDomainItems(t *testing.T) {
	allowed := []string{"api.github.com", "github.com"}
	blocked := []string{"malicious.example.com"}

	items := buildDomainItems(allowed, blocked)
	require.Len(t, items, 3, "Should have 3 items total")

	// First two should be allowed
	assert.Equal(t, "api.github.com", items[0].Domain, "First item should be api.github.com")
	assert.Contains(t, items[0].Status, "Allowed", "First item should be allowed")

	assert.Equal(t, "github.com", items[1].Domain, "Second item should be github.com")
	assert.Contains(t, items[1].Status, "Allowed", "Second item should be allowed")

	// Last one should be blocked
	assert.Equal(t, "malicious.example.com", items[2].Domain, "Third item should be malicious.example.com")
	assert.Contains(t, items[2].Status, "Blocked", "Third item should be blocked")
}

func TestBuildDomainItems_EcosystemAnnotation(t *testing.T) {
	allowed := []string{"registry.npmjs.org", "pypi.org"}
	items := buildDomainItems(allowed, nil)

	require.Len(t, items, 2, "Should have 2 items")

	// registry.npmjs.org should be in the node ecosystem
	assert.Equal(t, "registry.npmjs.org", items[0].Domain, "First domain should be registry.npmjs.org")
	assert.Equal(t, "node", items[0].Ecosystem, "registry.npmjs.org should be in node ecosystem")

	// pypi.org should be in the python ecosystem
	assert.Equal(t, "pypi.org", items[1].Domain, "Second domain should be pypi.org")
	assert.Equal(t, "python", items[1].Ecosystem, "pypi.org should be in python ecosystem")
}

func TestNewDomainsCommand(t *testing.T) {
	cmd := NewDomainsCommand()
	assert.NotNil(t, cmd, "NewDomainsCommand should return a command")
	assert.Equal(t, "domains [workflow]", cmd.Use, "Command use should be 'domains [workflow]'")
	assert.NotEmpty(t, cmd.Short, "Command should have a short description")
	assert.NotEmpty(t, cmd.Long, "Command should have a long description")

	// Check --json flag exists
	jsonFlag := cmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "Command should have --json flag")

	// Check max 1 argument
	err := cmd.Args(cmd, []string{"a", "b"})
	require.Error(t, err, "Command should reject more than 1 argument")

	err = cmd.Args(cmd, []string{})
	require.NoError(t, err, "Command should accept 0 arguments")

	err = cmd.Args(cmd, []string{"workflow-name"})
	require.NoError(t, err, "Command should accept 1 argument")
}

func TestExtractWorkflowDomainConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("workflow with network config", func(t *testing.T) {
		content := `---
engine: claude
network:
  allowed:
    - github
    - python
  blocked:
    - bad.example.com
---
# Test
`
		path := filepath.Join(tmpDir, "test.md")
		err := os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err, "Failed to write test file")

		engineID, network, _, _ := extractWorkflowDomainConfig(path)
		assert.Equal(t, "claude", engineID, "Engine should be claude")
		require.NotNil(t, network, "Network should not be nil")
		assert.Equal(t, []string{"github", "python"}, network.Allowed, "Allowed should match")
		assert.Equal(t, []string{"bad.example.com"}, network.Blocked, "Blocked should match")
	})

	t.Run("workflow without network config defaults to copilot", func(t *testing.T) {
		content := `---
engine: copilot
---
# Test
`
		path := filepath.Join(tmpDir, "no-network.md")
		err := os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err, "Failed to write test file")

		engineID, network, _, _ := extractWorkflowDomainConfig(path)
		assert.Equal(t, "copilot", engineID, "Engine should be copilot")
		assert.Nil(t, network, "Network should be nil when not configured")
	})

	t.Run("nonexistent file defaults to copilot", func(t *testing.T) {
		engineID, network, _, _ := extractWorkflowDomainConfig("/nonexistent/file.md")
		assert.Equal(t, "copilot", engineID, "Engine should default to copilot")
		assert.Nil(t, network, "Network should be nil for nonexistent file")
	})
}

func TestComputeAllowedDomains(t *testing.T) {
	t.Run("copilot engine without network config", func(t *testing.T) {
		domains := computeAllowedDomains("copilot", nil, nil, nil)
		// Copilot has default domains
		assert.NotEmpty(t, domains, "Should have default Copilot domains")
		assert.True(t, slices.Contains(domains, "api.github.com"), "Should contain api.github.com")
	})

	t.Run("returns empty for unknown engine with empty network", func(t *testing.T) {
		network := &workflow.NetworkPermissions{
			Allowed: []string{},
		}
		domains := computeAllowedDomains("custom", network, nil, nil)
		assert.Empty(t, domains, "Should return empty for explicit empty allowed list")
	})
}

func TestRunListDomains_RepoRoot(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")

	repoRoot := filepath.Join(originalDir, "..", "..")
	err = os.Chdir(repoRoot)
	require.NoError(t, err, "Failed to change to repository root")
	defer os.Chdir(originalDir) //nolint:errcheck

	t.Run("JSON output from repo root", func(t *testing.T) {
		err := RunListDomains(true)
		assert.NoError(t, err, "RunListDomains JSON should not error from repo root")
	})

	t.Run("text output from repo root", func(t *testing.T) {
		err := RunListDomains(false)
		assert.NoError(t, err, "RunListDomains text should not error from repo root")
	})
}

func TestRunListDomains_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.MkdirAll(".github/workflows", 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Write two workflow files
	for _, wf := range []struct{ name, engine string }{
		{"wf-a", "copilot"},
		{"wf-b", "claude"},
	} {
		content := "---\nengine: " + wf.engine + "\n---\n# Test\n"
		path := filepath.Join(".github", "workflows", wf.name+".md")
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err, "Failed to write workflow file")
	}

	// Capture JSON by redirecting stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "Failed to create pipe")
	os.Stdout = w

	err = RunListDomains(true)
	require.NoError(t, err, "RunListDomains should not error")

	w.Close()
	os.Stdout = oldStdout

	outputBytes, err := io.ReadAll(r)
	require.NoError(t, err, "Failed to read pipe output")
	r.Close()

	output := string(outputBytes)
	assert.NotEmpty(t, output, "JSON output should not be empty")

	var summaries []WorkflowDomainsSummary
	err = json.Unmarshal(outputBytes, &summaries)
	require.NoError(t, err, "JSON output should be valid JSON array")
	assert.Len(t, summaries, 2, "Should have 2 workflow summaries")
}
