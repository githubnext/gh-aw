package workflow

import (
	"fmt"
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

// filterNonCopilotAssignees returns assignees excluding "copilot"
func filterNonCopilotAssignees(assignees []string) []string {
	var result []string
	for _, a := range assignees {
		if a != "copilot" {
			result = append(result, a)
		}
	}
	return result
}

// buildCopilotCodingAgentAssignmentStep generates a post-step for assigning Copilot coding agent to created issues
// This step uses the agent token with full precedence chain
func buildCopilotCodingAgentAssignmentStep(configToken, safeOutputsToken string) []string {
	var steps []string

	// Choose the first non-empty custom token for precedence
	effectiveCustomToken := configToken
	if effectiveCustomToken == "" {
		effectiveCustomToken = safeOutputsToken
	}

	// Get the effective agent token with full precedence chain
	effectiveToken := getEffectiveCopilotCodingAgentGitHubToken(effectiveCustomToken)

	steps = append(steps, "      - name: Assign Copilot to created issues\n")
	steps = append(steps, "        if: steps.create_issue.outputs.issues_to_assign_copilot != ''\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io);\n")
	// Load script from external file using require()
	steps = append(steps, "            const { main } = require('/opt/gh-aw/actions/assign_copilot_to_created_issues.cjs');\n")
	steps = append(steps, "            await main({ github, context, core, exec, io });\n")

	return steps
}

// buildCreateOutputIssueJob creates the create_issue job
func (c *Compiler) buildCreateOutputIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.create-issue configuration is required")
	}

	if createIssueLog.Enabled() {
		createIssueLog.Printf("Building create-issue job: workflow=%s, main_job=%s, assignees=%d, labels=%d",
			data.Name, mainJobName, len(data.SafeOutputs.CreateIssues.Assignees), len(data.SafeOutputs.CreateIssues.Labels))
	}

	// Build custom environment variables specific to create-issue using shared helpers
	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", data.SafeOutputs.CreateIssues.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", data.SafeOutputs.CreateIssues.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", data.SafeOutputs.CreateIssues.AllowedLabels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", data.SafeOutputs.CreateIssues.AllowedRepos)...)

	// Add expires value if set
	if data.SafeOutputs.CreateIssues.Expires > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_EXPIRES: \"%d\"\n", data.SafeOutputs.CreateIssues.Expires))
	}

	// Add group flag if set
	customEnvVars = append(customEnvVars, buildTemplatableBoolEnvVar("GH_AW_ISSUE_GROUP", data.SafeOutputs.CreateIssues.Group)...)
	if data.SafeOutputs.CreateIssues.Group != nil {
		createIssueLog.Print("Issue grouping flag set")
	}

	// Add close-older-issues flag if enabled
	customEnvVars = append(customEnvVars, buildTemplatableBoolEnvVar("GH_AW_CLOSE_OLDER_ISSUES", data.SafeOutputs.CreateIssues.CloseOlderIssues)...)
	if data.SafeOutputs.CreateIssues.CloseOlderIssues != nil {
		createIssueLog.Print("Close older issues flag set")
	}

	// Add footer flag if explicitly set to false
	if data.SafeOutputs.CreateIssues.Footer != nil && *data.SafeOutputs.CreateIssues.Footer == "false" {
		customEnvVars = append(customEnvVars, "          GH_AW_FOOTER: \"false\"\n")
		createIssueLog.Print("Footer disabled - XML markers will be included but visible footer content will be omitted")
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreateIssues.TargetRepoSlug)...)

	// Check if copilot is in assignees - if so, we'll output issues for assign_to_agent job
	assignCopilot := hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees)
	if assignCopilot {
		customEnvVars = append(customEnvVars, "          GH_AW_ASSIGN_COPILOT: \"true\"\n")
		createIssueLog.Print("Copilot assignment requested - will output issues_to_assign_copilot for assign_to_agent job")
	}

	// Build post-steps for non-copilot assignees only
	// Copilot assignment must be done in a separate step with the agent token
	var postSteps []string

	// Get the effective GitHub token to use for gh CLI
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	nonCopilotAssignees := filterNonCopilotAssignees(data.SafeOutputs.CreateIssues.Assignees)
	if len(nonCopilotAssignees) > 0 {
		postSteps = buildCopilotParticipantSteps(CopilotParticipantConfig{
			Participants:       nonCopilotAssignees,
			ParticipantType:    "assignee",
			CustomToken:        data.SafeOutputs.CreateIssues.GitHubToken,
			SafeOutputsToken:   safeOutputsToken,
			ConditionStepID:    "create_issue",
			ConditionOutputKey: "issue_number",
		})
	}

	// Add post-step for copilot assignment using agent token
	if assignCopilot {
		postSteps = append(postSteps, buildCopilotCodingAgentAssignmentStep(data.SafeOutputs.CreateIssues.GitHubToken, safeOutputsToken)...)
	}

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number":     "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":        "${{ steps.create_issue.outputs.issue_url }}",
		"temporary_id_map": "${{ steps.create_issue.outputs.temporary_id_map }}",
	}

	// Add issues_to_assign_copilot output if copilot assignment is requested
	if assignCopilot {
		outputs["issues_to_assign_copilot"] = "${{ steps.create_issue.outputs.issues_to_assign_copilot }}"
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "create_issue",
		StepName:       "Create Output Issue",
		StepID:         "create_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCreateIssueScript(),
		ScriptName:     "create_issue", // For custom action mode
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		PostSteps:      postSteps,
		Token:          data.SafeOutputs.CreateIssues.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CreateIssues.TargetRepoSlug,
	})
}
