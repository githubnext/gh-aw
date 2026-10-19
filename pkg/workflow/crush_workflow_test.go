//go:build !integration && !windows

package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadCrushSample(t *testing.T) *EngineDefinition {
	t.Helper()
	var sample struct {
		Engine EngineDefinition `yaml:"engine"`
	}
	readConformanceFrontmatter(t, "../../.github/workflows/shared/crush.md", &sample)
	require.NotNil(t, sample.Engine.Behaviors)
	return &sample.Engine
}

func crushHarnessCommand(t *testing.T, directory string, env ...string) *exec.Cmd {
	t.Helper()
	node, err := exec.LookPath("node")
	require.NoError(t, err)
	harness := filepath.Join(directory, "crush_harness.cjs")
	require.NoError(t, os.WriteFile(harness, []byte(loadCrushSample(t).Behaviors.HarnessScript), 0o600))
	reflectPath, err := filepath.Abs("../../actions/setup/js/awf_reflect.cjs")
	require.NoError(t, err)
	encodedPath, err := json.Marshal(reflectPath)
	require.NoError(t, err)
	reflectSource := "module.exports = { ...require(" + string(encodedPath) + `),
fetchAWFReflect: async () => JSON.parse(process.env.CRUSH_TEST_REFLECT || "{}") };`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "awf_reflect.cjs"), []byte(reflectSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".crush.json"), []byte(`{"mcp":{"fixture":{"type":"http","url":"http://127.0.0.1:1/mcp"}},"options":{"disable_metrics":true}}`), 0o600))
	prompt := filepath.Join(directory, "prompt with spaces.txt")
	require.NoError(t, os.WriteFile(prompt, []byte("--help $(not-a-command)\nfixture prompt"), 0o600))
	fixture := filepath.Join(directory, "capture.cjs")
	require.NoError(t, os.WriteFile(fixture, []byte(`
const fs = require("node:fs");
fs.writeFileSync("captured.json", JSON.stringify({
  args: process.argv.slice(2),
  prompt: fs.readFileSync(0, "utf8"),
  config: JSON.parse(fs.readFileSync(".crush.json", "utf8"))
}));
process.exitCode = Number(process.env.CRUSH_TEST_EXIT || 0);
`), 0o600))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	args := append([]string{harness, node, fixture}, loadCrushSample(t).Behaviors.Execution.Args...)
	cmd := exec.CommandContext(ctx, node, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	cmd.Env = append([]string{
		"PATH=" + filepath.Dir(node) + ":/usr/bin:/bin", "HOME=" + directory,
		"GITHUB_WORKSPACE=" + directory, "GH_AW_PROMPT=" + prompt,
		"OPENAI_API_KEY=fixture-not-a-secret", "ANTHROPIC_API_KEY=fixture-not-a-secret",
	}, env...)
	return cmd
}

func TestCrushSampleProviderExecution(t *testing.T) {
	def := loadCrushSample(t)
	require.Equal(t, "0.98.1", def.Version)
	assert.Contains(t, def.Behaviors.Execution.Args, "--verbose")
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)
	assert.True(t, engine.GetCapabilities().MCP)
	assert.Contains(t, strings.Join(flattenSteps(engine.GetInstallationSteps(&WorkflowData{
		Name: "Crush", EngineConfig: &EngineConfig{Version: def.Version},
	})), "\n"), "@charmland/crush@0.98.1")
	for _, test := range []struct {
		model, provider, providerType, prefix, key string
	}{
		{"copilot/gpt-6.1-sol", "github", "openai", "", "$OPENAI_API_KEY"},
		{"copilot/auto", "github", "openai-compat", "", "$OPENAI_API_KEY"},
		{"openai/gpt-5.4", "openai", "openai", "/v1", "$OPENAI_API_KEY"},
		{"openai/gpt-4o", "openai", "openai-compat", "/v1", "$OPENAI_API_KEY"},
		{"anthropic/claude-sonnet-4.6", "anthropic", "anthropic", "", "$ANTHROPIC_API_KEY"},
	} {
		t.Run(test.model, func(t *testing.T) {
			directory := t.TempDir()
			aliases := filepath.Join(directory, "hostaliases")
			require.NoError(t, os.WriteFile(aliases, []byte("api-proxy localhost\n"), 0o600))
			reflected, err := json.Marshal(map[string]any{
				"ok": true, "reflectData": map[string]any{"endpoints": []any{
					map[string]any{"configured": true, "provider": test.provider, "models_url": "http://api-proxy:10002" + test.prefix + "/models"},
				}},
			})
			require.NoError(t, err)
			cmd := crushHarnessCommand(t, directory,
				"CRUSH_MODEL="+test.model, "GH_AW_LLM_PROVIDER="+test.provider,
				"AWF_REFLECT_ENABLED=1", "CRUSH_TEST_REFLECT="+string(reflected), "HOSTALIASES="+aliases)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.NotContains(t, string(output), "fixture-not-a-secret")
			assert.NotContains(t, string(output), "--help $(not-a-command)")
			assert.Contains(t, string(output), "LLM endpoint discovery complete")
			assert.Contains(t, string(output), "configuration ready (provider type: "+test.providerType+"; MCP servers: 1)")
			assert.Contains(t, string(output), "prompt loaded (")
			assert.Contains(t, string(output), "starting Crush CLI")
			assert.Contains(t, string(output), "Crush CLI finished after ")
			assert.Contains(t, string(output), "(exit: 0; signal: none)")
			var captured struct {
				Args   []string `json:"args"`
				Prompt string   `json:"prompt"`
				Config struct {
					Providers map[string]struct {
						Type   string `json:"type"`
						URL    string `json:"base_url"`
						APIKey string `json:"api_key"`
					} `json:"providers"`
					MCP     map[string]any `json:"mcp"`
					Options map[string]any `json:"options"`
				} `json:"config"`
			}
			content, err := os.ReadFile(filepath.Join(directory, "captured.json"))
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(content, &captured))
			provider := captured.Config.Providers["awf-proxy"]
			assert.Equal(t, test.providerType, provider.Type)
			assert.Equal(t, "http://host.docker.internal:10002"+test.prefix, provider.URL)
			assert.Equal(t, test.key, provider.APIKey)
			assert.Contains(t, captured.Config.MCP, "fixture")
			assert.Equal(t, true, captured.Config.Options["disable_metrics"])
			_, model, ok := strings.Cut(test.model, "/")
			require.True(t, ok)
			assert.Equal(t, []string{"run", "--quiet", "--verbose", "--model", "awf-proxy/" + model}, captured.Args)
			assert.Equal(t, "--help $(not-a-command)\nfixture prompt", captured.Prompt)
			info, err := os.Stat(filepath.Join(directory, ".crush.json"))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		})
	}
}

func TestCrushHarnessDoesNotInheritOpenStdin(t *testing.T) {
	directory := t.TempDir()
	cmd := crushHarnessCommand(t, directory, "CRUSH_MODEL=openai/test-model",
		"GH_AW_LLM_PROVIDER=openai", "OPENAI_BASE_URL=http://127.0.0.1:1/v1")
	prompt := "--help $(not-a-command)\n" + strings.Repeat("large fixture prompt\n", 20000)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "prompt with spaces.txt"), []byte(prompt), 0o600))
	stdin, writer, err := os.Pipe()
	require.NoError(t, err)
	defer stdin.Close()
	defer writer.Close()
	cmd.Stdin = stdin
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	var captured struct {
		Args   []string `json:"args"`
		Prompt string   `json:"prompt"`
	}
	content, err := os.ReadFile(filepath.Join(directory, "captured.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(content, &captured))
	assert.Equal(t, prompt, captured.Prompt)
	assert.NotContains(t, strings.Join(captured.Args, " "), prompt)
	assert.NotContains(t, string(output), "$(not-a-command)")
}

func TestCrushHarnessPublishedCLI(t *testing.T) {
	command := os.Getenv("GH_AW_CRUSH_TEST_CLI")
	if command == "" {
		t.Skip("set GH_AW_CRUSH_TEST_CLI to the pinned Crush binary for local mock-provider validation")
	}
	for _, failure := range []bool{false, true} {
		name := "native MCP and inference"
		if failure {
			name = "provider rejection"
		}
		t.Run(name, func(t *testing.T) {
			requests := make(chan map[string]any, 20)
			mcpCalls := make(chan string, 20)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("invalid fixture request: %v", err)
					http.Error(w, "invalid fixture request", http.StatusBadRequest)
					return
				}
				if r.URL.Path == "/mcp" {
					method, _ := request["method"].(string)
					select {
					case mcpCalls <- method:
					default:
						t.Error("offline MCP request budget exceeded")
						http.Error(w, "offline request budget exceeded", http.StatusTooManyRequests)
						return
					}
					if strings.HasPrefix(method, "notifications/") {
						w.WriteHeader(http.StatusAccepted)
						return
					}
					var result any
					switch method {
					case "initialize":
						result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "offline-fixture", "version": "1"}}
					case "tools/list":
						result = map[string]any{"tools": []any{map[string]any{"name": "fixture_check", "description": "Read the offline fixture", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}}}}
					case "tools/call":
						params, ok := request["params"].(map[string]any)
						if !assert.True(t, ok, "fixture tool call must have parameters") {
							http.Error(w, "missing tool parameters", http.StatusBadRequest)
							return
						}
						assert.Equal(t, "fixture_check", params["name"])
						assert.Equal(t, map[string]any{}, params["arguments"])
						result = map[string]any{"content": []any{map[string]any{"type": "text", "text": "native MCP verified"}}}
					default:
						result = map[string]any{}
					}
					w.Header().Set("Content-Type", "application/json")
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request["id"], "result": result}))
					return
				}
				assert.Equal(t, "/v1/chat/completions", r.URL.Path)
				assert.Equal(t, "Bearer fixture-not-a-secret", r.Header.Get("Authorization"))
				select {
				case requests <- request:
				default:
					t.Error("offline inference request budget exceeded")
					http.Error(w, "offline request budget exceeded", http.StatusTooManyRequests)
					return
				}
				if failure {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, err := w.Write([]byte(`{"error":{"message":"offline provider rejection","type":"invalid_request_error"}}`))
					assert.NoError(t, err)
					return
				}
				delta := map[string]any{"role": "assistant", "content": "Crush integration verified"}
				finish := "stop"
				messages, err := json.Marshal(request["messages"])
				if !assert.NoError(t, err) {
					http.Error(w, "invalid fixture messages", http.StatusBadRequest)
					return
				}
				if tools, ok := request["tools"].([]any); ok && !strings.Contains(string(messages), `"role":"tool"`) {
					for _, tool := range tools {
						definition, ok := tool.(map[string]any)
						if !assert.True(t, ok, "tool must be an object") {
							http.Error(w, "invalid fixture tool", http.StatusBadRequest)
							return
						}
						function, ok := definition["function"].(map[string]any)
						if !assert.True(t, ok, "tool must define a function") {
							http.Error(w, "invalid fixture function", http.StatusBadRequest)
							return
						}
						if toolName, ok := function["name"].(string); ok && strings.Contains(toolName, "fixture_check") {
							delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{
								"index": 0, "id": "fixture-call", "type": "function", "function": map[string]any{"name": toolName, "arguments": "{}"},
							}}}
							finish = "tool_calls"
							break
						}
					}
				}
				w.Header().Set("Content-Type", "text/event-stream")
				chunk, err := json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})
				if !assert.NoError(t, err) {
					return
				}
				_, err = w.Write(append(append([]byte("data: "), chunk...), []byte("\n\ndata: [DONE]\n\n")...))
				assert.NoError(t, err)
			}))
			defer server.Close()

			directory := t.TempDir()
			cmd := crushHarnessCommand(t, directory, "CRUSH_MODEL=copilot/auto", "GH_AW_LLM_PROVIDER=github",
				"AWF_REFLECT_ENABLED=1", "CRUSH_DISABLE_DEFAULT_PROVIDERS=1", "CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1",
				"CRUSH_DISABLE_AUTO_UPDATE=1", "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1",
				"NO_PROXY=127.0.0.1,localhost", `CRUSH_TEST_REFLECT={"ok":true,"reflectData":{"endpoints":[{"provider":"copilot","configured":true,"models_url":"`+server.URL+`/v1/models"}]}}`)
			cmd.Args = append(cmd.Args[:2], append([]string{command}, loadCrushSample(t).Behaviors.Execution.Args...)...)
			config := loadCrushSample(t).Behaviors.ConfigFile.Content
			var nativeConfig map[string]any
			require.NoError(t, json.Unmarshal([]byte(config), &nativeConfig))
			nativeConfig["mcp"] = map[string]any{"fixture": map[string]any{"type": "http", "url": server.URL + "/mcp"}}
			content, err := json.Marshal(nativeConfig)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(directory, ".crush.json"), content, 0o600))
			stdin, writer, err := os.Pipe()
			require.NoError(t, err)
			defer stdin.Close()
			defer writer.Close()
			cmd.Stdin = stdin
			output, err := cmd.CombinedOutput()
			if failure {
				require.Error(t, err, "%s", output)
				assert.Contains(t, string(output), "offline provider rejection")
			} else {
				require.NoError(t, err, "%s", output)
				assert.Contains(t, string(output), "Crush integration verified")
				assert.Contains(t, string(output), "(exit: 0; signal: none)")
			}
			assert.NotContains(t, string(output), "fixture-not-a-secret")
			assert.NotContains(t, string(output), "--help $(not-a-command)")
			require.NotEmpty(t, requests)
			var transcript strings.Builder
			for len(requests) > 0 {
				request := <-requests
				assert.Equal(t, "auto", request["model"])
				messages, err := json.Marshal(request["messages"])
				require.NoError(t, err)
				transcript.Write(messages)
			}
			assert.Contains(t, transcript.String(), "--help $(not-a-command)")
			var methods []string
			for len(mcpCalls) > 0 {
				methods = append(methods, <-mcpCalls)
			}
			assert.Contains(t, methods, "initialize")
			assert.Contains(t, methods, "tools/list")
			if !failure {
				assert.Contains(t, methods, "tools/call")
				assert.Contains(t, transcript.String(), "native MCP verified")
			}
		})
	}
}

func TestCrushHarnessRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, model, provider, reflect, want string
	}{
		{"unqualified model", "gpt-5.4", "openai", `{}`, "CRUSH_MODEL must use provider/model format"},
		{"missing provider", "openai/gpt-5.4", "", `{}`, "GH_AW_LLM_PROVIDER is required"},
		{"failed discovery", "openai/gpt-5.4", "openai", `{"ok":false,"reason":"fixture unavailable"}`, "Unable to discover"},
		{"wrong provider", "openai/gpt-5.4", "openai", `{"ok":true,"reflectData":{"endpoints":[{"provider":"anthropic","configured":true,"models_url":"http://127.0.0.1:1/models"}]}}`, "No configured /reflect endpoint found for provider openai"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			cmd := crushHarnessCommand(t, directory, "CRUSH_MODEL="+test.model,
				"GH_AW_LLM_PROVIDER="+test.provider, "AWF_REFLECT_ENABLED=1", "CRUSH_TEST_REFLECT="+test.reflect)
			output, err := cmd.CombinedOutput()
			require.Error(t, err)
			assert.Contains(t, string(output), test.want)
			_, err = os.Stat(filepath.Join(directory, "captured.json"))
			assert.True(t, os.IsNotExist(err), "invalid configuration must not launch the CLI")
		})
	}
}

func TestCrushHarnessReportsChildFailure(t *testing.T) {
	cmd := crushHarnessCommand(t, t.TempDir(), "CRUSH_MODEL=openai/gpt-4o",
		"GH_AW_LLM_PROVIDER=openai", "OPENAI_BASE_URL=http://127.0.0.1:1/v1", "CRUSH_TEST_EXIT=7")
	output, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(output), "(exit: 7; signal: none)")
	assert.Contains(t, string(output), "Crush execution failed with exit code 7")
}

func TestCrushMCPAdapter(t *testing.T) {
	directory := t.TempDir()
	gateway := filepath.Join(directory, "gateway.json")
	require.NoError(t, os.WriteFile(gateway, []byte(`{"mcpServers":{
		"github":{"url":"http://old:1/mcp/github","headers":{"Authorization":"Bearer fixture"},"tools":["get_me"]},
		"stdio":{"command":"fixture-server","args":["--fixture"]},
		"safeoutputs":{"url":"http://old:1/mcp/safeoutputs"}
	}}`), 0o600))
	script := loadCrushSample(t).Behaviors.MCP.ConfigAdapter
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "-e", script)
	cmd.Env = []string{"MCP_GATEWAY_OUTPUT=" + gateway, "GITHUB_WORKSPACE=" + directory,
		"MCP_GATEWAY_PORT=1234", `GH_AW_MCP_CLI_SERVERS=["safeoutputs"]`}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	content, err := os.ReadFile(filepath.Join(directory, ".crush.json"))
	require.NoError(t, err)
	var config struct {
		MCP map[string]map[string]any `json:"mcp"`
	}
	require.NoError(t, json.Unmarshal(content, &config))
	assert.Len(t, config.MCP, 2)
	assert.Equal(t, "http://host.docker.internal:1234/mcp/github", config.MCP["github"]["url"])
	assert.Equal(t, "http", config.MCP["github"]["type"])
	assert.Equal(t, map[string]any{"Authorization": "Bearer fixture"}, config.MCP["github"]["headers"])
	assert.NotContains(t, config.MCP["github"], "tools")
	assert.Equal(t, "stdio", config.MCP["stdio"]["type"])
	assert.Equal(t, "fixture-server", config.MCP["stdio"]["command"])
	info, err := os.Stat(filepath.Join(directory, ".crush.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
