//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPiMCPToolCapabilityIndependentOfCLIProxy(t *testing.T) {
	compiler := NewCompiler()
	for _, engineID := range []string{"pi", "PI"} {
		for _, cliProxy := range []bool{false, true} {
			data := &WorkflowData{
				EngineConfig: &EngineConfig{ID: engineID},
				ParsedTools:  NewTools(map[string]any{"cli-proxy": cliProxy}),
			}
			assert.True(t, engineSupportsMCPToolCalls(compiler.engineCatalog, data),
				"Pi supports MCP regardless of the selected tool transport")
			assert.True(t, engineSupportsMCPToolCalls(nil, data),
				"registry fallback must also report native Pi MCP support")
		}
	}
}

func TestSafeOutputsPromptFollowsConfiguredTransport(t *testing.T) {
	for _, engineID := range []string{"pi", "copilot", "claude", "codex", "gemini"} {
		for _, transport := range []string{"default", "native", "cli-proxy", "gh-proxy", "no-shell"} {
			t.Run(engineID+"/"+transport, func(t *testing.T) {
				tools := map[string]any{}
				switch transport {
				case "native":
					tools["cli-proxy"] = false
				case "cli-proxy":
					tools["cli-proxy"] = true
				case "gh-proxy":
					tools["github"] = map[string]any{"mode": "gh-proxy"}
				case "no-shell":
					tools["bash"] = false
					tools["cli-proxy"] = false
				}
				data := &WorkflowData{
					EngineConfig: &EngineConfig{ID: engineID},
					Tools:        tools,
					ParsedTools:  NewTools(tools),
					SafeOutputs:  &SafeOutputsConfig{NoOp: &NoOpConfig{}},
				}
				sections := NewCompiler().collectPromptSections(data)
				var transportFiles []string
				for _, section := range sections {
					if section.IsFile && (section.Content == safeOutputsMCPTransportPromptFile ||
						section.Content == safeOutputsCLIOnlyTransportPromptFile) {
						transportFiles = append(transportFiles, section.Content)
					}
				}
				require.Len(t, transportFiles, 1)
				expected := safeOutputsMCPTransportPromptFile
				if transport == "cli-proxy" {
					expected = safeOutputsCLIOnlyTransportPromptFile
				}
				assert.Equal(t, expected, transportFiles[0])
			})
		}
	}
}
