// This file generates the GitHub Actions steps that provision AWF's
// apple-container runtime on a self-hosted bare-metal Apple Silicon runner.
//
// The step sequence mirrors the gVisor / docker-sbx / cloud-hypervisor pattern —
// preflight, install, activate, verify — but everything it emits has to be valid
// on macOS with BSD userland and bash 3.2, which is what /bin/bash is there.
//
// Ordering is load-bearing and matches the dependency chain exactly:
//
//  1. Host preflight   — refuse an ineligible runner before anything is fetched.
//  2. CLI setup        — verify a preinstalled `container`, or install the
//                        pinned, checksum- and signature-verified package.
//  3. Service start    — start the API server, install the default kernel
//                        non-interactively, and pin one application root for the
//                        rest of the job.
//  4. Image pull       — populate Apple Container's own store, which the Docker
//                        pre-download earlier in the job cannot reach.
//
// Steps 1–3 run before the AWF binary is installed, so an unusable host costs
// seconds. The image pull runs last because it is the only step that needs a
// live service.

package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var appleContainerInstallLog = logger.New("workflow:apple_container_install")

// generateAppleContainerHostPreflightStep validates runner eligibility: a
// self-hosted Apple Silicon Mac on a supported macOS with Virtualization.framework
// available, a usable launchd user domain, and a working Docker daemon for the
// infrastructure containers.
func generateAppleContainerHostPreflightStep() GitHubActionStep {
	appleContainerInstallLog.Print("Generating apple-container host preflight step")
	return GitHubActionStep([]string{
		"      - name: Check host eligibility for apple-container",
		"        env:",
		"          GH_AW_APPLE_CONTAINER_MIN_MACOS: " + strconv.Itoa(constants.AppleContainerMinMacOSMajor),
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/apple_container_host_preflight.sh"`,
	})
}

// generateAppleContainerCLISetupStep verifies or installs the Apple `container`
// CLI.
//
// allowInstall reflects sandbox.agent.runtime-install. When it is false the step
// only verifies what the runner image already provides and fails with the exact
// version required; nothing is downloaded and no sudo is used. When it is true
// the step may install the pinned release, which is verified by SHA-256 and by
// its Apple Developer ID installer signature before `installer` runs.
func generateAppleContainerCLISetupStep(allowInstall bool) GitHubActionStep {
	appleContainerInstallLog.Printf("Generating apple-container CLI setup step (version %s, allowInstall=%t)", constants.DefaultAppleContainerVersion, allowInstall)

	runArgs := ""
	if allowInstall {
		runArgs = " --allow-install"
	}

	return GitHubActionStep([]string{
		"      - name: Set up Apple container CLI",
		"        env:",
		"          GH_AW_APPLE_CONTAINER_VERSION: " + constants.DefaultAppleContainerVersion,
		"          GH_AW_APPLE_CONTAINER_PKG_SHA256: " + constants.DefaultAppleContainerPkgSHA256,
		"          GH_AW_APPLE_CONTAINER_PKG_IDENTITY: " + strconv.Quote(constants.AppleContainerPkgSigningIdentity),
		"          GH_AW_APPLE_CONTAINER_MIN_CLI: " + constants.AppleContainerMinCLIVersion,
		"          GH_AW_APPLE_CONTAINER_MAX_CLI: " + constants.AppleContainerMaxCLIVersionExclusive,
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/apple_container_setup_cli.sh"` + runArgs,
	})
}

// generateAppleContainerServicesStep starts the Apple Container system services
// and exports the application root every later `container` invocation must use.
func generateAppleContainerServicesStep() GitHubActionStep {
	appleContainerInstallLog.Print("Generating apple-container service start step")
	return GitHubActionStep([]string{
		"      - name: Start Apple container services",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/apple_container_start_services.sh"`,
	})
}

// generateAppleContainerImagePullStep pulls the digest-pinned images into Apple
// Container's store. It returns an empty step when there is nothing to pull, so
// a workflow without a resolvable manifest does not emit a no-op step.
func generateAppleContainerImagePullStep(images []string) GitHubActionStep {
	if len(images) == 0 {
		appleContainerInstallLog.Print("No apple-container images to pull; skipping step")
		return GitHubActionStep([]string{})
	}

	appleContainerInstallLog.Printf("Generating apple-container image pull step for %d image(s)", len(images))

	var run strings.Builder
	run.WriteString(`        run: bash "${RUNNER_TEMP}/gh-aw/actions/apple_container_pull_images.sh"`)
	for _, image := range images {
		run.WriteString(" ")
		run.WriteString(shellQuoteForRun(image))
	}

	return GitHubActionStep([]string{
		"      - name: Pull Apple Container images",
		run.String(),
	})
}

// generateAppleContainerTeardownStep stops the guest and the system services and
// removes run-scoped state.
//
// It runs with if: always() and continue-on-error so a cancelled job still
// releases the runner, and so a teardown problem never rewrites the agent's own
// outcome.
func generateAppleContainerTeardownStep() GitHubActionStep {
	appleContainerInstallLog.Print("Generating apple-container teardown step")
	return GitHubActionStep([]string{
		"      - name: Tear down Apple Container",
		"        if: always()",
		"        continue-on-error: true",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/apple_container_teardown.sh"`,
	})
}

// appleContainerRuntimeImages returns the digest-pinned references that must
// exist in Apple Container's image store: the agent image the workload runs, and
// the apple-init image carrying the guest capability relay.
//
// It intentionally reuses the same resolution the AWF config emits, so the
// compiler cannot pull one reference while telling AWF to verify another. A role
// whose reference is not digest-pinned is dropped rather than pulled: AWF refuses
// a floating reference for this runtime, and the pull script refuses it too, so
// emitting it here would only turn a clear config error into a confusing pull
// failure.
func appleContainerRuntimeImages(workflowData *WorkflowData) []string {
	if !isAppleContainerRuntime(workflowData) {
		return nil
	}

	firewallConfig := getFirewallConfig(workflowData)
	imageTag := getAWFImageTag(firewallConfig)
	manifest := getSandboxAgentImages(workflowData)

	roles := []string{awfImageRoleAgent, awfImageRoleAppleInit}
	images := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))

	for _, role := range roles {
		image := defaultAWFImageForRole(role, imageTag)
		if manifest != nil {
			if override, ok := manifest[role]; ok && override != "" {
				image = override
			}
		}
		if image == "" || !awfPinnedImagePattern.MatchString(image) {
			appleContainerInstallLog.Printf("Skipping apple-container image role %q: reference %q is not digest-pinned", role, image)
			continue
		}
		if _, duplicate := seen[image]; duplicate {
			continue
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}

	return images
}

// generateAppleContainerSetupSteps returns the full provisioning sequence, in
// dependency order. It is the single entry point engines call so the ordering
// cannot drift between them.
func generateAppleContainerSetupSteps(workflowData *WorkflowData) []GitHubActionStep {
	if !isAppleContainerRuntime(workflowData) {
		return nil
	}

	steps := []GitHubActionStep{
		generateAppleContainerHostPreflightStep(),
		generateAppleContainerCLISetupStep(isRuntimeInstallEnabled(workflowData)),
		generateAppleContainerServicesStep(),
	}

	if pull := generateAppleContainerImagePullStep(appleContainerRuntimeImages(workflowData)); len(pull) > 0 {
		steps = append(steps, pull)
	}

	appleContainerInstallLog.Printf("Generated %d apple-container setup step(s)", len(steps))
	return steps
}

// shellQuoteForRun single-quotes a value for safe inclusion in a generated
// `run:` command line.
//
// Image references reaching this function are already validated as digest-pinned
// AWF references, but they originate in workflow frontmatter and are interpolated
// into a shell command, so quoting is applied unconditionally rather than on the
// assumption that validation ran first.
func shellQuoteForRun(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
