//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestFirewallArgsInCopilotEngine tests that custom firewall args are included in AWF command
func TestFirewallArgsInCopilotEngine(t *testing.T) {
	t.Run("no custom args uses only default flags", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that the command contains awf (AWF v0.15.0+ uses chroot mode by default)
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected command to contain 'sudo -E awf'")
		}

		// With config file support (default AWF version), domains appear in the JSON config
		// rather than as a --allow-domains CLI flag. Verify the config JSON is written.
		if !strings.Contains(stepContent, "allowDomains") {
			t.Error("Expected command to contain 'allowDomains' in the AWF config JSON")
		}

		if !strings.Contains(stepContent, "--log-level") {
			t.Error("Expected command to contain '--log-level'")
		}

		initSnippet := `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS=""`
		conditionSnippet := `if [[ "${DOCKER_HOST:-}" =~ ^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then`
		flagAssignmentSnippet := `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS="--docker-host-path-prefix /tmp/gh-aw"`
		argsRefSnippet := `${GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS}`

		initIdx := strings.Index(stepContent, initSnippet)
		conditionIdx := strings.Index(stepContent, conditionSnippet)
		flagIdx := strings.Index(stepContent, flagAssignmentSnippet)
		argsRefIdx := strings.Index(stepContent, argsRefSnippet)
		if initIdx == -1 || conditionIdx == -1 || flagIdx == -1 || argsRefIdx == -1 || !(initIdx < conditionIdx && conditionIdx < flagIdx && flagIdx < argsRefIdx) {
			t.Error("Expected command to initialize probe variable, evaluate DOCKER_HOST condition, assign docker-host-path-prefix flag, then expand args variable in AWF invocation")
		}

		// Verify that --log-dir is included in copilot args for log collection
		if !strings.Contains(stepContent, "--log-dir /tmp/gh-aw/sandbox/agent/logs/") {
			t.Error("Expected copilot command to contain '--log-dir /tmp/gh-aw/sandbox/agent/logs/' for log collection in firewall mode")
		}
	})

	t.Run("custom args are included in AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Args:    []string{"--custom-arg", "value", "--another-flag"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that custom args are included
		if !strings.Contains(stepContent, "--custom-arg") {
			t.Error("Expected command to contain custom arg '--custom-arg'")
		}

		if !strings.Contains(stepContent, "value") {
			t.Error("Expected command to contain custom arg value 'value'")
		}

		if !strings.Contains(stepContent, "--another-flag") {
			t.Error("Expected command to contain custom arg '--another-flag'")
		}

		// With config file support, domains appear in the JSON config (not as --allow-domains)
		if !strings.Contains(stepContent, "allowDomains") {
			t.Error("Expected command to still contain 'allowDomains' in the AWF config JSON")
		}
	})

	t.Run("custom args with spaces are properly escaped", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Args:    []string{"--message", "hello world", "--path", "/some/path with spaces"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that args with spaces are present (they should be escaped)
		if !strings.Contains(stepContent, "--message") {
			t.Error("Expected command to contain '--message' flag")
		}

		// The value might be escaped, so just check the flag exists
		if !strings.Contains(stepContent, "--path") {
			t.Error("Expected command to contain '--path' flag")
		}
	})

	t.Run("AWF uses chroot mode instead of individual binary mounts", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that AWF is used for transparent host access (AWF v0.15.0+)
		// Chroot mode is now the default, so no --enable-chroot flag is needed
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected AWF command for transparent host access")
		}

		// Verify that individual binary mounts are not used (chroot mode is default)
		if strings.Contains(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro") {
			t.Error("Individual binary mounts should not be present with default chroot mode")
		}
	})

	t.Run("AWF command includes image-tag with default version", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// With config file support (default AWF version), the image tag is expressed in the
		// JSON config file rather than as a --image-tag CLI flag.
		// Verify the image tag version appears in the config JSON.
		expectedVersion := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		if !strings.Contains(stepContent, expectedVersion) {
			t.Errorf("Expected AWF config JSON to contain image tag version '%s'", expectedVersion)
		}
		// imageTag field name must be present
		if !strings.Contains(stepContent, "imageTag") {
			t.Error("Expected AWF config JSON to contain 'imageTag' field")
		}
	})

	t.Run("AWF command includes image-tag with custom version", func(t *testing.T) {
		customVersion := "v0.5.0"
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customVersion,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Image tag is now always written to the JSON config file, never as a CLI flag.
		expectedImageTag := `"imageTag":"` + strings.TrimPrefix(customVersion, "v")
		if !strings.Contains(stepContent, expectedImageTag) {
			t.Errorf("Expected AWF config JSON to contain '%s', got:\n%s", expectedImageTag, stepContent)
		}

		// --image-tag must NOT appear as a CLI flag
		if strings.Contains(stepContent, "--image-tag") {
			t.Error("--image-tag should not appear as a CLI flag; it is in the config JSON")
		}
	})

	t.Run("skips docker-host-path-prefix probe and arg ref when AWF version is too old", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: "v0.25.42",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")
		stepContent := requireCopilotExecutionStep(t, steps)

		if strings.Contains(stepContent, `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS=""`) {
			t.Error("Expected command to skip docker-host-path-prefix probe variable initialization for unsupported AWF versions")
		}
		if strings.Contains(stepContent, `if [[ "${DOCKER_HOST:-}" =~ ^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$ ]]; then`) {
			t.Error("Expected command to skip docker-host-path-prefix DOCKER_HOST probe for unsupported AWF versions")
		}
		if strings.Contains(stepContent, `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS="--docker-host-path-prefix /tmp/gh-aw"`) {
			t.Error("Expected command to skip docker-host-path-prefix assignment for unsupported AWF versions")
		}
		if strings.Contains(stepContent, `${GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS}`) {
			t.Error("Expected command to skip docker-host-path-prefix args variable expansion for unsupported AWF versions")
		}
	})

	t.Run("AWF command includes ssl-bump flag when enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					SSLBump: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that --ssl-bump flag is included
		if !strings.Contains(stepContent, "--ssl-bump") {
			t.Error("Expected AWF command to contain '--ssl-bump' flag")
		}
	})

	t.Run("AWF command includes allow-urls with ssl-bump enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled:   true,
					SSLBump:   true,
					AllowURLs: []string{"https://github.com/githubnext/*", "https://api.github.com/repos/*"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that --ssl-bump flag is included
		if !strings.Contains(stepContent, "--ssl-bump") {
			t.Error("Expected AWF command to contain '--ssl-bump' flag")
		}

		// Check that --allow-urls is included with the comma-separated URLs
		if !strings.Contains(stepContent, "--allow-urls") {
			t.Error("Expected AWF command to contain '--allow-urls' flag")
		}

		if !strings.Contains(stepContent, "https://github.com/githubnext/*") {
			t.Error("Expected AWF command to contain URL pattern 'https://github.com/githubnext/*'")
		}
	})

	t.Run("AWF command does not include allow-urls without ssl-bump", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled:   true,
					SSLBump:   false, // SSL Bump disabled
					AllowURLs: []string{"https://github.com/githubnext/*"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that --ssl-bump flag is NOT included
		if strings.Contains(stepContent, "--ssl-bump") {
			t.Error("Expected AWF command to NOT contain '--ssl-bump' flag when SSLBump is false")
		}

		// Check that --allow-urls is NOT included when ssl-bump is disabled
		if strings.Contains(stepContent, "--allow-urls") {
			t.Error("Expected AWF command to NOT contain '--allow-urls' flag when SSLBump is false")
		}
	})
}
