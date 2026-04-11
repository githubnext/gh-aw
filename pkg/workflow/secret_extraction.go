package workflow

import (
	"maps"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var secretLog = logger.New("workflow:secret_extraction")

// Pre-compiled regex for secret extraction (performance optimization)
// Matches: ${{ secrets.SECRET_NAME }} or ${{ secrets.SECRET_NAME || 'default' }}
var secretExprPattern = regexp.MustCompile(`\$\{\{\s*secrets\.([A-Z_][A-Z0-9_]*)\s*(?:\|\|.*?)?\s*\}\}`)

// Pre-compiled regex patterns for ExtractSecretsFromValue (performance optimization)
var (
	// secretsExprFindPattern matches all ${{ ... }} expressions in a value
	secretsExprFindPattern = regexp.MustCompile(`\$\{\{[^}]+\}\}`)
	// secretsNamePattern extracts the secret variable name from an expression
	secretsNamePattern = regexp.MustCompile(`secrets\.([A-Z_][A-Z0-9_]*)`)
)

// SecretExpression represents a parsed secret expression
type SecretExpression struct {
	VarName  string // The secret variable name (e.g., "DD_API_KEY")
	FullExpr string // The full expression (e.g., "${{ secrets.DD_API_KEY }}")
}

// ExtractSecretName extracts just the secret name from a GitHub Actions expression
// Examples:
//   - "${{ secrets.DD_API_KEY }}" -> "DD_API_KEY"
//   - "${{ secrets.DD_SITE || 'datadoghq.com' }}" -> "DD_SITE"
//   - "plain value" -> ""
func ExtractSecretName(value string) string {
	matches := secretExprPattern.FindStringSubmatch(value)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// ExtractSecretsFromValue extracts all GitHub Actions secret expressions from a string value
// Returns a map of environment variable names to their full secret expressions
// This function detects secrets in both simple expressions and sub-expressions:
// Examples:
//   - "${{ secrets.DD_API_KEY }}" -> {"DD_API_KEY": "${{ secrets.DD_API_KEY }}"}
//   - "${{ secrets.DD_SITE || 'datadoghq.com' }}" -> {"DD_SITE": "${{ secrets.DD_SITE || 'datadoghq.com' }}"}
//   - "Bearer ${{ secrets.TOKEN }}" -> {"TOKEN": "${{ secrets.TOKEN }}"}
//   - "${{ github.workflow && secrets.TOKEN }}" -> {"TOKEN": "${{ github.workflow && secrets.TOKEN }}"}
//   - "${{ (github.actor || secrets.HIDDEN) }}" -> {"HIDDEN": "${{ (github.actor || secrets.HIDDEN) }}"}
func ExtractSecretsFromValue(value string) map[string]string {
	secrets := make(map[string]string)

	// Find all ${{ ... }} expressions in the value
	// Pattern matches from ${{ to }} allowing nested content
	expressions := secretsExprFindPattern.FindAllString(value, -1)

	// For each expression, check if it contains secrets.VARIABLE_NAME
	// This handles both simple cases like "${{ secrets.TOKEN }}"
	// and complex sub-expressions like "${{ github.workflow && secrets.TOKEN }}"
	for _, expr := range expressions {
		matches := secretsNamePattern.FindAllStringSubmatch(expr, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				varName := match[1]
				// Store the full expression that contains this secret
				secrets[varName] = expr
				secretLog.Printf("Extracted secret: %s from expression: %s", varName, expr)
			}
		}
	}

	if len(secrets) > 0 {
		secretLog.Printf("Extracted %d secrets from value", len(secrets))
	}
	return secrets
}

// ExtractSecretsFromMap extracts all secrets from a map of string values
// Returns a map of environment variable names to their full secret expressions
// Example:
//
//	Input: {"DD_API_KEY": "${{ secrets.DD_API_KEY }}", "DD_SITE": "${{ secrets.DD_SITE || 'default' }}"}
//	Output: {"DD_API_KEY": "${{ secrets.DD_API_KEY }}", "DD_SITE": "${{ secrets.DD_SITE || 'default' }}"}
func ExtractSecretsFromMap(values map[string]string) map[string]string {
	secretLog.Printf("Extracting secrets from map with %d entries", len(values))
	allSecrets := make(map[string]string)

	for _, value := range values {
		secrets := ExtractSecretsFromValue(value)
		maps.Copy(allSecrets, secrets)
	}

	secretLog.Printf("Extracted total of %d unique secrets from map", len(allSecrets))
	return allSecrets
}

// ExtractEnvExpressionsFromMap extracts all env variable expressions from a map of string values
// Returns a map of environment variable names to their full env expressions (including fallbacks)
// Example:
//
//	Input: {"SENTRY_HOST": "${{ env.SENTRY_HOST || 'https://sentry.io' }}", "DD_SITE": "${{ env.DD_SITE }}"}
//	Output: {"SENTRY_HOST": "${{ env.SENTRY_HOST || 'https://sentry.io' }}", "DD_SITE": "${{ env.DD_SITE }}"}
func ExtractEnvExpressionsFromMap(values map[string]string) map[string]string {
	secretLog.Printf("Extracting env expressions from map with %d entries", len(values))
	allEnvVars := make(map[string]string)

	for _, value := range values {
		envVars := ExtractEnvExpressionsFromValue(value)
		maps.Copy(allEnvVars, envVars)
	}

	secretLog.Printf("Extracted total of %d unique env expressions from map", len(allEnvVars))
	return allEnvVars
}

// ReplaceSecretsWithEnvVars replaces secret expressions in a value with environment variable references
// Example: "${{ secrets.DD_API_KEY }}" -> "\${DD_API_KEY}"
// The backslash is used to escape the ${} for proper JSON rendering in Copilot configs
func ReplaceSecretsWithEnvVars(value string, secrets map[string]string) string {
	result := value
	for varName, secretExpr := range secrets {
		// Replace ${{ secrets.VAR }} with \${VAR} (backslash-escaped for copilot JSON config)
		result = strings.ReplaceAll(result, secretExpr, "\\${"+varName+"}")
	}
	return result
}

// ExtractEnvExpressionsFromValue extracts all GitHub Actions env expressions from a string value
// Returns a map of environment variable names to their full env expressions
// Examples:
//   - "${{ env.SENTRY_HOST }}" -> {"SENTRY_HOST": "${{ env.SENTRY_HOST }}"}
//   - "${{ env.DD_SITE || 'default' }}" -> {"DD_SITE": "${{ env.DD_SITE || 'default' }}"}
func ExtractEnvExpressionsFromValue(value string) map[string]string {
	envExpressions := make(map[string]string)

	start := 0
	for {
		// Find the start of an expression
		startIdx := strings.Index(value[start:], "${{ env.")
		if startIdx == -1 {
			break
		}
		startIdx += start

		// Find the end of the expression
		endIdx := strings.Index(value[startIdx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + 2 // Include the closing }}

		// Extract the full expression
		fullExpr := value[startIdx:endIdx]

		// Extract the variable name from "env.VARIABLE_NAME" or "env.VARIABLE_NAME ||"
		envPart := strings.TrimPrefix(fullExpr, "${{ env.")
		envPart = strings.TrimSuffix(envPart, "}}")
		envPart = strings.TrimSpace(envPart)

		// Find the variable name (everything before space, ||, or end)
		varName := envPart
		if spaceIdx := strings.IndexAny(varName, " |"); spaceIdx != -1 {
			varName = varName[:spaceIdx]
		}

		// Store the variable name and full expression
		if varName != "" {
			envExpressions[varName] = fullExpr
			secretLog.Printf("Extracted env expression: %s", varName)
		}

		start = endIdx
	}

	return envExpressions
}

// ExtractAllExpressionsFromValue extracts ALL GitHub Actions ${{ }} expressions from a string
// and generates environment variable names for each.
// Returns a map of environment variable names to their full expressions.
// For secrets, the env var name is the secret name (e.g., "DD_API_KEY").
// For github context, the env var name is the uppercased dotted path prefixed with GH_AW_
// (e.g., ${{ github.workflow }} -> "GH_AW_GITHUB_WORKFLOW").
// For other expressions, a sanitized uppercase version is used.
func ExtractAllExpressionsFromValue(value string) map[string]string {
	result := make(map[string]string)

	expressions := secretsExprFindPattern.FindAllString(value, -1)
	for _, expr := range expressions {
		varName := expressionToEnvVarName(expr)
		if varName != "" {
			result[varName] = expr
			secretLog.Printf("Extracted expression: %s -> env var: %s", expr, varName)
		}
	}

	return result
}

// expressionToEnvVarName converts a GitHub Actions expression to a suitable environment variable name.
// Examples:
//   - "${{ secrets.DD_API_KEY }}" -> "DD_API_KEY"
//   - "${{ github.workflow }}" -> "GH_AW_GITHUB_WORKFLOW"
//   - "${{ steps.parse-guard-vars.outputs.blocked_users }}" -> "GH_AW_GUARD_BLOCKED_USERS"
func expressionToEnvVarName(expr string) string {
	inner := strings.TrimPrefix(expr, "${{")
	inner = strings.TrimSuffix(inner, "}}")
	inner = strings.TrimSpace(inner)

	// Handle secrets: ${{ secrets.X }} -> X
	if m := secretsNamePattern.FindStringSubmatch(expr); len(m) >= 2 {
		return m[1]
	}

	// Handle github context: ${{ github.X }} -> GH_AW_GITHUB_X
	if name, ok := strings.CutPrefix(inner, "github."); ok {
		return "GH_AW_GITHUB_" + sanitizeEnvVarName(name)
	}

	// Handle steps outputs: ${{ steps.X.outputs.Y }} -> GH_AW_STEP_X_Y
	if name, ok := strings.CutPrefix(inner, "steps."); ok {
		name = strings.ReplaceAll(name, ".outputs.", "_")
		return "GH_AW_STEP_" + sanitizeEnvVarName(name)
	}

	// General case: sanitize to valid env var name
	return "GH_AW_" + sanitizeEnvVarName(inner)
}

// sanitizeEnvVarName converts a string to a valid environment variable name
// by uppercasing and replacing non-alphanumeric characters with underscores.
func sanitizeEnvVarName(s string) string {
	upper := strings.ToUpper(s)
	var result strings.Builder
	for _, r := range upper {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	return result.String()
}

// ReplaceTemplateExpressionsWithEnvVars replaces all template expressions with environment variable references
// Handles: secrets.*, env.*, and github.workspace
// Examples:
//   - "${{ secrets.DD_API_KEY }}" -> "\${DD_API_KEY}"
//   - "${{ env.SENTRY_HOST }}" -> "\${SENTRY_HOST}"
//   - "${{ github.workspace }}" -> "\${GITHUB_WORKSPACE}"
func ReplaceTemplateExpressionsWithEnvVars(value string) string {
	result := value

	// Extract and replace secrets
	secrets := ExtractSecretsFromValue(value)
	for varName, secretExpr := range secrets {
		result = strings.ReplaceAll(result, secretExpr, "\\${"+varName+"}")
	}

	// Extract and replace env vars
	envVars := ExtractEnvExpressionsFromValue(value)
	for varName, envExpr := range envVars {
		result = strings.ReplaceAll(result, envExpr, "\\${"+varName+"}")
	}

	// Replace github.workspace with GITHUB_WORKSPACE env var
	result = strings.ReplaceAll(result, "${{ github.workspace }}", "\\${GITHUB_WORKSPACE}")

	return result
}
