package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

var bootstrapActionsRepoLog = logger.New("cli:bootstrap_profile_actions_repo")

func runBootstrapRepoVariableAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	bootstrapActionsRepoLog.Printf("Running repo variable action: repo=%s, name=%s", repo, action.Name)
	if _, exists := state.variables[action.Name]; exists {
		bootstrapActionsRepoLog.Printf("Skipping variable %s: already set on repo", action.Name)
		return false, nil
	}
	value, ok, err := resolveBootstrapTextValue(bootstrapRepositoryVariableEnvName(action.Name), action.Prompt, action.Description, action.Default, action.Enum, action.Optional)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := bootstrapUpsertVariable(ctx, repo, action.Name, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository variable "+action.Name))
	return true, nil
}

func runBootstrapRepoSecretAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	bootstrapActionsRepoLog.Printf("Running repo secret action: repo=%s, name=%s", repo, action.Name)
	if _, exists := state.secrets[action.Name]; exists {
		bootstrapActionsRepoLog.Printf("Skipping secret %s: already set on repo", action.Name)
		return false, nil
	}
	value, ok, err := resolveBootstrapSecretValue(bootstrapRepositorySecretEnvName(action.Name), action.Prompt, action.Description, action.Optional)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := bootstrapSetSecret(ctx, repo, action.Name, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.Name))
	return true, nil
}

func runBootstrapCopilotAuthAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState, usesActionsToken bool) (bool, error) {
	bootstrapActionsRepoLog.Printf("Running Copilot auth action: repo=%s, usesActionsToken=%v", repo, usesActionsToken)
	if usesActionsToken {
		bootstrapActionsRepoLog.Print("Skipping Copilot PAT setup: workflows already support Actions token auth")
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

func listBootstrapRepoVariableNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository variables...", "api", fmt.Sprintf("/repos/%s/actions/variables?per_page=100", repo), "--paginate", "--jq", ".variables[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository variables for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

func listBootstrapRepoSecretNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository secrets...", "api", fmt.Sprintf("/repos/%s/actions/secrets?per_page=100", repo), "--paginate", "--jq", ".secrets[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository secrets for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

func upsertBootstrapRepoVariable(repo, name, value string) error {
	target := defaultsTarget{}
	owner, repoName, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	target.scope = defaultsScopeRepo
	target.repoOwner = owner
	target.repoName = repoName
	return upsertDefaultsVariable(target, name, value)
}

func setBootstrapRepoSecret(repo, name, value string) error {
	owner, repoName, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	client, err := api.NewRESTClient(secretSetClientOptions(""))
	if err != nil {
		return err
	}
	return setRepoSecret(client, owner, repoName, name, value)
}
