package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	enclaveGitHubProxyContainerEnv = "AWF_ENCLAVE_GITHUB_PROXY_CONTAINER"
	enclaveGitHubProxyIdentityEnv  = "AWF_ENCLAVE_GITHUB_PROXY_IDENTITY"
	enclaveGitHubProxyCACertEnv    = "AWF_ENCLAVE_GITHUB_PROXY_CA_CERT"
	enclaveGitHubProxyRootKeyEnv   = "MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY"

	enclaveGitHubProxyPolicyEnv    = "MCP_GATEWAY_ENCLAVE_POLICY_JSON"
	enclaveGitHubProxyAliasEnv     = "ENCLAVE_GITHUB_PROXY_ALIAS"
	enclaveGitHubProxyRunLabel     = "com.github.gh-aw.enclave-github.run"
	enclaveGitHubProxyContainer    = "awmg-enclave-github-proxy"
	enclaveGitHubProxyNetworkAlias = "awf-enclave-github-proxy"
	enclaveGitHubProxyAudience     = "gh-aw-enclave-github"
	enclaveGitHubProxyPort         = 18443
	enclaveGitHubProxyMaxTTL       = 600
)

type enclaveGitHubProxyPolicy struct {
	Version                int                             `json:"version"`
	WorkflowRunID          string                          `json:"workflow_run_id"`
	Profile                string                          `json:"profile"`
	Audience               string                          `json:"audience"`
	Repositories           []enclaveGitHubPolicyRepository `json:"repositories"`
	PublicMinimumIntegrity string                          `json:"public_min_integrity"`
	AllowedOperations      []string                        `json:"allowed_operations"`
	MaxCapabilityTTL       int                             `json:"max_capability_ttl_seconds"`
}

type enclaveGitHubPolicyRepository struct {
	Repo        string `json:"repo"`
	Sensitivity string `json:"sensitivity"`
}

var enclaveGitHubProfileOperations = map[string][]string{
	enclaveGitHubIssuesProfile: {
		"issues.comments.list",
		"issues.get",
		"issues.list",
	},
}

func buildEnclaveGitHubProxyPolicyJSON(workflowData *WorkflowData, workflowRunID string) (string, error) {
	enclave := enclaveGitHubIssuesConfig(workflowData)
	if enclave == nil {
		return "", nil
	}
	operations, err := operationsForEnclaveGitHubProfile(enclaveGitHubIssuesProfile)
	if err != nil {
		return "", err
	}

	repositories := make([]enclaveGitHubPolicyRepository, 0, len(enclave.Repos))
	for _, repo := range enclave.Repos {
		repositories = append(repositories, enclaveGitHubPolicyRepository{
			Repo:        repo.Repo,
			Sensitivity: repo.Sensitivity,
		})
	}

	policy := enclaveGitHubProxyPolicy{
		Version:                1,
		WorkflowRunID:          workflowRunID,
		Profile:                enclaveGitHubIssuesProfile,
		Audience:               enclaveGitHubProxyAudience,
		Repositories:           repositories,
		PublicMinimumIntegrity: effectivePrimaryGitHubIntegrityFloor(workflowData),
		AllowedOperations:      operations,
		MaxCapabilityTTL:       enclaveGitHubProxyMaxTTL,
	}
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal enclave GitHub proxy policy: %w", err)
	}
	return string(policyJSON), nil
}

func operationsForEnclaveGitHubProfile(profile string) ([]string, error) {
	operations, ok := enclaveGitHubProfileOperations[profile]
	if !ok {
		return nil, fmt.Errorf("unsupported enclave GitHub profile: %s", profile)
	}
	return operations, nil
}

func effectivePrimaryGitHubIntegrityFloor(workflowData *WorkflowData) string {
	if workflowData != nil {
		if workflowData.ParsedTools != nil &&
			workflowData.ParsedTools.GitHub != nil &&
			workflowData.ParsedTools.GitHub.MinIntegrity != "" {
			return string(workflowData.ParsedTools.GitHub.MinIntegrity)
		}
		if githubTool, ok := workflowData.Tools["github"].(map[string]any); ok {
			if minIntegrity, ok := githubTool["min-integrity"].(string); ok && minIntegrity != "" {
				return minIntegrity
			}
		}
	}
	return string(GitHubIntegrityApproved)
}

func (c *Compiler) generateStartEnclaveGitHubProxyStep(yaml *strings.Builder, workflowData *WorkflowData) error {
	if !enclaveGitHubIssuesEnabled(workflowData) {
		return nil
	}

	policyTemplate, err := buildEnclaveGitHubProxyPolicyJSON(workflowData, "")
	if err != nil {
		return err
	}

	ensureDefaultMCPGatewayConfig(workflowData)
	containerImage := resolveProxyContainerImage(workflowData.SandboxConfig.MCP)
	containerImage = applyContainerPinMappingFromData(containerImage, workflowData)

	yaml.WriteString("      - name: Start Enclave GitHub Proxy\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", getEffectiveEnclaveGitHubToken())
	writeProxyUpstreamEnv(yaml)
	fmt.Fprintf(yaml, "          ENCLAVE_GITHUB_PROXY_IMAGE: %s\n", quoteYAMLEnvValue(containerImage))
	fmt.Fprintf(yaml, "          %s: %s\n", enclaveGitHubProxyAliasEnv, enclaveGitHubProxyNetworkAlias)
	fmt.Fprintf(yaml, "          ENCLAVE_GITHUB_PROXY_POLICY_TEMPLATE: %s\n", quoteYAMLEnvValue(policyTemplate))
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/start_enclave_github_proxy.sh\"\n")
	return nil
}

func (c *Compiler) generateStopEnclaveGitHubProxyStep(yaml *strings.Builder, workflowData *WorkflowData) {
	if !enclaveGitHubIssuesEnabled(workflowData) {
		return
	}

	yaml.WriteString("      - name: Stop Enclave GitHub Proxy\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/stop_enclave_github_proxy.sh\"\n")
}
