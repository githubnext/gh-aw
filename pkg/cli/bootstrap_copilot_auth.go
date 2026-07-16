package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// runBootstrapRequireOwnerType validates that the repository owner matches the
// type required by the bootstrap action (e.g., "Organization" vs "User").
func runBootstrapRequireOwnerType(ctx context.Context, repo string, action repositoryPackageBootstrapAction) error {
	owner, _, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	ownerType, err := bootstrapCheckOwnerType(ctx, owner)
	if err != nil {
		return err
	}
	normalized := normalizeSetupOwnerType(ownerType)
	if action.Value != "" && action.Value != "any" && normalized != action.Value {
		return fmt.Errorf("owner %s is %s, but bootstrap profile requires %s. Example: set config[].value to %s or use a repository owned by a matching account type", owner, normalized, action.Value, normalized)
	}
	return nil
}

// runBootstrapCopilotAuthAction ensures that the Copilot PAT secret required
// for workflows using non-actions-token auth is present on the repository.
// It is a no-op when the selected workflows already grant copilot-requests:write
// via the GitHub Actions token.
func runBootstrapCopilotAuthAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState, usesActionsToken bool) (bool, error) {
	if usesActionsToken {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping Copilot PAT setup because selected workflows already support GitHub Actions token auth."))
		return false, nil
	}
	if _, exists := state.secrets[action.Secret]; exists {
		return false, nil
	}
	value, ok, err := resolveBootstrapSecretValue(action.Secret, "Copilot fine-grained PAT", "Enter a fine-grained personal access token starting with github_pat_.", false)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := stringutil.ValidateCopilotPAT(value); err != nil {
		return false, err
	}
	if err := bootstrapSetSecret(ctx, repo, action.Secret, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.Secret))
	return true, nil
}

// profileSourcesUseActionsTokenCopilotAuth reports whether all Copilot
// workflows in sources already grant copilot-requests:write via the GitHub
// Actions token, making a separate Copilot PAT unnecessary.
func profileSourcesUseActionsTokenCopilotAuth(ctx context.Context, sources []string) (bool, error) {
	if len(sources) == 0 {
		return false, nil
	}
	resolved, err := ResolveWorkflows(ctx, sources, false)
	if err != nil {
		return false, err
	}
	hasCopilot := false
	for _, candidate := range resolved.Workflows {
		if candidate == nil || candidate.IsActionWorkflow || candidate.IsPackageSkillFile || candidate.IsPackageAgentFile {
			continue
		}
		engine := strings.TrimSpace(candidate.Engine)
		if engine != "" && engine != "copilot" {
			continue
		}
		hasCopilot = true
		if !workflowGrantsCopilotRequestsWrite(candidate.Content) {
			return false, nil
		}
	}
	return hasCopilot, nil
}

// workflowGrantsCopilotRequestsWrite reports whether the workflow content
// declares permissions.copilot-requests: write in its frontmatter.
func workflowGrantsCopilotRequestsWrite(content []byte) bool {
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || frontmatter == nil {
		return false
	}
	permissions, ok := frontmatter.Frontmatter["permissions"].(map[string]any)
	if !ok {
		return false
	}
	level, ok := permissions[string(workflow.PermissionCopilotRequests)].(string)
	return ok && strings.TrimSpace(level) == "write"
}
