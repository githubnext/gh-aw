// Package parser — sub_agent_extractor.go
//
// This file provides inline sub-agent parsing for workflow markdown files.
//
// # Inline Sub-Agents
//
// A sub-agent is a secondary agent definition embedded directly in the same
// markdown file as the primary workflow. Each sub-agent has its own frontmatter
// block plus a prompt body. Sub-agents appear after the main workflow body and
// are delimited by level-2 Markdown headings:
//
//	## agent: `name`        ← opens a sub-agent block
//
// An agent block ends at the next level-2 Markdown heading (## ...) or end
// of file. The name must be a lowercase identifier (letters, digits, hyphens,
// underscores; must start with a letter).
//
// Both the agent marker and any subsequent H2 section heading render as visible
// section headings in any Markdown preview (GitHub, VS Code, etc.).
//
// # Supported Frontmatter Fields
//
// Only the following fields are valid in a sub-agent frontmatter block.
// Any other field is stripped at runtime with a warning.
//
//   - description: Human-readable description of the sub-agent's role.
//   - model: AI model to use.  Default is "inherited" (uses the parent
//     workflow's model when not set).
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
//	## agent: `planner`
//	---
//	model: claude-haiku-4.5
//	description: Plans the work for the issue
//	---
//	You are a planning specialist.
//
//	## agent: `executor`
//	---
//	description: Executes the plan
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
// subsequent prompt generation, while the sub-agent files are written at
// runtime by interpolate_prompt.cjs after runtime imports are inlined.

package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// GetEngineSubAgentDir returns the relative directory (from repo root / tmp base) used
// to store inline sub-agent files for a given engine.
//
// Each engine has a dedicated config directory:
//
//	claude   → .claude/agents
//	codex    → .codex/agents
//	gemini   → .gemini/agents
//	others   → .agents/agents  (Copilot default)
func GetEngineSubAgentDir(engineID string) string {
	switch strings.ToLower(engineID) {
	case "claude":
		return ".claude/agents"
	case "codex":
		return ".codex/agents"
	case "gemini":
		return ".gemini/agents"
	default:
		return ".agents/agents"
	}
}

// GetEngineSubAgentExt returns the file extension used for inline sub-agent files
// for a given engine.
//
//	claude / codex / gemini → .md
//	others                  → .agent.md  (Copilot default)
func GetEngineSubAgentExt(engineID string) string {
	switch strings.ToLower(engineID) {
	case "claude", "codex", "gemini":
		return ".md"
	default:
		return ".agent.md"
	}
}

// InlineSubAgent holds a single sub-agent definition extracted from a workflow
// markdown file's body using the ## agent: `name` syntax.
type InlineSubAgent struct {
	// Name is the identifier taken from the ## agent: `name` line.
	// It is lowercase and safe to use as a filename.
	Name string

	// Content is the raw text between the ## agent: `name` line and the next
	// level-2 Markdown heading (## ...) or EOF. It typically includes a YAML
	// frontmatter block (---...---) followed by the sub-agent's prompt body,
	// but the format is not enforced — it varies by engine.
	Content string
}

// subAgentSeparatorRegex matches the inline sub-agent start marker line.
//
// Format (anchored to line boundaries via (?m)):
//
// ## agent: `name`
//
// Rules:
//   - A level-2 Markdown heading (##)
//   - One or more whitespace characters between "##" and "agent:"
//   - One or more whitespace characters between "agent:" and the backtick-enclosed name
//   - Agent name: starts with a lowercase letter, followed by lowercase letters,
//     digits, hyphens, or underscores
//   - Optional trailing whitespace
var subAgentSeparatorRegex = regexp.MustCompile("(?m)^##[ \t]+agent:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$")

// h2HeadingRegex matches the start of any level-2 Markdown heading (## space/tab).
// An agent block extends from its start marker to the next H2 heading or EOF.
var h2HeadingRegex = regexp.MustCompile(`(?m)^##[ \t]`)

// ExtractInlineSubAgents splits markdown into the main workflow section and any
// inline sub-agent definitions.
//
// It scans the markdown body for ## agent: `name` start markers. Content before
// the first start marker is returned as mainMarkdown (trimmed of trailing
// newlines). Each start marker opens a sub-agent whose content spans to the
// next level-2 Markdown heading (## ...) or EOF — whichever comes first.
//
// If no start markers are found the original markdown is returned unchanged and
// agents is nil.
func ExtractInlineSubAgents(markdown string) (mainMarkdown string, agents []InlineSubAgent, err error) {
	// Find all start markers (returned in document order by FindAllStringSubmatchIndex).
	allStarts := subAgentSeparatorRegex.FindAllStringSubmatchIndex(markdown, -1)
	if len(allStarts) == 0 {
		// No start markers — return unchanged.
		return markdown, nil, nil
	}

	// Validate that all agent names are unique.
	seen := make(map[string]struct{})
	for _, m := range allStarts {
		name := markdown[m[2]:m[3]]
		if _, exists := seen[name]; exists {
			return "", nil, fmt.Errorf("duplicate inline sub-agent name %q", name)
		}
		seen[name] = struct{}{}
	}

	// Main markdown is everything before the first start marker.
	mainMarkdown = strings.TrimRight(markdown[:allStarts[0][0]], "\n")

	// Collect the byte offset of every H2 heading in the document.
	// These positions are used to find the boundary where each agent block ends.
	var h2Positions []int
	for _, m := range h2HeadingRegex.FindAllStringIndex(markdown, -1) {
		h2Positions = append(h2Positions, m[0])
	}

	// nextH2After returns the byte offset of the first H2 heading at or after
	// 'offset', or len(markdown) when none exists.
	nextH2After := func(offset int) int {
		for _, pos := range h2Positions {
			if pos >= offset {
				return pos
			}
		}
		return len(markdown)
	}

	// Extract each agent block.
	for _, m := range allStarts {
		name := markdown[m[2]:m[3]]

		// Content starts on the line after the start marker.
		lineEnd := m[1]
		if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
			lineEnd++
		}

		// Content ends at the next H2 heading after the start marker line, or EOF.
		contentEnd := nextH2After(lineEnd)

		content := strings.TrimSpace(markdown[lineEnd:contentEnd])
		agents = append(agents, InlineSubAgent{Name: name, Content: content})
	}

	return mainMarkdown, agents, nil
}
