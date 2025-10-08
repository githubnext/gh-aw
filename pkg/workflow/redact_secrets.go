package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// CollectSecretReferences extracts all secret references from the workflow YAML
// This scans for patterns like ${{ secrets.SECRET_NAME }} or secrets.SECRET_NAME
func CollectSecretReferences(yamlContent string) []string {
	secretsMap := make(map[string]bool)

	// Pattern to match ${{ secrets.SECRET_NAME }} or secrets.SECRET_NAME
	// This matches both with and without the ${{ }} wrapper
	secretPattern := regexp.MustCompile(`secrets\.([A-Z][A-Z0-9_]*)`)

	matches := secretPattern.FindAllStringSubmatch(yamlContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			secretsMap[match[1]] = true
		}
	}

	// Convert map to sorted slice for consistent ordering
	secrets := make([]string, 0, len(secretsMap))
	for secret := range secretsMap {
		secrets = append(secrets, secret)
	}

	// Sort for consistent output
	SortStrings(secrets)

	return secrets
}

// generateSecretRedactionStep generates a workflow step that redacts secrets from files in /tmp
func (c *Compiler) generateSecretRedactionStep(yaml *strings.Builder, workflowData *WorkflowData, engine CodingAgentEngine, yamlContent string) {
	// Extract secret references from the generated YAML
	secretReferences := CollectSecretReferences(yamlContent)

	// If no secrets found, skip the redaction step
	if len(secretReferences) == 0 {
		return
	}

	yaml.WriteString("      - name: Redact secrets in logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the embedded JavaScript code
	jsCode := redactSecretsScript

	// Indent the JavaScript code properly for YAML
	lines := strings.Split(jsCode, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			yaml.WriteString("            " + line + "\n")
		} else {
			yaml.WriteString("\n")
		}
	}

	// Add environment variables
	yaml.WriteString("        env:\n")

	// Pass the list of secret names as a comma-separated string
	yaml.WriteString(fmt.Sprintf("          GITHUB_AW_SECRET_NAMES: '%s'\n", strings.Join(secretReferences, ",")))

	// Pass the actual secret values as environment variables so they can be redacted
	// Each secret will be available as an environment variable
	for _, secretName := range secretReferences {
		yaml.WriteString(fmt.Sprintf("          SECRET_%s: ${{ secrets.%s }}\n", secretName, secretName))
	}
}

// validateSecretReferences validates that secret references are valid
func validateSecretReferences(secrets []string) error {
	// Secret names must be valid environment variable names
	secretNamePattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

	for _, secret := range secrets {
		if !secretNamePattern.MatchString(secret) {
			return fmt.Errorf("invalid secret name: %s", secret)
		}
	}

	return nil
}
