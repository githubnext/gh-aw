//go:build !integration

package workflow

import (
	"context"
	"strings"
	"testing"

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

	err := compiler.validatePluginSupport(&WorkflowData{
		AI:      "claude",
		Plugins: []string{"octo-org/agent-plugin@" + testPluginSHA},
	})
	require.ErrorContains(t, err, `plugins are not supported by engine "claude"`)
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
