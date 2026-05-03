package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var experimentsLog = logger.New("cli:experiments_command")

// experimentsBranchPrefix is the git branch prefix used to identify experiment state branches.
const experimentsBranchPrefix = "experiments/"

// ExperimentState represents the state.json format stored in experiments/* branches.
// This matches the format written by pick_experiment.cjs.
type ExperimentState struct {
	Counts map[string]map[string]int `json:"counts"` // experiment name → variant → count
	Runs   []ExperimentRunRecord     `json:"runs,omitempty"`
}

// ExperimentRunRecord represents a single workflow run in the state history.
type ExperimentRunRecord struct {
	RunID       string            `json:"run_id"`
	Timestamp   string            `json:"timestamp"`
	Assignments map[string]string `json:"assignments"`
}

// ExperimentVariantStats holds counts for all variants of one named A/B experiment.
type ExperimentVariantStats struct {
	Name     string         `json:"name"`
	Variants map[string]int `json:"variants"` // variant → count
	Total    int            `json:"total"`
}

// ExperimentInfo represents a single experiment workflow for list output.
type ExperimentInfo struct {
	WorkflowID  string `json:"workflow_id" console:"header:Workflow"`
	Branch      string `json:"branch" console:"header:Branch"`
	Experiments int    `json:"experiments" console:"header:Experiments"`
	TotalRuns   int    `json:"total_runs" console:"header:Total Runs"`
	LastRun     string `json:"last_run" console:"header:Last Run"`
}

// ExperimentDetails represents detailed information about a specific experiment workflow.
type ExperimentDetails struct {
	WorkflowID  string                   `json:"workflow_id"`
	Branch      string                   `json:"branch"`
	TotalRuns   int                      `json:"total_runs"`
	Experiments []ExperimentVariantStats `json:"experiments"`
	RecentRuns  []ExperimentRunRecord    `json:"recent_runs,omitempty"`
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
experiments/my-workflow). Each branch stores a state.json file written by the
workflow's pick_experiment step, containing variant counts and run history.

Available subcommands:
  • list    - List all experiment workflow branches (default)
  • analyze - Analyze a specific experiment workflow in detail

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` experiments                        # List all experiments (default)
  ` + string(constants.CLIExtensionPrefix) + ` experiments list                   # List all experiments
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --json            # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow    # Analyze experiments/my-workflow
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --json  # Analyze in JSON format`,
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
		Short: "List all experiment workflow branches",
		Long: `List all experiment workflow branches in the repository.

Reads the state.json file from each experiments/* branch and shows a summary
of each workflow's A/B experiments: number of experiments defined, total runs,
and timestamp of the most recent run.

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
		Short: "Analyze a specific experiment workflow in detail",
		Long: `Analyze a specific experiment workflow in detail.

The experiment argument is the workflow ID (branch name without the "experiments/"
prefix, e.g. "my-workflow" for the "experiments/my-workflow" branch).

Reads the state.json file from the branch and shows per-variant counts, total
runs, and the most recent run assignments.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow              # Analyze experiments/my-workflow
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --json       # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --repo owner/repo  # Analyze in a specific repository`,
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No experiment workflow branches found (branches matching experiments/* pattern)."))
		return nil
	}

	count := len(experiments)
	if count == 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found 1 experiment workflow"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d experiment workflows", count)))
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

// fetchLocalExperiments lists experiment branches and reads their state from the local git repo.
func fetchLocalExperiments() ([]ExperimentInfo, error) {
	experimentsLog.Print("Fetching local experiment branches via git for-each-ref")

	cmd := exec.Command("git", "for-each-ref",
		"--sort=-committerdate",
		"--format=%(refname:short)",
		"refs/remotes/origin/"+experimentsBranchPrefix+"*",
		"refs/heads/"+experimentsBranchPrefix+"*",
	)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return []ExperimentInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list experiment branches: %w", err)
	}

	seen := make(map[string]bool)
	var experiments []ExperimentInfo

	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		workflowID := extractExperimentName(line)
		if workflowID == "" || seen[workflowID] {
			continue
		}
		seen[workflowID] = true

		branchName := experimentsBranchPrefix + workflowID
		// Prefer remote ref; fall back to local.
		ref := "origin/" + branchName
		if !gitRefExists(ref) {
			ref = branchName
		}
		state := readLocalExperimentState(ref)
		experiments = append(experiments, experimentInfoFromState(workflowID, branchName, state))
	}

	return experiments, nil
}

// fetchRemoteExperiments lists experiment branches and reads their state via the GitHub API.
func fetchRemoteExperiments(repoOverride string) ([]ExperimentInfo, error) {
	experimentsLog.Printf("Fetching remote experiment branches: repo=%s", repoOverride)

	args := []string{"api", "repos/{owner}/{repo}/branches",
		"--paginate",
		"--jq", `[.[] | select(.name | startswith("` + experimentsBranchPrefix + `")) | .name]`,
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

	branchNames, err := parsePagedJSONArray[string](string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse branch list: %w", err)
	}

	var experiments []ExperimentInfo
	for _, branchName := range branchNames {
		workflowID := strings.TrimPrefix(branchName, experimentsBranchPrefix)
		state := readRemoteExperimentState(repoOverride, branchName)
		experiments = append(experiments, experimentInfoFromState(workflowID, branchName, state))
	}

	return experiments, nil
}

// fetchLocalExperimentDetails reads the state.json from a local experiment branch.
func fetchLocalExperimentDetails(branchName, workflowID string) (*ExperimentDetails, error) {
	experimentsLog.Printf("Fetching local experiment details: branch=%s", branchName)

	ref := "origin/" + branchName
	if !gitRefExists(ref) {
		if !gitRefExists(branchName) {
			return nil, fmt.Errorf("experiment branch %q not found locally (tried origin/%s and %s)",
				branchName, branchName, branchName)
		}
		ref = branchName
	}

	state := readLocalExperimentState(ref)
	return experimentDetailsFromState(workflowID, branchName, state), nil
}

// fetchRemoteExperimentDetails reads the state.json from a remote experiment branch.
func fetchRemoteExperimentDetails(repoOverride, branchName, workflowID string) (*ExperimentDetails, error) {
	experimentsLog.Printf("Fetching remote experiment details: repo=%s, branch=%s", repoOverride, branchName)

	// Verify the branch exists.
	encodedBranch := url.PathEscape(branchName)
	checkArgs := []string{"api",
		"repos/{owner}/{repo}/branches/" + encodedBranch,
		"--jq", ".name",
		"--repo", repoOverride,
	}
	checkCmd := workflow.ExecGH(checkArgs...)
	if _, err := checkCmd.Output(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "404") || strings.Contains(stderr, "not found") {
				return nil, fmt.Errorf("experiment %q not found in %s", workflowID, repoOverride)
			}
			return nil, fmt.Errorf("failed to fetch experiment branch (exit %d): %s", exitErr.ExitCode(), stderr)
		}
		return nil, fmt.Errorf("failed to fetch experiment branch: %w", err)
	}

	state := readRemoteExperimentState(repoOverride, branchName)
	return experimentDetailsFromState(workflowID, branchName, state), nil
}

// readLocalExperimentState reads state.json from a local git ref (e.g. "origin/experiments/foo").
// Returns an empty state when the file is absent or cannot be parsed.
func readLocalExperimentState(ref string) *ExperimentState {
	cmd := exec.Command("git", "show", ref+":state.json")
	out, err := cmd.Output()
	if err != nil {
		return emptyExperimentState()
	}
	return parseExperimentState(out)
}

// readRemoteExperimentState fetches state.json from an experiments/* branch via the GitHub API.
// Returns an empty state on any error (branch missing, file absent, parse failure).
func readRemoteExperimentState(repoOverride, branchName string) *ExperimentState {
	args := []string{"api",
		"repos/{owner}/{repo}/contents/state.json",
		"--field", "ref=" + branchName,
		"--jq", ".content",
		"--repo", repoOverride,
	}
	cmd := workflow.ExecGH(args...)
	out, err := cmd.Output()
	if err != nil {
		return emptyExperimentState()
	}

	// GitHub API returns base64-encoded content with embedded newlines for line-wrapping.
	// Strip all whitespace before decoding.
	b64 := strings.Join(strings.Fields(strings.TrimSpace(string(out))), "")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		experimentsLog.Printf("Failed to base64-decode state.json from %s: %v", branchName, err)
		return emptyExperimentState()
	}
	return parseExperimentState(decoded)
}

// parseExperimentState unmarshals raw JSON into an ExperimentState.
// Returns an empty state when parsing fails or the data is invalid.
func parseExperimentState(data []byte) *ExperimentState {
	var state ExperimentState
	if err := json.Unmarshal(data, &state); err != nil {
		return emptyExperimentState()
	}
	// Validate: state.json must have a counts object.
	if state.Counts == nil {
		state.Counts = map[string]map[string]int{}
	}
	return &state
}

// emptyExperimentState returns a zero-value ExperimentState with an initialised Counts map.
func emptyExperimentState() *ExperimentState {
	return &ExperimentState{Counts: map[string]map[string]int{}}
}

// experimentInfoFromState builds an ExperimentInfo summary from a state.json.
func experimentInfoFromState(workflowID, branchName string, state *ExperimentState) ExperimentInfo {
	return ExperimentInfo{
		WorkflowID:  workflowID,
		Branch:      branchName,
		Experiments: len(state.Counts),
		TotalRuns:   experimentTotalRuns(state),
		LastRun:     experimentLastRun(state),
	}
}

// experimentDetailsFromState builds ExperimentDetails from a state.json.
func experimentDetailsFromState(workflowID, branchName string, state *ExperimentState) *ExperimentDetails {
	experiments := make([]ExperimentVariantStats, 0, len(state.Counts))
	for name, variants := range state.Counts {
		total := 0
		for _, c := range variants {
			total += c
		}
		experiments = append(experiments, ExperimentVariantStats{
			Name:     name,
			Variants: variants,
			Total:    total,
		})
	}
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].Name < experiments[j].Name
	})

	recentRuns := state.Runs
	const maxRecentRuns = 10
	if len(recentRuns) > maxRecentRuns {
		recentRuns = recentRuns[len(recentRuns)-maxRecentRuns:]
	}

	return &ExperimentDetails{
		WorkflowID:  workflowID,
		Branch:      branchName,
		TotalRuns:   experimentTotalRuns(state),
		Experiments: experiments,
		RecentRuns:  recentRuns,
	}
}

// experimentTotalRuns returns the total number of runs recorded in the state.
// Prefers the runs array length when non-empty; falls back to summing all variant counts.
func experimentTotalRuns(state *ExperimentState) int {
	if len(state.Runs) > 0 {
		return len(state.Runs)
	}
	total := 0
	for _, variants := range state.Counts {
		for _, c := range variants {
			total += c
		}
	}
	return total
}

// experimentLastRun returns the date (YYYY-MM-DD) of the most recent run, or "" if unknown.
func experimentLastRun(state *ExperimentState) string {
	if len(state.Runs) == 0 {
		return ""
	}
	ts := state.Runs[len(state.Runs)-1].Timestamp
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// extractExperimentName extracts the workflow ID from a branch ref.
//
//	"origin/experiments/my-workflow" → "my-workflow"
//	"experiments/my-workflow"        → "my-workflow"
//	"experiments/"                   → "" (bare prefix, rejected by callers)
func extractExperimentName(ref string) string {
	ref = strings.TrimPrefix(ref, "origin/")
	if !strings.HasPrefix(ref, experimentsBranchPrefix) {
		return ""
	}
	// An empty result here (bare "experiments/" ref) is acceptable: callers
	// guard against empty workflow IDs with `if workflowID == ""` checks.
	return strings.TrimPrefix(ref, experimentsBranchPrefix)
}

// gitRefExists reports whether a git ref exists locally.
func gitRefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}

// printExperimentDetails renders experiment details to stderr in human-readable form.
func printExperimentDetails(d *ExperimentDetails) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Experiment workflow: "+d.WorkflowID))
	fmt.Fprintf(os.Stderr, "  Branch:     %s\n", d.Branch)
	fmt.Fprintf(os.Stderr, "  Total runs: %d\n", d.TotalRuns)

	if len(d.Experiments) > 0 {
		fmt.Fprintln(os.Stderr, "\nExperiments:")
		for _, exp := range d.Experiments {
			fmt.Fprintf(os.Stderr, "  %s (total: %d):\n", exp.Name, exp.Total)
			// Sort variants for deterministic display.
			type kv struct {
				k string
				v int
			}
			pairs := make([]kv, 0, len(exp.Variants))
			for k, v := range exp.Variants {
				pairs = append(pairs, kv{k, v})
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
			for _, p := range pairs {
				pct := 0
				if exp.Total > 0 {
					pct = p.v * 100 / exp.Total
				}
				fmt.Fprintf(os.Stderr, "    %-20s %d (%d%%)\n", p.k, p.v, pct)
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "\nNo experiment data found (state.json not present or empty).")
	}

	if len(d.RecentRuns) > 0 {
		fmt.Fprintln(os.Stderr, "\nRecent runs:")
		for _, run := range d.RecentRuns {
			date := run.Timestamp
			if len(date) >= 10 {
				date = date[:10]
			}
			fmt.Fprintf(os.Stderr, "  %s  %-16s  %s\n", date, run.RunID, formatAssignments(run.Assignments))
		}
	}
}

// formatAssignments formats a map of experiment→variant as "k=v, k=v" sorted by key.
func formatAssignments(assignments map[string]string) string {
	if len(assignments) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(assignments))
	for k := range assignments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+assignments[k])
	}
	return strings.Join(parts, ", ")
}

// parsePagedJSONArray parses multiple JSON arrays (one per page from --paginate)
// concatenated in the output and returns a merged slice.
func parsePagedJSONArray[T any](output string) ([]T, error) {
	var result []T
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var page []T
		if err := decoder.Decode(&page); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		result = append(result, page...)
	}
	return result, nil
}
