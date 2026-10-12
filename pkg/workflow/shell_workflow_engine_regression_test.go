package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

// GitHub-provider Codex disables native shell execution even when bash is
// declared. These workflows need shell access for task work or CLI transport.
func TestRecurringShellWorkflowsUseBashCapableEngine(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	for _, name := range []string{
		"daily-go-test-parallelizer",
		"daily-cli-performance",
		"purelock",
		"issue-triage-agent",
	} {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", name+".md"))
			require.NoError(t, err)
			result, err := parser.ExtractFrontmatterFromContent(string(content))
			require.NoError(t, err)
			engineMap, ok := result.Frontmatter["engine"].(map[string]any)
			require.True(t, ok, "workflow must declare its engine and inference provider")
			require.Equal(t, "github", engineMap["model-provider"], "preserve GitHub inference")
			engineID, ok := engineMap["id"].(string)
			require.True(t, ok)
			engine, err := GetGlobalEngineRegistry().GetEngine(engineID)
			require.NoError(t, err)
			require.True(t, engine.GetCapabilities().BashCommandAllowlist,
				"%s requires a shell-capable GitHub inference engine; Codex disables its native shell in this mode", name)

			tools, ok := result.Frontmatter["tools"].(map[string]any)
			require.True(t, ok)
			require.False(t, HasBashExplicitRestriction(tools), "required shell execution must not be disabled")
			if name == "issue-triage-agent" {
				require.Equal(t, true, tools["cli-proxy"], "triage uses shell-based CLI transport")
			} else {
				require.Contains(t, tools, "bash", "repository work requires declared shell access")
			}
		})
	}
}
