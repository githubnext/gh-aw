package workflow

import "strings"

const defaultSafeOutputsURLPolicy = "allowlist"

// resolveSafeOutputsURLPolicy returns the normalized URL policy value.
// Defaults to allowlist for backwards compatibility.
func resolveSafeOutputsURLPolicy(config *SafeOutputsConfig) string {
	if config == nil {
		return defaultSafeOutputsURLPolicy
	}
	policy := strings.ToLower(strings.TrimSpace(config.URLPolicy))
	if policy == "" {
		return defaultSafeOutputsURLPolicy
	}
	return policy
}

func formatReputationAPIKeyExpression(secretOrExpression string) string {
	value := strings.TrimSpace(secretOrExpression)
	if value == "" {
		return ""
	}
	if isGitHubActionsExpression(value) {
		return value
	}
	if strings.HasPrefix(value, "secrets.") {
		return wrapGitHubExpression(value)
	}
	return wrapGitHubExpression("secrets." + value)
}

func appendSafeOutputsURLPolicyEnvLines(lines *[]string, indent string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}
	if strings.TrimSpace(data.SafeOutputs.URLPolicy) == "" && data.SafeOutputs.Reputation == nil {
		return
	}

	policy := resolveSafeOutputsURLPolicy(data.SafeOutputs)
	*lines = append(*lines, formatYAMLEnv(indent, "GH_AW_URL_POLICY", policy))

	if data.SafeOutputs.Reputation == nil {
		return
	}

	if provider := strings.TrimSpace(data.SafeOutputs.Reputation.Provider); provider != "" {
		*lines = append(*lines, formatYAMLEnv(indent, "GH_AW_URL_REPUTATION_PROVIDER", strings.ToLower(provider)))
	}
	if apiKeyValue := formatReputationAPIKeyExpression(data.SafeOutputs.Reputation.APIKeySecret); apiKeyValue != "" {
		*lines = append(*lines, formatYAMLEnv(indent, "GH_AW_URL_REPUTATION_API_KEY", apiKeyValue))
	}
}
