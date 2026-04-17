package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var unknownToolsToMCPServersCodemodLog = logger.New("cli:codemod_tools_unknown_to_mcp_servers")

var knownBuiltInToolsForCodemod = map[string]bool{
	"github":            true,
	"playwright":        true,
	"agentic-workflows": true,
	"cache-memory":      true,
	"repo-memory":       true,
	"bash":              true,
	"edit":              true,
	"web-fetch":         true,
	"web-search":        true,
	"safety-prompt":     true,
	"timeout":           true,
	"startup-timeout":   true,
	"mount-as-clis":     true,
}

type toolsBlockEntry struct {
	key    string
	start  int
	end    int
	indent string
	lines  []string
}

// getToolsUnknownToMCPServersCodemod migrates unknown tools entries to mcp-servers.
func getToolsUnknownToMCPServersCodemod() Codemod {
	return Codemod{
		ID:           "tools-unknown-to-mcp-servers",
		Name:         "Migrate unknown tools entries to mcp-servers",
		Description:  "Moves unknown entries from 'tools' to 'mcp-servers'. Adds a command placeholder and wraps list values under 'tools'.",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok || len(toolsMap) == 0 {
				return content, false, nil
			}

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				toolsIndex, toolsEndIndex, entries, found := findTopLevelToolsEntries(lines)
				if !found || len(entries) == 0 {
					return lines, false
				}

				unknownEntries := make([]toolsBlockEntry, 0)
				unknownByStart := make(map[int]toolsBlockEntry)
				for _, entry := range entries {
					if knownBuiltInToolsForCodemod[entry.key] {
						continue
					}
					unknownEntries = append(unknownEntries, entry)
					unknownByStart[entry.start] = entry
				}

				if len(unknownEntries) == 0 {
					return lines, false
				}

				shouldRemoveToolsBlock := len(unknownEntries) == len(entries)
				removeLine := make([]bool, len(lines))
				if shouldRemoveToolsBlock {
					for i := toolsIndex; i < toolsEndIndex; i++ {
						removeLine[i] = true
					}
				} else {
					for _, entry := range unknownEntries {
						for i := entry.start; i < entry.end; i++ {
							removeLine[i] = true
						}
					}
				}

				filtered := make([]string, 0, len(lines))
				for i, line := range lines {
					if !removeLine[i] {
						filtered = append(filtered, line)
					}
				}

				mcpServerLines := make([]string, 0)
				for _, entry := range unknownEntries {
					toolValue := toolsMap[entry.key]
					mcpServerLines = append(mcpServerLines, buildMCPServerEntryLines(entry, toolValue)...)
				}

				updated := insertMCPServerEntries(filtered, mcpServerLines)
				unknownToolsToMCPServersCodemodLog.Printf("Migrated %d unknown tools entries to mcp-servers", len(unknownEntries))
				return updated, true
			})
		},
	}
}

func findTopLevelToolsEntries(lines []string) (int, int, []toolsBlockEntry, bool) {
	toolsIndex := -1
	toolsIndent := ""
	toolsEnd := len(lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "tools:") {
			toolsIndex = i
			toolsIndent = getIndentation(line)
			break
		}
	}

	if toolsIndex == -1 {
		return -1, -1, nil, false
	}

	for i := toolsIndex + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if hasExitedBlock(lines[i], toolsIndent) {
			toolsEnd = i
			break
		}
	}

	entryIndentLen := len(toolsIndent) + 2
	entries := make([]toolsBlockEntry, 0)

	for i := toolsIndex + 1; i < toolsEnd; {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}

		indent := getIndentation(lines[i])
		if len(indent) != entryIndentLen || strings.HasPrefix(trimmed, "-") || !strings.Contains(trimmed, ":") {
			i++
			continue
		}

		key := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[0])
		start := i
		i++

		for i < toolsEnd {
			nextTrimmed := strings.TrimSpace(lines[i])
			if nextTrimmed == "" || strings.HasPrefix(nextTrimmed, "#") {
				i++
				continue
			}

			nextIndent := getIndentation(lines[i])
			if len(nextIndent) == entryIndentLen && strings.Contains(nextTrimmed, ":") && !strings.HasPrefix(nextTrimmed, "-") {
				break
			}

			if len(nextIndent) < entryIndentLen {
				break
			}

			i++
		}

		entries = append(entries, toolsBlockEntry{
			key:    key,
			start:  start,
			end:    i,
			indent: indent,
			lines:  append([]string(nil), lines[start:i]...),
		})
	}

	return toolsIndex, toolsEnd, entries, true
}

func buildMCPServerEntryLines(entry toolsBlockEntry, toolValue any) []string {
	switch typed := toolValue.(type) {
	case []any:
		return buildMCPServerFromToolList(entry.indent, entry.key, typed)
	case []string:
		values := make([]any, 0, len(typed))
		for _, v := range typed {
			values = append(values, v)
		}
		return buildMCPServerFromToolList(entry.indent, entry.key, values)
	case map[string]any:
		block := append([]string(nil), entry.lines...)
		if !hasTopLevelCommandField(block) {
			insertion := entry.indent + "  command: \"...\" # TODO: fill in command"
			block = append(block[:1], append([]string{insertion}, block[1:]...)...)
		}
		return block
	default:
		return []string{
			fmt.Sprintf("%s%s:", entry.indent, entry.key),
			entry.indent + "  command: \"...\" # TODO: fill in command",
		}
	}
}

func buildMCPServerFromToolList(indent, name string, values []any) []string {
	lines := []string{
		fmt.Sprintf("%s%s:", indent, name),
		indent + "  command: \"...\" # TODO: fill in command",
		indent + "  tools:",
	}
	for _, value := range values {
		lines = append(lines, fmt.Sprintf("%s    - %s", indent, formatYAMLListItem(value)))
	}
	return lines
}

func formatYAMLListItem(value any) string {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return "\"\""
		}
		return typed
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func hasTopLevelCommandField(lines []string) bool {
	if len(lines) == 0 {
		return false
	}

	serverIndentLen := len(getIndentation(lines[0]))
	expectedIndentLen := serverIndentLen + 2

	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(getIndentation(lines[i])) == expectedIndentLen && strings.HasPrefix(trimmed, "command:") {
			return true
		}
	}

	return false
}

func insertMCPServerEntries(lines []string, entries []string) []string {
	mcpIndex := -1
	mcpIndent := ""
	mcpEnd := len(lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "mcp-servers:") {
			mcpIndex = i
			mcpIndent = getIndentation(line)
			break
		}
	}

	if mcpIndex == -1 {
		result := append([]string{}, lines...)
		result = append(result, "mcp-servers:")
		result = append(result, entries...)
		return result
	}

	for i := mcpIndex + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if hasExitedBlock(lines[i], mcpIndent) {
			mcpEnd = i
			break
		}
	}

	result := make([]string, 0, len(lines)+len(entries))
	result = append(result, lines[:mcpEnd]...)
	result = append(result, entries...)
	result = append(result, lines[mcpEnd:]...)
	return result
}
