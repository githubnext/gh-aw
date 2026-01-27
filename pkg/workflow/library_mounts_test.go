package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLibraryMounts(t *testing.T) {
	t.Run("returns library directory mounts", func(t *testing.T) {
		mounts := GetLibraryMounts()

		require.NotEmpty(t, mounts, "Should return library mounts")
		assert.Contains(t, mounts, "--mount", "Should include --mount flag")
		assert.Contains(t, mounts, "/usr/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:ro",
			"Should include x86_64-linux-gnu library directory mount")
	})

	t.Run("mounts are read-only", func(t *testing.T) {
		mounts := GetLibraryMounts()

		for i, m := range mounts {
			if m == "--mount" {
				continue
			}
			// Every non-flag value should end with :ro
			if i > 0 && mounts[i-1] == "--mount" {
				assert.True(t, strings.HasSuffix(m, ":ro"),
					"Mount should be read-only: %s", m)
			}
		}
	})

	t.Run("mounts are in correct format", func(t *testing.T) {
		mounts := GetLibraryMounts()

		// Should have pairs of --mount and path
		assert.Equal(t, 0, len(mounts)%2, "Should have even number of mount args")

		for i := 0; i < len(mounts); i += 2 {
			assert.Equal(t, "--mount", mounts[i], "Even indices should be --mount flags")
			assert.Contains(t, mounts[i+1], ":", "Mount path should contain source:dest format")
		}
	})
}

func TestHasMountedBinaries(t *testing.T) {
	t.Run("returns true when binaries list is populated", func(t *testing.T) {
		assert.True(t, HasMountedBinaries(), "Should return true since MountedBinaries is defined")
	})
}

func TestMountedBinaries(t *testing.T) {
	t.Run("contains expected binaries", func(t *testing.T) {
		assert.Contains(t, MountedBinaries, "/usr/bin/date", "Should include date binary")
		assert.Contains(t, MountedBinaries, "/usr/bin/gh", "Should include gh binary")
		assert.Contains(t, MountedBinaries, "/usr/bin/yq", "Should include yq binary")
	})
}

func TestLibraryDirectories(t *testing.T) {
	t.Run("contains x86_64-linux-gnu directory", func(t *testing.T) {
		assert.Contains(t, LibraryDirectories, "/usr/lib/x86_64-linux-gnu",
			"Should include x86_64-linux-gnu library directory")
	})
}

func TestLibraryMountsInCopilotEngine(t *testing.T) {
	t.Run("library mounts are included when firewall is enabled", func(t *testing.T) {
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

		require.NotEmpty(t, steps, "Should return execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// Verify library mounts are present
		assert.Contains(t, stepContent, "--mount /usr/lib/x86_64-linux-gnu:/usr/lib/x86_64-linux-gnu:ro",
			"Should include library directory mount for shared library support")
	})

	t.Run("library mounts appear after binary mounts", func(t *testing.T) {
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

		require.NotEmpty(t, steps, "Should return execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// Find positions of mounts in the command
		ghMountPos := strings.Index(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro")
		libMountPos := strings.Index(stepContent, "--mount /usr/lib/x86_64-linux-gnu")

		assert.Greater(t, ghMountPos, -1, "gh mount should be present")
		assert.Greater(t, libMountPos, -1, "library mount should be present")
		assert.Greater(t, libMountPos, ghMountPos, "library mount should appear after binary mounts")
	})

	t.Run("library mounts are NOT included when firewall is disabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		require.NotEmpty(t, steps, "Should return execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// Verify library mounts are NOT present (no AWF)
		assert.NotContains(t, stepContent, "/usr/lib/x86_64-linux-gnu",
			"Should not include library mounts when firewall is disabled")
	})
}
