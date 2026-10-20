//go:build !integration && !windows

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGooseReleaseRuntime(t *testing.T) {
	// Native runtime coverage is opt-in; unit tests do not replace it or verify the supplied binary's checksum.
	binary := os.Getenv("GH_AW_TEST_GOOSE_BINARY")
	if binary == "" {
		t.Skip("set GH_AW_TEST_GOOSE_BINARY to a verified Goose v" + loadGooseSample(t).Version + " binary")
	}
	dir := t.TempDir()
	var toolCalls, inferenceCalls atomic.Int32
	server := httptest.NewServer(gooseRuntimeHandler(t, &toolCalls, &inferenceCalls))
	defer server.Close()
	harness := filepath.Join(dir, "goose_harness.cjs")
	require.NoError(t, os.WriteFile(harness, []byte(loadGooseSample(t).Behaviors.HarnessScript), 0o600))
	reflect := `exports.fetchAWFReflect = async () => ({ok: true, reflectData: {}});
exports.resolveOpenAICompatibleEndpointFromReflect = () => ({provider: "github", host: process.env.TEST_ENDPOINT, basePath: "chat/completions"});`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "awf_reflect.cjs"), []byte(reflect), 0o600))
	prompt := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(prompt, []byte("Call the probe echo tool once, then answer Done."), 0o600))
	config := filepath.Join(dir, "mcp.json")
	servers := map[string]map[string]any{
		"probe": {"url": server.URL + "/mcp/probe", "headers": map[string]string{"Authorization": "Bearer test-only"}},
	}
	servers["probe"]["tools"] = []string{"echo"}
	content, err := json.Marshal(map[string]any{"mcpServers": servers})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(config, content, 0o600))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", harness, binary)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "TMPDIR=" + dir,
		"GH_AW_ENGINE_VERSION=" + loadGooseSample(t).Version,
		"GH_AW_MCP_CONFIG=" + config, "GH_AW_PROMPT=" + prompt,
		"GOOSE_MODEL=copilot/test-model", "GOOSE_MODE=auto",
		"GOOSE_DISABLE_SESSION_NAMING=true", "GH_AW_MAX_TURNS=3",
		"OPENAI_API_KEY=test-only", "AWF_REFLECT_ENABLED=1",
		"GH_AW_LLM_PROVIDER=github", "TEST_ENDPOINT=" + server.URL}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	assert.NotContains(t, string(output), "Failed to start extension")
	assert.Contains(t, string(output), `"type":"complete"`)
	assert.Contains(t, string(output), `"name":"probe__echo"`)
	assert.Equal(t, int32(1), toolCalls.Load(), "real CLI must perform an authenticated MCP call")
	assert.Equal(t, int32(2), inferenceCalls.Load(), "real CLI must resume inference after the tool result")
	parser := exec.Command("node", "-e", `const fs = require("fs"); const { parseGooseLog } = require("./parse_goose_log.cjs"); process.stdout.write(JSON.stringify(parseGooseLog(fs.readFileSync(0, "utf8")).logEntries));`)
	parser.Dir = "../../actions/setup/js"
	parser.Stdin = strings.NewReader(string(output))
	parsed, err := parser.CombinedOutput()
	require.NoError(t, err, "%s", parsed)
	var events []struct {
		Type string `json:"type"`
		Data struct {
			ToolName string `json:"toolName"`
			Usage    struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(parsed, &events))
	var toolResult, usageResult bool
	for _, event := range events {
		assert.Contains(t, event.Type, ".")
		if event.Type == "tool.execution_complete" && event.Data.ToolName == "probe__echo" {
			toolResult = true
		}
		if event.Type == "session.result" {
			usageResult = true
			assert.Equal(t, 20, event.Data.Usage.InputTokens)
			assert.Equal(t, 10, event.Data.Usage.OutputTokens)
		}
	}
	assert.True(t, toolResult, "native MCP call must survive unified-session parsing")
	assert.True(t, usageResult, "native usage must survive unified-session parsing")
}

func gooseRuntimeHandler(t *testing.T, tools, inference *atomic.Int32) http.Handler {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "probe", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo a test value"}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		tools.Add(1)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "probe passed"}}}, nil, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "excluded", Description: "Tool excluded by the configured allowlist"}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		t.Error("Goose called a tool outside the configured allowlist")
		return nil, nil, errors.New("tool is not allowed")
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{JSONResponse: true, Stateless: true})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat/completions" {
			gooseRuntimeCompletion(t, w, r, inference.Add(1))
			return
		}
		if r.URL.Path != "/mcp/probe" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		assert.Equal(t, "Bearer test-only", r.Header.Get("Authorization"))
		handler.ServeHTTP(w, r)
	})
}

func gooseRuntimeCompletion(t *testing.T, w http.ResponseWriter, r *http.Request, turn int32) {
	t.Helper()
	var request struct {
		Model  string           `json:"model"`
		Stream bool             `json:"stream"`
		Tools  []map[string]any `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Errorf("decode test completion: %v", err)
		http.Error(w, "invalid test request", http.StatusBadRequest)
		return
	}
	assert.Equal(t, "test-model", request.Model)
	message := map[string]any{"role": "assistant", "content": "Done"}
	finish := "stop"
	found := false
	for _, tool := range request.Tools {
		function := tool["function"].(map[string]any)
		found = found || function["name"] == "probe__echo"
		assert.NotEqual(t, "probe__excluded", function["name"], "upstream extension leases must preserve the configured tool allowlist")
	}
	if turn == 1 {
		assert.True(t, found, "authenticated MCP tool must be advertised on the first model turn")
		message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{
			map[string]any{"index": 0, "id": "probe-call", "type": "function", "function": map[string]string{"name": "probe__echo", "arguments": "{}"}},
		}}
		finish = "tool_calls"
	}
	result := map[string]any{"id": fmt.Sprintf("test-completion-%d", turn), "object": "chat.completion", "model": "test-model",
		"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": finish}},
		"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}}
	if request.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		result["object"] = "chat.completion.chunk"
		result["choices"] = []any{map[string]any{"index": 0, "delta": message, "finish_reason": finish}}
		data, err := json.Marshal(result)
		if err != nil {
			t.Errorf("marshal test completion: %v", err)
			return
		}
		_, err = fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
		if err != nil {
			t.Errorf("write streaming test completion: %v", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		t.Errorf("encode test completion: %v", err)
	}
}

func TestGooseRuntimeCompletion(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%t", stream), func(t *testing.T) {
			for turn := int32(1); turn <= 2; turn++ {
				body := fmt.Sprintf(`{"model":"test-model","stream":%t,"tools":[{"function":{"name":"probe__echo"}}]}`, stream)
				request := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(body))
				response := httptest.NewRecorder()
				gooseRuntimeCompletion(t, response, request, turn)
				require.Equal(t, http.StatusOK, response.Code)
				data := response.Body.String()
				if stream {
					assert.Equal(t, "text/event-stream", response.Header().Get("Content-Type"))
					require.Contains(t, data, "\n\ndata: [DONE]\n\n")
					data = strings.TrimSuffix(strings.TrimPrefix(data, "data: "), "\n\ndata: [DONE]\n\n")
				}
				var completion struct {
					Choices []struct {
						FinishReason string `json:"finish_reason"`
					} `json:"choices"`
				}
				require.NoError(t, json.Unmarshal([]byte(data), &completion))
				require.Len(t, completion.Choices, 1)
				if turn == 1 {
					assert.Equal(t, "tool_calls", completion.Choices[0].FinishReason)
					assert.Contains(t, data, "probe__echo")
				} else {
					assert.Equal(t, "stop", completion.Choices[0].FinishReason)
					assert.Contains(t, data, "Done")
				}
			}
		})
	}
}

func TestGooseSmokeEvidenceContract(t *testing.T) {
	source, err := os.ReadFile("../../.github/workflows/smoke-goose.md")
	require.NoError(t, err)
	for _, expected := range []string{
		"max-turns: 30", "max-ai-credits: 20", "fetch-homepage:",
		"model: copilot/claude-haiku-5.5",
		"Assert Goose smoke evidence", "if: always()", `"Goose execution failed"`,
		`["bash", "build", "fileWrite", "runtime", "webFetch"]`,
		`"No successful native GitHub MCP round-trip"`,
		"mcpscripts fetch-homepage", "safeoutputs create-issue",
	} {
		assert.Contains(t, string(source), expected)
	}
	assert.NotContains(t, string(source), "Use the web-fetch MCP tool")
	assert.NotContains(t, string(source), "shared/gh.md", "native MCP smoke testing must not enable gh-proxy mode")
	parts := strings.SplitN(string(source), "---", 3)
	require.Len(t, parts, 3)
}
