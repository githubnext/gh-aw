package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

// isPreInstallBootstrapType reports whether the config action type should run
// before engine selection and workflow installation in the add-wizard flow.
func isPreInstallBootstrapType(actionType string) bool {
	switch actionType {
	case "repo-variable", "repo-secret":
		return true
	default:
		return false
	}
}

// splitBootstrapProfile splits a resolved bootstrap profile into pre-install and
// post-install parts for the add-wizard flow. Pre-install actions (repo-variable
// and repo-secret) run before engine selection; all remaining actions run after
// the workflow PR has been created.
//
// Either returned value may be nil if there are no actions in that phase.
func splitBootstrapProfile(profile *resolvedBootstrapProfile) (preInstall, postInstall *resolvedBootstrapProfile) {
	if profile == nil || profile.Profile == nil || len(profile.Profile.Config) == 0 {
		return nil, nil
	}

	var preActions, postActions []repositoryPackageBootstrapAction
	for _, action := range profile.Profile.Config {
		if isPreInstallBootstrapType(action.Type) {
			preActions = append(preActions, action)
		} else {
			postActions = append(postActions, action)
		}
	}

	if len(preActions) > 0 {
		preInstall = &resolvedBootstrapProfile{
			PackageID: profile.PackageID,
			Source:    profile.Source,
			Profile:   &repositoryPackageBootstrap{Config: preActions},
		}
	}
	if len(postActions) > 0 {
		postInstall = &resolvedBootstrapProfile{
			PackageID: profile.PackageID,
			Source:    profile.Source,
			Profile:   &repositoryPackageBootstrap{Config: postActions},
		}
	}
	return preInstall, postInstall
}

// printBootstrapConfigTODO prints a TODO checklist of manual steps required by the
// config entries in the package manifest. Called by the non-interactive
// "add" command after workflows have been installed.
func printBootstrapConfigTODO(w io.Writer, profile *resolvedBootstrapProfile) {
	if profile == nil || profile.Profile == nil || len(profile.Profile.Config) == 0 {
		return
	}

	bootstrapLog.Printf("Printing bootstrap config TODO: package=%s, actions=%d", profile.PackageID, len(profile.Profile.Config))
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, console.FormatInfoMessage("Post-installation steps from "+profile.PackageID+":"))

	for _, action := range profile.Profile.Config {
		switch action.Type {
		case "require-owner-type":
			fmt.Fprintf(w, "  ☐ Verify repository owner type: %s\n", action.Value)
		case "repo-variable":
			line := "  ☐ Set repository variable: " + action.Name
			if action.Prompt != "" {
				line += " — " + action.Prompt
			}
			if action.Optional {
				line += " (optional)"
			}
			fmt.Fprintln(w, line)
		case "repo-secret":
			line := "  ☐ Set repository secret: " + action.Name
			if action.Prompt != "" {
				line += " — " + action.Prompt
			}
			if action.Optional {
				line += " (optional)"
			}
			fmt.Fprintln(w, line)
		case "github-app":
			appLabel := action.AppName
			if appLabel == "" {
				appLabel = "GitHub App"
			}
			fmt.Fprintf(w, "  ☐ Configure %s (variable: %s, secret: %s)\n",
				appLabel, action.AppIDVariable, action.PrivateKeySecret)
		case "copilot-auth":
			secret := action.Secret
			if secret == "" {
				secret = "COPILOT_GITHUB_TOKEN"
			}
			fmt.Fprintf(w, "  ☐ Set Copilot PAT secret: %s\n", secret)
		case "commit-and-push":
			fmt.Fprintf(w, "  ☐ Commit and push local changes — %s\n", action.Message)
		case "handoff":
			fmt.Fprintln(w, console.FormatInfoMessage(action.Message))
		}
	}

	fmt.Fprintln(w, "")
}

// executeBootstrapConfigForAdd runs the bootstrap config actions interactively.
// Used by add-wizard after the workflow PR has been created and merged.
func executeBootstrapConfigForAdd(ctx context.Context, repo string, sources []string, profile *resolvedBootstrapProfile, useCopilotRequests bool, verbose bool) error {
	if profile == nil || profile.Profile == nil || len(profile.Profile.Config) == 0 {
		return nil
	}

	if repo == "" {
		return errors.New("--repo OWNER/REPO is required to apply bootstrap config steps interactively")
	}

	bootstrapLog.Printf("Applying bootstrap config for add: repo=%s, package=%s, actions=%d, useCopilotRequests=%t", repo, profile.PackageID, len(profile.Profile.Config), useCopilotRequests)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying setup steps from "+profile.PackageID+"..."))
	repoDir, err := gitutil.FindGitRoot()
	if err != nil {
		bootstrapLog.Printf("Could not determine git root for add bootstrap config: %v", err)
	}

	return executeBootstrapProfile(ctx, bootstrapProfileRunConfig{
		Repo:               repo,
		RepoDir:            repoDir,
		Sources:            sources,
		Profile:            profile,
		UseCopilotRequests: useCopilotRequests,
		Verbose:            verbose,
	})
}
