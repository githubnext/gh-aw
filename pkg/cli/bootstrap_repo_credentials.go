package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/repoutil"
)

// runBootstrapRepoVariableAction ensures the repository variable described by
// action exists. It resolves the value interactively or from an environment
// variable and skips when the variable is already present in state.
func runBootstrapRepoVariableAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	if _, exists := state.variables[action.Name]; exists {
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

// runBootstrapRepoSecretAction ensures the repository secret described by
// action exists. It resolves the value interactively or from an environment
// variable and skips when the secret is already present in state.
func runBootstrapRepoSecretAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	if _, exists := state.secrets[action.Name]; exists {
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

// listBootstrapRepoVariableNames returns the names of all Actions variables
// currently set on the repository, sorted alphabetically.
func listBootstrapRepoVariableNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository variables...", "api", fmt.Sprintf("/repos/%s/actions/variables?per_page=100", repo), "--paginate", "--jq", ".variables[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository variables for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

// listBootstrapRepoSecretNames returns the names of all Actions secrets
// currently set on the repository, sorted alphabetically.
func listBootstrapRepoSecretNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository secrets...", "api", fmt.Sprintf("/repos/%s/actions/secrets?per_page=100", repo), "--paginate", "--jq", ".secrets[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository secrets for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

// upsertBootstrapRepoVariable creates or updates an Actions variable on the
// repository using the defaults upsert helper.
func upsertBootstrapRepoVariable(ctx context.Context, repo, name, value string) error {
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

// setBootstrapRepoSecret creates or updates an Actions secret on the repository.
func setBootstrapRepoSecret(ctx context.Context, repo, name, value string) error {
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
