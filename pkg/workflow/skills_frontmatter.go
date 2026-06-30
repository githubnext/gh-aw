package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var skillSpecRegexp = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*)?@[0-9a-f]{40}$`)
var skillSpecExpressionRefRegexp = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*)?@\$\{\{.+\}\}$`)
var githubActionsExpressionRegexp = regexp.MustCompile(`^\$\{\{.+\}\}$`)

func validateFrontmatterSkills(frontmatter map[string]any) error {
	rawSkills, hasSkills := frontmatter["skills"]
	if !hasSkills {
		return nil
	}

	skills, ok := rawSkills.([]any)
	if !ok {
		return errors.New("skills must be an array of skill references")
	}

	for i, rawSkill := range skills {
		skillSpec, ok := rawSkill.(string)
		if !ok || strings.TrimSpace(skillSpec) == "" {
			return fmt.Errorf("skills[%d] must be a non-empty string", i)
		}
		if githubActionsExpressionRegexp.MatchString(skillSpec) || skillSpecExpressionRefRegexp.MatchString(skillSpec) {
			continue
		}
		if !skillSpecRegexp.MatchString(skillSpec) {
			return fmt.Errorf(
				"skills[%d] must use owner/repo@<40-char-sha>, owner/repo/skill/path@<40-char-sha>, or a GitHub Actions expression: %q",
				i,
				skillSpec,
			)
		}
	}

	return nil
}

func isRepositorySkillSpec(skillSpec string) bool {
	base, _, _ := strings.Cut(skillSpec, "@")
	// owner/repo has exactly one slash; owner/repo/skill/path has two or more.
	// Expression-only specs have no static @ suffix and are treated as path-scoped
	// until the resolved runtime value is inspected by the install step.
	return strings.Count(base, "/") == 1
}
