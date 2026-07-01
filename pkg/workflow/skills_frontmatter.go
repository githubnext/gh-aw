package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var skillsFrontmatterLog = logger.New("workflow:skills_frontmatter")

var skillSpecRegexp = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*)?@[0-9a-f]{40}$`)
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
		return fmt.Errorf("skills[%d] must be a non-empty string. Example: skills[%d]: \"owner/repo@abc1234...\"", idx, idx)
	}
	if !skillSpecRegexp.MatchString(skillSpec) {
		return fmt.Errorf(
			"skills[%d] must use owner/repo@<40-char-sha> or owner/repo/skill/path@<40-char-sha> (got %q). Example: skills[%d]: \"owner/repo@abcdef1234567890abcdef1234567890abcdef12\"",
			idx,
			skillSpec,
			idx,
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
		return errors.New("skills must be an array of skill references. Example: skills: [\"owner/repo@sha\"]")
	}

	skillsFrontmatterLog.Printf("validateFrontmatterSkills: validating %d skill entr(ies)", len(skills))

	for i, rawSkill := range skills {
		switch typed := rawSkill.(type) {
		case string:
			if err := validateSkillSpecValue(typed, i); err != nil {
				return err
			}
		case map[string]any:
			if len(typed) == 0 {
				return fmt.Errorf("skills[%d] must include a non-empty skill field. Example: skills[%d]: {skill: \"owner/repo@sha\"}", i, i)
			}
			skillValue, hasSkill := typed["skill"]
			if !hasSkill {
				return fmt.Errorf("skills[%d].skill is required. Example: skills[%d].skill: \"owner/repo@sha\"", i, i)
			}
			skillSpec, ok := skillValue.(string)
			if !ok {
				return fmt.Errorf("skills[%d].skill must be a string. Example: skills[%d].skill: \"owner/repo@sha\"", i, i)
			}
			if strings.TrimSpace(skillSpec) == "" {
				return fmt.Errorf("skills[%d].skill must be a non-empty string. Example: skills[%d].skill: \"owner/repo@sha\"", i, i)
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
					return fmt.Errorf("skills[%d].github-token must be a string. Example: skills[%d].github-token: \"${{ secrets.MY_TOKEN }}\"", i, i)
				}
				if !githubTokenExpressionRegexp.MatchString(token) {
					return fmt.Errorf(
						"skills[%d].github-token must be a valid GitHub token expression. Example: skills[%d].github-token: \"${{ secrets.NAME }}\" or \"${{ needs.auth.outputs.token }}\"",
						i,
						i,
					)
				}
			}
			if app, hasApp := typed["github-app"]; hasApp {
				appMap, ok := app.(map[string]any)
				if !ok {
					return fmt.Errorf("skills[%d].github-app must be an object. Example: skills[%d].github-app: {client-id: \"Iv1.abc\", private-key: \"...\"}", i, i)
				}
				parsed := parseAppConfig(appMap)
				if !parsed.hasRequiredCredentials() {
					return fmt.Errorf("skills[%d].github-app must include non-empty client-id/app-id and private-key. Example: skills[%d].github-app: {client-id: \"Iv1.abc\", private-key: \"...\"}", i, i)
				}
			}
		default:
			return fmt.Errorf("skills[%d] must be a string or object. Example: skills[%d]: \"owner/repo@sha\" or {skill: \"owner/repo@sha\"}", i, i)
		}
	}

	return nil
}

func isRepositorySkillSpec(skillSpec string) bool {
	base, _, _ := strings.Cut(skillSpec, "@")
	// owner/repo has exactly one slash; owner/repo/skill/path has two or more.
	return strings.Count(base, "/") == 1
}

func parseRawSkillReferences(rawSkills []any) []SkillReference {
	skillsFrontmatterLog.Printf("parseRawSkillReferences: parsing %d raw skill entr(ies)", len(rawSkills))
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
	skillsFrontmatterLog.Printf("parseRawSkillReferences: parsed %d skill reference(s)", len(refs))
	return refs
}
