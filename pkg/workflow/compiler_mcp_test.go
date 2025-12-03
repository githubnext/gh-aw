package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)
func TestGenerateCustomMCPCodexWorkflowConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name       string
		toolConfig map[string]any
		expected   []string // expected strings in output
		wantErr    bool
	}{
		{
			name: "valid stdio mcp server",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "custom-mcp-server",
				"args":    []any{"--option", "value"},
				"env": map[string]any{
					"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
				},
			},
			expected: []string{
				"[mcp_servers.custom_server]",
				"command = \"custom-mcp-server\"",
				"--option",
				"\"CUSTOM_TOKEN\" = \"${CUSTOM_TOKEN}\"",
			},
			wantErr: false,
		},
		{
			name: "server with http type should be rendered for codex",
			toolConfig: map[string]any{
				"type": "http",
				"url":  "https://example.com/api",
			},
			expected: []string{
				"[mcp_servers.custom_server]",
				"url = \"https://example.com/api\"",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := engine.renderCodexMCPConfig(&yaml, "custom_server", tt.toolConfig)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateCustomMCPCodexWorkflowConfigForTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := yaml.String()
				for _, expected := range tt.expected {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
					}
				}
			}
		})
	}
}

func TestGenerateCustomMCPClaudeWorkflowConfig(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name       string
		toolConfig map[string]any
		isLast     bool
		expected   []string // expected strings in output
		wantErr    bool
	}{
		{
			name: "valid stdio mcp server",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "custom-mcp-server",
				"args":    []any{"--option", "value"},
				"env": map[string]any{
					"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
				},
			},
			isLast: true,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"custom-mcp-server\"",
				"\"--option\"",
				"\"CUSTOM_TOKEN\": \"${CUSTOM_TOKEN}\"",
				"              }",
			},
			wantErr: false,
		},
		{
			name: "not last server",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "valid-server",
			},
			isLast: false,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"valid-server\"",
				"              },", // should have comma since not last
			},
			wantErr: false,
		},
		{
			name: "mcp config with direct fields",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "trello_mcp"},
			},
			isLast: true,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"python\"",
				"\"-m\"",
				"\"trello_mcp\"",
				"              }",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := engine.renderClaudeMCPConfig(&yaml, "custom_server", tt.toolConfig, tt.isLast)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateCustomMCPCodexWorkflowConfigForTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := yaml.String()
				for _, expected := range tt.expected {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
					}
				}
			}
		})
	}
}

func TestMergeAllowedListsFromMultipleIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "multiple-includes-test")

	// Create first include file with Bash tools (new format)
	include1Content := `---
on: push
tools:
  bash: ["ls", "cat", "echo"]
---

# Include 1
First include file with bash tools.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with Bash tools (new format)
	include2Content := `---
on: push
tools:
  bash: ["grep", "find", "ls"] # ls is duplicate
---

# Include 2
Second include file with bash tools.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both files (new format)
	mainContent := fmt.Sprintf(`---
on: push
engine: claude
strict: false
tools:
  bash: ["pwd"] # Additional command in main file
---

# Test Workflow for Multiple Includes

@include %s

Some content here.

@include %s

More content.
`, filepath.Base(include1File), filepath.Base(include2File))

	// Test now with simplified structure - no includes, just main file
	// Create a simple workflow file with claude.Bash tools (no includes) (new format)
	simpleContent := `---
on: push
engine: claude
strict: false
tools:
  bash: ["pwd", "ls", "cat"]
---

# Simple Test Workflow

This is a simple test workflow with Bash tools.
`

	simpleFile := filepath.Join(tmpDir, "simple-workflow.md")
	if err := os.WriteFile(simpleFile, []byte(simpleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the simple workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(simpleFile); err != nil {
		t.Fatalf("Unexpected error compiling simple workflow: %v", err)
	}

	// Read the generated lock file for simple workflow
	simpleLockFile := strings.TrimSuffix(simpleFile, ".md") + ".lock.yml"
	simpleContent2, err := os.ReadFile(simpleLockFile)
	if err != nil {
		t.Fatalf("Failed to read simple lock file: %v", err)
	}

	simpleLockContent := string(simpleContent2)
	// t.Logf("Simple workflow lock file content: %s", simpleLockContent)

	// Check if simple case works first
	expectedSimpleCommands := []string{"pwd", "ls", "cat"}
	for _, cmd := range expectedSimpleCommands {
		expectedTool := fmt.Sprintf("Bash(%s)", cmd)
		if !strings.Contains(simpleLockContent, expectedTool) {
			t.Errorf("Expected simple lock file to contain '%s' but it didn't.", expectedTool)
		}
	}

	// Now proceed with the original test
	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that all bash commands from all includes are present in allowed_tools
	expectedCommands := []string{"pwd", "ls", "cat", "echo", "grep", "find"}

	// The allowed_tools should contain Bash(command) for each command
	for _, cmd := range expectedCommands {
		expectedTool := fmt.Sprintf("Bash(%s)", cmd)
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected lock file to contain '%s' but it didn't.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that 'ls' appears only once in the allowed-tools line (no duplicates in functionality)
	// We need to check specifically in the --allowed-tools line in CLI args, not in comments
	allowedToolsLinePattern := `--allowed-tools ([^\n]+)`
	re := regexp.MustCompile(allowedToolsLinePattern)
	matches := re.FindStringSubmatch(lockContent)
	if len(matches) < 2 {
		t.Errorf("Could not find --allowed-tools line in lock file")
	} else {
		allowedToolsValue := matches[1]
		bashLsCount := strings.Count(allowedToolsValue, "Bash(ls)")
		if bashLsCount != 1 {
			t.Errorf("Expected 'Bash(ls)' to appear exactly once in allowed-tools value, but found %d occurrences in: %s", bashLsCount, allowedToolsValue)
		}
	}
}

func TestMergeCustomMCPFromMultipleIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "custom-mcp-includes-test")

	// Create first include file with custom MCP server
	include1Content := `---
mcp-servers:
  notionApi:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["create_page", "search_pages"]
---

# Include 1
First include file with custom MCP server.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with different custom MCP server
	include2Content := `---
mcp-servers:
  trelloApi:
    command: "python"
    args: ["-m", "trello_mcp"]
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["create_card", "list_boards"]
---

# Include 2
Second include file with different custom MCP server.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create third include file with overlapping custom MCP server (same name, compatible config)
	include3Content := `---
mcp-servers:
  notionApi:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["list_databases", "query_database"]  # Different allowed tools - should be merged
  customTool:
    command: "custom-tool"
    allowed: ["tool1", "tool2"]
---

# Include 3
Third include file with compatible MCP server configuration.
`
	include3File := filepath.Join(tmpDir, "include3.md")
	if err := os.WriteFile(include3File, []byte(include3Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes all files and has its own custom MCP
	mainContent := fmt.Sprintf(`---
on: push
mcp-servers:
  mainCustomApi:
    command: "main-custom-server"
    allowed: ["main_tool1", "main_tool2"]
tools:
  github:
    allowed: ["list_issues", "create_issue"]
  edit:
---

# Test Workflow for Custom MCP Merging

{{#import %s}}

Some content here.

{{#import %s}}

More content.

{{#import %s}}

Final content.
`, filepath.Base(include1File), filepath.Base(include2File), filepath.Base(include3File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that all custom MCP tools from all includes are present in allowed_tools
	expectedCustomMCPTools := []string{
		// From include3 notionApi (last one wins, overrides include1)
		"notionApi(list_databases)",
		"notionApi(query_database)",
		// From include2 trelloApi
		"trelloApi(create_card)",
		"trelloApi(list_boards)",
		// From include3 customTool
		"customTool(tool1)",
		"customTool(tool2)",
		// From main file
		"mainCustomApi(main_tool1)",
		"mainCustomApi(main_tool2)",
		// Standard github MCP tools
		"github(list_issues)",
		"github(create_issue)",
	}

	// Check that all expected custom MCP tools are present
	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that the notionApi tools from both include1 and include3 are present
	// This shows that MCP servers with the same name get their 'allowed' arrays merged
	expectedAllTools := []string{
		"notionApi(create_page)",    // from include1
		"notionApi(search_pages)",   // from include1
		"notionApi(list_databases)", // from include3
		"notionApi(query_database)", // from include3
	}
	for _, expectedTool := range expectedAllTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected merged tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that Claude tools from all includes are present
	expectedClaudeTools := []string{
		"Read", "Write", // from includes
		"LS", "Task", // always present
	}
	for _, expectedTool := range expectedClaudeTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected Claude tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that custom MCP configurations are properly generated in the setup
	// The configuration should use the last import for the same tool name (include3 for notionApi)
	// Check for notionApi configuration (should contain container reference from include3)
	if !strings.Contains(lockContent, `"notionApi"`) {
		t.Errorf("Expected notionApi configuration from includes not found in lock file")
	}
	// The env should be present
	if !strings.Contains(lockContent, `NOTION_TOKEN`) {
		t.Errorf("Expected notionApi env configuration not found in lock file")
	}

	// Check for trelloApi configuration (from include2)
	if !strings.Contains(lockContent, `"trelloApi"`) {
		t.Errorf("Expected trelloApi configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `TRELLO_TOKEN`) {
		t.Errorf("Expected trelloApi env configuration not found in lock file")
	}

	// Check for mainCustomApi configuration
	if !strings.Contains(lockContent, `"mainCustomApi"`) {
		t.Errorf("Expected mainCustomApi configuration not found in lock file")
	}
}

func TestCustomMCPOnlyInIncludes(t *testing.T) {
	// Test case where custom MCPs are only defined in includes, not in main file
	tmpDir := testutil.TempDir(t, "custom-mcp-includes-only-test")

	// Create include file with custom MCP server
	includeContent := `---
mcp-servers:
  customApi:
    command: "custom-server"
    args: ["--config", "/path/to/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data", "delete_data"]
---

# Include with Custom MCP
Include file with custom MCP server only.
`
	includeFile := filepath.Join(tmpDir, "include.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file with only standard tools
	mainContent := fmt.Sprintf(`---
on: push
tools:
  github:
    allowed: ["list_issues"]
  edit:
---

# Test Workflow with Custom MCP Only in Include

{{#import %s}}

Content using custom API from include.
`, filepath.Base(includeFile))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that custom MCP tools from include are present
	expectedCustomMCPTools := []string{
		"customApi(get_data)",
		"customApi(post_data)",
		"customApi(delete_data)",
	}

	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' from include not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that custom MCP configuration is properly generated
	if !strings.Contains(lockContent, `"customApi"`) {
		t.Errorf("Expected customApi MCP server configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"--config"`) {
		t.Errorf("Expected customApi args configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `API_KEY`) {
		t.Errorf("Expected customApi env configuration not found in lock file")
	}
}

func TestCustomMCPMergingConflictDetection(t *testing.T) {
	// Test that conflicting MCP configurations result in errors
	tmpDir := testutil.TempDir(t, "custom-mcp-conflict-test")

	// Create first include file with custom MCP server
	include1Content := `---
on: push
tools:
  apiServer:
    mcp:
      type: stdio
    command: "server-v1"
    args: ["--port", "8080"]
    env:
      API_KEY: "{{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data"]
---

# Include 1
First include file with apiServer MCP.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with CONFLICTING custom MCP server (same name, different command)
	include2Content := `---
on: push
tools:
  apiServer:
    mcp:
      type: stdio
    command: "server-v2"  # Different command - should cause conflict
    args: ["--port", "9090"]  # Different args - should cause conflict
    env:
      API_KEY: "{{ secrets.API_KEY }}"  # Same env - should be OK
    allowed: ["delete_data", "update_data"]  # Different allowed - should be merged
---

# Include 2
Second include file with conflicting apiServer MCP.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both conflicting files
	mainContent := fmt.Sprintf(`---
on: push
tools:
  github:
    allowed: ["list_issues"]
---

# Test Workflow with Conflicting MCPs

@include %s

@include %s

This should fail due to conflicting MCP configurations.
`, filepath.Base(include1File), filepath.Base(include2File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - this should produce an error due to conflicting configurations
	compiler := NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(mainFile)

	// We expect this to fail due to conflicting MCP configurations
	if err == nil {
		t.Errorf("Expected compilation to fail due to conflicting MCP configurations, but it succeeded")
	} else {
		// Check that the error message mentions the conflict
		errorStr := err.Error()
		if !strings.Contains(errorStr, "conflict") && !strings.Contains(errorStr, "apiServer") {
			t.Errorf("Expected error to mention MCP conflict for 'apiServer', but got: %v", err)
		}
	}
}

func TestCustomMCPMergingFromMultipleIncludes(t *testing.T) {
	// Test that tools from imports with the same MCP server name get merged
	tmpDir := testutil.TempDir(t, "custom-mcp-merge-test")

	// Create first include file with custom MCP server
	include1Content := `---
mcp-servers:
  apiServer:
    command: "shared-server"
    args: ["--config", "/shared/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data"]
---

# Include 1
First include file with apiServer MCP.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with same MCP server but different allowed list
	include2Content := `---
mcp-servers:
  apiServer:
    command: "shared-server"
    args: ["--config", "/shared/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["delete_data", "update_data"]  # Different allowed - should merge with include1
---

# Include 2
Second include file with apiServer MCP that merges with include1.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both files
	mainContent := fmt.Sprintf(`---
on: push
tools:
  github:
    allowed: ["list_issues"]
---

# Test Workflow with Merged Allowed Arrays

{{#import %s}}

{{#import %s}}

This should merge the allowed lists from both imports.
`, filepath.Base(include1File), filepath.Base(include2File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - this should succeed
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with compatible MCPs: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that tools from both imports are present (merged, not overridden)
	expectedMergedTools := []string{
		"apiServer(get_data)",
		"apiServer(post_data)",
		"apiServer(delete_data)",
		"apiServer(update_data)",
	}

	for _, expectedTool := range expectedMergedTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected merged MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that the MCP server configuration is present
	if !strings.Contains(lockContent, `"apiServer"`) {
		t.Errorf("Expected apiServer MCP configuration not found in lock file")
	}
}

func TestMCPImageField(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "mcp-container-test")

	tests := []struct {
		name           string
		frontmatter    string
		expectedInLock []string // Strings that should appear in the lock file
		notExpected    []string // Strings that should NOT appear in the lock file
		expectError    bool
		errorContains  string
	}{
		{
			name: "simple container field",
			frontmatter: `---
on: push
strict: false
mcp-servers:
  notionApi:
    container: mcp/notion
    allowed: ["create_page", "search"]
---`,
			expectedInLock: []string{
				`"notionApi"`,
				`"mcp/notion"`,
			},
			expectError: false,
		},
		{
			name: "container with environment variables",
			frontmatter: `---
on: push
strict: false
mcp-servers:
  notionApi:
    container: mcp/notion:v1.2.3
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
      API_URL: "https://api.notion.com"
    allowed: ["create_page"]
---`,
			expectedInLock: []string{
				`"mcp/notion:v1.2.3"`,
				`NOTION_TOKEN`,
				`API_URL`,
			},
			expectError: false,
		},
		{
			name: "multiple MCP servers with container fields",
			frontmatter: `---
on: push
strict: false
mcp-servers:
  notionApi:
    container: mcp/notion
    allowed: ["create_page"]
  trelloApi:
    container: mcp/trello:latest
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["list_boards"]
---`,
			expectedInLock: []string{
				`"notionApi"`,
				`"trelloApi"`,
				`"mcp/notion"`,
				`"mcp/trello:latest"`,
				`TRELLO_TOKEN`,
			},
			expectError: false,
		},
	}

	compiler := NewCompiler(false, "", "test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow for container field.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that expected strings are present
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockContent, expected) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", expected, lockContent)
				}
			}

			// Check that unexpected strings are NOT present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(lockContent, notExpected) {
					t.Errorf("Lock file should NOT contain '%s' but it did.\nContent:\n%s", notExpected, lockContent)
				}
			}
		})
	}
}

