package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var experimentsLog = logger.New("cli:experiments_command")

const (
	// experimentsBranchPrefix is the git branch prefix used to identify experiments.
	experimentsBranchPrefix = "experiments/"

	// experimentsStaleThresholdDays is the number of days after which an experiment
	// branch is considered stale.
	experimentsStaleThresholdDays = 30
)

// ExperimentInfo represents a single experiment for list output.
type ExperimentInfo struct {
	Name       string `json:"name" console:"header:Name"`
	Author     string `json:"author" console:"header:Author"`
	LastCommit string `json:"last_commit" console:"header:Last Commit"`
	Status     string `json:"status" console:"header:Status"`
	Branch     string `json:"branch" console:"header:Branch"`
}

// ExperimentCommit represents a single commit within an experiment.
type ExperimentCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// ExperimentPR represents a pull request associated with an experiment.
type ExperimentPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// ExperimentDetails represents detailed information about a specific experiment.
type ExperimentDetails struct {
	Name        string             `json:"name"`
	Branch      string             `json:"branch"`
	Author      string             `json:"author"`
	LastCommit  string             `json:"last_commit"`
	Status      string             `json:"status"`
	CommitCount int                `json:"commit_count"`
	Commits     []ExperimentCommit `json:"commits"`
	PRs         []ExperimentPR     `json:"prs"`
}

// ExperimentsListConfig holds configuration for the experiments list subcommand.
type ExperimentsListConfig struct {
	RepoOverride string
	JSONOutput   bool
}

// ExperimentsAnalyzeConfig holds configuration for the experiments analyze subcommand.
type ExperimentsAnalyzeConfig struct {
	ExperimentName string
	RepoOverride   string
	JSONOutput     bool
}

// NewExperimentsCommand creates the experiments command with its subcommands.
func NewExperimentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "experiments",
		Hidden: true,
		Short:  "Explore ongoing experiments in the repository",
		Long: `Explore ongoing experiments in the repository.

Experiments are tracked via git branches with the "experiments/" prefix (e.g.,
experiments/my-feature). This command helps discover, list, and analyze those branches.

Available subcommands:
  • list    - List all experiment branches (default)
  • analyze - Analyze a specific experiment in detail

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` experiments                        # List all experiments (default)
  ` + string(constants.CLIExtensionPrefix) + ` experiments list                   # List all experiments
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --json            # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-feature     # Analyze experiments/my-feature
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-feature --json  # Analyze in JSON format`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsList(ExperimentsListConfig{
				RepoOverride: repoOverride,
				JSONOutput:   jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	cmd.AddCommand(NewExperimentsListSubcommand())
	cmd.AddCommand(NewExperimentsAnalyzeSubcommand())

	return cmd
}

// NewExperimentsListSubcommand creates the experiments list subcommand.
func NewExperimentsListSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all experiment branches",
		Long: `List all experiment branches in the repository.

Experiments are identified by git branches with the "experiments/" prefix.
For each experiment, shows the name, author of the last commit, date of the
last commit, and status (active or stale).

An experiment is considered "active" if its last commit was within the past
30 days, and "stale" otherwise.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` experiments list                             # List all experiments
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --json                      # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --repo owner/repo           # List from a specific repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsList(ExperimentsListConfig{
				RepoOverride: repoOverride,
				JSONOutput:   jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	return cmd
}

// NewExperimentsAnalyzeSubcommand creates the experiments analyze subcommand.
func NewExperimentsAnalyzeSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <experiment>",
		Short: "Analyze a specific experiment in detail",
		Long: `Analyze a specific experiment branch in detail.

The experiment argument is the name of the experiment without the "experiments/"
prefix (e.g., "my-feature" for the "experiments/my-feature" branch).

Shows recent commits on the experiment branch, associated pull requests (both
open and closed), and a status summary.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-feature              # Analyze experiments/my-feature
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-feature --json       # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-feature --repo owner/repo  # Analyze in a specific repository`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsAnalyze(ExperimentsAnalyzeConfig{
				ExperimentName: args[0],
				RepoOverride:   repoOverride,
				JSONOutput:     jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	return cmd
}

// RunExperimentsList lists all experiment branches.
func RunExperimentsList(config ExperimentsListConfig) error {
	experimentsLog.Printf("Listing experiments: repo=%s, json=%v", config.RepoOverride, config.JSONOutput)

	var experiments []ExperimentInfo
	var err error

	if config.RepoOverride != "" {
		experiments, err = fetchRemoteExperiments(config.RepoOverride)
	} else {
		experiments, err = fetchLocalExperiments()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	if config.JSONOutput {
		jsonBytes, err := json.MarshalIndent(experiments, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	if len(experiments) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No experiment branches found (branches matching experiments/* pattern)."))
		return nil
	}

	count := len(experiments)
	if count == 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found 1 experiment"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d experiments", count)))
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(experiments))

	return nil
}

// RunExperimentsAnalyze analyzes a specific experiment branch.
func RunExperimentsAnalyze(config ExperimentsAnalyzeConfig) error {
	experimentsLog.Printf("Analyzing experiment: name=%s, repo=%s, json=%v",
		config.ExperimentName, config.RepoOverride, config.JSONOutput)

	branchName := experimentsBranchPrefix + config.ExperimentName

	var details *ExperimentDetails
	var err error

	if config.RepoOverride != "" {
		details, err = fetchRemoteExperimentDetails(config.RepoOverride, branchName, config.ExperimentName)
	} else {
		details, err = fetchLocalExperimentDetails(branchName, config.ExperimentName)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	if config.JSONOutput {
		jsonBytes, err := json.MarshalIndent(details, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	printExperimentDetails(details)
	return nil
}

// fetchLocalExperiments lists experiment branches from the local git repository.
func fetchLocalExperiments() ([]ExperimentInfo, error) {
	experimentsLog.Print("Fetching local experiment branches via git for-each-ref")

	cmd := exec.Command("git", "for-each-ref",
		"--sort=-committerdate",
		"--format=%(refname:short)|%(authorname)|%(committerdate:format:%Y-%m-%d)|%(subject)",
		"refs/remotes/origin/"+experimentsBranchPrefix+"*",
		"refs/heads/"+experimentsBranchPrefix+"*",
	)
	output, err := cmd.Output()
	if err != nil {
		// Not a fatal error if there are simply no branches or no remote named origin
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return []ExperimentInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list experiment branches: %w", err)
	}

	return parseForEachRefOutput(string(output)), nil
}

// parseForEachRefOutput parses the output of git for-each-ref into ExperimentInfo slice.
func parseForEachRefOutput(output string) []ExperimentInfo {
	seen := make(map[string]bool)
	var experiments []ExperimentInfo

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}

		fullRef := parts[0] // e.g. origin/experiments/my-feature or experiments/my-feature
		author := parts[1]
		dateStr := parts[2]

		// Extract experiment name from branch ref
		name := extractExperimentName(fullRef)
		if name == "" {
			continue
		}

		// Deduplicate: prefer remote (origin/) over local branch
		if seen[name] {
			continue
		}
		seen[name] = true

		branch := experimentsBranchPrefix + name
		status := computeExperimentStatus(dateStr)

		experiments = append(experiments, ExperimentInfo{
			Name:       name,
			Author:     author,
			LastCommit: dateStr,
			Status:     status,
			Branch:     branch,
		})
	}

	return experiments
}

// extractExperimentName extracts the experiment name from a branch ref.
// e.g. "origin/experiments/my-feature" -> "my-feature"
//
//	"experiments/my-feature"        -> "my-feature"
func extractExperimentName(ref string) string {
	// Strip remote prefix (origin/)
	ref = strings.TrimPrefix(ref, "origin/")

	if !strings.HasPrefix(ref, experimentsBranchPrefix) {
		return ""
	}
	return strings.TrimPrefix(ref, experimentsBranchPrefix)
}

// computeExperimentStatus returns "active" or "stale" based on the last commit date string.
func computeExperimentStatus(dateStr string) string {
	if dateStr == "" {
		return "unknown"
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "unknown"
	}
	if time.Since(t) <= time.Duration(experimentsStaleThresholdDays)*24*time.Hour {
		return "active"
	}
	return "stale"
}

// fetchRemoteExperiments lists experiment branches from a remote repository via GitHub API.
func fetchRemoteExperiments(repoOverride string) ([]ExperimentInfo, error) {
	experimentsLog.Printf("Fetching remote experiment branches: repo=%s", repoOverride)

	// Fetch all branches from the repository; GitHub API doesn't support prefix filters
	args := []string{"api", "repos/{owner}/{repo}/branches",
		"--paginate",
		"--jq", "[.[] | select(.name | startswith(\"" + experimentsBranchPrefix + "\")) | {name: .name, sha: .commit.sha}]",
		"--repo", repoOverride,
	}
	cmd := workflow.ExecGH(args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to fetch branches (exit %d): %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to fetch branches: %w", err)
	}

	// Parse paged JSON output — gh api --paginate emits one JSON array per page
	type branchRef struct {
		Name string `json:"name"`
		SHA  string `json:"sha"`
	}
	allBranches, err := parsePagedJSONArray[branchRef](string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse branch list: %w", err)
	}

	var experiments []ExperimentInfo
	for _, b := range allBranches {
		name := strings.TrimPrefix(b.Name, experimentsBranchPrefix)

		// Fetch commit details for this branch
		commitDate, commitAuthor := fetchCommitDetails(repoOverride, b.SHA)

		experiments = append(experiments, ExperimentInfo{
			Name:       name,
			Author:     commitAuthor,
			LastCommit: commitDate,
			Status:     computeExperimentStatus(commitDate),
			Branch:     b.Name,
		})
	}

	return experiments, nil
}

// fetchCommitDetails fetches the date and author of a commit by SHA.
func fetchCommitDetails(repoOverride, sha string) (date, author string) {
	if sha == "" {
		return "", ""
	}
	args := []string{"api", "repos/{owner}/{repo}/commits/" + sha,
		"--jq", ".commit.author.date[:10] + \"|\" + .commit.author.name",
		"--repo", repoOverride,
	}
	cmd := workflow.ExecGH(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// fetchLocalExperimentDetails fetches detailed information about a specific experiment branch
// from the local git repository.
func fetchLocalExperimentDetails(branchName, experimentName string) (*ExperimentDetails, error) {
	experimentsLog.Printf("Fetching local experiment details: branch=%s", branchName)

	// Resolve full ref: prefer remote, fall back to local branch
	remoteBranch := "origin/" + branchName
	if !gitRefExists(remoteBranch) {
		if !gitRefExists(branchName) {
			return nil, fmt.Errorf("experiment branch %q not found locally (tried %s and %s)",
				experimentName, remoteBranch, branchName)
		}
		remoteBranch = branchName
	}

	// Fetch recent commits on the branch
	commits, err := fetchLocalCommits(remoteBranch)
	if err != nil {
		return nil, err
	}

	// Determine status from most recent commit date
	status := "unknown"
	lastCommit := ""
	author := ""
	if len(commits) > 0 {
		lastCommit = commits[0].Date
		author = commits[0].Author
		status = computeExperimentStatus(lastCommit)
	}

	// Fetch associated pull requests via gh CLI
	prs, err := fetchPRsForBranch(branchName, "")
	if err != nil {
		experimentsLog.Printf("Failed to fetch PRs (non-fatal): %v", err)
		prs = []ExperimentPR{}
	}

	return &ExperimentDetails{
		Name:        experimentName,
		Branch:      branchName,
		Author:      author,
		LastCommit:  lastCommit,
		Status:      status,
		CommitCount: len(commits),
		Commits:     commits,
		PRs:         prs,
	}, nil
}

// fetchRemoteExperimentDetails fetches detailed information about a specific experiment
// from a remote repository via GitHub API.
func fetchRemoteExperimentDetails(repoOverride, branchName, experimentName string) (*ExperimentDetails, error) {
	experimentsLog.Printf("Fetching remote experiment details: repo=%s, branch=%s", repoOverride, branchName)

	// Fetch branch info
	encodedBranch := strings.ReplaceAll(branchName, "/", "%2F")
	args := []string{"api",
		"repos/{owner}/{repo}/branches/" + encodedBranch,
		"--repo", repoOverride,
	}
	cmd := workflow.ExecGH(args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "404") || strings.Contains(stderr, "not found") {
				return nil, fmt.Errorf("experiment %q not found in %s", experimentName, repoOverride)
			}
			return nil, fmt.Errorf("failed to fetch experiment branch (exit %d): %s", exitErr.ExitCode(), stderr)
		}
		return nil, fmt.Errorf("failed to fetch experiment branch: %w", err)
	}

	type remoteBranchInfo struct {
		Commit struct {
			SHA    string `json:"sha"`
			Commit struct {
				Author struct {
					Name string `json:"name"`
					Date string `json:"date"`
				} `json:"author"`
			} `json:"commit"`
		} `json:"commit"`
	}
	var branchInfo remoteBranchInfo
	if err := json.Unmarshal(out, &branchInfo); err != nil {
		return nil, fmt.Errorf("failed to parse branch info: %w", err)
	}

	author := branchInfo.Commit.Commit.Author.Name
	lastCommit := ""
	if branchInfo.Commit.Commit.Author.Date != "" {
		lastCommit = branchInfo.Commit.Commit.Author.Date[:10]
	}
	status := computeExperimentStatus(lastCommit)

	// Fetch recent commits via GitHub API
	commits, err := fetchRemoteCommits(repoOverride, branchName)
	if err != nil {
		experimentsLog.Printf("Failed to fetch commits (non-fatal): %v", err)
		commits = []ExperimentCommit{}
	}

	// Fetch associated pull requests
	prs, err := fetchPRsForBranch(branchName, repoOverride)
	if err != nil {
		experimentsLog.Printf("Failed to fetch PRs (non-fatal): %v", err)
		prs = []ExperimentPR{}
	}

	return &ExperimentDetails{
		Name:        experimentName,
		Branch:      branchName,
		Author:      author,
		LastCommit:  lastCommit,
		Status:      status,
		CommitCount: len(commits),
		Commits:     commits,
		PRs:         prs,
	}, nil
}

// gitRefExists checks if a git ref exists locally.
func gitRefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}

// fetchLocalCommits fetches recent commits on a branch using git log.
func fetchLocalCommits(branch string) ([]ExperimentCommit, error) {
	// Limit to 10 most recent commits for readability
	cmd := exec.Command("git", "log", branch,
		"--max-count=10",
		"--format=%H|%an|%cd|%s",
		"--date=format:%Y-%m-%d",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits for branch %q: %w", branch, err)
	}

	var commits []ExperimentCommit
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, ExperimentCommit{
			SHA:     parts[0][:min(7, len(parts[0]))],
			Author:  parts[1],
			Date:    parts[2],
			Message: parts[3],
		})
	}
	return commits, nil
}

// fetchRemoteCommits fetches recent commits from a remote repository via GitHub API.
func fetchRemoteCommits(repoOverride, branchName string) ([]ExperimentCommit, error) {
	args := []string{"api",
		"repos/{owner}/{repo}/commits",
		"--field", "sha=" + branchName,
		"--field", "per_page=10",
		"--jq", `[.[] | {sha: .sha[:7], author: .commit.author.name, date: .commit.author.date[:10], message: (.commit.message | split("\n")[0])}]`,
		"--repo", repoOverride,
	}
	cmd := workflow.ExecGH(args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits: %w", err)
	}

	var commits []ExperimentCommit
	if err := json.Unmarshal(out, &commits); err != nil {
		return nil, fmt.Errorf("failed to parse commits: %w", err)
	}
	return commits, nil
}

// fetchPRsForBranch fetches pull requests associated with the given branch.
func fetchPRsForBranch(branchName, repoOverride string) ([]ExperimentPR, error) {
	// Search for PRs with head or base matching the experiment branch
	args := []string{"pr", "list",
		"--head", branchName,
		"--state", "all",
		"--json", "number,title,state,url",
		"--limit", "20",
	}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	cmd := workflow.ExecGH(args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	type ghPR struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		URL    string `json:"url"`
	}
	var ghPRs []ghPR
	if err := json.Unmarshal(out, &ghPRs); err != nil {
		return nil, fmt.Errorf("failed to parse PRs: %w", err)
	}

	prs := make([]ExperimentPR, 0, len(ghPRs))
	for _, p := range ghPRs {
		prs = append(prs, ExperimentPR(p))
	}
	return prs, nil
}

// printExperimentDetails renders the experiment details to stderr in human-readable form.
func printExperimentDetails(d *ExperimentDetails) {
	var statusIcon string
	switch d.Status {
	case "active":
		statusIcon = "✓"
	case "stale":
		statusIcon = "⚠"
	default:
		statusIcon = "●"
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Experiment: %s (%s %s)", d.Name, statusIcon, d.Status),
	))
	fmt.Fprintf(os.Stderr, "  Branch:      %s\n", d.Branch)
	fmt.Fprintf(os.Stderr, "  Author:      %s\n", d.Author)
	fmt.Fprintf(os.Stderr, "  Last Commit: %s\n", d.LastCommit)
	fmt.Fprintf(os.Stderr, "  Commits:     %d\n", d.CommitCount)

	if len(d.Commits) > 0 {
		fmt.Fprintln(os.Stderr, "\nRecent commits:")
		for _, c := range d.Commits {
			fmt.Fprintf(os.Stderr, "  %s  %s  %s  %s\n", c.SHA, c.Date, c.Author, c.Message)
		}
	}

	if len(d.PRs) > 0 {
		fmt.Fprintln(os.Stderr, "\nPull requests:")
		for _, pr := range d.PRs {
			fmt.Fprintf(os.Stderr, "  #%d [%s] %s\n", pr.Number, pr.State, pr.Title)
			fmt.Fprintf(os.Stderr, "     %s\n", pr.URL)
		}
	} else {
		fmt.Fprintln(os.Stderr, "\nNo pull requests found for this branch.")
	}
}

// parsePagedJSONArray parses multiple JSON arrays (one per page from --paginate)
// concatenated in the output and returns a merged slice.
func parsePagedJSONArray[T any](output string) ([]T, error) {
	var result []T
	decoder := json.NewDecoder(strings.NewReader(output))
	for decoder.More() {
		var page []T
		if err := decoder.Decode(&page); err != nil {
			return nil, err
		}
		result = append(result, page...)
	}
	return result, nil
}
