package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestGetAWFImageTag tests the getAWFImageTag helper function
func TestGetAWFImageTag(t *testing.T) {
	t.Run("returns default version when firewall config is nil", func(t *testing.T) {
		result := getAWFImageTag(nil)
		expected := string(constants.DefaultFirewallVersion)
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("returns default version when version is empty", func(t *testing.T) {
		config := &FirewallConfig{
			Enabled: true,
			Version: "",
		}
		result := getAWFImageTag(config)
		expected := string(constants.DefaultFirewallVersion)
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("returns custom version when specified", func(t *testing.T) {
		customVersion := "v0.5.0"
		config := &FirewallConfig{
			Enabled: true,
			Version: customVersion,
		}
		result := getAWFImageTag(config)
		if result != customVersion {
			t.Errorf("Expected %s, got %s", customVersion, result)
		}
	})
}

// TestClaudeEngineAWFImageTag tests that Claude engine includes --image-tag in AWF commands
func TestClaudeEngineAWFImageTag(t *testing.T) {
	t.Run("AWF command includes image-tag with default version", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --image-tag is included with default version
		expectedImageTag := "--image-tag " + string(constants.DefaultFirewallVersion)
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}
	})

	t.Run("AWF command includes image-tag with custom version", func(t *testing.T) {
		customVersion := "v0.5.0"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customVersion,
				},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --image-tag is included with custom version
		expectedImageTag := "--image-tag " + customVersion
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}
	})
}

// TestCodexEngineAWFImageTag tests that Codex engine includes --image-tag in AWF commands
func TestCodexEngineAWFImageTag(t *testing.T) {
	t.Run("AWF command includes image-tag with default version", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --image-tag is included with default version
		expectedImageTag := "--image-tag " + string(constants.DefaultFirewallVersion)
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}
	})

	t.Run("AWF command includes image-tag with custom version", func(t *testing.T) {
		customVersion := "v0.5.0"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customVersion,
				},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that --image-tag is included with custom version
		expectedImageTag := "--image-tag " + customVersion
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF command to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}
	})
}
