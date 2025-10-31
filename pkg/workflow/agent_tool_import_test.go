package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAgentFileToolImport tests that tools from custom agent files are imported and mapped correctly
func TestAgentFileToolImport(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "agent-tool-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents directory: %v", err)
	}
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	tests := []struct {
		name                string
		agentContent        string
		workflowContent     string
		expectEdit          bool
		expectGitHub        bool
		expectBash          bool
		expectWarning       bool
		expectedWarningText string
	}{
		{
			name: "agent with edit tools",
			agentContent: `---
name: test-agent
tools:
  - createFile
  - editFiles
---

# Test Agent

This agent creates and edits files.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# Test Workflow

Test workflow content.
`,
			expectEdit:   true,
			expectGitHub: true, // Default github tool always present
			expectBash:   false,
		},
		{
			name: "agent with github tools",
			agentContent: `---
name: search-agent
tools:
  - search
  - codeSearch
  - getFile
---

# Search Agent

This agent searches code.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# Search Workflow

Search workflow content.
`,
			expectEdit:   false,
			expectGitHub: true, // Always present
			expectBash:   false,
		},
		{
			name: "agent with bash tools",
			agentContent: `---
name: command-agent
tools:
  - runCommand
---

# Command Agent

This agent runs commands.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# Command Workflow

Command workflow content.
`,
			expectEdit:   false,
			expectGitHub: true, // Default github tool always present
			expectBash:   true,
		},
		{
			name: "agent with mixed tools",
			agentContent: `---
name: mixed-agent
tools:
  - createFile
  - search
  - runCommand
---

# Mixed Agent

This agent does everything.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# Mixed Workflow

Mixed workflow content.
`,
			expectEdit:   true,
			expectGitHub: true, // Always present
			expectBash:   true,
		},
		{
			name: "agent with unknown tools",
			agentContent: `---
name: unknown-agent
tools:
  - createFile
  - unknownTool
  - anotherUnknown
---

# Unknown Agent

This agent has unknown tools.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# Unknown Workflow

Unknown workflow content.
`,
			expectEdit:          true,
			expectGitHub:        true, // Default github tool always present
			expectBash:          false,
			expectWarning:       true,
			expectedWarningText: "unknownTool",
		},
		{
			name: "agent with no tools",
			agentContent: `---
name: no-tools-agent
---

# No Tools Agent

This agent has no tools specified.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
---

# No Tools Workflow

No tools workflow content.
`,
			expectEdit:   false,
			expectGitHub: true, // Default github tool always present
			expectBash:   false,
		},
		{
			name: "workflow tools override agent tools",
			agentContent: `---
name: override-agent
tools:
  - createFile
---

# Override Agent

Agent with createFile tool.
`,
			workflowContent: `---
on: issues
imports:
  - ../agents/test-agent.md
tools:
  web-fetch:
---

# Override Workflow

Workflow with its own tools.
`,
			expectEdit:   true, // From agent
			expectGitHub: true, // Default github tool always present
			expectBash:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create agent file with a consistent name
			agentFileName := "test-agent.md"
			agentFilePath := filepath.Join(agentsDir, agentFileName)
			if err := os.WriteFile(agentFilePath, []byte(tt.agentContent), 0644); err != nil {
				t.Fatalf("Failed to create agent file: %v", err)
			}

			// Create workflow file
			workflowFilePath := filepath.Join(workflowsDir, strings.ReplaceAll(tt.name, " ", "-")+".md")
			if err := os.WriteFile(workflowFilePath, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatalf("Failed to create workflow file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(true, "", "")
			workflowData, err := compiler.ParseWorkflowFile(workflowFilePath)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Check tools
			if tt.expectEdit {
				if workflowData.ParsedTools.Edit == nil {
					t.Error("Expected edit tool to be present")
				}
			} else {
				if workflowData.ParsedTools.Edit != nil {
					t.Errorf("Did not expect edit tool to be present, but it is: %+v", workflowData.ParsedTools.Edit)
				}
			}

			if tt.expectGitHub {
				if workflowData.ParsedTools.GitHub == nil {
					t.Error("Expected github tool to be present")
				}
			} else {
				if workflowData.ParsedTools.GitHub != nil {
					t.Errorf("Did not expect github tool to be present, but it is: %+v", workflowData.ParsedTools.GitHub)
				}
			}

			if tt.expectBash {
				if workflowData.ParsedTools.Bash == nil {
					t.Error("Expected bash tool to be present")
				}
			} else {
				if workflowData.ParsedTools.Bash != nil {
					t.Errorf("Did not expect bash tool to be present, but it is: %+v", workflowData.ParsedTools.Bash)
				}
			}

			// Check warnings for unknown tools
			if tt.expectWarning {
				if compiler.warningCount == 0 {
					t.Error("Expected warning for unknown tools, but got none")
				}
			}
		})
	}
}

// TestAgentToolsDoNotOverrideWorkflowTools tests that workflow-level tools take precedence
func TestAgentToolsDoNotOverrideWorkflowTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-tool-precedence-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentsDir := filepath.Join(tmpDir, ".github", "agents")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents directory: %v", err)
	}
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create agent with github tools
	agentContent := `---
name: github-agent
tools:
  - search
  - getFile
---

# GitHub Agent

This agent uses GitHub tools.
`
	agentFilePath := filepath.Join(agentsDir, "github-agent.md")
	if err := os.WriteFile(agentFilePath, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to create agent file: %v", err)
	}

	// Create workflow that specifies its own github tool configuration
	workflowContent := `---
on: issues
imports:
  - ../agents/github-agent.md
tools:
  github:
    allowed:
      - list_commits
      - get_repository
---

# Test Workflow

Workflow with custom github tool config.
`
	workflowFilePath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowFilePath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "")
	workflowData, err := compiler.ParseWorkflowFile(workflowFilePath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify that both agent and workflow tools are merged
	if workflowData.ParsedTools.GitHub == nil {
		t.Fatal("Expected github tool to be present")
	}

	// Check that the allowed list contains tools from both agent and workflow
	allowed := workflowData.ParsedTools.GitHub.Allowed
	if allowed == nil {
		t.Fatal("Expected github allowed list to be present")
	}

	// Should have tools from both sources
	hasSearchCode := false
	hasGetFile := false
	hasListCommits := false
	hasGetRepo := false

	for _, tool := range allowed {
		switch tool {
		case "search_code":
			hasSearchCode = true
		case "get_file_contents":
			hasGetFile = true
		case "list_commits":
			hasListCommits = true
		case "get_repository":
			hasGetRepo = true
		}
	}

	if !hasSearchCode {
		t.Error("Expected search_code from agent to be in allowed list")
	}
	if !hasGetFile {
		t.Error("Expected get_file_contents from agent to be in allowed list")
	}
	if !hasListCommits {
		t.Error("Expected list_commits from workflow to be in allowed list")
	}
	if !hasGetRepo {
		t.Error("Expected get_repository from workflow to be in allowed list")
	}
}
