package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func hasCloudHypervisorEnclaves(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}
	for _, enclave := range workflowData.Enclaves {
		if enclave != nil && enclave.Runtime == string(AgentRuntimeCloudHypervisor) {
			return true
		}
	}
	return false
}

func validateCloudHypervisorEnclavePrimaryRuntime(workflowData *WorkflowData) error {
	if !hasCloudHypervisorEnclaves(workflowData) {
		return nil
	}
	agent := getAgentConfig(workflowData)
	if agent == nil {
		return nil
	}
	switch agent.Runtime {
	case AgentRuntimeDockerSudoIptables:
		return errors.New("enclaves cloud-hypervisor runtime is incompatible with sandbox.agent.runtime: docker-sudo-iptables; use sandbox.agent.runtime: docker")
	case "sbx", "nvx":
		return fmt.Errorf("enclaves cloud-hypervisor runtime is incompatible with sandbox.agent.runtime: %s; use a Docker primary agent runtime", agent.Runtime)
	default:
		return nil
	}
}

func validateCloudHypervisorEnclaves(workflowData *WorkflowData) error {
	if !hasCloudHypervisorEnclaves(workflowData) && (!enclavesEnabled(workflowData) || !isCloudHypervisorRuntime(workflowData)) {
		return nil
	}
	if isArcDindTopology(workflowData) {
		return errors.New("enclaves cloud-hypervisor runtime is incompatible with runner.topology: arc-dind; use GitHub-hosted Ubuntu x86_64 KVM runners")
	}
	if isCliProxyNeeded(workflowData) {
		return errors.New("enclaves cloud-hypervisor runtime is incompatible with tools.github.mode: gh-proxy and integrity-reactions; remove those settings")
	}
	args := customAWFArgs(workflowData)
	primaryRuntime := string(AgentRuntimeDocker)
	if isCloudHypervisorRuntime(workflowData) {
		primaryRuntime = string(AgentRuntimeCloudHypervisor)
	}
	if err := validateCloudHypervisorEnclaveAWFArgs(args, primaryRuntime); err != nil {
		return err
	}
	for i, enclave := range workflowData.Enclaves {
		if enclave.Runtime != string(AgentRuntimeCloudHypervisor) {
			return errors.New("enclaves cloud-hypervisor runtime cannot be mixed with Docker or default enclave runtimes; set runtime: cloud-hypervisor on every enclave entry")
		}
		if isCloudHypervisorRuntime(workflowData) && enclave.Agent != nil {
			if err := validateCloudHypervisorAgentAcceptance(workflowData, enclave); err != nil {
				return fmt.Errorf("enclaves[%d] agent runtime: cloud-hypervisor is not supported with a cloud-hypervisor primary outside the experimental acceptance fixture; use a static script enclave: %w", i, err)
			}
		}
		if enclave.Image != "" {
			return fmt.Errorf("enclaves[%d].image is incompatible with runtime: cloud-hypervisor; remove the image override to use release-attested enclave artifacts", i)
		}
		if enclave.Dynamic != nil {
			return fmt.Errorf("enclaves[%d].dynamic is incompatible with runtime: cloud-hypervisor; use static repos", i)
		}
		if enclave.Agent != nil {
			if strings.TrimSpace(enclave.Agent.Model) == "" {
				return fmt.Errorf("enclaves[%d].agent.model is required for runtime: cloud-hypervisor", i)
			}
			if enclave.Agent.Engine != "" && enclave.Agent.Engine != "copilot" {
				return fmt.Errorf("enclaves[%d].agent.engine %q is not implemented for runtime: cloud-hypervisor; use copilot", i, enclave.Agent.Engine)
			}
			if enclave.Agent.GitHub != nil || enclaveGitHubToolsConfig(enclave) != nil {
				return fmt.Errorf("enclaves[%d] cloud-hypervisor static GitHub tools require a scoped executor bearer handoff that is not yet supported; remove agent.github and agent.tools.github", i)
			}
			if hasEnabledAWFArg(args, "--no-enable-api-proxy") {
				return fmt.Errorf("enclaves[%d] cloud-hypervisor agent runtime requires the API proxy; remove --no-enable-api-proxy", i)
			}
			if !cloudHypervisorEnclaveCopilotRouteConfigured(workflowData) {
				return fmt.Errorf("enclaves[%d] cloud-hypervisor agent runtime requires a configured Copilot API proxy provider route (COPILOT_GITHUB_TOKEN or COPILOT_PROVIDER_API_KEY); use a GitHub-backed primary engine or configure Copilot BYOK credentials", i)
			}
		}
	}
	minVersion := constants.AWFEnclaveCloudHypervisorMinVersion
	if isCloudHypervisorRuntime(workflowData) {
		minVersion = constants.AWFCloudHypervisorPrimaryEnclaveMinVersion
	}
	return validateCloudHypervisorEnclaveVersion(getFirewallConfig(workflowData), minVersion)
}

const cloudHypervisorAgentAcceptanceFeature = "cloud-hypervisor-agent-acceptance"

func validateCloudHypervisorAgentAcceptance(data *WorkflowData, enclave *EnclaveConfig) error {
	// Require a checked-in boolean opt-in, not an ambient GH_AW_FEATURES override.
	if enabled, ok := data.Features[cloudHypervisorAgentAcceptanceFeature].(bool); !ok || !enabled {
		return errors.New("set features.cloud-hypervisor-agent-acceptance: true only for the official acceptance fixture")
	}
	if data.WorkflowID != "smoke-enclave-cloud-hypervisor-copilot" ||
		strings.TrimSpace(stripExpressionWrapper(data.If)) != "github.repository == 'github/gh-aw-firewall' && github.ref_type == 'branch'" {
		return errors.New("agent acceptance requires workflow smoke-enclave-cloud-hypervisor-copilot and if: github.repository == 'github/gh-aw-firewall' && github.ref_type == 'branch'")
	}
	manualOnly := false
	switch triggers := data.RawFrontmatter["on"].(type) {
	case string:
		manualOnly = triggers == "workflow_dispatch"
	case map[string]any:
		if len(triggers) == 1 {
			_, manualOnly = triggers["workflow_dispatch"]
		}
	}
	if !manualOnly {
		return errors.New("agent acceptance requires on: workflow_dispatch without other triggers")
	}
	if len(data.Enclaves) != 1 || len(enclave.Repos) != 1 {
		return errors.New("agent acceptance requires exactly one agent enclave with only the static public github/gh-aw-firewall repository")
	}
	for _, repo := range enclave.Repos {
		if repo == nil || repo.Repo != "github/gh-aw-firewall" || repo.Sensitivity != "public" {
			return errors.New("agent acceptance requires only the static public github/gh-aw-firewall repository")
		}
	}
	if ResolveEngineID(data) != "copilot" || enclave.Agent.Engine != "copilot" ||
		enclave.Agent.Model != "claude-sonnet-5" || enclave.Agent.Profile != "" {
		return errors.New("agent acceptance requires a Copilot primary and agent.engine: copilot, agent.model: claude-sonnet-5 without a profile override")
	}
	if enclave.Timeout != 90 || enclave.MaxOutputBytes != 256 || enclave.MaxInvocations != 1 ||
		enclave.Agent.MaxTaskBytes != 4096 || enclave.Agent.MaxModelRequests != 8 || enclave.Agent.MaxModelTokens != 2048 {
		return errors.New("agent acceptance requires timeout: 90, max-output-bytes: 256, max-invocations: 1, agent.max-task-bytes: 4096, agent.max-model-requests: 8 and agent.max-model-tokens: 2048")
	}
	if firewall := getFirewallConfig(data); firewall == nil || firewall.Version != "v0.28.62" {
		return errors.New("agent acceptance requires sandbox.agent.version: v0.28.62 and its release-attested guests; branch-built host acceptance is not a released runtime capability")
	}
	return nil
}

func validateCloudHypervisorEnclaveAWFArgs(args []string, primaryRuntime string) error {
	if hasEnabledAWFArg(args, "--enable-dind") {
		return errors.New("enclaves cloud-hypervisor runtime is incompatible with --enable-dind; remove DinD")
	}
	for index, arg := range args {
		if arg == "--docker-host-path-prefix" || strings.HasPrefix(arg, "--docker-host-path-prefix=") {
			return errors.New("enclaves cloud-hypervisor runtime is incompatible with --docker-host-path-prefix; remove the Docker host path prefix")
		}
		name, runtime, _ := strings.Cut(arg, "=")
		if name == "--container-runtime" {
			if nextIndex := index + 1; arg == name && nextIndex >= 0 && nextIndex < len(args) {
				runtime = args[nextIndex]
			}
			if runtime != primaryRuntime {
				return fmt.Errorf("enclaves cloud-hypervisor runtime requires --container-runtime %s to match sandbox.agent.runtime", primaryRuntime)
			}
		}
	}
	return nil
}

func validateCloudHypervisorEnclaveVersion(firewallConfig *FirewallConfig, minVersion constants.Version) error {
	if !awfVersionAtLeast(firewallConfig, minVersion) {
		effectiveVersion := string(constants.DefaultFirewallVersion)
		if firewallConfig != nil && firewallConfig.Version != "" {
			effectiveVersion = firewallConfig.Version
		}
		return fmt.Errorf("enclaves cloud-hypervisor runtime requires AWF %s or newer, but the effective version is %s; set sandbox.agent.version to %s or newer", minVersion, effectiveVersion, minVersion)
	}
	return nil
}

func cloudHypervisorEnclaveCopilotRouteConfigured(workflowData *WorkflowData) bool {
	env := make(map[string]string)
	engineID := ResolveEngineID(workflowData)
	if engineID == "" || engineID == "copilot" {
		engine := NewCopilotEngine()
		env = engine.buildCopilotBaseStepEnv(workflowData, engine.ResolveLLMProvider(workflowData), "", isCopilotBYOKMode(workflowData, true), hasCopilotRequestsWritePermission(workflowData))
	} else if engine, err := GetGlobalEngineRegistry().GetEngine(engineID); err == nil {
		if resolver, ok := engine.(InferenceProviderResolver); ok && resolver.ResolveLLMProvider(workflowData) == LLMProviderGitHub {
			env["COPILOT_GITHUB_TOKEN"] = llmProviderSecretExpression(LLMProviderGitHub, workflowData)
		}
	}
	applyEngineAndAgentEnv(env, workflowData, enclavesLog)
	applyMCPScriptsSecretEnv(env, workflowData)
	if engineID == "" {
		engineID = "copilot"
	}
	engine, err := GetGlobalEngineRegistry().GetEngine(engineID)
	if err != nil {
		return false
	}
	env = FilterEnvForSecrets(env, engine.GetRequiredSecretNames(workflowData))
	// Provider routes are configured by credentials in the emitted step environment,
	// not by apiProxy.providers (pricing overlays) or a target hostname alone.
	return strings.TrimSpace(env["COPILOT_GITHUB_TOKEN"]) != "" || strings.TrimSpace(env["COPILOT_PROVIDER_API_KEY"]) != ""
}
