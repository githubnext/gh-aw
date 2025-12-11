package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var campaignLog = logger.New("cli:campaigns")

// CampaignSpec defines a first-class campaign configuration loaded from YAML.
//
// Files are discovered from the local repository under:
//   campaigns/*.campaign.yaml or campaigns/*.campaign.yml
//
// This provides a thin, declarative layer on top of existing agentic
// workflows and repo-memory conventions.
type CampaignSpec struct {
	ID          string   `yaml:"id" json:"id" console:"header:ID"`
	Name        string   `yaml:"name" json:"name" console:"header:Name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty" console:"header:Description,omitempty"`

	// Version is an optional spec version string (for example: v1).
	// When omitted, it defaults to v1 during validation.
	Version string `yaml:"version,omitempty" json:"version,omitempty" console:"header:Version,omitempty"`

	// Workflows associates this campaign with one or more workflow IDs
	// (basename of the Markdown file without .md).
	Workflows []string `yaml:"workflows,omitempty" json:"workflows,omitempty" console:"header:Workflows,omitempty"`

	// MemoryPaths documents where this campaign writes its repo-memory
	// (for example: memory/campaigns/incident-*/**).
	MemoryPaths []string `yaml:"memory_paths,omitempty" json:"memory_paths,omitempty" console:"header:Memory Paths,omitempty"`

	// MetricsGlob is an optional glob (relative to the repository root)
	// used to locate JSON metrics snapshots stored in the
	// memory/campaigns branch. When set, `gh aw campaign status` will
	// attempt to read the latest matching metrics file and surface a few
	// key fields.
	MetricsGlob string `yaml:"metrics_glob,omitempty" json:"metrics_glob,omitempty" console:"header:Metrics Glob,omitempty"`

	// Owners lists the primary human owners for this campaign.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty" console:"header:Owners,omitempty"`

	// ExecutiveSponsors lists executive stakeholders or sponsors who are
	// accountable for the outcome of this campaign.
	ExecutiveSponsors []string `yaml:"executive_sponsors,omitempty" json:"executive_sponsors,omitempty" console:"header:Executive Sponsors,omitempty"`

	// RiskLevel is an optional free-form field (e.g. low/medium/high).
	RiskLevel string `yaml:"risk_level,omitempty" json:"risk_level,omitempty" console:"header:Risk Level,omitempty"`

	// TrackerLabel describes the label used to associate issues/PRs with
	// this campaign (for example: campaign:incident-response).
	TrackerLabel string `yaml:"tracker_label,omitempty" json:"tracker_label,omitempty" console:"header:Tracker Label,omitempty"`

	// State describes the lifecycle stage of the campaign definition.
	// Valid values are: planned, active, paused, completed, archived.
	State string `yaml:"state,omitempty" json:"state,omitempty" console:"header:State,omitempty"`

	// Tags provide free-form categorization for reporting (for example:
	// security, modernization, rollout).
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" console:"header:Tags,omitempty"`

	// AllowedSafeOutputs documents which safe-outputs operations this
	// campaign is expected to use (for example: create-issue,
	// create-pull-request). This is currently informational but can be
	// enforced by validation in the future.
	AllowedSafeOutputs []string `yaml:"allowed_safe_outputs,omitempty" json:"allowed_safe_outputs,omitempty" console:"header:Allowed Safe Outputs,omitempty"`

	// ApprovalPolicy describes high-level approval expectations for this
	// campaign (for example: number of approvals and required roles).
	ApprovalPolicy *CampaignApprovalPolicy `yaml:"approval_policy,omitempty" json:"approval_policy,omitempty"`

	// ConfigPath is populated at load time with the relative path of
	// the YAML file on disk, to help users locate definitions.
	ConfigPath string `yaml:"-" json:"config_path" console:"header:Config Path"`
}

// CampaignApprovalPolicy captures basic approval expectations for a
// campaign. It is intentionally lightweight and advisory; enforcement
// is left to workflows and organizational process.
type CampaignApprovalPolicy struct {
	RequiredApprovals int      `yaml:"required_approvals,omitempty" json:"required_approvals,omitempty"`
	RequiredRoles     []string `yaml:"required_roles,omitempty" json:"required_roles,omitempty"`
	ChangeControl     bool     `yaml:"change_control,omitempty" json:"change_control,omitempty"`
}

// loadCampaignSpecs scans the repository for campaign spec files and returns
// a slice of CampaignSpec. If the campaigns directory does not exist, it
// returns an empty slice and no error.
func loadCampaignSpecs(rootDir string) ([]CampaignSpec, error) {
	campaignLog.Printf("Loading campaign specs from rootDir=%s", rootDir)

	campaignsDir := filepath.Join(rootDir, "campaigns")
	entries, err := os.ReadDir(campaignsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			campaignLog.Print("No campaigns directory found; returning empty list")
			return []CampaignSpec{}, nil
		}
		return nil, fmt.Errorf("failed to read campaigns directory '%s': %w", campaignsDir, err)
	}

	var specs []CampaignSpec

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".campaign.yaml") && !strings.HasSuffix(name, ".campaign.yml") {
			continue
		}

		path := filepath.Join(campaignsDir, name)
		campaignLog.Printf("Found campaign spec file: %s", path)

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read campaign spec '%s': %w", path, err)
		}

		var spec CampaignSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse campaign spec '%s': %w", path, err)
		}

		// Default ID from filename when missing
		if strings.TrimSpace(spec.ID) == "" {
			base := strings.TrimSuffix(strings.TrimSuffix(name, ".campaign.yaml"), ".campaign.yml")
			spec.ID = base
		}

		// Name falls back to ID when not provided
		if strings.TrimSpace(spec.Name) == "" {
			spec.Name = spec.ID
		}

		spec.ConfigPath = filepath.ToSlash(filepath.Join("campaigns", name))
		specs = append(specs, spec)
	}

	campaignLog.Printf("Loaded %d campaign specs", len(specs))
	return specs, nil
}

// filterCampaignSpecs filters campaigns by a simple substring match on ID or
// Name (case-insensitive). When pattern is empty, all campaigns are returned.
func filterCampaignSpecs(specs []CampaignSpec, pattern string) []CampaignSpec {
	if pattern == "" {
		return specs
	}

	var filtered []CampaignSpec
	lowerPattern := strings.ToLower(pattern)

	for _, spec := range specs {
		if strings.Contains(strings.ToLower(spec.ID), lowerPattern) || strings.Contains(strings.ToLower(spec.Name), lowerPattern) {
			filtered = append(filtered, spec)
		}
	}

	return filtered
}

// validateCampaignSpec performs lightweight semantic validation of a
// single CampaignSpec and returns a slice of human-readable problems.
//
// It is intentionally conservative – it does not fail on every
// possible issue, but focuses on the most important invariants for
// enterprise usage.
func validateCampaignSpec(spec *CampaignSpec) []string {
	var problems []string

	trimmedID := strings.TrimSpace(spec.ID)
	if trimmedID == "" {
		problems = append(problems, "id is required and must be non-empty")
	} else {
		// Enforce a simple, URL-safe pattern for IDs
		for _, ch := range trimmedID {
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			problems = append(problems, "id must use only lowercase letters, digits, and hyphens ("+trimmedID+")")
			break
		}
	}

	if strings.TrimSpace(spec.Name) == "" {
		problems = append(problems, "name should be provided (falls back to id, but explicit names are recommended)")
	}

	if len(spec.Workflows) == 0 {
		problems = append(problems, "workflows should list at least one workflow implementing this campaign")
	}

	if strings.TrimSpace(spec.TrackerLabel) == "" {
		problems = append(problems, "tracker_label should be set to link issues and PRs to this campaign")
	} else if !strings.Contains(spec.TrackerLabel, ":") {
		problems = append(problems, "tracker_label should follow a namespaced pattern (for example: campaign:security-q1-2025)")
	}

	// Normalize and validate version/state when present.
	if strings.TrimSpace(spec.Version) == "" {
		// Default version for v1 specs when omitted.
		spec.Version = "v1"
	}

	if spec.State != "" {
		switch spec.State {
		case "planned", "active", "paused", "completed", "archived":
			// valid
		default:
			problems = append(problems, "state must be one of: planned, active, paused, completed, archived")
		}
	}

	return problems
}

// CampaignRuntimeStatus represents the live status of a campaign, including
// compiled workflow state and basic issue/PR counts derived from the tracker
// label.
type CampaignRuntimeStatus struct {
	ID           string   `json:"id" console:"header:ID"`
	Name         string   `json:"name" console:"header:Name"`
	TrackerLabel string   `json:"tracker_label,omitempty" console:"header:Tracker Label,omitempty"`
	Workflows    []string `json:"workflows,omitempty" console:"header:Workflows,omitempty"`
	Compiled     string   `json:"compiled" console:"header:Compiled"`

	IssuesOpen   int `json:"issues_open,omitempty" console:"header:Issues Open,omitempty"`
	IssuesClosed int `json:"issues_closed,omitempty" console:"header:Issues Closed,omitempty"`
	PRsOpen      int `json:"prs_open,omitempty" console:"header:PRs Open,omitempty"`
	PRsMerged    int `json:"prs_merged,omitempty" console:"header:PRs Merged,omitempty"`

	// Optional metrics from repo-memory (when MetricsGlob is set and a
	// matching JSON snapshot is found on the memory/campaigns branch).
	MetricsTasksTotal         int     `json:"metrics_tasks_total,omitempty" console:"header:Tasks Total,omitempty"`
	MetricsTasksCompleted     int     `json:"metrics_tasks_completed,omitempty" console:"header:Tasks Completed,omitempty"`
	MetricsVelocityPerDay     float64 `json:"metrics_velocity_per_day,omitempty" console:"header:Velocity/Day,omitempty"`
	MetricsEstimatedCompletion string  `json:"metrics_estimated_completion,omitempty" console:"header:ETA,omitempty"`
}

// CampaignMetricsSnapshot describes the JSON structure used by campaign
// metrics snapshots written into the memory/campaigns branch.
//
// This mirrors the example in the campaigns guide:
//   {
//     "date": "2025-01-16",
//     "campaign_id": "security-q1-2025",
//     "tasks_total": 200,
//     "tasks_completed": 15,
//     "tasks_in_progress": 30,
//     "tasks_blocked": 5,
//     "velocity_per_day": 7.5,
//     "estimated_completion": "2025-02-12"
//   }
type CampaignMetricsSnapshot struct {
	Date               string  `json:"date,omitempty"`
	CampaignID         string  `json:"campaign_id,omitempty"`
	TasksTotal         int     `json:"tasks_total,omitempty"`
	TasksCompleted     int     `json:"tasks_completed,omitempty"`
	TasksInProgress    int     `json:"tasks_in_progress,omitempty"`
	TasksBlocked       int     `json:"tasks_blocked,omitempty"`
	VelocityPerDay     float64 `json:"velocity_per_day,omitempty"`
	EstimatedCompletion string  `json:"estimated_completion,omitempty"`
}

// computeCompiledStateForCampaign inspects the compiled state of all
// workflows referenced by a campaign. It returns:
//   "Yes"   - all referenced workflows exist and are compiled & up-to-date
//   "No"    - at least one workflow exists but is missing a lock file or is stale
//   "Missing workflow" - at least one referenced workflow markdown file does not exist
//   "N/A"   - campaign does not reference any workflows
func computeCompiledStateForCampaign(spec CampaignSpec) string {
	if len(spec.Workflows) == 0 {
		return "N/A"
	}

	workflowsDir := getWorkflowsDir()
	compiledAll := true
	missingAny := false

	for _, wf := range spec.Workflows {
		mdPath := filepath.Join(workflowsDir, wf+".md")
		lockPath := mdPath + ".lock.yml"

		mdInfo, err := os.Stat(mdPath)
		if err != nil {
			campaignLog.Printf("Workflow markdown not found for campaign '%s': %s", spec.ID, mdPath)
			missingAny = true
			compiledAll = false
			continue
		}

		lockInfo, err := os.Stat(lockPath)
		if err != nil {
			campaignLog.Printf("Lock file not found for workflow '%s' in campaign '%s': %s", wf, spec.ID, lockPath)
			compiledAll = false
			continue
		}

		if mdInfo.ModTime().After(lockInfo.ModTime()) {
			campaignLog.Printf("Lock file out of date for workflow '%s' in campaign '%s'", wf, spec.ID)
			compiledAll = false
		}
	}

	if missingAny {
		return "Missing workflow"
	}
	if compiledAll {
		return "Yes"
	}
	return "No"
}

// ghIssueOrPRState is a tiny helper struct for decoding gh issue/pr list
// output when using --json state.
type ghIssueOrPRState struct {
	State string `json:"state"`
}

// fetchCampaignItemCounts uses gh CLI (via workflow.ExecGH) to fetch basic
// counts of issues and pull requests tagged with the given tracker label.
//
// If trackerLabel is empty or any errors occur, it falls back to zeros and
// logs at debug level instead of failing the command.
func fetchCampaignItemCounts(trackerLabel string) (issuesOpen, issuesClosed, prsOpen, prsMerged int) {
	if strings.TrimSpace(trackerLabel) == "" {
		return 0, 0, 0, 0
	}

	// Issues
	issueCmd := workflow.ExecGH("issue", "list", "--label", trackerLabel, "--state", "all", "--json", "state")
	issueOutput, err := issueCmd.Output()
	if err == nil && len(issueOutput) > 0 && json.Valid(issueOutput) {
		var issues []ghIssueOrPRState
		if err := json.Unmarshal(issueOutput, &issues); err == nil {
			for _, it := range issues {
				state := strings.ToLower(strings.TrimSpace(it.State))
				if state == "open" {
					issuesOpen++
				} else {
					issuesClosed++
				}
			}
		} else if err != nil {
			campaignLog.Printf("Failed to decode issue list for tracker label '%s': %v", trackerLabel, err)
		}
	} else if err != nil {
		campaignLog.Printf("Failed to fetch issues for tracker label '%s': %v", trackerLabel, err)
	}

	// Pull requests
	prCmd := workflow.ExecGH("pr", "list", "--label", trackerLabel, "--state", "all", "--json", "state")
	prOutput, err := prCmd.Output()
	if err == nil && len(prOutput) > 0 && json.Valid(prOutput) {
		var prs []ghIssueOrPRState
		if err := json.Unmarshal(prOutput, &prs); err == nil {
			for _, it := range prs {
				state := strings.ToLower(strings.TrimSpace(it.State))
				switch state {
				case "open":
					prsOpen++
				case "merged":
					prsMerged++
				}
			}
		} else if err != nil {
			campaignLog.Printf("Failed to decode PR list for tracker label '%s': %v", trackerLabel, err)
		}
	} else if err != nil {
		campaignLog.Printf("Failed to fetch PRs for tracker label '%s': %v", trackerLabel, err)
	}

	return issuesOpen, issuesClosed, prsOpen, prsMerged
}

// fetchCampaignMetricsFromRepoMemory attempts to load the latest JSON
// metrics snapshot matching the provided glob from the
// memory/campaigns branch. It is best-effort: errors are logged and
// treated as "no metrics" rather than failing the command.
func fetchCampaignMetricsFromRepoMemory(metricsGlob string) (*CampaignMetricsSnapshot, error) {
	if strings.TrimSpace(metricsGlob) == "" {
		return nil, nil
	}

	// List all files in the memory/campaigns branch
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "memory/campaigns")
	output, err := cmd.Output()
	if err != nil {
		campaignLog.Printf("Unable to list repo-memory branch for metrics (memory/campaigns): %v", err)
		return nil, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var matches []string
	for scanner.Scan() {
		pathStr := strings.TrimSpace(scanner.Text())
		if pathStr == "" {
			continue
		}
		matched, err := path.Match(metricsGlob, pathStr)
		if err != nil {
			campaignLog.Printf("Invalid metrics_glob '%s': %v", metricsGlob, err)
			return nil, nil
		}
		if matched {
			matches = append(matches, pathStr)
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Pick the lexicographically last match as the "latest" snapshot.
	latest := matches[0]
	for _, m := range matches[1:] {
		if m > latest {
			latest = m
		}
	}

	showArg := fmt.Sprintf("memory/campaigns:%s", latest)
	showCmd := exec.Command("git", "show", showArg)
	fileData, err := showCmd.Output()
	if err != nil {
		campaignLog.Printf("Failed to read metrics file '%s' from memory/campaigns: %v", latest, err)
		return nil, nil
	}

	var snapshot CampaignMetricsSnapshot
	if err := json.Unmarshal(fileData, &snapshot); err != nil {
		campaignLog.Printf("Failed to decode metrics JSON from '%s': %v", latest, err)
		return nil, nil
	}

	return &snapshot, nil
}

// runCampaignStatus is the implementation for the `gh aw campaign` command.
// It loads campaign specs from the local repository and renders them either
// as a console table or JSON.
func runCampaignStatus(pattern string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := loadCampaignSpecs(cwd)
	if err != nil {
		return err
	}

	specs = filterCampaignSpecs(specs, pattern)

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(specs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaigns as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.yaml' to define campaigns."))
		return nil
	}

	// Render table to stdout for human-friendly output
	output := console.RenderStruct(specs)
	fmt.Print(output)
	return nil
}

// runCampaignRuntimeStatus builds a higher-level view of campaign specs with
// live information derived from GitHub (issue/PR counts) and compiled
// workflow state.
func runCampaignRuntimeStatus(pattern string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := loadCampaignSpecs(cwd)
	if err != nil {
		return err
	}

	specs = filterCampaignSpecs(specs, pattern)
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.yaml' to define campaigns."))
		return nil
	}

	var statuses []CampaignRuntimeStatus
	for _, spec := range specs {
		compiled := computeCompiledStateForCampaign(spec)
		issuesOpen, issuesClosed, prsOpen, prsMerged := fetchCampaignItemCounts(spec.TrackerLabel)

		var metricsTasksTotal, metricsTasksCompleted int
		var metricsVelocity float64
		var metricsETA string
		if strings.TrimSpace(spec.MetricsGlob) != "" {
			if snapshot, err := fetchCampaignMetricsFromRepoMemory(spec.MetricsGlob); err != nil {
				campaignLog.Printf("Failed to fetch metrics for campaign '%s': %v", spec.ID, err)
			} else if snapshot != nil {
				metricsTasksTotal = snapshot.TasksTotal
				metricsTasksCompleted = snapshot.TasksCompleted
				metricsVelocity = snapshot.VelocityPerDay
				metricsETA = snapshot.EstimatedCompletion
			}
		}

		statuses = append(statuses, CampaignRuntimeStatus{
			ID:           spec.ID,
			Name:         spec.Name,
			TrackerLabel: spec.TrackerLabel,
			Workflows:    spec.Workflows,
			Compiled:     compiled,
			IssuesOpen:   issuesOpen,
			IssuesClosed: issuesClosed,
			PRsOpen:      prsOpen,
			PRsMerged:    prsMerged,
			MetricsTasksTotal:         metricsTasksTotal,
			MetricsTasksCompleted:     metricsTasksCompleted,
			MetricsVelocityPerDay:     metricsVelocity,
			MetricsEstimatedCompletion: metricsETA,
		})
	}

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaign status as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	output := console.RenderStruct(statuses)
	fmt.Print(output)
	return nil
}

// runCampaignValidate loads campaign specs and validates them, returning
// a structured report. When strict is true, the command will exit with
// a non-zero status if any problems are found.
type CampaignValidationResult struct {
	ID         string   `json:"id" console:"header:ID"`
	Name       string   `json:"name" console:"header:Name"`
	ConfigPath string   `json:"config_path" console:"header:Config Path"`
	Problems   []string `json:"problems,omitempty" console:"header:Problems,omitempty"`
}

func runCampaignValidate(pattern string, jsonOutput bool, strict bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	specs, err := loadCampaignSpecs(cwd)
	if err != nil {
		return err
	}

	specs = filterCampaignSpecs(specs, pattern)
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No campaign specs found. Add files under 'campaigns/*.campaign.yaml' to define campaigns."))
		return nil
	}

	var results []CampaignValidationResult
	var totalProblems int

	for i := range specs {
		problems := validateCampaignSpec(&specs[i])
		if len(problems) > 0 {
			campaignLog.Printf("Validation problems for campaign '%s' (%s): %v", specs[i].ID, specs[i].ConfigPath, problems)
		}

		results = append(results, CampaignValidationResult{
			ID:         specs[i].ID,
			Name:       specs[i].Name,
			ConfigPath: specs[i].ConfigPath,
			Problems:   problems,
		})
		totalProblems += len(problems)
	}

	if jsonOutput {
		jsonBytes, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal campaign validation results as JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		output := console.RenderStruct(results)
		fmt.Print(output)
	}

	if strict && totalProblems > 0 {
		return fmt.Errorf("campaign validation failed: %d problem(s) found across %d campaign(s)", totalProblems, len(results))
	}

	return nil
}

// createCampaignSpecSkeleton creates a new campaign spec YAML file under
// campaigns/ with a minimal skeleton definition. It returns the
// relative file path created.
func createCampaignSpecSkeleton(rootDir, id string, force bool) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("campaign id is required")
	}

	// Reuse the same simple rules as validateCampaignSpec for IDs
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return "", fmt.Errorf("campaign id must use only lowercase letters, digits, and hyphens (%s)", id)
	}

	campaignsDir := filepath.Join(rootDir, "campaigns")
	if err := os.MkdirAll(campaignsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create campaigns directory: %w", err)
	}

	fileName := id + ".campaign.yaml"
	fullPath := filepath.Join(campaignsDir, fileName)
	relPath := filepath.ToSlash(filepath.Join("campaigns", fileName))

	if _, err := os.Stat(fullPath); err == nil && !force {
		return "", fmt.Errorf("campaign spec already exists at %s (use --force to overwrite)", relPath)
	}

	name := strings.ReplaceAll(id, "-", " ")
	name = strings.Title(name)

	spec := CampaignSpec{
		ID:           id,
		Name:         name,
		Version:      "v1",
		State:        "planned",
		TrackerLabel: "campaign:" + id,
	}

	data, err := yaml.Marshal(&spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal campaign spec: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write campaign spec file '%s': %w", relPath, err)
	}

	return relPath, nil
}

// NewCampaignCommand creates the `gh aw campaign` command that surfaces
// first-class campaign definitions from YAML files.
func NewCampaignCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaign [filter]",
		Short: "Inspect first-class campaign definitions from campaigns/*.campaign.yaml",
		Long: `List and inspect first-class campaign definitions declared in YAML files.

Campaigns are defined using YAML files under the local repository:

  campaigns/*.campaign.yaml

Each file describes a campaign pattern (ID, name, owners, associated
workflows, repo-memory paths, and risk level). This command provides a
single place to see all campaigns configured for the repo.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign             # List all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign security    # Filter campaigns by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign --json      # Output campaign definitions as JSON
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runCampaignStatus(pattern, jsonOutput)
		},
	}

	cmd.Flags().Bool("json", false, "Output campaign definitions in JSON format")

	// Subcommand: campaign status
	statusCmd := &cobra.Command{
		Use:   "status [filter]",
		Short: "Show live status for campaigns (compiled workflows, issues, PRs)",
		Long: `Show live status for campaigns, including whether referenced workflows
are compiled and basic issue/PR counts derived from the campaign's
tracker label.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign status              # Status for all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign status security     # Filter by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign status --json       # JSON status output
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			return runCampaignRuntimeStatus(pattern, jsonOutput)
		},
	}

	statusCmd.Flags().Bool("json", false, "Output campaign status in JSON format")
	cmd.AddCommand(statusCmd)

	// Subcommand: campaign new
	newCmd := &cobra.Command{
		Use:   "new <id>",
		Short: "Create a new campaign spec skeleton under campaigns/",
		Long: `Create a new campaign spec skeleton file under campaigns/.

The file will be created as campaigns/<id>.campaign.yaml with a basic
structure (id, name, version, state, tracker_label). You can then
update owners, workflows, memory paths, metrics_glob, and governance
fields to match your initiative.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign new security-q1-2025
  ` + constants.CLIExtensionPrefix + ` campaign new modernization-winter2025 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			force, _ := cmd.Flags().GetBool("force")

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			path, err := createCampaignSpecSkeleton(cwd, id, force)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign spec at "+path))
			return nil
		},
	}

	newCmd.Flags().Bool("force", false, "Overwrite existing spec file if it already exists")
	cmd.AddCommand(newCmd)

	// Subcommand: campaign validate
	validateCmd := &cobra.Command{
		Use:   "validate [filter]",
		Short: "Validate campaign spec files for common issues",
		Long: `Validate campaign spec files under campaigns/*.campaign.yaml.

This command performs lightweight semantic validation of campaign
definitions (IDs, tracker labels, workflows, lifecycle state, and
other key fields). By default it exits with a non-zero status when
problems are found.

Examples:
  ` + constants.CLIExtensionPrefix + ` campaign validate              # Validate all campaigns
  ` + constants.CLIExtensionPrefix + ` campaign validate security     # Filter by ID or name
  ` + constants.CLIExtensionPrefix + ` campaign validate --json       # JSON validation report
  ` + constants.CLIExtensionPrefix + ` campaign validate --no-strict  # Report problems without failing`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pattern string
			if len(args) > 0 {
				pattern = args[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			strict, _ := cmd.Flags().GetBool("strict")
			return runCampaignValidate(pattern, jsonOutput, strict)
		},
	}

	validateCmd.Flags().Bool("json", false, "Output campaign validation results in JSON format")
	validateCmd.Flags().Bool("strict", true, "Exit with non-zero status if any problems are found")
	cmd.AddCommand(validateCmd)

	return cmd
}
