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

type InlineSkill struct {
	Name    string
	Content string
}

var inlineSkillSeparatorRegex = regexp.MustCompile("(?m)^##[ \t]+skill:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$")

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
	h2Positions := collectH2Positions(markdown)
	for _, m := range allStarts {
		name, content := extractInlineSection(markdown, m, h2Positions)
		inlineSkillLog.Printf("Extracted inline skill %q (content length: %d)", name, len(content))
		skills = append(skills, InlineSkill{Name: name, Content: content})
	}

	inlineSkillLog.Printf("Extraction complete: %d skill(s), main markdown length: %d", len(skills), len(mainMarkdown))
	return mainMarkdown, skills, nil
}

func validateUniqueInlineSkillNames(markdown string, allStarts [][]int) error {
	return validateUniqueInlineSectionNames(markdown, allStarts, func(name string) error {
		inlineSkillLog.Printf("Duplicate inline skill name: %q", name)
		return fmt.Errorf("duplicate inline skill name %q", name)
	})
}
