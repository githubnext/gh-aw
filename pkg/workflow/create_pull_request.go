package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var createPRLog = logger.New("workflow:create_pull_request")

var createPRStringOrArrayFields = []string{"reviewers", "team-reviewers", "assignees"}
var createPRExpressionArrayFields = []string{"labels", "allowed-repos", "allowed-base-branches", "allowed-branches"}

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
	BranchPrefix                   string   `yaml:"branch-prefix,omitempty"` // Optional prefix for the pull request branch name (e.g. "signed/"). Applied before the agent-specified or auto-generated branch name.
	TitlePrefix                    string   `yaml:"title-prefix,omitempty"`
	Labels                         []string `yaml:"labels,omitempty"`
	AllowedLabels                  []string `yaml:"allowed-labels,omitempty"`                      // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	Reviewers                      []string `yaml:"reviewers,omitempty"`                           // List of users/bots to assign as reviewers to the pull request
	TeamReviewers                  []string `yaml:"team-reviewers,omitempty"`                      // List of team slugs to assign as team reviewers to the pull request
	Assignees                      []string `yaml:"assignees,omitempty"`                           // List of users to assign to any fallback issue created by create-pull-request
	FallbackLabels                 []string `yaml:"fallback-labels,omitempty"`                     // List of labels to apply to fallback issues created when PR creation cannot proceed. If omitted, fallback issues reuse PR labels.
	Draft                          *string  `yaml:"draft,omitempty"`                               // Pointer to distinguish between unset (nil), literal bool, and expression values
	IfNoChanges                    string   `yaml:"if-no-changes,omitempty"`                       // Behavior when no changes to push: "warn" (default), "error", or "ignore"
	AllowEmpty                     *string  `yaml:"allow-empty,omitempty"`                         // Allow creating PR without patch file or with empty patch (useful for preparing feature branches)
	TargetRepoSlug                 string   `yaml:"target-repo,omitempty"`                         // Target repository in format "owner/repo" for cross-repository pull requests
	AllowedRepos                   []string `yaml:"allowed-repos,omitempty"`                       // List of additional repositories that pull requests can be created in (additionally to the target-repo)
	AllowedBaseBranches            []string `yaml:"allowed-base-branches,omitempty"`               // List of allowed base branch globs (e.g. "release/*"). Enables agent-provided `base` override when configured.
	AllowedBranches                []string `yaml:"allowed-branches,omitempty"`                    // List of allowed source branch globs (e.g. "feature/*"). Branch in create_pull_request payload must match when configured.
	MaxPatchSize                   int      `yaml:"max-patch-size,omitempty"`                      // Maximum allowed patch size in KB for create-pull-request only. Overrides safe-outputs.max-patch-size when set.
	MaxPatchFiles                  int      `yaml:"max-patch-files,omitempty"`                     // Maximum allowed unique files in create-pull-request patch only. Overrides safe-outputs.max-patch-files when set.
	Expires                        int      `yaml:"expires,omitempty"`                             // Hours until the pull request expires and should be automatically closed (only for same-repo PRs)
	AutoMerge                      *string  `yaml:"auto-merge,omitempty"`                          // Enable auto-merge for the pull request when all required checks pass
	BaseBranch                     string   `yaml:"base-branch,omitempty"`                         // Base branch for the pull request (defaults to github.ref_name if not specified)
	Footer                         *string  `yaml:"footer,omitempty"`                              // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
	FallbackAsIssue                *bool    `yaml:"fallback-as-issue,omitempty"`                   // When true (default), creates an issue if PR creation fails. When false, no fallback occurs and issues: write permission is not requested.
	AutoCloseIssue                 *string  `yaml:"auto-close-issue,omitempty"`                    // Auto-add "Fixes #N" closing keyword when triggered from an issue (default: true). Set to false to prevent auto-closing the triggering issue on PR merge. Accepts a boolean or a GitHub Actions expression.
	GithubTokenForExtraEmptyCommit string   `yaml:"github-token-for-extra-empty-commit,omitempty"` // Token used to push an empty commit to trigger CI events. Use a PAT or "app" for GitHub App auth.
	ManifestFilesPolicy            *string  `yaml:"protected-files,omitempty"`                     // Controls protected-file protection: "request_review" (default) creates a PR and submits a REQUEST_CHANGES review, "blocked" hard-blocks, "allowed" permits all changes, and "fallback-to-issue" creates a review issue instead of a PR.
	ProtectedFilesExclude          []string `yaml:"-"`                                             // Files/prefixes to exclude from the default protected list (from object-form protected-files.exclude). Not sourced from YAML directly; populated during pre-processing.
	AllowedFiles                   []string `yaml:"allowed-files,omitempty"`                       // Strict allowlist of glob patterns for files eligible for create. Checked independently of protected-files; both checks must pass.
	ExcludedFiles                  []string `yaml:"excluded-files,omitempty"`                      // List of glob patterns for files to exclude from the patch using git :(exclude) pathspecs. Matching files are stripped by git at generation time and will not appear in the commit or be subject to allowed-files or protected-files checks.
	PreserveBranchName             bool     `yaml:"preserve-branch-name,omitempty"`                // When true, skips the random salt suffix on agent-specified branch names. Invalid characters are still replaced for security; casing is always preserved. Useful when CI enforces branch naming conventions (e.g. Jira keys in uppercase).
	RecreateRef                    bool     `yaml:"recreate-ref,omitempty"`                        // When true (and preserve-branch-name is true), allows the handler to force-delete an existing remote branch ref and recreate it from the agent's local HEAD. When false (default), an existing remote branch causes a fallback to issue (or push_failed). Useful for long-lived reusable branches whose previous PR was merged.
	PatchFormat                    string   `yaml:"patch-format,omitempty"`                        // Transport format for packaging changes: "bundle" (default, uses git bundle and preserves merge topology/per-commit metadata) or "am" (uses git format-patch).
	SignedCommits                  *bool    `yaml:"signed-commits,omitempty"`                      // When false, skips GitHub GraphQL signed commits and pushes the local git history directly. Default is true.
	AllowWorkflows                 bool     `yaml:"allow-workflows,omitempty"`                     // When true, adds workflows: write to the GitHub App token. Requires safe-outputs.github-app to be configured.
}

// parseCreatePullRequestsConfig handles only create-pull-request (singular) configuration
func (c *Compiler) parseCreatePullRequestsConfig(outputMap map[string]any) *CreatePullRequestsConfig {
	// Check for singular form only
	if _, exists := outputMap["create-pull-request"]; !exists {
		createPRLog.Print("No create-pull-request configuration found")
		return nil
	}

	var protectedFilesExclude []string
	config := parseCreateEntityConfig(
		outputMap,
		"create-pull-request",
		CreateParseOptions{
			BoolFields:    []string{"draft", "allow-empty", "auto-merge", "footer", "auto-close-issue"},
			IntFields:     []string{"max"},
			HandleExpires: true,
		},
		createPRLog,
		func(err error) *CreatePullRequestsConfig {
			createPRLog.Printf("Failed to unmarshal config: %v", err)
			// For backward compatibility, handle nil/empty config
			return &CreatePullRequestsConfig{}
		},
		func(configData map[string]any) bool {
			coerceStringOrArrayFields(configData, createPRStringOrArrayFields, createPRLog)

			// Pre-process protected-files: supports string enum OR object form {policy, exclude}.
			// Object form is preprocessed to extract the policy (stored back as string) and
			// the exclude list (stored in a local variable and assigned to the config after unmarshaling).
			protectedFilesExclude = preprocessProtectedFilesField(configData, createPRLog)

			// Validate protected-files string enum after object-form preprocessing.
			validateStringEnumField(configData, "protected-files", []string{"blocked", "allowed", "fallback-to-issue", "request_review"}, createPRLog)

			// Pre-process patch-format: valid values are "bundle" (default) and "am".
			validateStringEnumField(configData, "patch-format", []string{"am", "bundle"}, createPRLog)

			// Pre-process list fields that also accept a GitHub Actions expression string.
			// An expression is wrapped in a single-element []string so the []string struct field
			// can receive it after YAML unmarshaling; the handler config builder later re-emits it
			// as a JSON string for runtime evaluation.
			for _, field := range createPRExpressionArrayFields {
				if err := preprocessStringArrayFieldAsTemplatable(configData, field, createPRLog); err != nil {
					createPRLog.Printf("Invalid %s value: %v", field, err)
					return false
				}
			}

			return true
		},
		func(_ map[string]any, config *CreatePullRequestsConfig, expiresDisabled bool) {
			if expiresDisabled {
				createPRLog.Print("Pull request expiration disabled")
			}

			// Log expires if configured
			if config.Expires > 0 {
				createPRLog.Printf("Pull request expiration configured: %d hours", config.Expires)
			}

			// Apply the exclude list extracted from the object-form protected-files field.
			config.ProtectedFilesExclude = protectedFilesExclude

			// Set default max if not explicitly configured (default is 1)
			if config.Max == nil {
				config.Max = defaultIntStr(1)
				createPRLog.Print("Using default max count: 1")
			} else {
				createPRLog.Printf("Pull request max count configured: %s", *config.Max)
			}
		},
	)
	if config == nil {
		return nil
	}

	return config
}
