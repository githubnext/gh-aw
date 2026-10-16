//go:build !integration

package workflow

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/require"
)

func sharedGoMakeTools(t *testing.T) *MCPScriptsConfig {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "go-make.md"))
	require.NoError(t, err)
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
	require.NoError(t, err)
	scripts, ok := frontmatter.Frontmatter["mcp-scripts"].(map[string]any)
	require.True(t, ok)
	config, ok := parseMCPScriptsMap(scripts)
	require.True(t, ok)
	return config
}

func TestSharedGoMakeArguments(t *testing.T) {
	node, err := exec.LookPath("node")
	require.NoError(t, err, "Node.js is required to execute the shared Go/Make wrappers")
	config := sharedGoMakeTools(t)
	for _, toolName := range []string{"go", "make"} {
		t.Run(toolName, func(t *testing.T) {
			tool := config.Tools[toolName]
			require.NotNil(t, tool)
			require.Equal(t, MCPParamTypeArray, tool.Inputs["args"].Type)
			require.Equal(t, &MCPScriptParamItems{Type: MCPParamTypeString}, tool.Inputs["args"].Items)
			require.True(t, tool.Inputs["args"].Required)
			require.Empty(t, tool.Run)
			require.Empty(t, tool.Py)
			for _, test := range []struct {
				name string
				args []string
			}{
				{"multiple arguments", []string{"test", "./..."}},
				{"multiple targets", []string{"fmt", "build", "test-unit"}},
				{"paths with spaces", []string{"build", "-o", "path with spaces/output", "./other path"}},
				{"literal quotes and backslashes", []string{"test", `"quoted"`, "it's", `path\with\backslashes`, `trailing\`}},
				{"empty arguments", []string{"test", "", ""}},
				{"embedded whitespace", []string{"test", " \targument\nwith whitespace  "}},
				{"no arguments", []string{}},
				{"literal globs", []string{"test", "*.go", "[ab]*", "?"}},
				{"literal shell syntax", []string{"test", "$(touch injected)", "`touch injected`", "$HOME", "~", ";", ">", "injected", "#literal"}},
			} {
				t.Run(test.name, func(t *testing.T) {
					dir, script := prepareSharedGoMakeTool(t, tool)
					input, err := json.Marshal(map[string]any{"args": test.args})
					require.NoError(t, err)
					cmd := exec.Command(node, script)
					cmd.Dir = dir
					cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
					cmd.Stdin = bytes.NewReader(input)
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					require.NoError(t, cmd.Run(), stderr.String())
					var args []string
					record, err := os.ReadFile(filepath.Join(dir, "args.json"))
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(record, &args))
					require.Equal(t, test.args, args)
					var result struct {
						Stdout  string            `json:"stdout"`
						Stderr  string            `json:"stderr"`
						Outputs map[string]string `json:"outputs"`
					}
					require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
					require.Equal(t, "command stdout\n", result.Stdout)
					require.Equal(t, "command stderr\n", result.Stderr)
					require.Empty(t, result.Outputs)
					require.NoFileExists(t, filepath.Join(dir, "injected"))
				})
			}
			for _, input := range []string{`{"args":"test ./..."}`, `{"args":["test",1]}`, `{"args":null}`, `{}`, `{"args":[{}]}`} {
				t.Run("invalid "+input, func(t *testing.T) {
					dir, script := prepareSharedGoMakeTool(t, tool)
					cmd := exec.Command(node, script)
					cmd.Dir = dir
					cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
					cmd.Stdin = strings.NewReader(input)
					output, err := cmd.CombinedOutput()
					require.Error(t, err)
					require.Contains(t, string(output), "args must be an array of strings")
					require.NoFileExists(t, filepath.Join(dir, "args.json"))
				})
			}
			t.Run("exit status", func(t *testing.T) {
				dir, script := prepareSharedGoMakeTool(t, tool)
				cmd := exec.Command(node, script)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "TOOL_EXIT_CODE=7")
				cmd.Stdin = strings.NewReader(`{"args":["test","./..."]}`)
				output, err := cmd.CombinedOutput()
				require.Error(t, err)
				require.Contains(t, string(output), "exit code: 7")
				require.Contains(t, string(output), "command stdout")
				require.Contains(t, string(output), "command stderr")
			})
			t.Run("missing executable", func(t *testing.T) {
				dir, script := prepareSharedGoMakeTool(t, tool)
				require.NoError(t, os.Remove(filepath.Join(dir, tool.Name)))
				cmd := exec.Command(node, script)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "PATH="+dir)
				cmd.Stdin = strings.NewReader(`{"args":[]}`)
				output, err := cmd.CombinedOutput()
				require.Error(t, err)
				require.Contains(t, string(output), "ENOENT")
			})
		})
	}
}

func prepareSharedGoMakeTool(t *testing.T, tool *MCPScriptToolConfig) (string, string) {
	t.Helper()
	dir := t.TempDir()
	stub := `#!/usr/bin/env node
require("node:fs").writeFileSync("args.json", JSON.stringify(process.argv.slice(2)));
console.log("command stdout");
console.error("command stderr");
process.exit(Number(process.env.TOOL_EXIT_CODE || 0));
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, tool.Name), []byte(stub), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "match.go"), nil, 0600))
	for _, helper := range []string{"mcp-scripts-runner.cjs", "mcp_handler_process.cjs", "error_helpers.cjs"} {
		content, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "js", helper))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, helper), content, 0600))
	}
	script := filepath.Join(dir, "wrapper.cjs")
	require.NoError(t, os.WriteFile(script, []byte(GenerateMCPScriptJavaScriptToolScript(tool)), 0600))
	return dir, script
}

func TestSharedGoMakeToolConfig(t *testing.T) {
	var config MCPScriptsConfigJSON
	require.NoError(t, json.Unmarshal([]byte(GenerateMCPScriptsToolsConfig(sharedGoMakeTools(t))), &config))
	require.Len(t, config.Tools, 2)
	for _, tool := range config.Tools {
		require.Equal(t, tool.Name+".cjs", tool.Handler)
		require.Equal(t, []any{"args"}, tool.InputSchema["required"])
		properties, ok := tool.InputSchema["properties"].(map[string]any)
		require.True(t, ok)
		args, ok := properties["args"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "array", args["type"])
		require.Equal(t, map[string]any{"type": "string"}, args["items"])
		t.Run(tool.Name+" schema validation", func(t *testing.T) {
			node, err := exec.LookPath("node")
			require.NoError(t, err)
			for _, test := range []struct {
				name      string
				value     any
				errorPath string
			}{
				{"empty array", map[string]any{"args": []any{}}, ""},
				{"strings", map[string]any{"args": []any{"test", "./...", "", "path with spaces", "$(literal)"}}, ""},
				{"number", map[string]any{"args": []any{"test", 1}}, "args[1]"},
				{"object", map[string]any{"args": []any{map[string]any{}}}, "args[0]"},
				{"boolean", map[string]any{"args": []any{true}}, "args[0]"},
				{"null item", map[string]any{"args": []any{nil}}, "args[0]"},
				{"nested array", map[string]any{"args": []any{[]any{"test"}}}, "args[0]"},
				{"string", map[string]any{"args": "test ./..."}, "args"},
				{"null", map[string]any{"args": nil}, "args"},
				{"missing", map[string]any{}, "args"},
			} {
				t.Run(test.name, func(t *testing.T) {
					input, err := json.Marshal(map[string]any{"schema": tool.InputSchema, "value": test.value})
					require.NoError(t, err)
					cmd := exec.Command(node, "-e", `
const { validateValueAgainstSchema } = require("../../actions/setup/js/validation.cjs");
const { schema, value } = JSON.parse(require("node:fs").readFileSync(0, "utf8"));
process.stdout.write(JSON.stringify(validateValueAgainstSchema(value, schema)));
`)
					cmd.Stdin = bytes.NewReader(input)
					output, err := cmd.CombinedOutput()
					require.NoError(t, err, string(output))
					if test.errorPath == "" {
						require.Equal(t, "null", string(output))
					} else {
						var validationError struct {
							Path string `json:"path"`
						}
						require.NoError(t, json.Unmarshal(output, &validationError))
						require.Equal(t, test.errorPath, validationError.Path)
					}
				})
			}
		})
	}
}
