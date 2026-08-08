// This file generates the GitHub Actions steps required to install, authenticate,
// and pre-flight-test the Docker sbx microVM runtime for sandbox.agent.runtime: docker-sbx.
//
// The steps emitted are (in order):
//  1. KVM availability check – fails fast when nested virtualisation is absent.
//  2. Docker Hub secrets check – verifies DOCKER_PAT / DOCKER_USERNAME are present.
//  3. sbx installation      – adds the Docker apt repo and installs the docker-sbx package.
//  4. sbx auth & daemon     – authenticates with Docker Hub, starts the daemon, resets and
//     re-initialises the allow-all policy, then pre-pulls the template image.
//  5. sbx pre-flight smoke  – creates a throwaway sandbox, execs a command, then removes it.
//
// All five steps must be injected BEFORE the AWF installation step so the sbx runtime
// is available when AWF starts the agent inside a microVM.
//
// Shell script sources live in actions/setup/sh/:
//   - docker_sbx_kvm_check.sh        (no sudo)
//   - docker_sbx_secrets_check.sh    (no sudo)
//   - sudo_docker_sbx_install.sh     (requires sudo)
//   - docker_sbx_daemon.sh           (no sudo)
//   - docker_sbx_preflight.sh        (no sudo)
//   - docker_sbx_credential_refresh.sh (no sudo, always emitted)

package workflow

import "github.com/github/gh-aw/pkg/logger"

// dockerSbxInstallLog traces which docker-sbx runtime steps are emitted during
// compilation. Enable with DEBUG=workflow:docker_sbx_install (or workflow:*).
var dockerSbxInstallLog = logger.New("workflow:docker_sbx_install")

// generateDockerSbxKVMCheckStep creates a fail-fast step that verifies the runner
// has KVM support before spending time on sbx installation.
func generateDockerSbxKVMCheckStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx KVM availability check step")
	return GitHubActionStep([]string{
		"      - name: Check KVM availability for docker-sbx",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_kvm_check.sh"`,
	})
}

// generateDockerSbxSecretsCheckStep creates a step that verifies the DOCKER_PAT
// and DOCKER_USERNAME secrets are present before attempting sbx install.
func generateDockerSbxSecretsCheckStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx Docker Hub secrets check step")
	return GitHubActionStep([]string{
		"      - name: Check Docker Hub secrets for docker-sbx",
		"        id: docker-sbx-secrets",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_secrets_check.sh"`,
	})
}

// generateDockerSbxActivationSecretsCheckStep creates a soft-failing activation
// step that records whether docker-sbx can run without failing the workflow.
func generateDockerSbxActivationSecretsCheckStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx activation Docker Hub secrets check step")
	return GitHubActionStep([]string{
		"      - name: Check Docker Hub secrets for docker-sbx",
		"        id: docker-sbx-secrets",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		"          DOCKER_SBX_SECRETS_SOFT_FAIL: 'true'",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_secrets_check.sh"`,
	})
}

// generateDockerSbxInstallStep creates a GitHub Actions step that installs the
// docker-sbx package via the official Docker apt repository. Requires sudo.
func generateDockerSbxInstallStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx package install step")
	return GitHubActionStep([]string{
		"      - name: Install docker-sbx",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/sudo_docker_sbx_install.sh"`,
	})
}

// generateDockerSbxAuthAndDaemonStep creates a step that starts the sbx daemon,
// authenticates with Docker Hub, resets and re-initialises the sbx policy, and
// pre-pulls the sandbox template image.
func generateDockerSbxAuthAndDaemonStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx daemon start and Docker Hub authentication step")
	return GitHubActionStep([]string{
		"      - name: Start docker-sbx daemon and authenticate",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_daemon.sh"`,
	})
}

// generateDockerSbxCredentialRefreshStep creates a step that re-authenticates the
// sbx daemon with Docker Hub immediately before AWF runs the agent. This step does
// not require sudo and is always emitted when docker-sbx is the runtime, regardless
// of the runtime-install setting.
func generateDockerSbxCredentialRefreshStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx credential refresh step (pre-AWF re-authentication)")
	return GitHubActionStep([]string{
		"      - name: Refresh sbx credentials",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_credential_refresh.sh"`,
	})
}

// generateDockerSbxPreFlightStep creates a step that verifies the sbx stack works
// end-to-end before the MCP gateway and AWF container setup begins.
func generateDockerSbxPreFlightStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx pre-flight smoke test step")
	return GitHubActionStep([]string{
		"      - name: docker-sbx pre-flight smoke test",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/docker_sbx_preflight.sh"`,
	})
}
