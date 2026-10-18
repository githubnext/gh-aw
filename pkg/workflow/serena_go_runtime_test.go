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
	for _, tc := range []struct {
		name   string
		engine string
		mixed  bool
	}{
		{name: "copilot", engine: "copilot"},
		{name: "codex", engine: "codex"},
		{name: "mixed-languages", engine: "copilot", mixed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"serena.md", "serena-go.md"} {
				content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "mcp", name))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), content, 0o600))
			}
			imports := "  - serena-go.md\n"
			if tc.mixed {
				imports = "  - uses: serena-go.md\n    with:\n      languages: [go, typescript]\n"
			}
			source := "---\non: workflow_dispatch\nengine: " + tc.engine + "\nimports:\n" + imports + "---\nAnalyze Go with Serena.\n"
			workflowPath := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(source), 0o600))
			require.NoError(t, NewCompiler().CompileWorkflow(workflowPath))
			content, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
			require.NoError(t, err)
			lock := string(content)
			if tc.mixed {
				require.Contains(t, lock, "typescript")
			}

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
			require.NotContains(t, server, ":/root/go")
			require.Equal(t, 1, strings.Count(server, ":rw"), "only the existing workspace mount may be writable")
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
	for _, scenario := range []string{
		"mounted", "outside-mount", "missing-binary", "install-fails",
		"missing-gopls", "gopls-version-fails", "symbols-fail", "symbols-empty",
	} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			cache := filepath.Join(dir, "tool cache")
			root := filepath.Join(cache, "go", "1.26.9", "x64")
			if scenario == "outside-mount" {
				root = filepath.Join(dir, "outside")
			}
			mockBin := filepath.Join(dir, "mock-bin")
			nodeBin := filepath.Join(cache, "node", "24", "x64", "bin")
			goplsBin := filepath.Join(cache, "gh-aw", "gopls", "v0.23.0", "bin")
			require.NoError(t, os.MkdirAll(mockBin, 0o700))
			require.NoError(t, os.MkdirAll(nodeBin, 0o700))
			require.NoError(t, os.MkdirAll(filepath.Join(root, "bin"), 0o700))
			mock := `#!/bin/sh
set -eu
case "$1" in
  env)
    [ "$2" = GOROOT ]
    printf '%s\n' "$TEST_GO_ROOT"
    ;;
  install)
    [ "$2" = golang.org/x/tools/gopls@v0.23.0 ]
    [ "$GOTOOLCHAIN" = local ]
    [ "$GOBIN" = "$TEST_GOPLS_BIN" ]
    [ "$TEST_SCENARIO" != install-fails ] || exit 44
    if [ "$TEST_SCENARIO" != missing-gopls ]; then
      mkdir -p "$GOBIN"
      cp "$TEST_GOPLS_SOURCE" "$GOBIN/gopls"
    fi
    ;;
  version)
    printf 'go version mocked\n'
    ;;
  *) exit 43 ;;
esac
`
			require.NoError(t, os.WriteFile(filepath.Join(mockBin, "go"), []byte(mock), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(nodeBin, "node"), []byte("#!/bin/sh\nprintf 'node mocked\\n'\n"), 0o700))
			goplsSource := filepath.Join(dir, "mock-gopls")
			gopls := `#!/bin/sh
set -eu
if [ "$1" = version ]; then
  [ "$TEST_SCENARIO" != gopls-version-fails ] || exit 45
  printf 'golang.org/x/tools/gopls v0.23.0\n'
  exit 0
fi
[ "$1" = -remote= ]
[ "$2" = symbols ]
[ "$GOTOOLCHAIN" = local ]
[ "$GOPROXY" = off ]
[ "$GOSUMDB" = off ]
[ "$(pwd)" = "$(dirname "$3")" ]
grep -q '^func SerenaGoReady() {}$' "$3"
[ -f "$(dirname "$3")/go.mod" ]
[ "$TEST_SCENARIO" != symbols-fail ] || exit 46
if [ "$TEST_SCENARIO" != symbols-empty ]; then
  printf 'SerenaGoReady Function 3:6-3:19\n'
fi
`
			require.NoError(t, os.WriteFile(goplsSource, []byte(gopls), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(mockBin, "timeout"), []byte("#!/bin/sh\n[ \"$1\" = 60s ] || exit 47\nshift\nexec \"$@\"\n"), 0o700))
			if scenario != "missing-binary" {
				require.NoError(t, os.WriteFile(filepath.Join(root, "bin", "go"), []byte(mock), 0o700))
			}
			envFile := filepath.Join(dir, "github-env")
			require.NoError(t, os.WriteFile(envFile, nil, 0o600))
			cmd := exec.Command("bash", "-c", script)
			runnerPath := mockBin + ":" + nodeBin + ":/usr/bin:/bin"
			cmd.Env = []string{
				"PATH=" + runnerPath, "TEST_GO_ROOT=" + root, "RUNNER_TOOL_CACHE=" + cache, "GITHUB_ENV=" + envFile,
				"TEST_GOPLS_BIN=" + goplsBin, "TEST_GOPLS_SOURCE=" + goplsSource, "TEST_SCENARIO=" + scenario,
				"TMPDIR=" + dir,
			}
			output, err := cmd.CombinedOutput()
			exported, readErr := os.ReadFile(envFile)
			require.NoError(t, readErr)
			if scenario == "mounted" {
				require.NoError(t, err, "%s", output)
				require.Equal(t, "GH_AW_SERENA_GO_PATH="+root+"/bin:"+goplsBin+":"+runnerPath+"\n", string(exported))
				// Model the gateway's variable expansion with its own PATH, not the runner's.
				exposedPath := strings.TrimSuffix(strings.TrimPrefix(string(exported), "GH_AW_SERENA_GO_PATH="), "\n")
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
				probe := exec.Command("bash", "-c", "command -v go; command -v gopls; command -v node")
				probe.Env = []string{"PATH=" + resolvedPath}
				paths, err := probe.CombinedOutput()
				require.NoError(t, err, "%s", paths)
				require.Equal(t, root+"/bin/go\n"+goplsBin+"/gopls\n"+nodeBin+"/node\n", string(paths))
			} else {
				require.Error(t, err)
				if scenario != "install-fails" && scenario != "gopls-version-fails" {
					require.Contains(t, string(output), "Serena Go")
				}
				require.Empty(t, exported)
			}
			probes, err := filepath.Glob(filepath.Join(dir, "tmp.*"))
			require.NoError(t, err)
			require.Empty(t, probes, "readiness fixture must be removed on success and failure")
		})
	}
}
