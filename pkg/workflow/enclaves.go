package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var enclavesLog = logger.New("workflow:enclaves")

const (
	enclaveMCPServerName          = "awf-enclave"
	enclaveMCPUpstreamURL         = "http://awf-enclave-mcp:8080/mcp"
	enclaveMCPCapabilityEnv       = "AWF_ENCLAVE_MCP_CAPABILITY"
	enclaveMCPGatewayContainerEnv = "AWF_ENCLAVE_MCP_GATEWAY_CONTAINER"
	enclaveMCPGatewayEndpointEnv  = "AWF_ENCLAVE_MCP_GATEWAY_ENDPOINT"
	enclaveMCPGatewayIdentityEnv  = "AWF_ENCLAVE_MCP_GATEWAY_IDENTITY"
	enclaveGitHubMCPAgentIDEnv    = "AWF_ENCLAVE_GITHUB_MCP_AGENT_ID"
	enclaveMCPReadinessTimeoutEnv = "AWF_ENCLAVE_MCP_READINESS_TIMEOUT_MS"
	enclaveMCPDeferredServersEnv  = "GH_AW_MCP_DEFERRED_SERVERS"
	enclaveMCPGatewayRunLabel     = "com.github.gh-aw.mcpg.run"
	enclaveMCPGatewayContainer    = "awmg-mcpg"
	enclaveGitHubIssuesProfile    = "issues-read-v1"
	enclaveMCPConnectTimeout      = 120
	enclaveMCPReadinessTimeoutMS  = 120000
	maxEnclaveTimingBucketSeconds = 4800
	enclaveMCPTransportAllowance  = 60
)

func enclaveGitHubMCPAgentPolicy(workflowData *WorkflowData) MCPGatewayAgentPolicy {
	repos := make([]string, 0)
	if enclave := enclaveGitHubIssuesConfig(workflowData); enclave != nil {
		for _, repo := range enclave.Repos {
			if repo != nil {
				repos = append(repos, repo.Repo)
			}
		}
	}
	return MCPGatewayAgentPolicy{
		Servers: []string{"github"},
		Tools:   map[string][]string{"github": {"list_issues", "issue_read"}},
		AllowOnly: map[string]any{
			"repos":         repos,
			"min-integrity": "approved",
		},
	}
}

var enclaveRepoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)

// EnclavesConfig configures AWF-owned, finite-disclosure private repository executors.
// Each executor type may appear at most once.
type EnclavesConfig []*EnclaveConfig

type EnclaveRepository struct {
	Repo        string `json:"repo"`
	Sensitivity string `json:"sensitivity"`
}

type EnclaveConfig struct {
	Script         *ScriptEnclaveConfig `json:"script,omitempty"`
	Agent          *AgentEnclaveConfig  `json:"agent,omitempty"`
	Repos          []*EnclaveRepository `json:"repos"`
	Runtime        string               `json:"runtime,omitempty"`
	Image          string               `json:"image,omitempty"`
	Timeout        int                  `json:"timeout,omitempty"`
	MemoryLimit    string               `json:"memory-limit,omitempty"`
	CPULimit       string               `json:"cpu-limit,omitempty"`
	PIDsLimit      int                  `json:"pids-limit,omitempty"`
	TmpfsLimit     string               `json:"tmpfs-limit,omitempty"`
	MaxOutputBytes int                  `json:"max-output-bytes,omitempty"`
	MaxInvocations int                  `json:"max-invocations,omitempty"`
}

type ScriptEnclaveConfig struct {
	MaxScriptBytes int `json:"max-script-bytes,omitempty"`
}

type AgentEnclaveConfig struct {
	Engine           string                    `json:"engine,omitempty"`
	Profile          string                    `json:"profile,omitempty"`
	Model            string                    `json:"model,omitempty"`
	MaxTaskBytes     int                       `json:"max-task-bytes,omitempty"`
	MaxModelRequests int                       `json:"max-model-requests,omitempty"`
	MaxModelTokens   int                       `json:"max-model-tokens,omitempty"`
	GitHub           *AgentEnclaveGitHubConfig `json:"github,omitempty"`
}

type AgentEnclaveGitHubConfig struct {
	CLI string `json:"cli"`
}

// UnmarshalJSON preserves the explicit null marker produced by YAML `script:`.
func (e *EnclaveConfig) UnmarshalJSON(data []byte) error {
	type enclaveAlias EnclaveConfig
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded enclaveAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*e = EnclaveConfig(decoded)
	if script, ok := raw["script"]; ok && string(script) == "null" {
		e.Script = &ScriptEnclaveConfig{}
	}
	return nil
}

func enclavesEnabled(workflowData *WorkflowData) bool {
	return workflowData != nil && len(workflowData.Enclaves) > 0
}

func enclaveGitHubIssuesEnabled(workflowData *WorkflowData) bool {
	return enclaveGitHubIssuesConfig(workflowData) != nil
}

func enclaveGitHubIssuesConfig(workflowData *WorkflowData) *EnclaveConfig {
	if workflowData == nil {
		return nil
	}
	for _, enclave := range workflowData.Enclaves {
		if enclave != nil &&
			enclave.Agent != nil &&
			enclave.Agent.GitHub != nil &&
			enclave.Agent.GitHub.CLI == enclaveGitHubIssuesProfile {
			return enclave
		}
	}
	return nil
}

func enabledEnclaveTools(workflowData *WorkflowData) []string {
	var tools []string
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil {
			continue
		}
		if enclave.Script != nil {
			tools = append(tools, "enclave_run_script")
		}
		if enclave.Agent != nil {
			tools = append(tools, "enclave_run_agent")
		}
	}
	return tools
}

func enclaveToolTimeout(workflowData *WorkflowData) int {
	if !enclavesEnabled(workflowData) {
		return 0
	}
	return maxEnclaveTimingBucketSeconds + enclaveMCPTransportAllowance
}

func validateEnclavesConfig(workflowData *WorkflowData) error {
	if !enclavesEnabled(workflowData) {
		return nil
	}
	enclavesLog.Printf("Validating %d enclave config(s)", len(workflowData.Enclaves))
	if !isAWFNetworkIsolationEnabled(workflowData) {
		enclavesLog.Print("Rejecting enclaves: AWF network isolation is not enabled")
		return errors.New("enclaves requires AWF network isolation; enable the agent sandbox with a network-isolated runtime such as sandbox.agent.runtime: docker")
	}
	seenTypes := make(map[string]struct{}, len(workflowData.Enclaves))
	repositorySensitivities := make(map[string]string)
	for i, enclave := range workflowData.Enclaves {
		if err := validateEnclaveEntry(i, enclave, seenTypes, repositorySensitivities); err != nil {
			return err
		}
	}
	if err := validateEnclaveTrustedSensitivityVersion(workflowData); err != nil {
		return err
	}
	if err := validateEnclaveGitHubIssuesVersions(workflowData); err != nil {
		return err
	}
	return nil
}

func validateEnclaveEntry(index int, enclave *EnclaveConfig, seenTypes map[string]struct{}, repositorySensitivities map[string]string) error {
	if enclave == nil {
		return fmt.Errorf("enclaves[%d] must be an object. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
	}
	enclaveType, ok := enclaveExecutor(enclave)
	if !ok {
		return fmt.Errorf("enclaves[%d] must contain exactly one of script or agent. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
	}
	if _, ok := seenTypes[enclaveType]; ok {
		return fmt.Errorf("enclaves contains duplicate executor type %q; each type may appear at most once. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential\n  - agent:\n      model: gpt-5\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", enclaveType)
	}
	seenTypes[enclaveType] = struct{}{}
	if enclaveType == "agent" && enclave.Agent.Model == "" {
		return fmt.Errorf("enclaves[%d].agent.model is required. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
	}
	if enclaveType == "agent" && enclave.Agent.GitHub != nil && enclave.Agent.GitHub.CLI != enclaveGitHubIssuesProfile {
		return fmt.Errorf("enclaves[%d].agent.github.cli must be %q", index, enclaveGitHubIssuesProfile)
	}
	nonPublicRepositories, err := validateEnclaveRepositories(index, enclave, repositorySensitivities)
	if err != nil {
		return err
	}
	if enclaveType == "agent" && enclave.Agent.GitHub != nil && nonPublicRepositories > 1 {
		return fmt.Errorf("enclaves[%d].agent.github.cli %q supports at most one non-public repository, but %d were configured", index, enclaveGitHubIssuesProfile, nonPublicRepositories)
	}
	return nil
}

func validateEnclaveRepositories(index int, enclave *EnclaveConfig, repositorySensitivities map[string]string) (int, error) {
	if len(enclave.Repos) == 0 {
		return 0, fmt.Errorf("enclaves[%d].repos must contain at least one repository. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
	}
	seenInEnclave := make(map[string]struct{}, len(enclave.Repos))
	nonPublicRepositories := 0
	for repoIndex, repo := range enclave.Repos {
		if repo == nil {
			return 0, fmt.Errorf("enclaves[%d].repos[%d] must be an object. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, repoIndex)
		}
		parts := strings.SplitN(repo.Repo, "/", 2)
		if !enclaveRepoPattern.MatchString(repo.Repo) || len(parts) != 2 || parts[1] == "." || parts[1] == ".." || strings.Contains(parts[1], "..") {
			return 0, fmt.Errorf("enclaves[%d].repos[%d].repo must be a bare owner/repository slug (e.g. org/my-repo). Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, repoIndex)
		}
		key := strings.ToLower(repo.Repo)
		if _, ok := seenInEnclave[key]; ok {
			return 0, fmt.Errorf("enclaves[%d].repos contains duplicate repository %q; each repository may appear at most once per enclave entry. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, repo.Repo)
		}
		seenInEnclave[key] = struct{}{}
		switch repo.Sensitivity {
		case "public", "trusted", "internal", "confidential", "sealed":
		default:
			return 0, fmt.Errorf("enclaves[%d].repos[%d].sensitivity must be public, trusted, internal, confidential, or sealed. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, repoIndex)
		}
		if repo.Sensitivity != "public" && repo.Sensitivity != "trusted" {
			nonPublicRepositories++
		}
		if sensitivity, ok := repositorySensitivities[key]; ok && sensitivity != repo.Sensitivity {
			return 0, fmt.Errorf("repository %q must use the same sensitivity across enclave types; all enclave entries for a given repository must declare the same sensitivity. Example:\n\nenclaves:\n  - script:\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential\n  - agent:\n      model: gpt-5\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", repo.Repo)
		}
		repositorySensitivities[key] = repo.Sensitivity
	}
	return nonPublicRepositories, nil
}

func validateEnclaveGitHubIssuesVersions(workflowData *WorkflowData) error {
	if !enclaveGitHubIssuesEnabled(workflowData) {
		return nil
	}

	firewallConfig := getFirewallConfig(workflowData)
	if !awfVersionAtLeast(firewallConfig, constants.AWFEnclaveGitHubIssuesMinVersion) {
		effectiveVersion := string(constants.DefaultFirewallVersion)
		if firewallConfig != nil && firewallConfig.Version != "" {
			effectiveVersion = firewallConfig.Version
		}
		return fmt.Errorf("enclaves[].agent.github.cli %q requires AWF %s or newer, but the effective version is %s", enclaveGitHubIssuesProfile, constants.AWFEnclaveGitHubIssuesMinVersion, effectiveVersion)
	}

	effectiveVersion := string(constants.DefaultMCPGatewayVersion)
	if workflowData.SandboxConfig != nil &&
		workflowData.SandboxConfig.MCP != nil &&
		workflowData.SandboxConfig.MCP.Version != "" {
		effectiveVersion = workflowData.SandboxConfig.MCP.Version
	}
	if !versionAtLeast(effectiveVersion, string(constants.DefaultMCPGatewayVersion), string(constants.MCPGEnclaveGitHubIssuesMinVersion)) {
		return fmt.Errorf("enclaves[].agent.github.cli %q requires MCPG %s or newer, but the effective version is %s; set sandbox.mcp.version to %s or newer", enclaveGitHubIssuesProfile, constants.MCPGEnclaveGitHubIssuesMinVersion, effectiveVersion, constants.MCPGEnclaveGitHubIssuesMinVersion)
	}
	return nil
}

func validateEnclaveTrustedSensitivityVersion(workflowData *WorkflowData) error {
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil {
			continue
		}
		for _, repo := range enclave.Repos {
			if repo != nil && repo.Sensitivity == "trusted" {
				firewallConfig := getFirewallConfig(workflowData)
				if !awfVersionAtLeast(firewallConfig, constants.AWFEnclaveTrustedSensitivityMinVersion) {
					effectiveVersion := string(constants.DefaultFirewallVersion)
					if firewallConfig != nil && firewallConfig.Version != "" {
						effectiveVersion = firewallConfig.Version
					}
					return fmt.Errorf("enclaves[].repos sensitivity %q requires AWF %s or newer, but the effective version is %s", "trusted", constants.AWFEnclaveTrustedSensitivityMinVersion, effectiveVersion)
				}
				return nil
			}
		}
	}
	return nil
}

func enclaveExecutor(enclave *EnclaveConfig) (string, bool) {
	if enclave.Script != nil && enclave.Agent == nil {
		return "script", true
	}
	if enclave.Agent != nil && enclave.Script == nil {
		return "agent", true
	}
	return "", false
}

func buildAWFEnclavesConfig(config EnclavesConfig) []map[string]any {
	if len(config) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(config))
	for _, enclave := range config {
		enclaveType, ok := enclaveExecutor(enclave)
		if !ok {
			continue
		}
		values := make(map[string]any)
		repos := make([]map[string]any, 0, len(enclave.Repos))
		for _, repo := range enclave.Repos {
			repos = append(repos, map[string]any{"repo": repo.Repo, "sensitivity": repo.Sensitivity})
		}
		values["repos"] = repos
		addEnclaveString(values, "runtime", enclave.Runtime)
		addEnclaveString(values, "image", enclave.Image)
		addEnclaveInt(values, "timeout", enclave.Timeout)
		addEnclaveString(values, "memoryLimit", enclave.MemoryLimit)
		addEnclaveString(values, "cpuLimit", enclave.CPULimit)
		addEnclaveInt(values, "pidsLimit", enclave.PIDsLimit)
		addEnclaveString(values, "tmpfsLimit", enclave.TmpfsLimit)
		addEnclaveInt(values, "maxOutputBytes", enclave.MaxOutputBytes)
		addEnclaveInt(values, "maxInvocations", enclave.MaxInvocations)
		if enclaveType == "script" {
			script := make(map[string]any)
			addEnclaveInt(script, "maxScriptBytes", enclave.Script.MaxScriptBytes)
			values["script"] = script
		} else {
			agent := make(map[string]any)
			addEnclaveString(agent, "engine", enclave.Agent.Engine)
			addEnclaveString(agent, "profile", enclave.Agent.Profile)
			addEnclaveString(agent, "model", enclave.Agent.Model)
			addEnclaveInt(agent, "maxTaskBytes", enclave.Agent.MaxTaskBytes)
			addEnclaveInt(agent, "maxModelRequests", enclave.Agent.MaxModelRequests)
			addEnclaveInt(agent, "maxModelTokens", enclave.Agent.MaxModelTokens)
			if enclave.Agent.GitHub != nil {
				agent["github"] = map[string]any{"cli": enclave.Agent.GitHub.CLI}
			}
			values["agent"] = agent
		}
		result = append(result, values)
	}
	enclavesLog.Printf("Built %d AWF enclave config(s) from %d entries", len(result), len(config))
	return result
}

func addEnclaveString(values map[string]any, key, value string) {
	if value != "" {
		values[key] = value
	}
}

func addEnclaveInt(values map[string]any, key string, value int) {
	if value != 0 {
		values[key] = value
	}
}

func writeEnclaveMCPJSON(yaml *strings.Builder, workflowData *WorkflowData, isLast bool) {
	fmt.Fprintf(yaml, "              %q: {\n", enclaveMCPServerName)
	yaml.WriteString("                \"type\": \"http\",\n")
	fmt.Fprintf(yaml, "                \"url\": %q,\n", enclaveMCPUpstreamURL)
	fmt.Fprintf(yaml, "                \"headers\": {\"Authorization\": \"Bearer \\${%s}\"},\n", enclaveMCPCapabilityEnv)
	fmt.Fprintf(yaml, "                \"tools\": [")
	for i, tool := range enabledEnclaveTools(workflowData) {
		if i > 0 {
			yaml.WriteString(", ")
		}
		fmt.Fprintf(yaml, "%q", tool)
	}
	yaml.WriteString("],\n")
	fmt.Fprintf(yaml, "                \"connectTimeout\": %d,\n", enclaveMCPConnectTimeout)
	fmt.Fprintf(yaml, "                \"toolTimeout\": %d\n", enclaveToolTimeout(workflowData))
	yaml.WriteString("              }")
	if !isLast {
		yaml.WriteString(",")
	}
	yaml.WriteString("\n")
}

func writeEnclaveMCPTOML(yaml *strings.Builder, workflowData *WorkflowData) {
	yaml.WriteString("          \n")
	fmt.Fprintf(yaml, "          [mcp_servers.%s]\n", enclaveMCPServerName)
	yaml.WriteString("          type = \"http\"\n")
	fmt.Fprintf(yaml, "          url = %q\n", enclaveMCPUpstreamURL)
	fmt.Fprintf(yaml, "          headers = { Authorization = \"Bearer $%s\" }\n", enclaveMCPCapabilityEnv)
	fmt.Fprintf(yaml, "          tools = [")
	for i, tool := range enabledEnclaveTools(workflowData) {
		if i > 0 {
			yaml.WriteString(", ")
		}
		fmt.Fprintf(yaml, "%q", tool)
	}
	yaml.WriteString("]\n")
	fmt.Fprintf(yaml, "          connectTimeout = %d\n", enclaveMCPConnectTimeout)
	fmt.Fprintf(yaml, "          toolTimeout = %d\n", enclaveToolTimeout(workflowData))
}
