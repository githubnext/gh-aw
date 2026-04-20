//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_ExtractFrontmatterFromContent validates the documented
// behavior of ExtractFrontmatterFromContent as described in the package README.md.
//
// Specification: Extracts YAML frontmatter between --- delimiters from markdown.
// The markdown body that follows the frontmatter serves as the AI agent's prompt text.
func TestSpec_PublicAPI_ExtractFrontmatterFromContent(t *testing.T) {
	t.Run("extracts YAML frontmatter between --- delimiters", func(t *testing.T) {
		content := "---\non: push\n---\n# My Workflow\nSome prompt text."
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err,
			"ExtractFrontmatterFromContent should not error on valid frontmatter")
		require.NotNil(t, result,
			"ExtractFrontmatterFromContent should return non-nil result")
		assert.NotNil(t, result.Frontmatter["on"],
			"result.Frontmatter should contain the 'on' key from YAML")
	})

	t.Run("markdown body follows frontmatter block", func(t *testing.T) {
		content := "---\non: push\n---\n# My Workflow\nPrompt text here."
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err,
			"ExtractFrontmatterFromContent should not error on valid content")
		assert.Contains(t, result.Markdown, "Prompt text here",
			"result.Markdown should contain the body text after frontmatter")
	})

	t.Run("content without frontmatter delimiter returns empty frontmatter", func(t *testing.T) {
		content := "# Just markdown\nNo frontmatter here."
		result, err := ExtractFrontmatterFromContent(content)
		require.NoError(t, err,
			"ExtractFrontmatterFromContent should not error on content without frontmatter")
		assert.Empty(t, result.Frontmatter,
			"result.Frontmatter should be empty when no --- delimiter is present")
	})
}

// TestSpec_PublicAPI_ExtractMarkdownContent validates the documented behavior
// of ExtractMarkdownContent as described in the package README.md.
//
// Specification: Returns the markdown body (everything after frontmatter).
func TestSpec_PublicAPI_ExtractMarkdownContent(t *testing.T) {
	t.Run("returns body after frontmatter block", func(t *testing.T) {
		content := "---\non: push\n---\n# Agent Prompt\nDo the thing."
		body, err := ExtractMarkdownContent(content)
		require.NoError(t, err,
			"ExtractMarkdownContent should not error on valid content")
		assert.Contains(t, body, "Do the thing",
			"ExtractMarkdownContent should return text after the frontmatter block")
	})

	t.Run("content without frontmatter returns full content as body", func(t *testing.T) {
		content := "# Just markdown\nNo frontmatter."
		body, err := ExtractMarkdownContent(content)
		require.NoError(t, err,
			"ExtractMarkdownContent should not error on content without frontmatter")
		assert.Contains(t, body, "No frontmatter",
			"ExtractMarkdownContent should return content as-is when no frontmatter present")
	})
}

// TestSpec_ScheduleDetection_IsCronExpression validates the documented behavior
// of IsCronExpression as described in the package README.md.
//
// Specification: Detects whether a string is already a cron expression.
func TestSpec_ScheduleDetection_IsCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "standard 5-field cron returns true",
			input:    "0 9 * * *",
			expected: true,
		},
		{
			name:     "every-5-minutes cron returns true",
			input:    "*/5 * * * *",
			expected: true,
		},
		{
			name:     "natural language schedule returns false",
			input:    "every day at 9am",
			expected: false,
		},
		{
			name:     "empty string returns false",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCronExpression(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsCronExpression(%q) should detect cron format correctly", tt.input)
		})
	}
}

// TestSpec_ScheduleDetection_IsDailyCron validates the documented behavior of
// IsDailyCron as described in the package README.md.
//
// Specification: Detects whether a cron expression runs daily.
func TestSpec_ScheduleDetection_IsDailyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "daily cron at 9am returns true",
			input:    "0 9 * * *",
			expected: true,
		},
		{
			name:     "weekly cron returns false",
			input:    "0 9 * * 1",
			expected: false,
		},
		{
			name:     "hourly cron returns false",
			input:    "0 * * * *",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDailyCron(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsDailyCron(%q) should detect daily cron correctly", tt.input)
		})
	}
}

// TestSpec_ScheduleDetection_IsHourlyCron validates the documented behavior of
// IsHourlyCron as described in the package README.md.
//
// Specification: Detects whether a cron expression runs hourly.
func TestSpec_ScheduleDetection_IsHourlyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			// The implementation requires the hour field to be an interval
			// pattern (*/N) rather than plain *. "0 */1 * * *" runs hourly.
			name:     "hourly cron with interval pattern returns true",
			input:    "0 */1 * * *",
			expected: true,
		},
		{
			name:     "daily cron returns false",
			input:    "0 9 * * *",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHourlyCron(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsHourlyCron(%q) should detect hourly cron correctly", tt.input)
		})
	}
}

// TestSpec_ScheduleDetection_IsWeeklyCron validates the documented behavior of
// IsWeeklyCron as described in the package README.md.
//
// Specification: Detects whether a cron expression runs weekly.
func TestSpec_ScheduleDetection_IsWeeklyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "weekly on Monday returns true",
			input:    "0 9 * * 1",
			expected: true,
		},
		{
			name:     "daily cron returns false",
			input:    "0 9 * * *",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWeeklyCron(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsWeeklyCron(%q) should detect weekly cron correctly", tt.input)
		})
	}
}

// TestSpec_PublicAPI_LevenshteinDistance validates the documented behavior of
// LevenshteinDistance as described in the package README.md.
//
// Specification: Computes edit distance between two strings.
func TestSpec_PublicAPI_LevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{
			name:     "identical strings have distance 0",
			a:        "hello",
			b:        "hello",
			expected: 0,
		},
		{
			name:     "one insertion has distance 1",
			a:        "cat",
			b:        "cats",
			expected: 1,
		},
		{
			name:     "one substitution has distance 1",
			a:        "cat",
			b:        "bat",
			expected: 1,
		},
		{
			name:     "empty string has distance equal to length of other",
			a:        "",
			b:        "abc",
			expected: 3,
		},
		{
			name:     "both empty have distance 0",
			a:        "",
			b:        "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LevenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.expected, result,
				"LevenshteinDistance(%q, %q) should compute correct edit distance", tt.a, tt.b)
		})
	}
}

// TestSpec_PublicAPI_IsValidGitHubIdentifier validates the documented behavior
// of IsValidGitHubIdentifier as described in the package README.md.
//
// Specification: Validates a GitHub username/org/repo name.
func TestSpec_PublicAPI_IsValidGitHubIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple lowercase name is valid",
			input:    "myrepo",
			expected: true,
		},
		{
			name:     "name with hyphens is valid",
			input:    "my-repo",
			expected: true,
		},
		{
			name:     "name with digits is valid",
			input:    "repo123",
			expected: true,
		},
		{
			name:     "empty string is invalid",
			input:    "",
			expected: false,
		},
		{
			name:     "name with slash is invalid",
			input:    "owner/repo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidGitHubIdentifier(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsValidGitHubIdentifier(%q) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_IsMCPType validates the documented behavior of IsMCPType
// as described in the package README.md.
//
// Specification: Validates an MCP transport type string.
// ValidMCPTypes contains "stdio", "http", "local".
func TestSpec_PublicAPI_IsMCPType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "stdio is a valid MCP type",
			input:    "stdio",
			expected: true,
		},
		{
			name:     "http is a valid MCP type",
			input:    "http",
			expected: true,
		},
		{
			name:     "local is a valid MCP type",
			input:    "local",
			expected: true,
		},
		{
			name:     "unknown type is invalid",
			input:    "grpc",
			expected: false,
		},
		{
			name:     "empty string is invalid",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMCPType(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsMCPType(%q) should validate against documented MCP transport types", tt.input)
		})
	}
}

// TestSpec_Constants_ValidMCPTypes validates the documented ValidMCPTypes
// variable values as described in the package README.md.
//
// Specification: ValidMCPTypes contains "stdio", "http", "local".
func TestSpec_Constants_ValidMCPTypes(t *testing.T) {
	assert.Contains(t, ValidMCPTypes, "stdio",
		"ValidMCPTypes should contain 'stdio' per specification")
	assert.Contains(t, ValidMCPTypes, "http",
		"ValidMCPTypes should contain 'http' per specification")
	assert.Contains(t, ValidMCPTypes, "local",
		"ValidMCPTypes should contain 'local' per specification")
	assert.Len(t, ValidMCPTypes, 3,
		"ValidMCPTypes should contain exactly the 3 documented types")
}

// TestSpec_PublicAPI_ParseImportDirective validates the documented behavior of
// ParseImportDirective as described in the package README.md.
//
// Specification: Parses a single @import or @include line.
func TestSpec_PublicAPI_ParseImportDirective(t *testing.T) {
	t.Run("@import directive is parsed correctly", func(t *testing.T) {
		line := "@import shared/base.md"
		result := ParseImportDirective(line)
		require.NotNil(t, result,
			"ParseImportDirective should return non-nil for valid @import line")
		assert.Equal(t, "shared/base.md", result.Path,
			"ParseImportDirective should extract the path from @import directive")
	})

	t.Run("@include directive is parsed correctly", func(t *testing.T) {
		line := "@include shared/tools.md"
		result := ParseImportDirective(line)
		require.NotNil(t, result,
			"ParseImportDirective should return non-nil for valid @include line")
		assert.Equal(t, "shared/tools.md", result.Path,
			"ParseImportDirective should extract the path from @include directive")
	})

	t.Run("non-directive line returns nil", func(t *testing.T) {
		line := "# Just a heading"
		result := ParseImportDirective(line)
		assert.Nil(t, result,
			"ParseImportDirective should return nil for non-directive lines")
	})
}

// TestSpec_PublicAPI_NewImportCache validates the documented behavior of
// NewImportCache as described in the package README.md.
//
// Specification: Creates a new import cache rooted at the repository.
// ImportCache is designed for use within a single goroutine per compilation run.
func TestSpec_PublicAPI_NewImportCache(t *testing.T) {
	t.Run("creates non-nil cache for given repo root", func(t *testing.T) {
		cache := NewImportCache("/path/to/repo")
		assert.NotNil(t, cache,
			"NewImportCache should return a non-nil ImportCache")
	})

	t.Run("creates separate cache instances", func(t *testing.T) {
		cache1 := NewImportCache("/repo/a")
		cache2 := NewImportCache("/repo/b")
		assert.NotSame(t, cache1, cache2,
			"NewImportCache should create separate cache instances for concurrent compilations")
	})
}
