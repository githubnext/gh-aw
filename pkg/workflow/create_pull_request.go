package workflow

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
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

// buildCreateOutputPullRequestJob creates the create_pull_request job
func (c *Compiler) buildCreateOutputPullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequests == nil {
		return nil, errors.New("safe-outputs.create-pull-request configuration is required")
	}

	if createPRLog.Enabled() {
		draftValue := "true" // Default
		if data.SafeOutputs.CreatePullRequests.Draft != nil {
			draftValue = *data.SafeOutputs.CreatePullRequests.Draft
		}
		fallbackAsIssue := getFallbackAsIssue(data.SafeOutputs.CreatePullRequests)
		createPRLog.Printf("Building create-pull-request job: workflow=%s, main_job=%s, draft=%v, reviewers=%d, fallback_as_issue=%v",
			data.Name, mainJobName, draftValue, len(data.SafeOutputs.CreatePullRequests.Reviewers), fallbackAsIssue)
	}

	// Build pre-steps for patch download, checkout, and git config
	var preSteps []string

	// Step 1: Download patch artifact from unified agent-artifacts
	preSteps = append(preSteps, "      - name: Download patch artifact\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, "          name: agent-artifacts\n")
	preSteps = append(preSteps, "          path: /tmp/gh-aw/\n")

	// Step 2: Checkout repository
	// Step 3: Configure Git credentials
	// Pass the target repo to configure git remote correctly for cross-repo operations
	// Use token precedence chain instead of hardcoded github.token
	// Precedence: create-pull-request config token > safe-outputs token > GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
	var configToken string
	if data.SafeOutputs.CreatePullRequests != nil {
		configToken = data.SafeOutputs.CreatePullRequests.GitHubToken
	}
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}
	// Get effective token (handles fallback to GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)
	gitToken := getEffectiveSafeOutputGitHubToken(effectiveCustomToken)

	// Use the resolved token for checkout
	preSteps = buildCheckoutRepository(preSteps, c, data.SafeOutputs.CreatePullRequests.TargetRepoSlug, gitToken)

	preSteps = append(preSteps, c.generateGitConfigurationStepsWithToken(gitToken, data.SafeOutputs.CreatePullRequests.TargetRepoSlug)...)

	// Build custom environment variables specific to create-pull-request
	var customEnvVars []string
	// Pass the workflow ID for branch naming
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", mainJobName))
	// Pass custom base branch only if explicitly configured; JS will resolve dynamically otherwise
	if data.SafeOutputs.CreatePullRequests.BaseBranch != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CUSTOM_BASE_BRANCH: %q\n", data.SafeOutputs.CreatePullRequests.BaseBranch))
	}
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", data.SafeOutputs.CreatePullRequests.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_LABELS", data.SafeOutputs.CreatePullRequests.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_ALLOWED_LABELS", data.SafeOutputs.CreatePullRequests.AllowedLabels)...)
	// Pass draft setting - default to true for backwards compatibility
	if data.SafeOutputs.CreatePullRequests.Draft != nil {
		customEnvVars = append(customEnvVars, buildTemplatableBoolEnvVar("GH_AW_PR_DRAFT", data.SafeOutputs.CreatePullRequests.Draft)...)
	} else {
		customEnvVars = append(customEnvVars, "          GH_AW_PR_DRAFT: \"true\"\n")
	}

	// Pass the if-no-changes configuration
	ifNoChanges := data.SafeOutputs.CreatePullRequests.IfNoChanges
	if ifNoChanges == "" {
		ifNoChanges = "warn" // Default value
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))

	// Pass the allow-empty configuration
	if data.SafeOutputs.CreatePullRequests.AllowEmpty != nil {
		customEnvVars = append(customEnvVars, buildTemplatableBoolEnvVar("GH_AW_PR_ALLOW_EMPTY", data.SafeOutputs.CreatePullRequests.AllowEmpty)...)
	} else {
		customEnvVars = append(customEnvVars, "          GH_AW_PR_ALLOW_EMPTY: \"false\"\n")
	}

	// Pass the auto-merge configuration
	if data.SafeOutputs.CreatePullRequests.AutoMerge != nil {
		customEnvVars = append(customEnvVars, buildTemplatableBoolEnvVar("GH_AW_PR_AUTO_MERGE", data.SafeOutputs.CreatePullRequests.AutoMerge)...)
	} else {
		customEnvVars = append(customEnvVars, "          GH_AW_PR_AUTO_MERGE: \"false\"\n")
	}

	// Pass the fallback-as-issue configuration - default to true for backwards compatibility
	if data.SafeOutputs.CreatePullRequests.FallbackAsIssue != nil {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_FALLBACK_AS_ISSUE: \"%t\"\n", *data.SafeOutputs.CreatePullRequests.FallbackAsIssue))
	} else {
		customEnvVars = append(customEnvVars, "          GH_AW_PR_FALLBACK_AS_ISSUE: \"true\"\n")
	}

	// Pass the maximum patch size configuration
	maxPatchSize := 1024 // Default value
	if data.SafeOutputs != nil && data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))

	// Pass activation comment information if available (for updating the comment with PR link)
	// These outputs are only available when reaction is configured in the workflow
	if data.AIReaction != "" && data.AIReaction != "none" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	}

	// Add expires value if set (only for same-repo PRs - when target-repo is not set)
	if data.SafeOutputs.CreatePullRequests.Expires > 0 && data.SafeOutputs.CreatePullRequests.TargetRepoSlug == "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_EXPIRES: \"%d\"\n", data.SafeOutputs.CreatePullRequests.Expires))
	}

	// Add footer flag if explicitly set to false
	if data.SafeOutputs.CreatePullRequests.Footer != nil && *data.SafeOutputs.CreatePullRequests.Footer == "false" {
		customEnvVars = append(customEnvVars, "          GH_AW_FOOTER: \"false\"\n")
		createPRLog.Print("Footer disabled - XML markers will be included but visible footer content will be omitted")
	}

	// Add extra empty commit token (for pushing an empty commit to trigger CI)
	// Defaults to GH_AW_CI_TRIGGER_TOKEN when not explicitly configured
	ciTriggerToken := data.SafeOutputs.CreatePullRequests.GithubTokenForExtraEmptyCommit
	switch ciTriggerToken {
	case "app":
		customEnvVars = append(customEnvVars, "          GH_AW_CI_TRIGGER_TOKEN: ${{ steps.safe-outputs-app-token.outputs.token || '' }}\n")
		createPRLog.Print("Extra empty commit using GitHub App token")
	case "default", "":
		// Use the magic GH_AW_CI_TRIGGER_TOKEN secret (default behavior when not explicitly configured)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CI_TRIGGER_TOKEN: %s\n", getEffectiveCITriggerGitHubToken("")))
		createPRLog.Print("Extra empty commit using GH_AW_CI_TRIGGER_TOKEN")
	default:
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CI_TRIGGER_TOKEN: %s\n", ciTriggerToken))
		createPRLog.Printf("Extra empty commit using explicit token")
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreatePullRequests.TargetRepoSlug)...)

	// Build post-steps for reviewers if configured
	var postSteps []string
	if len(data.SafeOutputs.CreatePullRequests.Reviewers) > 0 {
		// Get the effective GitHub token to use for gh CLI
		var safeOutputsToken string
		if data.SafeOutputs != nil {
			safeOutputsToken = data.SafeOutputs.GitHubToken
		}

		postSteps = buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       data.SafeOutputs.CreatePullRequests.Reviewers,
			ParticipantType:    "reviewer",
			CustomToken:        data.SafeOutputs.CreatePullRequests.GitHubToken,
			SafeOutputsToken:   safeOutputsToken,
			ConditionStepID:    "create_pull_request",
			ConditionOutputKey: "pull_request_url",
		})
	}

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
		"issue_number":        "${{ steps.create_pull_request.outputs.issue_number }}",
		"issue_url":           "${{ steps.create_pull_request.outputs.issue_url }}",
		"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
		"fallback_used":       "${{ steps.create_pull_request.outputs.fallback_used }}",
		"error_message":       "${{ steps.create_pull_request.outputs.error_message }}",
	}

	// Choose permissions based on fallback-as-issue setting
	fallbackAsIssue := getFallbackAsIssue(data.SafeOutputs.CreatePullRequests)
	var permissions *Permissions
	if fallbackAsIssue {
		// Default: include issues: write for fallback behavior
		permissions = NewPermissionsContentsWriteIssuesWritePRWrite()
		createPRLog.Print("Using permissions with issues:write (fallback-as-issue enabled)")
	} else {
		// Fallback disabled: only need contents: write and pull-requests: write
		permissions = NewPermissionsContentsWritePRWrite()
		createPRLog.Print("Using permissions without issues:write (fallback-as-issue disabled)")
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "create_pull_request",
		StepName:       "Create Pull Request",
		StepID:         "create_pull_request",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         "", // Legacy - handler manager uses require() to load handler from /tmp/gh-aw/actions
		Permissions:    permissions,
		Outputs:        outputs,
		PreSteps:       preSteps,
		PostSteps:      postSteps,
		Token:          data.SafeOutputs.CreatePullRequests.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CreatePullRequests.TargetRepoSlug,
	})
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
