package workflow

import (
	"errors"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var toolsValidationLog = logger.New("workflow:tools_validation")

// validateBashToolConfig validates that bash tool configuration is explicit (not nil/anonymous)
func validateBashToolConfig(tools *Tools, workflowName string) error {
	if tools == nil {
		return nil
	}

	// Check if bash is present in the raw map but Bash field is nil
	// This indicates the anonymous syntax (bash:) was used
	if rawMap := tools.ToMap(); rawMap != nil {
		if _, hasBash := rawMap["bash"]; hasBash && tools.Bash == nil {
			toolsValidationLog.Printf("Invalid bash tool configuration in workflow: %s", workflowName)
			return errors.New("invalid bash tool configuration: anonymous syntax 'bash:' is not supported. Use 'bash: true' (enable all commands), 'bash: false' (disable), or 'bash: [\"cmd1\", \"cmd2\"]' (specific commands). Run 'gh aw fix' to automatically migrate")
		}
	}

	return nil
}

// isGitToolAllowed checks if git commands are allowed in bash tool configuration
func isGitToolAllowed(tools *Tools) bool {
	if tools == nil {
		// No tools configured - defaults will be applied which include git for PR operations
		return true
	}

	if tools.Bash == nil {
		// No bash tool configured - defaults will be applied which include git for PR operations
		return true
	}

	// If AllowedCommands is nil or empty, check which case it is:
	// - nil AllowedCommands = bash: true (all commands allowed, including git)
	// - empty slice = bash: false (explicitly disabled)
	if tools.Bash.AllowedCommands == nil {
		// bash: true - all commands allowed
		return true
	}

	if len(tools.Bash.AllowedCommands) == 0 {
		// bash: false or bash: [] - explicitly disabled or no commands
		return false
	}

	// Check if git is in the allowed commands list
	for _, cmd := range tools.Bash.AllowedCommands {
		if cmd == "*" {
			// Wildcard allows all commands
			return true
		}
		if cmd == "git" {
			// Exact match for git command
			return true
		}
		// Check for git with wildcards: "git *", "git:*", "git checkout:*", etc.
		if strings.HasPrefix(cmd, "git ") || strings.HasPrefix(cmd, "git:") {
			return true
		}
	}

	return false
}

// validateGitHubToolConfig validates that the GitHub tool configuration does not
// specify both app and github-token at the same time, as only one authentication
// method is allowed.
func validateGitHubToolConfig(tools *Tools, workflowName string) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	if tools.GitHub.App != nil && tools.GitHub.GitHubToken != "" {
		toolsValidationLog.Printf("Invalid GitHub tool configuration in workflow: %s", workflowName)
		return errors.New("invalid GitHub tool configuration: 'tools.github.app' and 'tools.github.github-token' cannot both be set. Use one authentication method: either 'app' (GitHub App) or 'github-token' (personal access token)")
	}

	return nil
}

// validateGitHubGuardPolicy validates the GitHub guard policy configuration.
// Guard policy fields (repos, min-integrity) are specified flat under github:.
// Both fields must be present if either is specified.
func validateGitHubGuardPolicy(tools *Tools, workflowName string) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	github := tools.GitHub
	hasRepos := github.Repos != nil
	hasMinIntegrity := github.MinIntegrity != ""

	// No guard policy fields present - nothing to validate
	if !hasRepos && !hasMinIntegrity {
		return nil
	}

	// Validate repos field (required when min-integrity is set)
	if !hasRepos {
		toolsValidationLog.Printf("Missing repos in guard policy for workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'github.repos' is required. Use 'all', 'public', or an array of repository patterns (e.g., ['owner/repo', 'owner/*'])")
	}

	// Validate repos format
	if err := validateReposScope(github.Repos, workflowName); err != nil {
		return err
	}

	// Validate min-integrity field (required when repos is set)
	if !hasMinIntegrity {
		toolsValidationLog.Printf("Missing min-integrity in guard policy for workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'github.min-integrity' is required. Valid values: 'none', 'reader', 'writer', 'merged'")
	}

	// Validate min-integrity value
	validIntegrityLevels := map[GitHubIntegrityLevel]bool{
		GitHubIntegrityNone:   true,
		GitHubIntegrityReader: true,
		GitHubIntegrityWriter: true,
		GitHubIntegrityMerged: true,
	}

	if !validIntegrityLevels[github.MinIntegrity] {
		toolsValidationLog.Printf("Invalid min-integrity level '%s' in workflow: %s", github.MinIntegrity, workflowName)
		return errors.New("invalid guard policy: 'github.min-integrity' must be one of: 'none', 'reader', 'writer', 'merged'. Got: '" + string(github.MinIntegrity) + "'")
	}

	return nil
}

// validateReposScope validates the repos field in the guard policy
func validateReposScope(repos any, workflowName string) error {
	// Case 1: String value ("all" or "public")
	if reposStr, ok := repos.(string); ok {
		if reposStr != "all" && reposStr != "public" {
			toolsValidationLog.Printf("Invalid repos string '%s' in workflow: %s", reposStr, workflowName)
			return errors.New("invalid guard policy: 'github.repos' string must be 'all' or 'public'. Got: '" + reposStr + "'")
		}
		return nil
	}

	// Case 2a: Array of patterns from YAML parsing ([]any)
	if reposArray, ok := repos.([]any); ok {
		if len(reposArray) == 0 {
			toolsValidationLog.Printf("Empty repos array in workflow: %s", workflowName)
			return errors.New("invalid guard policy: 'github.repos' array cannot be empty. Provide at least one repository pattern")
		}

		for i, item := range reposArray {
			pattern, ok := item.(string)
			if !ok {
				toolsValidationLog.Printf("Non-string item in repos array at index %d in workflow: %s", i, workflowName)
				return errors.New("invalid guard policy: 'github.repos' array must contain only strings")
			}

			if err := validateRepoPattern(pattern, workflowName); err != nil {
				return err
			}
		}

		return nil
	}

	// Case 2b: Array of patterns from programmatic construction ([]string)
	if reposArray, ok := repos.([]string); ok {
		if len(reposArray) == 0 {
			toolsValidationLog.Printf("Empty repos array in workflow: %s", workflowName)
			return errors.New("invalid guard policy: 'github.repos' array cannot be empty. Provide at least one repository pattern")
		}

		for _, pattern := range reposArray {
			if err := validateRepoPattern(pattern, workflowName); err != nil {
				return err
			}
		}

		return nil
	}

	// Invalid type
	toolsValidationLog.Printf("Invalid repos type in workflow: %s", workflowName)
	return errors.New("invalid guard policy: 'github.repos' must be 'all', 'public', or an array of repository patterns")
}

// validateRepoPattern validates a single repository pattern
func validateRepoPattern(pattern string, workflowName string) error {
	// Pattern must be lowercase
	if strings.ToLower(pattern) != pattern {
		toolsValidationLog.Printf("Repository pattern '%s' is not lowercase in workflow: %s", pattern, workflowName)
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' must be lowercase")
	}

	// Check for valid pattern formats:
	// 1. owner/repo (exact match)
	// 2. owner/* (owner wildcard)
	// 3. owner/re* (repository prefix wildcard)
	parts := strings.Split(pattern, "/")
	if len(parts) != 2 {
		toolsValidationLog.Printf("Invalid repository pattern '%s' in workflow: %s", pattern, workflowName)
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' must be in format 'owner/repo', 'owner/*', or 'owner/prefix*'")
	}

	owner := parts[0]
	repo := parts[1]

	// Validate owner part (must be non-empty and contain only valid characters)
	if owner == "" {
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' has empty owner")
	}

	if !isValidOwnerOrRepo(owner) {
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' has invalid owner. Must contain only lowercase letters, numbers, hyphens, and underscores")
	}

	// Validate repo part
	if repo == "" {
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' has empty repository name")
	}

	// Allow wildcard '*' or prefix with trailing '*'
	if repo != "*" && !isValidOwnerOrRepo(strings.TrimSuffix(repo, "*")) {
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' has invalid repository name. Must contain only lowercase letters, numbers, hyphens, underscores, or be '*' or 'prefix*'")
	}

	// Validate that wildcard is only at the end (not in the middle)
	if strings.Contains(strings.TrimSuffix(repo, "*"), "*") {
		return errors.New("invalid guard policy: repository pattern '" + pattern + "' has wildcard in the middle. Wildcards only allowed at the end (e.g., 'prefix*')")
	}

	return nil
}

// isValidOwnerOrRepo checks if a string contains only valid GitHub owner/repo characters
func isValidOwnerOrRepo(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

// Note: validateGitToolForSafeOutputs was removed because git commands are automatically
// injected by the compiler when safe-outputs needs them (see compiler_safe_outputs.go).
// The validation was misleading - it would fail even though the compiler would add the
// necessary git commands during compilation.
