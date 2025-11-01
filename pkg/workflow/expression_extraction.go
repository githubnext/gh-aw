package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ExpressionMapping represents a mapping between a GitHub expression and its environment variable
type ExpressionMapping struct {
	Original string // The original ${{ ... }} expression
	EnvVar   string // The GH_AW_ prefixed environment variable name
	Content  string // The expression content without ${{ }}
}

// ExpressionExtractor extracts GitHub Actions expressions from markdown content
// and creates environment variable mappings for them
type ExpressionExtractor struct {
	mappings map[string]*ExpressionMapping // key is the original expression
	counter  int
}

// NewExpressionExtractor creates a new ExpressionExtractor
func NewExpressionExtractor() *ExpressionExtractor {
	return &ExpressionExtractor{
		mappings: make(map[string]*ExpressionMapping),
		counter:  0,
	}
}

// ExtractExpressions extracts all ${{ ... }} expressions from the markdown content
// and creates environment variable mappings for each unique expression
func (e *ExpressionExtractor) ExtractExpressions(markdown string) ([]*ExpressionMapping, error) {
	// Regular expression to match GitHub Actions expressions: ${{ ... }}
	// Use (?s) flag for dotall mode, non-greedy matching
	expressionRegex := regexp.MustCompile(`\$\{\{(.*?)\}\}`)

	matches := expressionRegex.FindAllStringSubmatch(markdown, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// Extract the full original expression including ${{ }}
		originalExpr := match[0]

		// Extract the content (without ${{ }})
		content := strings.TrimSpace(match[1])

		// Skip if we've already seen this expression
		if _, exists := e.mappings[originalExpr]; exists {
			continue
		}

		// Generate environment variable name
		envVar := e.generateEnvVarName(content)

		// Create mapping
		mapping := &ExpressionMapping{
			Original: originalExpr,
			EnvVar:   envVar,
			Content:  content,
		}

		e.mappings[originalExpr] = mapping
	}

	// Convert map to sorted slice for consistent ordering
	var result []*ExpressionMapping
	for _, mapping := range e.mappings {
		result = append(result, mapping)
	}

	// Sort by original expression for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Original < result[j].Original
	})

	return result, nil
}

// generateEnvVarName generates a unique environment variable name for an expression
// Uses a hash of the content to create a stable, collision-free name
func (e *ExpressionExtractor) generateEnvVarName(content string) string {
	// Use SHA256 hash to generate a unique identifier
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	// Use first 8 characters of hash for brevity
	shortHash := hashStr[:8]

	// Create environment variable name
	return fmt.Sprintf("GH_AW_EXPR_%s", strings.ToUpper(shortHash))
}

// ReplaceExpressionsWithEnvVars replaces all ${{ ... }} expressions in the markdown
// with references to their corresponding environment variables
func (e *ExpressionExtractor) ReplaceExpressionsWithEnvVars(markdown string) string {
	result := markdown

	// Sort mappings by length of original expression (longest first)
	// This ensures we replace longer expressions before shorter ones
	// to avoid partial replacements
	var mappings []*ExpressionMapping
	for _, mapping := range e.mappings {
		mappings = append(mappings, mapping)
	}
	sort.Slice(mappings, func(i, j int) bool {
		return len(mappings[i].Original) > len(mappings[j].Original)
	})

	// Replace each expression with its environment variable reference
	for _, mapping := range mappings {
		// Use ${VAR_NAME} syntax for safety in shell scripts
		envVarRef := fmt.Sprintf("${%s}", mapping.EnvVar)
		result = strings.ReplaceAll(result, mapping.Original, envVarRef)
	}

	return result
}

// GetMappings returns all expression mappings
func (e *ExpressionExtractor) GetMappings() []*ExpressionMapping {
	var result []*ExpressionMapping
	for _, mapping := range e.mappings {
		result = append(result, mapping)
	}

	// Sort by environment variable name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].EnvVar < result[j].EnvVar
	})

	return result
}

// ExtractExpressionsFromStringSlice extracts expressions from a slice of strings
// Returns the number of expressions found
func (e *ExpressionExtractor) ExtractExpressionsFromStringSlice(values []string) int {
	count := 0
	for _, value := range values {
		// Extract expressions from this value
		_, err := e.ExtractExpressions(value)
		if err == nil && len(e.mappings) > count {
			count = len(e.mappings) - count
		}
	}
	return count
}

// ExtractExpressionsFromMap extracts expressions from a map's values
// Returns the number of expressions found
func (e *ExpressionExtractor) ExtractExpressionsFromMap(values map[string]string) int {
	count := 0
	for _, value := range values {
		// Extract expressions from this value
		_, err := e.ExtractExpressions(value)
		if err == nil && len(e.mappings) > count {
			count = len(e.mappings) - count
		}
	}
	return count
}

// ReplaceExpressionsInStringSlice replaces expressions in a slice of strings
// Returns a new slice with replacements applied
func (e *ExpressionExtractor) ReplaceExpressionsInStringSlice(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = e.ReplaceExpressionsWithEnvVars(value)
	}
	return result
}

// ReplaceExpressionsInMap replaces expressions in a map's values
// Returns a new map with replacements applied
func (e *ExpressionExtractor) ReplaceExpressionsInMap(values map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range values {
		result[key] = e.ReplaceExpressionsWithEnvVars(value)
	}
	return result
}

// ExtractExpressionsFromTools extracts all GitHub expressions from workflow tools
// This includes expressions in MCP server args, env, headers, and other configuration
func (e *ExpressionExtractor) ExtractExpressionsFromTools(tools map[string]any) error {
	for _, toolValue := range tools {
		// Convert to map if possible
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		// Extract from args array
		if args, ok := toolConfig["args"].([]any); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					_, _ = e.ExtractExpressions(argStr)
				}
			}
		}

		// Extract from entrypoint-args array
		if entrypointArgs, ok := toolConfig["entrypoint-args"].([]any); ok {
			for _, arg := range entrypointArgs {
				if argStr, ok := arg.(string); ok {
					_, _ = e.ExtractExpressions(argStr)
				}
			}
		}

		// Extract from env map
		if env, ok := toolConfig["env"].(map[string]any); ok {
			for _, value := range env {
				if valueStr, ok := value.(string); ok {
					_, _ = e.ExtractExpressions(valueStr)
				}
			}
		}

		// Extract from headers map (for HTTP MCP)
		if headers, ok := toolConfig["headers"].(map[string]any); ok {
			for _, value := range headers {
				if valueStr, ok := value.(string); ok {
					_, _ = e.ExtractExpressions(valueStr)
				}
			}
		}

		// Extract from proxy-args array
		if proxyArgs, ok := toolConfig["proxy-args"].([]any); ok {
			for _, arg := range proxyArgs {
				if argStr, ok := arg.(string); ok {
					_, _ = e.ExtractExpressions(argStr)
				}
			}
		}

		// Extract from command string
		if command, ok := toolConfig["command"].(string); ok {
			_, _ = e.ExtractExpressions(command)
		}

		// Extract from url string (for HTTP MCP)
		if url, ok := toolConfig["url"].(string); ok {
			_, _ = e.ExtractExpressions(url)
		}
	}

	return nil
}

// ReplaceExpressionsInTools creates a deep copy of tools with expressions replaced
// Returns a new tools map with all expressions replaced by environment variable references
func (e *ExpressionExtractor) ReplaceExpressionsInTools(tools map[string]any) map[string]any {
	result := make(map[string]any)

	for toolName, toolValue := range tools {
		// Convert to map if possible
		toolConfig, ok := toolValue.(map[string]any)
		if !ok {
			// If not a map, copy as-is
			result[toolName] = toolValue
			continue
		}

		// Create a copy of the tool config
		newConfig := make(map[string]any)
		for key, value := range toolConfig {
			newConfig[key] = value
		}

		// Replace in args array
		if args, ok := toolConfig["args"].([]any); ok {
			newArgs := make([]any, len(args))
			for i, arg := range args {
				if argStr, ok := arg.(string); ok {
					newArgs[i] = e.ReplaceExpressionsWithEnvVars(argStr)
				} else {
					newArgs[i] = arg
				}
			}
			newConfig["args"] = newArgs
		}

		// Replace in entrypoint-args array
		if entrypointArgs, ok := toolConfig["entrypoint-args"].([]any); ok {
			newEntrypointArgs := make([]any, len(entrypointArgs))
			for i, arg := range entrypointArgs {
				if argStr, ok := arg.(string); ok {
					newEntrypointArgs[i] = e.ReplaceExpressionsWithEnvVars(argStr)
				} else {
					newEntrypointArgs[i] = arg
				}
			}
			newConfig["entrypoint-args"] = newEntrypointArgs
		}

		// Replace in env map
		if env, ok := toolConfig["env"].(map[string]any); ok {
			newEnv := make(map[string]any)
			for key, value := range env {
				if valueStr, ok := value.(string); ok {
					newEnv[key] = e.ReplaceExpressionsWithEnvVars(valueStr)
				} else {
					newEnv[key] = value
				}
			}
			newConfig["env"] = newEnv
		}

		// Replace in headers map (for HTTP MCP)
		if headers, ok := toolConfig["headers"].(map[string]any); ok {
			newHeaders := make(map[string]any)
			for key, value := range headers {
				if valueStr, ok := value.(string); ok {
					newHeaders[key] = e.ReplaceExpressionsWithEnvVars(valueStr)
				} else {
					newHeaders[key] = value
				}
			}
			newConfig["headers"] = newHeaders
		}

		// Replace in proxy-args array
		if proxyArgs, ok := toolConfig["proxy-args"].([]any); ok {
			newProxyArgs := make([]any, len(proxyArgs))
			for i, arg := range proxyArgs {
				if argStr, ok := arg.(string); ok {
					newProxyArgs[i] = e.ReplaceExpressionsWithEnvVars(argStr)
				} else {
					newProxyArgs[i] = arg
				}
			}
			newConfig["proxy-args"] = newProxyArgs
		}

		// Replace in command string
		if command, ok := toolConfig["command"].(string); ok {
			newConfig["command"] = e.ReplaceExpressionsWithEnvVars(command)
		}

		// Replace in url string (for HTTP MCP)
		if url, ok := toolConfig["url"].(string); ok {
			newConfig["url"] = e.ReplaceExpressionsWithEnvVars(url)
		}

		result[toolName] = newConfig
	}

	return result
}
