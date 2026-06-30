package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/gitutil"
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
		_, ref, ok := strings.Cut(skillSpec, "@")
		if !ok || !gitutil.IsValidFullSHA(ref) {
			return fmt.Errorf("skills[%d] reference must be pinned to a full lowercase 40-character SHA: %q", i, skillSpec)
		}
	}

	return nil
}

func isRepositorySkillSpec(skillSpec string) bool {
	base, _, _ := strings.Cut(skillSpec, "@")
	return strings.Count(base, "/") == 1
}
