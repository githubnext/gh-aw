package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var createIssueLog = logger.New("workflow:create_issue")

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`
	Labels               []string `yaml:"labels,omitempty"`
	AllowedLabels        []string `yaml:"allowed-labels,omitempty"`       // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	AllowedFields        []string `yaml:"allowed-fields,omitempty"`       // Optional list of allowed issue field names. If omitted or empty, any issue fields are allowed. Use ["*"] to explicitly allow all.
	Assignees            []string `yaml:"assignees,omitempty"`            // List of users/bots to assign the issue to
	DeduplicateByTitle   any      `yaml:"deduplicate-by-title,omitempty"` // When true or 0, deduplicate by exact title match. When set to a positive integer N, also allow fuzzy matches up to edit distance N. When false or omitted, disable title-based deduplication.
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`          // Target repository in format "owner/repo" for cross-repository issues
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"`        // List of additional repositories that issues can be created in
	CloseOlderIssues     *string  `yaml:"close-older-issues,omitempty"`   // When true, close older issues with same title prefix or labels as "not planned"
	CloseOlderKey        string   `yaml:"close-older-key,omitempty"`      // Optional explicit deduplication key for close-older matching. When set, uses gh-aw-close-key marker instead of workflow-id markers.
	GroupByDay           *string  `yaml:"group-by-day,omitempty"`         // When true, if an open issue was already created today (UTC), post new content as a comment on it instead of creating a duplicate. Works best with close-older-issues: true.
	Expires              int      `yaml:"expires,omitempty"`              // Hours until the issue expires and should be automatically closed
	Group                *string  `yaml:"group,omitempty"`                // If true, group issues as sub-issues under a parent issue (workflow ID is used as group identifier)
	Footer               *string  `yaml:"footer,omitempty"`               // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
}

// parseCreateIssuesConfig handles create-issue configuration
func (c *Compiler) parseCreateIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	return parseCreateEntityConfig(
		outputMap,
		"create-issue",
		CreateParseOptions{
			BoolFields:    []string{"close-older-issues", "group", "footer", "group-by-day"},
			IntFields:     []string{"max"},
			HandleExpires: true,
		},
		createIssueLog,
		func(err error) *CreateIssuesConfig {
			createIssueLog.Printf("Failed to unmarshal config: %v", err)
			// For backward compatibility, handle nil/empty config
			return &CreateIssuesConfig{}
		},
		func(configData map[string]any) bool {
			coerceStringOrArrayFields(configData, []string{"assignees"}, createIssueLog)
			return true
		},
		func(_ map[string]any, config *CreateIssuesConfig, expiresDisabled bool) {
			// Set default max if not specified
			if config.Max == nil {
				config.Max = defaultIntStr(1)
			}

			// Log expires if configured or explicitly disabled
			if expiresDisabled {
				createIssueLog.Print("Issue expiration explicitly disabled")
			} else if config.Expires > 0 {
				createIssueLog.Printf("Issue expiration configured: %d hours", config.Expires)
			}
		},
	)
}

// hasCopilotAssignee checks if "copilot" is in the assignees list
func hasCopilotAssignee(assignees []string) bool {
	return slices.Contains(assignees, "copilot")
}
