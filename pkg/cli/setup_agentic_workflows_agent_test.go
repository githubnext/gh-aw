package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSetupAgenticWorkflowsAgent(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new setup agentic workflows agent file",
			existingContent: "",
			expectedContent: strings.TrimSpace(setupAgenticWorkflowsAgentTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: setupAgenticWorkflowsAgentTemplate,
			expectedContent: strings.TrimSpace(setupAgenticWorkflowsAgentTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified Setup\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(setupAgenticWorkflowsAgentTemplate),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			// Change to temp directory and initialize git repo for findGitRoot to work
			oldWd, _ := os.Getwd()
			defer func() {
				_ = os.Chdir(oldWd)
			}()
			err := os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Initialize git repo
			if err := exec.Command("git", "init").Run(); err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}

			agentsDir := filepath.Join(tempDir, ".github", "agents")
			agentPath := filepath.Join(agentsDir, "setup-agentic-workflows.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(agentsDir, 0755); err != nil {
					t.Fatalf("Failed to create agents directory: %v", err)
				}
				if err := os.WriteFile(agentPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial setup agent: %v", err)
				}
			}

			// Call the function with skipInstructions=false to test the functionality
			err = ensureSetupAgenticWorkflowsAgent(false, false)
			if err != nil {
				t.Fatalf("ensureSetupAgenticWorkflowsAgent() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(agentPath); os.IsNotExist(err) {
				t.Fatalf("Expected setup agentic workflows agent file to exist")
			}

			// Check content
			content, err := os.ReadFile(agentPath)
			if err != nil {
				t.Fatalf("Failed to read setup agent: %v", err)
			}

			contentStr := strings.TrimSpace(string(content))
			expectedStr := strings.TrimSpace(tt.expectedContent)

			if contentStr != expectedStr {
				t.Errorf("Expected content does not match.\nExpected first 100 chars: %q\nActual first 100 chars: %q",
					expectedStr[:min(100, len(expectedStr))],
					contentStr[:min(100, len(contentStr))])
			}
		})
	}
}

func TestEnsureSetupAgenticWorkflowsAgent_WithSkipInstructionsTrue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory and initialize git repo for findGitRoot to work
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	agentsDir := filepath.Join(tempDir, ".github", "agents")
	agentPath := filepath.Join(agentsDir, "setup-agentic-workflows.md")

	// Call the function with skipInstructions=true
	err = ensureSetupAgenticWorkflowsAgent(false, true)
	if err != nil {
		t.Fatalf("ensureSetupAgenticWorkflowsAgent() returned error: %v", err)
	}

	// Check that file does not exist
	if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
		t.Fatalf("Expected setup agent file to not exist when skipInstructions=true")
	}
}

func TestSetupAgenticWorkflowsAgentContainsRequiredSections(t *testing.T) {
	// Verify the template contains all required sections
	requiredSections := []string{
		"Configure Secrets for Your Chosen Agent",
		"copilot",
		"claude",
		"codex",
		"COPILOT_CLI_TOKEN",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"/create-agentic-workflow",
		"gh secret set",
	}

	content := strings.TrimSpace(setupAgenticWorkflowsAgentTemplate)

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("Template missing required section: %q", section)
		}
	}
}

func TestSetupAgenticWorkflowsAgentHasValidDocumentationLinks(t *testing.T) {
	// Verify the template contains documentation links
	requiredLinks := []string{
		"https://githubnext.github.io/gh-aw/reference/engines/",
		"https://github.com/settings/tokens",
	}

	content := strings.TrimSpace(setupAgenticWorkflowsAgentTemplate)

	for _, link := range requiredLinks {
		if !strings.Contains(content, link) {
			t.Errorf("Template missing required documentation link: %q", link)
		}
	}
}
