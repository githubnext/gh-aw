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
