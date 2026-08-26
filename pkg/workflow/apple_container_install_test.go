//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stepText joins a generated step's lines so tests can assert on the rendered
// YAML rather than on line indices, which shift whenever a comment is added.
func stepText(step GitHubActionStep) string {
	return strings.Join([]string(step), "\n")
}

// ── Setup step generation ───────────────────────────────────────────────────

// newAppleContainerWorkflowWithPinnedImages returns a baseline whose AWF image
// manifest is fully digest-pinned, which is what the image pull step requires.
func newAppleContainerWorkflowWithPinnedImages() *WorkflowData {
	const pinned = "ghcr.io/github/gh-aw-firewall/%s:v0.28.9@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Images = map[string]string{
		awfImageRoleSquid:     strings.ReplaceAll(pinned, "%s", "squid"),
		awfImageRoleAgent:     strings.ReplaceAll(pinned, "%s", "agent"),
		awfImageRoleAPIProxy:  strings.ReplaceAll(pinned, "%s", "api-proxy"),
		awfImageRoleAppleInit: strings.ReplaceAll(pinned, "%s", "apple-init"),
	}
	return workflowData
}

func TestAppleContainerSetupStepsOrdering(t *testing.T) {
	t.Parallel()

	steps := generateAppleContainerSetupSteps(newAppleContainerWorkflowWithPinnedImages())
	require.Len(t, steps, 4, "preflight, CLI setup, service start, image pull")

	// Order is a correctness property, not a preference: an ineligible host must
	// be rejected before anything is downloaded, the CLI must exist before the
	// services start, and the services must be up before any image is pulled.
	assert.Contains(t, stepText(steps[0]), "apple_container_host_preflight.sh")
	assert.Contains(t, stepText(steps[1]), "apple_container_setup_cli.sh")
	assert.Contains(t, stepText(steps[2]), "apple_container_start_services.sh")
	assert.Contains(t, stepText(steps[3]), "apple_container_pull_images.sh")
}

// TestAppleContainerSetupStepsOmitPullWithoutPinnedImages documents that the pull
// step is skipped when no digest-pinned manifest is available. AWF's manifest
// validation is the right place to report a floating reference; emitting a pull
// step that is guaranteed to fail would only bury that diagnostic.
func TestAppleContainerSetupStepsOmitPullWithoutPinnedImages(t *testing.T) {
	t.Parallel()

	steps := generateAppleContainerSetupSteps(newAppleContainerWorkflow())
	require.Len(t, steps, 3)
	for _, step := range steps {
		assert.NotContains(t, stepText(step), "apple_container_pull_images.sh")
	}
}

func TestAppleContainerSetupStepsSkippedForOtherRuntimes(t *testing.T) {
	t.Parallel()

	for _, runtime := range []AgentRuntime{
		AgentRuntimeDocker,
		AgentRuntimeDockerSudoIptables,
		AgentRuntimeGVisor,
		AgentRuntimeDockerSbx,
		AgentRuntimeCloudHypervisor,
	} {
		t.Run(string(runtime), func(t *testing.T) {
			t.Parallel()
			workflowData := newAppleContainerWorkflow()
			workflowData.SandboxConfig.Agent.Runtime = runtime
			assert.Empty(t, generateAppleContainerSetupSteps(workflowData),
				"apple-container provisioning must not leak into other runtimes")
		})
	}

	assert.Empty(t, generateAppleContainerSetupSteps(nil))
}

func TestAppleContainerHostPreflightStep(t *testing.T) {
	t.Parallel()

	text := stepText(generateAppleContainerHostPreflightStep())
	assert.Contains(t, text, "apple_container_host_preflight.sh")
	assert.Contains(t, text, "GH_AW_APPLE_CONTAINER_MIN_MACOS: 26",
		"the macOS floor must reach the script rather than being hardcoded twice")
}

func TestAppleContainerCLISetupStepPinsVersionAndDigest(t *testing.T) {
	t.Parallel()

	text := stepText(generateAppleContainerCLISetupStep(false))

	assert.Contains(t, text, "GH_AW_APPLE_CONTAINER_VERSION: "+constants.DefaultAppleContainerVersion)
	assert.Contains(t, text, "GH_AW_APPLE_CONTAINER_PKG_SHA256: "+constants.DefaultAppleContainerPkgSHA256)
	assert.Contains(t, text, constants.AppleContainerPkgSigningIdentity)
	assert.Contains(t, text, "GH_AW_APPLE_CONTAINER_MIN_CLI: "+constants.AppleContainerMinCLIVersion)
	assert.Contains(t, text, "GH_AW_APPLE_CONTAINER_MAX_CLI: "+constants.AppleContainerMaxCLIVersionExclusive)

	// "latest" must never appear: an unpinned installer would defeat both the
	// digest pin and AWF's CLI version contract.
	assert.NotContains(t, text, "latest")
}

func TestAppleContainerCLISetupStepGatesInstallOnRuntimeInstall(t *testing.T) {
	t.Parallel()

	assert.NotContains(t, stepText(generateAppleContainerCLISetupStep(false)), "--allow-install",
		"without runtime-install the step only verifies a preinstalled CLI")
	assert.Contains(t, stepText(generateAppleContainerCLISetupStep(true)), "--allow-install")
}

// TestAppleContainerSetupStepsHonourRuntimeInstall pins the default. Like gVisor
// and docker-sbx, provisioning is on unless the workflow opts out, and the script
// still prefers a compatible preinstalled CLI over installing anything.
func TestAppleContainerSetupStepsHonourRuntimeInstall(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	assert.Contains(t, stepText(generateAppleContainerSetupSteps(workflowData)[1]), "--allow-install",
		"runtime-install defaults to enabled")

	runtimeInstall := false
	workflowData.SandboxConfig.Agent.RuntimeInstall = &runtimeInstall
	assert.NotContains(t, stepText(generateAppleContainerSetupSteps(workflowData)[1]), "--allow-install",
		"runtime-install: false must reduce the step to verification only")
}

func TestAppleContainerTeardownStepAlwaysRuns(t *testing.T) {
	t.Parallel()

	text := stepText(generateAppleContainerTeardownStep())
	assert.Contains(t, text, "apple_container_teardown.sh")
	// A cancelled or failed job must still release the guest and the run-scoped
	// state on a persistent runner, and teardown must never rewrite the agent's
	// own outcome.
	assert.Contains(t, text, "if: always()")
	assert.Contains(t, text, "continue-on-error: true")
}

// ── Image preparation ───────────────────────────────────────────────────────

func TestAppleContainerRuntimeImagesUseDigestPins(t *testing.T) {
	t.Parallel()

	const pinned = "ghcr.io/github/gh-aw-firewall/%s:v0.28.9@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Images = map[string]string{
		awfImageRoleSquid:     strings.ReplaceAll(pinned, "%s", "squid"),
		awfImageRoleAgent:     strings.ReplaceAll(pinned, "%s", "agent"),
		awfImageRoleAPIProxy:  strings.ReplaceAll(pinned, "%s", "api-proxy"),
		awfImageRoleAppleInit: strings.ReplaceAll(pinned, "%s", "apple-init"),
	}

	images := appleContainerRuntimeImages(workflowData)
	require.Len(t, images, 2, "only the agent and apple-init images enter the Apple Container store")
	assert.Contains(t, images, strings.ReplaceAll(pinned, "%s", "agent"))
	assert.Contains(t, images, strings.ReplaceAll(pinned, "%s", "apple-init"))

	// Squid and the API proxy stay in Docker: they are infrastructure the guest
	// never runs, and pulling them into the Apple store would waste time and
	// disk for nothing.
	assert.NotContains(t, images, strings.ReplaceAll(pinned, "%s", "squid"))
	assert.NotContains(t, images, strings.ReplaceAll(pinned, "%s", "api-proxy"))
}

func TestAppleContainerRuntimeImagesDropFloatingReferences(t *testing.T) {
	t.Parallel()

	// The default manifest is tag-based, not digest-pinned. AWF refuses a floating
	// reference for this runtime, so the compiler must not hand one to the pull
	// script either: the manifest error is the correct diagnostic, not a pull
	// failure inside a VM.
	images := appleContainerRuntimeImages(newAppleContainerWorkflow())
	for _, image := range images {
		assert.Contains(t, image, "@sha256:", "every pulled reference must be digest-pinned")
	}

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Images = map[string]string{
		awfImageRoleAgent:     "ghcr.io/github/gh-aw-firewall/agent:v0.28.9",
		awfImageRoleAppleInit: "ghcr.io/github/gh-aw-firewall/apple-init:v0.28.9",
	}
	assert.Empty(t, appleContainerRuntimeImages(workflowData),
		"floating references must be dropped rather than pulled")
}

func TestAppleContainerRuntimeImagesEmptyForOtherRuntimes(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Runtime = AgentRuntimeDocker
	assert.Empty(t, appleContainerRuntimeImages(workflowData))
}

func TestAppleContainerImagePullStepQuotesReferences(t *testing.T) {
	t.Parallel()

	const reference = "ghcr.io/github/gh-aw-firewall/agent:v1@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	text := stepText(generateAppleContainerImagePullStep([]string{reference}))
	assert.Contains(t, text, "apple_container_pull_images.sh")
	assert.Contains(t, text, "'"+reference+"'", "references are single-quoted in the generated run: line")

	assert.Empty(t, generateAppleContainerImagePullStep(nil), "no images means no step")
}

// TestAppleContainerAppleInitStaysOutOfDockerPredownload proves the two image
// stores stay separate in both directions: Docker never pre-pulls apple-init, and
// the Apple store never receives the infrastructure images.
func TestAppleContainerAppleInitStaysOutOfDockerPredownload(t *testing.T) {
	t.Parallel()

	const pinned = "ghcr.io/github/gh-aw-firewall/%s:v0.28.9@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.Agent.Images = map[string]string{
		awfImageRoleSquid:     strings.ReplaceAll(pinned, "%s", "squid"),
		awfImageRoleAgent:     strings.ReplaceAll(pinned, "%s", "agent"),
		awfImageRoleAPIProxy:  strings.ReplaceAll(pinned, "%s", "api-proxy"),
		awfImageRoleAppleInit: strings.ReplaceAll(pinned, "%s", "apple-init"),
	}

	dockerImages := collectDockerImages(workflowData.Tools, workflowData, ActionModeScript)
	for _, image := range dockerImages {
		assert.NotContains(t, image, "apple-init",
			"docker pull cannot populate Apple Container's store, so apple-init must be excluded")
	}
	// The infrastructure images are still pre-pulled for Docker; only the agent
	// crosses the hypervisor boundary.
	assert.Contains(t, dockerImages, strings.ReplaceAll(pinned, "%s", "squid"))
}

// ── Pinned Apple Container release ──────────────────────────────────────────

func TestAppleContainerPinnedVersionIsInsideAWFContractRange(t *testing.T) {
	t.Parallel()

	// AWF's init-image contract is >= 0.4.0 and < 1.0.0. apple/container has a 1.x
	// line, and picking the newest release without checking would silently land
	// outside the window: a major bump may relocate the real vminitd inside the
	// init image and boot a guest with no capability relay.
	assert.Equal(t, "0.4.0", constants.AppleContainerMinCLIVersion)
	assert.Equal(t, "1.0.0", constants.AppleContainerMaxCLIVersionExclusive)
	assert.True(t, strings.HasPrefix(constants.DefaultAppleContainerVersion, "0."),
		"the pinned release must be a 0.x release to stay inside AWF's contract range")
	assert.NotContains(t, constants.DefaultAppleContainerVersion, "latest")
}

func TestAppleContainerPkgDigestIsAFullSHA256(t *testing.T) {
	t.Parallel()

	digest := constants.DefaultAppleContainerPkgSHA256
	assert.Len(t, digest, 64, "a SHA-256 hex digest is 64 characters")
	assert.Equal(t, strings.ToLower(digest), digest, "digests are compared lowercased")
	for _, r := range digest {
		assert.Contains(t, "0123456789abcdef", string(r), "digest must be lowercase hex")
	}
}

func TestAppleContainerSigningIdentityIsAppleContainerization(t *testing.T) {
	t.Parallel()

	// The digest proves the bytes; the identity proves the signer. Losing the
	// identity check would let a checksum bump alone launder a package from
	// somebody else's Developer ID.
	assert.Contains(t, constants.AppleContainerPkgSigningIdentity, "Developer ID Installer")
	assert.Contains(t, constants.AppleContainerPkgSigningIdentity, "Apple Inc. - Containerization")
	assert.Equal(t, "com.apple.container-installer", constants.AppleContainerPkgIdentifier)
}

// ── No topology, no guest network ───────────────────────────────────────────

// TestAppleContainerGeneratesNoTopologyAttach re-asserts the layer 1 invariant
// against the layer 2 code paths: nothing added here may reintroduce a
// Docker-network peer for the guest.
func TestAppleContainerGeneratesNoTopologyAttach(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	assert.Empty(t, buildAWFTopologyAttachList(workflowData))

	for _, step := range generateAppleContainerSetupSteps(newAppleContainerWorkflowWithPinnedImages()) {
		text := stepText(step)
		for _, forbidden := range []string{
			"--topology-attach",
			"topologyAttach",
			"--net=host",
			"--network host",
			"/var/run/docker.sock",
			"DOCKER_HOST",
		} {
			assert.NotContains(t, text, forbidden,
				"apple-container provisioning must not grant the guest a network route or a Docker socket")
		}
	}

	assert.NotContains(t, stepText(generateAppleContainerTeardownStep()), "/var/run/docker.sock")
}

// ── Runtime profile ─────────────────────────────────────────────────────────

func TestAppleContainerProfileSupportsRuntimeInstall(t *testing.T) {
	t.Parallel()

	profile := sandboxRuntimeProfiles[AgentRuntimeAppleContainer]
	assert.True(t, profile.SupportsRuntimeInstall)
	assert.True(t, profile.Rootless, "AWF runs as the runner user; 'container' is unprivileged")
	assert.False(t, profile.SupportsHostAccess, "the guest has no NIC")

	// No other runtime's provisioning support may change as a side effect.
	assert.True(t, sandboxRuntimeProfiles[AgentRuntimeGVisor].SupportsRuntimeInstall)
	assert.True(t, sandboxRuntimeProfiles[AgentRuntimeDockerSbx].SupportsRuntimeInstall)
	assert.False(t, sandboxRuntimeProfiles[AgentRuntimeDocker].SupportsRuntimeInstall)
	assert.False(t, sandboxRuntimeProfiles[AgentRuntimeCloudHypervisor].SupportsRuntimeInstall)
}

// ── Shell quoting ───────────────────────────────────────────────────────────

func TestShellQuoteForRun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "'plain'", shellQuoteForRun("plain"))
	assert.Equal(t, `'it'\''s'`, shellQuoteForRun("it's"))
	// A value carrying shell metacharacters must stay a single literal argument.
	assert.Equal(t, "'; rm -rf /'", shellQuoteForRun("; rm -rf /"))
	assert.Equal(t, "'$(whoami)'", shellQuoteForRun("$(whoami)"))
}
