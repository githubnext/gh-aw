package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

func runBootstrapCommitAndPushAction(ctx context.Context, repoDir string, action repositoryPackageBootstrapAction) error {
	if repoDir == "" {
		return errors.New("bootstrap commit-and-push requires a local checkout directory. Example: rerun from a git checkout and then rerun gh aw add from that checkout")
	}

	pending, err := bootstrapRepoHasPendingChanges(ctx, repoDir)
	if err != nil {
		return err
	}
	if !pending {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping commit and push because the local checkout is already clean."))
		return nil
	}

	if _, err := runBootstrapGitCommand(ctx, repoDir, "add", "-A"); err != nil {
		return err
	}
	if _, err := runBootstrapGitCommand(ctx, repoDir, "commit", "-m", action.Message); err != nil {
		return err
	}
	branch, err := getCurrentBranchIn(repoDir)
	if err != nil {
		return fmt.Errorf("failed to determine current branch for bootstrap commit-and-push: %w", err)
	}
	if _, err := runBootstrapGitCommand(ctx, repoDir, "push", "-u", "origin", branch); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Committed and pushed bootstrap changes"))
	return nil
}

func bootstrapRepoHasPendingChanges(ctx context.Context, repoDir string) (bool, error) {
	output, err := runBootstrapGitCommand(ctx, repoDir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func runBootstrapGitCommand(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("failed to run git %s in %s: %w\n%s", strings.Join(args, " "), repoDir, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}
