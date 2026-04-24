package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceLog = logger.New("workflow:maintenance_workflow")

// generateInstallCLISteps generates YAML steps to install or build the gh-aw CLI.
// In dev mode: builds from source using Setup Go + Build gh-aw (./gh-aw binary available)
// In release mode: installs the released CLI via the setup-cli action (gh aw available)
// In action mode: installs the released CLI via the gh-aw-actions/setup-cli action (gh aw available)
// When resolver is non-nil, attempts to resolve the setup-cli action to a SHA-pinned reference.
func generateInstallCLISteps(actionMode ActionMode, version string, actionTag string, resolver ActionSHAResolver) string {
	if actionMode == ActionModeDev {
		return `      - name: Setup Go
        uses: ` + getActionPin("actions/setup-go") + `
        with:
          go-version-file: go.mod
          cache: true

      - name: Build gh-aw
        run: make build

`
	}

	cliTag := actionTag
	if cliTag == "" {
		cliTag = version
	}

	// Action mode: use setup-cli action from external gh-aw-actions repository
	if actionMode == ActionModeAction {
		actionRepo := GitHubActionsOrgRepo + "/setup-cli"
		ref := resolveActionRef(actionRepo, cliTag, resolver)
		return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
	}

	// Release mode: use setup-cli action (consistent with copilot-setup-steps.yml)
	actionRepo := GitHubOrgRepo + "/actions/setup-cli"
	ref := resolveActionRef(actionRepo, cliTag, resolver)
	return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
}

// resolveActionRef attempts to resolve an action repo@tag to a SHA-pinned reference
// using the provided resolver. If the resolver is nil or resolution fails, it returns
// the tag-based reference (repo@tag).
func resolveActionRef(actionRepo, tag string, resolver ActionSHAResolver) string {
	if resolver != nil && tag != "" && tag != "dev" {
		sha, err := resolver.ResolveSHA(actionRepo, tag)
		if err != nil {
			maintenanceLog.Printf("Failed to resolve SHA for %s@%s: %v, falling back to tag reference", actionRepo, tag, err)
		} else if sha != "" {
			return formatActionReference(actionRepo, sha, tag)
		}
	}
	return actionRepo + "@" + tag
}

// getCLICmdPrefix returns the CLI command prefix based on action mode.
// In dev mode: "./gh-aw" (local binary built from source)
// In release mode: "gh aw" (installed via gh extension)
func getCLICmdPrefix(actionMode ActionMode) string {
	if actionMode == ActionModeDev {
		return "./gh-aw"
	}
	return "gh aw"
}

// FetchDefaultBranch queries the GitHub API to determine the default branch of the
// given repository slug (owner/repo). Returns "main" as a fallback when the slug is
// empty, not in owner/repo format, or when the API call fails.
func FetchDefaultBranch(slug string) string {
	const fallback = "main"
	if slug == "" || strings.Count(slug, "/") != 1 {
		maintenanceLog.Printf("No valid repository slug, using default branch fallback: %s", fallback)
		return fallback
	}
	maintenanceLog.Printf("Fetching default branch for repository: %s", slug)
	output, err := RunGH("Fetching default branch...", "api", "/repos/"+slug, "--jq", ".default_branch")
	if err != nil {
		maintenanceLog.Printf("Failed to fetch default branch for %s: %v, falling back to %s", slug, err, fallback)
		return fallback
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		maintenanceLog.Printf("Empty default branch response for %s, falling back to %s", slug, fallback)
		return fallback
	}
	maintenanceLog.Printf("Default branch for %s: %s", slug, branch)
	return branch
}

// GenerateMaintenanceWorkflow generates the agentics-maintenance.yml workflow
// if any workflows use the expires field for discussions or issues.
// When repoConfig is non-nil and repoConfig.MaintenanceDisabled is true the
// maintenance workflow is deleted and the function returns immediately.
// repoSlug is the owner/repo slug used to determine the default branch for the push
// trigger; pass an empty string to fall back to "main".
func GenerateMaintenanceWorkflow(workflowDataList []*WorkflowData, workflowDir string, version string, actionMode ActionMode, actionTag string, verbose bool, repoConfig *RepoConfig, repoSlug string) error {
	maintenanceLog.Print("Checking if maintenance workflow is needed")

	// Respect explicit opt-out from aw.json: maintenance: false
	if repoConfig != nil && repoConfig.MaintenanceDisabled {
		return handleMaintenanceDisabled(workflowDataList, workflowDir)
	}

	// Determine the runs-on value to use for all maintenance jobs.
	const defaultRunsOn = "ubuntu-slim"
	var configuredRunsOn RunsOnValue
	if repoConfig != nil && repoConfig.Maintenance != nil {
		configuredRunsOn = repoConfig.Maintenance.RunsOn
	}
	runsOnValue := FormatRunsOn(configuredRunsOn, defaultRunsOn)

	// Scan workflows for expires fields and track the minimum expires value
	hasExpires, minExpires := scanWorkflowsForExpires(workflowDataList)

	// Get the setup action reference (local or remote based on mode).
	// Use the first available WorkflowData's ActionResolver to enable SHA pinning.
	// Computed early so it is available in the !hasExpires path for side-repo workflows.
	var resolver ActionSHAResolver
	if len(workflowDataList) > 0 && workflowDataList[0].ActionResolver != nil {
		resolver = workflowDataList[0].ActionResolver
	}

	if !hasExpires {
		maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")

		// Delete existing maintenance workflow file if it exists (no expires means no need for maintenance)
		maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
		if _, err := os.Stat(maintenanceFile); err == nil {
			maintenanceLog.Printf("Deleting existing maintenance workflow: %s", maintenanceFile)
			if err := os.Remove(maintenanceFile); err != nil {
				return fmt.Errorf("failed to delete maintenance workflow: %w", err)
			}
			maintenanceLog.Print("Maintenance workflow deleted successfully")
		}

		// Even without expires, side-repo targets still need maintenance workflows
		// for safe_outputs, create_labels, and validate operations.
		return generateAllSideRepoMaintenanceWorkflows(workflowDataList, workflowDir, version, actionMode, actionTag, runsOnValue, resolver, false, 0)
	}

	maintenanceLog.Printf("Generating maintenance workflow for expired discussions, issues, and pull requests (minimum expires: %d hours)", minExpires)

	// Convert hours to days for cron schedule generation
	minExpiresDays := minExpires / 24
	if minExpires%24 > 0 {
		minExpiresDays++ // Round up partial days
	}

	// Generate cron schedule based on minimum expires value
	cronSchedule, scheduleDesc := generateMaintenanceCron(minExpiresDays)
	maintenanceLog.Printf("Maintenance schedule: %s (%s)", cronSchedule, scheduleDesc)

	// Fetch the default branch for the push trigger (dev mode only)
	// Resolved here to avoid passing it through multiple layers; empty slug falls back to "main"
	defaultBranch := FetchDefaultBranch(repoSlug)

	// Generate the YAML content for the maintenance workflow
	content := buildMaintenanceWorkflowYAML(cronSchedule, scheduleDesc, minExpiresDays, runsOnValue, actionMode, version, actionTag, resolver, configuredRunsOn, defaultBranch)

	// Write the maintenance workflow file
	maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
	maintenanceLog.Printf("Writing maintenance workflow to %s", maintenanceFile)

	if err := os.WriteFile(maintenanceFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write maintenance workflow: %w", err)
	}

	maintenanceLog.Print("Maintenance workflow generated successfully")

	// Generate side-repo maintenance workflows for any SideRepoOps targets detected.
	if err := generateAllSideRepoMaintenanceWorkflows(workflowDataList, workflowDir, version, actionMode, actionTag, runsOnValue, resolver, hasExpires, minExpiresDays); err != nil {
		return err
	}

	return nil
}

// handleMaintenanceDisabled handles the case where maintenance is disabled in repo config.
// It warns about workflows that use expires and deletes any existing maintenance workflow.
func handleMaintenanceDisabled(workflowDataList []*WorkflowData, workflowDir string) error {
	maintenanceLog.Print("Maintenance disabled via repo config, skipping generation")

	// Warn if any workflow uses expires — those features rely on maintenance
	// and will silently become no-ops when it is disabled.
	for _, workflowData := range workflowDataList {
		if workflowData.SafeOutputs == nil {
			continue
		}
		usesExpires := (workflowData.SafeOutputs.CreateDiscussions != nil && workflowData.SafeOutputs.CreateDiscussions.Expires > 0) ||
			(workflowData.SafeOutputs.CreateIssues != nil && workflowData.SafeOutputs.CreateIssues.Expires > 0) ||
			(workflowData.SafeOutputs.CreatePullRequests != nil && workflowData.SafeOutputs.CreatePullRequests.Expires > 0)
		if usesExpires {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Workflow '%s' uses the 'expires' field but maintenance is disabled in aw.json. "+
					"Expiration will not run until maintenance is re-enabled.", workflowData.Name)))
		}
	}

	maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
	if _, err := os.Stat(maintenanceFile); err == nil {
		maintenanceLog.Printf("Deleting existing maintenance workflow: %s", maintenanceFile)
		if err := os.Remove(maintenanceFile); err != nil {
			return fmt.Errorf("failed to delete maintenance workflow: %w", err)
		}
	}
	return nil
}

// scanWorkflowsForExpires checks all workflow data for expires fields and returns
// whether any expires fields are set and the minimum expires value in hours.
func scanWorkflowsForExpires(workflowDataList []*WorkflowData) (bool, int) {
	hasExpires := false
	minExpires := 0 // Track minimum expires value in hours

	for _, workflowData := range workflowDataList {
		if workflowData.SafeOutputs == nil {
			continue
		}
		// Check for expired discussions
		if workflowData.SafeOutputs.CreateDiscussions != nil {
			if workflowData.SafeOutputs.CreateDiscussions.Expires > 0 {
				hasExpires = true
				expires := workflowData.SafeOutputs.CreateDiscussions.Expires
				maintenanceLog.Printf("Workflow %s has expires field set to %d hours for discussions", workflowData.Name, expires)
				if minExpires == 0 || expires < minExpires {
					minExpires = expires
				}
			}
		}
		// Check for expired issues
		if workflowData.SafeOutputs.CreateIssues != nil {
			if workflowData.SafeOutputs.CreateIssues.Expires > 0 {
				hasExpires = true
				expires := workflowData.SafeOutputs.CreateIssues.Expires
				maintenanceLog.Printf("Workflow %s has expires field set to %d hours for issues", workflowData.Name, expires)
				if minExpires == 0 || expires < minExpires {
					minExpires = expires
				}
			}
		}
		// Check for expired pull requests
		if workflowData.SafeOutputs.CreatePullRequests != nil {
			if workflowData.SafeOutputs.CreatePullRequests.Expires > 0 {
				hasExpires = true
				expires := workflowData.SafeOutputs.CreatePullRequests.Expires
				maintenanceLog.Printf("Workflow %s has expires field set to %d hours for pull requests", workflowData.Name, expires)
				if minExpires == 0 || expires < minExpires {
					minExpires = expires
				}
			}
		}
	}

	return hasExpires, minExpires
}
