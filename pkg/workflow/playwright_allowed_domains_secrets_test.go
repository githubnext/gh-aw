package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestPlaywrightAllowedDomainsSecretHandling tests that secrets in allowed_domains
// are properly extracted and replaced with environment variable references
func TestPlaywrightAllowedDomainsSecretHandling(t *testing.T) {
	tests := []struct {
		name                 string
		workflow             string
		expectEnvVar         string
		expectEnvVarValue    string
		expectMCPConfigValue string
		expectRedaction      bool
	}{
		{
			name: "Single secret in allowed_domains",
			workflow: `---
on: issues
engine: copilot
tools:
  playwright:
    allowed_domains:
      - "${{ secrets.TEST_DOMAIN }}"
---

# Test workflow

Test secret in allowed_domains.
`,
			expectEnvVar:         "PLAYWRIGHT_ALLOWED_DOMAIN_TEST_DOMAIN",
			expectEnvVarValue:    "${{ secrets.TEST_DOMAIN }}",
			expectMCPConfigValue: "${PLAYWRIGHT_ALLOWED_DOMAIN_TEST_DOMAIN}",
			expectRedaction:      true,
		},
		{
			name: "Multiple secrets in allowed_domains",
			workflow: `---
on: issues
engine: copilot
tools:
  playwright:
    allowed_domains:
      - "${{ secrets.API_KEY }}"
      - "example.com"
      - "${{ secrets.ANOTHER_SECRET }}"
---

# Test workflow

Test multiple secrets in allowed_domains.
`,
			expectEnvVar:         "PLAYWRIGHT_ALLOWED_DOMAIN_API_KEY",
			expectEnvVarValue:    "${{ secrets.API_KEY }}",
			expectMCPConfigValue: "${PLAYWRIGHT_ALLOWED_DOMAIN_API_KEY}",
			expectRedaction:      true,
		},
		{
			name: "No secrets in allowed_domains",
			workflow: `---
on: issues
engine: copilot
tools:
  playwright:
    allowed_domains:
      - "example.com"
      - "test.org"
---

# Test workflow

Test no secrets in allowed_domains.
`,
			expectEnvVar:         "PLAYWRIGHT_ALLOWED_DOMAIN_",
			expectEnvVarValue:    "",
			expectMCPConfigValue: "example.com",
			expectRedaction:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-playwright-secrets-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content to file
			if _, err := tmpFile.WriteString(tt.workflow); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Create compiler and compile workflow
			compiler := NewCompiler(false, "", "test")
			compiler.SetSkipValidation(true)

			// Parse the workflow file to get WorkflowData
			workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to parse workflow file: %v", err)
			}

			// Generate YAML
			yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check if environment variable is set in Setup MCPs step
			if tt.expectRedaction {
				if !strings.Contains(yamlContent, tt.expectEnvVar+": "+tt.expectEnvVarValue) {
					t.Errorf("Expected environment variable %s with value %s not found in Setup MCPs step", tt.expectEnvVar, tt.expectEnvVarValue)
				}
			} else {
				if strings.Contains(yamlContent, tt.expectEnvVar) {
					t.Errorf("Unexpected environment variable %s found when no secrets present", tt.expectEnvVar)
				}
			}

			// Check if MCP config uses environment variable reference
			if tt.expectRedaction {
				if !strings.Contains(yamlContent, tt.expectMCPConfigValue) {
					t.Errorf("Expected MCP config to contain %s but it didn't", tt.expectMCPConfigValue)
				}

				// Ensure the secret expression itself is NOT in the MCP config JSON
				// (it should only be in env vars and redaction step)
				mcpConfigStart := strings.Index(yamlContent, "cat > /home/runner/.copilot/mcp-config.json << EOF")
				mcpConfigEnd := strings.Index(yamlContent[mcpConfigStart:], "EOF\n")
				if mcpConfigStart != -1 && mcpConfigEnd != -1 {
					mcpConfig := yamlContent[mcpConfigStart : mcpConfigStart+mcpConfigEnd]
					if strings.Contains(mcpConfig, "${{ secrets.") {
						t.Errorf("MCP config should not contain secret expressions, found secret in config")
					}
				}
			}

			// Check if secret is in redaction list
			if tt.expectRedaction {
				// Extract the secret name from the workflow
				secretName := ""
				if strings.Contains(tt.workflow, "TEST_DOMAIN") {
					secretName = "TEST_DOMAIN"
				} else if strings.Contains(tt.workflow, "API_KEY") {
					secretName = "API_KEY"
				}

				if secretName != "" {
					expectedRedactionEnv := "SECRET_" + secretName + ": ${{ secrets." + secretName + " }}"
					if !strings.Contains(yamlContent, expectedRedactionEnv) {
						t.Errorf("Expected secret %s to be in redaction step environment variables", secretName)
					}
				}
			}
		})
	}
}

// TestExtractSecretsFromAllowedDomains tests the helper function
func TestExtractSecretsFromAllowedDomains(t *testing.T) {
	tests := []struct {
		name            string
		allowedDomains  []string
		expectedSecrets map[string]string
	}{
		{
			name: "Single secret",
			allowedDomains: []string{
				"${{ secrets.TEST_DOMAIN }}",
			},
			expectedSecrets: map[string]string{
				"TEST_DOMAIN": "${{ secrets.TEST_DOMAIN }}",
			},
		},
		{
			name: "Multiple secrets",
			allowedDomains: []string{
				"${{ secrets.API_KEY }}",
				"example.com",
				"${{ secrets.ANOTHER_SECRET }}",
			},
			expectedSecrets: map[string]string{
				"API_KEY":        "${{ secrets.API_KEY }}",
				"ANOTHER_SECRET": "${{ secrets.ANOTHER_SECRET }}",
			},
		},
		{
			name: "Secrets with whitespace",
			allowedDomains: []string{
				"${{secrets.TEST_SECRET}}",
				"${{  secrets.SPACED_SECRET  }}",
			},
			expectedSecrets: map[string]string{
				"TEST_SECRET":   "${{secrets.TEST_SECRET}}",
				"SPACED_SECRET": "${{  secrets.SPACED_SECRET  }}",
			},
		},
		{
			name: "No secrets",
			allowedDomains: []string{
				"example.com",
				"test.org",
			},
			expectedSecrets: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets := extractSecretsFromAllowedDomains(tt.allowedDomains)

			if len(secrets) != len(tt.expectedSecrets) {
				t.Errorf("Expected %d secrets, got %d", len(tt.expectedSecrets), len(secrets))
			}

			for secretName, secretExpr := range tt.expectedSecrets {
				if actual, ok := secrets[secretName]; !ok {
					t.Errorf("Expected secret %s not found", secretName)
				} else if actual != secretExpr {
					t.Errorf("Expected secret %s to have expression %s, got %s", secretName, secretExpr, actual)
				}
			}
		})
	}
}

// TestReplaceSecretsInAllowedDomains tests the helper function
func TestReplaceSecretsInAllowedDomains(t *testing.T) {
	tests := []struct {
		name            string
		allowedDomains  []string
		secrets         map[string]string
		expectedDomains []string
	}{
		{
			name: "Replace single secret",
			allowedDomains: []string{
				"${{ secrets.TEST_DOMAIN }}",
			},
			secrets: map[string]string{
				"TEST_DOMAIN": "${{ secrets.TEST_DOMAIN }}",
			},
			expectedDomains: []string{
				"${PLAYWRIGHT_ALLOWED_DOMAIN_TEST_DOMAIN}",
			},
		},
		{
			name: "Replace multiple secrets",
			allowedDomains: []string{
				"${{ secrets.API_KEY }}",
				"example.com",
				"${{ secrets.ANOTHER_SECRET }}",
			},
			secrets: map[string]string{
				"API_KEY":        "${{ secrets.API_KEY }}",
				"ANOTHER_SECRET": "${{ secrets.ANOTHER_SECRET }}",
			},
			expectedDomains: []string{
				"${PLAYWRIGHT_ALLOWED_DOMAIN_API_KEY}",
				"example.com",
				"${PLAYWRIGHT_ALLOWED_DOMAIN_ANOTHER_SECRET}",
			},
		},
		{
			name: "No secrets to replace",
			allowedDomains: []string{
				"example.com",
				"test.org",
			},
			secrets: map[string]string{},
			expectedDomains: []string{
				"example.com",
				"test.org",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceSecretsInAllowedDomains(tt.allowedDomains, tt.secrets)

			if len(result) != len(tt.expectedDomains) {
				t.Errorf("Expected %d domains, got %d", len(tt.expectedDomains), len(result))
			}

			for i, expected := range tt.expectedDomains {
				if result[i] != expected {
					t.Errorf("Expected domain[%d] to be %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}
