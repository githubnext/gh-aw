// This file provides focused MCP mount syntax validation helpers.
//
// # MCP Mount Validation
//
// validateMCPMountsSyntax() validates mount strings for containerized stdio MCP
// servers. Required format (MCP Gateway v0.1.5+):
//
//	source:destination:mode
//
// where mode is either "ro" or "rw".

package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
)

var mcpMountValidationLog = newValidationLogger("mcp_mount")

// validateMCPMountsSyntax validates that mount strings in a custom MCP server config
// follow the correct syntax required by MCP Gateway v0.1.5+.
// Expected format: "source:destination:mode" where mode is either "ro" or "rw".
func validateMCPMountsSyntax(toolName string, mountsRaw any) error {
	var mounts []string

	switch v := mountsRaw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				mounts = append(mounts, s)
			}
		}
	case []string:
		mounts = v
	default:
		mcpMountValidationLog.Printf("Invalid mounts type for tool %q: expected array, got %T", toolName, mountsRaw)
		return fmt.Errorf("tool '%s' mcp configuration 'mounts' must be an array of strings.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"\n\nSee: %s", toolName, toolName, constants.DocsToolsURL)
	}

	mcpMountValidationLog.Printf("Validating %d mount(s) for tool %q", len(mounts), toolName)
	for i, mount := range mounts {
		source, dest, mode, err := validateMountStringFormat(mount)
		if err != nil {
			if source == "" && dest == "" && mode == "" {
				mcpMountValidationLog.Printf("Mount[%d] format error for tool %q: %q", i, toolName, mount)
				return fmt.Errorf("tool '%s' mcp configuration mounts[%d] must follow 'source:destination:mode' format, got: %q.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"\n\nSee: %s", toolName, i, mount, toolName, constants.DocsToolsURL)
			}
			mcpMountValidationLog.Printf("Mount[%d] invalid mode for tool %q: got %q", i, toolName, mode)
			return fmt.Errorf("tool '%s' mcp configuration mounts[%d] mode must be 'ro' or 'rw', got: %q.\n\nExample:\ntools:\n  %s:\n    container: \"my-registry/my-tool\"\n    mounts:\n      - \"/host/path:/container/path:ro\"  # read-only\n      - \"/host/path:/container/path:rw\"  # read-write\n\nSee: %s", toolName, i, mode, toolName, constants.DocsToolsURL)
		}
		mcpMountValidationLog.Printf("Mount[%d] valid for tool %q: source=%s, dest=%s, mode=%s", i, toolName, source, dest, mode)
	}

	return nil
}
