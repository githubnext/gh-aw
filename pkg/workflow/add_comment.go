package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var addCommentLog = logger.New("workflow:add_comment")

// AddCommentConfig is a deprecated alias of AddCommentsConfig.
type AddCommentConfig = AddCommentsConfig

// AddCommentsConfig holds configuration for creating GitHub issue/PR comments from agent output
type AddCommentsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
	Target                 string   `yaml:"target,omitempty"`                    // Target for comments: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug         string   `yaml:"target-repo,omitempty"`               // Target repository in format "owner/repo" for cross-repository comments
	AllowedRepos           []string `yaml:"allowed-repos,omitempty"`             // List of additional repositories that comments can be added to (additionally to the target-repo)
	HideOlderComments      *string  `yaml:"hide-older-comments,omitempty"`       // When true, minimizes/hides all previous comments from the same workflow before creating the new comment
	HideOlderCommentsMatch []string `yaml:"hide-older-comments-match,omitempty"` // Internal list populated from hide-older-comments.match and passed to the JS handler as exact workflow ID matches
	AllowedReasons         []string `yaml:"allowed-reasons,omitempty"`           // List of allowed reasons for hiding older comments (default: all reasons allowed)
	Issues                 *bool    `yaml:"issues,omitempty"`                    // When false, excludes issues:write permission and issues from event condition. Default (nil or true) includes issues:write.
	PullRequests           *bool    `yaml:"pull-requests,omitempty"`             // When false, excludes pull-requests:write permission and PRs from event condition. Default (nil or true) includes pull-requests:write.
	Discussions            *bool    `yaml:"discussions,omitempty"`               // When true, includes discussions:write permission. Default (nil or false) excludes discussions:write.
	Footer                 *string  `yaml:"footer,omitempty"`                    // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
}

// parseCommentsConfig handles add-comment configuration
func (c *Compiler) parseCommentsConfig(outputMap map[string]any) *AddCommentsConfig {
	// Check if the key exists
	if _, exists := outputMap["add-comment"]; !exists {
		return nil
	}

	// Get config data for pre-processing before YAML unmarshaling
	configData, _ := outputMap["add-comment"].(map[string]any)

	if err := preprocessHideOlderCommentsConfig(configData, addCommentLog); err != nil {
		addCommentLog.Printf("Invalid hide-older-comments configuration: %v", err)
		return nil
	}

	// Pre-process templatable bool fields
	if err := preprocessBoolFieldAsString(configData, "hide-older-comments", addCommentLog); err != nil {
		addCommentLog.Printf("Invalid hide-older-comments value: %v", err)
		return nil
	}
	if err := preprocessBoolFieldAsString(configData, "footer", addCommentLog); err != nil {
		addCommentLog.Printf("Invalid footer value: %v", err)
		return nil
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", addCommentLog); err != nil {
		addCommentLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Pre-process list fields that also accept a GitHub Actions expression string.
	if err := preprocessStringArrayFieldAsTemplatable(configData, "allowed-repos", addCommentLog); err != nil {
		addCommentLog.Printf("Invalid allowed-repos value: %v", err)
		return nil
	}

	config := parseConfigScaffold(outputMap, "add-comment", addCommentLog, func(err error) *AddCommentsConfig {
		addCommentLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		return &AddCommentsConfig{}
	})
	if config == nil {
		return nil
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}

	return config
}

func preprocessHideOlderCommentsConfig(configData map[string]any, debugLog *logger.Logger) error {
	if configData == nil {
		return nil
	}

	raw, exists := configData["hide-older-comments"]
	if !exists || raw == nil {
		return nil
	}

	objectConfig, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	if enabledRaw, hasEnabled := objectConfig["enabled"]; hasEnabled {
		switch enabled := enabledRaw.(type) {
		case bool:
			configData["hide-older-comments"] = enabled
		case string:
			if !isExpression(enabled) {
				return fmt.Errorf("field %q must be a boolean or a GitHub Actions expression", "hide-older-comments.enabled")
			}
			configData["hide-older-comments"] = enabled
		default:
			return fmt.Errorf("field %q must be a boolean or a GitHub Actions expression", "hide-older-comments.enabled")
		}
	} else {
		configData["hide-older-comments"] = true
	}

	if matchRaw, hasMatch := objectConfig["match"]; hasMatch {
		configData["hide-older-comments-match"] = parseStringSliceAny(matchRaw, debugLog)
	}

	return nil
}

// buildAddCommentPermissions computes the permissions for the add_comment job based on config.
// Issues: nil or true → issues:write (default: true)
// PullRequests: nil or true → pull-requests:write (default: true)
// Discussions: true → discussions:write (default: false)
func buildAddCommentPermissions(config *AddCommentsConfig) *Permissions {
	permMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	}
	if config == nil || config.Issues == nil || *config.Issues {
		permMap[PermissionIssues] = PermissionWrite
	}
	if config == nil || config.PullRequests == nil || *config.PullRequests {
		permMap[PermissionPullRequests] = PermissionWrite
	}
	if config != nil && config.Discussions != nil && *config.Discussions {
		permMap[PermissionDiscussions] = PermissionWrite
	}
	return NewPermissionsFromMap(permMap)
}
