package workflow

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
)

var toolsValidationLog = newValidationLogger("tools")

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

// validateGitHubReadOnly validates that read-only: false is not set for the GitHub tool.
// The GitHub MCP server always operates in read-only mode; write access is not permitted.
func validateGitHubReadOnly(tools *Tools, workflowName string) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	if !tools.GitHub.ReadOnly {
		toolsValidationLog.Printf("Invalid read-only configuration in workflow: %s", workflowName)
		return errors.New("invalid GitHub tool configuration: 'tools.github.read-only: false' is not allowed. The GitHub MCP server always operates in read-only mode. Remove the 'read-only' field or set it to 'true'")
	}

	return nil
}

// validateGitHubToolConfig validates that the GitHub tool configuration does not
// specify both app and github-token at the same time, as only one authentication
// method is allowed.
func validateGitHubToolConfig(tools *Tools, workflowName string) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	if tools.GitHub.GitHubApp != nil && tools.GitHub.GitHubToken != "" {
		toolsValidationLog.Printf("Invalid GitHub tool configuration in workflow: %s", workflowName)
		return errors.New("invalid GitHub tool configuration: 'tools.github.github-app' and 'tools.github.github-token' cannot both be set. Use one authentication method: either 'github-app' (GitHub App) or 'github-token' (personal access token)")
	}

	return nil
}

// validateGitHubGuardPolicy validates the GitHub guard policy configuration.
// Guard policy fields (allowed-repos, min-integrity) are specified flat under github:.
// Note: 'repos' is a deprecated alias for 'allowed-repos'.
// If allowed-repos (or deprecated alias repos) is not specified but min-integrity is, allowed-repos defaults to "all".
func validateGitHubGuardPolicy(tools *Tools, workflowName string) error {
	if tools == nil || tools.GitHub == nil {
		return nil
	}

	github := tools.GitHub
	// AllowedRepos is populated from either 'allowed-repos' (preferred) or deprecated 'repos' during parsing
	hasRepos := github.AllowedRepos != nil
	hasMinIntegrity := github.MinIntegrity != ""
	// blocked-users / approval-labels / trusted-users can be an array or a
	// GitHub Actions expression string.
	hasBlockedUsers := len(github.BlockedUsers) > 0 || github.BlockedUsersExpr != ""
	hasApprovalLabels := len(github.ApprovalLabels) > 0 || github.ApprovalLabelsExpr != ""
	hasTrustedUsers := len(github.TrustedUsers) > 0 || github.TrustedUsersExpr != ""

	// blocked-users, trusted-users, and approval-labels require a guard policy (min-integrity)
	if (hasBlockedUsers || hasApprovalLabels || hasTrustedUsers) && !hasMinIntegrity {
		toolsValidationLog.Printf("blocked-users/trusted-users/approval-labels without guard policy in workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'github.blocked-users', 'github.trusted-users', and 'github.approval-labels' require 'github.min-integrity' to be set")
	}

	// No guard policy fields present - nothing to validate
	if !hasRepos && !hasMinIntegrity {
		return nil
	}

	// Default allowed-repos to "all" when not specified
	if !hasRepos {
		toolsValidationLog.Printf("Defaulting allowed-repos (repos) to 'all' in guard policy for workflow: %s", workflowName)
		github.AllowedRepos = "all"
	}

	// Validate repos format
	if err := validateReposScope(github.AllowedRepos, workflowName); err != nil {
		return err
	}

	// Validate min-integrity field (required when repos is set)
	if !hasMinIntegrity {
		toolsValidationLog.Printf("Missing min-integrity in guard policy for workflow: %s", workflowName)
		return errors.New("invalid guard policy: 'github.min-integrity' is required. Valid values: 'none', 'unapproved', 'approved', 'merged'")
	}

	// Validate min-integrity value
	validIntegrityLevels := map[GitHubIntegrityLevel]bool{
		GitHubIntegrityNone:       true,
		GitHubIntegrityUnapproved: true,
		GitHubIntegrityApproved:   true,
		GitHubIntegrityMerged:     true,
	}

	if !validIntegrityLevels[github.MinIntegrity] {
		toolsValidationLog.Printf("Invalid min-integrity level '%s' in workflow: %s", github.MinIntegrity, workflowName)
		return errors.New("invalid guard policy: 'github.min-integrity' must be one of: 'none', 'unapproved', 'approved', 'merged'. Got: '" + string(github.MinIntegrity) + "'")
	}

	// Validate blocked-users (must be non-empty strings; expressions are accepted as-is)
	for i, user := range github.BlockedUsers {
		if user == "" {
			toolsValidationLog.Printf("Empty blocked-users entry at index %d in workflow: %s", i, workflowName)
			return errors.New("invalid guard policy: 'github.blocked-users' entries must not be empty strings")
		}
	}

	// Validate approval-labels (must be non-empty strings; expressions are accepted as-is)
	for i, label := range github.ApprovalLabels {
		if label == "" {
			toolsValidationLog.Printf("Empty approval-labels entry at index %d in workflow: %s", i, workflowName)
			return errors.New("invalid guard policy: 'github.approval-labels' entries must not be empty strings")
		}
	}

	// Validate trusted-users (must be non-empty strings; expressions are accepted as-is)
	for i, user := range github.TrustedUsers {
		if user == "" {
			toolsValidationLog.Printf("Empty trusted-users entry at index %d in workflow: %s", i, workflowName)
			return errors.New("invalid guard policy: 'github.trusted-users' entries must not be empty strings")
		}
	}

	return nil
}

// validateReposScope validates the repos field in the guard policy
func validateReposScope(repos any, workflowName string) error {
	// Case 1: String value ("all" or "public")
	if reposStr, ok := repos.(string); ok {
		if reposStr != "all" && reposStr != "public" {
			toolsValidationLog.Printf("Invalid repos string '%s' in workflow: %s", reposStr, workflowName)
			return errors.New("invalid guard policy: 'github.allowed-repos' string must be 'all' or 'public'. Got: '" + reposStr + "'")
		}
		return nil
	}

	// Case 2a: Array of patterns from YAML parsing ([]any)
	if reposArray, ok := repos.([]any); ok {
		if len(reposArray) == 0 {
			toolsValidationLog.Printf("Empty repos array in workflow: %s", workflowName)
			return errors.New("invalid guard policy: 'github.allowed-repos' array cannot be empty. Provide at least one repository pattern")
		}

		for i, item := range reposArray {
			pattern, ok := item.(string)
			if !ok {
				toolsValidationLog.Printf("Non-string item in repos array at index %d in workflow: %s", i, workflowName)
				return errors.New("invalid guard policy: 'github.allowed-repos' array must contain only strings")
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
			return errors.New("invalid guard policy: 'github.allowed-repos' array cannot be empty. Provide at least one repository pattern")
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
	return errors.New("invalid guard policy: 'github.allowed-repos' must be 'all', 'public', or an array of repository patterns")
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

// ValidateGitHubToolsAgainstToolsets validates that all allowed GitHub tools have their
// corresponding toolsets enabled in the configuration.
func ValidateGitHubToolsAgainstToolsets(allowedTools []string, enabledToolsets []string) error {
	githubToolToToolsetLog.Printf("Validating GitHub tools against toolsets: allowed_tools=%d, enabled_toolsets=%d", len(allowedTools), len(enabledToolsets))

	if len(allowedTools) == 0 {
		githubToolToToolsetLog.Print("No tools to validate, skipping")
		// No specific tools restricted, validation not needed
		return nil
	}

	// Create a set of enabled toolsets for fast lookup
	enabledSet := make(map[string]bool)
	for _, toolset := range enabledToolsets {
		enabledSet[toolset] = true
	}
	githubToolToToolsetLog.Printf("Enabled toolsets: %v", enabledToolsets)

	// Track missing toolsets and which tools need them
	missingToolsets := make(map[string][]string) // toolset -> list of tools that need it

	// Track unknown tools for suggestions
	var unknownTools []string
	var suggestions []string

	for _, tool := range allowedTools {
		// Skip wildcard - it means "allow all tools"
		if tool == "*" {
			continue
		}

		requiredToolset, exists := GitHubToolToToolsetMap[tool]
		if !exists {
			githubToolToToolsetLog.Printf("Tool %s not found in mapping, checking for typo", tool)

			// Get all valid tool names for suggestion
			validTools := make([]string, 0, len(GitHubToolToToolsetMap))
			for validTool := range GitHubToolToToolsetMap {
				validTools = append(validTools, validTool)
			}
			sort.Strings(validTools)

			// Try to find close matches
			matches := parser.FindClosestMatches(tool, validTools, 1)
			if len(matches) > 0 {
				githubToolToToolsetLog.Printf("Found suggestion for unknown tool %s: %s", tool, matches[0])
				unknownTools = append(unknownTools, tool)
				suggestions = append(suggestions, fmt.Sprintf("%s → %s", tool, matches[0]))
			} else {
				githubToolToToolsetLog.Printf("No suggestion found for unknown tool: %s", tool)
				unknownTools = append(unknownTools, tool)
			}
			// Tool not in our mapping - this could be a new tool or a typo
			// We'll skip validation for unknown tools to avoid false positives
			continue
		}

		if !enabledSet[requiredToolset] {
			githubToolToToolsetLog.Printf("Tool %s requires missing toolset: %s", tool, requiredToolset)
			missingToolsets[requiredToolset] = append(missingToolsets[requiredToolset], tool)
		}
	}

	// Report unknown tools with suggestions if any were found
	if len(unknownTools) > 0 {
		githubToolToToolsetLog.Printf("Found %d unknown tools", len(unknownTools))
		var errMsg strings.Builder
		errMsg.WriteString(fmt.Sprintf("Unknown GitHub tool(s): %s\n\n", formatList(unknownTools)))

		if len(suggestions) > 0 {
			errMsg.WriteString("Did you mean:\n")
			for _, s := range suggestions {
				errMsg.WriteString(fmt.Sprintf("  %s\n", s))
			}
			errMsg.WriteString("\n")
		}

		// Show a few examples of valid tools
		validTools := make([]string, 0, len(GitHubToolToToolsetMap))
		for tool := range GitHubToolToToolsetMap {
			validTools = append(validTools, tool)
		}
		sort.Strings(validTools)

		exampleCount := min(10, len(validTools))
		errMsg.WriteString(fmt.Sprintf("Valid GitHub tools include: %s\n\n", formatList(validTools[:exampleCount])))
		errMsg.WriteString("See all tools: https://github.com/github/gh-aw/blob/main/pkg/workflow/data/github_tool_to_toolset.json")

		return fmt.Errorf("%s", errMsg.String())
	}

	if len(missingToolsets) > 0 {
		githubToolToToolsetLog.Printf("Validation failed: missing %d toolsets", len(missingToolsets))
		return NewGitHubToolsetValidationError(missingToolsets)
	}

	githubToolToToolsetLog.Print("Validation successful: all tools have required toolsets")
	return nil
}
