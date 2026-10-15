//go:build !integration && !windows

package workflow

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
  config: JSON.parse(fs.readFileSync(".crush.json", "utf8"))
}));
process.exitCode = Number(process.env.CRUSH_TEST_EXIT || 0);
`), 0o600))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, node, harness, node, fixture, "run", "--quiet")
	cmd.Env = append([]string{
		"PATH=" + filepath.Dir(node), "HOME=" + directory,
		"GITHUB_WORKSPACE=" + directory, "GH_AW_PROMPT=" + prompt,
		"OPENAI_API_KEY=fixture-not-a-secret", "ANTHROPIC_API_KEY=fixture-not-a-secret",
	}, env...)
	return cmd
}

func TestCrushSampleProviderExecution(t *testing.T) {
	def := loadCrushSample(t)
	require.Equal(t, "0.98.1", def.Version)
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
			var captured struct {
				Args   []string `json:"args"`
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
			assert.Equal(t, []string{"run", "--quiet", "--model", "awf-proxy/" + model, "--help $(not-a-command)\nfixture prompt"}, captured.Args)
			info, err := os.Stat(filepath.Join(directory, ".crush.json"))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
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
