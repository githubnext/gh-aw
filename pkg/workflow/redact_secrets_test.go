package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCollectSecretsPattern(t *testing.T) {
	tests := []struct {
		name       string
		engineID   string
		wantMatch  []string
		wantNoMatch []string
	}{
		{
			name:     "Claude engine patterns",
			engineID: "claude",
			wantMatch: []string{
				"sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuv",
			},
			wantNoMatch: []string{
				"sk-ant-short", // Too short
				"not-a-secret",
			},
		},
		{
			name:     "Copilot engine patterns",
			engineID: "copilot",
			wantMatch: []string{
				"ghp_" + strings.Repeat("a", 36),                      // Classic PAT
				"github_pat_" + strings.Repeat("a", 82),               // Fine-grained PAT
			},
			wantNoMatch: []string{
				"ghp_short", // Too short
				"not-a-token",
			},
		},
		{
			name:     "Codex engine patterns",
			engineID: "codex",
			wantMatch: []string{
				"sk-" + strings.Repeat("a", 32),          // OpenAI key
				"sk-proj-" + strings.Repeat("a", 20),     // Project key
			},
			wantNoMatch: []string{
				"sk-short", // Too short
				"not-a-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock workflow data
			workflowData := &WorkflowData{}

			// Get the appropriate engine
			var engine CodingAgentEngine
			switch tt.engineID {
			case "claude":
				engine = NewClaudeEngine()
			case "copilot":
				engine = NewCopilotEngine()
			case "codex":
				engine = NewCodexEngine()
			default:
				t.Fatalf("Unknown engine: %s", tt.engineID)
			}

			pattern := CollectSecretsPattern(workflowData, engine)
			if pattern == "" {
				t.Fatal("Expected non-empty pattern")
			}

			// Validate the pattern is a valid regex
			err := validateSecretsPattern(pattern)
			if err != nil {
				t.Fatalf("Invalid regex pattern: %v", err)
			}

			// Test pattern matches expected secrets
			for _, secret := range tt.wantMatch {
				matched := matchesPattern(secret, pattern)
				if !matched {
					t.Errorf("Pattern should match secret (truncated for safety)")
					t.Logf("Pattern: %s", pattern)
				}
			}

			// Test pattern doesn't match non-secrets
			for _, nonSecret := range tt.wantNoMatch {
				matched := matchesPattern(nonSecret, pattern)
				if matched {
					t.Errorf("Pattern should not match non-secret: %s", nonSecret)
					t.Logf("Pattern: %s", pattern)
				}
			}
		})
	}
}

// matchesPattern tests if a string matches the given regex pattern
func matchesPattern(s, pattern string) bool {
	// Compile the pattern and test if it matches
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func TestSecretRedactionStepGeneration(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "secret-redaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow file
	testWorkflow := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Workflow

Test workflow for secret redaction.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testWorkflow), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify the redaction step is present (copilot engine has declared output files)
	if !strings.Contains(lockStr, "Redact secrets from files in /tmp") {
		t.Error("Expected redaction step in generated workflow")
	}

	// Verify the environment variable is set
	if !strings.Contains(lockStr, "GITHUB_AW_SECRETS_PATTERN") {
		t.Error("Expected GITHUB_AW_SECRETS_PATTERN environment variable")
	}

	// Verify the redaction step uses actions/github-script
	if !strings.Contains(lockStr, "uses: actions/github-script@v8") {
		t.Error("Expected redaction step to use actions/github-script@v8")
	}

	// Verify the redaction step runs with if: always()
	redactionStepIdx := strings.Index(lockStr, "Redact secrets from files in /tmp")
	if redactionStepIdx == -1 {
		t.Fatal("Redaction step not found")
	}

	// Check that if: always() appears near the redaction step
	redactionSection := lockStr[redactionStepIdx:min(redactionStepIdx+500, len(lockStr))]
	if !strings.Contains(redactionSection, "if: always()") {
		t.Error("Expected redaction step to have 'if: always()' condition")
	}
}

func TestSecretRedactionWithClaudeEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude-redaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testWorkflow := `---
on: push
permissions:
  contents: read
engine: claude
---

# Test Claude Workflow

Test workflow for Claude engine secret redaction.
`

	testFile := filepath.Join(tmpDir, "test-claude.md")
	if err := os.WriteFile(testFile, []byte(testWorkflow), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Claude engine doesn't declare output files by default, so redaction step should not be present
	// unless we modify it to declare output files
	// For now, this test documents the current behavior
	if strings.Contains(lockStr, "Redact secrets from files in /tmp") {
		// If this starts appearing, it means Claude engine now declares output files
		t.Log("Claude engine now includes secret redaction step (output files declared)")
	}
}

func TestValidateSecretsPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "empty pattern",
			pattern: "",
			wantErr: false,
		},
		{
			name:    "valid simple pattern",
			pattern: "sk-[a-zA-Z0-9]+",
			wantErr: false,
		},
		{
			name:    "valid alternation pattern",
			pattern: "ghp_[a-zA-Z0-9]{36,}|github_pat_[a-zA-Z0-9_]{82,}",
			wantErr: false,
		},
		{
			name:    "invalid pattern - unclosed bracket",
			pattern: "sk-[a-zA-Z0-9",
			wantErr: true,
		},
		{
			name:    "invalid pattern - unclosed parenthesis",
			pattern: "(sk-[a-zA-Z0-9]+",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretsPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSecretsPattern() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
