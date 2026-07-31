//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBoundedQueryRuntimeMatrix(t *testing.T) {
	agentRuntimes := []struct {
		name                     string
		runtime                  AgentRuntime
		sudo                     bool
		expectedContainerRuntime string
	}{
		{name: "docker"},
		{name: "gvisor", runtime: AgentRuntimeGVisor, sudo: true, expectedContainerRuntime: "gvisor"},
		{name: "sbx", runtime: AgentRuntimeDockerSbx, sudo: true},
	}
	queryRuntimes := []BoundedQueryRuntime{
		BoundedQueryRuntimeDocker,
		BoundedQueryRuntimeGVisor,
		BoundedQueryRuntimeSbx,
	}

	for _, agentRuntime := range agentRuntimes {
		for _, queryRuntime := range queryRuntimes {
			t.Run(agentRuntime.name+"-agent/"+queryRuntime+"-query", func(t *testing.T) {
				workflowData := &WorkflowData{
					EngineConfig:   &EngineConfig{ID: "copilot"},
					TimeoutMinutes: "timeout-minutes: 30",
					Tools: map[string]any{
						"github": map[string]any{"mode": "remote"},
					},
					NetworkPermissions: &NetworkPermissions{
						Firewall: &FirewallConfig{
							Enabled: true,
							Version: string(constants.AWFBoundedQueriesMinVersion),
						},
					},
					SandboxConfig: &SandboxConfig{
						Agent: &AgentSandboxConfig{
							ID:                    "awf",
							Version:               string(constants.AWFBoundedQueriesMinVersion),
							Runtime:               agentRuntime.runtime,
							SudoExplicitlyEnabled: agentRuntime.sudo,
						},
					},
					ParsedTools: &ToolsConfig{
						GitHub: &GitHubToolConfig{
							Mode: GitHubMCPModeRemote,
							BoundedQueries: &BoundedQueriesConfig{
								PrivateRepos: []*BoundedQueryPrivateRepo{
									{Repo: "my-org/internal-service", Sensitivity: "internal"},
								},
								Runtime: queryRuntime,
							},
						},
					},
				}

				require.NoError(t, validateSandboxConfig(workflowData))
				require.NoError(t, validateBoundedQueriesConfig(workflowData))

				jsonStr, err := BuildAWFConfigJSON(AWFCommandConfig{
					EngineName:     "copilot",
					AllowedDomains: "github.com",
					WorkflowData:   workflowData,
				})
				require.NoError(t, err)

				var config AWFConfigFile
				require.NoError(t, json.Unmarshal([]byte(jsonStr), &config))
				require.NotNil(t, config.BoundedQueries)
				assert.Equal(t, queryRuntime, config.BoundedQueries.Runtime,
					"query runtime must be emitted verbatim without fallback")

				if agentRuntime.expectedContainerRuntime == "" {
					if config.Container != nil {
						assert.Empty(t, config.Container.ContainerRuntime)
					}
				} else {
					require.NotNil(t, config.Container)
					assert.Equal(t, agentRuntime.expectedContainerRuntime, config.Container.ContainerRuntime)
				}
			})
		}
	}
}

func TestBoundedQueryRuntimeVersionGate(t *testing.T) {
	for _, runtime := range []BoundedQueryRuntime{
		BoundedQueryRuntimeDocker,
		BoundedQueryRuntimeGVisor,
		BoundedQueryRuntimeSbx,
	} {
		t.Run(runtime, func(t *testing.T) {
			makeWorkflow := func(version string) *WorkflowData {
				return &WorkflowData{
					SandboxConfig: &SandboxConfig{
						Agent: &AgentSandboxConfig{ID: "awf", Version: version},
					},
					ParsedTools: &ToolsConfig{
						GitHub: &GitHubToolConfig{
							BoundedQueries: &BoundedQueriesConfig{
								PrivateRepos: []*BoundedQueryPrivateRepo{
									{Repo: "my-org/internal-service", Sensitivity: "internal"},
								},
								Runtime: runtime,
							},
						},
					},
				}
			}

			err := validateBoundedQueriesConfig(makeWorkflow("v0.27.99"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), string(constants.AWFBoundedQueriesMinVersion))
			assert.NoError(t, validateBoundedQueriesConfig(
				makeWorkflow(string(constants.AWFBoundedQueriesMinVersion))))
		})
	}
}

func TestSbxBoundedQueryFrontmatterCompilation(t *testing.T) {
	workflowsDir := t.TempDir()
	markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
network:
  allowed:
    - example.com
sandbox:
  agent:
    id: awf
    version: v0.28.0
tools:
  github:
    bounded-queries:
      private-repos:
        - repo: my-org/internal-service
          sensitivity: internal
      runtime: sbx
---

# Test sbx bounded-query runtime
`
	testFile := filepath.Join(workflowsDir, "test-sbx-bounded-query.md")
	require.NoError(t, os.WriteFile(testFile, []byte(markdown), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-sbx-bounded-query.lock.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "sbx")
	assert.NotContains(t, string(lockContent), `"runtime":"docker-sbx"`)
	assert.True(t, usesSbxBoundedQueryRuntime(&WorkflowData{
		ParsedTools: &ToolsConfig{
			GitHub: &GitHubToolConfig{
				BoundedQueries: &BoundedQueriesConfig{Runtime: BoundedQueryRuntimeSbx},
			},
		},
	}))
}
