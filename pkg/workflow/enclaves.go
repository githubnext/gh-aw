package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	enclaveGitHubDelegationEnv    = "AWF_ENCLAVE_GITHUB_DELEGATION_CONTROL_CAPABILITY"
	enclaveMCPGatewayRunLabel     = "com.github.gh-aw.mcpg.run"
	enclaveMCPGatewayContainer    = "awmg-mcpg"
	enclaveGitHubIssuesProfile    = "issues-read-v1"
	enclaveDynamicGitHubPolicy    = "github-repository-read-v1"
	enclaveDynamicController      = "github-repository-delegation-v1"
	enclaveMCPConnectTimeout      = 120
	enclaveMCPReadinessTimeoutMS  = 120000
	maxEnclaveTimingBucketSeconds = 4800
	enclaveMCPTransportAllowance  = 60
)

var enclaveAgentGitHubSupportedTools = map[string]struct{}{
	"list_issues": {},
	"issue_read":  {},
}

var enclaveAgentGitHubDefaultTools = []string{"list_issues", "issue_read"}

var enclaveAgentGitHubValidIntegrityLevels = map[GitHubIntegrityLevel]struct{}{
	GitHubIntegrityNone:       {},
	GitHubIntegrityUnapproved: {},
	GitHubIntegrityApproved:   {},
	GitHubIntegrityMerged:     {},
}

func enclaveGitHubMCPAgentPolicy(workflowData *WorkflowData) MCPGatewayAgentPolicy {
	repos := make([]string, 0)
	tools := append([]string(nil), enclaveAgentGitHubDefaultTools...)
	minIntegrity := string(GitHubIntegrityApproved)
	if enclave := enclaveStaticGitHubAgentConfig(workflowData); enclave != nil {
		repos = enclaveGitHubAllowedRepos(enclave)
		if github := enclaveGitHubToolsConfig(enclave); github != nil {
			tools = append([]string(nil), github.Allowed...)
			if github.MinIntegrity != "" {
				minIntegrity = string(github.MinIntegrity)
			}
		}
	}
	return MCPGatewayAgentPolicy{
		Servers: []string{"github"},
		Tools:   map[string][]string{"github": tools},
		AllowOnly: map[string]any{
			"repos":         repos,
			"min-integrity": minIntegrity,
		},
	}
}

var enclaveRepoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)
var enclaveDynamicSelectorPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38})/[a-z0-9._-]{1,100}$`)
var enclaveDynamicOwnerPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38})$`)
var enclaveAuditLabelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

// EnclavesConfig configures AWF-owned, finite-disclosure private repository executors.
// Each executor type may appear at most once.
type EnclavesConfig []*EnclaveConfig

type EnclaveRepository struct {
	Repo        string `json:"repo"`
	Sensitivity string `json:"sensitivity"`
}

type EnclaveConfig struct {
	Script         *ScriptEnclaveConfig  `json:"script,omitempty"`
	Agent          *AgentEnclaveConfig   `json:"agent,omitempty"`
	Repos          []*EnclaveRepository  `json:"repos"`
	Dynamic        *DynamicEnclavePolicy `json:"dynamic,omitempty"`
	Runtime        string                `json:"runtime,omitempty"`
	Image          string                `json:"image,omitempty"`
	Timeout        int                   `json:"timeout,omitempty"`
	MemoryLimit    string                `json:"memory-limit,omitempty"`
	CPULimit       string                `json:"cpu-limit,omitempty"`
	PIDsLimit      int                   `json:"pids-limit,omitempty"`
	TmpfsLimit     string                `json:"tmpfs-limit,omitempty"`
	MaxOutputBytes int                   `json:"max-output-bytes,omitempty"`
	MaxInvocations int                   `json:"max-invocations,omitempty"`
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
	Tools            *AgentEnclaveToolsConfig  `json:"tools,omitempty"`
}

type AgentEnclaveGitHubConfig struct {
	CLI string `json:"cli"`
}

type AgentEnclaveToolsConfig struct {
	GitHub *AgentEnclaveGitHubToolConfig `json:"github,omitempty"`
}

type AgentEnclaveGitHubToolConfig struct {
	Allowed      []string             `json:"allowed,omitempty"`
	AllowedRepos GitHubReposScope     `json:"allowed-repos,omitempty"`
	MinIntegrity GitHubIntegrityLevel `json:"min-integrity,omitempty"`
}

type DynamicEnclavePolicy struct {
	AllowedOwners       []string              `json:"allowed-owners,omitempty"`
	AllowedRepositories []string              `json:"allowed-repositories,omitempty"`
	Sensitivity         string                `json:"sensitivity"`
	GitHubPolicy        string                `json:"github-policy"`
	MaxRepositories     int                   `json:"max-repositories"`
	Quotas              *DynamicEnclaveQuotas `json:"quotas,omitempty"`
	AuditLabels         []string              `json:"audit-labels,omitempty"`
	ExpiresAt           string                `json:"expires-at"`
}

type DynamicEnclaveQuotas struct {
	MaxInvocations      int `json:"max-invocations"`
	MaxOutputBytes      int `json:"max-output-bytes"`
	MaxExecutionSeconds int `json:"max-execution-seconds"`
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
	return enclaveStaticGitHubAgentConfig(workflowData) != nil
}

func enclaveDynamicRepositoryPolicyConfig(workflowData *WorkflowData) *EnclaveConfig {
	if workflowData == nil {
		return nil
	}
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil || enclave.Agent == nil {
			continue
		}
		if enclave.Dynamic != nil {
			return enclave
		}
	}
	return nil
}

func enclaveDynamicRepositoryPolicyEnabled(workflowData *WorkflowData) bool {
	return enclaveDynamicRepositoryPolicyConfig(workflowData) != nil
}

func enclaveGitHubDelegationEnabled(workflowData *WorkflowData) bool {
	return enclaveGitHubIssuesEnabled(workflowData) || enclaveDynamicRepositoryPolicyEnabled(workflowData)
}

func enclaveStaticGitHubAgentConfig(workflowData *WorkflowData) *EnclaveConfig {
	if workflowData == nil {
		return nil
	}
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil || enclave.Agent == nil || enclave.Dynamic != nil {
			continue
		}
		if enclave.Agent.GitHub != nil && enclave.Agent.GitHub.CLI == enclaveGitHubIssuesProfile {
			return enclave
		}
		if enclaveGitHubToolsConfig(enclave) != nil {
			return enclave
		}
	}
	return nil
}

func enclaveGitHubToolsConfig(enclave *EnclaveConfig) *AgentEnclaveGitHubToolConfig {
	if enclave == nil || enclave.Agent == nil || enclave.Agent.Tools == nil {
		return nil
	}
	return enclave.Agent.Tools.GitHub
}

func enclaveGitHubAllowedRepos(enclave *EnclaveConfig) []string {
	github := enclaveGitHubToolsConfig(enclave)
	if github != nil && len(github.AllowedRepos) > 0 {
		return append([]string(nil), github.AllowedRepos...)
	}
	repos := make([]string, 0, len(enclave.Repos))
	for _, repo := range enclave.Repos {
		if repo != nil {
			repos = append(repos, repo.Repo)
		}
	}
	return repos
}

func enabledEnclaveTools(workflowData *WorkflowData) []string {
	var tools []string
	agentEnabled := false
	for _, enclave := range workflowData.Enclaves {
		if enclave == nil {
			continue
		}
		if enclave.Script != nil {
			tools = append(tools, "enclave_run_script")
		}
		if enclave.Agent != nil && !agentEnabled {
			tools = append(tools, "enclave_run_agent")
			agentEnabled = true
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
	if enclaveType == "agent" && enclave.Agent.GitHub != nil && enclaveGitHubToolsConfig(enclave) != nil {
		return fmt.Errorf("enclaves[%d].agent.github and enclaves[%d].agent.tools.github cannot both be set. Use only one enclave GitHub configuration shape. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n      tools:\n        github:\n          allowed: [list_issues]\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, index)
	}
	if enclaveType == "agent" && enclave.Agent.GitHub != nil && enclave.Agent.GitHub.CLI != "" && enclave.Agent.GitHub.CLI != enclaveGitHubIssuesProfile {
		return fmt.Errorf("enclaves[%d].agent.github.cli must be %q", index, enclaveGitHubIssuesProfile)
	}
	if enclaveType == "script" && enclave.Dynamic != nil {
		return fmt.Errorf("enclaves[%d].dynamic is only supported for agent entries; script enclaves must declare static repos", index)
	}
	if len(enclave.Repos) > 0 && enclave.Dynamic != nil {
		return fmt.Errorf("enclaves[%d] must declare either static repos or dynamic, not both", index)
	}
	if len(enclave.Repos) == 0 && enclave.Dynamic == nil {
		return fmt.Errorf("enclaves[%d] must declare either non-empty static repos or a dynamic repository policy", index)
	}
	nonPublicRepositories := 0
	var err error
	if enclave.Dynamic != nil {
		if err := validateDynamicEnclavePolicy(index, enclave); err != nil {
			return err
		}
	} else {
		nonPublicRepositories, err = validateEnclaveRepositories(index, enclave, repositorySensitivities)
	}
	if err != nil {
		return err
	}
	if enclaveType == "agent" && enclaveGitHubToolsConfig(enclave) != nil {
		if err := validateEnclaveGitHubTools(index, enclave); err != nil {
			return err
		}
	}
	if enclaveType == "agent" && enclave.Agent.GitHub != nil && enclave.Agent.GitHub.CLI == enclaveGitHubIssuesProfile && nonPublicRepositories > 1 {
		return fmt.Errorf("enclaves[%d].agent.github.cli %q supports at most one non-public repository, but %d were configured", index, enclaveGitHubIssuesProfile, nonPublicRepositories)
	}
	return nil
}

func validateDynamicEnclavePolicy(index int, enclave *EnclaveConfig) error {
	policy := enclave.Dynamic
	if policy == nil {
		return nil
	}
	if len(policy.AllowedOwners) == 0 && len(policy.AllowedRepositories) == 0 {
		return fmt.Errorf("enclaves[%d].dynamic must declare allowed-owners or allowed-repositories", index)
	}
	for ownerIndex, owner := range policy.AllowedOwners {
		if !enclaveDynamicOwnerPattern.MatchString(owner) {
			return fmt.Errorf("enclaves[%d].dynamic.allowed-owners[%d] must be a canonical lowercase ASCII owner matching ^[a-z0-9](?:[a-z0-9-]{0,38})$", index, ownerIndex)
		}
	}
	for repoIndex, repo := range policy.AllowedRepositories {
		if !isCanonicalDynamicRepositorySelector(repo) {
			return fmt.Errorf("enclaves[%d].dynamic.allowed-repositories[%d] must use canonical dynamic selector form ^[a-z0-9](?:[a-z0-9-]{0,38})/(?!\\.\\.?$)(?!.*\\.\\.)[a-z0-9._-]{1,100}$", index, repoIndex)
		}
	}
	switch policy.Sensitivity {
	case "public", "trusted", "internal", "confidential", "sealed":
	default:
		return fmt.Errorf("enclaves[%d].dynamic.sensitivity must be public, trusted, internal, confidential, or sealed", index)
	}
	if policy.GitHubPolicy != enclaveDynamicGitHubPolicy {
		return fmt.Errorf("enclaves[%d].dynamic.github-policy must be %q", index, enclaveDynamicGitHubPolicy)
	}
	if policy.MaxRepositories <= 0 {
		return fmt.Errorf("enclaves[%d].dynamic.max-repositories must be a positive finite integer", index)
	}
	if policy.Quotas == nil || policy.Quotas.MaxInvocations <= 0 || policy.Quotas.MaxOutputBytes <= 0 || policy.Quotas.MaxExecutionSeconds <= 0 {
		return fmt.Errorf("enclaves[%d].dynamic.quotas must declare positive max-invocations, max-output-bytes, and max-execution-seconds", index)
	}
	if len(policy.AuditLabels) == 0 {
		return fmt.Errorf("enclaves[%d].dynamic.audit-labels must contain at least one audit label", index)
	}
	for labelIndex, label := range policy.AuditLabels {
		if !enclaveAuditLabelPattern.MatchString(label) {
			return fmt.Errorf("enclaves[%d].dynamic.audit-labels[%d] must match [A-Za-z0-9][A-Za-z0-9_.:-]{0,127}", index, labelIndex)
		}
	}
	if err := validateDynamicEnclaveBounds(index, enclave, policy); err != nil {
		return err
	}
	if github := enclaveGitHubToolsConfig(enclave); github != nil {
		if len(github.AllowedRepos) > 0 || github.MinIntegrity != "" {
			return fmt.Errorf("enclaves[%d].agent.tools.github cannot set allowed-repos or min-integrity with dynamic repository policy", index)
		}
		if len(github.Allowed) > 0 && !sameStringSet(github.Allowed, enclaveAgentGitHubDefaultTools) {
			return fmt.Errorf("enclaves[%d].agent.tools.github.allowed must match %s when dynamic.github-policy is %q", index, strings.Join(enclaveAgentGitHubDefaultTools, ", "), enclaveDynamicGitHubPolicy)
		}
	}
	return nil
}

func validateDynamicEnclaveBounds(index int, enclave *EnclaveConfig, policy *DynamicEnclavePolicy) error {
	expiresAt, err := time.Parse(time.RFC3339, policy.ExpiresAt)
	if err != nil {
		return fmt.Errorf("enclaves[%d].dynamic.expires-at must be an absolute RFC3339 timestamp: %w", index, err)
	}
	if expiresAt.After(time.Now().UTC().Add(time.Duration(enclave.Timeout) * time.Second)) {
		return fmt.Errorf("enclaves[%d].dynamic.expires-at must not exceed the enclave job lifetime", index)
	}
	cpuLimit, err := strconv.ParseFloat(enclave.CPULimit, 64)
	if err != nil || cpuLimit <= 0 {
		return fmt.Errorf("enclaves[%d].cpu-limit must be a positive finite value", index)
	}
	if enclave.Timeout <= 0 || enclave.MemoryLimit == "" || enclave.PIDsLimit <= 0 || enclave.TmpfsLimit == "" || enclave.MaxOutputBytes <= 0 || enclave.MaxInvocations <= 0 {
		return fmt.Errorf("enclaves[%d] dynamic agent entries must declare finite timeout, memory-limit, cpu-limit, pids-limit, tmpfs-limit, max-output-bytes, and max-invocations", index)
	}
	if enclave.Agent.MaxTaskBytes <= 0 || enclave.Agent.MaxModelRequests <= 0 || enclave.Agent.MaxModelTokens <= 0 {
		return fmt.Errorf("enclaves[%d].agent dynamic entries must declare finite max-task-bytes, max-model-requests, and max-model-tokens", index)
	}
	return nil
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, value := range a {
		seen[value]++
	}
	for _, value := range b {
		seen[value]--
		if seen[value] < 0 {
			return false
		}
	}
	return true
}

func isCanonicalDynamicRepositorySelector(repo string) bool {
	if !enclaveDynamicSelectorPattern.MatchString(repo) {
		return false
	}
	parts := strings.SplitN(repo, "/", 2)
	return len(parts) == 2 && parts[1] != "." && parts[1] != ".." && !strings.Contains(parts[1], "..")
}

func validateEnclaveGitHubTools(index int, enclave *EnclaveConfig) error {
	github := enclaveGitHubToolsConfig(enclave)
	if github == nil {
		return nil
	}
	if len(github.Allowed) == 0 {
		return fmt.Errorf("enclaves[%d].agent.tools.github.allowed must contain at least one supported tool. Supported values: list_issues, issue_read. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n      tools:\n        github:\n          allowed: [list_issues]\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
	}
	for _, tool := range github.Allowed {
		if _, ok := enclaveAgentGitHubSupportedTools[tool]; !ok {
			return fmt.Errorf("enclaves[%d].agent.tools.github.allowed contains unsupported tool %q. Supported values: list_issues, issue_read. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n      tools:\n        github:\n          allowed: [list_issues, issue_read]\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, tool)
		}
	}
	if github.MinIntegrity != "" {
		if _, ok := enclaveAgentGitHubValidIntegrityLevels[github.MinIntegrity]; !ok {
			return fmt.Errorf("enclaves[%d].agent.tools.github.min-integrity must be one of: none, unapproved, approved, merged. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n      tools:\n        github:\n          allowed: [list_issues]\n          min-integrity: approved\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index)
		}
	}
	if len(github.AllowedRepos) == 0 {
		return nil
	}
	allowedRepos := make(map[string]struct{}, len(enclave.Repos))
	for _, repo := range enclave.Repos {
		if repo != nil {
			allowedRepos[strings.ToLower(repo.Repo)] = struct{}{}
		}
	}
	for _, repo := range github.AllowedRepos {
		if _, ok := allowedRepos[strings.ToLower(repo)]; !ok {
			return fmt.Errorf("enclaves[%d].agent.tools.github.allowed-repos entry %q must be declared in enclaves[%d].repos. Example:\n\nenclaves:\n  - agent:\n      model: gpt-5\n      tools:\n        github:\n          allowed: [list_issues]\n          allowed-repos: [org/my-repo]\n    repos:\n      - repo: org/my-repo\n        sensitivity: confidential", index, repo, index)
		}
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
	if enclave := enclaveStaticGitHubAgentConfig(workflowData); enclave != nil {
		awfMinVersion := constants.AWFEnclaveGitHubIssuesMinVersion
		mcpgMinVersion := constants.MCPGEnclaveGitHubIssuesMinVersion
		fieldPath := fmt.Sprintf("enclaves[].agent.github.cli %q", enclaveGitHubIssuesProfile)
		if enclaveGitHubToolsConfig(enclave) != nil {
			mcpgMinVersion = constants.MCPGEnclaveAgentToolsMinVersion
			fieldPath = "enclaves[].agent.tools.github"
		}
		if err := validateEnclaveComponentVersions(workflowData, fieldPath, awfMinVersion, mcpgMinVersion); err != nil {
			return err
		}
	}
	if enclaveDynamicRepositoryPolicyEnabled(workflowData) {
		return validateEnclaveComponentVersions(
			workflowData,
			"enclaves[].dynamic",
			constants.AWFDynamicRepositoryEnclaveMinVersion,
			constants.MCPGDynamicRepositoryDelegationMinVersion,
		)
	}
	return nil
}

func validateEnclaveComponentVersions(workflowData *WorkflowData, fieldPath string, awfMinVersion, mcpgMinVersion constants.Version) error {
	firewallConfig := getFirewallConfig(workflowData)
	if !awfVersionAtLeast(firewallConfig, awfMinVersion) {
		effectiveVersion := string(constants.DefaultFirewallVersion)
		if firewallConfig != nil && firewallConfig.Version != "" {
			effectiveVersion = firewallConfig.Version
		}
		return fmt.Errorf("%s requires AWF %s or newer, but the effective version is %s", fieldPath, awfMinVersion, effectiveVersion)
	}

	effectiveVersion := string(constants.DefaultMCPGatewayVersion)
	if workflowData.SandboxConfig != nil &&
		workflowData.SandboxConfig.MCP != nil &&
		workflowData.SandboxConfig.MCP.Version != "" {
		effectiveVersion = workflowData.SandboxConfig.MCP.Version
	}
	if !versionAtLeast(effectiveVersion, string(constants.DefaultMCPGatewayVersion), string(mcpgMinVersion)) {
		return fmt.Errorf("%s requires MCPG %s or newer, but the effective version is %s; set sandbox.mcp.version to %s or newer", fieldPath, mcpgMinVersion, effectiveVersion, mcpgMinVersion)
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
		if enclave.Dynamic != nil {
			values["dynamic"] = buildAWFDynamicEnclavePolicy(enclave)
		} else {
			repos := make([]map[string]any, 0, len(enclave.Repos))
			for _, repo := range enclave.Repos {
				repos = append(repos, map[string]any{"repo": repo.Repo, "sensitivity": repo.Sensitivity})
			}
			values["repos"] = repos
		}
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
			if enclaveGitHubToolsConfig(enclave) != nil {
				agent["github"] = map[string]any{"cli": enclaveGitHubIssuesProfile}
			}
			values["agent"] = agent
		}
		result = append(result, values)
	}
	enclavesLog.Printf("Built %d AWF enclave config(s) from %d entries", len(result), len(config))
	return result
}

func buildAWFDynamicEnclavePolicy(enclave *EnclaveConfig) map[string]any {
	policy := enclave.Dynamic
	return map[string]any{
		"allowedOwners":       stringSliceOrEmpty(policy.AllowedOwners),
		"allowedRepositories": stringSliceOrEmpty(policy.AllowedRepositories),
		"sensitivity":         policy.Sensitivity,
		"executor":            "agent",
		"githubPolicy": map[string]any{
			"version": enclaveDynamicGitHubPolicy,
			"tools":   append([]string(nil), enclaveAgentGitHubDefaultTools...),
		},
		"maxRepositories": policy.MaxRepositories,
		"limits": map[string]any{
			"timeoutSeconds":   enclave.Timeout,
			"memoryLimit":      enclave.MemoryLimit,
			"cpuLimit":         enclave.CPULimit,
			"pidsLimit":        enclave.PIDsLimit,
			"tmpfsLimit":       enclave.TmpfsLimit,
			"maxOutputBytes":   enclave.MaxOutputBytes,
			"maxTaskBytes":     enclave.Agent.MaxTaskBytes,
			"maxModelRequests": enclave.Agent.MaxModelRequests,
			"maxModelTokens":   enclave.Agent.MaxModelTokens,
		},
		"quotas": map[string]any{
			"maxInvocations":      policy.Quotas.MaxInvocations,
			"maxOutputBytes":      policy.Quotas.MaxOutputBytes,
			"maxExecutionSeconds": policy.Quotas.MaxExecutionSeconds,
		},
		"auditLabels": append([]string(nil), policy.AuditLabels...),
		"expiresAt":   policy.ExpiresAt,
	}
}

func stringSliceOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
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
