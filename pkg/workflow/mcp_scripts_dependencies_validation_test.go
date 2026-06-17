//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestValidateMCPScriptDependencies(t *testing.T) {
	compiler := NewCompiler()

	t.Run("valid dependencies", func(t *testing.T) {
		err := compiler.validateMCPScriptDependencies(&WorkflowData{
			MCPScripts: &MCPScriptsConfig{
				Tools: map[string]*MCPScriptToolConfig{
					"js-tool": {
						Script:       "return { ok: true }",
						Dependencies: []string{"lodash@4.17.21", "@scope/pkg@1.0.0"},
					},
					"py-tool": {
						Py:           "print('ok')",
						Dependencies: []string{"requests==2.32.3", "urllib3==2.2.1"},
					},
					"go-tool": {
						Go:           `fmt.Println("ok")`,
						Dependencies: []string{"github.com/google/uuid@v1.6.0"},
					},
					"sh-tool": {
						Run:          "echo ok",
						Dependencies: []string{"jq=1.6-2.1", "curl=8.5.0"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("invalid dependency name", func(t *testing.T) {
		err := compiler.validateMCPScriptDependencies(&WorkflowData{
			MCPScripts: &MCPScriptsConfig{
				Tools: map[string]*MCPScriptToolConfig{
					"fetch-url": {
						Py:           "print('ok')",
						Dependencies: []string{"re quests"},
					},
				},
			},
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `invalid dependency name "re quests" for tool "fetch-url"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("floating dependency rejected", func(t *testing.T) {
		err := compiler.validateMCPScriptDependencies(&WorkflowData{
			MCPScripts: &MCPScriptsConfig{
				Tools: map[string]*MCPScriptToolConfig{
					"fetch-url": {
						Py:           "print('ok')",
						Dependencies: []string{"requests"},
					},
				},
			},
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `dependency "requests" for tool "fetch-url" is not pinned to a release tag`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
