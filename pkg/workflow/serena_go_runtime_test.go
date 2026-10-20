//go:build !integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSerenaGoRuntimeExposure(t *testing.T) {
	for _, engine := range []string{"copilot", "codex"} {
		t.Run(engine, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"serena.md", "serena-go.md"} {
				content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "mcp", name))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), content, 0o600))
			}
			source := "---\non: workflow_dispatch\nengine: " + engine + "\nimports:\n  - serena-go.md\n---\nAnalyze Go with Serena.\n"
			workflowPath := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(source), 0o600))
			require.NoError(t, NewCompiler().CompileWorkflow(workflowPath))
			content, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
			require.NoError(t, err)
			lock := string(content)

			require.Contains(t, lock, `GH_AW_SERENA_GO_PATH: ${{ env.GH_AW_SERENA_GO_PATH }}`)
			require.Contains(t, lock, ` -e GH_AW_SERENA_GO_PATH`)
			start := strings.Index(lock, `"serena": {`)
			require.GreaterOrEqual(t, start, 0)
			// The first engine config may contain only client timeouts.
			serverStart := strings.Index(lock[start:], `"entrypoint": "/workspaces/serena/.venv/bin/serena"`)
			require.GreaterOrEqual(t, serverStart, 0)
			server := lock[start+serverStart:]
			end := strings.Index(server, "\n              }")
			require.Positive(t, end)
			server = server[:end]
			require.Contains(t, server, `\${GH_AW_SERENA_GO_PATH}:\${PATH}`)
			require.Contains(t, server, `\${RUNNER_TOOL_CACHE}:\${RUNNER_TOOL_CACHE}:ro`)
			require.Contains(t, server, `"start-mcp-server"`)
			require.Contains(t, server, `\${GITHUB_WORKSPACE}`)
			pathParts := strings.SplitN(server, `"PATH": "`, 2)
			require.Len(t, pathParts, 2)
			serverPath := strings.SplitN(pathParts[1], `"`, 2)[0]

			var compiled struct {
				Jobs map[string]struct {
					Steps []struct {
						Name string `yaml:"name"`
						Run  string `yaml:"run"`
					} `yaml:"steps"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(content, &compiled))
			var exposure string
			for _, step := range compiled.Jobs["agent"].Steps {
				if step.Name == "Expose Go to Serena container" {
					exposure = step.Run
				}
			}
			require.NotEmpty(t, exposure)
			require.Less(t, strings.Index(lock, "- name: Expose Go to Serena container"), strings.Index(lock, "- name: Start MCP Gateway"))
			testSerenaGoExposureScript(t, exposure, serverPath)
		})
	}
}

func testSerenaGoExposureScript(t *testing.T, script, serverPath string) {
	t.Helper()
	for _, scenario := range []string{"mounted", "outside-mount", "missing-binary"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			cache := filepath.Join(dir, "tool cache")
			root := filepath.Join(cache, "go", "1.27.3", "x64")
			if scenario == "outside-mount" {
				root = filepath.Join(dir, "outside")
			}
			mockBin := filepath.Join(dir, "mock-bin")
			nodeBin := filepath.Join(cache, "node", "24", "x64", "bin")
			require.NoError(t, os.MkdirAll(mockBin, 0o700))
			require.NoError(t, os.MkdirAll(nodeBin, 0o700))
			require.NoError(t, os.MkdirAll(filepath.Join(root, "bin"), 0o700))
			mock := "#!/bin/sh\nif [ \"$1\" = env ] && [ \"$2\" = GOROOT ]; then\n  [ -z \"${GOROOT:-}\" ] && [ \"$GOTOOLCHAIN\" = local ] || exit 1\n  printf '%s\\n' \"$TEST_GO_ROOT\"\nelse\n  [ \"$GOROOT\" = \"$TEST_GO_ROOT\" ] || exit 1\n  printf 'go version mocked\\n'\nfi\n"
			require.NoError(t, os.WriteFile(filepath.Join(mockBin, "go"), []byte(mock), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(nodeBin, "node"), []byte("#!/bin/sh\nprintf 'node mocked\\n'\n"), 0o700))
			if scenario != "missing-binary" {
				require.NoError(t, os.WriteFile(filepath.Join(root, "bin", "go"), []byte(mock), 0o700))
			}
			envFile := filepath.Join(dir, "github-env")
			require.NoError(t, os.WriteFile(envFile, nil, 0o600))
			cmd := exec.Command("bash", "-c", script)
			runnerPath := mockBin + ":" + nodeBin + ":/usr/bin:/bin"
			cmd.Env = []string{"PATH=" + runnerPath, "TEST_GO_ROOT=" + root, "RUNNER_TOOL_CACHE=" + cache, "GITHUB_ENV=" + envFile, "GOROOT=" + filepath.Join(cache, "go", "1.26.9", "x64")}
			output, err := cmd.CombinedOutput()
			exported, readErr := os.ReadFile(envFile)
			require.NoError(t, readErr)
			if scenario == "mounted" {
				require.NoError(t, err, "%s", output)
				require.Equal(t, "GOROOT="+root+"\nGH_AW_SERENA_GO_PATH="+root+"/bin:"+runnerPath+"\n", string(exported))
				// Model the gateway's variable expansion with its own PATH, not the runner's.
				exposedPath := strings.TrimSuffix(strings.TrimPrefix(strings.SplitN(string(exported), "\n", 2)[1], "GH_AW_SERENA_GO_PATH="), "\n")
				resolvedPath := os.Expand(strings.ReplaceAll(serverPath, `\$`, `$`), func(name string) string {
					switch name {
					case "GH_AW_SERENA_GO_PATH":
						return exposedPath
					case "PATH":
						return "/usr/bin:/bin"
					default:
						t.Fatalf("unexpected server PATH variable: %s", name)
						return ""
					}
				})
				probe := exec.Command("bash", "-c", "command -v go; command -v node")
				probe.Env = []string{"PATH=" + resolvedPath}
				paths, err := probe.CombinedOutput()
				require.NoError(t, err, "%s", paths)
				require.Equal(t, root+"/bin/go\n"+nodeBin+"/node\n", string(paths))
			} else {
				require.Error(t, err)
				require.Contains(t, string(output), "Serena Go runtime")
				require.Empty(t, exported)
			}
		})
	}
}
