// Package parser — sub_agent_extractor.go
//
// This file provides inline sub-agent parsing for workflow markdown files.
//
// # Inline Sub-Agents
//
// A sub-agent is a secondary agent definition embedded directly in the same
// markdown file as the primary workflow. Each sub-agent has its own frontmatter
// block plus a prompt body. Sub-agents appear after the main workflow body and
// are delimited by a pair of level-2 Markdown headings:
//
//	## agent: name        ← opens a sub-agent block
//	## end: name          ← optionally closes it (same name required)
//
// Both markers render as visible section headings in any Markdown preview
// (GitHub, VS Code, etc.) while remaining clearly distinguishable from regular
// document headings. The agent name must start with a letter and contain only
// alphanumeric characters, hyphens, and underscores.
//
// The end marker is optional: if absent, the block extends to the next
// ## agent: line or end of file. Using ## end: name is recommended when other
// content may be inserted after the agent block (e.g. auto-generated sections),
// so that inserted text is not accidentally captured as part of the agent.
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
//	## agent: planner
//	---
//	engine: copilot
//	tools:
//	  github:
//	    toolsets: [issues, pull_requests]
//	---
//	You are a planning specialist.
//	## end: planner
//
//	## agent: executor
//	---
//	engine: copilot
//	tools:
//	  github:
//	    toolsets: [pull_requests]
//	---
//	You are an execution specialist.
//	## end: executor
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
	"sort"
	"strings"
)

// InlineSubAgent holds a single sub-agent definition extracted from a workflow
// markdown file's body using the ## agent: name / ## end: name syntax.
type InlineSubAgent struct {
	// Name is the identifier taken from the ## agent: name line.
	// It is safe to use as a filename (alphanumeric, hyphens, underscores).
	Name string

	// Content is the raw text between the ## agent: name and ## end: name lines
	// (or the next ## agent: line / EOF when no end marker is present). It
	// typically includes a YAML frontmatter block (---...---) followed by the
	// sub-agent's prompt body, but the format is not enforced — it varies by engine.
	Content string
}

// subAgentSeparatorRegex matches the inline sub-agent start marker line.
//
// Format (anchored to line boundaries via (?m)):
//
//	## agent: name
//
// Rules:
//   - A level-2 Markdown heading (##)
//   - One or more whitespace characters between "##" and "agent:"
//   - One or more whitespace characters between "agent:" and the name
//   - Agent name: starts with a letter, followed by alphanumeric / hyphen / underscore
//   - Optional trailing whitespace
var subAgentSeparatorRegex = regexp.MustCompile(`(?m)^##[ \t]+agent:[ \t]+([a-zA-Z][a-zA-Z0-9_-]*)[ \t]*$`)

// subAgentEndRegex matches the optional inline sub-agent end marker line.
//
// Format (anchored to line boundaries via (?m)):
//
//	## end: name
//
// Rules mirror subAgentSeparatorRegex; the name must match the corresponding
// ## agent: name line for the marker to take effect.
var subAgentEndRegex = regexp.MustCompile(`(?m)^##[ \t]+end:[ \t]+([a-zA-Z][a-zA-Z0-9_-]*)[ \t]*$`)

// markerKind distinguishes start markers (## agent: name) from end markers
// (## end: name) during event-driven parsing.
type markerKind int

const (
	startMarkerKind markerKind = iota
	endMarkerKind
)

// agentMarker represents a single parsed marker line.
type agentMarker struct {
	kind      markerKind
	name      string
	lineStart int // byte offset of the first character of the marker line
	lineEnd   int // byte offset of the first character after the marker line (past '\n')
}

// ExtractInlineSubAgents splits markdown into the main workflow section and any
// inline sub-agent definitions.
//
// It scans the markdown body for ## agent: name start markers and optional
// ## end: name end markers. Content before the first start marker is returned
// as mainMarkdown (trimmed of trailing newlines). Each start marker opens a
// sub-agent whose content spans to the matching ## end: name line, the next
// ## agent: line, or EOF — whichever comes first.
//
// An ## end: name line whose name does not match the currently open agent is
// silently treated as plain text (included in the current agent's content).
//
// If no start markers are found the original markdown is returned unchanged and
// agents is nil.
func ExtractInlineSubAgents(markdown string) (mainMarkdown string, agents []InlineSubAgent, err error) {
	// Collect all start and end markers in document order.
	var markers []agentMarker

	for _, m := range subAgentSeparatorRegex.FindAllStringSubmatchIndex(markdown, -1) {
		lineEnd := m[1]
		if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
			lineEnd++
		}
		markers = append(markers, agentMarker{
			kind:      startMarkerKind,
			name:      markdown[m[2]:m[3]],
			lineStart: m[0],
			lineEnd:   lineEnd,
		})
	}

	for _, m := range subAgentEndRegex.FindAllStringSubmatchIndex(markdown, -1) {
		lineEnd := m[1]
		if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
			lineEnd++
		}
		markers = append(markers, agentMarker{
			kind:      endMarkerKind,
			name:      markdown[m[2]:m[3]],
			lineStart: m[0],
			lineEnd:   lineEnd,
		})
	}

	// Sort all markers by their position in the document.
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].lineStart < markers[j].lineStart
	})

	// Find the index of the first start marker.
	firstStart := -1
	for i, m := range markers {
		if m.kind == startMarkerKind {
			firstStart = i
			break
		}
	}

	if firstStart == -1 {
		// No start markers — return unchanged.
		return markdown, nil, nil
	}

	// Validate that all start-marker names are unique.
	seen := make(map[string]struct{})
	for _, m := range markers {
		if m.kind == startMarkerKind {
			if _, exists := seen[m.name]; exists {
				return "", nil, fmt.Errorf("duplicate inline sub-agent name %q", m.name)
			}
			seen[m.name] = struct{}{}
		}
	}

	// Main markdown is everything before the first start marker.
	mainMarkdown = strings.TrimRight(markdown[:markers[firstStart].lineStart], "\n")

	// Walk markers from the first start marker, tracking the open agent.
	var currentName string
	var contentStart int

	for i := firstStart; i < len(markers); i++ {
		m := markers[i]

		switch m.kind {
		case startMarkerKind:
			// Close the currently open agent (if any) at this line's start.
			if currentName != "" {
				content := strings.TrimSpace(markdown[contentStart:m.lineStart])
				agents = append(agents, InlineSubAgent{Name: currentName, Content: content})
			}
			// Open the new agent; its content starts on the line after this marker.
			currentName = m.name
			contentStart = m.lineEnd

		case endMarkerKind:
			// Only take effect when the name matches the currently open agent.
			if currentName == m.name {
				content := strings.TrimSpace(markdown[contentStart:m.lineStart])
				agents = append(agents, InlineSubAgent{Name: currentName, Content: content})
				currentName = ""
			}
			// If the name does not match (or no agent is open), the line is plain
			// text that is already included in contentStart..next-event range; no
			// action needed here.
		}
	}

	// Close any agent that was still open at EOF.
	if currentName != "" {
		content := strings.TrimSpace(markdown[contentStart:])
		agents = append(agents, InlineSubAgent{Name: currentName, Content: content})
	}

	return mainMarkdown, agents, nil
}
