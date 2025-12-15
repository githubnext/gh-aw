package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillDetectionWithCopilot(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create a SKILL.md file
	skillFile := filepath.Join(tempDir, "test-skill", "SKILL.md")
	os.MkdirAll(filepath.Dir(skillFile), 0755)
	skillContent := `---
name: test-skill
description: A test skill for validation
---

# Test Skill

This is a test skill.
`
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	// Create a workflow that imports the skill file
	workflowContent := `---
engine: copilot
on:
  issues:
    types: [opened]
imports:
  - test-skill/SKILL.md
---

# Test Workflow

This workflow imports a skill file.
`
	workflowFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Compile the workflow
	c := NewCompiler(false, "", "test")

	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := c.CompileWorkflow(workflowFile)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check that warning was printed for copilot engine
	if !strings.Contains(output, "SKILL files detected") {
		t.Errorf("Expected warning about SKILL files for copilot engine, got: %s", output)
	}
	if !strings.Contains(output, "does not support Agent Skills natively") {
		t.Errorf("Expected warning message about lack of native support, got: %s", output)
	}
}

func TestSkillDetectionWithClaude(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create a SKILL.md file
	skillFile := filepath.Join(tempDir, "test-skill", "SKILL.md")
	os.MkdirAll(filepath.Dir(skillFile), 0755)
	skillContent := `---
name: test-skill
description: A test skill for validation
---

# Test Skill

This is a test skill.
`
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	// Create a workflow that imports the skill file
	workflowContent := `---
engine: claude
on:
  issues:
    types: [opened]
imports:
  - test-skill/SKILL.md
---

# Test Workflow

This workflow imports a skill file.
`
	workflowFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Compile the workflow
	c := NewCompiler(false, "", "test")

	// Capture stderr to check for NO warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := c.CompileWorkflow(workflowFile)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check that NO warning was printed for claude engine
	if strings.Contains(output, "does not support Agent Skills natively") {
		t.Errorf("Unexpected warning for claude engine which supports skills: %s", output)
	}
}

func TestSkillDetectionMultipleFiles(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create multiple SKILL.md files
	skill1File := filepath.Join(tempDir, "skill1", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill1File), 0755)
	skill1Content := `---
name: skill-one
description: First test skill
---

# Skill One
`
	if err := os.WriteFile(skill1File, []byte(skill1Content), 0644); err != nil {
		t.Fatalf("Failed to create skill1 file: %v", err)
	}

	skill2File := filepath.Join(tempDir, "skill2", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill2File), 0755)
	skill2Content := `---
name: skill-two
description: Second test skill
---

# Skill Two
`
	if err := os.WriteFile(skill2File, []byte(skill2Content), 0644); err != nil {
		t.Fatalf("Failed to create skill2 file: %v", err)
	}

	// Create a workflow that imports both skill files
	workflowContent := `---
engine: copilot
on:
  issues:
    types: [opened]
imports:
  - skill1/SKILL.md
  - skill2/SKILL.md
---

# Test Workflow

This workflow imports multiple skill files.
`
	workflowFile := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Compile the workflow
	c := NewCompiler(false, "", "test")

	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := c.CompileWorkflow(workflowFile)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check that warning mentions both files
	if !strings.Contains(output, "skill1/SKILL.md") {
		t.Errorf("Expected warning to mention skill1/SKILL.md, got: %s", output)
	}
	if !strings.Contains(output, "skill2/SKILL.md") {
		t.Errorf("Expected warning to mention skill2/SKILL.md, got: %s", output)
	}
}
