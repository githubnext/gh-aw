package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

// orgUpdateLog logs org-mode update progress and rate-limit telemetry for debugging.
var orgUpdateLog = logger.New("cli:update_org")

// orgUpdateCoreBuffer preserves a 500-request safety margin on the core API budget
// before org-mode applies a delay to avoid exhausting the hourly quota mid-run.
const orgUpdateCoreBuffer = 500

// orgUpdateSearchBuffer preserves at least one search request because GitHub's search
// quota is much smaller and org discovery only needs a narrow cushion between pages.
const orgUpdateSearchBuffer = 1

// orgUpdateCriticalConsumed is the number of core API requests that, once consumed,
// marks the budget as critical (i.e. remaining <= limit-1000). At that point org-mode
// stops processing additional repositories instead of risking quota exhaustion or a
// long wait for the hourly reset.
const orgUpdateCriticalConsumed = 1000

// errOrgRateLimitCritical signals that the GitHub API budget reached a critical level
// and remaining work should be short-circuited rather than waiting for the reset.
var errOrgRateLimitCritical = errors.New("github api rate limit reached critical level")

var previewOrgRepoUpdatesFn = previewOrgRepoUpdates
var runUpdateForTargetRepoFn = runUpdateForTargetRepo
var waitForOrgRateLimitFn = waitForOrgRateLimit
var createIssueForOrgRepoFn = createIssueForOrgRepo

type orgRateLimitResponse struct {
	Resources struct {
		Core   rateLimitResource `json:"core"`
		Search rateLimitResource `json:"search"`
	} `json:"resources"`
}

type orgWorkflowPreview struct {
	Name       string
	Path       string
	CurrentRef string
	LatestRef  string
	Redirected bool
	EditedAt   time.Time
}

type orgRepoPreview struct {
	Repo           string
	TotalWorkflows int
	Workflows      []orgWorkflowPreview
	OldestEdit     time.Time
}

func runUpdateForOrg(ctx context.Context, org string, repoGlobs []string, opts UpdateWorkflowsOptions, createPR bool, createIssue bool, verbose bool) error {
	clearUpdateResolutionCaches()

	if strings.TrimSpace(org) == "" {
		return errors.New("--org cannot be empty")
	}
	if err := validateRepoGlobs(repoGlobs); err != nil {
		return err
	}

	// Handle Ctrl-C / SIGTERM so an interrupted run still renders the report it
	// gathered so far instead of exiting abruptly.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Discovering repositories in "+org+" with source-managed workflows..."))
	repoPaths, err := searchOrgWorkflowReposFn(ctx, org, verbose)
	if err != nil {
		return err
	}

	if len(repoPaths) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories found with source-managed workflows"))
		return nil
	}

	repos := filterOrgRepos(repoPaths, repoGlobs)
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories matched the requested --repos filters"))
		return nil
	}

	total := len(repos)
	orgUpdateLog.Printf("Previewing updates for %d repositories in %s", total, org)

	previewByRepo := make([]orgRepoPreview, 0, len(repos))
	stopped := false
	for i, repo := range repos {
		// Honor a cancellation signal between repositories so we can still show
		// the report for the work completed so far.
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Cancellation requested; stopping after %d/%d repositories", i, total)))
			orgUpdateLog.Printf("Context canceled during preview at repo %d/%d: %v", i, total, ctx.Err())
			stopped = true
			break
		}

		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("[%d/%d] Inspecting %s", i+1, total, repo)))

		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil {
			if errors.Is(err, errOrgRateLimitCritical) {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("GitHub API budget critical; stopping after %d/%d repositories and reporting what was found", i, total)))
				orgUpdateLog.Printf("Rate limit critical during preview at repo %d/%d", i, total)
				stopped = true
				break
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo, err)))
			}
		}

		preview, err := previewOrgRepoUpdatesFn(ctx, repo, opts, verbose)
		if err != nil {
			// A single repository failing (parse error, transient API issue, etc.)
			// must not abort the whole org run; log it and keep going.
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", repo, err)))
			orgUpdateLog.Printf("Failed to preview updates for %s: %v", repo, err)
			continue
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("%s: %d workflow(s), %d with updates", repo, preview.TotalWorkflows, len(preview.Workflows)),
		))
		if len(preview.Workflows) == 0 {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Skipping "+repo+": already up to date"))
			}
			continue
		}
		previewByRepo = append(previewByRepo, preview)
	}

	if len(previewByRepo) == 0 {
		if stopped {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No updates found before processing stopped"))
			return nil
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("All matching repositories are already up to date"))
		return nil
	}

	slices.SortStableFunc(previewByRepo, func(a, b orgRepoPreview) int {
		if a.OldestEdit.IsZero() && b.OldestEdit.IsZero() {
			return strings.Compare(a.Repo, b.Repo)
		}
		if a.OldestEdit.IsZero() {
			return 1
		}
		if b.OldestEdit.IsZero() {
			return -1
		}
		if a.OldestEdit.Equal(b.OldestEdit) {
			return strings.Compare(a.Repo, b.Repo)
		}
		if a.OldestEdit.Before(b.OldestEdit) {
			return -1
		}
		return 1
	})

	// Always render the report of pending updates before applying anything; it is
	// cheap to compute and lets the user see results even if the run was stopped
	// early by a cancellation or a critical rate-limit condition.
	renderOrgPreviewReport(previewByRepo, createPR || createIssue)

	if !createPR && !createIssue {
		return nil
	}

	if createIssue {
		processed := 0
		for i, repo := range previewByRepo {
			if ctx.Err() != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Cancellation requested; created issues for %d/%d repositories", processed, len(previewByRepo))))
				orgUpdateLog.Printf("Context canceled during issue creation at %d/%d: %v", i, len(previewByRepo), ctx.Err())
				return nil
			}
			if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil {
				if errors.Is(err, errOrgRateLimitCritical) {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("GitHub API budget critical; created issues for %d/%d repositories", processed, len(previewByRepo))))
					orgUpdateLog.Printf("Rate limit critical during issue creation at %d/%d", i, len(previewByRepo))
					return nil
				}
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo.Repo, err)))
				}
			}
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("[%d/%d] Creating issue in %s", i+1, len(previewByRepo), repo.Repo)))
			if err := createIssueForOrgRepoFn(ctx, repo, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", repo.Repo, err)))
				orgUpdateLog.Printf("Failed to create issue in %s: %v", repo.Repo, err)
				continue
			}
			processed++
		}
		if processed == 0 {
			return errors.New("failed to create issues in any repository")
		}
		return nil
	}

	processed := 0
	for i, repo := range previewByRepo {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Cancellation requested; updated %d/%d repositories", processed, len(previewByRepo))))
			orgUpdateLog.Printf("Context canceled during update at %d/%d: %v", i, len(previewByRepo), ctx.Err())
			return nil
		}
		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil {
			if errors.Is(err, errOrgRateLimitCritical) {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("GitHub API budget critical; updated %d/%d repositories", processed, len(previewByRepo))))
				orgUpdateLog.Printf("Rate limit critical during update at %d/%d", i, len(previewByRepo))
				return nil
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo.Repo, err)))
			}
		}
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("[%d/%d] Updating %s", i+1, len(previewByRepo), repo.Repo)))
		if err := runUpdateForTargetRepoFn(ctx, repo.Repo, opts, true, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", repo.Repo, err)))
			orgUpdateLog.Printf("Failed to update %s: %v", repo.Repo, err)
			continue
		}
		processed++
	}
	if processed == 0 {
		return errors.New("failed to update any repository")
	}

	return nil
}

// renderOrgPreviewReport prints the discovered updates for each repository. It is
// intentionally cheap (no API calls) so it can be shown even when a run is stopped
// early by a cancellation signal or a critical rate-limit condition.
func renderOrgPreviewReport(previewByRepo []orgRepoPreview, applying bool) {
	if applying {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Repositories with updates available (%d):", len(previewByRepo))))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry-run preview of update pull requests:"))
	}
	for _, repo := range previewByRepo {
		fmt.Fprintf(os.Stderr, "- %s (%d workflow(s))\n", repo.Repo, repo.TotalWorkflows)
		for _, wf := range repo.Workflows {
			fmt.Fprintf(os.Stderr, "  - %s: %s -> %s\n", wf.Name, shortRef(wf.CurrentRef), shortRef(wf.LatestRef))
		}
	}
}

func previewOrgRepoUpdates(ctx context.Context, repo string, opts UpdateWorkflowsOptions, verbose bool) (orgRepoPreview, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return orgRepoPreview{}, fmt.Errorf("--org requires running inside a git repository: %w", err)
	}

	updatesDir, err := ensureUpdateTargetRepoGitignore(gitRoot)
	if err != nil {
		return orgRepoPreview{}, err
	}

	checkoutDir := filepath.Join(updatesDir, sanitizeRepoPath(repo))
	if err := shallowCloneTargetRepo(ctx, repo, checkoutDir); err != nil {
		return orgRepoPreview{}, err
	}

	workflowsDir := filepath.Join(checkoutDir, constants.GetWorkflowDir())
	workflows, err := findWorkflowsWithSource(workflowsDir, nil, verbose)
	if err != nil {
		return orgRepoPreview{}, fmt.Errorf("failed to scan workflows in shallow checkout: %w", err)
	}

	preview := orgRepoPreview{
		Repo:           repo,
		TotalWorkflows: len(workflows),
		Workflows:      make([]orgWorkflowPreview, 0, len(workflows)),
	}
	for _, wf := range workflows {
		sourceSpec, err := parseSourceSpec(wf.SourceSpec)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s/%s: failed to parse source: %v", repo, wf.Name, err)))
			orgUpdateLog.Printf("Failed to parse source for %s/%s: %v", repo, wf.Name, err)
			continue
		}
		name := normalizeWorkflowID(wf.Name)
		resolved, err := resolveRedirectedUpdateLocation(ctx, name, sourceSpec, opts.AllowMajor, verbose, opts.NoRedirect, opts.CoolDown)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s/%s: %v", repo, wf.Name, err)))
			orgUpdateLog.Printf("Failed to resolve update location for %s/%s: %v", repo, wf.Name, err)
			continue
		}
		if resolved.currentRef == resolved.latestRef && len(resolved.redirectHistory) == 0 && !opts.Force {
			continue
		}

		editedAt, err := getLatestWorkflowEditTimeFromCheckout(ctx, checkoutDir, wf.Path)
		if err == nil {
			if preview.OldestEdit.IsZero() || editedAt.Before(preview.OldestEdit) {
				preview.OldestEdit = editedAt
			}
		}
		preview.Workflows = append(preview.Workflows, orgWorkflowPreview{
			Name:       name,
			Path:       wf.Path,
			CurrentRef: resolved.currentRef,
			LatestRef:  resolved.latestRef,
			Redirected: len(resolved.redirectHistory) > 0,
			EditedAt:   editedAt,
		})
	}

	return preview, nil
}

func getLatestWorkflowEditTimeFromCheckout(ctx context.Context, checkoutDir, workflowPath string) (time.Time, error) {
	relativePath := workflowPath
	if filepath.IsAbs(workflowPath) {
		rel, err := filepath.Rel(checkoutDir, workflowPath)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to derive workflow path relative to checkout %s for %s: %w", checkoutDir, workflowPath, err)
		}
		relativePath = rel
	}

	cmd := exec.CommandContext(ctx, "git", "-C", checkoutDir, "log", "--max-count=1", "--format=%cI", "--", relativePath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read latest commit time for %s in checkout %s: %w", relativePath, checkoutDir, err)
	}
	date := strings.TrimSpace(string(output))
	if date == "" {
		return time.Time{}, fmt.Errorf("no commit date available for %s", workflowPath)
	}
	return time.Parse(time.RFC3339, date)
}

func waitForOrgRateLimit(ctx context.Context, resource string, verbose bool) error {
	output, err := workflow.RunGHContext(ctx, "Checking rate limit...", "api", "rate_limit")
	if err != nil {
		return err
	}

	var response orgRateLimitResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return err
	}

	limit := response.Resources.Core
	buffer := orgUpdateCoreBuffer
	if resource == "search" {
		limit = response.Resources.Search
		buffer = orgUpdateSearchBuffer
	}
	if limit.Limit == 0 {
		return nil
	}

	orgUpdateLog.Printf("GitHub %s rate limit: %d/%d remaining (reset at %s)", resource, limit.Remaining, limit.Limit, time.Unix(limit.Reset, 0).Format(time.RFC3339))

	// Critical level: once consumption reaches limit-1000 API units, stop processing
	// additional work rather than risking quota exhaustion or a long reset wait. The
	// search resource has a tiny quota, so the critical threshold only applies to core.
	if resource != "search" {
		criticalRemaining := limit.Limit - orgUpdateCriticalConsumed
		if criticalRemaining > buffer && limit.Remaining <= criticalRemaining {
			orgUpdateLog.Printf("GitHub %s rate limit critical: %d/%d remaining (<= %d)", resource, limit.Remaining, limit.Limit, criticalRemaining)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("GitHub %s API budget critical: %d/%d remaining (consumed %d API units)", resource, limit.Remaining, limit.Limit, limit.Limit-limit.Remaining),
			))
			return errOrgRateLimitCritical
		}
	}

	if limit.Remaining <= buffer {
		resetAt := time.Unix(limit.Reset, 0)
		waitFor := time.Until(resetAt) + rateLimitResetBuffer
		if waitFor > 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Applying a %s delay to avoid reaching the GitHub %s rate limit (%d/%d remaining)", waitFor.Round(time.Second), resource, limit.Remaining, limit.Limit),
			))
			timer := time.NewTimer(waitFor)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
			return nil
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(
			fmt.Sprintf("GitHub %s rate limit OK: %d/%d remaining", resource, limit.Remaining, limit.Limit),
		))
	}
	return nil
}

// createIssueForOrgRepo opens a GitHub issue in the target repository listing
// the source-managed workflows that have updates available. The issue title and
// body are formatted so maintainers can act on the report without running
// gh aw locally first.
func createIssueForOrgRepo(ctx context.Context, preview orgRepoPreview, verbose bool) error {
	title, body := buildOrgUpdateIssue(preview)

	endpoint := fmt.Sprintf("/repos/%s/issues", preview.Repo)
	_, err := workflow.RunGHContext(ctx, "Creating issue...",
		"api",
		"--method", "POST",
		endpoint,
		"-f", "title="+title,
		"-f", "body="+body,
	)
	if err != nil {
		return fmt.Errorf("failed to create issue in %s: %w", preview.Repo, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created issue in "+preview.Repo))
	return nil
}

func buildOrgUpdateIssue(preview orgRepoPreview) (string, string) {
	title := "[aw] Updates available"

	var body strings.Builder
	body.WriteString("## Agentic Workflows Update Available\n\n")
	body.WriteString("The `gh aw update` command found source-managed workflow updates for this repository.\n\n")
	body.WriteString("**Workflows with updates:**\n\n")
	for _, wf := range preview.Workflows {
		fmt.Fprintf(&body, "- `%s`: `%s` -> `%s`\n", wf.Name, shortRef(wf.CurrentRef), shortRef(wf.LatestRef))
	}
	body.WriteString("\n### How to apply\n\n")
	body.WriteString("- **Via @copilot**: Add a comment `@copilot update agentic workflows` on this issue\n")
	body.WriteString("- **Via CLI**: Run `gh aw update` in your local checkout\n")

	return title, body.String()
}
