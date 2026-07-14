//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateDockerSbxInstallSteps verifies that all four docker-sbx install step
// generators produce non-empty output with the expected key content.
func TestGenerateDockerSbxInstallSteps(t *testing.T) {
	t.Run("KVM check step", func(t *testing.T) {
		step := generateDockerSbxKVMCheckStep()
		require.NotEmpty(t, step, "KVM check step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "kvm", "must check for KVM availability")
		assert.Contains(t, content, "test -e /dev/kvm", "must check /dev/kvm exists")
		assert.Contains(t, content, "exit 1", "must fail with exit 1 when KVM is absent")
	})

	t.Run("secrets check step", func(t *testing.T) {
		step := generateDockerSbxSecretsCheckStep()
		require.NotEmpty(t, step, "secrets check step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "DOCKER_PAT", "must check DOCKER_PAT secret")
		assert.Contains(t, content, "DOCKER_USERNAME", "must check DOCKER_USERNAME secret")
		assert.Contains(t, content, "secrets.DOCKER_PAT", "must reference secrets.DOCKER_PAT")
		assert.Contains(t, content, "secrets.DOCKER_USERNAME", "must reference secrets.DOCKER_USERNAME")
		assert.Contains(t, content, "exit 1", "must fail with exit 1 when secrets are missing")
	})

	t.Run("install step", func(t *testing.T) {
		step := generateDockerSbxInstallStep()
		require.NotEmpty(t, step, "install step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "docker-sbx", "must install docker-sbx package")
		assert.Contains(t, content, "sbx version", "must verify sbx is installed")
		assert.Contains(t, content, "chmod 666 /dev/kvm", "must fix KVM permissions")
		assert.Contains(t, content, "get.docker.com", "must add Docker apt repo")
	})

	t.Run("auth and daemon step", func(t *testing.T) {
		step := generateDockerSbxAuthAndDaemonStep()
		require.NotEmpty(t, step, "auth and daemon step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "sbx daemon start", "must start sbx daemon")
		assert.Contains(t, content, "docker login", "must authenticate with Docker")
		assert.Contains(t, content, "sbx login", "must authenticate sbx with Docker Hub")
		assert.Contains(t, content, "sbx policy reset", "must reset sbx policy")
		assert.Contains(t, content, "sbx policy init allow-all", "must init sbx allow-all policy")
		assert.Contains(t, content, "docker/sandbox-templates:shell-docker", "must pre-pull template image")
		assert.Contains(t, content, `export DOCKER_CONFIG="$(mktemp -d)"`, "must isolate Docker auth in a temporary config")
		assert.Contains(t, content, `trap 'rm -rf "${DOCKER_CONFIG}"' EXIT`, "must clean up temporary Docker auth on exit")
		// Secrets must be passed via env, not inline in the run: block
		assert.Contains(t, content, "DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}", "must use env for DOCKER_PAT")
		assert.Contains(t, content, "DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}", "must use env for DOCKER_USERNAME")
		// The run: section must use env var references (${DOCKER_PAT_VAL}) not raw secret expressions.
		// Extract the run: body to verify secret expressions don't appear in shell commands.
		parts := strings.SplitN(content, "run: |", 2)
		require.Len(t, parts, 2, "step must have a run: section")
		runBody := parts[1]
		assert.NotContains(t, runBody, "${{ secrets.DOCKER_PAT }}",
			"raw secrets.DOCKER_PAT expression must not appear in shell commands")
		assert.NotContains(t, runBody, "${{ secrets.DOCKER_USERNAME }}",
			"raw secrets.DOCKER_USERNAME expression must not appear in shell commands")
	})

	t.Run("pre-flight step", func(t *testing.T) {
		step := generateDockerSbxPreFlightStep()
		require.NotEmpty(t, step, "pre-flight step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "sbx create", "must create a test sandbox")
		assert.Contains(t, content, "test-sandbox-direct", "must use a named test sandbox")
		assert.Contains(t, content, "sbx exec", "must exec a command in the sandbox")
		assert.Contains(t, content, "uname -a", "must run uname -a as smoke test")
		assert.Contains(t, content, "trap cleanup EXIT", "must register cleanup for smoke-test failures")
		assert.Contains(t, content, "sbx stop", "must stop the test sandbox")
		assert.Contains(t, content, "sbx rm", "must remove the test sandbox")
		assert.Contains(t, content, "✅ sbx ready", "must confirm readiness")
	})

	t.Run("credential refresh step", func(t *testing.T) {
		step := generateDockerSbxCredentialRefreshStep()
		require.NotEmpty(t, step, "credential refresh step must not be empty")
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Refresh sbx credentials", "must have correct step name")
		assert.Contains(t, content, "sbx login", "must re-authenticate with sbx login")
		assert.Contains(t, content, "DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}", "must use env for DOCKER_PAT")
		assert.Contains(t, content, "DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}", "must use env for DOCKER_USERNAME")
		assert.Contains(t, content, "✅ sbx credentials refreshed", "must confirm refresh")
		// The run: section must use env var references, not raw secret expressions.
		parts := strings.SplitN(content, "run: |", 2)
		require.Len(t, parts, 2, "step must have a run: section")
		runBody := parts[1]
		assert.NotContains(t, runBody, "${{ secrets.DOCKER_PAT }}",
			"raw secrets expression must not appear in shell commands")
		assert.NotContains(t, runBody, "${{ secrets.DOCKER_USERNAME }}",
			"raw secrets expression must not appear in shell commands")
	})
}

// TestDockerSbxInstallStepOrderInBuildNpmEngineInstallStepsWithAWF verifies that all
// docker-sbx pre-flight steps are emitted BEFORE the AWF install step.
func TestDockerSbxInstallStepOrderInBuildNpmEngineInstallStepsWithAWF(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				NetworkIsolation:      false, // will be overridden by isDockerSbxRuntime
				SudoExplicitlyEnabled: true,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}

	steps := BuildNpmEngineInstallStepsWithAWF(nil, workflowData)
	require.NotEmpty(t, steps, "must generate installation steps")

	// Locate the key steps by their content.
	kvmIdx := -1
	secretsIdx := -1
	installIdx := -1
	authIdx := -1
	preflightIdx := -1
	awfIdx := -1
	for i, step := range steps {
		content := strings.Join(step, "\n")
		switch {
		case strings.Contains(content, "Check KVM availability"):
			kvmIdx = i
		case strings.Contains(content, "Check Docker Hub secrets"):
			secretsIdx = i
		case strings.Contains(content, "Install docker-sbx"):
			installIdx = i
		case strings.Contains(content, "Start docker-sbx daemon"):
			authIdx = i
		case strings.Contains(content, "pre-flight smoke test"):
			preflightIdx = i
		case strings.Contains(content, "install_awf_binary.sh"):
			awfIdx = i
		}
	}

	require.NotEqual(t, -1, kvmIdx, "KVM check step must be present")
	require.NotEqual(t, -1, secretsIdx, "secrets check step must be present")
	require.NotEqual(t, -1, installIdx, "docker-sbx install step must be present")
	require.NotEqual(t, -1, authIdx, "auth and daemon step must be present")
	require.NotEqual(t, -1, preflightIdx, "pre-flight step must be present")
	require.NotEqual(t, -1, awfIdx, "AWF install step must be present")

	// All docker-sbx steps must precede the AWF install step.
	assert.Less(t, kvmIdx, awfIdx, "KVM check step must come before AWF install")
	assert.Less(t, secretsIdx, awfIdx, "secrets check step must come before AWF install")
	assert.Less(t, installIdx, awfIdx, "docker-sbx install step must come before AWF install")
	assert.Less(t, authIdx, awfIdx, "auth and daemon step must come before AWF install")
	assert.Less(t, preflightIdx, awfIdx, "pre-flight step must come before AWF install")
	// And they must be in the correct logical order relative to each other.
	assert.Less(t, kvmIdx, secretsIdx, "KVM check must come before secrets check")
	assert.Less(t, secretsIdx, installIdx, "secrets check must come before install")
	assert.Less(t, installIdx, authIdx, "install must come before auth/daemon")
	assert.Less(t, authIdx, preflightIdx, "auth/daemon must come before pre-flight")
}

// TestDockerSbxAWFArgs verifies that --container-runtime sbx is added to AWF args
// when docker-sbx runtime is configured.
func TestDockerSbxAWFArgs(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "copilot",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFContainerRuntimeMinVersion)},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:                    "awf",
					Runtime:               AgentRuntimeDockerSbx,
					SudoExplicitlyEnabled: true,
				},
			},
		},
	}

	args := BuildAWFArgs(config)

	// Must include --container-runtime sbx
	found := false
	for i, arg := range args {
		if arg == "--container-runtime" && i+1 < len(args) && args[i+1] == "sbx" {
			found = true
			break
		}
	}
	assert.True(t, found, "AWF args must include --container-runtime sbx for docker-sbx runtime")
}

func TestDockerSbxAWFArgsSuppressesTTY(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "claude",
		UsesTTY:    true,
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "claude"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFContainerRuntimeMinVersion)},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:                    "awf",
					Runtime:               AgentRuntimeDockerSbx,
					SudoExplicitlyEnabled: true,
				},
			},
		},
	}

	args := BuildAWFArgs(config)
	assert.NotContains(t, strings.Join(args, " "), "--tty", "docker-sbx must suppress --tty to avoid sbx pty timeouts")
}

// TestDockerSbxAWFArgsAbsentByDefault verifies that --container-runtime sbx is NOT
// added when no runtime is configured.
func TestDockerSbxAWFArgsAbsentByDefault(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "copilot",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
		},
	}

	args := BuildAWFArgs(config)
	argStr := strings.Join(args, " ")
	assert.NotContains(t, argStr, "--container-runtime", "AWF args must not include --container-runtime when no runtime is set")
}

// TestDockerSbxAWFArgsVersionGated verifies that --container-runtime sbx is omitted when
// the effective AWF version predates AWFContainerRuntimeMinVersion.
func TestDockerSbxAWFArgsVersionGated(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "copilot",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				// Pin to a version that predates containerRuntime support.
				Firewall: &FirewallConfig{Enabled: true, Version: "v0.27.29"},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:                    "awf",
					Runtime:               AgentRuntimeDockerSbx,
					SudoExplicitlyEnabled: true,
				},
			},
		},
	}

	args := BuildAWFArgs(config)
	assert.NotContains(t, strings.Join(args, " "), "--container-runtime",
		"AWF args must omit --container-runtime when the AWF version is too old")
}

// TestDockerSbxAWFConfigJSON verifies that the AWF config JSON for docker-sbx does NOT
// include containerRuntime (docker-sbx is not an OCI runtime) but DOES include
// host.docker.internal in allowDomains and sets network.isolation: true.
func TestDockerSbxAWFConfigJSON(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig:   &EngineConfig{ID: "copilot"},
			TimeoutMinutes: "timeout-minutes: 30",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:                    "awf",
					Runtime:               AgentRuntimeDockerSbx,
					SudoExplicitlyEnabled: true,
				},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	// docker-sbx is not an OCI runtime — containerRuntime must NOT appear in JSON.
	assert.NotContains(t, jsonStr, `"containerRuntime"`,
		"docker-sbx must not set container.containerRuntime in AWF config JSON")

	// host.docker.internal must be in network.allowDomains so the microVM can
	// reach host-published services (api-proxy, MCP gateway, Squid proxy).
	assert.Contains(t, jsonStr, "host.docker.internal",
		"AWF config JSON must include host.docker.internal in network.allowDomains for docker-sbx")

	// network.isolation must be true for docker-sbx.
	assert.Contains(t, jsonStr, `"isolation":true`,
		"AWF config JSON must have network.isolation: true for docker-sbx")

	// docker-sbx must pass a concrete agent timeout to AWF.
	assert.Contains(t, jsonStr, `"agentTimeout":30`,
		"AWF config JSON must include container.agentTimeout for docker-sbx")
}

func TestDockerSbxEngineCLIWiring(t *testing.T) {
	workflowData := &WorkflowData{
		Name:          "test-workflow",
		EngineConfig:  &EngineConfig{ID: "claude"},
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeDockerSbx, SudoExplicitlyEnabled: true}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}

	t.Run("claude install and execution use sbx-visible CLI path", func(t *testing.T) {
		engine := NewClaudeEngine()
		installSteps := engine.GetInstallationSteps(workflowData)
		installContent := strings.Join(flattenSteps(installSteps), "\n")
		assert.Contains(t, installContent, `npm install --prefix "${RUNNER_TEMP}/gh-aw/engine-cli" @anthropic-ai/claude-code@`+string(constants.DefaultClaudeCodeVersion))
		assert.Contains(t, installContent, `ln -sf "../node_modules/.bin/claude" "${RUNNER_TEMP}/gh-aw/engine-cli/bin/claude"`)

		execSteps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")
		require.NotEmpty(t, execSteps)
		execContent := strings.Join(execSteps[0], "\n")
		assert.Contains(t, execContent, `export PATH="${RUNNER_TEMP}/gh-aw/engine-cli/bin:$PATH"`)
	})

	t.Run("docker-sbx keeps engine and MCP CLI PATH setup independent", func(t *testing.T) {
		workflowData.ParsedTools = &ToolsConfig{
			CLIProxy: true,
			Custom: map[string]MCPServerConfig{
				"myserver": {},
			},
		}
		workflowData.Tools = map[string]any{
			"myserver": map[string]any{
				"mode": "remote",
			},
		}

		engine := NewClaudeEngine()
		execSteps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")
		require.NotEmpty(t, execSteps)
		execContent := strings.Join(execSteps[0], "\n")

		assert.Contains(t, execContent, `export PATH="${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH"`)
		assert.Contains(t, execContent, `export PATH="${RUNNER_TEMP}/gh-aw/engine-cli/bin:$PATH"`)

		mcpIdx := strings.Index(execContent, `export PATH="${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH"`)
		engineIdx := strings.Index(execContent, `export PATH="${RUNNER_TEMP}/gh-aw/engine-cli/bin:$PATH"`)
		assert.GreaterOrEqual(t, mcpIdx, 0)
		assert.GreaterOrEqual(t, engineIdx, 0)
		assert.Less(t, mcpIdx, engineIdx, "engine path export must run after MCP export so engine CLI takes PATH precedence")
	})

	t.Run("codex install and execution use sbx-visible CLI path", func(t *testing.T) {
		engine := NewCodexEngine()
		workflowData.EngineConfig = &EngineConfig{ID: "codex"}
		installSteps := engine.GetInstallationSteps(workflowData)
		installContent := strings.Join(flattenSteps(installSteps), "\n")
		assert.Contains(t, installContent, `npm install --ignore-scripts --prefix "${RUNNER_TEMP}/gh-aw/engine-cli" @openai/codex@`+string(constants.DefaultCodexVersion))
		assert.Contains(t, installContent, `ln -sf "../node_modules/.bin/codex" "${RUNNER_TEMP}/gh-aw/engine-cli/bin/codex"`)

		execSteps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")
		require.NotEmpty(t, execSteps)
		execContent := strings.Join(execSteps[0], "\n")
		assert.Contains(t, execContent, `export PATH="${RUNNER_TEMP}/gh-aw/engine-cli/bin:$PATH"`)
	})
}

// flattenSteps joins a small slice of GitHubActionStep values so docker-sbx tests can
// assert across multi-step install blocks without repeating nested loops in each case.
func flattenSteps(steps []GitHubActionStep) []string {
	var lines []string
	for _, step := range steps {
		lines = append(lines, step...)
	}
	return lines
}

// TestDockerSbxNetworkIsolationAlwaysTrue verifies that isAWFNetworkIsolationEnabled
// returns true for docker-sbx even when sudo: true sets NetworkIsolation=false.
func TestDockerSbxNetworkIsolationAlwaysTrue(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				NetworkIsolation:      false, // sudo: true sets this to false normally
				SudoExplicitlyEnabled: true,
			},
		},
	}

	assert.True(t, isAWFNetworkIsolationEnabled(workflowData),
		"docker-sbx must always use network isolation regardless of sudo setting")
}

// TestDockerSbxContainerRuntimeEmpty verifies that getAgentContainerRuntime returns
// an empty string for docker-sbx (it is not an OCI runtime).
func TestDockerSbxContainerRuntimeEmpty(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeDockerSbx,
			},
		},
	}

	assert.Empty(t, getAgentContainerRuntime(workflowData),
		"docker-sbx must return empty string from getAgentContainerRuntime")
}

// TestDockerSbxValidation_ArcDindIncompatible verifies that docker-sbx + arc-dind is
// a compile-time error.
func TestDockerSbxValidation_ArcDindIncompatible(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				SudoExplicitlyEnabled: true,
			},
		},
		RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "docker-sbx + arc-dind must produce a compile-time error")
	assert.Contains(t, err.Error(), "arc-dind", "error must mention arc-dind")
	assert.Contains(t, err.Error(), "docker-sbx", "error must mention docker-sbx")
}

// TestDockerSbxValidation_SudoFalseRejected verifies that docker-sbx without
// sudo: true is a compile-time error.
func TestDockerSbxValidation_SudoFalseRejected(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				NetworkIsolation:      true,  // sudo: false
				SudoExplicitlyEnabled: false, // sudo not explicitly set
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "docker-sbx without sudo: true must produce a compile-time error")
	assert.Contains(t, err.Error(), "sudo: true", "error must mention sudo: true")
	assert.Contains(t, err.Error(), "docker-sbx", "error must mention docker-sbx")
}

// TestDockerSbxValidation_DefaultVersionRejected verifies that docker-sbx is rejected
// when the effective AWF version predates --container-runtime support.
func TestDockerSbxValidation_DefaultVersionRejected(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				NetworkIsolation:      false, // sudo: true → NetworkIsolation=false (overridden by isDockerSbxRuntime)
				SudoExplicitlyEnabled: true,
				// Pin to a version that predates containerRuntime support.
				Version: "v0.27.29",
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "docker-sbx with an AWF version predating containerRuntime support must fail validation")
	assert.Contains(t, err.Error(), string(constants.AWFContainerRuntimeMinVersion))
}

// TestDockerSbxValidation_MinVersionSatisfied verifies that docker-sbx passes validation
// when the effective AWF version supports --container-runtime.
func TestDockerSbxValidation_MinVersionSatisfied(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeDockerSbx,
				NetworkIsolation:      false,
				SudoExplicitlyEnabled: true,
				Version:               string(constants.AWFContainerRuntimeMinVersion),
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	assert.NoError(t, err, "docker-sbx with a supported AWF version must pass validation")
}

// TestDockerSbxStrictModeSudoSuppressed verifies that sandbox.agent.sudo: true combined
// with runtime: docker-sbx does NOT produce a strict-mode error (sudo is required for
// docker-sbx install and the deprecation warning is suppressed).
func TestDockerSbxStrictModeSudoSuppressed(t *testing.T) {
	sandboxConfig := &SandboxConfig{
		Agent: &AgentSandboxConfig{
			ID:                    "awf",
			Runtime:               AgentRuntimeDockerSbx,
			NetworkIsolation:      false,
			SudoExplicitlyEnabled: true,
		},
	}

	compiler := NewCompiler()
	compiler.strictMode = true

	err := compiler.validateStrictSandboxCustomization(sandboxConfig)
	assert.NoError(t, err, "sudo:true + runtime:docker-sbx must NOT produce a strict-mode error")
}

// TestIsDockerSbxRuntime verifies the isDockerSbxRuntime helper.
func TestIsDockerSbxRuntime(t *testing.T) {
	t.Run("returns false for nil workflow data", func(t *testing.T) {
		assert.False(t, isDockerSbxRuntime(nil))
	})

	t.Run("returns false when no sandbox config", func(t *testing.T) {
		assert.False(t, isDockerSbxRuntime(&WorkflowData{}))
	})

	t.Run("returns false when runtime is not docker-sbx", func(t *testing.T) {
		assert.False(t, isDockerSbxRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{ID: "awf"},
			},
		}))
	})

	t.Run("returns false when agent is disabled", func(t *testing.T) {
		assert.False(t, isDockerSbxRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:       "awf",
					Runtime:  AgentRuntimeDockerSbx,
					Disabled: true,
				},
			},
		}))
	})

	t.Run("returns true when runtime is docker-sbx", func(t *testing.T) {
		assert.True(t, isDockerSbxRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:      "awf",
					Runtime: AgentRuntimeDockerSbx,
				},
			},
		}))
	})
}

// TestDockerSbxFrontmatterExtraction verifies end-to-end that a workflow with
// sandbox.agent.runtime: docker-sbx compiles correctly and produces the expected output.
func TestDockerSbxFrontmatterExtraction(t *testing.T) {
	workflowsDir := t.TempDir()

	markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
network:
  allowed:
    - "example.com"
sandbox:
  agent:
    id: awf
    runtime: docker-sbx
    version: v0.28.0
    sudo: true
---

# Test docker-sbx Runtime
`

	testFile := filepath.Join(workflowsDir, "test-docker-sbx.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "compilation with runtime: docker-sbx must succeed")

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-docker-sbx.lock.yml"))
	require.NoError(t, err)
	lockStr := string(lockContent)

	// KVM check step must be present.
	assert.Contains(t, lockStr, "Check KVM availability", "compiled workflow must include KVM availability check")
	// Secrets check step must be present.
	assert.Contains(t, lockStr, "Check Docker Hub secrets", "compiled workflow must include Docker Hub secrets check")
	// docker-sbx install step must be present.
	assert.Contains(t, lockStr, "Install docker-sbx", "compiled workflow must include docker-sbx install step")
	// Auth and daemon step must be present.
	assert.Contains(t, lockStr, "Start docker-sbx daemon", "compiled workflow must include sbx daemon step")
	// Pre-flight step must be present.
	assert.Contains(t, lockStr, "pre-flight smoke test", "compiled workflow must include pre-flight step")
	// AWF install step must also be present.
	assert.Contains(t, lockStr, "Install AWF binary", "compiled workflow must include AWF install step")

	// All docker-sbx steps must appear before AWF install step.
	kvmPos := strings.Index(lockStr, "Check KVM availability")
	awfPos := strings.Index(lockStr, "Install AWF binary")
	assert.Less(t, kvmPos, awfPos, "KVM check step must precede AWF install step")

	// containerRuntime must NOT appear (docker-sbx is not an OCI runtime).
	assert.NotContains(t, lockStr, `"containerRuntime"`, "containerRuntime must not appear for docker-sbx")

	// host.docker.internal must be in the allowed domains.
	assert.Contains(t, lockStr, "host.docker.internal", "host.docker.internal must be in allowed domains")

	// --container-runtime sbx must appear in the AWF invocation.
	assert.Contains(t, lockStr, "--container-runtime sbx", "AWF invocation must include --container-runtime sbx")

	// Credential refresh step must be present.
	assert.Contains(t, lockStr, "Refresh sbx credentials", "compiled workflow must include credential refresh step before execution")

	// Credential refresh step must appear AFTER pre-flight but BEFORE execution.
	refreshPos := strings.Index(lockStr, "Refresh sbx credentials")
	preflightPos := strings.Index(lockStr, "pre-flight smoke test")
	execPos := strings.Index(lockStr, "agentic_execution")
	assert.Greater(t, refreshPos, preflightPos, "credential refresh must come after pre-flight step")
	assert.Less(t, refreshPos, execPos, "credential refresh must come before agent execution step")
}
