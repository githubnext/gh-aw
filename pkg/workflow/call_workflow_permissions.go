package workflow

import (
	"fmt"
	"os"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var callWorkflowPermissionsLog = logger.New("workflow:call_workflow_permissions")

// permissionLevelRank maps a permission level to a comparable rank where a higher
// number grants strictly more access (none < read < write). Used to determine
// whether one permission set covers another. Unknown or empty levels rank as 0.
func permissionLevelRank(level PermissionLevel) int {
	switch level {
	case PermissionWrite:
		return 2
	case PermissionRead:
		return 1
	default: // PermissionNone or empty
		return 0
	}
}

// findUncoveredWorkerPermissions returns the worker permission scopes (formatted as
// "scope: level") that the caller's declared permissions do not cover. A scope is
// uncovered when the caller grants a strictly lower level than the worker requires.
// The result is sorted for deterministic output; an empty result means the caller's
// declared permissions are sufficient for the worker.
func findUncoveredWorkerPermissions(caller, worker *Permissions) []string {
	if worker == nil {
		return nil
	}

	scopes := append(GetAllPermissionScopes(), PermissionCopilotRequests)
	var missing []string
	for _, scope := range scopes {
		workerLevel, workerWants := worker.Get(scope)
		if !workerWants || workerLevel == PermissionNone {
			continue
		}

		callerLevel := PermissionNone
		if caller != nil {
			if level, has := caller.Get(scope); has {
				callerLevel = level
			}
		}

		if permissionLevelRank(callerLevel) < permissionLevelRank(workerLevel) {
			missing = append(missing, fmt.Sprintf("%s: %s", scope, workerLevel))
		}
	}

	sort.Strings(missing)
	return missing
}

// extractJobPermissionsFromParsedWorkflow extracts and merges all job-level permissions
// from a parsed GitHub Actions workflow map. Returns the union of all jobs' permissions.
//
// Limitation: only explicit per-job permissions blocks are examined. Jobs that omit
// a permissions block inherit from the workflow-level permissions key and are therefore
// not counted here. If a worker workflow relies on workflow-level permissions inheritance
// instead of declaring permissions on each job, the returned set may be incomplete and
// the call-* job could still under-grant permissions at runtime. Workers called via
// call-workflow should declare explicit per-job permissions to ensure reliable extraction.
func extractJobPermissionsFromParsedWorkflow(workflow map[string]any) *Permissions {
	merged := NewPermissions()

	jobsSection, ok := workflow["jobs"]
	if !ok {
		return merged
	}

	jobsMap, ok := jobsSection.(map[string]any)
	if !ok {
		return merged
	}

	for jobName, jobConfig := range jobsMap {
		jobMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}

		permsValue, hasPerms := jobMap["permissions"]
		if !hasPerms {
			callWorkflowPermissionsLog.Printf("Job '%s' has no permissions block, skipping", jobName)
			continue
		}

		jobPerms := NewPermissionsParserFromValue(permsValue).ToPermissions()
		callWorkflowPermissionsLog.Printf("Merging permissions from job '%s'", jobName)
		merged.Merge(jobPerms)
	}

	return merged
}

// extractCallWorkflowPermissions returns the permission superset required by the worker
// workflow identified by workflowName. It resolves the file in priority order:
// .lock.yml > .yml > .md (same-batch compilation target).
//
// For compiled files (.lock.yml / .yml), permissions are extracted from each job's
// permissions block and unioned together. For .md sources, the frontmatter-level
// permissions field is used as a proxy (the compiler will turn it into per-job
// permissions when the worker is eventually compiled).
//
// The result is merged with the caller's declared permissions to form the effective
// permission envelope for the call-workflow job (see buildCallWorkflowJobs). This ensures
// the caller job always grants at least what the worker requires, preventing GitHub from
// rejecting the run at startup when the worker requests a level higher than the caller granted.
//
// Returns nil only when no workflow file is found or (for .md sources) when no
// permissions are present in the frontmatter. For compiled YAML workers the
// function always returns a non-nil *Permissions (possibly empty) because
// extractJobPermissionsFromParsedWorkflow initialises a fresh Permissions map
// regardless of whether any jobs declare a permissions block.
func extractCallWorkflowPermissions(workflowName, markdownPath string) (*Permissions, error) {
	fileResult, err := findWorkflowFile(workflowName, markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow file for '%s': %w", workflowName, err)
	}

	// Priority: .lock.yml > .yml > .md
	if fileResult.lockExists {
		return extractPermissionsFromYAMLFile(fileResult.lockPath)
	}

	if fileResult.ymlExists {
		return extractPermissionsFromYAMLFile(fileResult.ymlPath)
	}

	if fileResult.mdExists {
		return extractPermissionsFromMDFile(fileResult.mdPath)
	}

	// No file found — return nil so the caller omits the permissions block.
	callWorkflowPermissionsLog.Printf("No workflow file found for '%s', skipping permissions", workflowName)
	return nil, nil
}

// extractPermissionsFromYAMLFile reads a .lock.yml or .yml workflow file, parses it,
// and returns the merged permissions from all its jobs.
func extractPermissionsFromYAMLFile(filePath string) (*Permissions, error) {
	workflow, err := readWorkflowYAML(filePath)
	if err != nil {
		return nil, err
	}

	perms := extractJobPermissionsFromParsedWorkflow(workflow)
	callWorkflowPermissionsLog.Printf("Extracted permissions from YAML file %s", filePath)
	return perms, nil
}

// extractPermissionsFromMDFile reads a .md workflow source and uses the frontmatter-level
// permissions field as a proxy for the job permissions that will be generated when the
// worker is compiled.
func extractPermissionsFromMDFile(mdPath string) (*Permissions, error) {
	// mdPath originates from findWorkflowFile(), which validates paths via
	// isPathWithinDir() to prevent directory traversal before returning them.
	content, err := os.ReadFile(mdPath) // #nosec G304 -- path pre-validated by findWorkflowFile() via isPathWithinDir()
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow source %s: %w", mdPath, err)
	}

	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result == nil {
		callWorkflowPermissionsLog.Printf("Failed to extract frontmatter from %s: %v", mdPath, err)
		return nil, nil
	}

	permsValue, hasPerms := result.Frontmatter["permissions"]
	if !hasPerms {
		callWorkflowPermissionsLog.Printf("No permissions in frontmatter of %s", mdPath)
		return nil, nil
	}

	perms := NewPermissionsParserFromValue(permsValue).ToPermissions()
	callWorkflowPermissionsLog.Printf("Extracted permissions from .md source %s", mdPath)
	return perms, nil
}
