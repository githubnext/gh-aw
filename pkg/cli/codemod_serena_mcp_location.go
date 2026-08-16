package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var serenaMCPContainerLocationCodemodLog = logger.New("cli:codemod_serena_mcp_location")

// getSerenaMCPContainerLocationCodemod upgrades the legacy GitHub-hosted Serena MCP image
// to the project-maintained ghcr.io/oraios/serena container and the matching venv entrypoint.
func getSerenaMCPContainerLocationCodemod() Codemod {
	return Codemod{
		ID:           "serena-mcp-location-migration",
		Name:         "Update Serena MCP to the project-maintained container location",
		Description:  "Updates the legacy ghcr.io/github/serena-mcp-server image and the old 'serena' entrypoint to the current project-maintained ghcr.io/oraios/serena container and venv path.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasSerenaMCPServer(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return updateLegacySerenaMCPServerLocation(lines)
			})
			if applied {
				serenaMCPContainerLocationCodemodLog.Print("Updated Serena MCP server to the project-maintained container location")
			}
			return newContent, applied, err
		},
	}
}

func hasSerenaMCPServer(frontmatter map[string]any) bool {
	mcpServersValue, hasMCPServers := frontmatter["mcp-servers"]
	if !hasMCPServers {
		return false
	}
	mcpServersMap, ok := mcpServersValue.(map[string]any)
	if !ok {
		return false
	}
	for serverName := range mcpServersMap {
		if strings.EqualFold(serverName, "serena") {
			return true
		}
	}
	return false
}

func updateLegacySerenaMCPServerLocation(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var inMCPServers bool
	var mcpServersIndent string
	var inSerenaBlock bool
	var serenaIndent string
	var serenaBlockStart int
	var serenaBlockEnd int

	flushSerenaBlock := func() {
		if serenaBlockStart >= serenaBlockEnd {
			return
		}
		block := result[serenaBlockStart:serenaBlockEnd]
		updated, changed := rewriteSerenaMCPBlock(block)
		if changed {
			result = append(result[:serenaBlockStart], updated...)
			modified = true
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "mcp-servers:") {
			inMCPServers = true
			mcpServersIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inMCPServers && trimmed != "" && !strings.HasPrefix(trimmed, "#") && hasExitedBlock(line, mcpServersIndent) {
			inMCPServers = false
			if inSerenaBlock {
				flushSerenaBlock()
			}
			inSerenaBlock = false
		}

		if inMCPServers && strings.HasPrefix(trimmed, "serena:") {
			if inSerenaBlock {
				flushSerenaBlock()
			}
			inSerenaBlock = true
			serenaIndent = getIndentation(line)
			result = append(result, line)
			serenaBlockStart = len(result)
			serenaBlockEnd = len(result)
			continue
		}

		if inSerenaBlock && trimmed != "" && !strings.HasPrefix(trimmed, "#") && hasExitedBlock(line, serenaIndent) {
			flushSerenaBlock()
			inSerenaBlock = false
		}

		result = append(result, line)
		if inSerenaBlock {
			serenaBlockEnd = len(result)
		}
	}

	if inSerenaBlock {
		flushSerenaBlock()
	}

	return result, modified
}

// rewriteSerenaMCPBlock rewrites the container and entrypoint lines within a single
// Serena MCP server block. The entrypoint is only rewritten when the same block's
// container references the legacy image, regardless of the order the fields appear in.
func rewriteSerenaMCPBlock(block []string) ([]string, bool) {
	legacyContainer := false
	for _, line := range block {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "container:") {
			if _, changed := rewriteSerenaMCPContainerValue(line); changed {
				legacyContainer = true
			}
		}
	}

	var result []string
	var modified bool
	for _, line := range block {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "container:") {
			updated, changed := rewriteSerenaMCPContainerValue(line)
			if changed {
				result = append(result, updated)
				modified = true
				continue
			}
		}

		if legacyContainer && strings.HasPrefix(trimmed, "entrypoint:") {
			updated, changed := rewriteSerenaMCPEntrypointValue(line)
			if changed {
				result = append(result, updated)
				modified = true
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}

func rewriteSerenaMCPContainerValue(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "container:") {
		return line, false
	}

	valuePart := strings.TrimSpace(strings.TrimPrefix(trimmed, "container:"))
	if valuePart == "" {
		return line, false
	}

	value := strings.Trim(valuePart, "\"'")
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "ghcr.io/oraios/serena") || strings.HasPrefix(lowerValue, "ghcr.io/serena-ai/serena") {
		return line, false
	}
	if !strings.Contains(lowerValue, "serena") {
		return line, false
	}
	if !strings.Contains(lowerValue, "serena-mcp-server") && !strings.Contains(lowerValue, "ghcr.io/github/serena-mcp-server") && !strings.Contains(lowerValue, "ghcr.io/serena-ai/serena") {
		return line, false
	}

	return formatYAMLScalarValue(line, "container", "ghcr.io/oraios/serena:latest"), true
}

func rewriteSerenaMCPEntrypointValue(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "entrypoint:") {
		return line, false
	}

	valuePart := strings.TrimSpace(strings.TrimPrefix(trimmed, "entrypoint:"))
	value := strings.Trim(valuePart, "\"'")
	if value == "/workspaces/serena/.venv/bin/serena" {
		return line, false
	}
	lowerValue := strings.ToLower(value)
	if !strings.EqualFold(value, "serena") && !strings.HasSuffix(lowerValue, "/serena") && !strings.HasSuffix(lowerValue, "\\serena") {
		return line, false
	}

	return formatYAMLScalarValue(line, "entrypoint", "/workspaces/serena/.venv/bin/serena"), true
}

func formatYAMLScalarValue(line, key, value string) string {
	leadingWhitespace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	quote := "\""
	if strings.Contains(line, "'") && !strings.Contains(line, "\"") {
		quote = "'"
	}
	return leadingWhitespace + key + ": " + quote + value + quote
}
