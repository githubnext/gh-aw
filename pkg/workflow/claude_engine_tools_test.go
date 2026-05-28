//go:build !integration

package workflow

import (
	"sort"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestClaudeEngineComputeAllowedTools(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("base tool cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeBaseToolCases())
	})
	t.Run("cache memory cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeCacheMemoryToolCases())
	})
	t.Run("mcp tool cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeMCPToolCases())
	})
	t.Run("bash wildcard cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeBashWildcardCases())
	})
	t.Run("neutral tool cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeNeutralToolCases())
	})
	t.Run("wildcard normalization cases", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeWildcardNormalizationCases())
	})
}

type claudeAllowedToolsTestCase struct {
	name        string
	tools       map[string]any
	safeOutputs *SafeOutputsConfig
	expected    string
}

func testClaudeAllowedToolsCases(t *testing.T, engine *ClaudeEngine, tests []claudeAllowedToolsTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClaudeAllowedTools(t, engine, tt.tools, tt.safeOutputs, tt.expected)
		})
	}
}

func testClaudeAllowedTools(
	t *testing.T,
	engine *ClaudeEngine,
	tools map[string]any,
	safeOutputs *SafeOutputsConfig,
	expected string,
) {
	t.Helper()

	compiler := NewCompiler()
	cacheMemoryConfig, _ := compiler.extractCacheMemoryConfigFromMap(tools)
	result := engine.computeAllowedClaudeToolsString(tools, safeOutputs, cacheMemoryConfig, nil, nil)

	expectedTools := testClaudeToolSet(expected)
	actualTools := testClaudeToolSet(result)

	if len(expectedTools) != len(actualTools) {
		t.Errorf("Expected %d tools, got %d tools. Expected: '%s', Actual: '%s'",
			len(expectedTools), len(actualTools), expected, result)
		return
	}

	for expectedTool := range expectedTools {
		if !actualTools[expectedTool] {
			t.Errorf("Expected tool '%s' not found in result: '%s'", expectedTool, result)
		}
	}

	for actualTool := range actualTools {
		if !expectedTools[actualTool] {
			t.Errorf("Unexpected tool '%s' found in result: '%s'", actualTool, result)
		}
	}
}

func testClaudeToolSet(input string) map[string]bool {
	tools := make(map[string]bool)
	if input == "" {
		return tools
	}

	for tool := range strings.SplitSeq(input, ",") {
		tools[strings.TrimSpace(tool)] = true
	}

	return tools
}

func testClaudeBaseToolCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "bash with specific commands (neutral format)",
			tools:    map[string]any{"bash": []any{"echo", "ls"}},
			expected: "Bash(echo),Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "bash with nil value (all commands allowed)",
			tools:    map[string]any{"bash": nil},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "neutral web tools",
			tools:    map[string]any{"web-fetch": nil, "web-search": nil},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch",
		},
	}
}

func testClaudeCacheMemoryToolCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "cache-memory tool (provides file system access with path-specific cache tools)",
			tools:    map[string]any{"cache-memory": map[string]any{"key": "test-memory-key"}},
			expected: "Bash(cat /tmp/gh-aw/cache-memory/),Bash(cat > /tmp/gh-aw/cache-memory/),Bash(mkdir -p /tmp/gh-aw/cache-memory/),Bash(mv /tmp/gh-aw/cache-memory/),BashOutput,Edit(/tmp/gh-aw/cache-memory/*),ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit(/tmp/gh-aw/cache-memory/*),NotebookRead,Read,Read(/tmp/gh-aw/cache-memory/*),Task,TodoWrite,Write(/tmp/gh-aw/cache-memory/*)",
		},
		{
			name:     "cache-memory with boolean true",
			tools:    map[string]any{"cache-memory": true},
			expected: "Bash(cat /tmp/gh-aw/cache-memory/),Bash(cat > /tmp/gh-aw/cache-memory/),Bash(mkdir -p /tmp/gh-aw/cache-memory/),Bash(mv /tmp/gh-aw/cache-memory/),BashOutput,Edit(/tmp/gh-aw/cache-memory/*),ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit(/tmp/gh-aw/cache-memory/*),NotebookRead,Read,Read(/tmp/gh-aw/cache-memory/*),Task,TodoWrite,Write(/tmp/gh-aw/cache-memory/*)",
		},
		{
			name:     "cache-memory with nil value (no value specified)",
			tools:    map[string]any{"cache-memory": nil},
			expected: "Bash(cat /tmp/gh-aw/cache-memory/),Bash(cat > /tmp/gh-aw/cache-memory/),Bash(mkdir -p /tmp/gh-aw/cache-memory/),Bash(mv /tmp/gh-aw/cache-memory/),BashOutput,Edit(/tmp/gh-aw/cache-memory/*),ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit(/tmp/gh-aw/cache-memory/*),NotebookRead,Read,Read(/tmp/gh-aw/cache-memory/*),Task,TodoWrite,Write(/tmp/gh-aw/cache-memory/*)",
		},
		{
			name: "cache-memory with github tools",
			tools: map[string]any{
				"cache-memory": true,
				"github":       map[string]any{"allowed": []any{"get_repository"}},
			},
			expected: "Bash(cat /tmp/gh-aw/cache-memory/),Bash(cat > /tmp/gh-aw/cache-memory/),Bash(mkdir -p /tmp/gh-aw/cache-memory/),Bash(mv /tmp/gh-aw/cache-memory/),BashOutput,Edit(/tmp/gh-aw/cache-memory/*),ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit(/tmp/gh-aw/cache-memory/*),NotebookRead,Read,Read(/tmp/gh-aw/cache-memory/*),Task,TodoWrite,Write(/tmp/gh-aw/cache-memory/*),mcp__github__get_repository",
		},
	}
}

func testClaudeMCPToolCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "mcp tools",
			tools:    map[string]any{"github": map[string]any{"allowed": []any{"list_issues", "create_issue"}}},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__github__create_issue,mcp__github__list_issues",
		},
		{
			name:     "github tools without explicit allowed list (should use defaults)",
			tools:    map[string]any{"github": map[string]any{}},
			expected: testClaudeDefaultGitHubTools(),
		},
		{
			name: "mixed neutral and mcp tools",
			tools: map[string]any{
				"web-fetch":  nil,
				"web-search": nil,
				"github":     map[string]any{"allowed": []any{"list_issues"}},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch,mcp__github__list_issues",
		},
		{
			name: "custom mcp servers with new format",
			tools: map[string]any{
				"custom_server": map[string]any{"type": "stdio", "command": "server", "allowed": []any{"tool1", "tool2"}},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__custom_server__tool1,mcp__custom_server__tool2",
		},
	}
}

func testClaudeMCPWildcardToolCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name: "mcp server with wildcard access",
			tools: map[string]any{
				"notion": map[string]any{"type": "stdio", "command": "notion-server", "allowed": []any{"*"}},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__notion",
		},
		{
			name: "mixed mcp servers - one with wildcard, one with specific tools",
			tools: map[string]any{
				"notion": map[string]any{"type": "stdio", "command": "notion-server", "allowed": []any{"*"}},
				"github": map[string]any{"allowed": []any{"list_issues", "create_issue"}},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__github__create_issue,mcp__github__list_issues,mcp__notion",
		},
	}
}

func testClaudeBashWildcardCases() []claudeAllowedToolsTestCase {
	return append(
		testClaudeCacheMemoryBashCases(),
		testClaudeExplicitBashCases()...,
	)
}

func testClaudeCacheMemoryBashCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "cache-memory with unrestricted bash (no extra cache bash commands injected)",
			tools:    map[string]any{"cache-memory": true, "bash": []any{"*"}},
			expected: "Bash,BashOutput,Edit(/tmp/gh-aw/cache-memory/*),ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit(/tmp/gh-aw/cache-memory/*),NotebookRead,Read,Read(/tmp/gh-aw/cache-memory/*),Task,TodoWrite,Write(/tmp/gh-aw/cache-memory/*)",
		},
		{
			name:     "bash with * wildcard (should ignore other bash tools)",
			tools:    map[string]any{"bash": []any{"*"}},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "bash with * wildcard mixed with other commands (should ignore other commands)",
			tools:    map[string]any{"bash": []any{"echo", "ls", "*", "cat"}},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with * wildcard and other tools",
			tools: map[string]any{
				"bash":      []any{"*"},
				"web-fetch": nil,
				"github":    map[string]any{"allowed": []any{"list_issues"}},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,mcp__github__list_issues",
		},
	}
}

func testClaudeExplicitBashCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "bash with :* wildcard (should ignore other bash tools)",
			tools:    map[string]any{"bash": []any{":*"}},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "bash with :* wildcard mixed with other commands (should ignore other commands)",
			tools:    map[string]any{"bash": []any{"echo", "ls", ":*", "cat"}},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with :* wildcard and other tools",
			tools: map[string]any{
				"bash":      []any{":*"},
				"web-fetch": nil,
				"github":    map[string]any{"allowed": []any{"list_issues"}},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,mcp__github__list_issues",
		},
		{
			name:     "bash with single command should include implicit tools",
			tools:    map[string]any{"bash": []any{"ls"}},
			expected: "Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
	}
}

func testClaudeNeutralToolCases() []claudeAllowedToolsTestCase {
	return append(
		testClaudeNeutralBasicCases(),
		testClaudeNeutralExtendedCases()...,
	)
}

func testClaudeNeutralBasicCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "explicit KillBash and BashOutput should not duplicate",
			tools:    map[string]any{"bash": []any{"echo"}},
			expected: "Bash(echo),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "no bash tools means no implicit tools",
			tools:    map[string]any{"web-fetch": nil, "web-search": nil},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch",
		},
		{
			name:     "neutral bash tool",
			tools:    map[string]any{"bash": []any{"echo", "ls"}},
			expected: "Bash(echo),Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "neutral web-fetch tool",
			tools:    map[string]any{"web-fetch": nil},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch",
		},
	}
}

func testClaudeNeutralExtendedCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name:     "neutral web-search tool",
			tools:    map[string]any{"web-search": nil},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebSearch",
		},
		{
			name:     "neutral edit tool",
			tools:    map[string]any{"edit": nil},
			expected: "Edit,ExitPlanMode,Glob,Grep,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,Write",
		},
		{
			name:     "neutral edit tool explicitly disabled",
			tools:    map[string]any{"edit": false},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "mixed neutral and MCP tools",
			tools: map[string]any{
				"web-fetch": nil,
				"bash":      []any{"git status"},
				"github":    map[string]any{"allowed": []any{"list_issues"}},
			},
			expected: "Bash(git status),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,mcp__github__list_issues",
		},
	}
}

func testClaudeNeutralAllToolCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{
			name: "all neutral tools together",
			tools: map[string]any{
				"bash":       []any{"echo"},
				"web-fetch":  nil,
				"web-search": nil,
				"edit":       nil,
			},
			expected: "Bash(echo),BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch,Write",
		},
		{
			name:     "neutral bash with nil value (all commands)",
			tools:    map[string]any{"bash": nil},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name:     "neutral playwright tool",
			tools:    map[string]any{"playwright": nil},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__playwright__browser_click,mcp__playwright__browser_close,mcp__playwright__browser_console_messages,mcp__playwright__browser_drag,mcp__playwright__browser_evaluate,mcp__playwright__browser_file_upload,mcp__playwright__browser_fill_form,mcp__playwright__browser_handle_dialog,mcp__playwright__browser_hover,mcp__playwright__browser_install,mcp__playwright__browser_navigate,mcp__playwright__browser_navigate_back,mcp__playwright__browser_network_requests,mcp__playwright__browser_press_key,mcp__playwright__browser_resize,mcp__playwright__browser_select_option,mcp__playwright__browser_snapshot,mcp__playwright__browser_tabs,mcp__playwright__browser_take_screenshot,mcp__playwright__browser_type,mcp__playwright__browser_wait_for",
		},
	}
}

func testClaudeWildcardNormalizationCases() []claudeAllowedToolsTestCase {
	return append(
		testClaudeNeutralAllToolCases(),
		[]claudeAllowedToolsTestCase{
			{
				name:     "bash tool with trailing space-star is normalized to canonical Bash(cmd)",
				tools:    map[string]any{"bash": []any{"jq *"}},
				expected: "Bash(jq),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
			},
			{
				name:     "community-attribution-style wildcard entries normalize to canonical forms",
				tools:    map[string]any{"bash": []any{"jq *", "sed *", "awk *"}},
				expected: "Bash(awk),Bash(jq),Bash(sed),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
			},
			{
				name:     "wildcard and non-wildcard forms of same command are both accepted",
				tools:    map[string]any{"bash": []any{"jq *", "jq"}},
				expected: "Bash(jq),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
			},
			{
				name:     "mcp wildcard cases remain covered",
				tools:    testClaudeMCPWildcardToolCases()[0].tools,
				expected: testClaudeMCPWildcardToolCases()[0].expected,
			},
			{
				name:     "mixed mcp wildcard cases remain covered",
				tools:    testClaudeMCPWildcardToolCases()[1].tools,
				expected: testClaudeMCPWildcardToolCases()[1].expected,
			},
		}...,
	)
}

func testClaudeDefaultGitHubTools() string {
	base := "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite"
	var githubTools []string
	for _, tool := range constants.DefaultGitHubTools {
		githubTools = append(githubTools, "mcp__github__"+tool)
	}
	sort.Strings(githubTools)
	return base + "," + strings.Join(githubTools, ",")
}

func TestClaudeEngineComputeAllowedToolsDeduplicatesNormalizedBashEntries(t *testing.T) {
	engine := NewClaudeEngine()

	tools := map[string]any{
		"bash": []any{"jq *", "jq"},
	}

	cacheMemoryConfig, err := NewCompiler().extractCacheMemoryConfigFromMap(tools)
	if err != nil {
		t.Fatalf("extract cache-memory config: %v", err)
	}

	result := engine.computeAllowedClaudeToolsString(tools, nil, cacheMemoryConfig, nil, nil)
	expected := "Bash(jq),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite"
	if result != expected {
		t.Fatalf("unexpected allowed tools\nwant: %s\ngot:  %s", expected, result)
	}
}

func TestClaudeEngineComputeAllowedToolsWithSafeOutputs(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("basic safe outputs", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeSafeOutputBasicCases())
	})
	t.Run("extended safe outputs", func(t *testing.T) {
		testClaudeAllowedToolsCases(t, engine, testClaudeSafeOutputExtendedCases())
	})
}

func testClaudeSafeOutputBasicCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{name: "SafeOutputs with no tools - should add Write permission", tools: map[string]any{}, safeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}}, expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,Write,mcp__safeoutputs"},
		{name: "SafeOutputs with general Write permission - should not add specific Write", tools: map[string]any{"edit": nil}, safeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}}, expected: "Edit,ExitPlanMode,Glob,Grep,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,Write,mcp__safeoutputs"},
		{name: "No SafeOutputs - should not add Write permission", tools: map[string]any{}, safeOutputs: nil, expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite"},
	}
}

func testClaudeSafeOutputExtendedCases() []claudeAllowedToolsTestCase {
	return []claudeAllowedToolsTestCase{
		{name: "SafeOutputs with multiple output types", tools: map[string]any{"bash": nil, "edit": nil}, safeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}, AddComments: &AddCommentsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}, CreatePullRequests: &CreatePullRequestsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}}, expected: "Bash,BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,Write,mcp__safeoutputs"},
		{name: "SafeOutputs with MCP tools", tools: map[string]any{"github": map[string]any{"allowed": []any{"create_issue", "create_pull_request"}}}, safeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}}, expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,Write,mcp__github__create_issue,mcp__github__create_pull_request,mcp__safeoutputs"},
		{name: "SafeOutputs with neutral tools and create-pull-request", tools: map[string]any{"bash": []any{"echo", "ls"}, "web-fetch": nil, "edit": nil}, safeOutputs: &SafeOutputsConfig{CreatePullRequests: &CreatePullRequestsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")}}}, expected: "Bash(echo),Bash(ls),BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,WebFetch,Write,mcp__safeoutputs"},
	}
}

func TestClaudeEngineComputeAllowedToolsWithSandboxAllowWrite(t *testing.T) {
	engine := NewClaudeEngine()
	cacheMemoryConfig, err := NewCompiler().extractCacheMemoryConfigFromMap(map[string]any{})
	if err != nil {
		t.Fatalf("extract cache-memory config: %v", err)
	}

	sandboxConfig := &SandboxConfig{
		Agent: &AgentSandboxConfig{
			Config: &SandboxRuntimeConfig{
				Filesystem: &SRTFilesystemConfig{
					AllowWrite: []string{"/tmp"},
				},
			},
		},
	}

	got := engine.computeAllowedClaudeToolsString(map[string]any{}, nil, cacheMemoryConfig, nil, sandboxConfig)
	want := "Edit(/tmp/*),ExitPlanMode,Glob,Grep,LS,MultiEdit(/tmp/*),NotebookRead,Read,Read(/tmp/*),Task,TodoWrite,Write(/tmp/*)"
	if got != want {
		t.Fatalf("unexpected allowed tools\nwant: %s\ngot:  %s", want, got)
	}
}

func TestClaudeEngineAddsTmpByDefault(t *testing.T) {
	engine := NewClaudeEngine()
	cacheMemoryConfig, err := NewCompiler().extractCacheMemoryConfigFromMap(map[string]any{})
	if err != nil {
		t.Fatalf("extract cache-memory config: %v", err)
	}

	sandboxConfig := &SandboxConfig{
		Agent: &AgentSandboxConfig{
			Type: SandboxTypeAWF,
		},
	}

	got := engine.computeAllowedClaudeToolsString(map[string]any{}, nil, cacheMemoryConfig, nil, sandboxConfig)
	want := "Edit(/tmp/*),ExitPlanMode,Glob,Grep,LS,MultiEdit(/tmp/*),NotebookRead,Read,Read(/tmp/*),Task,TodoWrite,Write(/tmp/*)"
	if got != want {
		t.Fatalf("unexpected allowed tools\nwant: %s\ngot:  %s", want, got)
	}
}

func TestGenerateAllowedToolsComment(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name            string
		allowedToolsStr string
		indent          string
		expected        string
	}{
		{
			name:            "empty allowed tools",
			allowedToolsStr: "",
			indent:          "  ",
			expected:        "",
		},
		{
			name:            "single tool",
			allowedToolsStr: "Bash",
			indent:          "  ",
			expected:        "  # Allowed tools (sorted):\n  # - Bash\n",
		},
		{
			name:            "multiple tools",
			allowedToolsStr: "Bash,Edit,Read",
			indent:          "    ",
			expected:        "    # Allowed tools (sorted):\n    # - Bash\n    # - Edit\n    # - Read\n",
		},
		{
			name:            "tools with special characters",
			allowedToolsStr: "Bash(echo),mcp__github__issue_read,Write",
			indent:          "      ",
			expected:        "      # Allowed tools (sorted):\n      # - Bash(echo)\n      # - mcp__github__issue_read\n      # - Write\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.generateAllowedToolsComment(tt.allowedToolsStr, tt.indent)
			if result != tt.expected {
				t.Errorf("Expected comment:\n%q\nBut got:\n%q", tt.expected, result)
			}
		})
	}
}
