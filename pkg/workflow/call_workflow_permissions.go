package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var callWorkflowPermissionsLog = logger.New("workflow:call_workflow_permissions")

// extractJobPermissionsFromParsedWorkflow extracts and merges all job-level permissions
// from a parsed GitHub Actions workflow map. Returns the union of all jobs' permissions.
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
// Returns nil when no workflow file is found or no permissions are declared.
// The caller should omit the permissions block on the call-* job in that case.
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
	cleanPath := filepath.Clean(filePath)
	// filePath originates from findWorkflowFile(), which validates all paths via
	// isPathWithinDir() to prevent directory traversal before returning them.
	content, err := os.ReadFile(cleanPath) // #nosec G304 -- path pre-validated by findWorkflowFile() via isPathWithinDir()
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file %s: %w", filePath, err)
	}

	var workflow map[string]any
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow file %s: %w", filePath, err)
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
