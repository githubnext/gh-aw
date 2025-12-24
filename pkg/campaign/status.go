package campaign

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var statusLog = logger.New("campaign:status")

// ComputeCompiledState inspects the compiled state of all
// workflows referenced by a campaign. It returns:
//
//	"Yes"   - all referenced workflows exist and are compiled & up-to-date
//	"No"    - at least one workflow exists but is missing a lock file or is stale
//	"Missing workflow" - at least one referenced workflow markdown file does not exist
//	"N/A"   - campaign does not reference any workflows
func ComputeCompiledState(spec CampaignSpec, workflowsDir string) string {
	statusLog.Printf("Computing compiled state for campaign '%s' with %d workflows", spec.ID, len(spec.Workflows))

	if len(spec.Workflows) == 0 {
		return "N/A"
	}

	compiledAll := true
	missingAny := false

	for _, wf := range spec.Workflows {
		mdPath := filepath.Join(workflowsDir, wf+".md")
		lockPath := filepath.Join(workflowsDir, wf+".lock.yml")

		mdInfo, err := os.Stat(mdPath)
		if err != nil {
			statusLog.Printf("Workflow markdown not found for campaign '%s': %s", spec.ID, mdPath)
			missingAny = true
			compiledAll = false
			continue
		}

		lockInfo, err := os.Stat(lockPath)
		if err != nil {
			statusLog.Printf("Lock file not found for workflow '%s' in campaign '%s': %s", wf, spec.ID, lockPath)
			compiledAll = false
			continue
		}

		if mdInfo.ModTime().After(lockInfo.ModTime()) {
			statusLog.Printf("Lock file out of date for workflow '%s' in campaign '%s'", wf, spec.ID)
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

// FetchItemCounts uses gh CLI (via workflow.ExecGH) to fetch basic
// counts of issues and pull requests tagged with the given tracker label.
//
// If trackerLabel is empty or any errors occur, it falls back to zeros and
// logs at debug level instead of failing the command.
func FetchItemCounts(trackerLabel string) (issuesOpen, issuesClosed, prsOpen, prsMerged int) {
	statusLog.Printf("Fetching item counts for tracker label: %s", trackerLabel)

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
			statusLog.Printf("Failed to decode issue list for tracker label '%s': %v", trackerLabel, err)
		}
	} else if err != nil {
		statusLog.Printf("Failed to fetch issues for tracker label '%s': %v", trackerLabel, err)
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
			statusLog.Printf("Failed to decode PR list for tracker label '%s': %v", trackerLabel, err)
		}
	} else if err != nil {
		statusLog.Printf("Failed to fetch PRs for tracker label '%s': %v", trackerLabel, err)
	}

	return issuesOpen, issuesClosed, prsOpen, prsMerged
}

// FetchMetricsFromRepoMemory attempts to load the latest JSON
// metrics snapshot matching the provided glob from the
// memory/campaigns branch. It is best-effort: errors are logged and
// treated as "no metrics" rather than failing the command.
func FetchMetricsFromRepoMemory(metricsGlob string) (*CampaignMetricsSnapshot, error) {
	statusLog.Printf("Fetching metrics from repo memory with glob: %s", metricsGlob)

	if strings.TrimSpace(metricsGlob) == "" {
		return nil, nil
	}

	// List all files in the memory/campaigns branch
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "memory/campaigns")
	output, err := cmd.Output()
	if err != nil {
		statusLog.Printf("Unable to list repo-memory branch for metrics (memory/campaigns): %v", err)
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
			statusLog.Printf("Invalid metrics_glob '%s': %v", metricsGlob, err)
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
		statusLog.Printf("Failed to read metrics file '%s' from memory/campaigns: %v", latest, err)
		return nil, nil
	}

	var snapshot CampaignMetricsSnapshot
	if err := json.Unmarshal(fileData, &snapshot); err != nil {
		statusLog.Printf("Failed to decode metrics JSON from '%s': %v", latest, err)
		return nil, nil
	}

	return &snapshot, nil
}

// FetchCursorFreshnessFromRepoMemory finds the latest cursor/checkpoint file
// matching cursorGlob in the memory/campaigns branch and returns the matched
// path along with a best-effort freshness timestamp derived from git history.
//
// Errors are treated as "no cursor" rather than failing the command.
func FetchCursorFreshnessFromRepoMemory(cursorGlob string) (cursorPath string, cursorUpdatedAt string) {
	statusLog.Printf("Fetching cursor freshness from repo memory with glob: %s", cursorGlob)

	if strings.TrimSpace(cursorGlob) == "" {
		return "", ""
	}

	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "memory/campaigns")
	output, err := cmd.Output()
	if err != nil {
		statusLog.Printf("Unable to list repo-memory branch for cursor (memory/campaigns): %v", err)
		return "", ""
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var matches []string
	for scanner.Scan() {
		pathStr := strings.TrimSpace(scanner.Text())
		if pathStr == "" {
			continue
		}
		matched, err := path.Match(cursorGlob, pathStr)
		if err != nil {
			statusLog.Printf("Invalid cursor_glob '%s': %v", cursorGlob, err)
			return "", ""
		}
		if matched {
			matches = append(matches, pathStr)
		}
	}

	if len(matches) == 0 {
		return "", ""
	}

	latest := matches[0]
	for _, m := range matches[1:] {
		if m > latest {
			latest = m
		}
	}

	// Best-effort: use git log to get the last commit time for this path
	// on the memory/campaigns branch.
	logCmd := exec.Command("git", "log", "-1", "--format=%cI", "memory/campaigns", "--", latest)
	logOut, err := logCmd.Output()
	if err != nil {
		statusLog.Printf("Failed to read cursor freshness for '%s' from memory/campaigns: %v", latest, err)
		return latest, ""
	}

	return latest, strings.TrimSpace(string(logOut))
}

// BuildRuntimeStatus builds a CampaignRuntimeStatus for a single campaign spec.
func BuildRuntimeStatus(spec CampaignSpec, workflowsDir string) CampaignRuntimeStatus {
	compiled := ComputeCompiledState(spec, workflowsDir)
	issuesOpen, issuesClosed, prsOpen, prsMerged := FetchItemCounts(spec.TrackerLabel)

	cursorPath, cursorUpdatedAt := FetchCursorFreshnessFromRepoMemory(spec.CursorGlob)

	var metricsTasksTotal, metricsTasksCompleted int
	var metricsVelocity float64
	var metricsETA string
	if strings.TrimSpace(spec.MetricsGlob) != "" {
		if snapshot, err := FetchMetricsFromRepoMemory(spec.MetricsGlob); err != nil {
			statusLog.Printf("Failed to fetch metrics for campaign '%s': %v", spec.ID, err)
		} else if snapshot != nil {
			metricsTasksTotal = snapshot.TasksTotal
			metricsTasksCompleted = snapshot.TasksCompleted
			metricsVelocity = snapshot.VelocityPerDay
			metricsETA = snapshot.EstimatedCompletion
		}
	}

	return CampaignRuntimeStatus{
		ID:                         spec.ID,
		Name:                       spec.Name,
		TrackerLabel:               spec.TrackerLabel,
		Workflows:                  spec.Workflows,
		Compiled:                   compiled,
		IssuesOpen:                 issuesOpen,
		IssuesClosed:               issuesClosed,
		PRsOpen:                    prsOpen,
		PRsMerged:                  prsMerged,
		MetricsTasksTotal:          metricsTasksTotal,
		MetricsTasksCompleted:      metricsTasksCompleted,
		MetricsVelocityPerDay:      metricsVelocity,
		MetricsEstimatedCompletion: metricsETA,
		CursorPath:                 cursorPath,
		CursorUpdatedAt:            cursorUpdatedAt,
	}
}
