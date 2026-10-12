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
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required to execute the shared Go/Make wrappers")
	}
	config := sharedGoMakeTools(t)
	for _, toolName := range []string{"go", "make"} {
		t.Run(toolName, func(t *testing.T) {
			tool := config.Tools[toolName]
			require.NotNil(t, tool)
			require.Equal(t, MCPParamTypeString, tool.Inputs["args"].Type)
			require.True(t, tool.Inputs["args"].Required)
			for _, test := range []struct {
				name  string
				input string
				args  []string
			}{
				{"multiple arguments", "test ./...", []string{"test", "./..."}},
				{"multiple targets", "fmt build test-unit", []string{"fmt", "build", "test-unit"}},
				{"quoted paths", `build -o "path with spaces/output" './other path'`, []string{"build", "-o", "path with spaces/output", "./other path"}},
				{"escaped spaces and quotes", `test ./path\ with\ spaces "a\"b" 'it'\''s'`, []string{"test", "./path with spaces", `a"b`, "it's"}},
				{"empty quoted arguments", `test "" ''`, []string{"test", "", ""}},
				{"whitespace", " \ttest\n./...  ", []string{"test", "./..."}},
				{"no arguments", "", []string{}},
				{"literal globs", "test *.go [ab]* ?", []string{"test", "*.go", "[ab]*", "?"}},
				{"literal shell syntax", "test '$(touch injected)' '`touch injected`' $HOME ~ ; > injected #literal", []string{"test", "$(touch injected)", "`touch injected`", "$HOME", "~", ";", ">", "injected", "#literal"}},
			} {
				t.Run(test.name, func(t *testing.T) {
					dir, script := prepareSharedGoMakeTool(t, tool)
					cmd := exec.Command("bash", script)
					cmd.Dir = dir
					cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "INPUT_ARGS="+test.input)
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					require.NoError(t, cmd.Run(), stderr.String())
					var args []string
					record, err := os.ReadFile(filepath.Join(dir, "args.json"))
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(record, &args))
					require.Equal(t, test.args, args)
					require.Contains(t, stdout.String(), "command stdout")
					require.Contains(t, stderr.String(), "command stderr")
					require.NoFileExists(t, filepath.Join(dir, "injected"))
				})
			}
			for _, input := range []string{`test "unterminated`, `test 'unterminated`, `test trailing\`} {
				t.Run("invalid "+input, func(t *testing.T) {
					dir, script := prepareSharedGoMakeTool(t, tool)
					cmd := exec.Command("bash", script)
					cmd.Dir = dir
					cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "INPUT_ARGS="+input)
					output, err := cmd.CombinedOutput()
					require.Error(t, err)
					require.Contains(t, string(output), "ValueError")
					require.NoFileExists(t, filepath.Join(dir, "args.json"))
				})
			}
			t.Run("exit status", func(t *testing.T) {
				dir, script := prepareSharedGoMakeTool(t, tool)
				cmd := exec.Command("bash", script)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "INPUT_ARGS=test ./...", "TOOL_EXIT_CODE=7")
				output, err := cmd.CombinedOutput()
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, 7, exitErr.ExitCode())
				require.Contains(t, string(output), "command stderr")
			})
		})
	}
}

func prepareSharedGoMakeTool(t *testing.T, tool *MCPScriptToolConfig) (string, string) {
	t.Helper()
	dir := t.TempDir()
	stub := `#!/usr/bin/env python3
import json
import os
import sys
with open("args.json", "w") as record:
    json.dump(sys.argv[1:], record)
print("command stdout")
print("command stderr", file=sys.stderr)
sys.exit(int(os.environ.get("TOOL_EXIT_CODE", "0")))
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, tool.Name), []byte(stub), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "match.go"), nil, 0600))
	script := filepath.Join(dir, "wrapper.sh")
	require.NoError(t, os.WriteFile(script, []byte(GenerateMCPScriptShellToolScript(tool)), 0600))
	return dir, script
}

func TestSharedGoMakeShellcheck(t *testing.T) {
	shellcheck, err := exec.LookPath("shellcheck")
	if err != nil {
		t.Skip("shellcheck is not installed")
	}
	config := sharedGoMakeTools(t)
	for _, toolName := range []string{"go", "make"} {
		t.Run(toolName, func(t *testing.T) {
			cmd := exec.Command(shellcheck, "--shell=bash", "-")
			cmd.Stdin = strings.NewReader(GenerateMCPScriptShellToolScript(config.Tools[toolName]))
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, string(output))
		})
	}
}
