package workflow

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.AddIssueComments != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.CreateRepositorySecurityAdvisories != nil ||
		safeOutputs.AddIssueLabels != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.MissingTool != nil
}
