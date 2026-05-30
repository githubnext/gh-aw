package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mountAsCLIsCodemodLog = logger.New("cli:codemod_mount_as_clis")

// getMountAsCLIsToCLIProxyCodemod creates a codemod that:
//  1. Renames tools.mount-as-clis to tools.cli-proxy.
//  2. Removes the deprecated features.mcp-cli flag.
func getMountAsCLIsToCLIProxyCodemod() Codemod {
	return Codemod{
		ID:           "mount-as-clis-to-cli-proxy",
		Name:         "Rename 'tools.mount-as-clis' to 'tools.cli-proxy' and remove 'features.mcp-cli'",
		Description:  "Renames the deprecated 'mount-as-clis:' field to 'cli-proxy:' inside the tools block, and removes the now-unnecessary 'features.mcp-cli: true' flag.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			hasMountAsCLIs := hasToolsMountAsCLIs(frontmatter)
			hasMCPCLIFeature := hasMCPCLIFeatureFlag(frontmatter)

			if !hasMountAsCLIs && !hasMCPCLIFeature {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result := lines
				modified := false

				if hasMountAsCLIs {
					result, modified = renameMountAsCLIsToCLIProxy(result)
				}

				if hasMCPCLIFeature {
					after, removedMCPCLI := removeFieldFromBlock(result, "mcp-cli", "features")
					if removedMCPCLI {
						result = after
						modified = true
					}
				}

				return result, modified
			})
			if applied {
				mountAsCLIsCodemodLog.Print("Renamed tools.mount-as-clis to tools.cli-proxy and removed features.mcp-cli")
			}
			return newContent, applied, err
		},
	}
}

// hasToolsMountAsCLIs returns true if tools.mount-as-clis exists.
func hasToolsMountAsCLIs(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasMountAsCLIs := toolsMap["mount-as-clis"]
	return hasMountAsCLIs
}

// hasMCPCLIFeatureFlag returns true if features.mcp-cli is set.
func hasMCPCLIFeatureFlag(frontmatter map[string]any) bool {
	featuresAny, hasFeatures := frontmatter["features"]
	if !hasFeatures {
		return false
	}
	featuresMap, ok := featuresAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasMCPCLI := featuresMap["mcp-cli"]
	return hasMCPCLI
}

// renameMountAsCLIsToCLIProxy renames 'mount-as-clis:' to 'cli-proxy:' inside the tools block.
func renameMountAsCLIsToCLIProxy(lines []string) ([]string, bool) {
	var result []string
	modified := false

	var inTools bool
	var toolsIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			result = append(result, line)
			continue
		}

		if !strings.HasPrefix(trimmed, "#") && inTools && hasExitedBlock(line, toolsIndent) {
			inTools = false
		}

		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inTools && strings.HasPrefix(trimmed, "mount-as-clis:") {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, toolsIndent) {
				newLine, replaced := findAndReplaceInLine(line, "mount-as-clis", "cli-proxy")
				if replaced {
					result = append(result, newLine)
					modified = true
					mountAsCLIsCodemodLog.Printf("Renamed 'mount-as-clis' to 'cli-proxy' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
