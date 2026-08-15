// This file generates GitHub Actions steps required to prepare AWF's preview
// cloud-hypervisor microVM runtime for sandbox.agent.runtime: cloud-hypervisor.

package workflow

import "github.com/github/gh-aw/pkg/logger"

var cloudHypervisorInstallLog = logger.New("workflow:cloud_hypervisor_install")

func generateCloudHypervisorKVMAccessStep() GitHubActionStep {
	cloudHypervisorInstallLog.Print("Generating cloud-hypervisor KVM access step")
	return GitHubActionStep([]string{
		"      - name: Grant runner access to KVM",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/cloud_hypervisor_kvm_access.sh"`,
	})
}

func generateCloudHypervisorHostPreflightStep() GitHubActionStep {
	cloudHypervisorInstallLog.Print("Generating cloud-hypervisor host eligibility preflight step")
	return GitHubActionStep([]string{
		"      - name: Check host eligibility for cloud-hypervisor",
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/cloud_hypervisor_host_preflight.sh"`,
	})
}

func generateCloudHypervisorBundleSetupStep(awfVersion string) GitHubActionStep {
	cloudHypervisorInstallLog.Printf("Generating cloud-hypervisor bundle setup step for AWF version %q", awfVersion)
	return GitHubActionStep([]string{
		"      - name: Download and verify cloud-hypervisor bundle",
		"        id: cloud-hypervisor-bundle",
		"        env:",
		"          GH_AW_AWF_VERSION: " + awfVersion,
		`        run: bash "${RUNNER_TEMP}/gh-aw/actions/cloud_hypervisor_setup_bundle.sh"`,
	})
}
