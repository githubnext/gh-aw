//go:build !integration

package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeEditCapabilities(t *testing.T) {
	for _, id := range []string{"copilot", "claude", "gemini", "pi", "codex", "agy"} {
		t.Run(id, func(t *testing.T) {
			engine, err := GetGlobalEngineRegistry().GetEngine(id)
			require.NoError(t, err)
			assert.True(t, engine.GetCapabilities().NativeEdit)
			assert.Equal(t, id != "codex" && id != "agy", engine.GetCapabilities().NativeEditDisable)
			for _, value := range []any{true, false, nil, map[string]any{}} {
				err := NewCompiler().validateNativeEditSupport(map[string]any{"edit": value}, engine)
				if value == false && !engine.GetCapabilities().NativeEditDisable {
					require.ErrorContains(t, err, "cannot enforce tools.edit: false")
				} else {
					require.NoError(t, err)
				}
			}
			assert.NoError(t, NewCompiler().validateNativeEditSupport(nil, engine))
		})
	}
}

func TestNativeEditBooleanCompilation(t *testing.T) {
	for _, id := range []string{"copilot", "claude", "gemini", "pi", "codex", "agy"} {
		for _, value := range []string{"true", "false", "", "null", "{}"} {
			t.Run(id+"/"+value, func(t *testing.T) {
				enabled := value != "false"
				content := fmt.Sprintf("---\non: workflow_dispatch\nengine: %s\ntools:\n  edit: %s\n---\nInspect the repository.\n", id, value)
				data, err := NewCompiler(WithSkipValidation(true)).ParseWorkflowString(content, "edit.md")
				if !enabled && (id == "codex" || id == "agy") {
					require.ErrorContains(t, err, "cannot enforce tools.edit: false")
					return
				}
				require.NoError(t, err)
				assert.Equal(t, !enabled, isToolExplicitlyFalse(data.Tools["edit"]))
				assert.Equal(t, enabled, data.ParsedTools.Edit != nil)
			})
		}
	}
}

func TestNativeEditImportPreservesExplicitBoolean(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			tools, err := NewCompiler().MergeTools(map[string]any{"edit": enabled}, fmt.Sprintf(`{"edit":%t}`, !enabled))
			require.NoError(t, err)
			assert.Equal(t, enabled, tools["edit"])
		})
	}
	for _, value := range []any{nil, map[string]any{}} {
		tools, err := NewCompiler().MergeTools(map[string]any{"edit": value}, `{"edit":false}`)
		require.NoError(t, err)
		assert.Equal(t, value, tools["edit"])
		tools, err = NewCompiler().MergeTools(map[string]any{"edit": false}, `{"edit":null}`)
		require.NoError(t, err)
		assert.Equal(t, false, tools["edit"])
	}
}

func TestNativeEditDisableSurvivesDefaulting(t *testing.T) {
	for _, extra := range []string{"sandbox:\n  agent: awf\n", "safe-outputs:\n  create-pull-request:\n"} {
		t.Run(strings.Split(extra, ":")[0], func(t *testing.T) {
			content := "---\non: workflow_dispatch\nengine: copilot\ntools:\n  edit: false\n" + extra + "---\nInspect the repository.\n"
			data, err := NewCompiler(WithSkipValidation(true)).ParseWorkflowString(content, "edit.md")
			require.NoError(t, err)
			assert.Equal(t, false, data.Tools["edit"])
			assert.Nil(t, data.ParsedTools.Edit)
			assert.Contains(t, data.ExplicitlyDisabledTools, "edit")
			args := NewCopilotEngine().buildCopilotFeatureArgs(data, nil, nil)
			assert.Contains(t, strings.Join(args, " "), "--deny-tool write")
		})
	}
}

func TestNativeEditDeclarativeCapabilities(t *testing.T) {
	content := `---
engine:
  id: editor
  display-name: Editor
  behaviors:
    capabilities:
      native-edit: true
      native-edit-disable: true
    execution:
      command-name: editor
---
Task
`
	parsed, err := parser.ExtractFrontmatterFromContent(content)
	require.NoError(t, err)
	require.NoError(t, parser.ValidateIncludedFileFrontmatterWithSchemaAndLocation(parsed.Frontmatter, "engine.md"))
	encoded, err := yaml.Marshal(parsed.Frontmatter)
	require.NoError(t, err)
	var definition engineDefinitionFile
	require.NoError(t, yaml.Unmarshal(encoded, &definition))
	engine, err := NewBehaviorDefinedEngine(&definition.Engine)
	require.NoError(t, err)
	assert.True(t, engine.GetCapabilities().NativeEdit)
	assert.True(t, engine.GetCapabilities().NativeEditDisable)
}

func TestNativeEditValidationBeforeToolFallback(t *testing.T) {
	definition := newHarnessEngineDefinition()
	unsupported := false
	definition.Behaviors.Capabilities.NativeEdit = &unsupported
	engine, err := NewBehaviorDefinedEngine(definition)
	require.NoError(t, err)
	require.NoError(t, NewCompiler().validateNativeEditSupport(nil, engine))
	for _, value := range []string{"true", "false", "", "null", "{}"} {
		t.Run(value, func(t *testing.T) {
			frontmatter, err := parser.ExtractFrontmatterFromContent(fmt.Sprintf("---\ntools:\n  edit: %s\n---\nTask\n", value))
			require.NoError(t, err)
			_, err = NewCompiler().resolveToolsConfiguration(frontmatter, frontmatter.Markdown, t.TempDir(), &parser.ImportsResult{}, engine, engine.GetID())
			require.ErrorContains(t, err, fmt.Sprintf("tools.edit: %t", value != "false"))
		})
	}
}

func TestNativeEditDoesNotEnableShell(t *testing.T) {
	for _, value := range []any{true, false, nil, map[string]any{}} {
		t.Run(fmt.Sprintf("%v", value), func(t *testing.T) {
			enabled := !isToolExplicitlyFalse(value)
			tools := map[string]any{"edit": value, "bash": false}
			data := &WorkflowData{Tools: tools, ParsedTools: NewTools(tools), BashDisabled: true}

			copilot := NewCopilotEngine()
			toolArgs := copilot.computeCopilotToolArguments(tools, nil, nil, data)
			assert.Equal(t, enabled, strings.Contains(strings.Join(toolArgs, " "), "--allow-tool write"))
			assert.NotContains(t, strings.Join(toolArgs, " "), "shell")
			args := copilot.buildCopilotFeatureArgs(data, nil, toolArgs)
			assert.Equal(t, !enabled, strings.Contains(strings.Join(args, " "), "--deny-tool write"))

			claude := NewClaudeEngine()
			allowed := claude.computeAllowedClaudeToolsString(tools, nil, nil, nil, nil, nil)
			denied := claudeDisabledTools(data, allowed)
			for _, name := range []string{"Edit", "Write", "MultiEdit", "NotebookEdit"} {
				assert.Equal(t, !enabled, slices.Contains(denied, name), "Claude %s", name)
			}
			assert.NotContains(t, allowed, "Bash")

			gemini := computeGeminiToolsCore(tools)
			assert.Equal(t, enabled, slices.Contains(gemini, "write_file"))
			assert.Equal(t, enabled, slices.Contains(gemini, "replace"))
			assert.NotContains(t, gemini, "run_shell_command")

			piArgs := strings.Join(NewPiEngine().buildPiArgs(data), " ")
			assert.Contains(t, piArgs, "--exclude-tools bash")
			assert.Equal(t, !enabled, strings.Contains(piArgs, "--exclude-tools edit,write"))
			var piPolicy map[string]any
			require.NoError(t, json.Unmarshal([]byte(piToolPolicyJSON(data)), &piPolicy))
			assert.Equal(t, value, piPolicy["edit"])
		})
	}
}

func TestCopilotNativeEditDenyOverridesWildcardBash(t *testing.T) {
	engine := NewCopilotEngine()
	data := &WorkflowData{Tools: map[string]any{"bash": []any{"*"}, "edit": false}}
	toolArgs := engine.computeCopilotToolArguments(data.Tools, nil, nil, data)
	require.Equal(t, []string{"--allow-all-tools"}, toolArgs)
	args := engine.buildCopilotFeatureArgs(data, nil, toolArgs)
	assert.Contains(t, strings.Join(args, " "), "--deny-tool write")
	assert.NotContains(t, args, "--allow-all-paths")
}
