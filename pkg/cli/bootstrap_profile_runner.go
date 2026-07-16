package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/github/gh-aw/pkg/workflow"
)

const (
	bootstrapProfileManifestTimeout  = 10 * time.Minute
	bootstrapProfileInstallPollDelay = 5 * time.Second
)

var (
	runBootstrapGHContext    = workflow.RunGHContext
	bootstrapIsInteractive   = tty.IsStderrTerminal
	bootstrapUpsertVariable  = upsertBootstrapRepoVariable
	bootstrapSetSecret       = setBootstrapRepoSecret
	bootstrapCreateGitHubApp = createBootstrapGitHubApp
	bootstrapCheckOwnerType  = checkSetupRepositoryOwnerType
)

type bootstrapProfileRunConfig struct {
	Repo     string
	RepoDir  string
	Sources  []string
	Profile  *resolvedBootstrapProfile
	Yes      bool
	PlanOnly bool
	Verbose  bool
	Force    bool
	// UseCopilotRequests indicates the user chose org-billing (copilot-requests) auth
	// instead of a PAT. When true, copilot-auth config actions are skipped because
	// the workflow already has permissions.copilot-requests: write injected.
	UseCopilotRequests bool
}

type bootstrapProfileExistingState struct {
	variables map[string]struct{}
	secrets   map[string]struct{}
}

func buildBootstrapProfilePlan(ctx context.Context, repo string, profile *resolvedBootstrapProfile, sources []string, repoReady bool) (bool, []string, error) {
	if profile == nil || profile.Profile == nil {
		return false, nil, nil
	}

	lines := make([]string, 0, len(profile.Profile.Config))
	if !repoReady {
		for _, action := range profile.Profile.Config {
			if err := validateBootstrapActionPreRepo(ctx, repo, action); err != nil {
				return false, nil, err
			}
			if bootstrapActionCanMutate(action, sources) {
				lines = append(lines, "- bootstrap profile will configure "+bootstrapActionPlanLabel(action))
			}
		}
		return len(lines) > 0, lines, nil
	}

	state, err := bootstrapProfileState(ctx, repo)
	if err != nil {
		return false, nil, err
	}
	usesActionsToken, err := profileSourcesUseActionsTokenCopilotAuth(ctx, sources)
	if err != nil {
		return false, nil, err
	}

	needsMutation := false
	for _, action := range profile.Profile.Config {
		pending, err := bootstrapActionNeedsMutation(ctx, repo, action, state, usesActionsToken)
		if err != nil {
			return false, nil, err
		}
		if pending {
			needsMutation = true
			lines = append(lines, "- bootstrap profile will configure "+bootstrapActionPlanLabel(action))
		}
	}

	bootstrapLog.Printf("Built bootstrap profile plan: repo=%s, needsMutation=%t, planLines=%d", repo, needsMutation, len(lines))
	return needsMutation, lines, nil
}

func executeBootstrapProfile(ctx context.Context, config bootstrapProfileRunConfig) error {
	if config.Profile == nil || config.Profile.Profile == nil {
		return nil
	}

	bootstrapLog.Printf("Executing bootstrap profile: repo=%s, actions=%d, useCopilotRequests=%t", config.Repo, len(config.Profile.Profile.Config), config.UseCopilotRequests)

	state, err := bootstrapProfileState(ctx, config.Repo)
	if err != nil {
		return err
	}
	usesActionsToken, err := profileSourcesUseActionsTokenCopilotAuth(ctx, config.Sources)
	if err != nil {
		return err
	}

	for _, action := range config.Profile.Profile.Config {
		pending, err := bootstrapActionNeedsMutation(ctx, config.Repo, action, state, usesActionsToken)
		if err != nil {
			return err
		}
		if !pending && action.Type != "handoff" {
			bootstrapLog.Printf("Skipping bootstrap action (no mutation needed): type=%s", action.Type)
			continue
		}

		bootstrapLog.Printf("Applying bootstrap action: type=%s", action.Type)
		switch action.Type {
		case "require-owner-type":
			if err := runBootstrapRequireOwnerType(ctx, config.Repo, action); err != nil {
				return err
			}
		case "repo-variable":
			applied, err := runBootstrapRepoVariableAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			if applied {
				state.variables[action.Name] = struct{}{}
			}
		case "repo-secret":
			applied, err := runBootstrapRepoSecretAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			if applied {
				state.secrets[action.Name] = struct{}{}
			}
		case "github-app":
			_, err := runBootstrapGitHubAppAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			state.variables[action.AppIDVariable] = struct{}{}
			state.secrets[action.PrivateKeySecret] = struct{}{}
		case "copilot-auth":
			if config.UseCopilotRequests {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping Copilot PAT setup because org Copilot billing is enabled."))
				continue
			}
			applied, err := runBootstrapCopilotAuthAction(ctx, config.Repo, action, state, usesActionsToken)
			if err != nil {
				return err
			}
			if applied {
				state.secrets[action.Secret] = struct{}{}
			}
		case "handoff":
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(action.Message))
		default:
			return fmt.Errorf("unsupported bootstrap action type %q. Example: use one of %s", action.Type, bootstrapActionTypeExample)
		}
	}

	return nil
}

func bootstrapProfileState(ctx context.Context, repo string) (*bootstrapProfileExistingState, error) {
	variableNames, err := listBootstrapRepoVariableNames(ctx, repo)
	if err != nil {
		return nil, err
	}
	secretNames, err := listBootstrapRepoSecretNames(ctx, repo)
	if err != nil {
		return nil, err
	}

	state := &bootstrapProfileExistingState{
		variables: make(map[string]struct{}, len(variableNames)),
		secrets:   make(map[string]struct{}, len(secretNames)),
	}
	for _, name := range variableNames {
		state.variables[name] = struct{}{}
	}
	for _, name := range secretNames {
		state.secrets[name] = struct{}{}
	}
	return state, nil
}

func bootstrapActionNeedsMutation(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState, usesActionsToken bool) (bool, error) {
	switch action.Type {
	case "require-owner-type":
		return false, runBootstrapRequireOwnerType(ctx, repo, action)
	case "repo-variable":
		_, exists := state.variables[action.Name]
		return !exists, nil
	case "repo-secret":
		_, exists := state.secrets[action.Name]
		return !exists, nil
	case "github-app":
		_, hasVar := state.variables[action.AppIDVariable]
		_, hasSecret := state.secrets[action.PrivateKeySecret]
		return !hasVar || !hasSecret, nil
	case "copilot-auth":
		_, hasSecret := state.secrets[action.Secret]
		return !hasSecret && !usesActionsToken, nil
	case "handoff":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported bootstrap action type %q. Example: use one of %s", action.Type, bootstrapActionTypeExample)
	}
}

func validateBootstrapActionPreRepo(ctx context.Context, repo string, action repositoryPackageBootstrapAction) error {
	if action.Type == "require-owner-type" {
		return runBootstrapRequireOwnerType(ctx, repo, action)
	}
	return nil
}

func bootstrapActionCanMutate(action repositoryPackageBootstrapAction, sources []string) bool {
	switch action.Type {
	case "repo-variable", "repo-secret", "github-app":
		return true
	case "copilot-auth":
		return true
	default:
		return false
	}
}

func bootstrapActionPlanLabel(action repositoryPackageBootstrapAction) string {
	switch action.Type {
	case "repo-variable":
		return "repository variable " + action.Name
	case "repo-secret":
		return "repository secret " + action.Name
	case "github-app":
		return fmt.Sprintf("GitHub App credentials (%s, %s)", action.AppIDVariable, action.PrivateKeySecret)
	case "copilot-auth":
		return "Copilot secret " + action.Secret
	default:
		return action.Type
	}
}
