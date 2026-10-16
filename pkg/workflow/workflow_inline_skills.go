package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var workflowInlineSkillsLog = logger.New("workflow:workflow_inline_skills")

// collectWorkflowInlineSkills returns the workflow-owned inline skills: the main workflow's
// ## skill: blocks followed by those from its imports. Sub-agent sections are stripped first,
// mirroring the activation-time extraction order in interpolate_prompt.cjs.
func collectWorkflowInlineSkills(markdown string, imported []parser.InlineSkill) []parser.InlineSkill {
	withoutSubAgents, _, err := parser.ExtractInlineSubAgents(markdown)
	if err != nil {
		withoutSubAgents = markdown
	}
	_, skills, err := parser.ExtractInlineSkills(withoutSubAgents)
	if err != nil {
		workflowInlineSkillsLog.Printf("Skipping main workflow inline skills: %v", err)
		skills = nil
	}
	workflowInlineSkillsLog.Printf("Collected inline skills: main=%d, imported=%d", len(skills), len(imported))
	return append(skills, imported...)
}
