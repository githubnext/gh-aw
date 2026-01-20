package cli

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var writePermissionsCodemodLog = logger.New("cli:codemod_permissions")

// getWritePermissionsCodemod creates a codemod for converting write permissions to read
func getWritePermissionsCodemod() Codemod {
	return Codemod{
		ID:           "write-permissions-to-read-migration",
		Name:         "Convert write permissions to read",
		Description:  "Converts all write permissions to read permissions to comply with the new security policy",
		IntroducedIn: "0.4.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if permissions exist
			permissionsValue, hasPermissions := frontmatter["permissions"]
			if !hasPermissions {
				return content, false, nil
			}

			// Check if any write permissions exist
			hasWritePermissions := false

			// Handle string shorthand (write-all, write)
			if strValue, ok := permissionsValue.(string); ok {
				if strValue == "write-all" || strValue == "write" {
					hasWritePermissions = true
				}
			}

			// Handle map format
			if mapValue, ok := permissionsValue.(map[string]any); ok {
				for _, value := range mapValue {
					if strValue, ok := value.(string); ok && strValue == "write" {
						hasWritePermissions = true
						break
					}
				}
			}

			if !hasWritePermissions {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			// Find and replace write permissions
			var modified bool
			var inPermissionsBlock bool
			var permissionsIndent string

			result := make([]string, len(frontmatterLines))

			for i, line := range frontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the permissions block
				if strings.HasPrefix(trimmedLine, "permissions:") {
					inPermissionsBlock = true
					permissionsIndent = getIndentation(line)

					// Handle shorthand on same line: "permissions: write-all" or "permissions: write"
					if strings.Contains(trimmedLine, ": write-all") {
						result[i] = strings.Replace(line, ": write-all", ": read-all", 1)
						modified = true
						writePermissionsCodemodLog.Printf("Replaced permissions: write-all with permissions: read-all on line %d", i+1)
						continue
					} else if strings.Contains(trimmedLine, ": write") && !strings.Contains(trimmedLine, "write-all") {
						result[i] = strings.Replace(line, ": write", ": read", 1)
						modified = true
						writePermissionsCodemodLog.Printf("Replaced permissions: write with permissions: read on line %d", i+1)
						continue
					}

					result[i] = line
					continue
				}

				// Check if we've left the permissions block
				if inPermissionsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					if hasExitedBlock(line, permissionsIndent) {
						inPermissionsBlock = false
					}
				}

				// Replace write with read if in permissions block
				if inPermissionsBlock && strings.Contains(trimmedLine, ": write") {
					// Preserve indentation and everything else
					// Extract the key, value, and any trailing comment
					parts := strings.SplitN(line, ":", 2)
					if len(parts) >= 2 {
						key := parts[0]
						valueAndComment := parts[1]

						// Replace "write" with "read" in the value part
						newValueAndComment := strings.Replace(valueAndComment, " write", " read", 1)
						result[i] = fmt.Sprintf("%s:%s", key, newValueAndComment)
						modified = true
						writePermissionsCodemodLog.Printf("Replaced write with read on line %d", i+1)
					} else {
						result[i] = line
					}
				} else {
					result[i] = line
				}
			}

			if !modified {
				return content, false, nil
			}

			// Reconstruct the content
			newContent := reconstructContent(result, markdown)
			writePermissionsCodemodLog.Print("Applied write permissions to read migration")
			return newContent, true, nil
		},
	}
}
