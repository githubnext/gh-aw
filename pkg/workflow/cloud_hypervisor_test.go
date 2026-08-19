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
	t.Run("KVM access step", func(t *testing.T) {
		step := generateCloudHypervisorKVMAccessStep()
		require.NotEmpty(t, step)
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Grant runner access to KVM")
		assert.Contains(t, content, "cloud_hypervisor_kvm_access.sh")
	})

	t.Run("host preflight step", func(t *testing.T) {
		step := generateCloudHypervisorHostPreflightStep()
		require.NotEmpty(t, step)
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Check host eligibility for cloud-hypervisor")
		assert.Contains(t, content, "cloud_hypervisor_host_preflight.sh")
	})

	t.Run("bundle setup step", func(t *testing.T) {
		step := generateCloudHypervisorBundleSetupStep("v0.28.1")
		require.NotEmpty(t, step)
		content := strings.Join(step, "\n")
		assert.Contains(t, content, "Download and verify cloud-hypervisor bundle")
		assert.Contains(t, content, "id: cloud-hypervisor-bundle")
		assert.Contains(t, content, "GH_AW_AWF_VERSION: v0.28.1")
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

	kvmAccessIdx := -1
	preflightIdx := -1
	bundleIdx := -1
	awfIdx := -1
	awfInstallContent := ""
	for i, step := range steps {
		content := strings.Join(step, "\n")
		switch {
		case strings.Contains(content, "Grant runner access to KVM"):
			kvmAccessIdx = i
		case strings.Contains(content, "Check host eligibility for cloud-hypervisor"):
			preflightIdx = i
		case strings.Contains(content, "Download and verify cloud-hypervisor bundle"):
			bundleIdx = i
		case strings.Contains(content, "install_awf_binary.sh"):
			awfIdx = i
			awfInstallContent = content
		}
	}

	require.NotEqual(t, -1, kvmAccessIdx)
	require.NotEqual(t, -1, preflightIdx)
	require.NotEqual(t, -1, bundleIdx)
	require.NotEqual(t, -1, awfIdx)
	assert.Less(t, kvmAccessIdx, preflightIdx)
	assert.Less(t, preflightIdx, bundleIdx)
	assert.Less(t, bundleIdx, awfIdx)
	assert.NotContains(t, awfInstallContent, "--rootless")
}

func TestCloudHypervisorAWFArgs(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "copilot",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFCloudHypervisorMinVersion)},
			},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		},
	}

	args := strings.Join(BuildAWFArgs(config), " ")
	assert.Contains(t, args, "--container-runtime cloud-hypervisor")
	assert.Contains(t, args, "--cloud-hypervisor-preview")
	assert.Contains(t, args, "--cloud-hypervisor-vcpus 2")
	assert.Contains(t, args, "--cloud-hypervisor-memory-mib 4096")
	assert.NotContains(t, args, "${{ steps.cloud-hypervisor-bundle.outputs.")
	assert.NotContains(t, args, "--mount")
}

func TestCloudHypervisorAWFCommandOmitsUnsupportedMountsAndTTY(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "claude",
		UsesTTY:    true,
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "claude"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFCloudHypervisorMinVersion)},
			},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeCloudHypervisor,
				Mounts:  []string{"/tmp/custom:/tmp/custom"},
			}},
		},
	}

	command := BuildAWFCommand(config)
	assert.Contains(t, command, "sudo --preserve-env awf")
	assert.Contains(t, command, `--cloud-hypervisor-virtiofsd-sha256 "${GH_AW_CLOUD_HYPERVISOR_VIRTIOFSD_SHA256}"`)
	assert.Contains(t, command, "GH_AW_CLOUD_HYPERVISOR_MAX_ATTEMPTS=2")
	assert.Contains(t, command, "Cloud Hypervisor guest connectivity probe failed")
	assert.Contains(t, command, "SANDBOX_NETWORK_INIT_FAILED")
	assert.Contains(t, command, "guest network state:")
	assert.NotContains(t, command, "--mount")
	assert.NotContains(t, command, "--tty")
	assert.NotContains(t, command, "--legacy-security")
	assert.NotContains(t, command, "--enable-host-access")
}

func TestDefaultAWFCommandDoesNotUseCloudHypervisorRetryWrapper(t *testing.T) {
	config := AWFCommandConfig{
		EngineName: "claude",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "claude"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeDocker}},
		},
	}

	command := BuildAWFCommand(config)
	assert.NotContains(t, command, "GH_AW_CLOUD_HYPERVISOR_MAX_ATTEMPTS")
	assert.NotContains(t, command, "SANDBOX_NETWORK_INIT_FAILED")
}

func TestCloudHypervisorFirewallLogsUsePrivilegedMode(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:      "awf",
			Runtime: AgentRuntimeCloudHypervisor,
		}},
	}

	step := generateFirewallLogParsingStep("cloud-hypervisor", workflowData)
	assert.NotContains(t, strings.Join(step, "\n"), "--rootless")
}

func TestCloudHypervisorAWFConfigJSON(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig:   &EngineConfig{ID: "copilot"},
			TimeoutMinutes: "timeout-minutes: 60",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			Tools:         map[string]any{"github": map[string]any{"mode": "gh-proxy"}},
			SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: AgentRuntimeCloudHypervisor}},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)
	assert.NotContains(t, jsonStr, `"containerRuntime"`)
	assert.NotContains(t, jsonStr, "host.docker.internal")
	assert.Contains(t, jsonStr, `"isolation":true`)
	assert.Contains(t, jsonStr, `"topologyAttach":["awmg-mcpg"]`)
	assert.NotContains(t, jsonStr, "awmg-cli-proxy")
	assert.Contains(t, jsonStr, `"agentTimeout":60`)
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

func TestCloudHypervisorValidationRequiresPreviewVersion(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:      "awf",
			Runtime: AgentRuntimeCloudHypervisor,
			Version: string(constants.AWFCloudHypervisorMinVersion),
		}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	require.NoError(t, validateSandboxConfig(workflowData))

	workflowData.SandboxConfig.Agent.Version = "v0.27.44"
	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, string(constants.AWFCloudHypervisorMinVersion))
}

func TestCloudHypervisorValidationRejectsGHProxy(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:      "awf",
			Runtime: AgentRuntimeCloudHypervisor,
			Version: string(constants.AWFCloudHypervisorMinVersion),
		}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFCloudHypervisorMinVersion)},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "gh-proxy"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "gh-proxy")
	require.ErrorContains(t, err, "cloud-hypervisor")
}

func TestCloudHypervisorValidationRejectsAllowHostPorts(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:             "awf",
			Runtime:        AgentRuntimeCloudHypervisor,
			Version:        string(constants.AWFCloudHypervisorMinVersion),
			AllowHostPorts: []int{8080},
		}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFCloudHypervisorMinVersion)},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "allow-host-ports")
	require.ErrorContains(t, err, "cloud-hypervisor")
}

func TestCloudHypervisorValidationRejectsEnclaves(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:      "awf",
			Runtime: AgentRuntimeCloudHypervisor,
			Version: string(constants.AWFCloudHypervisorMinVersion),
		}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFCloudHypervisorMinVersion)},
		},
		Tools:    map[string]any{"github": map[string]any{"mode": "remote"}},
		Enclaves: EnclavesConfig{{}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "enclaves")
	require.ErrorContains(t, err, "cloud-hypervisor")
}

func TestCloudHypervisorFrontmatterExtraction(t *testing.T) {
	workflowsDir := t.TempDir()

	markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
timeout-minutes: 60
sandbox:
  agent:
    id: awf
    runtime: cloud-hypervisor
    version: v0.28.1
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

	assert.Contains(t, lockStr, "Grant runner access to KVM")
	assert.Contains(t, lockStr, "Check host eligibility for cloud-hypervisor")
	assert.Contains(t, lockStr, "Download and verify cloud-hypervisor bundle")
	assert.Contains(t, lockStr, "GH_AW_AWF_VERSION: v0.28.1")
	assert.Contains(t, lockStr, "sudo --preserve-env awf")
	assert.NotContains(t, lockStr, `install_awf_binary.sh" v0.28.1 --rootless`)
	assert.NotContains(t, lockStr, `print_firewall_logs.sh" --rootless`)
	assert.Contains(t, lockStr, "--container-runtime cloud-hypervisor")
	assert.Contains(t, lockStr, "--cloud-hypervisor-preview")
	assert.Contains(t, lockStr, "--cloud-hypervisor-vcpus 2")
	assert.Contains(t, lockStr, "--cloud-hypervisor-memory-mib 4096")
	assert.Contains(t, lockStr, "--cloud-hypervisor-kernel \"${GH_AW_CLOUD_HYPERVISOR_KERNEL}\"")
	assert.Contains(t, lockStr, "--cloud-hypervisor-virtiofsd-sha256 \"${GH_AW_CLOUD_HYPERVISOR_VIRTIOFSD_SHA256}\"")
	assert.Contains(t, lockStr, "--cloud-hypervisor-supervisor-sha256 \"${GH_AW_CLOUD_HYPERVISOR_SUPERVISOR_SHA256}\"")
	assert.Contains(t, lockStr, `\"agentTimeout\":60`)
	assert.Contains(t, lockStr, `\"topologyAttach\":[\"awmg-mcpg\"]`)
	assert.NotContains(t, lockStr, "--mount")
	assert.NotContains(t, lockStr, "--tty")
	assert.NotContains(t, lockStr, "--legacy-security")
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
			script:   "cloud_hypervisor_kvm_access.sh",
			contains: []string{"RUNNER_ENVIRONMENT", "github-hosted", "ImageOS", "setfacl", "u:${runner_uid}:rw", "/dev/kvm", "-c /dev/kvm", "getfacl -ncp /dev/kvm", "-r /dev/kvm", "-w /dev/kvm"},
		},
		{
			script:   "cloud_hypervisor_host_preflight.sh",
			contains: []string{"RUNNER_ENVIRONMENT", "github-hosted", "ImageOS", "/dev/kvm", "test -c /dev/kvm", "cloud-hypervisor preview"},
		},
		{
			script:   "cloud_hypervisor_setup_bundle.sh",
			contains: []string{"cloud-hypervisor-test-x86_64.tar.gz", "cloud-hypervisor-test-x86_64.SHA256SUMS", "cloud-hypervisor-test-x86_64.manifest.json", "archive structure validated", "tar --no-same-owner --no-same-permissions", "validate_extracted_file", "vmlinux.bin", "rootfs.ext4", "awf-supervisor", "virtiofsd", "virtiofsd_path=", "virtiofsd_sha256="},
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
