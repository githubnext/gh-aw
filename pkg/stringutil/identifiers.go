package stringutil

import (
	"path/filepath"
	"strings"
)

// NormalizeWorkflowName removes .md and .lock.yml extensions from workflow names.
// This is used to standardize workflow identifiers regardless of the file format.
//
// The function checks for extensions in order of specificity:
// 1. Removes .lock.yml extension (the compiled workflow format)
// 2. Removes .md extension (the markdown source format)
// 3. Returns the name unchanged if no recognized extension is found
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeWorkflowName("weekly-research")           // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.md")        // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.lock.yml")  // returns "weekly-research"
//	NormalizeWorkflowName("my.workflow.md")            // returns "my.workflow"
func NormalizeWorkflowName(name string) string {
	// Remove .lock.yml extension first (longer extension)
	if strings.HasSuffix(name, ".lock.yml") {
		return strings.TrimSuffix(name, ".lock.yml")
	}

	// Remove .md extension
	if strings.HasSuffix(name, ".md") {
		return strings.TrimSuffix(name, ".md")
	}

	return name
}

// NormalizeSafeOutputIdentifier converts dashes to underscores for safe output identifiers.
// This standardizes identifier format from the user-facing dash-separated format
// to the internal underscore-separated format used in safe outputs configuration.
//
// Both dash-separated and underscore-separated formats are valid inputs.
// This function simply standardizes to the internal representation.
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeSafeOutputIdentifier("create-issue")      // returns "create_issue"
//	NormalizeSafeOutputIdentifier("create_issue")      // returns "create_issue" (unchanged)
//	NormalizeSafeOutputIdentifier("add-comment")       // returns "add_comment"
//	NormalizeSafeOutputIdentifier("update-pr")         // returns "update_pr"
func NormalizeSafeOutputIdentifier(identifier string) string {
	return strings.ReplaceAll(identifier, "-", "_")
}

// MarkdownToLockFile converts a workflow markdown file path to its compiled lock file path.
// This is the standard transformation for agentic workflow files.
//
// The function removes the .md extension and adds .lock.yml extension.
// If the input already has a .lock.yml extension, it returns the path unchanged.
//
// Examples:
//
//	MarkdownToLockFile("weekly-research.md")                    // returns "weekly-research.lock.yml"
//	MarkdownToLockFile(".github/workflows/test.md")             // returns ".github/workflows/test.lock.yml"
//	MarkdownToLockFile("workflow.lock.yml")                     // returns "workflow.lock.yml" (unchanged)
//	MarkdownToLockFile("my.workflow.md")                        // returns "my.workflow.lock.yml"
func MarkdownToLockFile(mdPath string) string {
	// If already a lock file, return unchanged
	if strings.HasSuffix(mdPath, ".lock.yml") {
		return mdPath
	}

	cleaned := filepath.Clean(mdPath)
	return strings.TrimSuffix(cleaned, ".md") + ".lock.yml"
}

// LockFileToMarkdown converts a compiled lock file path back to its markdown source path.
// This is used when navigating from compiled workflows back to source files.
//
// The function removes the .lock.yml extension and adds .md extension.
// If the input already has a .md extension, it returns the path unchanged.
//
// Examples:
//
//	LockFileToMarkdown("weekly-research.lock.yml")              // returns "weekly-research.md"
//	LockFileToMarkdown(".github/workflows/test.lock.yml")       // returns ".github/workflows/test.md"
//	LockFileToMarkdown("workflow.md")                           // returns "workflow.md" (unchanged)
//	LockFileToMarkdown("my.workflow.lock.yml")                  // returns "my.workflow.md"
func LockFileToMarkdown(lockPath string) string {
	// If already a markdown file, return unchanged
	if strings.HasSuffix(lockPath, ".md") {
		return lockPath
	}

	cleaned := filepath.Clean(lockPath)
	return strings.TrimSuffix(cleaned, ".lock.yml") + ".md"
}

// CampaignSpecToOrchestrator converts a campaign specification file to its generated orchestrator file.
// Campaign specs (.campaign.md) generate orchestrator workflows (.campaign.g.md).
//
// The function removes the .campaign.md extension and adds .campaign.g.md extension.
//
// Examples:
//
//	CampaignSpecToOrchestrator("test.campaign.md")              // returns "test.campaign.g.md"
//	CampaignSpecToOrchestrator(".github/workflows/prod.campaign.md") // returns ".github/workflows/prod.campaign.g.md"
func CampaignSpecToOrchestrator(specPath string) string {
	return strings.TrimSuffix(specPath, ".campaign.md") + ".campaign.g.md"
}

// CampaignOrchestratorToLockFile converts a campaign orchestrator file to its compiled lock file.
// Campaign orchestrators (.campaign.g.md) compile to lock files (.campaign.lock.yml).
//
// The function removes the .campaign.g.md extension and adds .campaign.lock.yml extension.
//
// Examples:
//
//	CampaignOrchestratorToLockFile("test.campaign.g.md")        // returns "test.campaign.lock.yml"
//	CampaignOrchestratorToLockFile(".github/workflows/prod.campaign.g.md") // returns ".github/workflows/prod.campaign.lock.yml"
func CampaignOrchestratorToLockFile(orchestratorPath string) string {
	baseName := strings.TrimSuffix(orchestratorPath, ".campaign.g.md")
	return baseName + ".campaign.lock.yml"
}

// CampaignSpecToLockFile converts a campaign specification file directly to its compiled lock file.
// This skips the intermediate orchestrator step for convenience.
//
// The function removes the .campaign.md extension and adds .campaign.lock.yml extension.
//
// Examples:
//
//	CampaignSpecToLockFile("test.campaign.md")                  // returns "test.campaign.lock.yml"
//	CampaignSpecToLockFile(".github/workflows/prod.campaign.md") // returns ".github/workflows/prod.campaign.lock.yml"
func CampaignSpecToLockFile(specPath string) string {
	return strings.TrimSuffix(specPath, ".campaign.md") + ".campaign.lock.yml"
}

// IsCampaignSpec returns true if the file path is a campaign specification file.
// Campaign specs end with .campaign.md and are user-authored files that define campaigns.
//
// Examples:
//
//	IsCampaignSpec("test.campaign.md")                          // returns true
//	IsCampaignSpec(".github/workflows/prod.campaign.md")        // returns true
//	IsCampaignSpec("test.campaign.g.md")                        // returns false (orchestrator)
//	IsCampaignSpec("test.md")                                   // returns false (workflow)
//	IsCampaignSpec("test.lock.yml")                             // returns false
func IsCampaignSpec(path string) bool {
	return strings.HasSuffix(path, ".campaign.md")
}

// IsCampaignOrchestrator returns true if the file path is a campaign orchestrator file.
// Campaign orchestrators end with .campaign.g.md and are generated from campaign specs.
// The .g. indicates "generated".
//
// Examples:
//
//	IsCampaignOrchestrator("test.campaign.g.md")                // returns true
//	IsCampaignOrchestrator(".github/workflows/prod.campaign.g.md") // returns true
//	IsCampaignOrchestrator("test.campaign.md")                  // returns false (spec)
//	IsCampaignOrchestrator("test.md")                           // returns false (workflow)
//	IsCampaignOrchestrator("test.lock.yml")                     // returns false
func IsCampaignOrchestrator(path string) bool {
	return strings.HasSuffix(path, ".campaign.g.md")
}

// IsAgenticWorkflow returns true if the file path is an agentic workflow file.
// Agentic workflows end with .md but are NOT campaign specs or campaign orchestrators.
//
// Examples:
//
//	IsAgenticWorkflow("test.md")                                // returns true
//	IsAgenticWorkflow("weekly-research.md")                     // returns true
//	IsAgenticWorkflow(".github/workflows/workflow.md")          // returns true
//	IsAgenticWorkflow("test.campaign.md")                       // returns false (campaign spec)
//	IsAgenticWorkflow("test.campaign.g.md")                     // returns false (orchestrator)
//	IsAgenticWorkflow("test.lock.yml")                          // returns false
func IsAgenticWorkflow(path string) bool {
	// Must end with .md
	if !strings.HasSuffix(path, ".md") {
		return false
	}
	// Must NOT be a campaign spec or orchestrator
	return !IsCampaignSpec(path) && !IsCampaignOrchestrator(path)
}

// IsLockFile returns true if the file path is a compiled lock file.
// Lock files end with .lock.yml and can be compiled from agentic workflows or campaign orchestrators.
//
// Examples:
//
//	IsLockFile("test.lock.yml")                                 // returns true
//	IsLockFile("test.campaign.lock.yml")                        // returns true
//	IsLockFile(".github/workflows/workflow.lock.yml")           // returns true
//	IsLockFile("test.md")                                       // returns false
//	IsLockFile("test.campaign.md")                              // returns false
func IsLockFile(path string) bool {
	return strings.HasSuffix(path, ".lock.yml")
}

// IsCampaignLockFile returns true if the file path is a compiled campaign lock file.
// Campaign lock files end with .campaign.lock.yml and are compiled from campaign orchestrators.
//
// Examples:
//
//	IsCampaignLockFile("test.campaign.lock.yml")                // returns true
//	IsCampaignLockFile(".github/workflows/prod.campaign.lock.yml") // returns true
//	IsCampaignLockFile("test.lock.yml")                         // returns false (regular lock)
//	IsCampaignLockFile("test.campaign.md")                      // returns false
//	IsCampaignLockFile("test.md")                               // returns false
func IsCampaignLockFile(path string) bool {
	return strings.HasSuffix(path, ".campaign.lock.yml")
}
