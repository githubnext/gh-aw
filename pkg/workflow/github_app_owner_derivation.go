package workflow

import (
	"fmt"
	"strings"
)

// inferSingleCheckoutRepositoryForGitHubAppOwner returns the single explicit checkout.repository
// value when the workflow targets exactly one distinct repository. It ignores the default checkout
// of the current repository and returns an empty string when multiple distinct repositories are
// configured.
func inferSingleCheckoutRepositoryForGitHubAppOwner(data *WorkflowData) string {
	if data == nil {
		return ""
	}

	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)
	var repository string
	for _, entry := range checkoutMgr.ordered {
		if entry.key.repository == "" {
			continue
		}
		if repository == "" {
			repository = entry.key.repository
			continue
		}
		if entry.key.repository != repository {
			return ""
		}
	}

	return repository
}

func buildGitHubAppOwnerResolutionSteps(repositoryExpr, stepName, stepID string) (string, []string) {
	if owner, ok := deriveLiteralGitHubAppOwner(repositoryExpr); ok {
		return owner, nil
	}
	if strings.TrimSpace(repositoryExpr) == "" {
		return "${{ github.repository_owner }}", nil
	}

	ownerStepID := stepID + "-owner"
	ownerStepName := strings.Replace(stepName, "Generate GitHub App token", "Derive GitHub App owner", 1)
	if ownerStepName == stepName {
		ownerStepName = "Derive GitHub App owner"
	}

	var steps []string
	steps = append(steps, fmt.Sprintf("      - name: %s\n", ownerStepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", ownerStepID))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_TARGET_REPOSITORY: %s\n", repositoryExpr))
	steps = append(steps, "        shell: bash\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          set -euo pipefail\n")
	steps = append(steps, "          repo=\"${GH_AW_TARGET_REPOSITORY%.wiki}\"\n")
	steps = append(steps, "          owner=\"${repo%%/*}\"\n")
	steps = append(steps, "          if [[ -z \"$repo\" || -z \"$owner\" || \"$owner\" == \"$repo\" ]]; then\n")
	steps = append(steps, "            echo \"Unable to derive GitHub App owner from repository: $GH_AW_TARGET_REPOSITORY\" >&2\n")
	steps = append(steps, "            exit 1\n")
	steps = append(steps, "          fi\n")
	steps = append(steps, "          echo \"owner=$owner\" >> \"$GITHUB_OUTPUT\"\n")

	return "${{ steps." + ownerStepID + ".outputs.owner }}", steps
}

func resolveGitHubAppOwner(app *GitHubAppConfig, ownerSourceRepository, stepName, stepID string) (string, []string) {
	if app != nil && strings.TrimSpace(app.Owner) != "" {
		return app.Owner, nil
	}
	return buildGitHubAppOwnerResolutionSteps(ownerSourceRepository, stepName, stepID)
}

func deriveLiteralGitHubAppOwner(repository string) (string, bool) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return "", false
	}

	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return "", false
	}

	owner := strings.TrimSpace(parts[0])
	if owner == "" || strings.Contains(owner, "${{") {
		return "", false
	}
	return owner, true
}
