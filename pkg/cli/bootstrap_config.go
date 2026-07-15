package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
)

// printBootstrapConfigTODO prints a TODO checklist of manual steps required by the
// config entries in the package manifest. Called by the non-interactive
// "add" command after workflows have been installed.
func printBootstrapConfigTODO(profile *resolvedBootstrapProfile) {
	if profile == nil || profile.Profile == nil || len(profile.Profile.Config) == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Post-installation steps from "+profile.PackageID+":"))

	for _, action := range profile.Profile.Config {
		switch action.Type {
		case "require-owner-type":
			fmt.Fprintf(os.Stderr, "  ✓ Repository owner type constraint: %s\n", action.Value)
		case "repo-variable":
			line := "  ☐ Set repository variable: " + action.Name
			if action.Prompt != "" {
				line += " — " + action.Prompt
			}
			if action.Optional {
				line += " (optional)"
			}
			fmt.Fprintln(os.Stderr, line)
		case "repo-secret":
			line := "  ☐ Set repository secret: " + action.Name
			if action.Prompt != "" {
				line += " — " + action.Prompt
			}
			if action.Optional {
				line += " (optional)"
			}
			fmt.Fprintln(os.Stderr, line)
		case "github-app":
			appLabel := action.AppName
			if appLabel == "" {
				appLabel = "GitHub App"
			}
			fmt.Fprintf(os.Stderr, "  ☐ Configure %s (variable: %s, secret: %s)\n",
				appLabel, action.AppIDVariable, action.PrivateKeySecret)
		case "copilot-auth":
			secret := action.Secret
			if secret == "" {
				secret = "COPILOT_GITHUB_TOKEN"
			}
			fmt.Fprintf(os.Stderr, "  ☐ Set Copilot PAT secret: %s\n", secret)
		case "handoff":
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(action.Message))
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run 'gh aw bootstrap --repo OWNER/REPO' to apply these steps interactively."))
	fmt.Fprintln(os.Stderr, "")
}

// executeBootstrapConfigForAdd runs the bootstrap config actions interactively.
// Used by add-wizard after the workflow PR has been created and merged.
func executeBootstrapConfigForAdd(ctx context.Context, repo string, sources []string, profile *resolvedBootstrapProfile, verbose bool) error {
	if profile == nil || profile.Profile == nil || len(profile.Profile.Config) == 0 {
		return nil
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying post-installation steps from "+profile.PackageID+"..."))

	return executeBootstrapProfile(ctx, bootstrapProfileRunConfig{
		Repo:    repo,
		Sources: sources,
		Profile: profile,
		Verbose: verbose,
	})
}
