package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitRepository_WithProjectBoard(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call the function with project board flag
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with project board returned error: %v", err)
	}

	// Verify standard files were created
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist")
	}

	// Verify orchestrator workflow was created
	orchestratorPath := filepath.Join(tempDir, ".github", "workflows", "orchestrator.md")
	if _, err := os.Stat(orchestratorPath); os.IsNotExist(err) {
		t.Errorf("Expected orchestrator workflow to exist at %s", orchestratorPath)
	}

	// Verify issue templates were created
	issueTemplatesDir := filepath.Join(tempDir, ".github", "ISSUE_TEMPLATE")
	if _, err := os.Stat(issueTemplatesDir); os.IsNotExist(err) {
		t.Errorf("Expected ISSUE_TEMPLATE directory to exist")
	}

	researchTemplatePath := filepath.Join(issueTemplatesDir, "research.yml")
	if _, err := os.Stat(researchTemplatePath); os.IsNotExist(err) {
		t.Errorf("Expected research.yml issue template to exist")
	}

	analysisTemplatePath := filepath.Join(issueTemplatesDir, "analysis.yml")
	if _, err := os.Stat(analysisTemplatePath); os.IsNotExist(err) {
		t.Errorf("Expected analysis.yml issue template to exist")
	}
}

func TestInitRepository_ProjectBoard_Idempotent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Call the function first time with project board
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with project board returned error on first call: %v", err)
	}

	// Call the function second time with project board
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with project board returned error on second call: %v", err)
	}

	// Verify files still exist
	orchestratorPath := filepath.Join(tempDir, ".github", "workflows", "orchestrator.md")
	if _, err := os.Stat(orchestratorPath); os.IsNotExist(err) {
		t.Errorf("Expected orchestrator workflow to exist after second call")
	}

	issueTemplatesDir := filepath.Join(tempDir, ".github", "ISSUE_TEMPLATE")
	if _, err := os.Stat(issueTemplatesDir); os.IsNotExist(err) {
		t.Errorf("Expected ISSUE_TEMPLATE directory to exist after second call")
	}
}
