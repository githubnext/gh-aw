package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGettingStartedPrompt(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedContent string
	}{
		{
			name:            "creates new getting started prompt file",
			existingContent: "",
			expectedContent: strings.TrimSpace(gettingStartedPromptTemplate),
		},
		{
			name:            "does not modify existing correct file",
			existingContent: gettingStartedPromptTemplate,
			expectedContent: strings.TrimSpace(gettingStartedPromptTemplate),
		},
		{
			name:            "updates modified file",
			existingContent: "# Modified Getting Started\n\nThis is a modified version.",
			expectedContent: strings.TrimSpace(gettingStartedPromptTemplate),
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

			promptsDir := filepath.Join(tempDir, ".github", "prompts")
			promptPath := filepath.Join(promptsDir, "aw-setup.prompt.md")

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(promptsDir, 0755); err != nil {
					t.Fatalf("Failed to create prompts directory: %v", err)
				}
				if err := os.WriteFile(promptPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial getting started prompt: %v", err)
				}
			}

			// Call the function with skipInstructions=false to test the functionality
			err = ensureGettingStartedPrompt(false, false)
			if err != nil {
				t.Fatalf("ensureGettingStartedPrompt() returned error: %v", err)
			}

			// Check that file exists
			if _, err := os.Stat(promptPath); os.IsNotExist(err) {
				t.Fatalf("Expected getting started prompt file to exist")
			}

			// Check content
			content, err := os.ReadFile(promptPath)
			if err != nil {
				t.Fatalf("Failed to read getting started prompt: %v", err)
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

func TestEnsureGettingStartedPrompt_WithSkipInstructionsTrue(t *testing.T) {
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

	promptsDir := filepath.Join(tempDir, ".github", "prompts")
	promptPath := filepath.Join(promptsDir, "aw-setup.prompt.md")

	// Call the function with skipInstructions=true
	err = ensureGettingStartedPrompt(false, true)
	if err != nil {
		t.Fatalf("ensureGettingStartedPrompt() returned error: %v", err)
	}

	// Check that file does not exist
	if _, err := os.Stat(promptPath); !os.IsNotExist(err) {
		t.Fatalf("Expected getting started prompt file to not exist when skipInstructions=true")
	}
}

func TestGettingStartedPromptContainsRequiredSections(t *testing.T) {
	// Verify the template contains all required sections
	requiredSections := []string{
		"Configure Your Agentic Workflow",
		"copilot",
		"claude",
		"codex",
		"COPILOT_CLI_TOKEN",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"/create-agentic-workflow",
		"gh secret set",
	}

	content := strings.TrimSpace(gettingStartedPromptTemplate)

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("Template missing required section: %q", section)
		}
	}
}

func TestGettingStartedPromptHasValidDocumentationLinks(t *testing.T) {
	// Verify the template contains documentation links
	requiredLinks := []string{
		"https://githubnext.github.io/gh-aw/reference/engines/",
		"https://githubnext.github.io/gh-aw/start-here/quick-start/",
		"https://github.com/settings/tokens",
	}

	content := strings.TrimSpace(gettingStartedPromptTemplate)

	for _, link := range requiredLinks {
		if !strings.Contains(content, link) {
			t.Errorf("Template missing required documentation link: %q", link)
		}
	}
}
