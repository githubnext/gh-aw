package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var createPRLog = logger.New("workflow:create_pull_request")

// getFallbackAsIssue returns the effective fallback-as-issue setting (defaults to true).
func getFallbackAsIssue(config *CreatePullRequestsConfig) bool {
	if config == nil || config.FallbackAsIssue == nil {
		return true // Default
	}
	return *config.FallbackAsIssue
}

// CreatePullRequestsConfig holds configuration for creating GitHub pull requests from agent output
type CreatePullRequestsConfig struct {
	BaseSafeOutputConfig           `yaml:",inline"`
	TitlePrefix                    string   `yaml:"title-prefix,omitempty"`
	Labels                         []string `yaml:"labels,omitempty"`
	AllowedLabels                  []string `yaml:"allowed-labels,omitempty"`                      // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Reviewers                      []string `yaml:"reviewers,omitempty"`                           // List of users/bots to assign as reviewers to the pull request
	Draft                          *string  `yaml:"draft,omitempty"`                               // Pointer to distinguish between unset (nil), literal bool, and expression values
	IfNoChanges                    string   `yaml:"if-no-changes,omitempty"`                       // Behavior when no changes to push: "warn" (default), "error", or "ignore"
	AllowEmpty                     *string  `yaml:"allow-empty,omitempty"`                         // Allow creating PR without patch file or with empty patch (useful for preparing feature branches)
	TargetRepoSlug                 string   `yaml:"target-repo,omitempty"`                         // Target repository in format "owner/repo" for cross-repository pull requests
	AllowedRepos                   []string `yaml:"allowed-repos,omitempty"`                       // List of additional repositories that pull requests can be created in (additionally to the target-repo)
	Expires                        int      `yaml:"expires,omitempty"`                             // Hours until the pull request expires and should be automatically closed (only for same-repo PRs)
	AutoMerge                      *string  `yaml:"auto-merge,omitempty"`                          // Enable auto-merge for the pull request when all required checks pass
	BaseBranch                     string   `yaml:"base-branch,omitempty"`                         // Base branch for the pull request (defaults to github.ref_name if not specified)
	Footer                         *string  `yaml:"footer,omitempty"`                              // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
	FallbackAsIssue                *bool    `yaml:"fallback-as-issue,omitempty"`                   // When true (default), creates an issue if PR creation fails. When false, no fallback occurs and issues: write permission is not requested.
	GithubTokenForExtraEmptyCommit string   `yaml:"github-token-for-extra-empty-commit,omitempty"` // Token used to push an empty commit to trigger CI events. Use a PAT or "app" for GitHub App auth.
}

// parsePullRequestsConfig handles only create-pull-request (singular) configuration
func (c *Compiler) parsePullRequestsConfig(outputMap map[string]any) *CreatePullRequestsConfig {
	// Check for singular form only
	if _, exists := outputMap["create-pull-request"]; !exists {
		createPRLog.Print("No create-pull-request configuration found")
		return nil
	}

	createPRLog.Print("Parsing create-pull-request configuration")

	// Get the config data to check for special cases before unmarshaling
	configData, _ := outputMap["create-pull-request"].(map[string]any)

	// Pre-process the reviewers field to convert single string to array BEFORE unmarshaling
	// This prevents YAML unmarshal errors when reviewers is a string instead of []string
	if configData != nil {
		if reviewers, exists := configData["reviewers"]; exists {
			if reviewerStr, ok := reviewers.(string); ok {
				// Convert single string to array
				configData["reviewers"] = []string{reviewerStr}
				createPRLog.Printf("Converted single reviewer string to array before unmarshaling")
			}
		}
	}

	// Pre-process the expires field if it's a string (convert to int before unmarshaling)
	if configData != nil {
		if expires, exists := configData["expires"]; exists {
			if _, ok := expires.(string); ok {
				// Parse the string format and replace with int
				expiresInt := parseExpiresFromConfig(configData)
				if expiresInt > 0 {
					configData["expires"] = expiresInt
					createPRLog.Printf("Converted expires from relative time format to hours: %d", expiresInt)
				}
			}
		}
	}

	// Pre-process templatable bool fields: convert literal booleans to strings so that
	// GitHub Actions expression strings (e.g. "${{ inputs.draft-prs }}") are also accepted.
	for _, field := range []string{"draft", "allow-empty", "auto-merge", "footer"} {
		if err := preprocessBoolFieldAsString(configData, field, createPRLog); err != nil {
			createPRLog.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", createPRLog); err != nil {
		createPRLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Unmarshal into typed config struct
	var config CreatePullRequestsConfig
	if err := unmarshalConfig(outputMap, "create-pull-request", &config, createPRLog); err != nil {
		createPRLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		config = CreatePullRequestsConfig{}
	}

	// Validate target-repo (wildcard "*" is not allowed)
	if validateTargetRepoSlug(config.TargetRepoSlug, createPRLog) {
		return nil // Invalid configuration, return nil to cause validation error
	}

	// Log expires if configured
	if config.Expires > 0 {
		createPRLog.Printf("Pull request expiration configured: %d hours", config.Expires)
	}

	// Set default max if not explicitly configured (default is 1)
	if config.Max == nil {
		config.Max = defaultIntStr(1)
		createPRLog.Print("Using default max count: 1")
	} else {
		createPRLog.Printf("Pull request max count configured: %s", *config.Max)
	}

	return &config
}
