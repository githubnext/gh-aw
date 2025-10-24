package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// escapeSingleQuote escapes single quotes and backslashes in a string to prevent injection
// when embedding data in single-quoted YAML strings
func escapeSingleQuote(s string) string {
	// First escape backslashes, then escape single quotes
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

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
func (c *Compiler) generateSecretRedactionStep(yaml *strings.Builder, yamlContent string) {
	// Extract secret references from the generated YAML
	secretReferences := CollectSecretReferences(yamlContent)

	// Always record that we're adding a secret redaction step, even if no secrets found
	// This is important for validation to ensure the step ordering is correct
	c.stepOrderTracker.RecordSecretRedaction("Redact secrets in logs")

	// If no secrets found, we still generate the step but it will be a no-op at runtime
	// This ensures consistent step ordering and validation
	if len(secretReferences) == 0 {
		// Generate a minimal no-op redaction step for validation purposes
		yaml.WriteString("      - name: Redact secrets in logs\n")
		yaml.WriteString("        if: always()\n")
		yaml.WriteString("        run: echo 'No secrets to redact'\n")
		return
	}

	yaml.WriteString("      - name: Redact secrets in logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the embedded JavaScript code without comments
	WriteJavaScriptToYAML(yaml, redactSecretsScript)

	// Add environment variables
	yaml.WriteString("        env:\n")

	// Pass the list of secret names as a comma-separated string
	// Escape each secret reference to prevent injection when embedding in YAML
	escapedRefs := make([]string, len(secretReferences))
	for i, ref := range secretReferences {
		escapedRefs[i] = escapeSingleQuote(ref)
	}
	yaml.WriteString(fmt.Sprintf("          GH_AW_SECRET_NAMES: '%s'\n", strings.Join(escapedRefs, ",")))

	// Pass the actual secret values as environment variables so they can be redacted
	// Each secret will be available as an environment variable
	for _, secretName := range secretReferences {
		// Escape secret name to prevent injection in YAML
		escapedSecretName := escapeSingleQuote(secretName)
		// Use original secretName in GitHub Actions expression since it's already validated
		// to only contain safe characters (uppercase letters, numbers, underscores)
		yaml.WriteString(fmt.Sprintf("          SECRET_%s: ${{ secrets.%s }}\n", escapedSecretName, secretName))
	}
}
