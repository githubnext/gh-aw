package inventory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var inventoryLog = logger.New("cli:inventory")

// ExtractWorkflowName extracts the normalized workflow name from a file path or filename.
// It handles various workflow file types:
//   - Regular workflows: "workflow.md" -> "workflow"
//   - Lock files: "workflow.lock.yml" -> "workflow"
//   - Campaign workflows: "campaign.campaign.md" -> "campaign"
//   - Campaign lock files: "campaign.campaign.lock.yml" -> "campaign"
//   - Generated campaign orchestrators: "campaign.campaign.g.md" -> "campaign"
//   - Paths: ".github/workflows/workflow.md" -> "workflow"
//
// The function returns the base workflow identifier without any file extensions.
func ExtractWorkflowName(path string) string {
	// Get base filename
	base := filepath.Base(path)

	// Handle special cases first before removing generic extensions
	// Generated campaign orchestrators: .campaign.g.md -> base name
	if strings.HasSuffix(base, ".campaign.g.md") {
		return strings.TrimSuffix(base, ".campaign.g.md")
	}

	// Campaign lock files: .campaign.lock.yml -> base name
	if strings.HasSuffix(base, ".campaign.lock.yml") {
		return strings.TrimSuffix(base, ".campaign.lock.yml")
	}

	// Campaign workflows: .campaign.md -> base name
	if strings.HasSuffix(base, ".campaign.md") {
		return strings.TrimSuffix(base, ".campaign.md")
	}

	// Lock files: .lock.yml -> base name
	if strings.HasSuffix(base, ".lock.yml") {
		return strings.TrimSuffix(base, ".lock.yml")
	}

	// Regular workflows: .md -> base name
	if strings.HasSuffix(base, ".md") {
		return strings.TrimSuffix(base, ".md")
	}

	// If no known extension, return as-is
	inventoryLog.Printf("Extracted workflow name: %s -> %s", path, base)
	return base
}

// NormalizeWorkflowName normalizes a workflow name or path provided by user input.
// It strips file extensions and extracts the base workflow name.
//
// Examples:
//   - "my-workflow" -> "my-workflow"
//   - "my-workflow.md" -> "my-workflow"
//   - ".github/workflows/my-workflow.md" -> "my-workflow"
//   - "my-workflow.lock.yml" -> "my-workflow"
//
// This is the same as ExtractWorkflowName but semantically indicates
// it's being used to normalize user-provided input.
func NormalizeWorkflowName(input string) string {
	return ExtractWorkflowName(input)
}

// GetWorkflowPath returns the markdown file path for a workflow name.
// If workflowsDir is empty, it uses ".github/workflows" as the default.
//
// Examples:
//   - GetWorkflowPath("my-workflow", "") -> ".github/workflows/my-workflow.md"
//   - GetWorkflowPath("my-workflow", "/path/to/workflows") -> "/path/to/workflows/my-workflow.md"
func GetWorkflowPath(workflowName, workflowsDir string) string {
	if workflowsDir == "" {
		workflowsDir = ".github/workflows"
	}

	// Ensure the workflow name doesn't have .md extension
	workflowName = NormalizeWorkflowName(workflowName)

	return filepath.Join(workflowsDir, workflowName+".md")
}

// GetLockFilePath returns the lock file path for a workflow.
//
// It handles different workflow types:
//   - Regular workflow: "workflow.md" -> "workflow.lock.yml"
//   - Campaign workflow: "campaign.campaign.md" -> "campaign.campaign.lock.yml"
//   - Generated orchestrator: "campaign.campaign.g.md" -> "campaign.campaign.lock.yml"
//
// If workflowPath is just a name without a path, it uses workflowsDir.
// If workflowsDir is empty, it uses ".github/workflows" as the default.
func GetLockFilePath(workflowPath, workflowsDir string) string {
	dir := filepath.Dir(workflowPath)
	base := filepath.Base(workflowPath)

	// If workflowPath is just a filename, use workflowsDir
	if dir == "." {
		if workflowsDir == "" {
			workflowsDir = ".github/workflows"
		}
		dir = workflowsDir
	}

	// Handle different workflow types
	if strings.HasSuffix(base, ".campaign.g.md") {
		// Generated orchestrator: campaign.campaign.g.md -> campaign.campaign.lock.yml
		lockName := strings.TrimSuffix(base, ".campaign.g.md") + ".campaign.lock.yml"
		return filepath.Join(dir, lockName)
	} else if strings.HasSuffix(base, ".campaign.md") {
		// Campaign workflow: campaign.campaign.md -> campaign.campaign.lock.yml
		lockName := strings.TrimSuffix(base, ".campaign.md") + ".campaign.lock.yml"
		return filepath.Join(dir, lockName)
	} else {
		// Regular workflow: workflow.md -> workflow.lock.yml
		lockName := strings.TrimSuffix(base, ".md") + ".lock.yml"
		return filepath.Join(dir, lockName)
	}
}

// WorkflowType represents the type of workflow file
type WorkflowType int

const (
	// WorkflowTypeRegular is a standard workflow (.md file)
	WorkflowTypeRegular WorkflowType = iota
	// WorkflowTypeCampaign is a campaign spec (.campaign.md file)
	WorkflowTypeCampaign
	// WorkflowTypeCampaignGenerated is a generated campaign orchestrator (.campaign.g.md file)
	WorkflowTypeCampaignGenerated
)

// WorkflowFile represents a discovered workflow file
type WorkflowFile struct {
	Name     string       // Normalized workflow name (without extensions)
	Path     string       // Full path to the markdown file
	Type     WorkflowType // Type of workflow
	LockPath string       // Path to corresponding lock file
}

// isWorkflowFile returns true if the file should be treated as a workflow file.
// README.md files are excluded as they are documentation, not workflows.
func isWorkflowFile(filename string) bool {
	base := strings.ToLower(filepath.Base(filename))
	return base != "readme.md"
}

// ListWorkflowFiles discovers all workflow files in the specified directory.
// If workflowsDir is empty, it uses ".github/workflows" as the default.
//
// Options:
//   - includeCampaigns: include campaign spec files (.campaign.md)
//   - includeGenerated: include generated files (.campaign.g.md, .lock.yml)
//
// By default, only regular workflow .md files are returned, and README.md is excluded.
func ListWorkflowFiles(workflowsDir string, includeCampaigns, includeGenerated bool) ([]WorkflowFile, error) {
	if workflowsDir == "" {
		workflowsDir = ".github/workflows"
	}

	inventoryLog.Printf("Listing workflow files in: %s (campaigns=%v, generated=%v)", workflowsDir, includeCampaigns, includeGenerated)

	// Check if directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		inventoryLog.Printf("Workflows directory does not exist: %s", workflowsDir)
		return nil, err
	}

	// Find all markdown files
	mdFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, err
	}

	var workflows []WorkflowFile

	for _, file := range mdFiles {
		if !isWorkflowFile(file) {
			inventoryLog.Printf("Skipping non-workflow file: %s", file)
			continue
		}

		base := filepath.Base(file)

		// Determine workflow type
		var workflowType WorkflowType
		if strings.HasSuffix(base, ".campaign.g.md") {
			if !includeGenerated {
				inventoryLog.Printf("Skipping generated file: %s", file)
				continue
			}
			workflowType = WorkflowTypeCampaignGenerated
		} else if strings.HasSuffix(base, ".campaign.md") {
			if !includeCampaigns {
				inventoryLog.Printf("Skipping campaign file: %s", file)
				continue
			}
			workflowType = WorkflowTypeCampaign
		} else {
			workflowType = WorkflowTypeRegular
		}

		name := ExtractWorkflowName(file)
		lockPath := GetLockFilePath(file, workflowsDir)

		workflow := WorkflowFile{
			Name:     name,
			Path:     file,
			Type:     workflowType,
			LockPath: lockPath,
		}

		workflows = append(workflows, workflow)
		inventoryLog.Printf("Found workflow: %s (type=%d, lock=%s)", name, workflowType, lockPath)
	}

	inventoryLog.Printf("Found %d workflow files", len(workflows))
	return workflows, nil
}
