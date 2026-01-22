package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuntimeMountsIntegration tests that runtime mounts are automatically
// contributed to the sandbox agent configuration during workflow compilation
func TestRuntimeMountsIntegration(t *testing.T) {
	tests := []struct {
		name             string
		workflowData     *WorkflowData
		expectedMounts   []string
		unexpectedMounts []string
	}{
		{
			name: "node runtime adds toolcache and npm cache mounts",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				CustomSteps: `      - name: Install deps
        run: npm install`,
			},
			expectedMounts: []string{
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
			},
		},
		{
			name: "python runtime adds toolcache and pip cache mounts",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				CustomSteps: `      - name: Run script
        run: python script.py`,
			},
			expectedMounts: []string{
				"/opt/hostedtoolcache/Python:/opt/hostedtoolcache/Python:ro",
				"/home/runner/.cache/pip:/home/runner/.cache/pip:rw",
			},
		},
		{
			name: "multiple runtimes add combined mounts",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				CustomSteps: `      - name: Install deps
        run: npm install
      - name: Run script
        run: python script.py`,
			},
			expectedMounts: []string{
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
				"/opt/hostedtoolcache/Python:/opt/hostedtoolcache/Python:ro",
				"/home/runner/.cache/pip:/home/runner/.cache/pip:rw",
			},
		},
		{
			name: "user-specified mounts are preserved",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
						Mounts: []string{
							"/custom/path:/custom/path:ro",
						},
					},
				},
				CustomSteps: `      - name: Install deps
        run: npm install`,
			},
			expectedMounts: []string{
				"/custom/path:/custom/path:ro",
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
			},
		},
		{
			name: "no runtime commands means no runtime mounts",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				CustomSteps: `      - name: Echo
        run: echo "hello"`,
			},
			expectedMounts:   []string{},
			unexpectedMounts: []string{"/opt/hostedtoolcache"},
		},
		{
			name: "go runtime adds toolcache, GOPATH, and build cache",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
				CustomSteps: `      - name: Build
        run: go build main.go`,
			},
			expectedMounts: []string{
				"/opt/hostedtoolcache/go:/opt/hostedtoolcache/go:ro",
				"/home/runner/go:/home/runner/go:rw",
				"/home/runner/.cache/go-build:/home/runner/.cache/go-build:rw",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Detect runtime requirements
			runtimeRequirements := DetectRuntimeRequirements(tt.workflowData)

			// Contribute runtime mounts to sandbox agent configuration
			if tt.workflowData.SandboxConfig != nil && tt.workflowData.SandboxConfig.Agent != nil {
				ContributeRuntimeMounts(tt.workflowData.SandboxConfig.Agent, runtimeRequirements)
			}

			// Verify expected mounts are present
			if tt.workflowData.SandboxConfig != nil && tt.workflowData.SandboxConfig.Agent != nil {
				agentMounts := tt.workflowData.SandboxConfig.Agent.Mounts
				for _, expectedMount := range tt.expectedMounts {
					assert.Contains(t, agentMounts, expectedMount,
						"Expected mount %q to be present in agent config", expectedMount)
				}

				// Verify unexpected mounts are not present
				for _, unexpectedMount := range tt.unexpectedMounts {
					for _, mount := range agentMounts {
						assert.NotContains(t, mount, unexpectedMount,
							"Unexpected mount pattern %q found in mount %q", unexpectedMount, mount)
					}
				}
			}
		})
	}
}

// TestRuntimeMountsInCopilotEngine tests that runtime mounts appear in the
// generated AWF command for the Copilot engine
func TestRuntimeMountsInCopilotEngine(t *testing.T) {
	t.Run("runtime mounts appear in AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			CustomSteps: `      - name: Install deps
        run: npm install`,
		}

		// Detect runtime requirements
		runtimeRequirements := DetectRuntimeRequirements(workflowData)
		require.Greater(t, len(runtimeRequirements), 0, "Should detect runtime requirements")

		// Contribute runtime mounts
		ContributeRuntimeMounts(workflowData.SandboxConfig.Agent, runtimeRequirements)

		// Generate execution steps
		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		require.Greater(t, len(steps), 0, "Should generate execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// Verify runtime mounts are included in AWF command
		assert.Contains(t, stepContent, "--mount /opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
			"AWF command should include node toolcache mount")
		assert.Contains(t, stepContent, "--mount /home/runner/.npm:/home/runner/.npm:rw",
			"AWF command should include npm cache mount")
	})

	t.Run("runtime mounts are sorted in AWF command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
					// Add custom mounts to test sorting with runtime mounts
					Mounts: []string{
						"/zzz/custom:/zzz/custom:ro", // Should appear after runtime mounts
						"/aaa/custom:/aaa/custom:ro", // Should appear before runtime mounts
					},
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			CustomSteps: `      - name: Build
        run: go build`,
		}

		// Detect runtime requirements
		runtimeRequirements := DetectRuntimeRequirements(workflowData)

		// Contribute runtime mounts
		ContributeRuntimeMounts(workflowData.SandboxConfig.Agent, runtimeRequirements)

		// Generate execution steps
		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		require.Greater(t, len(steps), 0, "Should generate execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// Find positions of mounts in the output
		aaaPos := strings.Index(stepContent, "--mount /aaa/custom:/aaa/custom:ro")
		goToolcachePos := strings.Index(stepContent, "--mount /opt/hostedtoolcache/go:/opt/hostedtoolcache/go:ro")
		zzzPos := strings.Index(stepContent, "--mount /zzz/custom:/zzz/custom:ro")

		// Verify all mounts are present
		assert.NotEqual(t, -1, aaaPos, "Should find custom /aaa mount")
		assert.NotEqual(t, -1, goToolcachePos, "Should find go toolcache mount")
		assert.NotEqual(t, -1, zzzPos, "Should find custom /zzz mount")

		// Verify mounts are in alphabetical order by checking positions
		// Custom mounts and runtime mounts should all be sorted together
		if aaaPos != -1 && goToolcachePos != -1 {
			assert.Less(t, aaaPos, goToolcachePos,
				"/aaa mount should appear before /opt mount (alphabetically)")
		}
		if goToolcachePos != -1 && zzzPos != -1 {
			assert.Less(t, goToolcachePos, zzzPos,
				"/opt mount should appear before /zzz mount (alphabetically)")
		}
	})
}
