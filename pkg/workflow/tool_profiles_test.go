//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestGoToolProfileCompilation(t *testing.T) {
	for _, engine := range []string{"copilot", "codex", "claude", "gemini"} {
		t.Run(engine, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/profile\n\ngo 1.24\n"), 0o600))
			tools := "tools:\n  profile: go\n"
			if engine == "codex" {
				tools += "  bash: false\n  cli-proxy: false\n"
			}
			source := "---\non: workflow_dispatch\npermissions:\n  contents: read\nengine: " + engine + "\n" + tools + "---\nAnalyze the Go code.\n"
			file := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
			compiler := NewCompiler()
			compiler.gitRoot = dir
			data, err := compiler.ParseWorkflowFile(file)
			require.NoError(t, err)
			require.Equal(t, "go", data.ParsedTools.Profile)
			require.NotContains(t, data.ParsedTools.Custom, "profile")
			require.Contains(t, data.ParsedTools.Custom, "gopls")
			require.Contains(t, data.Runtimes, "go")
			require.Equal(t, "1.26", data.Runtimes["go"].(map[string]any)["version"])
			require.Contains(t, data.SandboxConfig.Agent.Mounts, "${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/bin:${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/bin:ro")
			require.Contains(t, data.MarkdownContent, "## Go tool profile")
			require.NotContains(t, strings.Join(data.ImportPaths, "\n"), "@builtin:")
			require.NoError(t, compiler.CompileWorkflow(file))
			content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
			require.NoError(t, err)
			lock := string(content)
			for _, expected := range []string{
				"Setup Go", "Setup Go tool profile",
				"go install golang.org/x/tools/gopls@v0.23.0",
				"GH_AW_TOOL_PROFILE_GO_ROOT",
				`"gopls": {`,
				"go_workspace", "go_search", "go_file_context",
				"go_package_api", "go_diagnostics", "go_symbol_references",
				`"CGO_ENABLED": "0"`,
				`"GOTOOLCHAIN": "local"`,
				`"GOTELEMETRY": "off"`,
				`node:lts-alpine@sha256:`,
				`"entrypoint": "\${GH_AW_TOOL_PROFILE_GO_DIR}/bin/gopls"`,
				`"\${GH_AW_TOOL_PROFILE_GO_DIR}/bin:\${GH_AW_TOOL_PROFILE_GO_DIR}/bin:ro"`,
				`"\${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:ro"`,
				`"\${RUNNER_TOOL_CACHE}:\${RUNNER_TOOL_CACHE}:ro"`,
				`"GOROOT": "\${GH_AW_TOOL_PROFILE_GO_ROOT}"`,
				`"PATH": "\${GH_AW_TOOL_PROFILE_GO_PATH}:/usr/local/bin:/usr/bin:/bin"`,
			} {
				require.Contains(t, lock, expected)
			}
			require.Regexp(t, `"entrypointArgs":\s*\[\s*"mcp"\s*\]`, lock)
			require.Regexp(t, `"args":\s*\[\s*"--workdir",\s*"\\\$\{GITHUB_WORKSPACE\}"\s*\]`, lock)
			require.Regexp(t, `"tools":\s*\[\s*"go_workspace",\s*"go_search",\s*"go_file_context",\s*"go_package_api",\s*"go_diagnostics",\s*"go_symbol_references"\s*\]`, lock)
			require.Less(t, strings.Index(lock, "- name: Setup Go tool profile"), strings.Index(lock, "- name: Start MCP Gateway"))
			var compiled map[string]any
			require.NoError(t, yaml.Unmarshal(content, &compiled))
			require.Contains(t, lock, "go-version: '1.26'")
			require.NotContains(t, lock, "go-version-file: go.mod")
		})
	}
}

func TestGoToolProfileImportsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "shared"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared", "go.md"), []byte("---\ntools:\n  profile: go\n---\n"), 0o600))
	for _, tc := range []struct {
		name   string
		config string
	}{
		{"imported", ""},
		{"deduplicated", "tools:\n  profile: go\n"},
		{"bash disabled", "tools:\n  bash: false\n  cli-proxy: false\n"},
		{"bash empty", "tools:\n  bash: []\n  cli-proxy: false\n"},
		{"runtime override", "runtimes:\n  go:\n    version: '1.26.9'\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "---\non: workflow_dispatch\npermissions:\n  contents: read\nimports:\n  - shared/go.md\n" + tc.config + "---\nAnalyze Go.\n"
			file := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
			data, err := NewCompiler().ParseWorkflowFile(file)
			require.NoError(t, err)
			require.Equal(t, "go", data.ParsedTools.Profile)
			require.Equal(t, 1, strings.Count(data.PreAgentSteps, "name: Setup Go tool profile"))
			if tc.name == "bash disabled" || tc.name == "bash empty" {
				require.True(t, data.BashDisabled)
			}
			if tc.name == "runtime override" {
				require.Equal(t, "1.26.9", data.Runtimes["go"].(map[string]any)["version"])
			}
		})
	}
}

func TestGoToolProfileCodexShellChoices(t *testing.T) {
	for _, imported := range []bool{false, true} {
		for _, tc := range []struct {
			name     string
			shell    string
			cliProxy bool
			disabled bool
			wantErr  bool
		}{
			{name: "unrestricted", shell: "  bash: true\n", cliProxy: true},
			{name: "wildcard", shell: "  bash: ['*']\n", cliProxy: true},
			{name: "disabled", shell: "  bash: false\n", disabled: true},
			{name: "empty", shell: "  bash: []\n", disabled: true},
			{name: "unspecified", wantErr: true},
			{name: "restricted", shell: "  bash: ['go test']\n", wantErr: true},
		} {
			t.Run(fmt.Sprintf("imported=%t/%s", imported, tc.name), func(t *testing.T) {
				dir := t.TempDir()
				profile := "tools:\n  profile: go\n"
				if imported {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\ntools:\n  profile: go\n---\nAnalyze Go.\n"), 0o600))
					profile = "imports:\n  - shared.md\ntools:\n"
				}
				source := "---\non: workflow_dispatch\npermissions:\n  contents: read\nengine: codex\n" +
					profile + tc.shell + fmt.Sprintf("  cli-proxy: %t\n---\nAnalyze Go.\n", tc.cliProxy)
				file := filepath.Join(dir, "test.md")
				require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
				err := NewCompiler().CompileWorkflow(file)
				if tc.wantErr {
					require.ErrorContains(t, err, "does not support bash command allow-listing")
					return
				}
				require.NoError(t, err)
				content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
				require.NoError(t, err)
				lock := string(content)
				require.Contains(t, lock, `"gopls": {`)
				require.Contains(t, lock, "go_symbol_references")
				if tc.disabled {
					require.Contains(t, lock, "-c features.shell_tool=false")
					require.NotContains(t, lock, "export GH_AW_MCP_CLI_SERVERS=")
					require.NotContains(t, lock, "gopls --help")
				} else {
					require.NotContains(t, lock, "-c features.shell_tool=false")
					require.Contains(t, lock, "gopls --help")
				}
			})
		}
	}
}

func TestGoToolProfileRejectsDroppedTools(t *testing.T) {
	engine, err := NewBehaviorDefinedEngine(newHarnessEngineDefinition())
	require.NoError(t, err)
	frontmatter, err := parser.ExtractFrontmatterFromContent("---\non: workflow_dispatch\ntools:\n  profile: go\n---\nAnalyze Go.\n")
	require.NoError(t, err)
	_, err = NewCompiler().resolveToolsConfiguration(frontmatter, frontmatter.Markdown, t.TempDir(), &parser.ImportsResult{}, engine, engine.GetID())
	require.ErrorContains(t, err, "tools.profile requires an engine that preserves MCP tool restrictions")
}

func TestGoToolProfileRuntimeRequirements(t *testing.T) {
	profile := "{\"go\":{\"version\":\"1.26\"},\"node\":{}}"
	runtimes, err := mergeToolProfileRuntimes(map[string]any{"go": map[string]any{"version": "1.26.9"}}, profile)
	require.NoError(t, err)
	require.Equal(t, "1.26.9", runtimes["go"].(map[string]any)["version"])
	require.Contains(t, runtimes, "node")
	runtimes, err = mergeToolProfileRuntimes(map[string]any{"go": map[string]any{"extra-field": true}}, profile)
	require.NoError(t, err)
	require.Equal(t, "1.26", runtimes["go"].(map[string]any)["version"])
	require.Equal(t, true, runtimes["go"].(map[string]any)["extra-field"])
	runtimes, err = mergeRuntimes(runtimes, "{\"go\":{\"version\":\"1.27.1\"}}")
	require.NoError(t, err)
	require.Equal(t, "1.27.1", runtimes["go"].(map[string]any)["version"])
	_, err = mergeToolProfileRuntimes(nil, "{invalid")
	require.ErrorContains(t, err, "invalid tool profile runtimes")
}

func TestGoToolProfileCLIProxy(t *testing.T) {
	for _, engine := range []string{"copilot", "claude"} {
		t.Run(engine, func(t *testing.T) {
			dir := t.TempDir()
			source := "---\non: workflow_dispatch\npermissions:\n  contents: read\nengine: " + engine + "\ntools:\n  profile: go\n  cli-proxy: true\n---\nAnalyze Go.\n"
			file := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(file, []byte(source), 0o600))
			require.NoError(t, NewCompiler().CompileWorkflow(file))
			content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
			require.NoError(t, err)
			require.Contains(t, string(content), `"gopls": {`)
			require.Contains(t, string(content), "go_symbol_references")
		})
	}
}

func TestGoToolProfileSandboxValidation(t *testing.T) {
	for _, config := range []string{
		"agent: false",
		"agent:\n    runtime: cloud-hypervisor",
		"agent:\n    runtime: nvx",
	} {
		t.Run(config, func(t *testing.T) {
			data := &WorkflowData{
				ParsedTools: NewTools(map[string]any{"profile": "go"}),
			}
			var fm map[string]any
			require.NoError(t, yaml.Unmarshal([]byte("sandbox:\n  "+config), &fm))
			data.SandboxConfig = NewCompiler().extractSandboxConfig(fm)
			require.ErrorContains(t, validateToolProfileSandbox(data), "requires an enabled Docker agent sandbox")
		})
	}
}

func TestToolProfileSchemaAndRoundTrip(t *testing.T) {
	require.NoError(t, ValidateToolsSection(map[string]any{"profile": "go"}))
	for _, value := range []any{"python", "", false, []any{"go"}, map[string]any{}} {
		fm := map[string]any{"on": "workflow_dispatch", "tools": map[string]any{"profile": value}}
		require.Error(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(fm, "profile.md"))
	}
	tools := NewTools(map[string]any{"profile": "go"})
	require.Equal(t, "go", tools.Profile)
	require.Empty(t, tools.Custom)
	require.Equal(t, "go", tools.ToMap()["profile"])
	require.Equal(t, "go", (&ToolsConfig{Profile: "go"}).ToMap()["profile"])
}

func TestGoToolProfileSetupReadiness(t *testing.T) {
	content, err := builtinToolProfiles.ReadFile("data/tool-profiles/go.md")
	require.NoError(t, err)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	steps := result.Frontmatter["pre-agent-steps"].([]any)
	script := steps[0].(map[string]any)["run"].(string)
	for _, scenario := range []string{"ready", "outside-mount", "missing-go", "install-failed", "missing-gopls", "version-failed", "symbols-failed", "empty-symbols"} {
		t.Run(scenario, func(t *testing.T) {
			checkGoToolProfileSetup(t, script, scenario)
		})
	}
}

func checkGoToolProfileSetup(t *testing.T, script, scenario string) {
	t.Helper()
	dir := t.TempDir()
	cache := filepath.Join(dir, "tool cache")
	root := filepath.Join(cache, "go", "1.26.9", "x64")
	if scenario == "outside-mount" {
		root = filepath.Join(dir, "outside")
	}
	mockBin := filepath.Join(dir, "mock-bin")
	require.NoError(t, os.MkdirAll(mockBin, 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "bin"), 0o700))
	profile := filepath.Join(dir, "gh-aw", "tool-profiles", "go")
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(profile, "bin"), 0o700) })
	require.NoError(t, os.WriteFile(filepath.Join(mockBin, "go"), []byte(goProfileMockGo), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(mockBin, "timeout"), []byte("#!/bin/sh\nshift\nexec \"$@\"\n"), 0o700))
	if scenario != "missing-go" {
		require.NoError(t, os.WriteFile(filepath.Join(root, "bin", "go"), []byte(goProfileMockGo), 0o700))
	}
	envFile, pathFile := filepath.Join(dir, "env"), filepath.Join(dir, "path")
	require.NoError(t, os.WriteFile(envFile, nil, 0o600))
	require.NoError(t, os.WriteFile(pathFile, nil, 0o600))
	command := exec.Command("bash", "-c", script)
	command.Env = []string{
		"PATH=" + mockBin + ":/usr/bin:/bin", "TEST_GO_ROOT=" + root, "TEST_SCENARIO=" + scenario,
		"RUNNER_TOOL_CACHE=" + cache, "RUNNER_TEMP=" + dir, "GITHUB_ENV=" + envFile, "GITHUB_PATH=" + pathFile,
	}
	output, err := command.CombinedOutput()
	env, readErr := os.ReadFile(envFile)
	require.NoError(t, readErr)
	path, readErr := os.ReadFile(pathFile)
	require.NoError(t, readErr)
	if scenario != "ready" {
		require.Error(t, err, "%s", output)
		require.Empty(t, env)
		require.Empty(t, path)
		return
	}
	require.NoError(t, err, "%s", output)
	require.Contains(t, string(env), "GH_AW_TOOL_PROFILE_GO_PATH="+profile+"/bin:"+root+"/bin\n")
	require.Contains(t, string(env), "GOBIN="+profile+"/cache/bin\n")
	require.DirExists(t, filepath.Join(profile, "cache", "bin"))
	require.Equal(t, profile+"/bin\n"+profile+"/cache/bin\n", string(path))
}

const goProfileMockGo = `#!/bin/sh
if [ "$1" = env ]; then
  printf '%s\n' "$TEST_GO_ROOT"
  exit 0
fi
[ "$1" = install ] || exit 10
[ "$TEST_SCENARIO" != install-failed ] || exit 11
[ "$TEST_SCENARIO" != missing-gopls ] || exit 0
cat > "$GOBIN/gopls" <<'MOCK'
#!/bin/sh
if [ "$1" = version ]; then
  [ "$TEST_SCENARIO" != version-failed ] || exit 12
  printf 'gopls v0.23.0\n'
  exit 0
fi
[ "$1" = '-remote=' ] && [ "$2" = symbols ] && [ "$3" = profile.go ] || exit 13
[ "$GOPROXY" = off ] && [ "$GOSUMDB" = off ] && [ "$GOTOOLCHAIN" = local ] || exit 14
[ -f go.mod ] && [ -f profile.go ] || exit 15
[ "$TEST_SCENARIO" != symbols-failed ] || exit 16
[ "$TEST_SCENARIO" != empty-symbols ] || exit 0
printf 'GhAwGoProfileReady Function 3:6-3:24\n'
MOCK
chmod 700 "$GOBIN/gopls"
`
