package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var difcProxyCodemodLog = logger.New("cli:codemod_difc_proxy")

// getDIFCProxyToIntegrityProxyCodemod creates a codemod that migrates the deprecated
// 'features.difc-proxy' flag to the new 'tools.github.integrity-proxy' field.
//
// Migration rules:
//   - features.difc-proxy: true  → remove from features (proxy is enabled by default)
//   - features.difc-proxy: false → remove from features AND add
//     tools.github.integrity-proxy: false (to preserve the opt-out intent)
func getDIFCProxyToIntegrityProxyCodemod() Codemod {
	return Codemod{
		ID:           "features-difc-proxy-to-tools-github",
		Name:         "Migrate 'features.difc-proxy' to 'tools.github.integrity-proxy'",
		Description:  "Removes the deprecated 'features.difc-proxy' flag. The DIFC proxy is now enabled by default when guard policies are configured. If the flag was set to false, adds 'tools.github.integrity-proxy: false' to preserve the opt-out.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if features.difc-proxy exists
			flagValue, hasDIFCProxy := getDIFCProxyFlagValue(frontmatter)
			if !hasDIFCProxy {
				return content, false, nil
			}

			// Determine if we need to add integrity-proxy: false to tools.github
			addDisableFlag := !flagValue && hasToolsGithubMap(frontmatter)

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				// Step 1: remove features.difc-proxy
				result, modified := removeFieldFromBlock(lines, "difc-proxy", "features")
				if !modified {
					return lines, false
				}
				difcProxyCodemodLog.Print("Removed features.difc-proxy")

				// Step 2: add integrity-proxy: false to tools.github if needed
				if addDisableFlag {
					result = addIntegrityProxyFalseToToolsGitHub(result)
				}

				return result, true
			})
			if applied {
				if addDisableFlag {
					difcProxyCodemodLog.Print("Migrated features.difc-proxy: false → tools.github.integrity-proxy: false")
				} else {
					difcProxyCodemodLog.Print("Removed features.difc-proxy: true (proxy is now enabled by default)")
				}
			}
			return newContent, applied, err
		},
	}
}

// getDIFCProxyFlagValue returns the boolean value of features.difc-proxy and whether it exists.
func getDIFCProxyFlagValue(frontmatter map[string]any) (bool, bool) {
	featuresAny, hasFeatures := frontmatter["features"]
	if !hasFeatures {
		return false, false
	}
	featuresMap, ok := featuresAny.(map[string]any)
	if !ok {
		return false, false
	}
	val, hasFlag := featuresMap["difc-proxy"]
	if !hasFlag {
		return false, false
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal, true
	}
	// Non-boolean string values: treat non-empty as true
	if strVal, ok := val.(string); ok {
		return strVal != "", true
	}
	return false, true
}

// hasToolsGithubMap returns true if frontmatter has tools.github as a map (not just true/false).
func hasToolsGithubMap(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	githubAny, hasGitHub := toolsMap["github"]
	if !hasGitHub {
		return false
	}
	_, ok = githubAny.(map[string]any)
	return ok
}

// addIntegrityProxyFalseToToolsGitHub inserts 'integrity-proxy: false' inside the
// tools.github block. The line is inserted immediately after the 'github:' key line.
func addIntegrityProxyFalseToToolsGitHub(lines []string) []string {
	var result []string
	var inTools bool
	var toolsIndent string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track if we're in the tools block
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the tools block
		if inTools && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			if hasExitedBlock(line, toolsIndent) {
				inTools = false
			}
		}

		// Detect 'github:' inside tools and inject integrity-proxy: false after it
		if inTools && strings.HasPrefix(trimmed, "github:") {
			githubIndent := getIndentation(line)
			result = append(result, line)
			// Indent for github sub-fields is two more spaces than github:
			fieldIndent := githubIndent + "  "
			result = append(result, fieldIndent+"integrity-proxy: false")
			difcProxyCodemodLog.Printf("Added integrity-proxy: false to tools.github")
			inTools = false // done, no need to track further
			continue
		}

		result = append(result, line)
	}

	return result
}
