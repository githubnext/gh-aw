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

// ReplaceSecretsWithBashVars replaces secret expressions in a value with bash env var references.
// Example: "${{ secrets.DD_API_KEY }}" -> "${DD_API_KEY}"
// Unlike ReplaceSecretsWithEnvVars, this does NOT add a backslash prefix, so bash expands
// the variable at runtime. Used for non-Copilot MCP server env blocks where the step env block
// already holds the corresponding env vars (injected by collectMCPEnvironmentVariables),
// preventing direct secret interpolation in run blocks (RGS-008 compliance).
func ReplaceSecretsWithBashVars(value string) string {
	result := value
	secrets := ExtractSecretsFromValue(value)
	for varName, secretExpr := range secrets {
		result = strings.ReplaceAll(result, secretExpr, "${"+varName+"}")
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

// gitHubContextExprPattern matches ${{ github.PROPERTY }} expressions where PROPERTY is a
// simple dotted identifier (e.g., github.workflow, github.ref_name, github.run_id).
// Complex expressions with operators (||, &&) are excluded by the regex. Nested dotted
// properties such as github.event.issue.title may still match this pattern, but are only
// accepted later if they are present in gitHubContextEnvVarMap.
var gitHubContextExprPattern = regexp.MustCompile(`\$\{\{\s*github\.([a-z][a-z0-9_.]*)\s*\}\}`)

// gitHubContextEnvVarMap maps common github.* context properties to their corresponding
// GitHub Actions runner environment variables (always available on all runners).
// See: https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
var gitHubContextEnvVarMap = map[string]string{
	"workflow":         "GITHUB_WORKFLOW",
	"run_id":           "GITHUB_RUN_ID",
	"run_number":       "GITHUB_RUN_NUMBER",
	"run_attempt":      "GITHUB_RUN_ATTEMPT",
	"actor":            "GITHUB_ACTOR",
	"repository":       "GITHUB_REPOSITORY",
	"event_name":       "GITHUB_EVENT_NAME",
	"sha":              "GITHUB_SHA",
	"ref":              "GITHUB_REF",
	"ref_name":         "GITHUB_REF_NAME",
	"ref_type":         "GITHUB_REF_TYPE",
	"head_ref":         "GITHUB_HEAD_REF",
	"base_ref":         "GITHUB_BASE_REF",
	"server_url":       "GITHUB_SERVER_URL",
	"job":              "GITHUB_JOB",
	"action":           "GITHUB_ACTION",
	"workspace":        "GITHUB_WORKSPACE",
	"workflow_ref":     "GITHUB_WORKFLOW_REF",
	"workflow_sha":     "GITHUB_WORKFLOW_SHA",
	"repository_owner": "GITHUB_REPOSITORY_OWNER",
	"triggering_actor": "GITHUB_TRIGGERING_ACTOR",
	"token":            "GITHUB_TOKEN",
}

// ExtractGitHubContextExpressionsFromValue extracts all simple ${{ github.X }} expressions from a
// string value and maps them to their corresponding GitHub Actions runner environment variable names.
// Only well-known context properties present in gitHubContextEnvVarMap are extracted; nested
// properties like github.event.issue.title are matched by the regex but filtered out by the map.
// Returns a map of env var name -> full expression.
//
// Examples:
//   - "${{ github.workflow }}" -> {"GITHUB_WORKFLOW": "${{ github.workflow }}"}
//   - "${{ github.ref_name }}" -> {"GITHUB_REF_NAME": "${{ github.ref_name }}"}
//   - "${{ github.event.issue.title }}" -> {} (not in gitHubContextEnvVarMap, skipped)
func ExtractGitHubContextExpressionsFromValue(value string) map[string]string {
	result := make(map[string]string)

	matches := gitHubContextExprPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		property := match[1]
		fullExpr := match[0]

		if envVar, known := gitHubContextEnvVarMap[property]; known {
			result[envVar] = fullExpr
			secretLog.Printf("Extracted GitHub context expression: %s -> %s", fullExpr, envVar)
		}
	}

	return result
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
