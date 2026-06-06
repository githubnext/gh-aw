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
//   - SanitizeArtifactIdentifier: Creates clean identifiers for artifacts and user agents
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
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var stringsLog = logger.New("workflow:strings")

// SanitizeOptions configures the behavior of the SanitizeName function.
type SanitizeOptions = stringutil.SanitizeOptions

// SanitizeName sanitizes a string for use as an identifier, file name, or similar context.
func SanitizeName(name string, opts *SanitizeOptions) string {
	return stringutil.SanitizeName(name, opts)
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

// escapeYAMLSingleQuoted escapes single quotes for YAML single-quoted scalars by doubling each
// apostrophe per YAML 1.2.
func escapeYAMLSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
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
// Panics on crypto/rand failure when no seed is provided.
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
			panic("BUG: crypto/rand failed: " + err.Error())
		}
		tag = hex.EncodeToString(b)
	}
	if name == "" {
		return "GH_AW_" + tag + "_EOF"
	}
	return "GH_AW_" + upperName + "_" + tag + "_EOF"
}

// GenerateHeredocDelimiterFromContent creates a stable heredoc delimiter derived from the
// content it wraps. The 16-character hex tag is the first 8 bytes of SHA-256 of the content,
// so the delimiter is identical across builds whenever the content is unchanged and changes
// only when the content changes — reducing unnecessary diff noise and merge conflicts.
//
// The name prefix (e.g. "PROMPT", "MCP_CONFIG") is included for readability and to ensure
// that two different heredocs wrapping identical content still produce distinct delimiters.
func GenerateHeredocDelimiterFromContent(name string, content string) string {
	h := sha256.New()
	h.Write([]byte(strings.ToUpper(name)))
	h.Write([]byte(content))
	tag := hex.EncodeToString(h.Sum(nil)[:8])
	upperName := strings.ToUpper(name)
	if name == "" {
		return "GH_AW_" + tag + "_EOF"
	}
	return "GH_AW_" + upperName + "_" + tag + "_EOF"
}

// heredocDelimiterRE matches randomized heredoc delimiters of the form GH_AW_<NAME>_<16hexchars>_EOF.
// Used to normalize delimiters when comparing compiled output to skip unnecessary writes.
var heredocDelimiterRE = regexp.MustCompile(`GH_AW_([A-Z0-9_]+)_[0-9a-f]{16}_EOF`)

// normalizeHeredocDelimiters replaces randomized heredoc delimiter tokens with a stable
// placeholder so that two compilations of the same workflow compare as equal even though
// each run embeds different random tokens.
func normalizeHeredocDelimiters(content string) string {
	// Fast path: skip regex if content contains no heredoc delimiters
	if !strings.Contains(content, "GH_AW_") {
		return content
	}
	return heredocDelimiterRE.ReplaceAllString(content, "GH_AW_${1}_NORM_EOF")
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
	if strings.EqualFold(toolName, "bash") {
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

// SanitizeArtifactIdentifier sanitizes a workflow name to create a safe identifier
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
//	SanitizeArtifactIdentifier("My Workflow")         // returns "my-workflow"
//	SanitizeArtifactIdentifier("test_workflow")       // returns "test-workflow"
//	SanitizeArtifactIdentifier("@@@")                 // returns "github-agentic-workflow" (default)
//	SanitizeArtifactIdentifier("Weekly v2.0")         // returns "weekly-v2-0"
//
// This function uses the unified SanitizeName function with options configured
// to trim leading/trailing hyphens and return a default value for empty results.
// Hyphens are preserved by default in SanitizeName, not via PreserveSpecialChars.
//
// Note: Do not confuse with stringutil.SanitizeIdentifierName, which uses
// a different algorithm — it keeps [a-zA-Z0-9_] and replaces others with underscores,
// making it suitable for programming language identifiers (e.g. JavaScript, Python).
// SanitizeArtifactIdentifier instead produces hyphen-separated lowercase identifiers for
// workflow artifacts, job names, and user agent strings.
//
// See package documentation for guidance on when to use sanitize vs normalize patterns.
func SanitizeArtifactIdentifier(name string) string {
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
