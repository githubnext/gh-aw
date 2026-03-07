package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var permissionsReadCodemodLog = logger.New("cli:codemod_permissions_read")

// getPermissionsReadCodemod creates a codemod for converting invalid "read" and "write" shorthands
func getPermissionsReadCodemod() Codemod {
	return Codemod{
		ID:           "permissions-read-to-read-all",
		Name:         "Convert invalid permissions shorthand",
		Description:  "Converts 'permissions: read' to 'permissions: read-all' and 'permissions: write' to 'permissions: write-all' as per GitHub Actions spec",
		IntroducedIn: "0.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if permissions exist
			permissionsValue, hasPermissions := frontmatter["permissions"]
			if !hasPermissions {
				return content, false, nil
			}

			// Check if permissions uses invalid shorthand (read or write)
			hasInvalidShorthand := false
			if strValue, ok := permissionsValue.(string); ok {
				if strValue == "read" || strValue == "write" {
					hasInvalidShorthand = true
				}
			}

			if !hasInvalidShorthand {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				var modified bool
				result := make([]string, len(lines))
				for i, line := range lines {
					trimmedLine := strings.TrimSpace(line)

					// Check for permissions line with shorthand
					if strings.HasPrefix(trimmedLine, "permissions:") {
						// Handle shorthand on same line: "permissions: read" or "permissions: write"
						if strings.Contains(trimmedLine, ": read") && !strings.Contains(trimmedLine, "read-all") && !strings.Contains(trimmedLine, ": read\n") {
							// Make sure it's "permissions: read" and not "contents: read"
							if strings.TrimSpace(strings.Split(line, ":")[0]) == "permissions" {
								result[i] = strings.Replace(line, ": read", ": read-all", 1)
								modified = true
								permissionsReadCodemodLog.Printf("Replaced 'permissions: read' with 'permissions: read-all' on line %d", i+1)
								continue
							}
						} else if strings.Contains(trimmedLine, ": write") && !strings.Contains(trimmedLine, "write-all") {
							// Make sure it's "permissions: write" and not "contents: write"
							if strings.TrimSpace(strings.Split(line, ":")[0]) == "permissions" {
								result[i] = strings.Replace(line, ": write", ": write-all", 1)
								modified = true
								permissionsReadCodemodLog.Printf("Replaced 'permissions: write' with 'permissions: write-all' on line %d", i+1)
								continue
							}
						}
					}

					result[i] = line
				}
				return result, modified
			})
			if applied {
				permissionsReadCodemodLog.Print("Applied permissions read/write to read-all/write-all migration")
			}
			return newContent, applied, err
		},
	}
}
