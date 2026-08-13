package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/ctxutil"
)

// resolveFrontmatterSkillRefs pins non-SHA remote skill refs (owner/repo[/path]@ref) to
// their resolved commit SHA at compile time, using the compiler's shared action resolver
// (the same GitHub API/cache infrastructure used to pin "uses:" action references). Refs
// that are already a full 40-character SHA are left untouched. Remote skill entries with
// no ref specified (owner/repo[/path]@) are left unpinned and trigger an advisory warning
// recommending an explicit ref. Local skill paths and GitHub Actions expressions are
// ignored.
//
// data.SkillReferences and data.Skills are always populated together from the same
// frontmatter "skills" entries (in the same order), so resolving SkillReferences and
// mirroring the result into Skills keeps both in sync without resolving each entry twice.
func (c *Compiler) resolveFrontmatterSkillRefs(data *WorkflowData, markdownPath string) {
	if data == nil {
		return
	}
	if len(data.SkillReferences) > 0 {
		for i := range data.SkillReferences {
			data.SkillReferences[i].Skill = c.resolveSkillRefSpec(data, markdownPath, data.SkillReferences[i].Skill, i)
		}
		for i := range data.Skills {
			if i < len(data.SkillReferences) {
				data.Skills[i] = data.SkillReferences[i].Skill
			}
		}
		return
	}
	for i := range data.Skills {
		data.Skills[i] = c.resolveSkillRefSpec(data, markdownPath, data.Skills[i], i)
	}
}

// resolveSkillRefSpec resolves a single skills[] entry, returning the (possibly
// SHA-pinned) spec to use going forward. It never returns an error: resolution
// failures degrade to a warning and the original, unpinned spec is kept so
// compilation can proceed.
func (c *Compiler) resolveSkillRefSpec(data *WorkflowData, markdownPath, spec string, idx int) string {
	parsed := parseSkillRefSpec(spec)
	if !parsed.isRemote {
		return spec
	}

	if parsed.ref == "" {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
			fmt.Sprintf(
				"skills[%d] %q has no ref pinned; the skill will be installed from the repository's default branch on every run. "+
					"Pin a branch, tag, or commit SHA for reproducible builds, e.g. skills[%d]: \"%s@main\".",
				idx, parsed.trimmed, idx, parsed.repoPath)))
		c.IncrementWarningCount()
		return spec
	}

	if parsed.isFullSHA {
		skillsFrontmatterLog.Printf("skills[%d] %q is already SHA-pinned", idx, parsed.trimmed)
		return spec
	}

	if data.ActionResolver == nil {
		skillsFrontmatterLog.Printf("skills[%d]: no action resolver available, skipping SHA pinning for %q", idx, parsed.trimmed)
		return spec
	}

	sha, err := data.ActionResolver.ResolveSHA(ctxutil.OrBackground(data.Ctx), parsed.repoPath, parsed.ref)
	if err != nil {
		skillsFrontmatterLog.Printf("skills[%d]: failed to resolve ref %q for %q to a SHA: %v", idx, parsed.ref, parsed.repoPath, err)
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
			fmt.Sprintf(
				"skills[%d]: failed to resolve ref %q for %q to a commit SHA (%v); the workflow will use the unpinned ref as-is.",
				idx, parsed.ref, parsed.repoPath, err)))
		c.IncrementWarningCount()
		return spec
	}

	pinned := fmt.Sprintf("%s@%s", parsed.repoPath, sha)
	skillsFrontmatterLog.Printf("skills[%d]: pinned %q to %q", idx, parsed.trimmed, pinned)
	return pinned
}
