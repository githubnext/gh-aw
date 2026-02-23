//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGatewayLogs(t *testing.T) {
	tests := []struct {
		name          string
		logContent    string
		wantServers   int
		wantRequests  int
		wantToolCalls int
		wantErrors    int
		wantErr       bool
	}{
		{
			name: "valid gateway log with tool calls",
			logContent: `{"timestamp":"2024-01-12T10:00:00Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","method":"get_repository","duration":150.5,"input_size":100,"output_size":500,"status":"success"}
{"timestamp":"2024-01-12T10:00:01Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"list_issues","method":"list_issues","duration":250.3,"input_size":50,"output_size":1000,"status":"success"}
{"timestamp":"2024-01-12T10:00:02Z","level":"info","type":"request","event":"tool_call","server_name":"playwright","tool_name":"navigate","method":"navigate","duration":500.0,"input_size":200,"output_size":300,"status":"success"}
`,
			wantServers:   2,
			wantRequests:  3,
			wantToolCalls: 3,
			wantErrors:    0,
			wantErr:       false,
		},
		{
			name: "gateway log with errors",
			logContent: `{"timestamp":"2024-01-12T10:00:00Z","level":"error","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":50.0,"status":"error","error":"connection timeout"}
{"timestamp":"2024-01-12T10:00:01Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"list_issues","duration":100.0,"status":"success"}
`,
			wantServers:   1,
			wantRequests:  2,
			wantToolCalls: 2,
			wantErrors:    1,
			wantErr:       false,
		},
		{
			name: "gateway log with multiple servers",
			logContent: `{"timestamp":"2024-01-12T10:00:00Z","level":"info","type":"request","event":"rpc_call","server_name":"github","method":"list_repos","duration":100.0,"status":"success"}
{"timestamp":"2024-01-12T10:00:01Z","level":"info","type":"request","event":"rpc_call","server_name":"playwright","method":"screenshot","duration":200.0,"status":"success"}
{"timestamp":"2024-01-12T10:00:02Z","level":"info","type":"request","event":"rpc_call","server_name":"tavily","method":"search","duration":300.0,"status":"success"}
`,
			wantServers:   3,
			wantRequests:  3,
			wantToolCalls: 3,
			wantErrors:    0,
			wantErr:       false,
		},
		{
			name:         "empty log file",
			logContent:   "",
			wantServers:  0,
			wantRequests: 0,
			wantErrors:   0,
			wantErr:      false,
		},
		{
			name: "log with invalid JSON line",
			logContent: `{"timestamp":"2024-01-12T10:00:00Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":150.5,"status":"success"}
invalid json line
{"timestamp":"2024-01-12T10:00:02Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"list_issues","duration":250.3,"status":"success"}
`,
			wantServers:   1,
			wantRequests:  2,
			wantToolCalls: 2,
			wantErrors:    0,
			wantErr:       false, // Should continue parsing after invalid line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tmpDir := t.TempDir()

			// Write the test log content
			gatewayLogPath := filepath.Join(tmpDir, "gateway.jsonl")
			err := os.WriteFile(gatewayLogPath, []byte(tt.logContent), 0644)
			require.NoError(t, err, "Failed to write test gateway.jsonl")

			// Parse the gateway logs
			metrics, err := parseGatewayLogs(tmpDir, false)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, metrics)

			// Verify metrics
			assert.Len(t, metrics.Servers, tt.wantServers, "Server count mismatch")
			assert.Equal(t, tt.wantRequests, metrics.TotalRequests, "Total requests mismatch")
			assert.Equal(t, tt.wantToolCalls, metrics.TotalToolCalls, "Total tool calls mismatch")
			assert.Equal(t, tt.wantErrors, metrics.TotalErrors, "Total errors mismatch")
		})
	}
}

func TestParseGatewayLogsFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	metrics, err := parseGatewayLogs(tmpDir, false)

	require.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "gateway.jsonl not found")
}

func TestGatewayToolMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a log with multiple calls to the same tool
	logContent := `{"timestamp":"2024-01-12T10:00:00Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":100.0,"status":"success"}
{"timestamp":"2024-01-12T10:00:01Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":200.0,"status":"success"}
{"timestamp":"2024-01-12T10:00:02Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":300.0,"status":"success"}
`

	gatewayLogPath := filepath.Join(tmpDir, "gateway.jsonl")
	err := os.WriteFile(gatewayLogPath, []byte(logContent), 0644)
	require.NoError(t, err)

	metrics, err := parseGatewayLogs(tmpDir, false)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Verify server metrics
	require.Len(t, metrics.Servers, 1)
	server := metrics.Servers["github"]
	require.NotNil(t, server)
	assert.Equal(t, "github", server.ServerName)
	assert.Equal(t, 3, server.RequestCount)

	// Verify tool metrics
	require.Len(t, server.Tools, 1)
	tool := server.Tools["get_repository"]
	require.NotNil(t, tool)
	assert.Equal(t, "get_repository", tool.ToolName)
	assert.Equal(t, 3, tool.CallCount)
	assert.InDelta(t, 600.0, tool.TotalDuration, 0.001)
	assert.InDelta(t, 200.0, tool.AvgDuration, 0.001)
	assert.InDelta(t, 300.0, tool.MaxDuration, 0.001)
	assert.InDelta(t, 100.0, tool.MinDuration, 0.001)
}

func TestRenderGatewayMetricsTable(t *testing.T) {
	// Create metrics with some data
	metrics := &GatewayMetrics{
		TotalRequests:  10,
		TotalToolCalls: 8,
		TotalErrors:    2,
		Servers: map[string]*GatewayServerMetrics{
			"github": {
				ServerName:    "github",
				RequestCount:  6,
				ToolCallCount: 5,
				TotalDuration: 600.0,
				ErrorCount:    1,
				Tools: map[string]*GatewayToolMetrics{
					"get_repository": {
						ToolName:      "get_repository",
						CallCount:     3,
						TotalDuration: 300.0,
						AvgDuration:   100.0,
						MaxDuration:   150.0,
						MinDuration:   50.0,
						ErrorCount:    0,
					},
				},
			},
			"playwright": {
				ServerName:    "playwright",
				RequestCount:  4,
				ToolCallCount: 3,
				TotalDuration: 400.0,
				ErrorCount:    1,
				Tools: map[string]*GatewayToolMetrics{
					"navigate": {
						ToolName:      "navigate",
						CallCount:     2,
						TotalDuration: 200.0,
						AvgDuration:   100.0,
						MaxDuration:   120.0,
						MinDuration:   80.0,
						ErrorCount:    0,
					},
				},
			},
		},
	}

	// Test non-verbose output
	output := renderGatewayMetricsTable(metrics, false)
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "MCP Gateway Metrics")
	assert.Contains(t, output, "Total Requests: 10")
	assert.Contains(t, output, "Total Tool Calls: 8")
	assert.Contains(t, output, "Total Errors: 2")
	assert.Contains(t, output, "Servers: 2")
	assert.Contains(t, output, "github")
	assert.Contains(t, output, "playwright")

	// Test verbose output
	verboseOutput := renderGatewayMetricsTable(metrics, true)
	assert.NotEmpty(t, verboseOutput)
	assert.Contains(t, verboseOutput, "Tool Usage Details")
	assert.Contains(t, verboseOutput, "get_repository")
	assert.Contains(t, verboseOutput, "navigate")
}

func TestRenderGatewayMetricsTableEmpty(t *testing.T) {
	// Test with nil metrics
	output := renderGatewayMetricsTable(nil, false)
	assert.Empty(t, output)

	// Test with empty metrics
	emptyMetrics := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}
	output = renderGatewayMetricsTable(emptyMetrics, false)
	assert.Empty(t, output)
}

func TestGatewayTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "string equal to max",
			input:  "exactlyten",
			maxLen: 10,
			want:   "exactlyten",
		},
		{
			name:   "string longer than max",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "max length very small",
			input:  "test",
			maxLen: 2,
			want:   "te",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringutil.Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, result)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

func TestProcessGatewayLogEntry(t *testing.T) {
	metrics := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}

	// Test request entry
	entry := &GatewayLogEntry{
		Timestamp:  "2024-01-12T10:00:00Z",
		Event:      "tool_call",
		ServerName: "github",
		ToolName:   "get_repository",
		Duration:   150.5,
		InputSize:  100,
		OutputSize: 500,
		Status:     "success",
	}

	processGatewayLogEntry(entry, metrics, false)

	assert.Equal(t, 1, metrics.TotalRequests)
	assert.Equal(t, 1, metrics.TotalToolCalls)
	assert.Equal(t, 0, metrics.TotalErrors)
	assert.Len(t, metrics.Servers, 1)

	server := metrics.Servers["github"]
	require.NotNil(t, server)
	assert.Equal(t, 1, server.RequestCount)
	assert.Equal(t, 1, server.ToolCallCount)
	assert.InDelta(t, 150.5, server.TotalDuration, 0.001)

	// Test error entry
	errorEntry := &GatewayLogEntry{
		Timestamp:  "2024-01-12T10:00:01Z",
		Event:      "tool_call",
		ServerName: "github",
		ToolName:   "list_issues",
		Status:     "error",
		Error:      "connection timeout",
	}

	processGatewayLogEntry(errorEntry, metrics, false)

	assert.Equal(t, 2, metrics.TotalRequests)
	assert.Equal(t, 1, metrics.TotalErrors)
	assert.Equal(t, 1, server.ErrorCount)
}

func TestGetSortedServerNames(t *testing.T) {
	metrics := &GatewayMetrics{
		Servers: map[string]*GatewayServerMetrics{
			"github": {
				ServerName:   "github",
				RequestCount: 10,
			},
			"playwright": {
				ServerName:   "playwright",
				RequestCount: 5,
			},
			"tavily": {
				ServerName:   "tavily",
				RequestCount: 15,
			},
		},
	}

	names := getSortedServerNames(metrics)
	require.Len(t, names, 3)

	// Should be sorted by request count (descending)
	assert.Equal(t, "tavily", names[0])
	assert.Equal(t, "github", names[1])
	assert.Equal(t, "playwright", names[2])
}

func TestGatewayLogsWithMethodField(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with method field instead of tool_name
	logContent := `{"timestamp":"2024-01-12T10:00:00Z","level":"info","type":"request","event":"rpc_call","server_name":"github","method":"tools/list","duration":100.0,"status":"success"}
{"timestamp":"2024-01-12T10:00:01Z","level":"info","type":"request","event":"rpc_call","server_name":"github","method":"tools/call","duration":200.0,"status":"success"}
`

	gatewayLogPath := filepath.Join(tmpDir, "gateway.jsonl")
	err := os.WriteFile(gatewayLogPath, []byte(logContent), 0644)
	require.NoError(t, err)

	metrics, err := parseGatewayLogs(tmpDir, false)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.Len(t, metrics.Servers, 1)
	assert.Equal(t, 2, metrics.TotalRequests)
	assert.Equal(t, 2, metrics.TotalToolCalls)

	server := metrics.Servers["github"]
	require.NotNil(t, server)
	assert.Len(t, server.Tools, 2)

	// Check that methods were tracked as tools
	assert.Contains(t, server.Tools, "tools/list")
	assert.Contains(t, server.Tools, "tools/call")
}

func TestGatewayLogsParsingIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a comprehensive test log
	logContent := `{"timestamp":"2024-01-12T10:00:00.000Z","level":"info","type":"gateway","event":"startup","message":"Gateway started"}
{"timestamp":"2024-01-12T10:00:01.123Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","method":"get_repository","duration":150.5,"input_size":100,"output_size":500,"status":"success"}
{"timestamp":"2024-01-12T10:00:02.456Z","level":"info","type":"request","event":"tool_call","server_name":"github","tool_name":"list_issues","method":"list_issues","duration":250.3,"input_size":50,"output_size":1000,"status":"success"}
{"timestamp":"2024-01-12T10:00:03.789Z","level":"error","type":"request","event":"tool_call","server_name":"github","tool_name":"get_repository","duration":50.0,"status":"error","error":"rate limit exceeded"}
{"timestamp":"2024-01-12T10:00:04.012Z","level":"info","type":"request","event":"tool_call","server_name":"playwright","tool_name":"navigate","method":"navigate","duration":500.0,"input_size":200,"output_size":300,"status":"success"}
{"timestamp":"2024-01-12T10:00:05.345Z","level":"info","type":"request","event":"tool_call","server_name":"playwright","tool_name":"screenshot","method":"screenshot","duration":300.0,"input_size":50,"output_size":2000,"status":"success"}
{"timestamp":"2024-01-12T10:00:06.678Z","level":"info","type":"gateway","event":"shutdown","message":"Gateway shutting down"}
`

	gatewayLogPath := filepath.Join(tmpDir, "gateway.jsonl")
	err := os.WriteFile(gatewayLogPath, []byte(logContent), 0644)
	require.NoError(t, err)

	metrics, err := parseGatewayLogs(tmpDir, false)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Verify overall metrics
	assert.Len(t, metrics.Servers, 2, "Should have 2 servers")
	assert.Equal(t, 5, metrics.TotalRequests, "Should have 5 requests")
	assert.Equal(t, 5, metrics.TotalToolCalls, "Should have 5 tool calls")
	assert.Equal(t, 1, metrics.TotalErrors, "Should have 1 error")

	// Verify GitHub server metrics
	githubServer := metrics.Servers["github"]
	require.NotNil(t, githubServer)
	assert.Equal(t, 3, githubServer.RequestCount)
	assert.Equal(t, 3, githubServer.ToolCallCount)
	assert.Equal(t, 1, githubServer.ErrorCount)

	// Verify Playwright server metrics
	playwrightServer := metrics.Servers["playwright"]
	require.NotNil(t, playwrightServer)
	assert.Equal(t, 2, playwrightServer.RequestCount)
	assert.Equal(t, 2, playwrightServer.ToolCallCount)
	assert.Equal(t, 0, playwrightServer.ErrorCount)

	// Verify tool metrics
	assert.Len(t, githubServer.Tools, 2)
	assert.Len(t, playwrightServer.Tools, 2)

	// Verify GitHub tools
	getRepoTool := githubServer.Tools["get_repository"]
	require.NotNil(t, getRepoTool)
	assert.Equal(t, 2, getRepoTool.CallCount)
	assert.Equal(t, 1, getRepoTool.ErrorCount)

	listIssuesTool := githubServer.Tools["list_issues"]
	require.NotNil(t, listIssuesTool)
	assert.Equal(t, 1, listIssuesTool.CallCount)
	assert.Equal(t, 0, listIssuesTool.ErrorCount)

	// Test rendering
	output := renderGatewayMetricsTable(metrics, false)
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "github")
	assert.Contains(t, output, "playwright")

	// Test verbose rendering
	verboseOutput := renderGatewayMetricsTable(metrics, true)
	assert.Contains(t, verboseOutput, "Tool Usage Details")
	assert.Contains(t, verboseOutput, "get_repository")
	assert.Contains(t, verboseOutput, "list_issues")
	assert.Contains(t, verboseOutput, "navigate")
	assert.Contains(t, verboseOutput, "screenshot")

	// Verify time range was captured
	assert.False(t, metrics.StartTime.IsZero())
	assert.False(t, metrics.EndTime.IsZero())
	assert.True(t, metrics.EndTime.After(metrics.StartTime))
}

func TestParseGatewayLogsFromMCPLogsSubdirectory(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create mcp-logs subdirectory (path after artifact download)
	mcpLogsDir := filepath.Join(tmpDir, "mcp-logs")
	err := os.MkdirAll(mcpLogsDir, 0755)
	require.NoError(t, err, "should create mcp-logs directory")

	// Create test gateway.jsonl in mcp-logs subdirectory
	testLogContent := `{"timestamp":"2024-01-15T10:00:00Z","level":"info","event":"tool_call","server_name":"github","tool_name":"search_code","duration":250}
{"timestamp":"2024-01-15T10:00:01Z","level":"info","event":"tool_call","server_name":"github","tool_name":"list_issues","duration":180}
{"timestamp":"2024-01-15T10:00:02Z","level":"error","event":"tool_call","server_name":"github","tool_name":"create_issue","duration":100}
`
	gatewayLogPath := filepath.Join(mcpLogsDir, "gateway.jsonl")
	err = os.WriteFile(gatewayLogPath, []byte(testLogContent), 0644)
	require.NoError(t, err, "should write test gateway.jsonl in mcp-logs")

	// Test parsing from mcp-logs subdirectory
	metrics, err := parseGatewayLogs(tmpDir, false)
	require.NoError(t, err, "should parse gateway logs from mcp-logs subdirectory")
	require.NotNil(t, metrics, "metrics should not be nil")

	// Verify results
	assert.Equal(t, 3, metrics.TotalRequests, "should have 3 total requests")
	assert.Len(t, metrics.Servers, 1, "should have 1 server")

	// Verify server metrics
	githubMetrics, ok := metrics.Servers["github"]
	require.True(t, ok, "should have github server metrics")
	assert.Equal(t, 3, githubMetrics.RequestCount, "should have 3 total calls for github server")
}

func TestParseRPCMessages(t *testing.T) {
	tests := []struct {
		name          string
		logContent    string
		wantServers   int
		wantRequests  int
		wantToolCalls int
		wantErrors    int
		wantErr       bool
	}{
		{
			name: "valid rpc-messages with tool calls",
			logContent: `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_repository","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:00.150000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"result":{"content":[]}}}
{"timestamp":"2024-01-12T10:00:01.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_issues","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:01.250000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"result":{"content":[]}}}
`,
			wantServers:   1,
			wantRequests:  2,
			wantToolCalls: 2,
			wantErrors:    0,
			wantErr:       false,
		},
		{
			name: "rpc-messages with error response",
			logContent: `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_repository","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:00.050000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"connection timeout"}}}
`,
			wantServers:   1,
			wantRequests:  1,
			wantToolCalls: 1,
			wantErrors:    1,
			wantErr:       false,
		},
		{
			name: "rpc-messages with multiple servers",
			logContent: `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_repos","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:00.100000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"result":{}}}
{"timestamp":"2024-01-12T10:00:01.000000000Z","direction":"OUT","type":"REQUEST","server_id":"playwright","payload":{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"navigate","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:01.500000000Z","direction":"IN","type":"RESPONSE","server_id":"playwright","payload":{"jsonrpc":"2.0","id":2,"result":{}}}
`,
			wantServers:   2,
			wantRequests:  2,
			wantToolCalls: 2,
			wantErrors:    0,
			wantErr:       false,
		},
		{
			name: "rpc-messages skips non-tools/call methods",
			logContent: `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}}
{"timestamp":"2024-01-12T10:00:00.010000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}}
{"timestamp":"2024-01-12T10:00:01.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_repository","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:01.150000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"result":{}}}
`,
			wantServers:   1,
			wantRequests:  1,
			wantToolCalls: 1,
			wantErrors:    0,
			wantErr:       false,
		},
		{
			name:         "empty file",
			logContent:   "",
			wantServers:  0,
			wantRequests: 0,
			wantErrors:   0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "rpc-messages.jsonl")
			err := os.WriteFile(logPath, []byte(tt.logContent), 0644)
			require.NoError(t, err, "should write test rpc-messages.jsonl")

			metrics, err := parseRPCMessages(logPath, false)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "parseRPCMessages should not return error")
			require.NotNil(t, metrics, "metrics should not be nil")

			assert.Len(t, metrics.Servers, tt.wantServers, "server count mismatch")
			assert.Equal(t, tt.wantRequests, metrics.TotalRequests, "total requests mismatch")
			assert.Equal(t, tt.wantToolCalls, metrics.TotalToolCalls, "total tool calls mismatch")
			assert.Equal(t, tt.wantErrors, metrics.TotalErrors, "total errors mismatch")
		})
	}
}

func TestParseGatewayLogsFallsBackToRPCMessages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mcp-logs/rpc-messages.jsonl (no gateway.jsonl present)
	mcpLogsDir := filepath.Join(tmpDir, "mcp-logs")
	require.NoError(t, os.MkdirAll(mcpLogsDir, 0755))

	rpcContent := `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_issues","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:00.200000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"result":{}}}
`
	err := os.WriteFile(filepath.Join(mcpLogsDir, "rpc-messages.jsonl"), []byte(rpcContent), 0644)
	require.NoError(t, err, "should write rpc-messages.jsonl")

	// parseGatewayLogs should fall back to rpc-messages.jsonl
	metrics, err := parseGatewayLogs(tmpDir, false)
	require.NoError(t, err, "should fall back to rpc-messages.jsonl")
	require.NotNil(t, metrics, "metrics should not be nil")

	assert.Equal(t, 1, metrics.TotalRequests, "should have 1 request from rpc-messages.jsonl")
	assert.Len(t, metrics.Servers, 1, "should have 1 server")

	_, hasGitHub := metrics.Servers["github"]
	assert.True(t, hasGitHub, "should have github server")
}

func TestFindRPCMessagesPath(t *testing.T) {
	t.Run("rpc-messages in mcp-logs subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		mcpDir := filepath.Join(tmpDir, "mcp-logs")
		require.NoError(t, os.MkdirAll(mcpDir, 0755))
		rpcPath := filepath.Join(mcpDir, "rpc-messages.jsonl")
		require.NoError(t, os.WriteFile(rpcPath, []byte("{}"), 0644))

		result := findRPCMessagesPath(tmpDir)
		assert.Equal(t, rpcPath, result, "should find rpc-messages.jsonl in mcp-logs")
	})

	t.Run("rpc-messages in root directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		rpcPath := filepath.Join(tmpDir, "rpc-messages.jsonl")
		require.NoError(t, os.WriteFile(rpcPath, []byte("{}"), 0644))

		result := findRPCMessagesPath(tmpDir)
		assert.Equal(t, rpcPath, result, "should find rpc-messages.jsonl in root")
	})

	t.Run("mcp-logs subdirectory takes priority over root", func(t *testing.T) {
		tmpDir := t.TempDir()
		mcpDir := filepath.Join(tmpDir, "mcp-logs")
		require.NoError(t, os.MkdirAll(mcpDir, 0755))
		mcpPath := filepath.Join(mcpDir, "rpc-messages.jsonl")
		rootPath := filepath.Join(tmpDir, "rpc-messages.jsonl")
		require.NoError(t, os.WriteFile(mcpPath, []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(rootPath, []byte("{}"), 0644))

		result := findRPCMessagesPath(tmpDir)
		assert.Equal(t, mcpPath, result, "mcp-logs should take priority over root")
	})

	t.Run("not found returns empty string", func(t *testing.T) {
		tmpDir := t.TempDir()
		result := findRPCMessagesPath(tmpDir)
		assert.Empty(t, result, "should return empty string when not found")
	})
}

func TestBuildToolCallsFromRPCMessages(t *testing.T) {
	tmpDir := t.TempDir()

	rpcContent := `{"timestamp":"2024-01-12T10:00:00.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_issues","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:00.200000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":1,"result":{}}}
{"timestamp":"2024-01-12T10:00:01.000000000Z","direction":"OUT","type":"REQUEST","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_repository","arguments":{}}}}
{"timestamp":"2024-01-12T10:00:01.050000000Z","direction":"IN","type":"RESPONSE","server_id":"github","payload":{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"rate limit"}}}
`
	logPath := filepath.Join(tmpDir, "rpc-messages.jsonl")
	require.NoError(t, os.WriteFile(logPath, []byte(rpcContent), 0644))

	calls, err := buildToolCallsFromRPCMessages(logPath)
	require.NoError(t, err, "should build tool calls without error")
	require.Len(t, calls, 2, "should have 2 tool calls")

	// Find each call (order may vary)
	var listIssues, getRepo *MCPToolCall
	for i := range calls {
		switch calls[i].ToolName {
		case "list_issues":
			listIssues = &calls[i]
		case "get_repository":
			getRepo = &calls[i]
		}
	}

	require.NotNil(t, listIssues, "should have list_issues call")
	assert.Equal(t, "github", listIssues.ServerName, "server name should be github")
	assert.Equal(t, "success", listIssues.Status, "status should be success")
	assert.NotEmpty(t, listIssues.Duration, "duration should be set for paired request/response")

	require.NotNil(t, getRepo, "should have get_repository call")
	assert.Equal(t, "github", getRepo.ServerName, "server name should be github")
	assert.Equal(t, "error", getRepo.Status, "status should be error")
	assert.Equal(t, "rate limit", getRepo.Error, "error message should be set")
}
