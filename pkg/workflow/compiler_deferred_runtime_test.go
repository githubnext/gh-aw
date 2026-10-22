package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeferredPromptResolvesBeforeCommentMemoryAndInlineRestore(t *testing.T) {
	for _, inline := range []bool{false, true} {
		t.Run(map[bool]string{false: "runtime", true: "inline"}[inline], func(t *testing.T) {
			dir := t.TempDir()
			source := `---
on: workflow_dispatch
engine: copilot
strict: false
ambient-folders: [docs]
tools:
  comment-memory: true
jobs:
  release:
    needs: activation
    runs-on: ubuntu-latest
    outputs:
      release_id: ${{ steps.release.outputs.id }}
    steps:
      - id: release
        run: echo "id=123" >> "$GITHUB_OUTPUT"
---
# Release
Release: ${{ needs.release.outputs.release_id }}

## agent: ` + "`reader`" + `
---
description: Read the release
---
{{#runtime-import release-context.md}}
## end agent: ` + "`reader`" + `

## skill: ` + "`release`" + `
---
description: Read the release
---
{{#runtime-import release-context.md}}
## end skill: ` + "`release`" + `
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "release-context.md"), []byte("Release: ${{ needs.release.outputs.release_id }}\n"), 0600))
			sourcePath := filepath.Join(dir, "release.md")
			require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0600))
			compiler := NewCompiler()
			compiler.SetQuiet(true)
			compiler.inlinePrompt = inline
			require.NoError(t, compiler.CompileWorkflow(sourcePath))
			content, err := os.ReadFile(filepath.Join(dir, "release.lock.yml"))
			require.NoError(t, err)
			var workflow struct {
				Jobs map[string]struct {
					Steps []struct {
						Name string            `yaml:"name"`
						ID   string            `yaml:"id"`
						Env  map[string]string `yaml:"env"`
						With map[string]string `yaml:"with"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(content, &workflow))
			indexes := make(map[string]int)
			resolverCount := 0
			for i, step := range workflow.Jobs["agent"].Steps {
				if _, exists := indexes[step.Name]; !exists {
					indexes[step.Name] = i
				}
				if step.ID != constants.DeferredPromptStepID.String() {
					continue
				}
				resolverCount++
				assert.Equal(t, constants.DeferredPromptStepName, step.Name)
				assert.Equal(t, constants.AwPromptsFile, step.Env["GH_AW_PROMPT"])
				assert.Contains(t, step.With["script"], `inlineArtifacts: [{"directory":".github/agents","extension":".agent.md"},{"directory":".github/skills","extension":"/SKILL.md"}]`)
				hasReleaseOutput := false
				for _, value := range step.Env {
					hasReleaseOutput = hasReleaseOutput || value == "${{ needs.release.outputs.release_id }}"
				}
				assert.True(t, hasReleaseOutput)
			}
			require.Equal(t, 1, resolverCount)
			for _, name := range []string{
				"Download activation artifact",
				"Restore ambient folders from activation artifact",
				constants.DeferredPromptStepName,
				"Prepare comment memory files",
				"Restore inline sub-agents from activation artifact",
				"Restore inline skills from activation artifact",
			} {
				require.Contains(t, indexes, name)
			}
			assert.Less(t, indexes["Download activation artifact"], indexes[constants.DeferredPromptStepName])
			assert.Equal(t, indexes["Restore ambient folders from activation artifact"]+1, indexes[constants.DeferredPromptStepName])
			assert.Less(t, indexes[constants.DeferredPromptStepName], indexes["Prepare comment memory files"])
			assert.Less(t, indexes[constants.DeferredPromptStepName], indexes["Restore inline sub-agents from activation artifact"])
			assert.Less(t, indexes[constants.DeferredPromptStepName], indexes["Restore inline skills from activation artifact"])
		})
	}
}
