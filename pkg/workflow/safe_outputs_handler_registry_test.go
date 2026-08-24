package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerRegistryCategoriesAssembleCompleteRegistryExactlyOnce(t *testing.T) {
	expectedKeys := map[string]struct{}{
		"add_comment":                           {},
		"add_labels":                            {},
		"add_reviewer":                          {},
		"approve_workflow_run":                  {},
		"assign_milestone":                      {},
		"assign_to_agent":                       {},
		"assign_to_user":                        {},
		"autofix_code_scanning_alert":           {},
		"call_workflow":                         {},
		"close_discussion":                      {},
		"close_issue":                           {},
		"close_pull_request":                    {},
		"create_agent_session":                  {},
		"create_check_run":                      {},
		"create_code_scanning_alert":            {},
		"create_discussion":                     {},
		"create_issue":                          {},
		"create_project":                        {},
		"create_project_status_update":          {},
		"create_pull_request":                   {},
		"create_pull_request_review_comment":    {},
		"create_report_incomplete_issue":        {},
		"dismiss_pull_request_review":           {},
		"dispatch_repository":                   {},
		"dispatch_workflow":                     {},
		"hide_comment":                          {},
		"link_sub_issue":                        {},
		"mark_pull_request_as_ready_for_review": {},
		"merge_pull_request":                    {},
		"missing_data":                          {},
		"missing_tool":                          {},
		"noop":                                  {},
		"push_to_pull_request_branch":           {},
		"remove_labels":                         {},
		"replace_label":                         {},
		"reply_to_pull_request_review_comment":  {},
		"report_incomplete":                     {},
		"resolve_pull_request_review_thread":    {},
		"set_issue_field":                       {},
		"set_issue_type":                        {},
		"submit_pull_request_review":            {},
		"unassign_from_user":                    {},
		"update_discussion":                     {},
		"update_issue":                          {},
		"update_project":                        {},
		"update_pull_request":                   {},
		"update_release":                        {},
		"upload_artifact":                       {},
		"upload_asset":                          {},
		"upload_code_coverage":                  {},
	}

	categoryCounts := make(map[string]int)
	for _, buildCategory := range handlerRegistryCategoryBuilders {
		for key := range buildCategory() {
			categoryCounts[key]++
		}
	}

	require.Len(t, handlerRegistry, len(expectedKeys))
	require.Len(t, categoryCounts, len(expectedKeys))
	for key := range expectedKeys {
		assert.Equal(t, 1, categoryCounts[key], "handler %q must occur in exactly one category", key)
		assert.Contains(t, handlerRegistry, key)
	}
	for key := range handlerRegistry {
		assert.Contains(t, expectedKeys, key)
	}
}
