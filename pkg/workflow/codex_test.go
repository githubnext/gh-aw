package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCodexAIConfiguration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "codex-ai-test")

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	tests := []struct {
		name          string
		frontmatter   string
		expectedAI    string
		expectCodex   bool
		expectWarning bool
	}{
		{
			name: "default copilot ai",
			frontmatter: `---
on: push
tools:
  github:
    allowed: [list_issues]
---`,
			expectedAI:    "copilot",
			expectCodex:   false,
			expectWarning: false,
		},
		{
			name: "explicit claude ai",
			frontmatter: `---
on: push
engine: claude
tools:
  github:
    allowed: [list_issues]
---`,
			expectedAI:    "claude",
			expectCodex:   false,
			expectWarning: false,
		},
		{
			name: "codex ai",
			frontmatter: `---
on: push
engine: codex
tools:
  github:
    allowed: [list_issues]
---`,
			expectedAI:    "codex",
			expectCodex:   true,
			expectWarning: true,
		},
		{
			name: "codex ai without tools",
			frontmatter: `---
on: push
engine: codex
---`,
			expectedAI:    "codex",
			expectCodex:   true,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
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

			if tt.expectCodex {
				// Check that Node.js setup is present for codex
				if !strings.Contains(lockContent, "Setup Node.js") {
					t.Errorf("Expected lock file to contain 'Setup Node.js' step for codex but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "actions/setup-node@395ad3262231945c25e8478fd5baf05154b1d79f") {
					t.Errorf("Expected lock file to contain Node.js setup action for codex but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that codex installation is present
				if !strings.Contains(lockContent, "Install Codex") {
					t.Errorf("Expected lock file to contain 'Install Codex' step but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "npm install -g --silent @openai/codex") {
					t.Errorf("Expected lock file to contain codex installation command but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that codex command is present
				if !strings.Contains(lockContent, "Run Codex") {
					t.Errorf("Expected lock file to contain 'Run Codex' step but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "codex") && !strings.Contains(lockContent, "exec") {
					t.Errorf("Expected lock file to contain 'codex exec' command but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "CODEX_API_KEY") {
					t.Errorf("Expected lock file to contain 'CODEX_API_KEY' for codex but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that CODEX_HOME is set
				if !strings.Contains(lockContent, "CODEX_HOME: /tmp/gh-aw/mcp-config") {
					t.Errorf("Expected lock file to contain 'CODEX_HOME: /tmp/gh-aw/mcp-config' environment variable but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that config.toml is generated (not mcp-servers.json)
				if !strings.Contains(lockContent, "cat > /tmp/gh-aw/mcp-config/config.toml") {
					t.Errorf("Expected lock file to contain config.toml generation for codex but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "[mcp_servers.github]") {
					t.Errorf("Expected lock file to contain '[mcp_servers.github]' section in config.toml but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that history configuration is present
				if !strings.Contains(lockContent, "[history]") {
					t.Errorf("Expected lock file to contain '[history]' section in config.toml but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "persistence = \"none\"") {
					t.Errorf("Expected lock file to contain 'persistence = \"none\"' in config.toml but it didn't.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain mcp-servers.json
				if strings.Contains(lockContent, "mcp-servers.json") {
					t.Errorf("Expected lock file to NOT contain 'mcp-servers.json' when using codex.\nContent:\n%s", lockContent)
				}
				// Check that prompt printing step is present (regardless of engine)
				if !strings.Contains(lockContent, "Print prompt") {
					t.Errorf("Expected lock file to contain 'Print prompt' step but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "cat \"$GH_AW_PROMPT\"") {
					t.Errorf("Expected lock file to contain prompt printing command but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "} >> \"$GITHUB_STEP_SUMMARY\"") {
					t.Errorf("Expected lock file to contain grouped redirect to GITHUB_STEP_SUMMARY but it didn't.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain Claude Code
				if strings.Contains(lockContent, "Execute Claude Code Action") {
					t.Errorf("Expected lock file to NOT contain 'Execute Claude Code Action' step when using codex.\nContent:\n%s", lockContent)
				}
			} else if tt.expectedAI == "claude" {
				// Check that Claude Code CLI is present
				if !strings.Contains(lockContent, "Execute Claude Code CLI") {
					t.Errorf("Expected lock file to contain 'Execute Claude Code CLI' step but it didn't.\nContent:\n%s", lockContent)
				}
				// Check for installation step (npm install)
				expectedClaudeInstall := fmt.Sprintf("npm install -g --silent @anthropic-ai/claude-code@%s", constants.DefaultClaudeCodeVersion)
				if !strings.Contains(lockContent, expectedClaudeInstall) {
					t.Errorf("Expected lock file to contain npm install command (%s) but it didn't.\nContent:\n%s", expectedClaudeInstall, lockContent)
				}
				// Check for direct claude command (not npx)
				if !strings.Contains(lockContent, "claude --print") {
					t.Errorf("Expected lock file to contain claude command but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that prompt printing step is present
				if !strings.Contains(lockContent, "Print prompt") {
					t.Errorf("Expected lock file to contain 'Print prompt' step but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "cat \"$GH_AW_PROMPT\"") {
					t.Errorf("Expected lock file to contain prompt printing command but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "} >> \"$GITHUB_STEP_SUMMARY\"") {
					t.Errorf("Expected lock file to contain grouped redirect to GITHUB_STEP_SUMMARY but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that mcp-servers.json is generated (not config.toml)
				if !strings.Contains(lockContent, "cat > /tmp/gh-aw/mcp-config/mcp-servers.json") {
					t.Errorf("Expected lock file to contain mcp-servers.json generation for claude but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "\"mcpServers\":") {
					t.Errorf("Expected lock file to contain '\"mcpServers\":' section in mcp-servers.json but it didn't.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain codex
				if strings.Contains(lockContent, "codex exec") {
					t.Errorf("Expected lock file to NOT contain 'codex exec' when using claude.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain config.toml
				if strings.Contains(lockContent, "config.toml") {
					t.Errorf("Expected lock file to NOT contain 'config.toml' when using claude.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain CODEX_HOME
				if strings.Contains(lockContent, "CODEX_HOME") {
					t.Errorf("Expected lock file to NOT contain 'CODEX_HOME' when using claude.\nContent:\n%s", lockContent)
				}
			} else if tt.expectedAI == "copilot" {
				// Check that Copilot CLI is present
				if !strings.Contains(lockContent, "Execute GitHub Copilot CLI") {
					t.Errorf("Expected lock file to contain 'Execute GitHub Copilot CLI' step but it didn't.\nContent:\n%s", lockContent)
				}
				// Check for official install.sh script usage
				if !strings.Contains(lockContent, "https://raw.githubusercontent.com/github/copilot-cli/main/install.sh") ||
					!strings.Contains(lockContent, "export VERSION=") {
					t.Errorf("Expected lock file to contain Copilot installer using official install.sh script but it didn't.\nContent:\n%s", lockContent)
				}
				// Ensure script is downloaded to file before execution (not piped)
				if strings.Contains(lockContent, "gh.io/copilot-install | sudo bash") {
					t.Errorf("Lock file should not pipe installer directly to bash.\nContent:\n%s", lockContent)
				}
				// Check that prompt printing step is present
				if !strings.Contains(lockContent, "Print prompt") {
					t.Errorf("Expected lock file to contain 'Print prompt' step but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "cat \"$GH_AW_PROMPT\"") {
					t.Errorf("Expected lock file to contain prompt printing command but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "} >> \"$GITHUB_STEP_SUMMARY\"") {
					t.Errorf("Expected lock file to contain grouped redirect to GITHUB_STEP_SUMMARY but it didn't.\nContent:\n%s", lockContent)
				}
				// Check that mcp-config.json is generated (Copilot format)
				if !strings.Contains(lockContent, "cat > /home/runner/.copilot/mcp-config.json") {
					t.Errorf("Expected lock file to contain mcp-config.json generation for copilot but it didn't.\nContent:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "\"mcpServers\":") {
					t.Errorf("Expected lock file to contain '\"mcpServers\":' section in mcp-config.json but it didn't.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain codex
				if strings.Contains(lockContent, "codex exec") {
					t.Errorf("Expected lock file to NOT contain 'codex exec' when using copilot.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain config.toml
				if strings.Contains(lockContent, "config.toml") {
					t.Errorf("Expected lock file to NOT contain 'config.toml' when using copilot.\nContent:\n%s", lockContent)
				}
				// Ensure it does NOT contain CODEX_HOME
				if strings.Contains(lockContent, "CODEX_HOME") {
					t.Errorf("Expected lock file to NOT contain 'CODEX_HOME' when using copilot.\nContent:\n%s", lockContent)
				}
			}
		})
	}
}

func TestCodexMCPConfigGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "codex-mcp-test")

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	tests := []struct {
		name                 string
		frontmatter          string
		expectedAI           string
		expectConfigToml     bool
		expectMcpServersJson bool
		expectCodexHome      bool
	}{
		{
			name: "codex with github tools generates config.toml",
			frontmatter: `---
on: push
engine: codex
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "codex",
			expectConfigToml:     true,
			expectMcpServersJson: false,
			expectCodexHome:      true,
		},
		{
			name: "claude with github tools generates mcp-servers.json",
			frontmatter: `---
on: push
engine: claude
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "claude",
			expectConfigToml:     false,
			expectMcpServersJson: true,
			expectCodexHome:      false,
		},
		{
			name: "codex with docker github tools generates config.toml",
			frontmatter: `---
on: push
engine: codex
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "codex",
			expectConfigToml:     true,
			expectMcpServersJson: false,
			expectCodexHome:      true,
		},
		{
			name: "claude with docker github tools generates mcp-servers.json",
			frontmatter: `---
on: push
engine: claude
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "claude",
			expectConfigToml:     false,
			expectMcpServersJson: true,
			expectCodexHome:      false,
		},
		{
			name: "codex with services github tools generates config.toml",
			frontmatter: `---
on: push
engine: codex
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "codex",
			expectConfigToml:     true,
			expectMcpServersJson: false,
			expectCodexHome:      true,
		},
		{
			name: "claude with services github tools generates mcp-servers.json",
			frontmatter: `---
on: push
engine: claude
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectedAI:           "claude",
			expectConfigToml:     false,
			expectMcpServersJson: true,
			expectCodexHome:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testContent := tt.frontmatter + `

# Test MCP Configuration

This is a test workflow for MCP configuration with different AI engines.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
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

			// Test config.toml generation
			if tt.expectConfigToml {
				if !strings.Contains(lockContent, "cat > /tmp/gh-aw/mcp-config/config.toml") {
					t.Errorf("Expected config.toml generation but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "[mcp_servers.github]") {
					t.Errorf("Expected [mcp_servers.github] section but didn't find it in:\n%s", lockContent)
				}

				if !strings.Contains(lockContent, "command = \"docker\"") {
					t.Errorf("Expected docker command in config.toml but didn't find it in:\n%s", lockContent)
				}
				// Should NOT have services section (services mode removed)
				if strings.Contains(lockContent, "services:") {
					t.Errorf("Expected NO services section in workflow but found it in:\n%s", lockContent)
				}
			} else {
				if strings.Contains(lockContent, "config.toml") {
					t.Errorf("Expected NO config.toml but found it in:\n%s", lockContent)
				}
			}

			// Test mcp-servers.json generation
			if tt.expectMcpServersJson {
				if !strings.Contains(lockContent, "cat > /tmp/gh-aw/mcp-config/mcp-servers.json") {
					t.Errorf("Expected mcp-servers.json generation but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "\"mcpServers\":") {
					t.Errorf("Expected mcpServers section but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, "\"github\":") {
					t.Errorf("Expected github section in JSON but didn't find it in:\n%s", lockContent)
				}

				if !strings.Contains(lockContent, "\"command\": \"docker\"") {
					t.Errorf("Expected docker command in mcp-servers.json but didn't find it in:\n%s", lockContent)
				}
				// Should NOT have services section (services mode removed)
				if strings.Contains(lockContent, "services:") {
					t.Errorf("Expected NO services section in workflow but found it in:\n%s", lockContent)
				}
			} else {
				if strings.Contains(lockContent, "mcp-servers.json") {
					t.Errorf("Expected NO mcp-servers.json but found it in:\n%s", lockContent)
				}
			}

			// Test CODEX_HOME setting
			if tt.expectCodexHome {
				if !strings.Contains(lockContent, "CODEX_HOME: /tmp/gh-aw/mcp-config") {
					t.Errorf("Expected CODEX_HOME environment variable but didn't find it in:\n%s", lockContent)
				}
			} else {
				if strings.Contains(lockContent, "CODEX_HOME") {
					t.Errorf("Expected NO CODEX_HOME but found it in:\n%s", lockContent)
				}
			}

			// Verify AI type
			if tt.expectedAI == "codex" {
				if !strings.Contains(lockContent, "codex") && !strings.Contains(lockContent, "exec") {
					t.Errorf("Expected codex exec command but didn't find it in:\n%s", lockContent)
				}
				if strings.Contains(lockContent, "npx @anthropic-ai/claude-code") {
					t.Errorf("Expected NO claude CLI but found it in:\n%s", lockContent)
				}
			} else {
				// Check for direct claude command (not npx)
				if !strings.Contains(lockContent, "claude --print") {
					t.Errorf("Expected claude command but didn't find it in:\n%s", lockContent)
				}
				// Check for npm install
				if !strings.Contains(lockContent, "npm install -g --silent @anthropic-ai/claude-code") {
					t.Errorf("Expected npm install command but didn't find it in:\n%s", lockContent)
				}
			}
		})
	}
}

func TestCodexConfigField(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")
	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	tests := []struct {
		name         string
		frontmatter  string
		expectConfig string
	}{
		{
			name: "codex with custom config field",
			frontmatter: `---
on: push
engine:
  id: codex
  config: |
    [custom_section]
    key1 = "value1"
    key2 = "value2"
    
    [another_section]
    enabled = true
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectConfig: `[custom_section]
key1 = "value1"
key2 = "value2"

[another_section]
enabled = true`,
		},
		{
			name: "codex without config field",
			frontmatter: `---
on: push
engine: codex
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectConfig: "",
		},
		{
			name: "codex with empty config field",
			frontmatter: `---
on: push
engine:
  id: codex
  config: ""
tools:
  github:
    allowed: [issue_read, create_issue]
---`,
			expectConfig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Config Field

This is a test workflow for testing the config field functionality.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
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

			// Test that config.toml is generated for codex
			if !strings.Contains(lockContent, "cat > /tmp/gh-aw/mcp-config/config.toml") {
				t.Errorf("Expected config.toml generation but didn't find it in:\n%s", lockContent)
			}

			if tt.expectConfig != "" {
				// Check that custom config section is present
				if !strings.Contains(lockContent, "# Custom configuration") {
					t.Errorf("Expected custom configuration comment but didn't find it in:\n%s", lockContent)
				}

				// Check that the actual config content is included
				configLines := strings.Split(tt.expectConfig, "\n")
				for _, line := range configLines {
					if strings.TrimSpace(line) != "" {
						expectedLine := "          " + line
						if !strings.Contains(lockContent, expectedLine) {
							t.Errorf("Expected config line '%s' but didn't find it in:\n%s", expectedLine, lockContent)
						}
					}
				}
			} else {
				// Check that no custom config section is present
				if strings.Contains(lockContent, "# Custom configuration") {
					t.Errorf("Expected NO custom configuration but found it in:\n%s", lockContent)
				}
			}
		})
	}
}
