// This file provides utilities for processing GitHub Agentic Workflows.
//
// # String Processing Patterns
//
// This package implements two distinct patterns for string processing:
//
// ## Sanitize Pattern: Character Validity
//
// Sanitize functions remove or replace invalid characters to create valid identifiers,
// file names, or artifact names. Use sanitize functions when you need to ensure a string
// contains only valid characters for a specific context.
//
// Functions:
//   - SanitizeName: Configurable sanitization with character preservation options
//   - SanitizeWorkflowName: Sanitizes for artifact names and file paths (preserves dots, underscores)
//   - SanitizeWorkflowIDForCacheKey: Sanitizes workflow ID for use in cache keys (removes hyphens)
//   - sanitizeJobName: Sanitizes workflow name to a valid GitHub Actions job name
//   - sanitizeRefForPath: Sanitizes a git ref for use in a file path
//   - SanitizeIdentifier: Creates clean identifiers for user agents
//
// Example:
//
//	// User input with invalid characters
//	input := "My Workflow: Test/Build"
//	result := SanitizeWorkflowName(input)
//	// Returns: "my-workflow-test-build"
//
// ## Normalize Pattern: Format Standardization
//
// Normalize functions standardize format by removing extensions, converting between
// naming conventions, or applying consistent formatting rules. Use normalize functions
// when converting between different representations of the same logical entity.
//
// Functions:
//   - stringutil.NormalizeWorkflowName: Removes file extensions (.md, .lock.yml)
//   - stringutil.NormalizeSafeOutputIdentifier: Converts dashes to underscores
//
// Example:
//
//	// File name to base identifier
//	input := "weekly-research.md"
//	result := stringutil.NormalizeWorkflowName(input)
//	// Returns: "weekly-research"
//
// ## String Truncation
//
// Two truncation functions exist for different purposes:
//
// ShortenCommand (this package):
//   - Domain-specific for workflow log parsing
//   - Fixed 20-character length
//   - Replaces newlines with spaces (bash commands can be multi-line)
//   - Creates identifiers like "bash_echo hello world..."
//
// stringutil.Truncate:
//   - General-purpose string truncation
//   - Configurable maximum length
//   - No special character handling
//   - Used for display formatting in CLI output
//
// Choose based on your use case:
//   - Use ShortenCommand for bash command identifiers in workflow logs
//   - Use stringutil.Truncate for general string display truncation
//
// ## When to Use Each Pattern
//
// Use SANITIZE when:
//   - Processing user input that may contain invalid characters
//   - Creating identifiers, artifact names, or file paths
//   - Need to ensure character validity for a specific context
//
// Use NORMALIZE when:
//   - Converting between file names and identifiers (removing extensions)
//   - Standardizing naming conventions (dashes to underscores)
//   - Input is already valid but needs format conversion
//
// See scratchpad/string-sanitization-normalization.md for detailed guidance.

package workflow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var stringsLog = logger.New("workflow:strings")

var multipleHyphens = regexp.MustCompile(`-+`)

// SanitizeOptions configures the behavior of the SanitizeName function.
type SanitizeOptions struct {
	// PreserveSpecialChars is a list of special characters to preserve during sanitization.
	// Common characters include '.', '_'. If nil or empty, only alphanumeric and hyphens are preserved.
	PreserveSpecialChars []rune

	// TrimHyphens controls whether leading and trailing hyphens are removed from the result.
	// When true, hyphens at the start and end of the sanitized name are trimmed.
	TrimHyphens bool

	// DefaultValue is returned when the sanitized name is empty after all transformations.
	// If empty string, no default is applied.
	DefaultValue string
}

// SanitizeName sanitizes a string for use as an identifier, file name, or similar context.
// It provides configurable behavior through the SanitizeOptions parameter.
//
// The function performs the following transformations:
//   - Converts to lowercase
//   - Replaces common separators (colons, slashes, backslashes, spaces) with hyphens
//   - Replaces underscores with hyphens unless preserved in opts.PreserveSpecialChars
//   - Removes or replaces characters based on opts.PreserveSpecialChars
//   - Consolidates multiple consecutive hyphens into a single hyphen
//   - Optionally trims leading/trailing hyphens (controlled by opts.TrimHyphens)
//   - Returns opts.DefaultValue if the result is empty (controlled by opts.DefaultValue)
//
// Example:
//
//	// Preserve dots and underscores (like SanitizeWorkflowName)
//	opts := &SanitizeOptions{
//	    PreserveSpecialChars: []rune{'.', '_'},
//	}
//	SanitizeName("My.Workflow_Name", opts) // returns "my.workflow_name"
//
//	// Trim hyphens and use default (like SanitizeIdentifier)
//	opts := &SanitizeOptions{
//	    TrimHyphens:  true,
//	    DefaultValue: "default-name",
//	}
//	SanitizeName("@@@", opts) // returns "default-name"
func SanitizeName(name string, opts *SanitizeOptions) string {
	if stringsLog.Enabled() {
		preserveCount := 0
		trimHyphens := false
		if opts != nil {
			preserveCount = len(opts.PreserveSpecialChars)
			trimHyphens = opts.TrimHyphens
		}
		stringsLog.Printf("Sanitizing name: input=%q, preserve_chars=%d, trim_hyphens=%t",
			name, preserveCount, trimHyphens)
	}

	// Handle nil options
	if opts == nil {
		opts = &SanitizeOptions{}
	}

	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace common separators with hyphens
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, " ", "-")

	// Check if underscores should be preserved
	preserveUnderscore := slices.Contains(opts.PreserveSpecialChars, '_')

	// Replace underscores with hyphens if not preserved
	if !preserveUnderscore {
		result = strings.ReplaceAll(result, "_", "-")
	}

	// Build character preservation pattern based on options
	var preserveChars strings.Builder
	preserveChars.WriteString("a-z0-9-") // Always preserve alphanumeric and hyphens
	if len(opts.PreserveSpecialChars) > 0 {
		for _, char := range opts.PreserveSpecialChars {
			// Escape special regex characters
			switch char {
			case '.', '_':
				preserveChars.WriteRune(char)
			}
		}
	}

	// Create pattern for characters to remove/replace
	pattern := regexp.MustCompile(`[^` + preserveChars.String() + `]+`)

	// Replace unwanted characters with hyphens or empty based on context
	if len(opts.PreserveSpecialChars) > 0 {
		// Replace with hyphens (SanitizeWorkflowName behavior)
		result = pattern.ReplaceAllString(result, "-")
	} else {
		// Remove completely (SanitizeIdentifier behavior)
		result = pattern.ReplaceAllString(result, "")
	}

	// Consolidate multiple consecutive hyphens into a single hyphen
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Optionally trim leading/trailing hyphens
	if opts.TrimHyphens {
		result = strings.Trim(result, "-")
	}

	// Return default value if result is empty
	if result == "" && opts.DefaultValue != "" {
		stringsLog.Printf("Sanitized name is empty, using default: %q", opts.DefaultValue)
		return opts.DefaultValue
	}

	stringsLog.Printf("Sanitized name result: %q", result)
	return result
}

// SanitizeWorkflowName sanitizes a workflow name for use in artifact names and file paths.
// It converts the name to lowercase and replaces or removes characters that are invalid
// in YAML artifact names or filesystem paths.
//
// This is a SANITIZE function (character validity pattern). Use this when processing
// user input or workflow names that may contain invalid characters. Do NOT use this
// for removing file extensions - use stringutil.NormalizeWorkflowName instead.
//
// The function performs the following transformations:
//   - Converts to lowercase
//   - Replaces colons, slashes, backslashes, and spaces with hyphens
//   - Replaces any remaining special characters (except dots, underscores, and hyphens) with hyphens
//   - Consolidates multiple consecutive hyphens into a single hyphen
//
// Example inputs and outputs:
//
//	SanitizeWorkflowName("My Workflow: Test/Build")  // returns "my-workflow-test-build"
//	SanitizeWorkflowName("Weekly Research v2.0")     // returns "weekly-research-v2.0"
//	SanitizeWorkflowName("test_workflow")            // returns "test_workflow"
//
// See package documentation for guidance on when to use sanitize vs normalize patterns.
func SanitizeWorkflowName(name string) string {
	return SanitizeName(name, &SanitizeOptions{
		PreserveSpecialChars: []rune{'.', '_', '-'},
		TrimHyphens:          false,
		DefaultValue:         "",
	})
}

// ShortenCommand creates a short identifier for bash commands in workflow logs.
// It replaces newlines with spaces and truncates to 20 characters if needed.
//
// This is a domain-specific function for workflow log parsing. It creates
// unique identifiers for bash commands by:
//   - Replacing newlines with spaces (bash commands can be multi-line)
//   - Truncating to a fixed 20 characters with "..." suffix
//   - Producing identifiers like "bash_echo hello world..."
//
// For general-purpose string truncation with configurable length,
// use stringutil.Truncate instead.
func ShortenCommand(command string) string {
	// Take first 20 characters and remove newlines
	shortened := strings.ReplaceAll(command, "\n", " ")
	if len(shortened) > 20 {
		shortened = shortened[:20] + "..."
	}
	return shortened
}

// GenerateHeredocDelimiterFromSeed creates a stable heredoc delimiter derived from a seed
// (typically the workflow frontmatter hash hex string) so that repeated compilations of the
// same workflow produce identical lock files.
//
// When seed is non-empty, the 16-character hex tag is derived deterministically via
// HMAC-SHA256(key=seed, data=UPPER(name)), taking the first 8 bytes of the MAC.
// Using HMAC (with the seed as the key and the name as the message) avoids any
// length-extension or concatenation-collision concerns. This preserves the
// injection-resistance guarantee (an attacker who cannot control the frontmatter hash
// cannot predict the delimiter) while also making the compiled output stable.
//
// When seed is empty, the function falls back to crypto/rand — the same behaviour as
// GenerateHeredocDelimiter — so callers that lack a hash continue to work correctly.
func GenerateHeredocDelimiterFromSeed(name string, seed string) string {
	upperName := strings.ToUpper(name)
	var tag string
	if seed != "" {
		mac := hmac.New(sha256.New, []byte(seed))
		mac.Write([]byte(upperName))
		tag = hex.EncodeToString(mac.Sum(nil)[:8]) // first 8 bytes → 16 hex chars
	} else {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		tag = hex.EncodeToString(b)
	}
	if name == "" {
		return "GH_AW_" + tag + "_EOF"
	}
	return "GH_AW_" + upperName + "_" + tag + "_EOF"
}

// PrettifyToolName removes "mcp__" prefix and formats tool names nicely
func PrettifyToolName(toolName string) string {
	// Handle MCP tools: "mcp__github__search_issues" -> "github_search_issues"
	// Avoid colons and leave underscores as-is
	if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.Split(toolName, "__")
		if len(parts) >= 3 {
			provider := parts[1]
			method := strings.Join(parts[2:], "_")
			return fmt.Sprintf("%s_%s", provider, method)
		}
		// If format is unexpected, just remove the mcp__ prefix
		return strings.TrimPrefix(toolName, "mcp__")
	}

	// Handle bash specially - keep as "bash"
	if strings.ToLower(toolName) == "bash" {
		return "bash"
	}

	// Return other tool names as-is
	return toolName
}

// SanitizeWorkflowIDForCacheKey sanitizes a workflow ID for use in cache keys.
// It removes all hyphens and converts to lowercase to create a filesystem-safe identifier.
// Example: "Smoke-Copilot" -> "smokecopilot"
func SanitizeWorkflowIDForCacheKey(workflowID string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(workflowID)
	// Remove all hyphens
	sanitized = strings.ReplaceAll(sanitized, "-", "")
	return sanitized
}

// sanitizeJobName converts a workflow name to a valid GitHub Actions job name.
// It delegates normalization to NormalizeSafeOutputIdentifier (which converts
// hyphens to underscores), then converts underscores back to hyphens for
// GitHub Actions job name conventions.
func sanitizeJobName(workflowName string) string {
	normalized := stringutil.NormalizeSafeOutputIdentifier(workflowName)
	// NormalizeSafeOutputIdentifier uses underscores; convert to hyphens for job names
	return strings.ReplaceAll(normalized, "_", "-")
}

// sanitizeRefForPath sanitizes a git ref for use in a file path.
// Replaces characters that are problematic in file paths with safe alternatives.
func sanitizeRefForPath(ref string) string {
	// Replace slashes with dashes (for refs like "feature/my-branch")
	sanitized := strings.ReplaceAll(ref, "/", "-")
	// Replace other problematic characters
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	return sanitized
}

// SanitizeIdentifier sanitizes a workflow name to create a safe identifier
// suitable for use as a user agent string or similar context.
//
// This is a SANITIZE function (character validity pattern). Use this when creating
// identifiers that must be purely alphanumeric with hyphens, with no special characters
// preserved. Unlike SanitizeWorkflowName which preserves dots and underscores, this
// function removes ALL special characters except hyphens.
//
// The function:
//   - Converts to lowercase
//   - Replaces spaces and underscores with hyphens
//   - Removes non-alphanumeric characters (except hyphens)
//   - Consolidates multiple hyphens into a single hyphen
//   - Trims leading and trailing hyphens
//   - Returns "github-agentic-workflow" if the result would be empty
//
// Example inputs and outputs:
//
//	SanitizeIdentifier("My Workflow")         // returns "my-workflow"
//	SanitizeIdentifier("test_workflow")       // returns "test-workflow"
//	SanitizeIdentifier("@@@")                 // returns "github-agentic-workflow" (default)
//	SanitizeIdentifier("Weekly v2.0")         // returns "weekly-v2-0"
//
// This function uses the unified SanitizeName function with options configured
// to trim leading/trailing hyphens and return a default value for empty results.
// Hyphens are preserved by default in SanitizeName, not via PreserveSpecialChars.
//
// See package documentation for guidance on when to use sanitize vs normalize patterns.
func SanitizeIdentifier(name string) string {
	stringsLog.Printf("Sanitizing identifier: %s", name)
	result := SanitizeName(name, &SanitizeOptions{
		PreserveSpecialChars: []rune{},
		TrimHyphens:          true,
		DefaultValue:         "github-agentic-workflow",
	})
	if result != name {
		stringsLog.Printf("Sanitized identifier: %s -> %s", name, result)
	}
	return result
}

// formatList formats a list of strings as a comma-separated list with natural language conjunction
func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}
	return fmt.Sprintf("%s, and %s", formatList(items[:len(items)-1]), items[len(items)-1])
}
