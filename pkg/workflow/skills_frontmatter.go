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
var githubTokenExpressionRegexp = regexp.MustCompile(`^\$\{\{\s*(secrets\.[A-Za-z_][A-Za-z0-9_]*(\s*\|\|\s*secrets\.[A-Za-z_][A-Za-z0-9_]*)*|needs\.[A-Za-z_][A-Za-z0-9_]*\.outputs\.[A-Za-z_][A-Za-z0-9_]*)\s*\}\}$`)

// SkillReference describes a single skills[] entry in workflow frontmatter.
// It supports both legacy string-only entries and object entries with per-skill auth.
type SkillReference struct {
	Skill       string           `json:"skill,omitempty"`
	GitHubToken string           `json:"github-token,omitempty"`
	GitHubApp   *GitHubAppConfig `json:"github-app,omitempty"`
}

func validateSkillSpecValue(skillSpec string, idx int) error {
	if strings.TrimSpace(skillSpec) == "" {
		return fmt.Errorf("skills[%d] must be a non-empty string", idx)
	}
	if githubActionsExpressionRegexp.MatchString(skillSpec) || skillSpecExpressionRefRegexp.MatchString(skillSpec) {
		return nil
	}
	if !skillSpecRegexp.MatchString(skillSpec) {
		return fmt.Errorf(
			"skills[%d] must use owner/repo@<40-char-sha>, owner/repo/skill/path@<40-char-sha>, or a GitHub Actions expression: %q",
			idx,
			skillSpec,
		)
	}
	return nil
}

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
		switch typed := rawSkill.(type) {
		case string:
			if err := validateSkillSpecValue(typed, i); err != nil {
				return err
			}
		case map[string]any:
			if len(typed) == 0 {
				return fmt.Errorf("skills[%d] must include a non-empty skill field", i)
			}
			skillValue, hasSkill := typed["skill"]
			if !hasSkill {
				return fmt.Errorf("skills[%d].skill is required", i)
			}
			skillSpec, ok := skillValue.(string)
			if !ok {
				return fmt.Errorf("skills[%d].skill must be a string", i)
			}
			if strings.TrimSpace(skillSpec) == "" {
				return fmt.Errorf("skills[%d].skill must be a non-empty string", i)
			}
			if err := validateSkillSpecValue(skillSpec, i); err != nil {
				return err
			}
			for key := range typed {
				switch key {
				case "skill", "github-token", "github-app":
					// allowed
				default:
					return fmt.Errorf("skills[%d].%s is not supported; allowed fields are skill, github-token, github-app", i, key)
				}
			}
			_, hasToken := typed["github-token"]
			_, hasApp := typed["github-app"]
			if hasToken && hasApp {
				return fmt.Errorf("skills[%d]: github-token and github-app are mutually exclusive; use one or the other", i)
			}
			if tokenValue, hasToken := typed["github-token"]; hasToken {
				token, ok := tokenValue.(string)
				if !ok {
					return fmt.Errorf("skills[%d].github-token must be a string", i)
				}
				if !githubTokenExpressionRegexp.MatchString(token) {
					return fmt.Errorf(
						"skills[%d].github-token must be a valid GitHub token expression (e.g., '${{ secrets.NAME }}' or '${{ needs.auth.outputs.token }}')",
						i,
					)
				}
			}
			if app, hasApp := typed["github-app"]; hasApp {
				appMap, ok := app.(map[string]any)
				if !ok {
					return fmt.Errorf("skills[%d].github-app must be an object", i)
				}
				parsed := parseAppConfig(appMap)
				if !parsed.hasRequiredCredentials() {
					return fmt.Errorf("skills[%d].github-app must include non-empty client-id/app-id and private-key", i)
				}
			}
		default:
			return fmt.Errorf("skills[%d] must be a string or object", i)
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

func parseRawSkillReferences(rawSkills []any) []SkillReference {
	refs := make([]SkillReference, 0, len(rawSkills))
	for _, rawSkill := range rawSkills {
		switch typed := rawSkill.(type) {
		case string:
			skillSpec := strings.TrimSpace(typed)
			if skillSpec == "" {
				continue
			}
			refs = append(refs, SkillReference{Skill: skillSpec})
		case map[string]any:
			skillSpec, _ := typed["skill"].(string)
			if strings.TrimSpace(skillSpec) == "" {
				continue
			}
			ref := SkillReference{Skill: strings.TrimSpace(skillSpec)}
			if token, ok := typed["github-token"].(string); ok {
				ref.GitHubToken = token
			}
			if appMap, ok := typed["github-app"].(map[string]any); ok {
				ref.GitHubApp = parseAppConfig(appMap)
			}
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return nil
	}
	return refs
}
