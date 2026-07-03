package workflow

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/typeutil"
)

var safeOutputsConfigLog = logger.New("workflow:safe_outputs_config")

// ========================================
// Safe Output Configuration Types
// ========================================

// BaseSafeOutputConfig holds common configuration fields for all safe output types
type BaseSafeOutputConfig struct {
	Max                      *string          `yaml:"max,omitempty"`                        // Maximum number of items to create (supports integer or GitHub Actions expression)
	GitHubToken              string           `yaml:"github-token,omitempty"`               // GitHub token for this specific output type
	GitHubApp                *GitHubAppConfig `yaml:"github-app,omitempty"`                 // GitHub App credentials for minting a per-handler installation access token
	Staged                   *TemplatableBool `yaml:"staged,omitempty"`                     // Templatable preview-only mode for this specific output type
	NormalizeClosingKeywords *bool            `yaml:"normalize-closing-keywords,omitempty"` // When true for this output type, strip backticks from recognized issue-closing keywords in body fields.
	// Samples carries deterministic replay samples for the hidden `gh aw compile --use-samples` flag. Each entry is the JSON object passed to the corresponding MCP tool's `tools/call` arguments. Sample-only sidecar fields (e.g. `patch` for create_pull_request) are stripped before the call and used by the replay driver.
	Samples []map[string]any `yaml:"samples,omitempty"`
}

// SafeOutputsConfig holds configuration for automatic output routes
type SafeOutputsConfig struct {
	CreateIssues                           *CreateIssuesConfig                    `yaml:"create-issue,omitempty"`
	CreateDiscussions                      *CreateDiscussionsConfig               `yaml:"create-discussion,omitempty"`
	UpdateDiscussions                      *UpdateDiscussionsConfig               `yaml:"update-discussion,omitempty"`
	CloseDiscussions                       *CloseDiscussionsConfig                `yaml:"close-discussion,omitempty"`
	CloseIssues                            *CloseIssuesConfig                     `yaml:"close-issue,omitempty"`
	ClosePullRequests                      *ClosePullRequestsConfig               `yaml:"close-pull-request,omitempty"`
	MarkPullRequestAsReadyForReview        *MarkPullRequestAsReadyForReviewConfig `yaml:"mark-pull-request-as-ready-for-review,omitempty"`
	AddComments                            *AddCommentsConfig                     `yaml:"add-comment,omitempty"`
	CommentMemory                          *CommentMemoryConfig                   `yaml:"comment-memory,omitempty"` // Persist and update managed memory comments on issues/PRs
	CreatePullRequests                     *CreatePullRequestsConfig              `yaml:"create-pull-request,omitempty"`
	CreatePullRequestReviewComments        *CreatePullRequestReviewCommentsConfig `yaml:"create-pull-request-review-comment,omitempty"`
	SubmitPullRequestReview                *SubmitPullRequestReviewConfig         `yaml:"submit-pull-request-review,omitempty"`           // Submit a PR review with status (APPROVE, REQUEST_CHANGES, COMMENT)
	ReplyToPullRequestReviewComment        *ReplyToPullRequestReviewCommentConfig `yaml:"reply-to-pull-request-review-comment,omitempty"` // Reply to existing review comments on PRs
	ResolvePullRequestReviewThread         *ResolvePullRequestReviewThreadConfig  `yaml:"resolve-pull-request-review-thread,omitempty"`   // Resolve a review thread on a pull request
	CreateCodeScanningAlerts               *CreateCodeScanningAlertsConfig        `yaml:"create-code-scanning-alert,omitempty"`
	AutofixCodeScanningAlert               *AutofixCodeScanningAlertConfig        `yaml:"autofix-code-scanning-alert,omitempty"`
	CreateCheckRun                         *CreateCheckRunConfig                  `yaml:"create-check-run,omitempty"` // Create GitHub Check Runs to report agent analysis results
	AddLabels                              *AddLabelsConfig                       `yaml:"add-labels,omitempty"`
	RemoveLabels                           *RemoveLabelsConfig                    `yaml:"remove-labels,omitempty"`
	ReplaceLabel                           *ReplaceLabelConfig                    `yaml:"replace-label,omitempty"` // Replace one label with another in a single atomic operation
	AddReviewer                            *AddReviewerConfig                     `yaml:"add-reviewer,omitempty"`
	AssignMilestone                        *AssignMilestoneConfig                 `yaml:"assign-milestone,omitempty"`
	AssignToAgent                          *AssignToAgentConfig                   `yaml:"assign-to-agent,omitempty"`
	AssignToUser                           *AssignToUserConfig                    `yaml:"assign-to-user,omitempty"`     // Assign users to issues
	UnassignFromUser                       *UnassignFromUserConfig                `yaml:"unassign-from-user,omitempty"` // Remove assignees from issues
	UpdateIssues                           *UpdateIssuesConfig                    `yaml:"update-issue,omitempty"`
	UpdatePullRequests                     *UpdatePullRequestsConfig              `yaml:"update-pull-request,omitempty"` // Update GitHub pull request title/body
	MergePullRequest                       *MergePullRequestConfig                `yaml:"merge-pull-request,omitempty"`  // Merge pull requests under constrained policy checks
	PushToPullRequestBranch                *PushToPullRequestBranchConfig         `yaml:"push-to-pull-request-branch,omitempty"`
	UploadAssets                           *UploadAssetsConfig                    `yaml:"upload-asset,omitempty"`
	UploadArtifact                         *UploadArtifactConfig                  `yaml:"upload-artifact,omitempty"`              // Upload files as run-scoped GitHub Actions artifacts
	UpdateRelease                          *UpdateReleaseConfig                   `yaml:"update-release,omitempty"`               // Update GitHub release descriptions
	CreateAgentSessions                    *CreateAgentSessionConfig              `yaml:"create-agent-session,omitempty"`         // Create GitHub Copilot coding agent sessions
	UpdateProjects                         *UpdateProjectConfig                   `yaml:"update-project,omitempty"`               // Smart project board management (create/add/update)
	CreateProjects                         *CreateProjectsConfig                  `yaml:"create-project,omitempty"`               // Create GitHub Projects V2
	CreateProjectStatusUpdates             *CreateProjectStatusUpdateConfig       `yaml:"create-project-status-update,omitempty"` // Create GitHub project status updates
	LinkSubIssue                           *LinkSubIssueConfig                    `yaml:"link-sub-issue,omitempty"`               // Link issues as sub-issues
	HideComment                            *HideCommentConfig                     `yaml:"hide-comment,omitempty"`                 // Hide comments
	SetIssueType                           *SetIssueTypeConfig                    `yaml:"set-issue-type,omitempty"`               // Set the type of an issue (empty string clears the type)
	SetIssueField                          *SetIssueFieldConfig                   `yaml:"set-issue-field,omitempty"`              // Set a single issue field value by name/value
	DispatchWorkflow                       *DispatchWorkflowConfig                `yaml:"dispatch-workflow,omitempty"`            // Dispatch workflow_dispatch events to other workflows
	DispatchRepository                     *DispatchRepositoryConfig              `yaml:"dispatch_repository,omitempty"`          // Dispatch repository_dispatch events to external repositories
	CallWorkflow                           *CallWorkflowConfig                    `yaml:"call-workflow,omitempty"`                // Call reusable workflows via workflow_call fan-out
	MissingTool                            *MissingToolConfig                     `yaml:"missing-tool,omitempty"`                 // Optional for reporting missing functionality
	MissingData                            *MissingDataConfig                     `yaml:"missing-data,omitempty"`                 // Optional for reporting missing data required to achieve goals
	NoOp                                   *NoOpConfig                            `yaml:"noop,omitempty"`                         // No-op output for logging only (always available as fallback)
	ReportIncomplete                       *ReportIncompleteConfig                `yaml:"report-incomplete,omitempty"`            // Signal that the task could not be completed due to a tool or infrastructure failure
	ThreatDetection                        *ThreatDetectionConfig                 `yaml:"threat-detection,omitempty"`             // Threat detection configuration
	Jobs                                   map[string]*SafeJobConfig              `yaml:"jobs,omitempty"`                         // Safe-jobs configuration (moved from top-level)
	Scripts                                map[string]*SafeScriptConfig           `yaml:"scripts,omitempty"`                      // Custom inline handlers that run in the safe-output handler loop
	GitHubApp                              *GitHubAppConfig                       `yaml:"github-app,omitempty"`                   // GitHub App credentials for token minting
	URLs                                   string                                 `yaml:"urls,omitempty"`                         // URL sanitization policy: SafeOutputsURLsPolicyAllowedOnly (default) or SafeOutputsURLsPolicyAllowedOrCodeRegion
	AllowedDomains                         []string                               `yaml:"allowed-domains,omitempty"`              // Allowed domains for URL redaction, unioned with network.allowed; supports ecosystem identifiers
	AllowGitHubReferences                  []string                               `yaml:"allowed-github-references,omitempty"`    // Allowed repositories for GitHub references (e.g., ["repo", "org/repo2"])
	Staged                                 *TemplatableBool                       `yaml:"staged,omitempty"`                       // Templatable preview-only mode for all safe outputs
	Env                                    map[string]string                      `yaml:"env,omitempty"`                          // Environment variables to pass to safe output jobs
	GitHubToken                            string                                 `yaml:"github-token,omitempty"`                 // GitHub token for safe output jobs
	MaximumPatchSize                       int                                    `yaml:"max-patch-size,omitempty"`               // Maximum allowed patch size in KB (defaults to 4096)
	MaximumPatchFiles                      int                                    `yaml:"max-patch-files,omitempty"`              // Maximum allowed unique files per create-pull-request patch (defaults to 100)
	RunsOn                                 string                                 `yaml:"runs-on,omitempty"`                      // Runner configuration for safe-outputs jobs
	Messages                               *SafeOutputMessagesConfig              `yaml:"messages,omitempty"`                     // Custom message templates for footer and notifications
	Mentions                               *MentionsConfig                        `yaml:"mentions,omitempty"`                     // Configuration for @mention filtering in safe outputs
	Footer                                 *bool                                  `yaml:"footer,omitempty"`                       // Global footer control - when false, omits visible footer from all safe outputs (XML markers still included)
	GroupReports                           bool                                   `yaml:"group-reports,omitempty"`                // If true, create parent "Failed runs" issue for agent failures (default: false)
	ReportFailureAsIssue                   any                                    `yaml:"report-failure-as-issue,omitempty"`      // Controls failure issue creation: bool, templatable expression string, or []interface{} categories (parsed to ReportFailureAsIssueCategories/ExcludedCategories). Default: true
	ReportFailureAsIssueCategories         []string                               `yaml:"-"`                                      // Parsed failure categories for report-failure-as-issue (internal use only, included categories)
	ReportFailureAsIssueExcludedCategories []string                               `yaml:"-"`                                      // Parsed excluded failure categories for report-failure-as-issue (internal use only, categories starting with "!")
	FailureIssueRepo                       string                                 `yaml:"failure-issue-repo,omitempty"`           // Repository to create failure issues in (format: "owner/repo"), defaults to current repo
	MaxBotMentions                         *string                                `yaml:"max-bot-mentions,omitempty"`             // Maximum bot trigger references (e.g. 'fixes #123') allowed before filtering. Default: 10. Supports integer or GitHub Actions expression.
	Steps                                  []any                                  `yaml:"steps,omitempty"`                        // User-provided steps injected after setup/checkout and before safe-output code
	IDToken                                *string                                `yaml:"id-token,omitempty"`                     // Override id-token permission: "write" to force-add, "none" to disable auto-detection
	ConcurrencyGroup                       string                                 `yaml:"concurrency-group,omitempty"`            // Concurrency group for the safe-outputs job (cancel-in-progress is always false)
	Needs                                  []string                               `yaml:"needs,omitempty"`                        // Additional custom workflow jobs that safe_outputs should depend on
	Environment                            string                                 `yaml:"environment,omitempty"`                  // Override the GitHub deployment environment for the safe-outputs job (defaults to the top-level environment: field)
	Actions                                map[string]*SafeOutputActionConfig     `yaml:"actions,omitempty"`                      // Custom GitHub Actions mounted as safe output tools (resolved at compile time)
	TimeoutMinutes                         int                                    `yaml:"timeout-minutes,omitempty"`              // Timeout for the safe_outputs job in minutes. Defaults to 45.
	AutoInjectedCreateIssue                bool                                   `yaml:"-"`                                      // Internal: true when create-issues was automatically injected by the compiler (not user-configured)
}

// SafeOutputMessagesConfig holds custom message templates for safe-output footer and notification messages
type SafeOutputMessagesConfig struct {
	Footer                         string `yaml:"footer,omitempty" json:"footer,omitempty"`                                                    // Custom footer message template
	FooterInstall                  string `yaml:"footer-install,omitempty" json:"footerInstall,omitempty"`                                     // Custom installation instructions template
	FooterWorkflowRecompile        string `yaml:"footer-workflow-recompile,omitempty" json:"footerWorkflowRecompile,omitempty"`                // Custom footer template for workflow recompile issues
	FooterWorkflowRecompileComment string `yaml:"footer-workflow-recompile-comment,omitempty" json:"footerWorkflowRecompileComment,omitempty"` // Custom footer template for comments on workflow recompile issues
	StagedTitle                    string `yaml:"staged-title,omitempty" json:"stagedTitle,omitempty"`                                         // Custom styled mode title template
	StagedDescription              string `yaml:"staged-description,omitempty" json:"stagedDescription,omitempty"`                             // Custom staged mode description template
	AppendOnlyComments             bool   `yaml:"append-only-comments,omitempty" json:"appendOnlyComments,omitempty"`                          // If true, post run status as new comments instead of updating the activation comment
	ActivationComments             string `yaml:"activation-comments,omitempty" json:"activationComments,omitempty"`                           // If "false", disable all activation/fallback comments entirely. Supports templatable boolean values (literal "true"/"false" or GitHub Actions expressions). Empty/unset preserves default enabled behavior.
	RunStarted                     string `yaml:"run-started,omitempty" json:"runStarted,omitempty"`                                           // Custom workflow activation message template
	RunSuccess                     string `yaml:"run-success,omitempty" json:"runSuccess,omitempty"`                                           // Custom workflow success message template
	RunFailure                     string `yaml:"run-failure,omitempty" json:"runFailure,omitempty"`                                           // Custom workflow failure message template
	DetectionFailure               string `yaml:"detection-failure,omitempty" json:"detectionFailure,omitempty"`                               // Custom detection job failure message template
	PullRequestCreated             string `yaml:"pull-request-created,omitempty" json:"pullRequestCreated,omitempty"`                          // Custom message template for pull request creation link. Placeholders: {item_number}, {item_url}
	IssueCreated                   string `yaml:"issue-created,omitempty" json:"issueCreated,omitempty"`                                       // Custom message template for issue creation link. Placeholders: {item_number}, {item_url}
	CommitPushed                   string `yaml:"commit-pushed,omitempty" json:"commitPushed,omitempty"`                                       // Custom message template for commit push link. Placeholders: {commit_sha}, {short_sha}, {commit_url}
	AgentFailureIssue              string `yaml:"agent-failure-issue,omitempty" json:"agentFailureIssue,omitempty"`                            // Custom footer template for agent failure tracking issues
	AgentFailureComment            string `yaml:"agent-failure-comment,omitempty" json:"agentFailureComment,omitempty"`                        // Custom footer template for comments on agent failure tracking issues
	BodyHeader                     string `yaml:"body-header,omitempty" json:"bodyHeader,omitempty"`                                           // Custom header text prepended to every message body (issues, comments, PRs, discussions). Placeholders: {workflow_name}, {run_url}
}

// MentionsConfig holds configuration for @mention filtering in safe outputs
type MentionsConfig struct {
	// Enabled can be:
	//   true: mentions always allowed (error in strict mode)
	//   false: mentions always escaped
	//   nil: use default behavior with team members and context
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// AllowedCollaborators determines if repository collaborators can be mentioned (default: true)
	AllowedCollaborators *bool `yaml:"allowed-collaborators,omitempty" json:"allowedCollaborators,omitempty"`

	// AllowContext determines if mentions from event context are allowed (default: true)
	AllowContext *bool `yaml:"allow-context,omitempty" json:"allowContext,omitempty"`

	// Allowed is a list of user/bot names always allowed (bots not allowed by default)
	Allowed []string `yaml:"allowed,omitempty" json:"allowed,omitempty"`

	// AllowedTeams is a list of team slugs whose members are always allowed to be mentioned.
	// Accepts "team-slug" (resolved against the current org) or "org/team-slug" format.
	// Requires the workflow token to have read:org scope (a fine-grained PAT, classic PAT with
	// read:org, or a GitHub App with the Members:Read permission). The default GITHUB_TOKEN
	// does not include read:org and will produce a 403/404 warning; team members will be skipped
	// but the workflow will not fail.
	AllowedTeams []string `yaml:"allowed-teams,omitempty" json:"allowedTeams,omitempty"`

	// Max is the maximum number of mentions per message (default: 50)
	Max *int `yaml:"max,omitempty" json:"max,omitempty"`
}

// SecretMaskingConfig holds configuration for secret redaction behavior
type SecretMaskingConfig struct {
	Steps []map[string]any `yaml:"steps,omitempty"` // Additional secret redaction steps to inject after built-in redaction
}

// ========================================
// Safe Output Configuration Extraction
// ========================================
//
// ## Schema Generation Architecture
//
// MCP tool schemas for Safe Outputs are managed through a hybrid approach:
//
// ### Static Schemas (30+ built-in safe output types)
// Defined in: pkg/workflow/js/safe_outputs_tools.json
// - Embedded at compile time via //go:embed directive in pkg/workflow/js.go
// - Contains complete MCP tool definitions with inputSchema for all built-in types
// - Examples: create_issue, create_pull_request, add_comment, update_project, etc.
// - Accessed via GetSafeOutputsToolsJSON() function
//
// ### Dynamic Schema Generation (custom safe-jobs)
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateCustomJobToolDefinition() builds MCP tool schemas from SafeJobConfig
// - Converts job input definitions to JSON Schema format
// - Supports type mapping (string, boolean, number, choice/enum)
// - Enforces required fields and additionalProperties: false
// - Custom job tools are merged with static tools at runtime
//
// ### Schema Filtering
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateFilteredToolsJSON() filters tools based on enabled safe-outputs
// - Only includes tools that are configured in the workflow frontmatter
// - Reduces MCP gateway overhead by exposing only necessary tools
//
// ### Validation
// Implemented in: pkg/workflow/safe_outputs_tools_schema_test.go
// - TestSafeOutputsToolsJSONCompliesWithMCPSchema validates against MCP spec
// - TestEachToolHasRequiredMCPFields checks name, description, inputSchema
// - TestNoTopLevelOneOfAllOfAnyOf prevents unsupported schema constructs
//
// This architecture ensures schema consistency by:
// 1. Using embedded JSON for static schemas (single source of truth)
// 2. Programmatic generation for dynamic schemas (type-safe)
// 3. Automated validation in CI (regression prevention)
//

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	safeOutputsConfigLog.Print("Extracting safe-outputs configuration from frontmatter")

	var config *SafeOutputsConfig

	if output, exists := frontmatter["safe-outputs"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			safeOutputsConfigLog.Printf("Processing safe-outputs configuration with %d top-level keys", len(outputMap))
			config = &SafeOutputsConfig{}

			// Handle create-issue
			issuesConfig := c.parseCreateIssuesConfig(outputMap)
			if issuesConfig != nil {
				safeOutputsConfigLog.Print("Configured create-issue output handler")
				config.CreateIssues = issuesConfig
			}

			// Handle create-agent-session
			agentSessionConfig := c.parseAgentSessionConfig(outputMap)
			if agentSessionConfig != nil {
				config.CreateAgentSessions = agentSessionConfig
			}

			// Handle update-project (smart project board management)
			updateProjectConfig := c.parseUpdateProjectConfig(outputMap)
			if updateProjectConfig != nil {
				config.UpdateProjects = updateProjectConfig
			}

			// Handle create-project
			createProjectConfig := c.parseCreateProjectsConfig(outputMap)
			if createProjectConfig != nil {
				config.CreateProjects = createProjectConfig
			}

			// Handle create-project-status-update (project status updates)
			createProjectStatusUpdateConfig := c.parseCreateProjectStatusUpdateConfig(outputMap)
			if createProjectStatusUpdateConfig != nil {
				config.CreateProjectStatusUpdates = createProjectStatusUpdateConfig
			}

			// Handle create-discussion
			discussionsConfig := c.parseCreateDiscussionsConfig(outputMap)
			if discussionsConfig != nil {
				config.CreateDiscussions = discussionsConfig
			}

			// Handle close-discussion
			closeDiscussionsConfig := c.parseCloseDiscussionsConfig(outputMap)
			if closeDiscussionsConfig != nil {
				config.CloseDiscussions = closeDiscussionsConfig
			}

			// Handle close-issue
			closeIssuesConfig := c.parseCloseIssuesConfig(outputMap)
			if closeIssuesConfig != nil {
				config.CloseIssues = closeIssuesConfig
			}

			// Handle close-pull-request
			closePullRequestsConfig := c.parseClosePullRequestsConfig(outputMap)
			if closePullRequestsConfig != nil {
				config.ClosePullRequests = closePullRequestsConfig
			}

			// Handle mark-pull-request-as-ready-for-review
			markPRReadyConfig := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap)
			if markPRReadyConfig != nil {
				config.MarkPullRequestAsReadyForReview = markPRReadyConfig
			}

			// Handle add-comment
			commentsConfig := c.parseCommentsConfig(outputMap)
			if commentsConfig != nil {
				config.AddComments = commentsConfig
			}

			// Handle create-pull-request
			pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap)
			if pullRequestsConfig != nil {
				safeOutputsConfigLog.Print("Configured create-pull-request output handler")
				config.CreatePullRequests = pullRequestsConfig
			}

			// Handle create-pull-request-review-comment
			prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap)
			if prReviewCommentsConfig != nil {
				config.CreatePullRequestReviewComments = prReviewCommentsConfig
			}

			// Handle submit-pull-request-review
			submitPRReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap)
			if submitPRReviewConfig != nil {
				config.SubmitPullRequestReview = submitPRReviewConfig
			}

			// Handle reply-to-pull-request-review-comment
			replyToPRReviewCommentConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap)
			if replyToPRReviewCommentConfig != nil {
				config.ReplyToPullRequestReviewComment = replyToPRReviewCommentConfig
			}

			// Handle resolve-pull-request-review-thread
			resolvePRReviewThreadConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap)
			if resolvePRReviewThreadConfig != nil {
				config.ResolvePullRequestReviewThread = resolvePRReviewThreadConfig
			}

			// Handle create-code-scanning-alert
			securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap)
			if securityReportsConfig != nil {
				config.CreateCodeScanningAlerts = securityReportsConfig
			}

			// Handle autofix-code-scanning-alert
			autofixCodeScanningAlertConfig := c.parseAutofixCodeScanningAlertConfig(outputMap)
			if autofixCodeScanningAlertConfig != nil {
				config.AutofixCodeScanningAlert = autofixCodeScanningAlertConfig
			}

			// Handle create-check-run
			createCheckRunConfig := c.parseCreateCheckRunConfig(outputMap)
			if createCheckRunConfig != nil {
				config.CreateCheckRun = createCheckRunConfig
			}

			// Parse allowed-domains configuration (additional domains, unioned with network.allowed; supports ecosystem identifiers)
			if allowedDomains, exists := outputMap["allowed-domains"]; exists {
				if domainsArray, ok := allowedDomains.([]any); ok {
					var domainStrings []string
					for _, domain := range domainsArray {
						if domainStr, ok := domain.(string); ok {
							domainStrings = append(domainStrings, domainStr)
						}
					}
					config.AllowedDomains = domainStrings
					safeOutputsConfigLog.Printf("Configured allowed-domains with %d domain(s)", len(domainStrings))
				}
			}

			// Parse URL sanitization policy
			if urls, exists := outputMap["urls"]; exists {
				if urlsStr, ok := urls.(string); ok {
					config.URLs = urlsStr
				}
			}

			// Parse allowed-github-references configuration
			if allowGitHubRefs, exists := outputMap["allowed-github-references"]; exists {
				if refsArray, ok := allowGitHubRefs.([]any); ok {
					refStrings := []string{} // Initialize as empty slice, not nil
					for _, ref := range refsArray {
						if refStr, ok := ref.(string); ok {
							refStrings = append(refStrings, refStr)
						}
					}
					config.AllowGitHubReferences = refStrings
				}
			}

			// Parse add-labels configuration
			addLabelsConfig := c.parseAddLabelsConfig(outputMap)
			if addLabelsConfig != nil {
				config.AddLabels = addLabelsConfig
			}

			// Parse remove-labels configuration
			removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap)
			if removeLabelsConfig != nil {
				config.RemoveLabels = removeLabelsConfig
			}

			// Parse replace-label configuration
			replaceLabelConfig := c.parseReplaceLabelConfig(outputMap)
			if replaceLabelConfig != nil {
				config.ReplaceLabel = replaceLabelConfig
			}

			// Parse add-reviewer configuration
			addReviewerConfig := c.parseAddReviewerConfig(outputMap)
			if addReviewerConfig != nil {
				config.AddReviewer = addReviewerConfig
			}

			// Parse assign-milestone configuration
			assignMilestoneConfig := c.parseAssignMilestoneConfig(outputMap)
			if assignMilestoneConfig != nil {
				config.AssignMilestone = assignMilestoneConfig
			}

			// Handle assign-to-agent
			assignToAgentConfig := c.parseAssignToAgentConfig(outputMap)
			if assignToAgentConfig != nil {
				config.AssignToAgent = assignToAgentConfig
			}

			// Handle assign-to-user
			assignToUserConfig := c.parseAssignToUserConfig(outputMap)
			if assignToUserConfig != nil {
				config.AssignToUser = assignToUserConfig
			}

			// Handle unassign-from-user
			unassignFromUserConfig := c.parseUnassignFromUserConfig(outputMap)
			if unassignFromUserConfig != nil {
				config.UnassignFromUser = unassignFromUserConfig
			}

			// Handle update-issue
			updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap)
			if updateIssuesConfig != nil {
				config.UpdateIssues = updateIssuesConfig
			}

			// Handle update-discussion
			updateDiscussionsConfig := c.parseUpdateDiscussionsConfig(outputMap)
			if updateDiscussionsConfig != nil {
				config.UpdateDiscussions = updateDiscussionsConfig
			}

			// Handle update-pull-request
			updatePullRequestsConfig := c.parseUpdatePullRequestsConfig(outputMap)
			if updatePullRequestsConfig != nil {
				config.UpdatePullRequests = updatePullRequestsConfig
			}

			// Handle merge-pull-request
			mergePullRequestConfig := c.parseMergePullRequestConfig(outputMap)
			if mergePullRequestConfig != nil {
				config.MergePullRequest = mergePullRequestConfig
			}

			// Handle push-to-pull-request-branch
			pushToBranchConfig := c.parsePushToPullRequestBranchConfig(outputMap)
			if pushToBranchConfig != nil {
				config.PushToPullRequestBranch = pushToBranchConfig
			}

			// Handle upload-asset
			uploadAssetsConfig := c.parseUploadAssetConfig(outputMap)
			if uploadAssetsConfig != nil {
				config.UploadAssets = uploadAssetsConfig
			}

			// Handle upload-artifact
			uploadArtifactConfig := c.parseUploadArtifactConfig(outputMap)
			if uploadArtifactConfig != nil {
				config.UploadArtifact = uploadArtifactConfig
			}

			// Handle update-release
			updateReleaseConfig := c.parseUpdateReleaseConfig(outputMap)
			if updateReleaseConfig != nil {
				config.UpdateRelease = updateReleaseConfig
			}

			// Handle link-sub-issue
			linkSubIssueConfig := c.parseLinkSubIssueConfig(outputMap)
			if linkSubIssueConfig != nil {
				config.LinkSubIssue = linkSubIssueConfig
			}

			// Handle hide-comment
			hideCommentConfig := c.parseHideCommentConfig(outputMap)
			if hideCommentConfig != nil {
				config.HideComment = hideCommentConfig
			}

			// Handle set-issue-type
			setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap)
			if setIssueTypeConfig != nil {
				config.SetIssueType = setIssueTypeConfig
			}

			// Handle set-issue-field
			setIssueFieldConfig := c.parseSetIssueFieldConfig(outputMap)
			if setIssueFieldConfig != nil {
				config.SetIssueField = setIssueFieldConfig
			}

			// Handle dispatch-workflow
			dispatchWorkflowConfig := c.parseDispatchWorkflowConfig(outputMap)
			if dispatchWorkflowConfig != nil {
				config.DispatchWorkflow = dispatchWorkflowConfig
			}

			// Handle dispatch_repository
			dispatchRepositoryConfig := c.parseDispatchRepositoryConfig(outputMap)
			if dispatchRepositoryConfig != nil {
				config.DispatchRepository = dispatchRepositoryConfig
			}

			// Handle call-workflow
			callWorkflowConfig := c.parseCallWorkflowConfig(outputMap)
			if callWorkflowConfig != nil {
				config.CallWorkflow = callWorkflowConfig
			}

			// Handle missing-tool (parse configuration if present, or enable by default)
			missingToolConfig := c.parseMissingToolConfig(outputMap)
			if missingToolConfig != nil {
				config.MissingTool = missingToolConfig
			} else {
				// Enable missing-tool by default if safe-outputs exists and it wasn't explicitly disabled
				if _, exists := outputMap["missing-tool"]; !exists {
					trueVal := "true"
					config.MissingTool = &MissingToolConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
				}
			}

			// Handle missing-data (parse configuration if present, or enable by default)
			missingDataConfig := c.parseMissingDataConfig(outputMap)
			if missingDataConfig != nil {
				config.MissingData = missingDataConfig
			} else {
				// Enable missing-data by default if safe-outputs exists and it wasn't explicitly disabled
				if _, exists := outputMap["missing-data"]; !exists {
					trueVal := "true"
					config.MissingData = &MissingDataConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
				}
			}

			// Handle noop (parse configuration if present, or enable by default as fallback)
			noopConfig := c.parseNoOpConfig(outputMap)
			if noopConfig != nil {
				config.NoOp = noopConfig
			} else {
				// Enable noop by default if safe-outputs exists and it wasn't explicitly disabled
				// This ensures there's always a fallback for transparency
				if _, exists := outputMap["noop"]; !exists {
					config.NoOp = &NoOpConfig{}
					config.NoOp.Max = defaultIntStr(1) // Default max
					trueVal := "true"
					config.NoOp.ReportAsIssue = &trueVal // Default to reporting to issue
				}
			}

			// Handle report-incomplete (parse configuration if present, or enable by default)
			reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap)
			if reportIncompleteConfig != nil {
				config.ReportIncomplete = reportIncompleteConfig
			} else {
				// Enable report-incomplete by default if safe-outputs exists and it wasn't explicitly disabled.
				// This ensures agents always have a first-class channel to signal task incompletion.
				if _, exists := outputMap["report-incomplete"]; !exists {
					trueVal := "true"
					config.ReportIncomplete = &ReportIncompleteConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
				}
			}

			// Handle staged flag
			if err := preprocessBoolFieldAsString(outputMap, "staged", safeOutputsConfigLog); err != nil {
				safeOutputsConfigLog.Printf("staged: %v", err)
			} else if staged, exists := outputMap["staged"]; exists {
				if stagedStr, ok := staged.(string); ok && stagedStr != "" {
					value := TemplatableBool(stagedStr)
					config.Staged = &value
				}
			}
			if c.forceStaged {
				value := TemplatableBool("true")
				config.Staged = &value
			}

			// Handle env configuration
			if env, exists := outputMap["env"]; exists {
				if envMap, ok := env.(map[string]any); ok {
					config.Env = make(map[string]string)
					for key, value := range envMap {
						if valueStr, ok := value.(string); ok {
							config.Env[key] = valueStr
						}
					}
				}
			}

			// Handle github-token configuration
			if githubToken, exists := outputMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					config.GitHubToken = githubTokenStr
				}
			}

			// Handle max-patch-size configuration
			if maxPatchSize, exists := outputMap["max-patch-size"]; exists {
				switch v := maxPatchSize.(type) {
				case int:
					if v >= 1 {
						config.MaximumPatchSize = v
					}
				case int64:
					if v >= 1 {
						config.MaximumPatchSize = int(v)
					}
				case uint64:
					if v >= 1 {
						config.MaximumPatchSize = int(v)
					}
				case float64:
					intVal := int(v)
					// Warn if truncation occurs (value has fractional part)
					if v != float64(intVal) {
						safeOutputsConfigLog.Printf("max-patch-size: float value %.2f truncated to integer %d", v, intVal)
					}
					if intVal >= 1 {
						config.MaximumPatchSize = intVal
					}
				}
			}

			// Set default value if not specified or invalid
			if config.MaximumPatchSize == 0 {
				config.MaximumPatchSize = 4096 // Default to 4MB = 4096 KB
			}

			// Handle max-patch-files configuration (maximum unique files allowed in
			// a create-pull-request patch). Mirrors max-patch-size handling above,
			// with explicit bounds checks before narrowing to int so that very
			// large source values can't overflow/wrap into a negative or wrapped
			// number that would silently fall back to the default.
			if maxPatchFiles, exists := outputMap["max-patch-files"]; exists {
				switch v := maxPatchFiles.(type) {
				case int:
					if v >= 1 {
						config.MaximumPatchFiles = v
					}
				case int64:
					if v >= 1 {
						if v > int64(math.MaxInt) {
							safeOutputsConfigLog.Printf("max-patch-files: int64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.MaximumPatchFiles = math.MaxInt
						} else {
							config.MaximumPatchFiles = int(v)
						}
					}
				case uint64:
					if v >= 1 {
						if v > uint64(math.MaxInt) {
							safeOutputsConfigLog.Printf("max-patch-files: uint64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.MaximumPatchFiles = math.MaxInt
						} else {
							config.MaximumPatchFiles = int(v)
						}
					}
				case float64:
					// Reject NaN/Inf and clamp out-of-range floats before
					// narrowing — `int(NaN)` and `int(±Inf)` are
					// implementation-defined and can produce surprising
					// values (including 0, which would silently fall back
					// to the default).
					if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
						safeOutputsConfigLog.Printf("max-patch-files: float value %.2f is out of range, ignoring", v)
						break
					}
					intVal := int(v)
					if v != float64(intVal) {
						safeOutputsConfigLog.Printf("max-patch-files: float value %.2f truncated to integer %d", v, intVal)
					}
					if intVal >= 1 {
						config.MaximumPatchFiles = intVal
					}
				}
			}

			// Set default value if not specified or invalid
			if config.MaximumPatchFiles == 0 {
				config.MaximumPatchFiles = 100 // Default to 100 unique files
			}

			// Handle threat-detection
			threatDetectionConfig := c.parseThreatDetectionConfig(outputMap)
			if threatDetectionConfig != nil {
				config.ThreatDetection = threatDetectionConfig
			}

			// Handle runs-on configuration
			if runsOn, exists := outputMap["runs-on"]; exists {
				config.RunsOn = renderRunsOnSnippet(runsOn)
			}

			// Handle timeout-minutes configuration
			if timeoutMinutes, exists := outputMap["timeout-minutes"]; exists {
				switch v := timeoutMinutes.(type) {
				case int:
					if v >= 1 {
						config.TimeoutMinutes = v
					}
				case int64:
					if v >= 1 {
						if v > int64(math.MaxInt) {
							safeOutputsConfigLog.Printf("timeout-minutes: int64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.TimeoutMinutes = math.MaxInt
						} else {
							config.TimeoutMinutes = int(v)
						}
					}
				case uint64:
					if v >= 1 {
						if v > uint64(math.MaxInt) {
							safeOutputsConfigLog.Printf("timeout-minutes: uint64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.TimeoutMinutes = math.MaxInt
						} else {
							config.TimeoutMinutes = int(v)
						}
					}
				case float64:
					// Reject NaN/Inf and out-of-range floats before narrowing — int(NaN)/int(±Inf)
					// are implementation-defined and can produce surprising values.
					if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
						safeOutputsConfigLog.Printf("timeout-minutes: float value %.2f is out of range, ignoring", v)
						break
					}
					intVal := int(v)
					if v != float64(intVal) {
						safeOutputsConfigLog.Printf("timeout-minutes: float value %.2f truncated to integer %d", v, intVal)
					}
					if intVal >= 1 {
						config.TimeoutMinutes = intVal
					}
				}
			}

			// Handle messages configuration
			if messages, exists := outputMap["messages"]; exists {
				if messagesMap, ok := messages.(map[string]any); ok {
					config.Messages = parseMessagesConfig(messagesMap)
				}
			}

			// Handle activation-comments at safe-outputs top level (templatable boolean)
			if err := preprocessBoolFieldAsString(outputMap, "activation-comments", safeOutputsConfigLog); err != nil {
				safeOutputsConfigLog.Printf("activation-comments: %v", err)
			}
			if activationComments, exists := outputMap["activation-comments"]; exists {
				if activationCommentsStr, ok := activationComments.(string); ok && activationCommentsStr != "" {
					if config.Messages == nil {
						config.Messages = &SafeOutputMessagesConfig{}
					}
					config.Messages.ActivationComments = activationCommentsStr
				}
			}

			// Handle mentions configuration
			if mentions, exists := outputMap["mentions"]; exists {
				config.Mentions = parseMentionsConfig(mentions)
			}

			// Handle global footer flag
			if footer, exists := outputMap["footer"]; exists {
				if footerBool, ok := footer.(bool); ok {
					config.Footer = &footerBool
					safeOutputsConfigLog.Printf("Global footer control: %t", footerBool)
				}
			}

			// Handle group-reports flag
			if groupReports, exists := outputMap["group-reports"]; exists {
				if groupReportsBool, ok := groupReports.(bool); ok {
					config.GroupReports = groupReportsBool
					safeOutputsConfigLog.Printf("Group reports control: %t", groupReportsBool)
				}
			}

			// Handle report-failure-as-issue as templatable bool or array of categories.
			if reportFailureAsIssue, exists := outputMap["report-failure-as-issue"]; exists {
				// Support []any category filters.
				if categoriesList, ok := reportFailureAsIssue.([]any); ok {
					// Parse as array of category strings, separating included (no prefix) and excluded (! prefix)
					includedCategories := make([]string, 0, len(categoriesList))
					excludedCategories := make([]string, 0, len(categoriesList))
					for _, cat := range categoriesList {
						if catStr, ok := cat.(string); ok {
							if category, found := strings.CutPrefix(catStr, "!"); found {
								// Excluded category: "!" prefix was found and removed
								excludedCategories = append(excludedCategories, category)
							} else {
								// Included category: no prefix
								includedCategories = append(includedCategories, catStr)
							}
						}
					}
					config.ReportFailureAsIssue = reportFailureAsIssue // Preserve original value for proper serialization
					config.ReportFailureAsIssueCategories = includedCategories
					config.ReportFailureAsIssueExcludedCategories = excludedCategories
					if len(includedCategories) > 0 && len(excludedCategories) > 0 {
						safeOutputsConfigLog.Printf("Report failure as issue with include filter: %v, exclude filter: %v", includedCategories, excludedCategories)
					} else if len(includedCategories) > 0 {
						safeOutputsConfigLog.Printf("Report failure as issue with include filter: %v", includedCategories)
					} else if len(excludedCategories) > 0 {
						safeOutputsConfigLog.Printf("Report failure as issue with exclude filter: %v", excludedCategories)
					}
				} else {
					// Support bool and templatable string values.
					if err := preprocessBoolFieldAsString(outputMap, "report-failure-as-issue", safeOutputsConfigLog); err != nil {
						safeOutputsConfigLog.Printf("Failed to preprocess report-failure-as-issue field: %v (ignoring invalid value and leaving field unset)", err)
					} else {
						if reportFailureAsIssueStr, ok := outputMap["report-failure-as-issue"].(string); ok {
							switch reportFailureAsIssueStr {
							case "true":
								config.ReportFailureAsIssue = true
							case "false":
								config.ReportFailureAsIssue = false
							default:
								config.ReportFailureAsIssue = reportFailureAsIssueStr
							}
							safeOutputsConfigLog.Printf("Report failure as issue: %v", config.ReportFailureAsIssue)
						} else if reportFailureAsIssueBool, ok := outputMap["report-failure-as-issue"].(bool); ok {
							config.ReportFailureAsIssue = reportFailureAsIssueBool
							safeOutputsConfigLog.Printf("Report failure as issue: %t", reportFailureAsIssueBool)
						}
					}
				}
			}

			// Handle failure-issue-repo (repository for failure issues, format: "owner/repo")
			if failureIssueRepo, exists := outputMap["failure-issue-repo"]; exists {
				if failureIssueRepoStr, ok := failureIssueRepo.(string); ok && failureIssueRepoStr != "" {
					config.FailureIssueRepo = failureIssueRepoStr
					safeOutputsConfigLog.Printf("Failure issue repo: %s", failureIssueRepoStr)
				}
			}

			// Handle max-bot-mentions (templatable integer)
			if err := preprocessIntFieldAsString(outputMap, "max-bot-mentions", safeOutputsConfigLog); err != nil {
				safeOutputsConfigLog.Printf("max-bot-mentions: %v", err)
			} else if maxBotMentions, exists := outputMap["max-bot-mentions"]; exists {
				if maxBotMentionsStr, ok := maxBotMentions.(string); ok {
					config.MaxBotMentions = &maxBotMentionsStr
				}
			}

			// Handle steps (user-provided steps injected after checkout/setup, before safe-output code)
			if steps, exists := outputMap["steps"]; exists {
				if stepsList, ok := steps.([]any); ok {
					config.Steps = stepsList
					safeOutputsConfigLog.Printf("Configured %d user-provided steps for safe-outputs", len(stepsList))
				}
			}

			// Handle id-token permission override ("write" to force-add, "none" to disable auto-detection)
			if idToken, exists := outputMap["id-token"]; exists {
				if idTokenStr, ok := idToken.(string); ok {
					if idTokenStr == "write" || idTokenStr == "none" {
						config.IDToken = &idTokenStr
						safeOutputsConfigLog.Printf("Configured id-token permission override: %s", idTokenStr)
					} else {
						safeOutputsConfigLog.Printf("Warning: unrecognized safe-outputs id-token value %q (expected \"write\" or \"none\"); ignoring", idTokenStr)
					}
				}
			}

			// Handle concurrency-group configuration
			if concurrencyGroup, exists := outputMap["concurrency-group"]; exists {
				if concurrencyGroupStr, ok := concurrencyGroup.(string); ok && concurrencyGroupStr != "" {
					config.ConcurrencyGroup = concurrencyGroupStr
					safeOutputsConfigLog.Printf("Configured concurrency-group for safe-outputs job: %s", concurrencyGroupStr)
				}
			}

			// Handle needs configuration
			if needsValue, exists := outputMap["needs"]; exists {
				if needsArray, ok := needsValue.([]any); ok {
					for _, need := range needsArray {
						if needStr, ok := need.(string); ok && needStr != "" {
							config.Needs = append(config.Needs, needStr)
						}
					}
					if len(config.Needs) > 0 {
						safeOutputsConfigLog.Printf("Configured %d explicit safe-outputs needs dependency(ies)", len(config.Needs))
					}
				}
			}

			// Handle environment configuration (override for safe-outputs job; falls back to top-level environment)
			config.Environment = c.extractTopLevelYAMLSection(outputMap, "environment")
			if config.Environment != "" {
				safeOutputsConfigLog.Printf("Configured environment override for safe-outputs job: %s", config.Environment)
			}

			// Handle jobs (safe-jobs must be under safe-outputs)
			if jobs, exists := outputMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := NewCompiler() // Create a temporary compiler instance for parsing
					config.Jobs = c.parseSafeJobsConfig(jobsMap)
				}
			}

			// Handle scripts (inline handlers that run in the safe-output handler loop)
			if scripts, exists := outputMap["scripts"]; exists {
				if scriptsMap, ok := scripts.(map[string]any); ok {
					config.Scripts = parseSafeScriptsConfig(scriptsMap)
					safeOutputsConfigLog.Printf("Configured %d custom safe-output script(s)", len(config.Scripts))
				}
			}

			// Handle actions (custom GitHub Actions mounted as safe output tools)
			if actions, exists := outputMap["actions"]; exists {
				if actionsMap, ok := actions.(map[string]any); ok {
					config.Actions = parseActionsConfig(actionsMap)
					safeOutputsConfigLog.Printf("Configured %d custom safe-output action(s)", len(config.Actions))
				}
			}

			// Handle app configuration for GitHub App token minting
			if app, exists := outputMap["github-app"]; exists {
				if appMap, ok := app.(map[string]any); ok {
					config.GitHubApp = parseAppConfig(appMap)
				}
			}
		}
	}

	// Apply default threat detection whenever safe-outputs are configured and threat-detection
	// is not explicitly disabled. Detection is always on unless threat-detection is false.
	if config != nil && config.ThreatDetection == nil {
		if output, exists := frontmatter["safe-outputs"]; exists {
			if outputMap, ok := output.(map[string]any); ok {
				if _, exists := outputMap["threat-detection"]; !exists {
					// Only apply default if threat-detection key doesn't exist
					safeOutputsConfigLog.Print("Applying default threat-detection configuration")
					config.ThreatDetection = &ThreatDetectionConfig{}
				}
			}
		}
	}

	// Force-disable threat detection when --use-samples is active: the replay driver
	// emits synthetic outputs solely for deterministic end-to-end tests, and running
	// an LLM-backed detection pass would defeat that determinism.
	if config != nil && c.useSamples && config.ThreatDetection != nil {
		safeOutputsConfigLog.Print("Disabling threat-detection because --use-samples is set")
		config.ThreatDetection = nil
	}

	if config != nil {
		safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	} else {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
	}

	return config
}

// parseBaseSafeOutputConfig parses common fields (max, github-token, github-app, staged) from a config map.
// If defaultMax is provided (> 0), it will be set as the default value for config.Max
// before parsing the max field from configMap. Supports both integer values and GitHub
// Actions expression strings (e.g. "${{ inputs.max }}").
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	// Set default max if provided
	if defaultMax > 0 {
		safeOutputsConfigLog.Printf("Setting default max: %d", defaultMax)
		config.Max = defaultIntStr(defaultMax)
	}

	// Parse max (this will override the default if present in configMap)
	if max, exists := configMap["max"]; exists {
		switch v := max.(type) {
		case string:
			// Accept GitHub Actions expression strings
			if strings.HasPrefix(v, "${{") && strings.HasSuffix(v, "}}") {
				safeOutputsConfigLog.Printf("Parsed max as GitHub Actions expression: %s", v)
				config.Max = &v
			}
		default:
			// Convert integer/float64/etc to string via typeutil.ParseIntValue
			if maxInt, ok := typeutil.ParseIntValue(max); ok {
				safeOutputsConfigLog.Printf("Parsed max as integer: %d", maxInt)
				s := defaultIntStr(maxInt)
				config.Max = s
			}
		}
	}

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			safeOutputsConfigLog.Print("Parsed custom github-token from config")
			config.GitHubToken = githubTokenStr
		}
	}

	// Parse github-app (per-handler GitHub App credentials for token minting)
	if app, exists := configMap["github-app"]; exists {
		if appMap, ok := app.(map[string]any); ok {
			safeOutputsConfigLog.Print("Parsed custom github-app from config")
			config.GitHubApp = parseAppConfig(appMap)
		}
	}

	// Parse staged flag (per-handler staged mode)
	if err := preprocessBoolFieldAsString(configMap, "staged", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("Invalid staged value: %v", err)
	} else if staged, exists := configMap["staged"]; exists {
		if stagedStr, ok := staged.(string); ok && stagedStr != "" {
			safeOutputsConfigLog.Printf("Parsed staged flag: %s", stagedStr)
			value := TemplatableBool(stagedStr)
			config.Staged = &value
		}
	}

	// Parse samples list (hidden feature: deterministic replay samples for --use-samples).
	// Accepts either a YAML list of objects, or a single object that is auto-wrapped
	// into a one-element list. The JSON schema rejects scalar/string shapes so we
	// don't need a defensive YAML-string branch here.
	if samples, exists := configMap["samples"]; exists {
		parsed := parseSamplesValue(samples)
		if len(parsed) > 0 {
			safeOutputsConfigLog.Printf("Parsed %d samples entries", len(parsed))
			config.Samples = parsed
		}
	}
}

// parseSamplesValue normalizes a `samples` frontmatter value into a list of
// objects. Accepted shapes:
//   - YAML list of mappings: returned as-is
//   - single YAML mapping: wrapped into a one-element list
//
// Any other shape returns an empty slice — schema validation rejects those
// shapes upstream and we keep this parser strict to match.
func parseSamplesValue(samples any) []map[string]any {
	switch v := samples.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			} else if mStr, ok := item.(map[string]string); ok {
				converted := make(map[string]any, len(mStr))
				for k, s := range mStr {
					converted[k] = s
				}
				out = append(out, converted)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{v}
	default:
		return nil
	}
}

// SafeOutputStepConfig holds configuration for building a single safe output step
// within the consolidated safe-outputs job
type SafeOutputStepConfig struct {
	StepName                   string            // Human-readable step name (e.g., "Create Issue")
	StepID                     string            // Step ID for referencing outputs (e.g., "create_issue")
	Script                     string            // JavaScript script to execute (for inline mode)
	ScriptName                 string            // Name of the script in the registry (for file mode)
	CustomEnvVars              []string          // Environment variables specific to this step
	Condition                  ConditionNode     // Step-level condition (if clause)
	Token                      string            // GitHub token for this step
	UseCopilotRequestsToken    bool              // Whether to use Copilot requests token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use Copilot coding agent token preference chain
	PreSteps                   []string          // Optional steps to run before the script step
	PostSteps                  []string          // Optional steps to run after the script step
	Outputs                    map[string]string // Outputs from this step
	ContinueOnError            bool              // Whether to continue the job even if this step fails (continue-on-error: true)
}

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		safeOutputsConfigLog.Print("No safe-outputs configuration, skipping handler manager config")
		return
	}

	safeOutputsConfigLog.Print("Building handler manager configuration for safe-outputs")
	// config holds both per-handler configs (keyed by handler name, e.g. "add_comment") and
	// global runtime knobs (e.g. "mentions") that safe_output_handler_manager.cjs forwards to
	// specific handlers at startup. Handler names are the reserved keys defined in handlerRegistry;
	// non-handler keys ("mentions") are documented in safe_outputs_config_generation.go.
	config := make(map[string]any)

	// Collect engine-specific manifest files and path prefixes (AgentFileProvider interface).
	// These are merged with the global runtime-derived lists so that engine-specific
	// instruction files (e.g. CLAUDE.md, .claude/, AGENTS.md) are automatically protected.
	extraManifestFiles, extraPathPrefixes := c.getEngineAgentFileInfo(data)
	fullManifestFiles := getAllManifestFiles(extraManifestFiles...)
	fullPathPrefixes := getProtectedPathPrefixes(extraPathPrefixes...)

	// For workflow_call relay workflows, inject the resolved platform repo and ref into the
	// dispatch_workflow handler config so dispatch targets the host repo, not the caller's.
	safeOutputs := data.SafeOutputs
	if hasWorkflowCallTrigger(data.On) && safeOutputs.DispatchWorkflow != nil {
		if safeOutputs.DispatchWorkflow.TargetRepoSlug == "" {
			safeOutputs = safeOutputsWithDispatchTargetRepo(safeOutputs, "${{ needs.activation.outputs.target_repo }}")
			safeOutputsConfigLog.Print("Injecting target_repo into dispatch_workflow config for workflow_call relay")
		}
		if safeOutputs.DispatchWorkflow.TargetRef == "" {
			safeOutputs = safeOutputsWithDispatchTargetRef(safeOutputs, "${{ needs.activation.outputs.target_ref }}")
			safeOutputsConfigLog.Print("Injecting target_ref into dispatch_workflow config for workflow_call relay")
		}
	}

	// Build configuration for each handler using the registry
	for handlerName, builder := range handlerRegistry {
		handlerConfig := builder(safeOutputs)
		// Include handler if:
		// 1. It returns a non-nil config (explicitly enabled, even if empty)
		// 2. For auto-enabled handlers, include even with empty config
		if handlerConfig != nil {
			injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerConfig, data)
			injectCheckoutMapping(handlerName, handlerConfig, data)
			// Augment protected-files protection with engine-specific files for handlers that use it.
			if _, hasProtected := handlerConfig["protected_files"]; hasProtected {
				// Extract per-handler exclusions set by the handler builder (sentinel key).
				// These are compile-time overrides and must not be forwarded to the runtime.
				excludeFiles := ParseStringArrayFromConfig(handlerConfig, "_protected_files_exclude", nil)
				delete(handlerConfig, "_protected_files_exclude")

				handlerConfig["protected_files"] = sliceutil.Exclude(fullManifestFiles, excludeFiles...)
				filteredPrefixes := sliceutil.Exclude(fullPathPrefixes, excludeFiles...)
				if len(filteredPrefixes) > 0 {
					handlerConfig["protected_path_prefixes"] = filteredPrefixes
				} else {
					delete(handlerConfig, "protected_path_prefixes")
				}
				// Compute which top-level dot-folder prefixes are excluded so the runtime
				// dot-folder check can skip them.
				if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
					handlerConfig["protected_dot_folder_excludes"] = dotFolderExcludes
				}
			}
			safeOutputsConfigLog.Printf("Adding %s handler configuration", handlerName)
			config[handlerName] = handlerConfig
		}
	}

	// Include top-level mentions configuration so the handler manager can pass it to
	// markdown-producing handlers that call sanitizeContent with allowed aliases.
	if safeOutputs.Mentions != nil {
		mentionsCfg := buildMentionsHandlerConfig(safeOutputs.Mentions)
		if len(mentionsCfg) > 0 {
			config["mentions"] = mentionsCfg
		}
	}

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		safeOutputsConfigLog.Printf("Marshaling handler config with %d handlers", len(config))
		configJSON, err := json.Marshal(config)
		if err != nil {
			safeOutputsConfigLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
		safeOutputsConfigLog.Printf("Added handler config env var: size=%d bytes", len(configStr))
	} else {
		safeOutputsConfigLog.Print("No handlers configured, skipping config env var")
	}
}

// buildMentionsHandlerConfig converts a MentionsConfig into the map format used by
// GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG so safe_output_handler_manager.cjs can pass
// the top-level mentions policy through to mention-aware handlers.
func buildMentionsHandlerConfig(m *MentionsConfig) map[string]any {
	cfg := make(map[string]any)
	if m.Enabled != nil {
		cfg["enabled"] = *m.Enabled
	}
	if m.AllowedCollaborators != nil {
		cfg["allowedCollaborators"] = *m.AllowedCollaborators
	}
	if m.AllowContext != nil {
		cfg["allowContext"] = *m.AllowContext
	}
	if len(m.Allowed) > 0 {
		cfg["allowed"] = m.Allowed
	}
	if len(m.AllowedTeams) > 0 {
		cfg["allowedTeams"] = m.AllowedTeams
	}
	if m.Max != nil {
		cfg["max"] = *m.Max
	}
	return cfg
}

// safeOutputsWithDispatchTargetRepo returns a shallow copy of cfg with the dispatch_workflow
// TargetRepoSlug overridden to targetRepo. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRepo(cfg *SafeOutputsConfig, targetRepo string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRepoSlug = targetRepo
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// safeOutputsWithDispatchTargetRef returns a shallow copy of cfg with the dispatch_workflow
// TargetRef overridden to targetRef. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRef(cfg *SafeOutputsConfig, targetRef string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRef = targetRef
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// getEngineAgentFileInfo returns the engine-specific manifest filenames and path prefixes
// by type-asserting the active engine to AgentFileProvider.  Returns empty slices when
// the engine is not set or does not implement the interface.
func (c *Compiler) getEngineAgentFileInfo(data *WorkflowData) (manifestFiles []string, pathPrefixes []string) {
	if data == nil || data.EngineConfig == nil {
		return nil, nil
	}
	engine, err := c.engineRegistry.GetEngine(data.EngineConfig.ID)
	if err != nil {
		safeOutputsConfigLog.Printf("Engine lookup failed for %q: %v — skipping agent manifest file injection", data.EngineConfig.ID, err)
		return nil, nil
	}
	if engine == nil {
		return nil, nil
	}
	provider, ok := engine.(AgentFileProvider)
	if !ok {
		return nil, nil
	}
	safeOutputsConfigLog.Printf("Engine %s provides AgentFileProvider: files=%v, prefixes=%v",
		data.EngineConfig.ID, provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes())
	return provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes()
}
