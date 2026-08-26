// This file implements the compile-time contract for the apple-container sandbox
// runtime (AWF's Apple Virtualization.framework workload preview,
// gh-aw-firewall#7764).
//
// The runtime is only viable on a self-hosted bare-metal Apple Silicon runner
// (Darwin arm64, macOS 26+, kern.hv_support=1). GitHub-hosted macOS runners are
// themselves virtual machines with nested virtualisation unavailable, so they can
// never host the guest and must be rejected at compile time rather than failing
// late on the runner.
//
// gh-aw cannot inspect the runner, so it fails closed: only an explicitly
// self-hosted Apple Silicon label set is accepted. Anything the compiler cannot
// prove — a bare runner group, a GitHub Actions expression, a custom label, or a
// GitHub-hosted macos-* label — is an error that names the exact required syntax.

package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"gopkg.in/yaml.v3"
)

var appleContainerLog = logger.New("workflow:apple_container")

// Runner labels that together identify a self-hosted bare-metal Apple Silicon
// runner. These are the default labels the GitHub Actions runner applies on
// Darwin arm64, so they are recognisable without guessing.
const (
	appleContainerSelfHostedLabel = "self-hosted"
	appleContainerMacOSLabel      = "macos"
	appleContainerARM64Label      = "arm64"
)

// appleContainerRunnerExample is the exact runs-on syntax the compiler accepts.
// It is repeated in every error message so authors never have to guess.
const appleContainerRunnerExample = "runs-on: [self-hosted, macOS, ARM64]"

// appleContainerRejectedArchLabels are architecture labels that contradict the
// native arm64 requirement. Rosetta translation is refused by AWF, so an x86_64
// runner can never satisfy the runtime.
var appleContainerRejectedArchLabels = []string{"x64", "x86", "x86_64", "amd64"}

// appleContainerRejectedOSLabels are OS labels that contradict the macOS
// requirement.
var appleContainerRejectedOSLabels = []string{"linux", "windows"}

// runsOnLabelsFromYAMLSection extracts runner labels from a rendered runs-on YAML
// snippet such as the value stored in WorkflowData.RunsOn.
//
// It returns the labels and whether a runs-on value was present at all. Parsing
// failures are reported as "no labels" so callers fail closed.
func runsOnLabelsFromYAMLSection(section string) (labels []string, present bool) {
	trimmed := strings.TrimSpace(section)
	if trimmed == "" {
		return nil, false
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(trimmed), &parsed); err != nil {
		appleContainerLog.Printf("Failed to parse runs-on snippet: %v", err)
		return nil, true
	}
	value, ok := parsed["runs-on"]
	if !ok || value == nil {
		return nil, false
	}
	return extractRunnerLabels(value), true
}

// isGitHubHostedMacOSLabel reports whether a label selects a GitHub-hosted macOS
// image. Every GitHub-hosted macOS label is of the form "macos-<something>"
// (macos-latest, macos-15, macos-26, macos-15-xlarge, ...). The bare "macOS"
// label is the self-hosted runner's OS label and is not GitHub-hosted.
func isGitHubHostedMacOSLabel(label string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(label)), "macos-")
}

// hasRunnerLabel reports whether labels contains target (case-insensitive, after
// trimming surrounding whitespace).
func hasRunnerLabel(labels []string, target string) bool {
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), target) {
			return true
		}
	}
	return false
}

// isAppleContainerRunnerLabelSet reports whether a label set explicitly proves a
// self-hosted bare-metal Apple Silicon runner.
//
// Additional custom labels are allowed (a dedicated pool label is encouraged), but
// no GitHub-hosted macos-* label and no contradicting OS/arch label may be present.
func isAppleContainerRunnerLabelSet(labels []string) bool {
	if !hasRunnerLabel(labels, appleContainerSelfHostedLabel) ||
		!hasRunnerLabel(labels, appleContainerMacOSLabel) ||
		!hasRunnerLabel(labels, appleContainerARM64Label) {
		return false
	}
	for _, label := range labels {
		if isGitHubHostedMacOSLabel(label) {
			return false
		}
		if githubActionsExpressionPattern.MatchString(label) {
			return false
		}
	}
	for _, rejected := range appleContainerRejectedArchLabels {
		if hasRunnerLabel(labels, rejected) {
			return false
		}
	}
	for _, rejected := range appleContainerRejectedOSLabels {
		if hasRunnerLabel(labels, rejected) {
			return false
		}
	}
	return true
}

// appleContainerRunnerError builds the shared fail-closed runner error.
func appleContainerRunnerError(value, message string) error {
	return NewValidationError(
		"runs-on",
		value,
		message,
		fmt.Sprintf(
			"%s requires a self-hosted bare-metal Apple Silicon runner (Darwin arm64, macOS 26+, kern.hv_support=1). "+
				"GitHub-hosted macOS runners are virtual machines without nested virtualisation and can never host the guest.\n\n"+
				"Declare the runner explicitly:\n\n%s\n\n"+
				"Extra pool labels may be added (for example [self-hosted, macOS, ARM64, apple-container]), but the "+
				"self-hosted, macOS and ARM64 labels must all be present and no GitHub-hosted macos-* label may be used.\n\n"+
				"See: %s",
			AgentRuntimeAppleContainer, appleContainerRunnerExample, constants.DocsSandboxURL),
	)
}

// validateAppleContainerRunnerLabels validates a resolved runs-on label set for
// the apple-container runtime. present reports whether the workflow declared a
// runs-on value at all; an omitted runs-on defaults to ubuntu-latest and is
// therefore rejected.
func validateAppleContainerRunnerLabels(labels []string, present bool) error {
	if !present {
		return appleContainerRunnerError("(not set)",
			fmt.Sprintf("sandbox.agent.runtime: %s requires an explicit runs-on runner declaration", AgentRuntimeAppleContainer))
	}
	if len(labels) == 0 {
		return appleContainerRunnerError("(no labels)",
			fmt.Sprintf("runs-on must list explicit runner labels for sandbox.agent.runtime: %s; a runner group alone cannot prove a self-hosted Apple Silicon host", AgentRuntimeAppleContainer))
	}

	for _, label := range labels {
		if isGitHubHostedMacOSLabel(label) {
			return appleContainerRunnerError(label,
				fmt.Sprintf("GitHub-hosted macOS runner '%s' cannot run sandbox.agent.runtime: %s", label, AgentRuntimeAppleContainer))
		}
	}
	for _, label := range labels {
		if githubActionsExpressionPattern.MatchString(label) {
			return appleContainerRunnerError(label,
				fmt.Sprintf("runs-on expression '%s' cannot prove a self-hosted Apple Silicon host for sandbox.agent.runtime: %s", label, AgentRuntimeAppleContainer))
		}
	}
	for _, rejected := range appleContainerRejectedArchLabels {
		if hasRunnerLabel(labels, rejected) {
			return appleContainerRunnerError(strings.Join(labels, ", "),
				fmt.Sprintf("runner label '%s' contradicts the native arm64 requirement of sandbox.agent.runtime: %s (Rosetta translation is refused)", rejected, AgentRuntimeAppleContainer))
		}
	}
	for _, rejected := range appleContainerRejectedOSLabels {
		if hasRunnerLabel(labels, rejected) {
			return appleContainerRunnerError(strings.Join(labels, ", "),
				fmt.Sprintf("runner label '%s' contradicts the macOS requirement of sandbox.agent.runtime: %s", rejected, AgentRuntimeAppleContainer))
		}
	}

	if !isAppleContainerRunnerLabelSet(labels) {
		return appleContainerRunnerError(strings.Join(labels, ", "),
			fmt.Sprintf("runs-on does not prove a self-hosted Apple Silicon host for sandbox.agent.runtime: %s", AgentRuntimeAppleContainer))
	}

	appleContainerLog.Printf("apple-container runner accepted: %s", strings.Join(labels, ", "))
	return nil
}

// sandboxAgentRuntimeFromFrontmatter reads sandbox.agent.runtime from raw
// frontmatter. It returns an empty string when the path is absent or not a
// string, which keeps every non-apple-container workflow on the existing rules.
func sandboxAgentRuntimeFromFrontmatter(frontmatter map[string]any) string {
	sandbox, ok := frontmatter["sandbox"].(map[string]any)
	if !ok {
		return ""
	}
	agent, ok := sandbox["agent"].(map[string]any)
	if !ok {
		return ""
	}
	runtime, ok := agent["runtime"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(runtime)
}

// frontmatterSelectsAppleContainer reports whether raw frontmatter selects the
// apple-container sandbox runtime.
func frontmatterSelectsAppleContainer(frontmatter map[string]any) bool {
	return sandboxAgentRuntimeFromFrontmatter(frontmatter) == string(AgentRuntimeAppleContainer)
}

// appleContainerRejectedAWFArgs maps raw AWF arguments that apple-container
// refuses to the reason AWF gives. gh-aw surfaces these at compile time so the
// workflow never reaches a runner that will fail closed anyway.
var appleContainerRejectedAWFArgs = []struct {
	arg    string
	reason string
}{
	{"--legacy-security", "the guest has no NIC, so host and container iptables rules govern nothing"},
	{"--enable-host-access", "only allowlisted AWF capability sockets cross the VM boundary"},
	{"--allow-host-ports", "only allowlisted AWF capability sockets cross the VM boundary"},
	{"--allow-host-service-ports", "only allowlisted AWF capability sockets cross the VM boundary"},
	{"--dns-over-https", "the guest resolves no names at all and reaches every destination through the Squid capability"},
	{"--topology-attach", "externally owned peers are not published to macOS loopback and cannot be bridged into the guest"},
	{"--dind", "the guest never receives a Docker socket"},
	{"--docker-host-path-prefix", "the guest never receives a Docker socket"},
	{"--build-local", "the agent image is pulled through the Apple Container image store, not Docker's"},
	{"--agent-image", "only the default agent image is published as native arm64 and Rosetta translation is refused"},
	{"--sysroot-image", "the guest runs the agent image root filesystem directly and does not use the chroot sysroot"},
	{"--chroot-binaries-source-path", "the guest runs the agent image root filesystem directly and does not use the chroot sysroot"},
	{"--volume", "only the workspace and AWF-owned run directories are exposed to the guest"},
	{"--tty", "the Apple Container preview does not allocate a pseudo-TTY"},
	{"--ssl-bump", "SSL bump requires a locally built Squid image and a guest trust store AWF does not manage here"},
	{"--google-api-key", "the Vertex provider port is not part of the capability transport allowlist"},
	{"--no-network-isolation", "the Apple Container preview requires strict network isolation"},
}

// appleContainerIncompatibility builds a consistent incompatibility error keyed on
// sandbox.agent.runtime, matching the docker-sbx/cloud-hypervisor error shape.
func appleContainerIncompatibility(reason, suggestion string) error {
	return NewValidationError(
		"sandbox.agent.runtime",
		string(AgentRuntimeAppleContainer),
		reason,
		suggestion,
	)
}

// validateAppleContainerCompatibility enforces the full apple-container contract:
// the AWF version gate, the runner requirement, and every workflow feature AWF
// refuses under this runtime. It is a no-op for all other runtimes.
func validateAppleContainerCompatibility(workflowData *WorkflowData, agentConfig *AgentSandboxConfig) error {
	if agentConfig == nil || agentConfig.Disabled || agentConfig.Runtime != AgentRuntimeAppleContainer {
		return nil
	}

	// AWF version gate. This is checked first so an unsupported pin is reported
	// before any feature-level detail.
	firewallConfig := getFirewallConfig(workflowData)
	if !awfSupportsAppleContainer(firewallConfig) {
		effectiveVersion := string(constants.DefaultFirewallVersion)
		if firewallConfig != nil && firewallConfig.Version != "" {
			effectiveVersion = firewallConfig.Version
		}
		return appleContainerIncompatibility(
			fmt.Sprintf("apple-container requires AWF %s or newer", constants.AWFAppleContainerMinVersion),
			fmt.Sprintf("apple-container preview support (container.containerRuntime: apple-container plus appleContainer.previewEnabled) is only available in AWF %s+.\n\nThe effective AWF version is %s. Set sandbox.agent.version (or firewall.version) to %s or newer.", constants.AWFAppleContainerMinVersion, effectiveVersion, constants.AWFAppleContainerMinVersion),
		)
	}

	// Runner requirement. validateRunsOn already checks the raw frontmatter; this
	// re-checks the resolved workflow so directly constructed WorkflowData and any
	// future runs-on source cannot bypass the host requirement.
	if workflowData != nil {
		labels, present := runsOnLabelsFromYAMLSection(workflowData.RunsOn)
		if err := validateAppleContainerRunnerLabels(labels, present); err != nil {
			return err
		}
	}

	if isArcDindTopology(workflowData) {
		return appleContainerIncompatibility(
			"apple-container is incompatible with runner.topology: arc-dind",
			"apple-container requires a self-hosted bare-metal Apple Silicon runner with a local Unix-socket Docker daemon. "+
				"ARC DinD runners are Linux, use a split runner/daemon filesystem, and never give the guest a Docker socket. "+
				"Remove sandbox.agent.runtime: apple-container or change runner.topology.",
		)
	}

	if workflowData != nil && len(workflowData.Enclaves) > 0 {
		return appleContainerIncompatibility(
			"apple-container is incompatible with enclaves",
			"apple-container does not yet support the enclaves subsystem: the enclave MCP gateway is a Docker-network peer "+
				"that a NIC-less guest cannot reach. Remove the enclaves configuration, or change sandbox.agent.runtime.",
		)
	}

	if len(agentConfig.Mounts) > 0 {
		return appleContainerIncompatibility(
			"apple-container is incompatible with sandbox.agent.mounts",
			"apple-container exposes only the workspace and AWF-owned run directories to the guest; arbitrary volume mounts "+
				"are refused. Remove sandbox.agent.mounts, or change sandbox.agent.runtime.",
		)
	}

	if agentConfig.Config != nil && agentConfig.Config.Filesystem != nil && len(agentConfig.Config.Filesystem.AllowWrite) > 0 {
		return appleContainerIncompatibility(
			"apple-container is incompatible with sandbox.agent.config.filesystem.allowWrite",
			"AWF fails closed with \"filesystem.allowWrite is not yet supported by the apple-container runtime\". "+
				"Remove the allowWrite policy, or change sandbox.agent.runtime.",
		)
	}

	if firewallConfig != nil && firewallConfig.SSLBump {
		return appleContainerIncompatibility(
			"apple-container is incompatible with firewall ssl_bump",
			"SSL bump requires a locally built Squid image and a guest trust store that AWF does not manage for this runtime. "+
				"Remove ssl_bump (and allow_urls), or change sandbox.agent.runtime.",
		)
	}

	if isGeminiVertexWIF(workflowData) {
		return appleContainerIncompatibility(
			"apple-container is incompatible with Google Vertex AI credential isolation",
			"The Vertex provider port is not part of the apple-container capability transport allowlist. "+
				"Use a non-Vertex Gemini configuration, or change sandbox.agent.runtime.",
		)
	}

	args := customAWFArgs(workflowData)
	for _, rejected := range appleContainerRejectedAWFArgs {
		if hasEnabledAWFArg(args, rejected.arg) {
			return appleContainerIncompatibility(
				"apple-container does not support the AWF argument "+rejected.arg,
				fmt.Sprintf("AWF refuses %s under the apple-container runtime: %s.\n\nRemove the argument from sandbox.agent.args (or firewall.args), or change sandbox.agent.runtime.\n\nSee: %s", rejected.arg, rejected.reason, constants.DocsSandboxURL),
			)
		}
	}

	appleContainerLog.Print("apple-container runtime configured -- AWF version, runner, and feature compatibility checks passed")
	return nil
}
