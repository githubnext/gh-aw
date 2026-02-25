package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var checksLog = logger.New("cli:checks_command")

// CheckState represents the normalized CI state for a PR.
type CheckState string

const (
	// CheckStateFailed indicates one or more checks failed.
	CheckStateFailed CheckState = "failed"
	// CheckStatePending indicates checks are still running.
	CheckStatePending CheckState = "pending"
	// CheckStateNoChecks indicates no checks have been configured or triggered.
	CheckStateNoChecks CheckState = "no_checks"
	// CheckStatePolicyBlocked indicates policy or account gates are blocking the PR.
	CheckStatePolicyBlocked CheckState = "policy_blocked"
	// CheckStateSuccess indicates all checks passed.
	CheckStateSuccess CheckState = "success"
)

// ChecksConfig holds configuration for the checks command.
type ChecksConfig struct {
	Repo       string
	PRNumber   string
	JSONOutput bool
}

// PRCheckRun represents a single check run from the GitHub API.
type PRCheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

// PRCommitStatus represents a single commit status from the GitHub API.
type PRCommitStatus struct {
	State       string `json:"state"`
	Description string `json:"description"`
	Context     string `json:"context"`
	TargetURL   string `json:"target_url"`
}

// ChecksResult is the normalized output for the checks command.
type ChecksResult struct {
	State      CheckState       `json:"state"`
	PRNumber   string           `json:"pr_number"`
	HeadSHA    string           `json:"head_sha"`
	CheckRuns  []PRCheckRun     `json:"check_runs"`
	Statuses   []PRCommitStatus `json:"statuses"`
	TotalCount int              `json:"total_count"`
}

// NewChecksCommand creates the checks command.
func NewChecksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checks <pr-number>",
		Short: "Classify CI check state for a pull request",
		Long: `Classify CI check state for a pull request and emit a normalized result.

Maps PR check rollups to one of the following normalized states:
  success        - all checks passed
  failed         - one or more checks failed
  pending        - checks are still running or queued
  no_checks      - no checks configured or triggered
  policy_blocked - policy or account gates are blocking the PR

` + "Raw check run and commit status signals are included in JSON output." + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` checks 42                    # Classify checks for PR #42
  ` + string(constants.CLIExtensionPrefix) + ` checks 42 --repo owner/repo  # Specify repository
  ` + string(constants.CLIExtensionPrefix) + ` checks 42 --json             # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			config := ChecksConfig{
				Repo:       repo,
				PRNumber:   args[0],
				JSONOutput: jsonOutput,
			}

			return RunChecks(config)
		},
	}

	addRepoFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}

// RunChecks executes the checks command with the given configuration.
func RunChecks(config ChecksConfig) error {
	checksLog.Printf("Running checks: pr=%s, repo=%s", config.PRNumber, config.Repo)

	result, err := FetchChecksResult(config.Repo, config.PRNumber)
	if err != nil {
		return err
	}

	if config.JSONOutput {
		return printChecksJSON(result)
	}

	return printChecksText(result)
}

// FetchChecksResult fetches check runs and statuses for a PR and returns a classified result.
// This function is exported for use in tests and other packages.
func FetchChecksResult(repoOverride string, prNumber string) (*ChecksResult, error) {
	checksLog.Printf("Fetching checks result: repo=%s, pr=%s", repoOverride, prNumber)

	// Step 1: Resolve head SHA from PR
	headSHA, err := fetchPRHeadSHA(repoOverride, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR head SHA: %w", err)
	}
	checksLog.Printf("Resolved head SHA: %s", headSHA)

	// Step 2: Fetch check runs
	checkRuns, err := fetchCheckRuns(repoOverride, headSHA)
	if err != nil {
		// Non-fatal: continue with empty check runs
		checksLog.Printf("Failed to fetch check runs: %v", err)
		checkRuns = []PRCheckRun{}
	}

	// Step 3: Fetch commit statuses
	statuses, err := fetchCommitStatuses(repoOverride, headSHA)
	if err != nil {
		// Non-fatal: continue with empty statuses
		checksLog.Printf("Failed to fetch commit statuses: %v", err)
		statuses = []PRCommitStatus{}
	}

	state := classifyCheckState(checkRuns, statuses)

	return &ChecksResult{
		State:      state,
		PRNumber:   prNumber,
		HeadSHA:    headSHA,
		CheckRuns:  checkRuns,
		Statuses:   statuses,
		TotalCount: len(checkRuns) + len(statuses),
	}, nil
}

// execGHAPI runs a gh api subcommand and returns raw output.
// repoOverride is appended as --repo if non-empty; context labels error messages.
func execGHAPI(repoOverride string, context string, args ...string) ([]byte, error) {
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return nil, classifyGHAPIError(exitErr.ExitCode(), stderr, context, repoOverride)
		}
		return nil, fmt.Errorf("gh api call failed: %w", err)
	}
	return output, nil
}

// fetchPRHeadSHA fetches the head commit SHA for a given PR.
func fetchPRHeadSHA(repoOverride string, prNumber string) (string, error) {
	output, err := execGHAPI(repoOverride, prNumber,
		"api", "repos/{owner}/{repo}/pulls/"+prNumber, "--jq", ".head.sha")
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", errors.New(console.FormatErrorWithSuggestions(
			"PR #"+prNumber+" returned an empty SHA",
			[]string{
				"Verify that PR #" + prNumber + " exists and is accessible",
				"Check that the --repo flag points to the correct repository",
			},
		))
	}
	return sha, nil
}

// classifyGHAPIError converts a gh API exit error into a user-friendly, pre-formatted error.
func classifyGHAPIError(exitCode int, stderr string, prNumber string, repo string) error {
	checksLog.Printf("API error: exitCode=%d, stderr=%s", exitCode, stderr)

	lower := strings.ToLower(stderr)

	switch {
	case strings.Contains(lower, "404") || strings.Contains(lower, "not found"):
		repoHint := "the current repository"
		if repo != "" {
			repoHint = repo
		}
		return errors.New(console.FormatErrorWithSuggestions(
			fmt.Sprintf("PR #%s not found in %s", prNumber, repoHint),
			[]string{
				"Verify that the pull request number is correct",
				"Use --repo owner/repo to specify the target repository explicitly",
				"Ensure you have read access to the repository",
			},
		))
	case strings.Contains(lower, "403") || strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "bad credentials") || strings.Contains(lower, "401") ||
		strings.Contains(lower, "unauthorized"):
		return errors.New(console.FormatErrorWithSuggestions(
			"GitHub API authentication failed",
			[]string{
				"Run 'gh auth login' to authenticate with GitHub",
				"Ensure your token has the 'repo' scope for private repositories",
				"Check that GH_TOKEN or GITHUB_TOKEN is set correctly if using environment variables",
			},
		))
	default:
		return fmt.Errorf("gh api call failed (exit %d): %s", exitCode, stderr)
	}
}

// checkRunsAPIResponse is the envelope returned by the check-runs endpoint.
type checkRunsAPIResponse struct {
	TotalCount int          `json:"total_count"`
	CheckRuns  []PRCheckRun `json:"check_runs"`
}

// fetchCheckRuns fetches check runs for a commit SHA.
func fetchCheckRuns(repoOverride string, sha string) ([]PRCheckRun, error) {
	output, err := execGHAPI(repoOverride, sha,
		"api", "repos/{owner}/{repo}/commits/"+sha+"/check-runs", "--paginate")
	if err != nil {
		return nil, err
	}

	var resp checkRunsAPIResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse check runs response: %w", err)
	}

	return resp.CheckRuns, nil
}

// commitStatusAPIResponse is the envelope returned by the statuses endpoint.
type commitStatusAPIResponse struct {
	State    string           `json:"state"`
	Statuses []PRCommitStatus `json:"statuses"`
}

// fetchCommitStatuses fetches commit statuses (legacy Status API) for a commit SHA.
func fetchCommitStatuses(repoOverride string, sha string) ([]PRCommitStatus, error) {
	output, err := execGHAPI(repoOverride, sha,
		"api", "repos/{owner}/{repo}/commits/"+sha+"/status")
	if err != nil {
		return nil, err
	}

	var resp commitStatusAPIResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse commit status response: %w", err)
	}

	return resp.Statuses, nil
}

// policyCheckPatterns are patterns that indicate a policy/account-gate check rather than a
// product failure. These names come from GitHub's branch-protection rule enforcement.
var policyCheckPatterns = []string{
	"required status check",
	"branch protection",
	"mergeability",
	"repo policy",
	"policy check",
	"access control",
}

// isPolicyCheck returns true if the check run name looks like a policy/account-gate check.
func isPolicyCheck(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range policyCheckPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// classifyCheckState derives a normalized CheckState from raw check runs and commit statuses.
func classifyCheckState(checkRuns []PRCheckRun, statuses []PRCommitStatus) CheckState {
	if len(checkRuns) == 0 && len(statuses) == 0 {
		return CheckStateNoChecks
	}

	hasPending := false
	hasFailed := false
	hasPolicyBlocked := false

	for _, cr := range checkRuns {
		switch cr.Status {
		case "queued", "in_progress", "waiting", "requested", "pending":
			hasPending = true
		case "completed":
			switch cr.Conclusion {
			case "failure", "timed_out", "startup_failure":
				if isPolicyCheck(cr.Name) {
					hasPolicyBlocked = true
				} else {
					hasFailed = true
				}
			case "action_required":
				hasPolicyBlocked = true
			}
		}
	}

	for _, s := range statuses {
		switch s.State {
		case "pending":
			hasPending = true
		case "failure", "error":
			if isPolicyCheck(s.Context) {
				hasPolicyBlocked = true
			} else {
				hasFailed = true
			}
		}
	}

	switch {
	case hasPolicyBlocked && !hasFailed && !hasPending:
		return CheckStatePolicyBlocked
	case hasFailed:
		return CheckStateFailed
	case hasPending:
		return CheckStatePending
	default:
		return CheckStateSuccess
	}
}

// printChecksJSON prints the result as JSON to stdout.
func printChecksJSON(result *ChecksResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}
	return nil
}

// printChecksText prints the result in human-readable form to stderr.
func printChecksText(result *ChecksResult) error {
	switch result.State {
	case CheckStateSuccess:
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("PR #%s: all checks passed (%d total)", result.PRNumber, result.TotalCount)))
	case CheckStateFailed:
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("PR #%s: checks failed (%d total)", result.PRNumber, result.TotalCount)))
	case CheckStatePending:
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("PR #%s: checks pending (%d total)", result.PRNumber, result.TotalCount)))
	case CheckStateNoChecks:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("PR #%s: no checks configured or triggered", result.PRNumber)))
	case CheckStatePolicyBlocked:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("PR #%s: blocked by policy or account gate (%d total)", result.PRNumber, result.TotalCount)))
	}

	// Always print the normalized state to stdout for machine consumption.
	fmt.Println(string(result.State))
	return nil
}
