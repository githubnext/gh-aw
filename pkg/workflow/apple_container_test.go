//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// appleContainerRunsOn is the exact runs-on snippet the compiler accepts for the
// apple-container runtime, rendered the way extractTopLevelYAMLSection stores it.
const appleContainerRunsOn = "runs-on:\n  - self-hosted\n  - macOS\n  - ARM64"

// newAppleContainerWorkflow builds a minimal apple-container workflow that passes
// every check, so each test can mutate exactly one thing.
func newAppleContainerWorkflow() *WorkflowData {
	return &WorkflowData{
		RunsOn:       appleContainerRunsOn,
		EngineConfig: &EngineConfig{ID: "copilot"},
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
			ID:      "awf",
			Runtime: AgentRuntimeAppleContainer,
			Version: string(constants.AWFAppleContainerMinVersion),
		}},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true, Version: string(constants.AWFAppleContainerMinVersion)},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}
}

func TestAppleContainerRuntimeIsSupported(t *testing.T) {
	t.Parallel()

	assert.True(t, isSupportedAgentRuntime(AgentRuntimeAppleContainer))
	assert.Contains(t, supportedAgentRuntimeNames(), "apple-container")
	assert.Equal(t, "apple-container", string(AgentRuntimeAppleContainer))
}

func TestAppleContainerRuntimeProfile(t *testing.T) {
	t.Parallel()

	profile := resolveSandboxRuntimeProfile(&AgentSandboxConfig{Runtime: AgentRuntimeAppleContainer})
	assert.Equal(t, AgentRuntimeAppleContainer, profile.Runtime)
	assert.True(t, profile.NetworkIsolation, "AWF requires strict network isolation for apple-container")
	assert.False(t, profile.LegacySecurity, "apple-container rejects legacy iptables security")
	assert.True(t, profile.Rootless, "AWF itself runs as the runner user")
	assert.Equal(t, constants.AWFDefaultCommand.String(), profile.AWFCommand)
	assert.False(t, profile.SupportsRuntimeInstall, "layer 1 generates no Apple Container provisioning steps")
	assert.False(t, profile.SupportsHostAccess, "the guest has no NIC")
}

func TestAppleContainerRuntimePredicate(t *testing.T) {
	t.Parallel()

	assert.True(t, isAppleContainerRuntime(newAppleContainerWorkflow()))
	assert.False(t, isAppleContainerRuntime(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeDocker}},
	}))
	// A disabled sandbox never selects the runtime.
	assert.False(t, isAppleContainerRuntime(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeAppleContainer, Disabled: true}},
	}))
	assert.False(t, isAppleContainerRuntime(nil))
}

// ── AWF version gating ──────────────────────────────────────────────────────

func TestAppleContainerMinVersionIsAboveDefault(t *testing.T) {
	t.Parallel()

	// The runtime must fail closed on the default AWF version: gh-aw-firewall#7764
	// is not in any published release yet, so a workflow has to opt in explicitly.
	assert.False(t,
		versionAtLeast(string(constants.DefaultFirewallVersion), string(constants.DefaultFirewallVersion), string(constants.AWFAppleContainerMinVersion)),
		"AWFAppleContainerMinVersion must stay above DefaultFirewallVersion until the backend ships")
}

func TestAWFSupportsAppleContainerVersionGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{name: "default version is too old", version: "", expected: false},
		{name: "explicit older version", version: "v0.28.7", expected: false},
		{name: "one patch below minimum", version: "v0.28.8", expected: false},
		{name: "exact minimum", version: string(constants.AWFAppleContainerMinVersion), expected: true},
		{name: "newer minor", version: "v0.29.0", expected: true},
		{name: "latest", version: "latest", expected: true},
		{name: "non-semver branch name is conservative", version: "my-branch", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := awfSupportsAppleContainer(&FirewallConfig{Enabled: true, Version: tt.version})
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAppleContainerRejectedBelowMinVersion(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Version = "v0.28.8"
	workflowData.NetworkPermissions.Firewall.Version = "v0.28.8"

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, string(constants.AWFAppleContainerMinVersion))
	require.ErrorContains(t, err, "apple-container")
}

func TestAppleContainerRejectedOnDefaultAWFVersion(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Version = ""
	workflowData.NetworkPermissions.Firewall.Version = ""

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "the default AWF version must not silently accept apple-container")
	require.ErrorContains(t, err, string(constants.AWFAppleContainerMinVersion))
}

// ── Emitted AWF config ──────────────────────────────────────────────────────

func buildAppleContainerAWFConfigJSON(t *testing.T, workflowData *WorkflowData) string {
	t.Helper()
	workflowData.SandboxConfig = applySandboxDefaults(workflowData.SandboxConfig, workflowData.EngineConfig)
	jsonStr, err := BuildAWFConfigJSON(AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData:   workflowData,
	})
	require.NoError(t, err)
	return jsonStr
}

func TestAppleContainerAWFConfigEmitsBothSelectors(t *testing.T) {
	t.Parallel()

	jsonStr := buildAppleContainerAWFConfigJSON(t, newAppleContainerWorkflow())

	assert.Contains(t, jsonStr, `"containerRuntime":"apple-container"`,
		"AWF selects the backend through container.containerRuntime")
	assert.Contains(t, jsonStr, `"appleContainer":{"previewEnabled":true}`,
		"AWF also requires the explicit appleContainer.previewEnabled opt-in")
}

func TestAppleContainerAWFConfigVersionGated(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Version = "v0.28.8"
	workflowData.NetworkPermissions.Firewall.Version = "v0.28.8"

	jsonStr := buildAppleContainerAWFConfigJSON(t, workflowData)

	assert.NotContains(t, jsonStr, "apple-container",
		"the containerRuntime enum value must never reach an AWF that does not know it")
	assert.NotContains(t, jsonStr, "appleContainer",
		"the appleContainer section must never reach an AWF that does not know it")
}

func TestAppleContainerAWFConfigOmitsTopologyAttach(t *testing.T) {
	t.Parallel()

	jsonStr := buildAppleContainerAWFConfigJSON(t, newAppleContainerWorkflow())

	// AWF fails closed on any non-empty topologyAttach for this runtime.
	assert.NotContains(t, jsonStr, "topologyAttach")
	assert.NotContains(t, jsonStr, "awmg-mcpg")
	assert.Contains(t, jsonStr, `"isolation":true`)
}

func TestAppleContainerTopologyAttachListIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, buildAWFTopologyAttachList(newAppleContainerWorkflow()))
	// Other runtimes keep the existing behaviour.
	assert.Equal(t, []string{"awmg-mcpg"}, buildAWFTopologyAttachList(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeDocker}},
	}))
}

func TestAppleContainerContainerRuntimeString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "apple-container", getAgentContainerRuntime(newAppleContainerWorkflow()))
	// Existing runtimes are unchanged.
	assert.Equal(t, "gvisor", getAgentContainerRuntime(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeGVisor}},
	}))
	assert.Empty(t, getAgentContainerRuntime(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeCloudHypervisor}},
	}))
}

func TestAppleContainerSuppressesTTY(t *testing.T) {
	t.Parallel()

	args := appendTTYAndContainerRuntimeArgs(
		AWFCommandConfig{UsesTTY: true, WorkflowData: newAppleContainerWorkflow()},
		&FirewallConfig{Enabled: true, Version: string(constants.AWFAppleContainerMinVersion)},
	)
	assert.NotContains(t, args, "--tty")
	assert.NotContains(t, args, "--container-runtime",
		"apple-container is selected through the config file, not a CLI flag")
}

// ── Embedded AWF config schema ──────────────────────────────────────────────

func TestValidateAWFConfigJSON_AllowsAppleContainer(t *testing.T) {
	t.Parallel()

	err := validateAWFConfigJSON(`{"container":{"containerRuntime":"apple-container"},"appleContainer":{"previewEnabled":true}}`)
	require.NoError(t, err)
}

func TestValidateAWFConfigJSON_AppleContainerOptionalFields(t *testing.T) {
	t.Parallel()

	err := validateAWFConfigJSON(`{"appleContainer":{"previewEnabled":true,"cpus":8,"memory":"16G","cliPath":"/usr/local/bin/container"}}`)
	require.NoError(t, err)
}

func TestValidateAWFConfigJSON_RejectsInvalidAppleContainerFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config string
	}{
		{name: "unknown property", config: `{"appleContainer":{"previewEnabled":true,"vcpus":4}}`},
		{name: "cpus below minimum", config: `{"appleContainer":{"previewEnabled":true,"cpus":0}}`},
		{name: "cpus not an integer", config: `{"appleContainer":{"previewEnabled":true,"cpus":"4"}}`},
		{name: "memory wrong shape", config: `{"appleContainer":{"previewEnabled":true,"memory":"8 gigabytes"}}`},
		{name: "init image not digest pinned", config: `{"appleContainer":{"previewEnabled":true,"initImage":"ghcr.io/github/gh-aw-firewall/apple-init:v1"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, validateAWFConfigJSON(tt.config))
		})
	}
}

func TestValidateAWFConfigJSON_AllowsAppleInitImageRole(t *testing.T) {
	t.Parallel()

	err := validateAWFConfigJSON(`{"container":{"images":{"appleInit":"ghcr.io/github/gh-aw-firewall/apple-init:v0.28.9@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}}`)
	require.NoError(t, err)
}

// ── container.images appleInit role ─────────────────────────────────────────

func TestAppleContainerRequiresAppleInitImageRole(t *testing.T) {
	t.Parallel()

	assert.True(t, isKnownAWFImageRole(awfImageRoleAppleInit))
	assert.Contains(t, requiredAWFImageRoles(newAppleContainerWorkflow()), awfImageRoleAppleInit)
	assert.NotContains(t, requiredAWFImageRoles(&WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Runtime: AgentRuntimeDocker}},
	}), awfImageRoleAppleInit)
}

func TestAppleContainerManifestMustPinAppleInit(t *testing.T) {
	t.Parallel()

	const pinned = "ghcr.io/github/gh-aw-firewall/%s:v0.28.9@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Images = map[string]string{
		awfImageRoleSquid:    strings.ReplaceAll(pinned, "%s", "squid"),
		awfImageRoleAgent:    strings.ReplaceAll(pinned, "%s", "agent"),
		awfImageRoleAPIProxy: strings.ReplaceAll(pinned, "%s", "api-proxy"),
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, awfImageRoleAppleInit)

	workflowData.SandboxConfig.Agent.Images[awfImageRoleAppleInit] = strings.ReplaceAll(pinned, "%s", "apple-init")
	require.NoError(t, validateSandboxConfig(workflowData))
}

// ── Runner acceptance / rejection matrix ────────────────────────────────────

func TestAppleContainerRunnerMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		labels      []string
		present     bool
		wantErr     bool
		errContains string
	}{
		{name: "canonical self-hosted Apple Silicon", labels: []string{"self-hosted", "macOS", "ARM64"}},
		{name: "lowercase labels", labels: []string{"self-hosted", "macos", "arm64"}},
		{name: "extra dedicated pool label", labels: []string{"self-hosted", "macOS", "ARM64", "apple-container"}},
		{name: "different label order", labels: []string{"ARM64", "macOS", "self-hosted"}},

		{name: "github-hosted macos-latest", labels: []string{"macos-latest"}, wantErr: true, errContains: "macos-latest"},
		{name: "github-hosted macos-26", labels: []string{"macos-26"}, wantErr: true, errContains: "macos-26"},
		{name: "github-hosted macos-15-xlarge", labels: []string{"macos-15-xlarge"}, wantErr: true, errContains: "macos-15-xlarge"},
		{name: "github-hosted label mixed with self-hosted", labels: []string{"self-hosted", "macos-26", "ARM64"}, wantErr: true, errContains: "macos-26"},
		{name: "bare macOS without self-hosted", labels: []string{"macOS", "ARM64"}, wantErr: true},
		{name: "self-hosted macOS without arm64", labels: []string{"self-hosted", "macOS"}, wantErr: true},
		{name: "self-hosted arm64 without macOS", labels: []string{"self-hosted", "ARM64"}, wantErr: true},
		{name: "x64 architecture", labels: []string{"self-hosted", "macOS", "x64"}, wantErr: true, errContains: "x64"},
		{name: "conflicting arm64 and x64", labels: []string{"self-hosted", "macOS", "ARM64", "x64"}, wantErr: true, errContains: "x64"},
		{name: "linux label", labels: []string{"self-hosted", "linux", "ARM64", "macOS"}, wantErr: true, errContains: "linux"},
		{name: "ubuntu-latest", labels: []string{"ubuntu-latest"}, wantErr: true},
		{name: "expression", labels: []string{"${{ vars.RUNNER }}"}, wantErr: true, errContains: "cannot prove"},
		{name: "custom label alone", labels: []string{"self-hosted", "my-mac-pool"}, wantErr: true},
		{name: "runner group only", labels: nil, present: true, wantErr: true, errContains: "runner group"},
		{name: "runs-on omitted", labels: nil, wantErr: true, errContains: "explicit runs-on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			present := tt.present || len(tt.labels) > 0
			err := validateAppleContainerRunnerLabels(tt.labels, present)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.errContains != "" {
				require.ErrorContains(t, err, tt.errContains)
			}
			// Every rejection must name the exact accepted syntax.
			require.ErrorContains(t, err, appleContainerRunnerExample)
		})
	}
}

func TestAppleContainerRunnerValidatedFromWorkflowData(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.RunsOn = "runs-on: macos-26"

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "macos-26")
}

func TestAppleContainerRunnerDefaultUbuntuRejected(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.RunsOn = "runs-on: ubuntu-latest"

	require.Error(t, validateSandboxConfig(workflowData))
}

func TestRunsOnLabelsFromYAMLSection(t *testing.T) {
	t.Parallel()

	labels, present := runsOnLabelsFromYAMLSection(appleContainerRunsOn)
	require.True(t, present)
	assert.Equal(t, []string{"self-hosted", "macOS", "ARM64"}, labels)

	labels, present = runsOnLabelsFromYAMLSection("runs-on: ubuntu-latest")
	require.True(t, present)
	assert.Equal(t, []string{"ubuntu-latest"}, labels)

	labels, present = runsOnLabelsFromYAMLSection("runs-on:\n  group: my-group")
	require.True(t, present)
	assert.Empty(t, labels)

	_, present = runsOnLabelsFromYAMLSection("")
	assert.False(t, present)
}

// TestRunsOnValidationKeepsMacOSBanForOtherRuntimes guards the exemption: only the
// agent job's own runs-on is allowed to be macOS, and only for apple-container.
func TestRunsOnValidationMacOSExemptionIsScoped(t *testing.T) {
	t.Parallel()

	appleFrontmatter := func() map[string]any {
		return map[string]any{
			"sandbox": map[string]any{
				"agent": map[string]any{"runtime": "apple-container"},
			},
			"runs-on": []any{"self-hosted", "macOS", "ARM64"},
		}
	}

	t.Run("accepts the agent runner", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateRunsOn(appleFrontmatter(), "test.md"))
	})

	t.Run("still rejects macOS for runs-on-slim", func(t *testing.T) {
		t.Parallel()
		frontmatter := appleFrontmatter()
		frontmatter["runs-on-slim"] = "macos-latest"
		err := validateRunsOn(frontmatter, "test.md")
		require.Error(t, err)
		require.ErrorContains(t, err, "runs-on-slim")
	})

	t.Run("still rejects macOS for safe-outputs runners", func(t *testing.T) {
		t.Parallel()
		frontmatter := appleFrontmatter()
		frontmatter["safe-outputs"] = map[string]any{"runs-on": "macos-latest"}
		err := validateRunsOn(frontmatter, "test.md")
		require.Error(t, err)
		require.ErrorContains(t, err, "safe-outputs.runs-on")
	})

	t.Run("still rejects macOS without apple-container", func(t *testing.T) {
		t.Parallel()
		err := validateRunsOn(map[string]any{"runs-on": []any{"self-hosted", "macOS", "ARM64"}}, "test.md")
		require.Error(t, err)
		require.ErrorContains(t, err, macOSRunnerFAQURL)
	})

	t.Run("rejects github-hosted macOS with apple-container", func(t *testing.T) {
		t.Parallel()
		frontmatter := appleFrontmatter()
		frontmatter["runs-on"] = "macos-26"
		err := validateRunsOn(frontmatter, "test.md")
		require.Error(t, err)
		require.ErrorContains(t, err, "macos-26")
	})

	// runs-on can arrive from an import, which this validator cannot see. Deferring
	// is safe because validateSandboxConfig re-validates the merged runner.
	t.Run("defers an omitted runs-on to the post-merge check", func(t *testing.T) {
		t.Parallel()
		frontmatter := appleFrontmatter()
		delete(frontmatter, "runs-on")
		require.NoError(t, validateRunsOn(frontmatter, "test.md"))
	})
}

// ── Incompatible feature matrix ─────────────────────────────────────────────

func TestAppleContainerIncompatibleFeatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mutate      func(*WorkflowData)
		errContains string
	}{
		{
			name:        "arc-dind topology",
			mutate:      func(w *WorkflowData) { w.RunnerConfig = &RunnerConfig{Topology: RunnerTopologyArcDind} },
			errContains: "arc-dind",
		},
		{
			name:        "enclaves",
			mutate:      func(w *WorkflowData) { w.Enclaves = []*EnclaveConfig{{Script: &ScriptEnclaveConfig{}}} },
			errContains: "enclaves",
		},
		{
			name:        "volume mounts",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Mounts = []string{"/host:/guest:ro"} },
			errContains: "mounts",
		},
		{
			name: "filesystem allowWrite",
			mutate: func(w *WorkflowData) {
				w.SandboxConfig.Agent.Config = &SandboxRuntimeConfig{
					Filesystem: &SRTFilesystemConfig{AllowWrite: []string{"/workspace"}},
				}
			},
			errContains: "allowWrite",
		},
		{
			name:        "ssl bump",
			mutate:      func(w *WorkflowData) { w.NetworkPermissions.Firewall.SSLBump = true },
			errContains: "ssl_bump",
		},
		{
			name:        "legacy security argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--legacy-security"} },
			errContains: "--legacy-security",
		},
		{
			name:        "host access argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--enable-host-access"} },
			errContains: "--enable-host-access",
		},
		{
			name:        "dns over https argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--dns-over-https"} },
			errContains: "--dns-over-https",
		},
		{
			name:        "topology attach argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--topology-attach=awmg-mcpg"} },
			errContains: "--topology-attach",
		},
		{
			name:        "build local argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--build-local"} },
			errContains: "--build-local",
		},
		{
			name:        "custom agent image argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--agent-image=example/agent:act"} },
			errContains: "--agent-image",
		},
		{
			name:        "sysroot image argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--sysroot-image=example/sysroot"} },
			errContains: "--sysroot-image",
		},
		{
			name:        "chroot binaries source path argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--chroot-binaries-source-path=/opt/bin"} },
			errContains: "--chroot-binaries-source-path",
		},
		{
			name:        "volume argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--volume=/host:/guest"} },
			errContains: "--volume",
		},
		{
			name:        "tty argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--tty"} },
			errContains: "--tty",
		},
		{
			name:        "dind argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--dind"} },
			errContains: "--dind",
		},
		{
			name:        "google api key argument",
			mutate:      func(w *WorkflowData) { w.SandboxConfig.Agent.Args = []string{"--google-api-key"} },
			errContains: "--google-api-key",
		},
		{
			name:        "firewall args are checked too",
			mutate:      func(w *WorkflowData) { w.NetworkPermissions.Firewall.Args = []string{"--ssl-bump"} },
			errContains: "--ssl-bump",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			workflowData := newAppleContainerWorkflow()
			tt.mutate(workflowData)

			err := validateSandboxConfig(workflowData)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.errContains)
			require.ErrorContains(t, err, "apple-container")
		})
	}
}

func TestAppleContainerRejectsHostPortsAndRuntimeInstall(t *testing.T) {
	t.Parallel()

	t.Run("allow-host-ports", func(t *testing.T) {
		t.Parallel()
		workflowData := newAppleContainerWorkflow()
		workflowData.SandboxConfig.Agent.AllowHostPorts = []int{9000}
		err := validateSandboxConfig(workflowData)
		require.Error(t, err)
		require.ErrorContains(t, err, "allow-host-ports")
	})

	t.Run("services with published ports", func(t *testing.T) {
		t.Parallel()
		workflowData := newAppleContainerWorkflow()
		workflowData.ServicePortExpressions = "5432:5432"
		err := validateSandboxConfig(workflowData)
		require.Error(t, err)
		require.ErrorContains(t, err, "services")
	})

	t.Run("runtime-install", func(t *testing.T) {
		t.Parallel()
		workflowData := newAppleContainerWorkflow()
		runtimeInstall := true
		workflowData.SandboxConfig.Agent.RuntimeInstall = &runtimeInstall
		err := validateSandboxConfig(workflowData)
		require.Error(t, err)
		require.ErrorContains(t, err, "runtime-install")
	})
}

// TestAppleContainerBaselineValidates is the positive control for the matrix above.
func TestAppleContainerBaselineValidates(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateSandboxConfig(newAppleContainerWorkflow()))
}

// TestAppleContainerLeavesLinuxRuntimesUnchanged guards against collateral damage
// to the existing runtime profiles.
func TestAppleContainerLeavesLinuxRuntimesUnchanged(t *testing.T) {
	t.Parallel()

	for _, runtime := range []AgentRuntime{AgentRuntimeDocker, AgentRuntimeDockerSudoIptables, AgentRuntimeGVisor} {
		t.Run(string(runtime), func(t *testing.T) {
			t.Parallel()
			workflowData := &WorkflowData{
				RunsOn:        "runs-on: ubuntu-latest",
				EngineConfig:  &EngineConfig{ID: "copilot"},
				SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Runtime: runtime}},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
			}
			require.NoError(t, validateSandboxConfig(workflowData))
		})
	}
}
