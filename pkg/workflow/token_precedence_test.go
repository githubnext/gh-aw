package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitHubTokenPrecedence verifies that all generated workflows use the correct
// token precedence pattern: GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
func TestGitHubTokenPrecedence(t *testing.T) {
	tests := []struct {
		name              string
		engine            string
		workflow          string
		expectedTokenRefs []string // Token references that should follow GH_AW_GITHUB_TOKEN || GITHUB_TOKEN pattern
		allowedExceptions []string // Token references that are allowed to differ (e.g., COPILOT_CLI_TOKEN)
	}{
		{
			name:   "Claude engine with checkout and GitHub MCP",
			engine: "claude",
			workflow: `---
name: Test Token Precedence
on:
  push:
    branches: [main]
engine: claude
tools:
  github:
---
# Test workflow
Test token precedence in generated workflow.
`,
			expectedTokenRefs: []string{
				"token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}", // checkout
				"GITHUB_PERSONAL_ACCESS_TOKEN",                                     // GitHub MCP server
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Codex engine with GitHub MCP",
			engine: "codex",
			workflow: `---
name: Test Token Precedence Codex
on:
  push:
    branches: [main]
engine: codex
tools:
  github:
---
# Test workflow
Test token precedence in codex engine.
`,
			expectedTokenRefs: []string{
				"GITHUB_PERSONAL_ACCESS_TOKEN",
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Custom engine with GitHub MCP",
			engine: "custom",
			workflow: `---
name: Test Token Precedence Custom
on:
  push:
    branches: [main]
engine:
  id: custom
  steps:
    - name: Custom step
      run: echo "test"
tools:
  github:
---
# Test workflow
Test token precedence in custom engine.
`,
			expectedTokenRefs: []string{
				"GITHUB_PERSONAL_ACCESS_TOKEN",
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Copilot engine uses COPILOT_CLI_TOKEN but GitHub MCP uses GH_AW_GITHUB_TOKEN",
			engine: "copilot",
			workflow: `---
name: Test Token Precedence Copilot
on:
  push:
    branches: [main]
engine: copilot
tools:
  github:
---
# Test workflow
Test token precedence in copilot engine.
`,
			expectedTokenRefs: []string{
				"GITHUB_PERSONAL_ACCESS_TOKEN", // GitHub MCP server should use GH_AW_GITHUB_TOKEN
			},
			allowedExceptions: []string{
				"COPILOT_CLI_TOKEN", // Copilot CLI uses special token
			},
		},
		{
			name:   "Claude with agentic-workflows tool",
			engine: "claude",
			workflow: `---
name: Test Agentic Workflows Token
on:
  push:
    branches: [main]
engine: claude
tools:
  agentic-workflows:
---
# Test workflow
Test agentic workflows MCP server token.
`,
			expectedTokenRefs: []string{
				"GITHUB_TOKEN.*GH_AW_GITHUB_TOKEN.*GITHUB_TOKEN", // Agentic workflows MCP server
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Codex with agentic-workflows tool",
			engine: "codex",
			workflow: `---
name: Test Agentic Workflows Token Codex
on:
  push:
    branches: [main]
engine: codex
tools:
  agentic-workflows:
---
# Test workflow
Test agentic workflows MCP server token in codex.
`,
			expectedTokenRefs: []string{
				"GITHUB_TOKEN.*GH_AW_GITHUB_TOKEN.*GITHUB_TOKEN", // Agentic workflows MCP server in TOML
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Safe outputs without custom token",
			engine: "claude",
			workflow: `---
name: Test Safe Outputs Token
on:
  push:
    branches: [main]
engine: claude
safe-outputs:
  create-issue:
---
# Test workflow
Test safe outputs default token.
`,
			expectedTokenRefs: []string{
				"github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}", // Safe output job
			},
			allowedExceptions: []string{},
		},
		{
			name:   "Push to pull request branch",
			engine: "claude",
			workflow: `---
name: Test Push to PR Branch Token
on:
  pull_request:
    types: [opened]
engine: claude
safe-outputs:
  push-to-pull-request-branch:
---
# Test workflow
Test push to PR branch token.
`,
			expectedTokenRefs: []string{
				"token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}", // Checkout in push-to-PR job
			},
			allowedExceptions: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "token-precedence-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Write workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read generated lock file
			lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatal(err)
			}

			yamlContent := string(content)

			// Verify expected token references are present
			foundGHAWToken := strings.Contains(yamlContent, "GH_AW_GITHUB_TOKEN")
			if !foundGHAWToken {
				t.Errorf("Expected workflow to contain GH_AW_GITHUB_TOKEN reference, but it was not found")
				t.Logf("Generated workflow:\n%s", yamlContent)
			}

			// Check that all secret references follow the pattern (except allowed exceptions)
			lines := strings.Split(yamlContent, "\n")
			for i, line := range lines {
				// Skip comments
				if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue
				}

				// Skip SECRET_GITHUB_TOKEN which is used for redacting secrets in logs
				if strings.Contains(line, "SECRET_GITHUB_TOKEN") {
					continue
				}

				// Check if line contains secrets.GITHUB_TOKEN without GH_AW_GITHUB_TOKEN fallback
				if strings.Contains(line, "secrets.GITHUB_TOKEN") && !strings.Contains(line, "GH_AW_GITHUB_TOKEN") {
					// Check if this is an allowed exception
					isException := false
					for _, exception := range tt.allowedExceptions {
						if strings.Contains(line, exception) {
							isException = true
							break
						}
					}

					if !isException {
						t.Errorf("Line %d contains secrets.GITHUB_TOKEN without GH_AW_GITHUB_TOKEN fallback: %s", i+1, strings.TrimSpace(line))
					}
				}

				// Check for env var references that should use the pattern
				if (strings.Contains(line, "GITHUB_PERSONAL_ACCESS_TOKEN") || 
					(strings.Contains(line, "GITHUB_TOKEN") && !strings.Contains(line, "SECRET_GITHUB_TOKEN"))) &&
					strings.Contains(line, "${{") {
					// This should include GH_AW_GITHUB_TOKEN
					if !strings.Contains(line, "GH_AW_GITHUB_TOKEN") {
						// Check if this is an allowed exception
						isException := false
						for _, exception := range tt.allowedExceptions {
							if strings.Contains(line, exception) {
								isException = true
								break
							}
						}

						if !isException {
							t.Errorf("Line %d contains GitHub token reference without GH_AW_GITHUB_TOKEN: %s", i+1, strings.TrimSpace(line))
						}
					}
				}
			}
		})
	}
}

// TestTokenPrecedenceDocumentation verifies that the security documentation
// correctly describes the token precedence
func TestTokenPrecedenceDocumentation(t *testing.T) {
	// Read the security.md file
	securityDoc := filepath.Join("..", "..", "docs", "src", "content", "docs", "guides", "security.md")
	content, err := os.ReadFile(securityDoc)
	if err != nil {
		t.Skipf("Could not read security.md: %v", err)
		return
	}

	docContent := string(content)

	// Verify the documentation mentions GH_AW_GITHUB_TOKEN as primary
	if !strings.Contains(docContent, "GH_AW_GITHUB_TOKEN") {
		t.Error("security.md should document GH_AW_GITHUB_TOKEN")
	}

	// Verify it mentions the precedence order
	if !strings.Contains(docContent, "Primary override token") {
		t.Error("security.md should document token precedence with 'Primary override token'")
	}

	// Verify example usage
	if !strings.Contains(docContent, "GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN") {
		t.Error("security.md should show example of GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN pattern")
	}
}
