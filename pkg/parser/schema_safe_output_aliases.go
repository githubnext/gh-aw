package parser

import (
	"fmt"
	"strings"
)

// safeOutputsSchemaPath is the JSON schema path for the safe-outputs section.
const safeOutputsSchemaPath = "/safe-outputs"

// safeOutputAliases maps common agent mistakes and MCP tool name variations to their
// correct safe-output field names. Agents frequently use GitHub MCP tool names
// (e.g. "create_issue_comment") or underscore variants instead of the hyphenated
// safe-output fields (e.g. "add-comment"). These aliases enable the compiler to
// produce a precise "Did you mean 'X'?" suggestion instead of a generic field list.
var safeOutputAliases = map[string]string{
	// add-comment aliases (most common agent mistake: MCP tool name vs safe-output)
	"create-issue-comment": "add-comment",
	"create_issue_comment": "add-comment",
	"add_comment":          "add-comment",
	"add-issue-comment":    "add-comment",
	"add_issue_comment":    "add-comment",
	"post-comment":         "add-comment",
	"post_comment":         "add-comment",
	"create-comment":       "add-comment",
	"create_comment":       "add-comment",

	// underscore → hyphen for common operation fields
	"add_labels":                  "add-labels",
	"remove_labels":               "remove-labels",
	"replace_label":               "replace-label",
	"create_issue":                "create-issue",
	"close_issue":                 "close-issue",
	"update_issue":                "update-issue",
	"create_discussion":           "create-discussion",
	"close_discussion":            "close-discussion",
	"update_discussion":           "update-discussion",
	"create_pull_request":         "create-pull-request",
	"close_pull_request":          "close-pull-request",
	"update_pull_request":         "update-pull-request",
	"merge_pull_request":          "merge-pull-request",
	"assign_to_user":              "assign-to-user",
	"unassign_from_user":          "unassign-from-user",
	"assign_to_agent":             "assign-to-agent",
	"assign_milestone":            "assign-milestone",
	"hide_comment":                "hide-comment",
	"set_issue_type":              "set-issue-type",
	"set_issue_field":             "set-issue-field",
	"add_reviewer":                "add-reviewer",
	"link_sub_issue":              "link-sub-issue",
	"dispatch_workflow":           "dispatch-workflow",
	"update_release":              "update-release",
	"create_check_run":            "create-check-run",
	"upload_artifact":             "upload-artifact",
	"upload_asset":                "upload-asset",
	"update_project":              "update-project",
	"create_project":              "create-project",
	"report_failure_as_issue":     "report-failure-as-issue",
	"create_agent_session":        "create-agent-session",
	"create_agent_task":           "create-agent-task",
	"autofix_code_scanning_alert": "autofix-code-scanning-alert",
	"create_code_scanning_alert":  "create-code-scanning-alert",

	// longer pull-request operation fields
	"push_to_pull_request_branch":           "push-to-pull-request-branch",
	"submit_pull_request_review":            "submit-pull-request-review",
	"dismiss_pull_request_review":           "dismiss-pull-request-review",
	"create_pull_request_review_comment":    "create-pull-request-review-comment",
	"reply_to_pull_request_review_comment":  "reply-to-pull-request-review-comment",
	"resolve_pull_request_review_thread":    "resolve-pull-request-review-thread",
	"mark_pull_request_as_ready_for_review": "mark-pull-request-as-ready-for-review",
}

// safeOutputAliasSuggestion returns a "Did you mean 'X'?" suggestion when an unknown
// property under /safe-outputs matches a known alias for the correct field name.
// It returns an empty string when the error is not under safe-outputs, is not an
// additional-properties error, or when none of the invalid props match a known alias.
func safeOutputAliasSuggestion(errorMessage, jsonPath string) string {
	if jsonPath != safeOutputsSchemaPath {
		return ""
	}

	lowerError := strings.ToLower(errorMessage)
	if !strings.Contains(lowerError, "additional propert") || !strings.Contains(lowerError, "not allowed") {
		return ""
	}

	invalidProps := extractAdditionalPropertyNames(errorMessage)
	if len(invalidProps) == 0 {
		return ""
	}

	var suggestions []string
	seen := make(map[string]struct{})
	for _, prop := range invalidProps {
		canonical, ok := safeOutputAliases[prop]
		if !ok {
			continue
		}
		if _, already := seen[canonical]; already {
			continue
		}
		seen[canonical] = struct{}{}
		suggestions = append(suggestions, fmt.Sprintf("'%s'", canonical))
	}

	if len(suggestions) == 0 {
		return ""
	}

	if len(suggestions) == 1 {
		return fmt.Sprintf("Did you mean %s?", suggestions[0])
	}
	return fmt.Sprintf("Did you mean: %s?", strings.Join(suggestions, ", "))
}
