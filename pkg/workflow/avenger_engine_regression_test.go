//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

// These checkout workflows previously lost shell access with GitHub inference
// (github/gh-aw#66817, #68434, #68431, #68387).
func TestCheckoutWorkflowsUseBashCapableEngine(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, name := range []string{"avenger", "daily-go-test-parallelizer", "purelock", "daily-doc-updater"} {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", name+".md"))
			require.NoError(t, err)

			result, err := parser.ExtractFrontmatterFromContent(string(content))
			require.NoError(t, err)

			engineMap, ok := result.Frontmatter["engine"].(map[string]any)
			require.True(t, ok, "engine must be a mapping")
			engineID, ok := engineMap["id"].(string)
			require.True(t, ok, "engine.id must be a string")
			modelProvider, ok := engineMap["model-provider"].(string)
			require.True(t, ok, "engine.model-provider must be a string")
			require.Equal(t, "github", modelProvider, "retain GitHub inference")

			toolsMap, ok := result.Frontmatter["tools"].(map[string]any)
			require.True(t, ok, "tools must be a mapping")
			_, bashDeclared := toolsMap["bash"]
			require.True(t, bashDeclared, "checkout workflow must declare tools.bash")
			require.False(t, isBashExplicitlyRefused(toolsMap), "checkout workflow requires shell access")

			engine, err := GetGlobalEngineRegistry().GetEngine(engineID)
			require.NoError(t, err)
			data := &WorkflowData{EngineConfig: &EngineConfig{ID: engineID, LLMProvider: LLMProviderGitHub}}
			data.Tools = toolsMap
			require.True(t, promptShellAvailable(data, "cat", "", engine.GetCapabilities()),
				"checkout workflow requires shell access")
			if name != "avenger" {
				require.Equal(t, "codex", engineID, "retain the Codex engine")
				config, err := NewCodexEngine().buildNativeConfig(data, nil)
				require.NoError(t, err)
				features := config.Defaults["features"].(map[string]any)
				require.Equal(t, true, features["shell_tool"])
				require.Equal(t, false, features["code_mode"])
				require.Equal(t, false, features["code_mode_only"])

				lock, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", name+".lock.yml"))
				require.NoError(t, err)
				compiled := decodeCodexBootstrap(t, string(lock))
				compiledFeatures := compiled["features"].(map[string]any)
				require.Equal(t, true, compiledFeatures["shell_tool"])
				require.Equal(t, false, compiledFeatures["code_mode"])
				require.Equal(t, false, compiledFeatures["code_mode_only"])
			}
		})
	}
}
