package workflow

// Safe Output Environment Variable Manifest
//
// This file provides a comprehensive manifest of environment variable requirements
// for all safe output job types. The manifest enables:
//
// - **Documentation**: Clear reference of what env vars each job type needs
// - **Validation**: Programmatic validation of environment variable configuration
// - **Discovery**: Listing all supported job types and their requirements
//
// # Usage Example
//
//	// Get all supported job types
//	jobTypes := workflow.GetSupportedSafeOutputJobTypes()
//
//	// Get required variables for validation
//	required, err := workflow.GetRequiredEnvVarsForJobType("create_pull_request")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate environment configuration
//	providedVars := map[string]string{
//	    "GH_AW_WORKFLOW_NAME": "my-workflow",
//	    "GITHUB_TOKEN": "ghp_...",
//	}
//	missing := workflow.ValidateSafeOutputJobEnvVars("noop", providedVars)
//	if len(missing) > 0 {
//	    log.Printf("Missing required variables: %v", missing)
//	}
//
// # Future Integration
//
// This manifest is designed to be used by:
// - Workflow compiler for compile-time validation
// - CLI tools for configuration verification
// - Documentation generation for environment variables
// - Testing frameworks for mock environment setup
//
// The manifest is currently a reference implementation. Future work may integrate
// this with the workflow compiler's validation pipeline to catch configuration
// errors at compile time rather than runtime.

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var envManifestLog = logger.New("workflow:safe_output_env_manifest")

// SafeOutputEnvVar represents an environment variable requirement for a safe output job type
type SafeOutputEnvVar struct {
	Name        string `json:"name"`                   // Environment variable name (e.g., "GH_AW_WORKFLOW_ID")
	Required    bool   `json:"required"`               // Whether this variable is required
	Description string `json:"description"`            // Human-readable description
	DefaultValue string `json:"default_value,omitempty"` // Default value if not set (empty for required vars)
}

// SafeOutputJobManifest defines the environment variable requirements for a safe output job type
type SafeOutputJobManifest struct {
	JobType     string             `json:"job_type"`     // Job type identifier (e.g., "create_pull_request")
	Description string             `json:"description"`  // Human-readable description of the job type
	EnvVars     []SafeOutputEnvVar `json:"env_vars"`     // Environment variables for this job type
}

// GetSafeOutputEnvManifest returns the complete environment variable manifest for all safe output job types
func GetSafeOutputEnvManifest() map[string]SafeOutputJobManifest {
	envManifestLog.Print("Building safe output environment variable manifest")
	
	// Common variables that are set for all safe output jobs by buildStandardSafeOutputEnvVars()
	commonEnvVars := []SafeOutputEnvVar{
		{Name: "GH_AW_WORKFLOW_NAME", Required: true, Description: "Workflow name"},
		{Name: "GH_AW_WORKFLOW_SOURCE", Required: false, Description: "Workflow source file path"},
		{Name: "GH_AW_WORKFLOW_SOURCE_URL", Required: false, Description: "URL to workflow source file"},
		{Name: "GH_AW_TRACKER_ID", Required: false, Description: "Tracker ID for workflow runs"},
		{Name: "GH_AW_ENGINE_ID", Required: false, Description: "AI engine identifier"},
		{Name: "GH_AW_ENGINE_VERSION", Required: false, Description: "AI engine version"},
		{Name: "GH_AW_ENGINE_MODEL", Required: false, Description: "AI engine model"},
		{Name: "GH_AW_SAFE_OUTPUTS_STAGED", Required: false, Description: "Set to 'true' when in staged/trial mode"},
		{Name: "GH_AW_TARGET_REPO_SLUG", Required: false, Description: "Target repository for cross-repo operations (owner/repo)"},
		{Name: "GH_AW_SAFE_OUTPUT_MESSAGES", Required: false, Description: "JSON configuration for custom messages"},
		{Name: "GITHUB_TOKEN", Required: true, Description: "GitHub token for API calls"},
	}

	manifest := map[string]SafeOutputJobManifest{
		"create_pull_request": {
			JobType:     "create_pull_request",
			Description: "Creates a pull request from agent code changes",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_WORKFLOW_ID", Required: true, Description: "Main job name used for branch naming"},
				SafeOutputEnvVar{Name: "GH_AW_BASE_BRANCH", Required: true, Description: "Base branch from github.ref_name"},
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_PR_TITLE_PREFIX", Required: false, Description: "Prefix for PR titles"},
				SafeOutputEnvVar{Name: "GH_AW_PR_LABELS", Required: false, Description: "Comma-separated labels to apply to PR"},
				SafeOutputEnvVar{Name: "GH_AW_PR_ALLOWED_LABELS", Required: false, Description: "Comma-separated list of allowed labels"},
				SafeOutputEnvVar{Name: "GH_AW_PR_DRAFT", Required: false, Description: "Set to 'true' or 'false' for draft status", DefaultValue: "true"},
				SafeOutputEnvVar{Name: "GH_AW_PR_IF_NO_CHANGES", Required: false, Description: "Behavior when no changes: 'warn', 'error', or 'ignore'", DefaultValue: "warn"},
				SafeOutputEnvVar{Name: "GH_AW_PR_ALLOW_EMPTY", Required: false, Description: "Allow creating PR without changes", DefaultValue: "false"},
				SafeOutputEnvVar{Name: "GH_AW_MAX_PATCH_SIZE", Required: false, Description: "Maximum patch size in KB", DefaultValue: "1024"},
				SafeOutputEnvVar{Name: "GH_AW_PR_EXPIRES", Required: false, Description: "Days until PR expires and should be closed"},
				SafeOutputEnvVar{Name: "GH_AW_COMMENT_ID", Required: false, Description: "Comment ID from activation job (when reaction is enabled)"},
				SafeOutputEnvVar{Name: "GH_AW_COMMENT_REPO", Required: false, Description: "Comment repository from activation job (when reaction is enabled)"},
			),
		},
		"add_comment": {
			JobType:     "add_comment",
			Description: "Adds comments to issues, pull requests, or discussions",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_COMMENT_TARGET", Required: false, Description: "Target for comment: 'triggering' (default), '*' (any), or issue number"},
				SafeOutputEnvVar{Name: "GITHUB_AW_COMMENT_DISCUSSION", Required: false, Description: "Set to 'true' to target discussion comments"},
				SafeOutputEnvVar{Name: "GH_AW_HIDE_OLDER_COMMENTS", Required: false, Description: "Set to 'true' to minimize older comments"},
				SafeOutputEnvVar{Name: "GH_AW_ALLOWED_REASONS", Required: false, Description: "JSON array of allowed reasons for hiding comments"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_ISSUE_URL", Required: false, Description: "Issue URL output from create_issue job"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_ISSUE_NUMBER", Required: false, Description: "Issue number output from create_issue job"},
				SafeOutputEnvVar{Name: "GH_AW_TEMPORARY_ID_MAP", Required: false, Description: "Temporary ID map from create_issue job"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_DISCUSSION_URL", Required: false, Description: "Discussion URL output from create_discussion job"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_DISCUSSION_NUMBER", Required: false, Description: "Discussion number output from create_discussion job"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_PULL_REQUEST_URL", Required: false, Description: "PR URL output from create_pull_request job"},
				SafeOutputEnvVar{Name: "GH_AW_CREATED_PULL_REQUEST_NUMBER", Required: false, Description: "PR number output from create_pull_request job"},
			),
		},
		"create_issue": {
			JobType:     "create_issue",
			Description: "Creates GitHub issues from agent output",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_ISSUE_TITLE_PREFIX", Required: false, Description: "Prefix for issue titles"},
				SafeOutputEnvVar{Name: "GH_AW_ISSUE_LABELS", Required: false, Description: "Comma-separated labels to apply to issues"},
				SafeOutputEnvVar{Name: "GH_AW_ASSIGN_COPILOT", Required: false, Description: "Set to 'true' to assign copilot to created issues"},
			),
		},
		"create_discussion": {
			JobType:     "create_discussion",
			Description: "Creates GitHub discussions from agent output",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_DISCUSSION_CATEGORY", Required: false, Description: "Discussion category ID or name"},
				SafeOutputEnvVar{Name: "GH_AW_DISCUSSION_TITLE_PREFIX", Required: false, Description: "Prefix for discussion titles"},
				SafeOutputEnvVar{Name: "GH_AW_DISCUSSION_LABELS", Required: false, Description: "Comma-separated labels to apply to discussions"},
				SafeOutputEnvVar{Name: "GH_AW_CLOSE_OLDER_DISCUSSIONS", Required: false, Description: "Set to 'true' to close older discussions with same prefix or labels"},
			),
		},
		"add_labels": {
			JobType:     "add_labels",
			Description: "Adds labels to issues or pull requests",
			EnvVars:     append([]SafeOutputEnvVar{}, commonEnvVars...),
		},
		"missing_tool": {
			JobType:     "missing_tool",
			Description: "Reports missing tools or functionality requested by the agent",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_MISSING_TOOL_MAX", Required: false, Description: "Maximum number of missing tools to report"},
			),
		},
		"noop": {
			JobType:     "noop",
			Description: "No-operation job that logs messages without taking GitHub API actions",
			EnvVars:     append([]SafeOutputEnvVar{}, commonEnvVars...),
		},
		"create_agent_task": {
			JobType:     "create_agent_task",
			Description: "Creates agent tasks for follow-up work",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GITHUB_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GITHUB_REPOSITORY", Required: true, Description: "Repository slug (owner/repo)"},
				SafeOutputEnvVar{Name: "GITHUB_AW_TARGET_REPO", Required: false, Description: "Target repository for agent tasks"},
				SafeOutputEnvVar{Name: "GITHUB_AW_AGENT_TASK_BASE", Required: false, Description: "Base configuration for agent tasks"},
				SafeOutputEnvVar{Name: "GITHUB_REF_NAME", Required: false, Description: "Current branch name"},
			),
		},
		"create_code_scanning_alert": {
			JobType:     "create_code_scanning_alert",
			Description: "Creates code scanning alerts from security findings",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_WORKFLOW_FILENAME", Required: true, Description: "Workflow filename"},
				SafeOutputEnvVar{Name: "GH_AW_SECURITY_REPORT_DRIVER", Required: false, Description: "Driver for security reporting"},
				SafeOutputEnvVar{Name: "GH_AW_SECURITY_REPORT_MAX", Required: false, Description: "Maximum number of security reports"},
			),
		},
		"create_pr_review_comment": {
			JobType:     "create_pr_review_comment",
			Description: "Creates review comments on pull requests",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_PR_REVIEW_COMMENT_TARGET", Required: false, Description: "Target PR for review comments"},
				SafeOutputEnvVar{Name: "GH_AW_PR_REVIEW_COMMENT_SIDE", Required: false, Description: "Side of diff: 'LEFT' or 'RIGHT'"},
			),
		},
		"push_to_pull_request_branch": {
			JobType:     "push_to_pull_request_branch",
			Description: "Pushes code changes to an existing pull request branch",
			EnvVars: append(commonEnvVars,
				SafeOutputEnvVar{Name: "GH_AW_AGENT_OUTPUT", Required: true, Description: "Path to agent output file"},
				SafeOutputEnvVar{Name: "GH_AW_PUSH_TARGET", Required: true, Description: "Target for push operations"},
				SafeOutputEnvVar{Name: "GH_AW_PR_TITLE_PREFIX", Required: false, Description: "Prefix for PR titles"},
				SafeOutputEnvVar{Name: "GH_AW_PR_LABELS", Required: false, Description: "Comma-separated labels to apply"},
				SafeOutputEnvVar{Name: "GH_AW_COMMIT_TITLE_SUFFIX", Required: false, Description: "Suffix for commit messages"},
				SafeOutputEnvVar{Name: "GH_AW_PUSH_IF_NO_CHANGES", Required: false, Description: "Behavior when no changes to push"},
				SafeOutputEnvVar{Name: "GH_AW_MAX_PATCH_SIZE", Required: false, Description: "Maximum patch size in KB", DefaultValue: "1024"},
			),
		},
	}

	envManifestLog.Printf("Built manifest for %d safe output job types", len(manifest))
	return manifest
}

// GetRequiredEnvVarsForJobType returns the required environment variables for a specific safe output job type
func GetRequiredEnvVarsForJobType(jobType string) ([]string, error) {
	manifest := GetSafeOutputEnvManifest()
	
	jobManifest, exists := manifest[jobType]
	if !exists {
		return nil, fmt.Errorf("unknown safe output job type: %s", jobType)
	}

	var required []string
	for _, envVar := range jobManifest.EnvVars {
		if envVar.Required {
			required = append(required, envVar.Name)
		}
	}

	return required, nil
}

// GetAllEnvVarsForJobType returns all environment variables (required and optional) for a specific safe output job type
func GetAllEnvVarsForJobType(jobType string) ([]SafeOutputEnvVar, error) {
	manifest := GetSafeOutputEnvManifest()
	
	jobManifest, exists := manifest[jobType]
	if !exists {
		return nil, fmt.Errorf("unknown safe output job type: %s", jobType)
	}

	return jobManifest.EnvVars, nil
}

// ValidateSafeOutputJobEnvVars validates that required environment variables are set for a job type
// This is intended for use during workflow compilation to catch configuration errors early
func ValidateSafeOutputJobEnvVars(jobType string, providedEnvVars map[string]string) []string {
	requiredVars, err := GetRequiredEnvVarsForJobType(jobType)
	if err != nil {
		envManifestLog.Printf("Warning: unknown job type %s during validation", jobType)
		return nil
	}

	var missing []string
	for _, required := range requiredVars {
		if _, exists := providedEnvVars[required]; !exists {
			missing = append(missing, required)
		}
	}

	return missing
}

// GetSupportedSafeOutputJobTypes returns a list of all supported safe output job types
func GetSupportedSafeOutputJobTypes() []string {
	manifest := GetSafeOutputEnvManifest()
	
	jobTypes := make([]string, 0, len(manifest))
	for jobType := range manifest {
		jobTypes = append(jobTypes, jobType)
	}

	return jobTypes
}
