// This file provides command-line interface functionality for gh-aw.
// This file (gateway_logs_policy.go) contains guard policy enforcement logic
// for MCP gateway logs — error code classification and policy summary building.

package cli

// isGuardPolicyErrorCode returns true if the JSON-RPC error code indicates a
// guard policy enforcement decision.
func isGuardPolicyErrorCode(code int) bool {
	return code >= guardPolicyErrorCodeIntegrityBelowMin && code <= guardPolicyErrorCodeAccessDenied
}

// guardPolicyReasonFromCode returns a human-readable reason string for a guard policy error code.
func guardPolicyReasonFromCode(code int) string {
	switch code {
	case guardPolicyErrorCodeAccessDenied:
		return "access_denied"
	case guardPolicyErrorCodeRepoNotAllowed:
		return "repo_not_allowed"
	case guardPolicyErrorCodeInsufficientPerms:
		return "insufficient_permissions"
	case guardPolicyErrorCodePrivateRepoDenied:
		return "private_repo_denied"
	case guardPolicyErrorCodeBlockedUser:
		return "blocked_user"
	case guardPolicyErrorCodeIntegrityBelowMin:
		return "integrity_below_minimum"
	default:
		return "unknown"
	}
}

// buildGuardPolicySummary creates a GuardPolicySummary from GatewayMetrics.
func buildGuardPolicySummary(metrics *GatewayMetrics) *GuardPolicySummary {
	summary := &GuardPolicySummary{
		TotalBlocked:        metrics.TotalGuardBlocked,
		Events:              metrics.GuardPolicyEvents,
		BlockedToolCounts:   make(map[string]int),
		BlockedServerCounts: make(map[string]int),
	}

	for _, evt := range metrics.GuardPolicyEvents {
		// Categorize by error code
		switch evt.ErrorCode {
		case guardPolicyErrorCodeIntegrityBelowMin:
			summary.IntegrityBlocked++
		case guardPolicyErrorCodeRepoNotAllowed:
			summary.RepoScopeBlocked++
		case guardPolicyErrorCodeAccessDenied:
			summary.AccessDenied++
		case guardPolicyErrorCodeBlockedUser:
			summary.BlockedUserDenied++
		case guardPolicyErrorCodeInsufficientPerms:
			summary.PermissionDenied++
		case guardPolicyErrorCodePrivateRepoDenied:
			summary.PrivateRepoDenied++
		}

		// Track per-tool blocked counts
		if evt.ToolName != "" {
			summary.BlockedToolCounts[evt.ToolName]++
		}

		// Track per-server blocked counts
		if evt.ServerID != "" {
			summary.BlockedServerCounts[evt.ServerID]++
		}
	}

	return summary
}
