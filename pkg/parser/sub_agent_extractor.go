// Package parser — sub_agent_extractor.go
//
// This file provides inline sub-agent parsing for workflow markdown files.
//
// # Inline Sub-Agents
//
// A sub-agent is a secondary agent definition embedded directly in the same
// markdown file as the primary workflow. Each sub-agent has its own frontmatter
// block plus a prompt body. Sub-agents appear after the main workflow body and
// are separated from it (and from each other) by the special separator line:
//
//	<!-- @agent: name -->
//
// The separator must appear on its own line (optional surrounding whitespace is
// allowed). The agent name must start with a letter and contain only
// alphanumeric characters, hyphens, and underscores.
//
// # Example
//
//	---
//	engine: copilot
//	on:
//	  issues:
//	    types: [opened]
//	---
//	# Handle issue
//	Triage the issue and delegate work to sub-agents.
//
//	<!-- @agent: planner -->
//	---
//	engine: copilot
//	tools:
//	  github:
//	    toolsets: [issues, pull_requests]
//	---
//	You are a planning specialist.
//
//	<!-- @agent: executor -->
//	---
//	engine: copilot
//	tools:
//	  github:
//	    toolsets: [pull_requests]
//	---
//	You are an execution specialist.
//
// # Compilation Output
//
// During compilation the extracted sub-agents are written to the repository:
//   - Copilot engine: .github/agents/<name>.md
//   - Other engines: handled by the engine-specific compiler path
//
// # Wire-Up
//
// ExtractInlineSubAgents is called early in processToolsAndMarkdown so that
// the main workflow content (returned as mainMarkdown) is used for all
// subsequent prompt generation, while InlineSubAgents is populated on
// WorkflowData for the compilation output step.

package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// InlineSubAgent holds a single sub-agent definition extracted from a workflow
// markdown file's body using the <!-- @agent: name --> separator syntax.
type InlineSubAgent struct {
	// Name is the identifier taken from the separator line.
	// It is safe to use as a filename (alphanumeric, hyphens, underscores).
	Name string

	// Content is the raw text that follows the separator line up to the next
	// separator (or end of file). It typically includes a YAML frontmatter
	// block (---...---) followed by the sub-agent's prompt body, but the
	// format is not enforced — it varies by engine.
	Content string
}

// subAgentSeparatorRegex matches the inline sub-agent separator line.
//
// Format (anchored to line boundaries via (?m)):
//
//	<!-- @agent: name -->
//
// Rules:
//   - Optional horizontal whitespace before and after the comment
//   - Exactly one or more whitespace characters between "@agent:" and the name
//   - Agent name: starts with a letter, followed by alphanumeric / hyphen / underscore
//   - Optional horizontal whitespace after the name before "-->"
var subAgentSeparatorRegex = regexp.MustCompile(`(?m)^[ \t]*<!-- @agent:\s+([a-zA-Z][a-zA-Z0-9_-]*)\s*-->[ \t]*$`)

// ExtractInlineSubAgents splits markdown into the main workflow section and any
// inline sub-agent definitions.
//
// It scans the markdown body for <!-- @agent: name --> separator lines. Content
// before the first separator is returned as mainMarkdown (trimmed of trailing
// newlines). Each separator starts a new sub-agent whose content spans until
// the next separator or the end of the file.
//
// If no separators are found the original markdown is returned unchanged and
// agents is nil.
func ExtractInlineSubAgents(markdown string) (mainMarkdown string, agents []InlineSubAgent, err error) {
	allMatches := subAgentSeparatorRegex.FindAllStringSubmatchIndex(markdown, -1)

	if len(allMatches) == 0 {
		// No inline sub-agents — return the markdown unchanged.
		return markdown, nil, nil
	}

	// Validate that all agent names are unique.
	seen := make(map[string]struct{}, len(allMatches))
	for _, match := range allMatches {
		name := markdown[match[2]:match[3]]
		if _, exists := seen[name]; exists {
			return "", nil, fmt.Errorf("duplicate inline sub-agent name %q", name)
		}
		seen[name] = struct{}{}
	}

	// Main markdown is everything before the first separator, with trailing newlines stripped.
	mainMarkdown = strings.TrimRight(markdown[:allMatches[0][0]], "\n")

	agents = make([]InlineSubAgent, 0, len(allMatches))
	for i, match := range allMatches {
		// match[0]:match[1] = full separator line (including any leading/trailing spaces)
		// match[2]:match[3] = agent name capture group

		name := markdown[match[2]:match[3]]

		// Agent content starts immediately after the separator line.
		contentStart := match[1]
		// Skip the single newline that terminates the separator line (if present).
		if contentStart < len(markdown) && markdown[contentStart] == '\n' {
			contentStart++
		}

		// Agent content ends at the start of the next separator line, or at EOF.
		var contentEnd int
		if i+1 < len(allMatches) {
			contentEnd = allMatches[i+1][0]
		} else {
			contentEnd = len(markdown)
		}

		content := strings.TrimSpace(markdown[contentStart:contentEnd])

		agents = append(agents, InlineSubAgent{
			Name:    name,
			Content: content,
		})
	}

	return mainMarkdown, agents, nil
}
