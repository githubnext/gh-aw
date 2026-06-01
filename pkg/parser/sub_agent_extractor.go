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
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var subAgentLog = logger.New("parser:sub_agent_extractor")

// validSubAgentFrontmatterFields is the set of permitted keys in a sub-agent
// frontmatter block. Any key not in this set will produce a warning when
// ValidateInlineSubAgentsFrontmatter is called.
var validSubAgentFrontmatterFields = map[string]bool{
	"description": true,
	"model":       true,
}

// ValidateInlineSubAgentsFrontmatter performs best-effort frontmatter validation
// on every inline sub-agent section found in markdown.
//
// markdown should be the full content of a workflow file (including any
// top-level frontmatter block). The function strips the top-level frontmatter
// before scanning for ## agent: `name` markers so that the file-level
// frontmatter is not mistaken for sub-agent content.
//
// For each detected sub-agent the function:
//  1. Attempts to parse its embedded frontmatter block (--- … ---).
//  2. Reports unknown fields (anything other than "description" or "model").
//
// All issues are returned as human-readable warning strings. Callers must not
// fail compilation based on these messages — they are advisory only (best effort).
// If no sub-agents are found, or if no issues are detected, nil is returned.
func ValidateInlineSubAgentsFrontmatter(markdown string) []string {
	// Strip the top-level frontmatter to obtain only the markdown body.
	var body string
	if parsed, err := ExtractFrontmatterFromContent(markdown); err == nil {
		body = parsed.Markdown
	} else {
		body = markdown
	}
	return ValidateInlineSubAgentsInBody(body)
}

// ValidateInlineSubAgentsInBody performs best-effort frontmatter validation on
// inline sub-agent sections found in an already-stripped markdown body.
// Unlike ValidateInlineSubAgentsFrontmatter, it does not strip a top-level
// frontmatter block, making it suitable for callers that have already parsed
// the file and hold the markdown body separately.
//
// All issues are returned as human-readable warning strings (best-effort,
// never abort compilation). If no sub-agents are found or no issues are
// detected, nil is returned.
func ValidateInlineSubAgentsInBody(body string) []string {
	_, subAgents, err := ExtractInlineSubAgents(body)
	if err != nil {
		// Surface extraction errors (e.g. duplicate agent names) as a warning
		// rather than silently skipping validation.
		return []string{fmt.Sprintf("could not extract inline sub-agents: %v", err)}
	}
	if len(subAgents) == 0 {
		return nil
	}

	var warnings []string
	for _, agent := range subAgents {
		warnings = append(warnings, validateSubAgentFrontmatterFields(agent)...)
	}
	return warnings
}

// validateSubAgentFrontmatterFields parses the frontmatter block embedded in a
// single InlineSubAgent.Content and returns warning messages for any unknown fields.
func validateSubAgentFrontmatterFields(agent InlineSubAgent) []string {
	parsed, err := ExtractFrontmatterFromContent(agent.Content)
	if err != nil {
		return []string{fmt.Sprintf("sub-agent %q: could not parse frontmatter: %v", agent.Name, err)}
	}
	if len(parsed.Frontmatter) == 0 {
		return nil
	}

	var unknown []string
	for key := range parsed.Frontmatter {
		if !validSubAgentFrontmatterFields[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}

	sort.Strings(unknown) // deterministic order
	return []string{fmt.Sprintf(
		"sub-agent %q: unknown frontmatter field(s): %s (valid fields: description, model)",
		agent.Name, strings.Join(unknown, ", "),
	)}
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
	subAgentLog.Printf("Extracting inline sub-agents from markdown (length: %d)", len(markdown))
	allStarts := subAgentSeparatorRegex.FindAllStringSubmatchIndex(markdown, -1)
	if len(allStarts) == 0 {
		subAgentLog.Print("No inline sub-agent markers found")
		return markdown, nil, nil
	}

	subAgentLog.Printf("Found %d inline sub-agent marker(s)", len(allStarts))
	if err := validateUniqueSubAgentNames(markdown, allStarts); err != nil {
		return "", nil, err
	}

	mainMarkdown = strings.TrimRight(markdown[:allStarts[0][0]], "\n")
	h2Positions := collectH2Positions(markdown)
	for _, m := range allStarts {
		name, content := extractInlineSection(markdown, m, h2Positions)
		subAgentLog.Printf("Extracted sub-agent %q (content length: %d)", name, len(content))
		agents = append(agents, InlineSubAgent{Name: name, Content: content})
	}

	subAgentLog.Printf("Extraction complete: %d sub-agent(s), main markdown length: %d", len(agents), len(mainMarkdown))
	return mainMarkdown, agents, nil
}

func validateUniqueSubAgentNames(markdown string, allStarts [][]int) error {
	return validateUniqueInlineSectionNames(markdown, allStarts, func(name string) error {
		subAgentLog.Printf("Duplicate sub-agent name: %q", name)
		return fmt.Errorf("duplicate inline sub-agent name %q", name)
	})
}
