package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// extractStrictMode extracts strict mode setting from frontmatter
func (c *Compiler) extractStrictMode(frontmatter map[string]any) bool {
	if strict, exists := frontmatter["strict"]; exists {
		if strictBool, ok := strict.(bool); ok {
			return strictBool
		}
	}
	return false // Default to false if not specified or not a boolean
}

// validatePermissionsInStrictMode checks permissions in strict mode and warns about write permissions
func (c *Compiler) validatePermissionsInStrictMode(permissions string) {
	if permissions == "" {
		return
	}
	hasWritePermissions := strings.Contains(permissions, "write")
	if hasWritePermissions {
		fmt.Println(console.FormatWarningMessage("Strict mode: Found 'write' permissions. Consider using 'read' permissions only for better security."))
	}
}
