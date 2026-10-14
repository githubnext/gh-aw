package workflow

func isAzureOIDCWithAWF(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Auth == nil {
		return false
	}
	auth := workflowData.EngineConfig.Auth
	return auth.Type == "github-oidc" && auth.Provider == "azure" &&
		auth.AzureTenantID != "" && auth.AzureClientID != "" && isFirewallEnabled(workflowData)
}
