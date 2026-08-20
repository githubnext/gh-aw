//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPluginSHA = "1f181b37d3fe5862ab590648f25a292e345b5de6"

func TestValidatePlugins(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))

	for _, plugin := range []string{
		"octo-org/agent-plugin@main",
		"octo-org/agent-plugins/plugins/example@v1.2.3",
		"octo-org/agent-plugin@" + testPluginSHA,
	} {
		t.Run(plugin, func(t *testing.T) {
			require.NoError(t, compiler.validatePlugins(&WorkflowData{Plugins: []string{plugin}}))
		})
	}

	for _, test := range []struct {
		name   string
		plugin string
		error  string
	}{
		{name: "missing ref", plugin: "octo-org/agent-plugin", error: "expected owner/repository[/path]@ref"},
		{name: "empty ref", plugin: "octo-org/agent-plugin@", error: "expected owner/repository[/path]@ref"},
		{name: "local path", plugin: "./plugins/example", error: "expected owner/repository[/path]@ref"},
		{name: "expression", plugin: "${{ inputs.plugin }}", error: "expected owner/repository[/path]@ref"},
		{name: "truncated SHA", plugin: "octo-org/agent-plugin@1f181b3", error: "truncated or malformed commit SHA"},
		{name: "unsafe ref", plugin: "octo-org/agent-plugin@main..next", error: "unsupported characters"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := compiler.validatePlugins(&WorkflowData{Plugins: []string{test.plugin}})
			require.ErrorContains(t, err, test.error)
		})
	}
}

func TestValidatePluginsMergesOverlappingVersions(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	data := &WorkflowData{Plugins: []string{
		"octo-org/agent-plugin@v1.2.0",
		"octo-org/another-plugin@main",
		"octo-org/agent-plugin@v1.3.0",
		"octo-org/agent-plugin@v1",
	}}

	require.NoError(t, compiler.validatePlugins(data))
	assert.Equal(t, []string{
		"octo-org/agent-plugin@v1.3.0",
		"octo-org/another-plugin@main",
	}, data.Plugins)
}

func TestValidatePluginsRejectsOverlappingVersionConflicts(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))

	err := compiler.validatePlugins(&WorkflowData{Plugins: []string{
		"octo-org/agent-plugin@v1",
		"octo-org/agent-plugin@v2",
	}})
	require.ErrorContains(t, err, `plugin "octo-org/agent-plugin" is declared with incompatible semantic versions "v1" and "v2"`)

	err = compiler.validatePlugins(&WorkflowData{Plugins: []string{
		"octo-org/agent-plugin@main",
		"octo-org/agent-plugin@stable",
	}})
	require.ErrorContains(t, err, `plugin "octo-org/agent-plugin" is declared with conflicting refs "main" and "stable"`)
}

func TestValidatePluginSupport(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))

	copilot := NewCopilotEngine()
	assert.True(t, copilot.GetCapabilities().Plugins)
	_, implementsInstaller := any(copilot).(PluginInstallationProvider)
	assert.True(t, implementsInstaller)

	require.NoError(t, compiler.validatePluginSupport(&WorkflowData{
		AI:      "copilot",
		Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
	}))

	claude := NewClaudeEngine()
	assert.True(t, claude.GetCapabilities().Plugins)
	_, implementsInstaller = any(claude).(PluginInstallationProvider)
	assert.True(t, implementsInstaller)

	require.NoError(t, compiler.validatePluginSupport(&WorkflowData{
		AI:      "claude",
		Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
	}))

	err := compiler.validatePluginSupport(&WorkflowData{
		AI:      "codex",
		Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
	})
	require.ErrorContains(t, err, `plugins are not supported by engine "codex"`)
}

func TestCompileWorkflowRejectsImportedPluginsOnUnsupportedEngine(t *testing.T) {
	tmpDir := testutil.TempDir(t, "unsupported-imported-plugin")
	sharedPath := filepath.Join(tmpDir, "shared.md")
	require.NoError(t, os.WriteFile(sharedPath, []byte(`---
plugins:
  - octo-org/agent-plugin@`+testPluginSHA+`
---

Shared plugin configuration.
`), 0o644))

	workflowPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
engine: codex
imports:
  - shared.md
---

Run the workflow.
`), 0o644))

	err := NewCompiler(WithVersion("dev")).CompileWorkflow(workflowPath)
	require.ErrorContains(t, err, `plugins are not supported by engine "codex"`)
}

func TestCompileWorkflowInstallsImportedPlugins(t *testing.T) {
	tmpDir := testutil.TempDir(t, "imported-plugin")
	sharedPath := filepath.Join(tmpDir, "shared.md")
	require.NoError(t, os.WriteFile(sharedPath, []byte(`---
plugins:
  - octo-org/agent-plugin@`+testPluginSHA+`
---

Shared plugin configuration.
`), 0o644))

	workflowPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
engine: copilot
imports:
  - shared.md
---

Run the workflow.
`), 0o644))

	compiler := NewCompiler(WithVersion("dev"))
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
	require.NoError(t, err)
	assert.Contains(t, string(lockContent), "name: Install agent plugin octo-org/agent-plugin")
	assert.Contains(t, string(lockContent), "ref: "+testPluginSHA)
}

func TestResolveFrontmatterPluginRefs(t *testing.T) {
	t.Run("pins a ref from the action cache", func(t *testing.T) {
		cache := NewActionCache(testutil.TempDir(t, "plugin-ref-cache"))
		cache.Set("octo-org/agent-plugins/plugins/example", "main", testPluginSHA)

		data := &WorkflowData{
			Plugins:        []string{"octo-org/agent-plugins/plugins/example@main"},
			Ctx:            context.Background(),
			ActionResolver: NewActionResolver(cache),
		}
		err := NewCompiler(WithVersion("dev")).resolveFrontmatterPluginRefs(data)

		require.NoError(t, err)
		assert.Equal(t, []string{"octo-org/agent-plugins/plugins/example@" + testPluginSHA}, data.Plugins)
	})

	t.Run("keeps an exact SHA without a resolver", func(t *testing.T) {
		data := &WorkflowData{Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA}}
		err := NewCompiler(WithVersion("dev")).resolveFrontmatterPluginRefs(data)

		require.NoError(t, err)
		assert.Equal(t, "octo-org/agent-plugin@"+testPluginSHA, data.Plugins[0])
	})

	t.Run("fails when a ref cannot be pinned", func(t *testing.T) {
		data := &WorkflowData{Plugins: []string{"octo-org/agent-plugin@main"}}
		err := NewCompiler(WithVersion("dev")).resolveFrontmatterPluginRefs(data)

		require.ErrorContains(t, err, "no GitHub reference resolver is available")
		assert.Equal(t, "octo-org/agent-plugin@main", data.Plugins[0])
	})

	t.Run("fails when resolution fails", func(t *testing.T) {
		cache := NewActionCache(testutil.TempDir(t, "plugin-ref-failure"))
		resolver := NewActionResolver(cache)
		resolver.failedResolutions[formatActionCacheKey("octo-org/agent-plugin", "missing")] = struct{}{}
		data := &WorkflowData{
			Plugins:        []string{"octo-org/agent-plugin@missing"},
			ActionResolver: resolver,
		}

		err := NewCompiler(WithVersion("dev")).resolveFrontmatterPluginRefs(data)
		require.ErrorContains(t, err, "failed to resolve")
	})
}

func TestCopilotPluginInstallationSteps(t *testing.T) {
	engine := NewCopilotEngine()

	t.Run("installs a pinned root plugin", func(t *testing.T) {
		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		checkout := strings.Join(steps[0], "\n")
		assert.Contains(t, checkout, "name: Checkout agent plugin octo-org/agent-plugin")
		assert.Contains(t, checkout, "uses: actions/checkout@")
		assert.NotContains(t, checkout, "uses: actions/checkout@v")
		assert.Contains(t, checkout, "repository: octo-org/agent-plugin")
		assert.Contains(t, checkout, "ref: "+testPluginSHA)
		assert.Contains(t, checkout, "path: .gh-aw-plugins/plugin-0")
		assert.Contains(t, checkout, "persist-credentials: false")

		install := strings.Join(steps[1], "\n")
		assert.Contains(t, install, "name: Install agent plugin octo-org/agent-plugin")
		assert.Contains(t, install, "copilot plugin install ./.gh-aw-plugins/plugin-0")
	})

	t.Run("installs a plugin from a repository subpath", func(t *testing.T) {
		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugins/plugins/example@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		assert.Contains(t, strings.Join(steps[0], "\n"), "repository: octo-org/agent-plugins")
		assert.Contains(t, strings.Join(steps[1], "\n"), "copilot plugin install ./.gh-aw-plugins/plugin-0/plugins/example")
	})

	t.Run("uses a custom engine command", func(t *testing.T) {
		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			EngineConfig: &EngineConfig{Command: "/opt/copilot"},
			Plugins:      []string{"octo-org/agent-plugin@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		assert.Contains(t, strings.Join(steps[1], "\n"), "/opt/copilot plugin install")
	})

	t.Run("returns no steps without plugins", func(t *testing.T) {
		assert.Empty(t, engine.GetPluginInstallationSteps(&WorkflowData{}))
	})
}

func TestPluginInstallationFollowsEngineInstallation(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	data := &WorkflowData{
		Name:         "plugins-order",
		AI:           "copilot",
		EngineConfig: &EngineConfig{ID: "copilot"},
		Plugins:      []string{"octo-org/agent-plugin@" + testPluginSHA},
	}
	var generated strings.Builder

	_, err := compiler.generateEngineInstallAndPreAgentSteps(&generated, data, false)
	require.NoError(t, err)

	output := generated.String()
	engineInstallIndex := strings.Index(output, "name: Install GitHub Copilot CLI")
	pluginCheckoutIndex := strings.Index(output, "name: Checkout agent plugin")
	pluginInstallIndex := strings.Index(output, "name: Install agent plugin")
	require.NotEqual(t, -1, engineInstallIndex)
	require.NotEqual(t, -1, pluginCheckoutIndex)
	require.NotEqual(t, -1, pluginInstallIndex)
	assert.Less(t, engineInstallIndex, pluginCheckoutIndex)
	assert.Less(t, pluginCheckoutIndex, pluginInstallIndex)
}

func TestClaudePluginInstallation(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("checks plugins out without a CLI install command", func(t *testing.T) {
		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugins/plugins/example@" + testPluginSHA},
		})

		require.Len(t, steps, 1)
		checkout := strings.Join(steps[0], "\n")
		assert.Contains(t, checkout, "name: Checkout agent plugin octo-org/agent-plugins/plugins/example")
		assert.Contains(t, checkout, "repository: octo-org/agent-plugins")
		assert.Contains(t, checkout, "path: .gh-aw-plugins/plugin-0")
	})

	t.Run("loads plugin directories through --plugin-dir", func(t *testing.T) {
		args, _, _ := engine.buildClaudeCliArgs(&WorkflowData{
			Plugins: []string{
				"octo-org/agent-plugin@" + testPluginSHA,
				"octo-org/agent-plugins/plugins/example@" + testPluginSHA,
			},
		}, nil, "log.txt")

		joined := strings.Join(args, " ")
		assert.Contains(t, joined, "--plugin-dir ./.gh-aw-plugins/plugin-0")
		assert.Contains(t, joined, "--plugin-dir ./.gh-aw-plugins/plugin-1/plugins/example")
	})

	t.Run("adds no plugin flags without plugins", func(t *testing.T) {
		args, _, _ := engine.buildClaudeCliArgs(&WorkflowData{}, nil, "log.txt")
		assert.NotContains(t, strings.Join(args, " "), "--plugin-dir")
	})
}

func TestBehaviorDefinedEnginePluginInstallation(t *testing.T) {
	newEngine := func(t *testing.T, plugins *EnginePluginsDefinition) *BehaviorDefinedEngine {
		t.Helper()
		engine, err := NewBehaviorDefinedEngine(&EngineDefinition{
			ID:          "custom",
			DisplayName: "Custom",
			Behaviors: &EngineBehaviorDefinition{
				Plugins:   plugins,
				Execution: &EngineExecutionDefinition{CommandName: "custom-cli"},
			},
		})
		require.NoError(t, err)
		return engine
	}

	t.Run("stages plugins in the engine plugin directory", func(t *testing.T) {
		engine := newEngine(t, &EnginePluginsDefinition{Directory: ".kiro/powers"})
		assert.True(t, engine.GetCapabilities().Plugins)

		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugins/plugins/example@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		stage := strings.Join(steps[1], "\n")
		assert.Contains(t, stage, "name: Stage agent plugin octo-org/agent-plugins/plugins/example")
		assert.Contains(t, stage, `mkdir -p ".kiro/powers"`)
		assert.Contains(t, stage, `rm -rf ".kiro/powers/example"`)
		assert.Contains(t, stage, `cp -R "./.gh-aw-plugins/plugin-0/plugins/example" ".kiro/powers/example"`)
	})

	t.Run("expands home-relative plugin directories", func(t *testing.T) {
		engine := newEngine(t, &EnginePluginsDefinition{Directory: "~/.cursor/plugins/local"})

		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		assert.Contains(t, strings.Join(steps[1], "\n"), `cp -R "./.gh-aw-plugins/plugin-0" "$HOME/.cursor/plugins/local/agent-plugin"`)
	})

	t.Run("installs plugins through the engine CLI", func(t *testing.T) {
		engine := newEngine(t, &EnginePluginsDefinition{InstallArgs: []string{"plugin", "install"}})

		steps := engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
		})

		require.Len(t, steps, 2)
		assert.Contains(t, strings.Join(steps[1], "\n"), "custom-cli plugin install ./.gh-aw-plugins/plugin-0")
	})

	t.Run("disables plugins without a plugins behavior", func(t *testing.T) {
		engine := newEngine(t, nil)
		assert.False(t, engine.GetCapabilities().Plugins)
		assert.Empty(t, engine.GetPluginInstallationSteps(&WorkflowData{
			Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
		}))
	})

	t.Run("rejects incomplete or unsafe plugin behaviors", func(t *testing.T) {
		_, err := NewBehaviorDefinedEngine(&EngineDefinition{
			ID:        "custom",
			Behaviors: &EngineBehaviorDefinition{Plugins: &EnginePluginsDefinition{}},
		})
		require.ErrorContains(t, err, "without 'directory' or 'install-args'")

		_, err = NewBehaviorDefinedEngine(&EngineDefinition{
			ID:        "custom",
			Behaviors: &EngineBehaviorDefinition{Plugins: &EnginePluginsDefinition{Directory: "../escape"}},
		})
		require.ErrorContains(t, err, "unsupported behaviors.plugins.directory")
	})
}
