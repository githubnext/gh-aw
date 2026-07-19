package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

func validateGitHubToolsAgainstToolsetsCore(allowedTools []string, enabledToolsets []string) error {
	githubToolToToolsetLog.Printf("Validating GitHub tools against toolsets: allowed_tools=%d, enabled_toolsets=%d", len(allowedTools), len(enabledToolsets))
	if len(allowedTools) == 0 {
		githubToolToToolsetLog.Print("No tools to validate, skipping")
		// No specific tools restricted, validation not needed
		return nil
	}

	toolToToolsetMap, err := getGitHubToolToToolsetMap()
	if err != nil {
		return fmt.Errorf("failed to load GitHub tool-to-toolset mapping: %w", err)
	}

	enabledSet := makeEnabledToolsetSet(enabledToolsets)
	githubToolToToolsetLog.Printf("Enabled toolsets: %v", enabledToolsets)

	missingToolsets := make(map[string][]string) // toolset -> list of tools that need it
	var unknownTools []string
	var suggestions []string

	for _, tool := range allowedTools {
		// Skip wildcard - it means "allow all tools"
		if tool == "*" {
			continue
		}

		requiredToolset, exists := toolToToolsetMap[tool]
		if !exists {
			unknownTools, suggestions = recordUnknownGitHubTool(tool, toolToToolsetMap, unknownTools, suggestions)
			continue
		}

		if !setutil.Contains(enabledSet, requiredToolset) {
			githubToolToToolsetLog.Printf("Tool %s requires missing toolset: %s", tool, requiredToolset)
			missingToolsets[requiredToolset] = append(missingToolsets[requiredToolset], tool)
		}
	}

	// Report unknown tools with suggestions if any were found
	if len(unknownTools) > 0 {
		return buildUnknownGitHubToolError(unknownTools, suggestions, toolToToolsetMap)
	}

	if len(missingToolsets) > 0 {
		githubToolToToolsetLog.Printf("Validation failed: missing %d toolsets", len(missingToolsets))
		return NewGitHubToolsetValidationError(missingToolsets)
	}

	githubToolToToolsetLog.Print("Validation successful: all tools have required toolsets")
	return nil
}

func makeEnabledToolsetSet(enabledToolsets []string) map[string]struct{} {
	enabledSet := make(map[string]struct{})
	for _, toolset := range enabledToolsets {
		enabledSet[toolset] = struct{}{}
	}
	return enabledSet
}

func recordUnknownGitHubTool(tool string, toolToToolsetMap map[string]string, unknownTools, suggestions []string) ([]string, []string) {
	githubToolToToolsetLog.Printf("Tool %s not found in mapping, checking for typo", tool)
	validTools := sliceutil.SortedKeys(toolToToolsetMap)
	matches := parser.FindClosestMatches(tool, validTools, 1)
	unknownTools = append(unknownTools, tool)
	if len(matches) > 0 {
		githubToolToToolsetLog.Printf("Found suggestion for unknown tool %s: %s", tool, matches[0])
		suggestions = append(suggestions, fmt.Sprintf("%s → %s", tool, matches[0]))
	} else {
		githubToolToToolsetLog.Printf("No suggestion found for unknown tool: %s", tool)
	}
	return unknownTools, suggestions
}

func buildUnknownGitHubToolError(unknownTools, suggestions []string, toolToToolsetMap map[string]string) error {
	githubToolToToolsetLog.Printf("Found %d unknown tools", len(unknownTools))
	var errMsg strings.Builder
	fmt.Fprintf(&errMsg, "Unknown GitHub tool(s): %s\n\n", stringutil.FormatList(unknownTools))
	if len(suggestions) > 0 {
		errMsg.WriteString("Did you mean:\n")
		for _, s := range suggestions {
			fmt.Fprintf(&errMsg, "  %s\n", s)
		}
		errMsg.WriteString("\n")
	}
	validTools := sliceutil.SortedKeys(toolToToolsetMap)
	exampleCount := min(10, len(validTools))
	fmt.Fprintf(&errMsg, "Valid GitHub tools include: %s\n\n", stringutil.FormatList(validTools[:exampleCount]))
	errMsg.WriteString("See all tools: https://github.com/github/gh-aw/blob/main/pkg/workflow/data/github_tool_to_toolset.json")
	return errors.New(errMsg.String())
}
