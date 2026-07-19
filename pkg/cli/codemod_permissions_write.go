package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var writePermissionsCodemodLog = logger.New("cli:codemod_permissions")

// writeOnlyPermissions are permission scopes that only accept "write" or "none" as valid values.
// These must never be converted to "read" since "read" is not a valid value for them.
var writeOnlyPermissions = map[string]bool{
	"id-token":         true, // OIDC token: only write or none
	"copilot-requests": true, // Copilot authentication token: only write or none
}

// getMigrateWritePermissionsToReadCodemod creates a codemod for converting write permissions to read
func getMigrateWritePermissionsToReadCodemod() Codemod {
	return Codemod{
		ID:           "write-permissions-to-read-migration",
		Name:         "Convert write permissions to read",
		Description:  "Converts all write permissions to read permissions to comply with the new security policy",
		IntroducedIn: "0.4.0",
		Apply:        getMigrateWritePermissionsToReadCodemodApply,
	}
}

func getMigrateWritePermissionsToReadCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	// Check if permissions exist
	permissionsValue, hasPermissions := frontmatter["permissions"]
	if !hasPermissions {
		return content, false, nil
	}
	if !getMigrateWritePermissionsToReadCodemodHasWritePermissions(permissionsValue) {
		return content, false, nil
	}

	newContent, applied, err := applyFrontmatterLineTransform(content, getMigrateWritePermissionsToReadCodemodLines)
	if applied {
		writePermissionsCodemodLog.Print("Applied write permissions to read migration")
	}
	return newContent, applied, err
}

func getMigrateWritePermissionsToReadCodemodHasWritePermissions(permissionsValue any) bool {
	// Handle string shorthand (write-all, write)
	if strValue, ok := permissionsValue.(string); ok {
		return strValue == "write-all" || strValue == "write"
	}

	// Handle map format
	if mapValue, ok := permissionsValue.(map[string]any); ok {
		for key, value := range mapValue {
			// Skip write-only permissions (e.g. id-token, copilot-requests) since
			// "read" is not a valid value for them — they only accept "write" or "none"
			if writeOnlyPermissions[key] {
				continue
			}
			if strValue, ok := value.(string); ok && strValue == "write" {
				return true
			}
		}
	}
	return false
}

func getMigrateWritePermissionsToReadCodemodLines(lines []string) ([]string, bool) {
	var modified bool
	var inPermissionsBlock bool
	var permissionsIndent string
	result := make([]string, len(lines))
	for i, line := range lines {
		updatedLine, changed := getMigrateWritePermissionsToReadCodemodLine(line, i, &inPermissionsBlock, &permissionsIndent)
		if changed {
			modified = true
		}
		result[i] = updatedLine
	}
	return result, modified
}

func getMigrateWritePermissionsToReadCodemodLine(line string, index int, inPermissionsBlock *bool, permissionsIndent *string) (string, bool) {
	trimmedLine := strings.TrimSpace(line)

	// Track if we're in the permissions block
	if strings.HasPrefix(trimmedLine, "permissions:") {
		*inPermissionsBlock = true
		*permissionsIndent = getIndentation(line)
		return getMigrateWritePermissionsToReadCodemodPermissionsLine(line, trimmedLine, index)
	}

	// Check if we've left the permissions block
	if *inPermissionsBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
		if hasExitedBlock(line, *permissionsIndent) {
			*inPermissionsBlock = false
		}
	}

	if *inPermissionsBlock && strings.Contains(trimmedLine, ": write") {
		return getMigrateWritePermissionsToReadCodemodNestedLine(line, index)
	}
	return line, false
}

func getMigrateWritePermissionsToReadCodemodPermissionsLine(line string, trimmedLine string, index int) (string, bool) {
	// Handle shorthand on same line: "permissions: write-all" or "permissions: write"
	if strings.Contains(trimmedLine, ": write-all") {
		writePermissionsCodemodLog.Printf("Replaced permissions: write-all with permissions: read-all on line %d", index+1)
		return strings.Replace(line, ": write-all", ": read-all", 1), true
	} else if strings.Contains(trimmedLine, ": write") && !strings.Contains(trimmedLine, "write-all") {
		writePermissionsCodemodLog.Printf("Replaced permissions: write with permissions: read on line %d", index+1)
		return strings.Replace(line, ": write", ": read", 1), true
	}
	return line, false
}

func getMigrateWritePermissionsToReadCodemodNestedLine(line string, index int) (string, bool) {
	// Preserve indentation and everything else
	// Extract the key, value, and any trailing comment
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return line, false
	}
	key := parts[0]
	permKey := strings.TrimSpace(key)
	valueAndComment := parts[1]

	// Skip write-only permissions (e.g. id-token, copilot-requests) since
	// "read" is not a valid value for them — they only accept "write" or "none"
	if writeOnlyPermissions[permKey] {
		writePermissionsCodemodLog.Printf("Skipping write-only permission %q on line %d", permKey, index+1)
		return line, false
	}

	// Replace "write" with "read" in the value part
	newValueAndComment := strings.Replace(valueAndComment, " write", " read", 1)
	writePermissionsCodemodLog.Printf("Replaced write with read on line %d", index+1)
	return fmt.Sprintf("%s:%s", key, newValueAndComment), true
}
