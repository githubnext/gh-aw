// This file generates the GitHub Actions steps required to install, authenticate,
// and pre-flight-test the Docker sbx microVM runtime for sandbox.agent.runtime: docker-sbx.
//
// The steps emitted are (in order):
//  1. KVM availability check – fails fast when nested virtualisation is absent.
//  2. Docker Hub secrets check – fails fast when DOCKER_PAT / DOCKER_USERNAME are missing.
//  3. sbx installation      – adds the Docker apt repo and installs the docker-sbx package.
//  4. sbx auth & daemon     – authenticates with Docker Hub, starts the daemon, resets and
//     re-initialises the allow-all policy, then pre-pulls the template image.
//  5. sbx pre-flight smoke  – creates a throwaway sandbox, execs a command, then removes it.
//
// All five steps must be injected BEFORE the AWF installation step so the sbx runtime
// is available when AWF starts the agent inside a microVM.

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
		"        run: |",
		"          set -euo pipefail",
		`          echo "::group::KVM availability check"`,
		`          if ! lsmod | grep -q kvm; then`,
		`            echo "::error::KVM kernel module is not loaded. docker-sbx requires a KVM-capable runner with nested virtualisation enabled."`,
		`            exit 1`,
		`          fi`,
		`          if ! test -e /dev/kvm; then`,
		`            echo "::error::/dev/kvm is missing. docker-sbx requires the KVM device to be present on the runner."`,
		`            exit 1`,
		`          fi`,
		`          echo "KVM is available and /dev/kvm is present ✅"`,
		`          echo "::endgroup::"`,
	})
}

// generateDockerSbxSecretsCheckStep creates a fail-fast step that verifies the
// DOCKER_PAT and DOCKER_USERNAME secrets are present before attempting sbx install.
func generateDockerSbxSecretsCheckStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx Docker Hub secrets check step")
	return GitHubActionStep([]string{
		"      - name: Check Docker Hub secrets for docker-sbx",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		"        run: |",
		"          set -euo pipefail",
		`          echo "::group::Docker Hub secrets check"`,
		`          if [[ -z "${DOCKER_PAT_VAL}" ]]; then`,
		`            echo "::error::secrets.DOCKER_PAT is empty. docker-sbx requires a Docker Hub personal access token to pull the sandbox template image. Add a DOCKER_PAT secret to your repository."`,
		`            exit 1`,
		`          fi`,
		`          if [[ -z "${DOCKER_USERNAME_VAL}" ]]; then`,
		`            echo "::error::secrets.DOCKER_USERNAME is empty. docker-sbx requires a Docker Hub username. Add a DOCKER_USERNAME secret to your repository."`,
		`            exit 1`,
		`          fi`,
		`          echo "Docker Hub secrets are present ✅"`,
		`          echo "::endgroup::"`,
	})
}

// generateDockerSbxInstallStep creates a GitHub Actions step that installs the
// docker-sbx package via the official Docker apt repository.
func generateDockerSbxInstallStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx package install step")
	return GitHubActionStep([]string{
		"      - name: Install docker-sbx",
		"        run: |",
		"          set -euo pipefail",
		`          echo "::group::Install docker-sbx"`,
		`          # Add Docker apt repo without installing Docker Engine (already present).`,
		`          curl -fsSL https://get.docker.com | sudo REPO_ONLY=1 sh`,
		`          sudo apt-get install -y docker-sbx`,
		`          sbx version`,
		`          # Fix KVM permissions so the runner user can create microVMs.`,
		`          sudo chmod 666 /dev/kvm`,
		`          echo "docker-sbx installed successfully ✅"`,
		`          echo "::endgroup::"`,
	})
}

// generateDockerSbxAuthAndDaemonStep creates a step that:
//  1. Starts the sbx daemon.
//  2. Authenticates with Docker Hub (both docker and sbx CLIs).
//  3. Resets and re-initialises the sbx policy (required for mount policy).
//  4. Restarts the daemon and re-authenticates.
//  5. Pre-pulls the sandbox template image.
func generateDockerSbxAuthAndDaemonStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx daemon start and Docker Hub authentication step")
	return GitHubActionStep([]string{
		"      - name: Start docker-sbx daemon and authenticate",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		"        run: |",
		"          set -euo pipefail",
		`          export DOCKER_CONFIG="$(mktemp -d)"`,
		`          trap 'rm -rf "${DOCKER_CONFIG}"' EXIT`,
		`          echo "::group::Start sbx daemon"`,
		`          nohup sbx daemon start > /tmp/sbx-daemon.log 2>&1 &`,
		`          # Poll until daemon is running (up to 10 s).`,
		`          for i in $(seq 1 10); do`,
		`            if sbx daemon status 2>/dev/null | grep -q -i running; then`,
		`              echo "sbx daemon is running"`,
		`              break`,
		`            fi`,
		`            sleep 1`,
		`          done`,
		`          echo "::endgroup::"`,
		`          echo "::group::Authenticate with Docker Hub"`,
		`          printf '%s' "${DOCKER_PAT_VAL}" | docker login --username "${DOCKER_USERNAME_VAL}" --password-stdin`,
		`          printf '%s' "${DOCKER_PAT_VAL}" | sbx login --username "${DOCKER_USERNAME_VAL}" --password-stdin`,
		`          echo "::endgroup::"`,
		`          echo "::group::Reset and initialise sbx policy"`,
		`          sbx daemon stop || true`,
		`          sbx policy reset --force || true`,
		`          sbx policy init allow-all`,
		`          nohup sbx daemon start > /tmp/sbx-daemon.log 2>&1 &`,
		`          for i in $(seq 1 10); do`,
		`            if sbx daemon status 2>/dev/null | grep -q -i running; then`,
		`              echo "sbx daemon restarted"`,
		`              break`,
		`            fi`,
		`            sleep 1`,
		`          done`,
		`          printf '%s' "${DOCKER_PAT_VAL}" | docker login --username "${DOCKER_USERNAME_VAL}" --password-stdin`,
		`          printf '%s' "${DOCKER_PAT_VAL}" | sbx login --username "${DOCKER_USERNAME_VAL}" --password-stdin`,
		`          echo "::endgroup::"`,
		`          echo "::group::Pre-pull sandbox template image"`,
		`          docker pull docker/sandbox-templates:shell-docker`,
		`          echo "Template image ready ✅"`,
		`          echo "::endgroup::"`,
	})
}

// generateDockerSbxCredentialRefreshStep creates a step that re-authenticates the
// sbx daemon with Docker Hub immediately before AWF runs the agent. OAuth tokens
// obtained by `sbx login` during the daemon-setup step can expire or be invalidated
// by the policy-reset cycle, so a fresh login right before execution prevents
// "user is not authenticated to Docker" errors when AWF calls `sbx create`.
func generateDockerSbxCredentialRefreshStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx credential refresh step (pre-AWF re-authentication)")
	return GitHubActionStep([]string{
		"      - name: Refresh sbx credentials",
		"        env:",
		"          DOCKER_PAT_VAL: ${{ secrets.DOCKER_PAT }}",
		"          DOCKER_USERNAME_VAL: ${{ secrets.DOCKER_USERNAME }}",
		"        run: |",
		`          # Re-authenticate sbx immediately before AWF runs.`,
		`          # Docker Hub OAuth tokens from sbx login can expire between steps.`,
		`          printf '%s' "$DOCKER_PAT_VAL" | sbx login --username "$DOCKER_USERNAME_VAL" --password-stdin`,
		`          echo "✅ sbx credentials refreshed"`,
	})
}

// generateDockerSbxPreFlightStep creates a step that verifies the sbx stack works
// end-to-end before the MCP gateway and AWF container setup begins.
func generateDockerSbxPreFlightStep() GitHubActionStep {
	dockerSbxInstallLog.Print("Generating docker-sbx pre-flight smoke test step")
	return GitHubActionStep([]string{
		"      - name: docker-sbx pre-flight smoke test",
		"        run: |",
		"          set -euo pipefail",
		`          echo "::group::docker-sbx pre-flight smoke test"`,
		`          sandbox_name="test-sandbox-direct"`,
		`          cleanup() {`,
		`            sbx stop "${sandbox_name}" >/dev/null 2>&1 || true`,
		`            sbx rm --force "${sandbox_name}" >/dev/null 2>&1 || true`,
		`          }`,
		`          trap cleanup EXIT`,
		`          echo "y" | sbx create shell --name "${sandbox_name}" "${GITHUB_WORKSPACE}"`,
		`          sbx exec "${sandbox_name}" uname -a`,
		`          sbx stop "${sandbox_name}"`,
		`          echo "✅ sbx ready"`,
		`          echo "::endgroup::"`,
	})
}
