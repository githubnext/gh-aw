package parser

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var inlineSkillLog = logger.New("parser:inline_skill_extractor")

var validInlineSkillFrontmatterFields = map[string]bool{
	"description": true,
}

func ValidateInlineSkillsFrontmatter(markdown string) []string {
	var body string
	if parsed, err := ExtractFrontmatterFromContent(markdown); err == nil {
		body = parsed.Markdown
	} else {
		body = markdown
	}
	return ValidateInlineSkillsInBody(body)
}

func ValidateInlineSkillsInBody(body string) []string {
	_, skills, err := ExtractInlineSkills(body)
	if err != nil {
		return []string{fmt.Sprintf("could not extract inline skills: %v", err)}
	}
	if len(skills) == 0 {
		return nil
	}

	var warnings []string
	for _, skill := range skills {
		warnings = append(warnings, validateInlineSkillFrontmatterFields(skill)...)
	}
	return warnings
}

func validateInlineSkillFrontmatterFields(skill InlineSkill) []string {
	parsed, err := ExtractFrontmatterFromContent(skill.Content)
	if err != nil {
		return []string{fmt.Sprintf("skill %q: could not parse frontmatter: %v", skill.Name, err)}
	}
	if len(parsed.Frontmatter) == 0 {
		return nil
	}

	var unknown []string
	for key := range parsed.Frontmatter {
		if !validInlineSkillFrontmatterFields[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}

	sort.Strings(unknown)
	return []string{fmt.Sprintf(
		"skill %q: unknown frontmatter field(s): %s (valid fields: description)",
		skill.Name, strings.Join(unknown, ", "),
	)}
}

func GetEngineSkillDir(engineID string) string {
	switch strings.ToLower(engineID) {
	case "claude":
		return ".claude/skills"
	case "codex":
		return ".codex/skills"
	case "gemini":
		return ".gemini/skills"
	default:
		return ".github/skills"
	}
}

type InlineSkill struct {
	Name    string
	Content string
}

var inlineSkillSeparatorRegex = regexp.MustCompile("(?m)^##[ \t]+skill:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$")
var inlineSkillH2HeadingRegex = regexp.MustCompile(`(?m)^##[ \t]`)

func ExtractInlineSkills(markdown string) (mainMarkdown string, skills []InlineSkill, err error) {
	inlineSkillLog.Printf("Extracting inline skills from markdown (length: %d)", len(markdown))
	allStarts := inlineSkillSeparatorRegex.FindAllStringSubmatchIndex(markdown, -1)
	if len(allStarts) == 0 {
		inlineSkillLog.Print("No inline skill markers found")
		return markdown, nil, nil
	}

	inlineSkillLog.Printf("Found %d inline skill marker(s)", len(allStarts))
	if err := validateUniqueInlineSkillNames(markdown, allStarts); err != nil {
		return "", nil, err
	}

	mainMarkdown = strings.TrimRight(markdown[:allStarts[0][0]], "\n")
	h2Positions := collectInlineSkillH2Positions(markdown)
	for _, m := range allStarts {
		name, content := extractInlineSkill(markdown, m, h2Positions)
		inlineSkillLog.Printf("Extracted inline skill %q (content length: %d)", name, len(content))
		skills = append(skills, InlineSkill{Name: name, Content: content})
	}

	inlineSkillLog.Printf("Extraction complete: %d skill(s), main markdown length: %d", len(skills), len(mainMarkdown))
	return mainMarkdown, skills, nil
}

func validateUniqueInlineSkillNames(markdown string, allStarts [][]int) error {
	seen := make(map[string]struct{})
	for _, m := range allStarts {
		name := markdown[m[2]:m[3]]
		if _, exists := seen[name]; exists {
			inlineSkillLog.Printf("Duplicate inline skill name: %q", name)
			return fmt.Errorf("duplicate inline skill name %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func collectInlineSkillH2Positions(markdown string) []int {
	var h2Positions []int
	for _, m := range inlineSkillH2HeadingRegex.FindAllStringIndex(markdown, -1) {
		h2Positions = append(h2Positions, m[0])
	}
	return h2Positions
}

func extractInlineSkill(markdown string, marker []int, h2Positions []int) (string, string) {
	name := markdown[marker[2]:marker[3]]
	lineEnd := marker[1]
	if lineEnd < len(markdown) && markdown[lineEnd] == '\n' {
		lineEnd++
	}
	contentEnd := nextInlineSkillH2After(lineEnd, h2Positions, len(markdown))
	content := strings.TrimSpace(markdown[lineEnd:contentEnd])
	return name, content
}

func nextInlineSkillH2After(offset int, h2Positions []int, markdownLength int) int {
	for _, pos := range h2Positions {
		if pos >= offset {
			return pos
		}
	}
	return markdownLength
}
