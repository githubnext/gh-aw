// This file previously contained all MCP gateway log parsing logic.
// It has been split into concern-aligned files:
//   - gateway_logs_types.go      — type/struct definitions and constants
//   - gateway_logs_rpc.go        — parseRPCMessages, findRPCMessagesPath, buildToolCallsFromRPCMessages
//   - gateway_logs_parsing.go    — parseGatewayLogs, processGatewayLogEntry
//   - gateway_logs_aggregation.go — calculateGatewayAggregates, buildGuardPolicySummary
//   - gateway_logs_mcp.go        — extractMCPToolUsageData

package cli
