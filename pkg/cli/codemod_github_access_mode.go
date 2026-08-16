package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var githubAccessModeCodemodLog = logger.New("cli:codemod_github_access_mode")

// getGitHubModeHomogeneousEnumCodemod normalizes tools.github.mode / tools.github.type
// values to the homogeneous GitHub access-mode enum (cli / mcp-local / mcp-remote):
//
//	mode: gh-proxy -> mode: cli
//	mode: local    -> mode: mcp-local
//	mode: remote   -> mode: mcp-remote
//	type: local    -> mode: mcp-local   (folds legacy type into mode)
//	type: remote   -> mode: mcp-remote
func getGitHubModeHomogeneousEnumCodemod() Codemod {
	return Codemod{
		ID:           "github-mode-homogeneous-enum",
		Name:         "Normalize tools.github.mode/type to the homogeneous enum (cli/mcp-local/mcp-remote)",
		Description:  "Rewrites deprecated GitHub access values: mode gh-proxy -> cli, mode/type local -> mcp-local, mode/type remote -> mcp-remote (folding legacy type into mode).",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasLegacyGitHubModeOrType(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, normalizeGitHubModeTypeLines)
			if applied {
				githubAccessModeCodemodLog.Print("Normalized tools.github.mode/type to homogeneous enum")
			}
			return newContent, applied, err
		},
	}
}

// legacyGitHubModeValues maps deprecated tools.github.mode values to the homogeneous enum.
var legacyGitHubModeValues = map[string]string{
	"gh-proxy": "cli",
	"local":    "mcp-local",
	"remote":   "mcp-remote",
}

// legacyGitHubTypeValues maps deprecated tools.github.type values to the homogeneous mode enum.
var legacyGitHubTypeValues = map[string]string{
	"local":  "mcp-local",
	"remote": "mcp-remote",
}

func hasLegacyGitHubModeOrType(frontmatter map[string]any) bool {
	githubMap, ok := githubToolMap(frontmatter)
	if !ok {
		return false
	}
	if mode, ok := githubMap["mode"].(string); ok {
		if _, legacy := legacyGitHubModeValues[strings.ToLower(strings.TrimSpace(mode))]; legacy {
			return true
		}
	}
	if typeVal, ok := githubMap["type"].(string); ok {
		if _, legacy := legacyGitHubTypeValues[strings.ToLower(strings.TrimSpace(typeVal))]; legacy {
			return true
		}
	}
	return false
}

// githubToolMap returns the tools.github map from parsed frontmatter, if present.
func githubToolMap(frontmatter map[string]any) (map[string]any, bool) {
	toolsMap, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return nil, false
	}
	githubMap, ok := toolsMap["github"].(map[string]any)
	return githubMap, ok
}

// normalizeGitHubModeTypeLines rewrites tools.github.mode/type value lines within
// the tools.github block.
func normalizeGitHubModeTypeLines(lines []string) ([]string, bool) {
	var result []string
	modified := false
	hasMode := githubBlockHasMode(lines)

	var inTools, inGitHub bool
	var toolsIndent, githubIndent string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result = append(result, line)
			continue
		}

		if !strings.HasPrefix(trimmed, "#") {
			if inGitHub && hasExitedBlock(line, githubIndent) {
				inGitHub = false
			}
			if inTools && hasExitedBlock(line, toolsIndent) {
				inTools = false
				inGitHub = false
			}
		}

		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inTools && strings.HasPrefix(trimmed, "github:") && isDescendant(getIndentation(line), toolsIndent) {
			inGitHub = true
			githubIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inGitHub && isDescendant(getIndentation(line), githubIndent) {
			if strings.HasPrefix(trimmed, "mode:") {
				if newVal, ok := legacyGitHubModeValues[strings.ToLower(extractScalarValue(line))]; ok {
					result = append(result, rewriteKeyValueLine(line, "mode", newVal))
					modified = true
					continue
				}
			}
			if strings.HasPrefix(trimmed, "type:") {
				if newVal, ok := legacyGitHubTypeValues[strings.ToLower(extractScalarValue(line))]; ok {
					if hasMode {
						modified = true
						continue
					}
					result = append(result, rewriteKeyValueLine(line, "mode", newVal))
					modified = true
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}

func githubBlockHasMode(lines []string) bool {
	var inTools, inGitHub bool
	var toolsIndent, githubIndent string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if inGitHub && hasExitedBlock(line, githubIndent) {
			inGitHub = false
		}
		if inTools && hasExitedBlock(line, toolsIndent) {
			inTools = false
			inGitHub = false
		}
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			continue
		}
		if inTools && strings.HasPrefix(trimmed, "github:") && isDescendant(getIndentation(line), toolsIndent) {
			inGitHub = true
			githubIndent = getIndentation(line)
			continue
		}
		if inGitHub && isDescendant(getIndentation(line), githubIndent) && strings.HasPrefix(trimmed, "mode:") {
			return true
		}
	}
	return false
}

// getCliProxyToMCPModeCodemod migrates the top-level tools.cli-proxy boolean to the
// public tools.mcp-mode selector:
//
//	cli-proxy: true  -> mcp-mode: cli
//	cli-proxy: false -> mcp-mode: default
func getCliProxyToMCPModeCodemod() Codemod {
	return Codemod{
		ID:           "tools-cli-proxy-to-mcp-mode",
		Name:         "Rename 'tools.cli-proxy' to 'tools.mcp-mode'",
		Description:  "Migrates the deprecated tools.cli-proxy boolean to tools.mcp-mode: cli (true) or tools.mcp-mode: default (false).",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasTopLevelToolsCliProxy(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameCliProxyToMCPMode)
			if applied {
				githubAccessModeCodemodLog.Print("Renamed tools.cli-proxy to tools.mcp-mode")
			}
			return newContent, applied, err
		},
	}
}

func hasTopLevelToolsCliProxy(frontmatter map[string]any) bool {
	toolsMap, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return false
	}
	_, has := toolsMap["cli-proxy"]
	return has
}

// renameCliProxyToMCPMode rewrites the direct tools.cli-proxy child to tools.mcp-mode.
func renameCliProxyToMCPMode(lines []string) ([]string, bool) {
	var result []string
	modified := false

	var inTools bool
	var toolsIndent string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
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

		if inTools && strings.HasPrefix(trimmed, "cli-proxy:") && isDescendant(getIndentation(line), toolsIndent) {
			value := strings.ToLower(extractScalarValue(line))
			newVal := "cli"
			if value == "false" {
				newVal = "default"
			}
			result = append(result, rewriteKeyValueLine(line, "mcp-mode", newVal))
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

// extractScalarValue returns the scalar value of a "key: value" YAML line, stripped
// of surrounding quotes and any trailing comment.
func extractScalarValue(line string) string {
	_, value, found := strings.Cut(line, ":")
	if !found {
		return ""
	}
	if c := strings.Index(value, "#"); c >= 0 {
		value = value[:c]
	}
	value = strings.TrimSpace(value)
	return strings.Trim(value, `"'`)
}

// rewriteKeyValueLine rewrites a YAML "key: value" line with a new key and value,
// preserving indentation and any trailing comment.
func rewriteKeyValueLine(line, newKey, newValue string) string {
	indent := getIndentation(line)
	comment := ""
	if _, after, found := strings.Cut(line, ":"); found {
		if c := strings.Index(after, "#"); c >= 0 {
			comment = "  " + strings.TrimSpace(after[c:])
		}
	}
	return fmt.Sprintf("%s%s: %s%s", indent, newKey, newValue, comment)
}
