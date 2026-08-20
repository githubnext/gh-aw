package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/ctxutil"
	"github.com/github/gh-aw/pkg/gitutil"
)

func (c *Compiler) resolveFrontmatterPluginRefs(workflowData *WorkflowData) error {
	if workflowData == nil || len(workflowData.Plugins) == 0 {
		return nil
	}

	for i, plugin := range workflowData.Plugins {
		parsed := parseSkillRefSpec(plugin)
		if !parsed.isRemote || parsed.ref == "" {
			return fmt.Errorf("plugins[%d]: cannot pin invalid plugin reference %q", i, plugin)
		}
		if parsed.isFullSHA {
			continue
		}
		if workflowData.ActionResolver == nil {
			return fmt.Errorf("plugins[%d]: cannot resolve %q to a commit SHA because no GitHub reference resolver is available", i, plugin)
		}

		sha, err := workflowData.ActionResolver.ResolveSHA(
			ctxutil.OrBackground(workflowData.Ctx),
			parsed.repoPath,
			parsed.ref,
		)
		if err != nil {
			return fmt.Errorf("plugins[%d]: failed to resolve %q to a commit SHA: %w", i, plugin, err)
		}
		if !gitutil.IsValidFullSHA(sha) {
			return fmt.Errorf("plugins[%d]: resolved %q to invalid commit SHA %q", i, plugin, sha)
		}
		workflowData.Plugins[i] = parsed.repoPath + "@" + sha
	}

	return nil
}
