package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPGatewayLogNamingConvention verifies that no files reference the old
// "mcp-gateway.log" naming convention. All gateway logs should use "gateway.log"
// to maintain consistency across the codebase.
//
// This test prevents regression and ensures the naming convention is enforced.
func TestMCPGatewayLogNamingConvention(t *testing.T) {
	// Get the repository root
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err, "Failed to get repository root")

	// Define directories to search
	searchDirs := []string{
		filepath.Join(repoRoot, "pkg"),
		filepath.Join(repoRoot, "actions"),
		filepath.Join(repoRoot, "cmd"),
	}

	// Extensions to check
	extensions := []string{".go", ".cjs", ".js", ".sh", ".md"}

	violations := []string{}

	for _, dir := range searchDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-matching extensions
			if info.IsDir() {
				return nil
			}

			// Skip this test file itself
			if strings.HasSuffix(path, "mcp_gateway_log_naming_test.go") {
				return nil
			}

			ext := filepath.Ext(path)
			isTargetExt := false
			for _, targetExt := range extensions {
				if ext == targetExt {
					isTargetExt = true
					break
				}
			}
			if !isTargetExt {
				return nil
			}

			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			contentStr := string(content)

			// Check for the incorrect naming pattern "mcp-gateway.log"
			// Allow it in comments that explicitly mention NOT to use it (documentation)
			if strings.Contains(contentStr, "mcp-gateway.log") {
				// Check if it's in a comment explaining the naming convention
				lines := strings.Split(contentStr, "\n")
				hasActualUsage := false
				for _, line := range lines {
					if strings.Contains(line, "mcp-gateway.log") {
						// Allow if the line is a comment explaining NOT to use it
						trimmed := strings.TrimSpace(line)
						isDocumentation := strings.HasPrefix(trimmed, "//") ||
							strings.HasPrefix(trimmed, "#") ||
							strings.HasPrefix(trimmed, "*") ||
							strings.Contains(strings.ToLower(line), "not") ||
							strings.Contains(strings.ToLower(line), "should") ||
							strings.Contains(strings.ToLower(line), "must")

						if !isDocumentation {
							// Found actual usage (not documentation)
							hasActualUsage = true
							break
						}
					}
				}

				if hasActualUsage {
					relPath, _ := filepath.Rel(repoRoot, path)
					violations = append(violations, relPath)
				}
			}

			return nil
		})
		require.NoError(t, err, "Failed to walk directory: %s", dir)
	}

	// Assert no violations found
	assert.Empty(t, violations,
		"Found files using incorrect 'mcp-gateway.log' naming convention. "+
			"Should use 'gateway.log' instead. Files: %v", violations)
}

// TestMCPGatewayLogPathConsistency verifies that all references to gateway
// log files use the correct path structure: /tmp/gh-aw/mcp-logs/gateway.log
func TestMCPGatewayLogPathConsistency(t *testing.T) {
	// Get the repository root
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err, "Failed to get repository root")

	// Key files that should reference gateway.log
	keyFiles := []string{
		filepath.Join(repoRoot, "actions/setup/js/parse_mcp_gateway_log.cjs"),
		filepath.Join(repoRoot, "actions/setup/sh/start_mcp_gateway.sh"),
	}

	for _, filePath := range keyFiles {
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "Failed to read file: %s", filePath)

		contentStr := string(content)
		relPath, _ := filepath.Rel(repoRoot, filePath)

		// Verify the file uses "gateway.log"
		assert.Contains(t, contentStr, "gateway.log",
			"File %s should reference 'gateway.log'", relPath)

		// Verify the file uses the correct path structure
		assert.Contains(t, contentStr, "/tmp/gh-aw/mcp-logs/",
			"File %s should use path '/tmp/gh-aw/mcp-logs/'", relPath)

		// Note: Files may reference "mcp-gateway.log" in comments explaining
		// what NOT to use. This is acceptable documentation.
		// We only check that the actual log file paths use "gateway.log"
	}
}
