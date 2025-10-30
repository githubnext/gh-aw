package workflow

// GitHubToolToToolsetMap maps individual GitHub MCP tools to their respective toolsets
// This mapping is based on the documentation in .github/instructions/github-mcp-server.instructions.md
var GitHubToolToToolsetMap = map[string]string{
	// Context Toolset
	"get_me":           "context",
	"get_teams":        "context",
	"get_team_members": "context",

	// Repos Toolset
	"get_repository":    "repos",
	"get_file_contents": "repos",
	"search_code":       "repos",
	"list_commits":      "repos",
	"get_commit":        "repos",
	"get_latest_release": "repos",
	"list_releases":     "repos",
	"get_release_by_tag": "repos",
	"get_tag":           "repos",
	"list_tags":         "repos",
	"list_branches":     "repos",

	// Issues Toolset
	"issue_read":          "issues",
	"list_issues":         "issues",
	"create_issue":        "issues",
	"update_issue":        "issues",
	"search_issues":       "issues",
	"add_reaction":        "issues",
	"create_issue_comment": "issues",

	// Pull Requests Toolset
	"pull_request_read":    "pull_requests",
	"list_pull_requests":   "pull_requests",
	"get_pull_request":     "pull_requests",
	"create_pull_request":  "pull_requests",
	"search_pull_requests": "pull_requests",

	// Actions Toolset
	"list_workflows":                  "actions",
	"list_workflow_runs":              "actions",
	"get_workflow_run":                "actions",
	"download_workflow_run_artifact":  "actions",
	"get_workflow_run_usage":          "actions",
	"list_workflow_jobs":              "actions",
	"get_job_logs":                    "actions",
	"list_workflow_run_artifacts":     "actions",

	// Code Security Toolset
	"list_code_scanning_alerts": "code_security",
	"get_code_scanning_alert":   "code_security",
	"create_code_scanning_alert": "code_security",

	// Dependabot Toolset
	// (No specific tools listed in documentation, but toolset exists)

	// Discussions Toolset
	"list_discussions":  "discussions",
	"create_discussion": "discussions",

	// Experiments Toolset
	// (No specific tools listed in documentation, but toolset exists)

	// Gists Toolset
	"create_gist": "gists",
	"list_gists":  "gists",

	// Labels Toolset
	"get_label":    "labels",
	"list_labels":  "labels",
	"create_label": "labels",

	// Notifications Toolset
	"list_notifications":       "notifications",
	"mark_notifications_read":  "notifications",

	// Organizations Toolset
	"get_organization":   "orgs",
	"list_organizations": "orgs",

	// Projects Toolset
	// (No specific tools listed in documentation, but toolset exists)

	// Secret Protection Toolset
	"list_secret_scanning_alerts": "secret_protection",
	"get_secret_scanning_alert":   "secret_protection",

	// Security Advisories Toolset
	// (No specific tools listed in documentation, but toolset exists)

	// Stargazers Toolset
	// (No specific tools listed in documentation, but toolset exists)

	// Users Toolset
	"get_user":   "users",
	"list_users": "users",

	// Search Toolset
	"search_repositories": "search",
	"search_users":        "search",
}

// ValidateGitHubToolsAgainstToolsets validates that all allowed GitHub tools have their
// corresponding toolsets enabled in the configuration
func ValidateGitHubToolsAgainstToolsets(allowedTools []string, enabledToolsets []string) error {
	if len(allowedTools) == 0 {
		// No specific tools restricted, validation not needed
		return nil
	}

	// Create a set of enabled toolsets for fast lookup
	enabledSet := make(map[string]bool)
	for _, toolset := range enabledToolsets {
		enabledSet[toolset] = true
	}

	// Track missing toolsets and which tools need them
	missingToolsets := make(map[string][]string) // toolset -> list of tools that need it

	for _, tool := range allowedTools {
		requiredToolset, exists := GitHubToolToToolsetMap[tool]
		if !exists {
			// Tool not in our mapping - this could be a new tool or a typo
			// We'll skip validation for unknown tools to avoid false positives
			continue
		}

		if !enabledSet[requiredToolset] {
			missingToolsets[requiredToolset] = append(missingToolsets[requiredToolset], tool)
		}
	}

	if len(missingToolsets) > 0 {
		return NewGitHubToolsetValidationError(missingToolsets)
	}

	return nil
}
