// This file contains aggregation functions for MCP gateway log analysis.

package cli

import "github.com/github/gh-aw/pkg/logger"

var gatewayLogsAggregationLog = logger.New("cli:gateway_logs_aggregation")

// calculateGatewayAggregates calculates aggregate statistics
func calculateGatewayAggregates(metrics *GatewayMetrics) {
	gatewayLogsAggregationLog.Printf("Calculating gateway aggregates: servers=%d", len(metrics.Servers))
	for _, server := range metrics.Servers {
		for _, tool := range server.Tools {
			if tool.CallCount > 0 {
				tool.AvgDuration = tool.TotalDuration / float64(tool.CallCount)
			}
		}
	}
}

// buildGuardPolicySummary creates a GuardPolicySummary from GatewayMetrics.
func buildGuardPolicySummary(metrics *GatewayMetrics) *GuardPolicySummary {
	gatewayLogsAggregationLog.Printf("Building guard policy summary: totalBlocked=%d events=%d", metrics.TotalGuardBlocked, len(metrics.GuardPolicyEvents))
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
