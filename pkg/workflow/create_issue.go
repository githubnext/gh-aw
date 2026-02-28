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
	AllowedLabels        []string `yaml:"allowed-labels,omitempty"`     // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Assignees            []string `yaml:"assignees,omitempty"`          // List of users/bots to assign the issue to
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`        // Target repository in format "owner/repo" for cross-repository issues
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"`      // List of additional repositories that issues can be created in
	CloseOlderIssues     *string  `yaml:"close-older-issues,omitempty"` // When true, close older issues with same title prefix or labels as "not planned"
	Expires              int      `yaml:"expires,omitempty"`            // Hours until the issue expires and should be automatically closed
	Group                *string  `yaml:"group,omitempty"`              // If true, group issues as sub-issues under a parent issue (workflow ID is used as group identifier)
	Footer               *string  `yaml:"footer,omitempty"`             // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	// Check if the key exists
	if _, exists := outputMap["create-issue"]; !exists {
		return nil
	}

	createIssueLog.Print("Parsing create-issue configuration")

	// Get the config data to check for special cases before unmarshaling
	configData, _ := outputMap["create-issue"].(map[string]any)

	// Pre-process the expires field (convert to hours before unmarshaling)
	expiresDisabled := preprocessExpiresField(configData, createIssueLog)

	// Pre-process templatable bool fields: convert literal booleans to strings so that
	// GitHub Actions expression strings (e.g. "${{ inputs.close-older-issues }}") are also accepted.
	for _, field := range []string{"close-older-issues", "group", "footer"} {
		if err := preprocessBoolFieldAsString(configData, field, createIssueLog); err != nil {
			createIssueLog.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", createIssueLog); err != nil {
		createIssueLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Unmarshal into typed config struct
	var config CreateIssuesConfig
	if err := unmarshalConfig(outputMap, "create-issue", &config, createIssueLog); err != nil {
		createIssueLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		config = CreateIssuesConfig{}
	}

	// Handle single string assignee (YAML unmarshaling won't convert string to []string)
	if len(config.Assignees) == 0 && configData != nil {
		if assignees, exists := configData["assignees"]; exists {
			if assigneeStr, ok := assignees.(string); ok {
				config.Assignees = []string{assigneeStr}
				createIssueLog.Printf("Converted single assignee string to array: %v", config.Assignees)
			}
		}
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}

	// Validate target-repo (wildcard "*" is not allowed)
	if validateTargetRepoSlug(config.TargetRepoSlug, createIssueLog) {
		return nil // Invalid configuration, return nil to cause validation error
	}

	// Log expires if configured or explicitly disabled
	if expiresDisabled {
		createIssueLog.Print("Issue expiration explicitly disabled")
	} else if config.Expires > 0 {
		createIssueLog.Printf("Issue expiration configured: %d hours", config.Expires)
	}

	return &config
}

// hasCopilotAssignee checks if "copilot" is in the assignees list
func hasCopilotAssignee(assignees []string) bool {
	return slices.Contains(assignees, "copilot")
}
