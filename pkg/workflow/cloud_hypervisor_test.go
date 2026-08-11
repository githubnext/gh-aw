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

func TestGenerateCloudHypervisorSetupSteps(t *testing.T) {
	t.Run("host preflight step", func(t *testing.T) {
		step := generateCloudHypervisorHostPreflightStep()
		require.NotEmpty(t, step)
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Check host eligibility for cloud-hypervisor")
		assert.Contains(t, content, "cloud_hypervisor_host_preflight.sh")
	})

	t.Run("bundle setup step", func(t *testing.T) {
		step := generateCloudHypervisorBundleSetupStep("v0.28.0")
		require.NotEmpty(t, step)
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Download and verify cloud-hypervisor bundle")
		assert.Contains(t, content, "id: cloud-hypervisor-bundle")
		assert.Contains(t, content, "GH_AW_AWF_VERSION: v0.28.0")
		assert.Contains(t, content, "cloud_hypervisor_setup_bundle.sh")
	})
}

func TestCloudHypervisorInstallStepOrderInBuildNpmEngineInstallStepsWithAWF(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig:      &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		NetworkPermissions: &NetworkPermissions{Firewall: &FirewallConfig{Enabled: true}},
	}

	steps := BuildNpmEngineInstallStepsWithAWF(nil, workflowData)
	require.NotEmpty(t, steps)

	preflightIdx := -1
	bundleIdx := -1
	awfIdx := -1
	for i, step := range steps {
		content := strings.Join(step, "\n")
		switch {
		case strings.Contains(content, "Check host eligibility for cloud-hypervisor"):
			preflightIdx = i
		case strings.Contains(content, "Download and verify cloud-hypervisor bundle"):
			bundleIdx = i
		case strings.Contains(content, "install_awf_binary.sh"):
			awfIdx = i
		}
	}

	require.NotEqual(t, -1, preflightIdx)
	require.NotEqual(t, -1, bundleIdx)
	require.NotEqual(t, -1, awfIdx)
	assert.Less(t, preflightIdx, bundleIdx)
	assert.Less(t, bundleIdx, awfIdx)
}

func TestCloudHypervisorAWFArgs(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "copilot",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFContainerRuntimeMinVersion)},
			},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		},
	}

	args := strings.Join(BuildAWFArgs(config), " ")
	assert.Contains(t, args, "--container-runtime cloud-hypervisor")
	assert.Contains(t, args, "--cloud-hypervisor-preview")
	assert.NotContains(t, args, "${{ steps.cloud-hypervisor-bundle.outputs.")
}

func TestCloudHypervisorAWFConfigJSON(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig:   &EngineConfig{ID: "copilot"},
			TimeoutMinutes: "timeout-minutes: 30",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)
	assert.NotContains(t, jsonStr, `"containerRuntime"`)
	assert.Contains(t, jsonStr, "host.docker.internal")
	assert.Contains(t, jsonStr, `"isolation":true`)
	assert.Contains(t, jsonStr, `"agentTimeout":30`)
}

func TestCloudHypervisorValidationArcDindIncompatible(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		RunnerConfig:  &RunnerConfig{Topology: RunnerTopologyArcDind},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "arc-dind")
	require.ErrorContains(t, err, "cloud-hypervisor")
}

func TestCloudHypervisorFrontmatterExtraction(t *testing.T) {
	workflowsDir := t.TempDir()

	markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
sandbox:
  agent:
    id: awf
    runtime: cloud-hypervisor
    version: v0.28.0
---

# Test cloud-hypervisor Runtime
`

	testFile := filepath.Join(workflowsDir, "test-cloud-hypervisor.md")
	err := os.WriteFile(testFile, []byte(markdown), 0o644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err)

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-cloud-hypervisor.lock.yml"))
	require.NoError(t, err)
	lockStr := string(lockContent)

	assert.Contains(t, lockStr, "Check host eligibility for cloud-hypervisor")
	assert.Contains(t, lockStr, "Download and verify cloud-hypervisor bundle")
	assert.Contains(t, lockStr, "GH_AW_AWF_VERSION: v0.28.0")
	assert.Contains(t, lockStr, "--container-runtime cloud-hypervisor")
	assert.Contains(t, lockStr, "--cloud-hypervisor-preview")
	assert.Contains(t, lockStr, "--cloud-hypervisor-kernel \"${GH_AW_CLOUD_HYPERVISOR_KERNEL}\"")
	assert.Contains(t, lockStr, "--cloud-hypervisor-supervisor-sha256 \"${GH_AW_CLOUD_HYPERVISOR_SUPERVISOR_SHA256}\"")
}

func TestIsCloudHypervisorRuntime(t *testing.T) {
	assert.False(t, isCloudHypervisorRuntime(nil))
	assert.False(t, isCloudHypervisorRuntime(&WorkflowData{}))
	assert.True(t, isCloudHypervisorRuntime(&WorkflowData{SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeCloudHypervisor}}}))
}

func TestCloudHypervisorShellScriptContent(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	shDir := filepath.Join(wd, "..", "..", "actions", "setup", "sh")

	tests := []struct {
		script   string
		contains []string
	}{
		{
			script:   "cloud_hypervisor_host_preflight.sh",
			contains: []string{"RUNNER_ENVIRONMENT", "github-hosted", "ImageOS", "/dev/kvm", "cloud-hypervisor preview"},
		},
		{
			script:   "cloud_hypervisor_setup_bundle.sh",
			contains: []string{"cloud-hypervisor-test-x86_64.tar.gz", "SHA256SUMS", "manifest.json", "binary_path=", "binary_sha256="},
		},
	}

	for _, tc := range tests {
		t.Run(tc.script, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(shDir, tc.script))
			require.NoError(t, err)
			for _, expected := range tc.contains {
				assert.Contains(t, string(content), expected)
			}
		})
	}
}
