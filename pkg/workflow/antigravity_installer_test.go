package workflow

import (
	"strings"
	"testing"
)

func TestGenerateAntigravityInstallerSteps_Rootless(t *testing.T) {
	steps := GenerateAntigravityInstallerSteps("1.0.0", "Install Antigravity CLI", true)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join(steps[0], "\n")

	if !strings.Contains(stepContent, "--rootless") {
		t.Errorf("Expected step to contain --rootless flag, got:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "install_antigravity_cli.sh") {
		t.Errorf("Expected step to use install_antigravity_cli.sh, got:\n%s", stepContent)
	}
}

func TestGenerateAntigravityInstallerSteps_NoRootless(t *testing.T) {
	steps := GenerateAntigravityInstallerSteps("1.0.0", "Install Antigravity CLI", false)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join(steps[0], "\n")

	if strings.Contains(stepContent, "--rootless") {
		t.Errorf("Expected step to NOT contain --rootless flag, got:\n%s", stepContent)
	}
}
