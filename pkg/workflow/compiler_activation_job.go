package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var compilerActivationJobLog = logger.New("workflow:compiler_activation_job")

var activationMetadataTriggerFields = map[string]struct{}{
	"reaction":       {},
	"status-comment": {},
	"command":        {},
	"slash_command":  {},
	"label_command":  {},
	"stop-after":     {},
	"github-token":   {},
	"github-app":     {},
}

// buildActivationJob creates the activation job that handles timestamp checking, reactions, and locking.
// This job depends on the pre-activation job if it exists, and runs before the main agent job.
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool, workflowRunRepoSafety string, lockFilename string) (*Job, error) {
	ctx, err := c.newActivationJobBuildContext(data, preActivationJobCreated, workflowRunRepoSafety, lockFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to create activation job build context: %w", err)
	}

	if err := c.addActivationFeedbackAndValidationSteps(ctx); err != nil {
		return nil, err
	}
	if err := c.addActivationRepositoryAndOutputSteps(ctx); err != nil {
		return nil, err
	}
	if err := c.addActivationCommandAndLabelOutputs(ctx); err != nil {
		return nil, err
	}

	c.configureActivationNeedsAndCondition(ctx)
	compilerActivationJobLog.Print("Generating prompt in activation job")
	c.generatePromptInActivationJob(&ctx.steps, data, preActivationJobCreated, ctx.customJobsBeforeActivation)
	c.addActivationArtifactUploadStep(ctx)
	if len(ctx.steps) == 0 {
		ctx.steps = append(ctx.steps, "      - run: echo \"Activation success\"\n")
	}

	if c.actionMode.IsScript() {
		ctx.steps = append(ctx.steps, c.generateScriptModeCleanupStep())
	}

	return &Job{
		Name:                       string(constants.ActivationJobName),
		If:                         ctx.activationCondition,
		HasWorkflowRunSafetyChecks: workflowRunRepoSafety != "",
		RunsOn:                     c.formatFrameworkJobRunsOn(data),
		Permissions:                c.buildActivationPermissions(ctx),
		Environment:                c.buildActivationEnvironment(ctx),
		Steps:                      ctx.steps,
		Outputs:                    ctx.outputs,
		Needs:                      ctx.activationNeeds,
	}, nil
}

func addActivationInteractionPermissions(
	perms *Permissions,
	onSection string,
	hasReaction bool,
	reactionIncludesIssues bool,
	reactionIncludesPullRequests bool,
	reactionIncludesDiscussions bool,
	hasStatusComment bool,
	statusCommentIncludesIssues bool,
	statusCommentIncludesPullRequests bool,
	statusCommentIncludesDiscussions bool,
) {
	if perms == nil {
		return
	}
	permsMap := make(map[PermissionScope]PermissionLevel)
	addActivationInteractionPermissionsMap(
		permsMap,
		onSection,
		hasReaction,
		reactionIncludesIssues,
		reactionIncludesPullRequests,
		reactionIncludesDiscussions,
		hasStatusComment,
		statusCommentIncludesIssues,
		statusCommentIncludesPullRequests,
		statusCommentIncludesDiscussions,
	)
	for scope, level := range permsMap {
		perms.Set(scope, level)
	}
}

func addActivationInteractionPermissionsMap(
	permsMap map[PermissionScope]PermissionLevel,
	onSection string,
	hasReaction bool,
	reactionIncludesIssues bool,
	reactionIncludesPullRequests bool,
	reactionIncludesDiscussions bool,
	hasStatusComment bool,
	statusCommentIncludesIssues bool,
	statusCommentIncludesPullRequests bool,
	statusCommentIncludesDiscussions bool,
) {
	if !hasReaction && !hasStatusComment {
		return
	}

	// Fallback for unit tests or synthetic WorkflowData instances that do not populate the "on" section.
	// Real compiled workflows always have a populated trigger section.
	if onSection == "" {
		compilerActivationJobLog.Print("Empty on section while computing activation permissions; using broad fallback permissions")
		addBroadActivationInteractionPermissions(
			permsMap,
			hasReaction,
			reactionIncludesIssues,
			reactionIncludesPullRequests,
			reactionIncludesDiscussions,
			hasStatusComment,
			statusCommentIncludesIssues,
			statusCommentIncludesPullRequests,
			statusCommentIncludesDiscussions,
		)
		return
	}

	eventSet, eventSetParsed := activationEventSet(onSection)
	if !eventSetParsed {
		compilerActivationJobLog.Print("Unable to parse activation trigger events while computing permissions; using broad fallback permissions")
		addBroadActivationInteractionPermissions(
			permsMap,
			hasReaction,
			reactionIncludesIssues,
			reactionIncludesPullRequests,
			reactionIncludesDiscussions,
			hasStatusComment,
			statusCommentIncludesIssues,
			statusCommentIncludesPullRequests,
			statusCommentIncludesDiscussions,
		)
		return
	}

	hasIssuesEvent := eventSet["issues"]
	hasIssueCommentEvent := eventSet["issue_comment"]
	hasPullRequestEvent := eventSet["pull_request"]
	hasPullRequestReviewCommentEvent := eventSet["pull_request_review_comment"]
	hasDiscussionEvent := eventSet["discussion"]
	hasDiscussionCommentEvent := eventSet["discussion_comment"]

	if hasReaction {
		// Reactions on issues, issue comments, and pull requests use issues endpoints.
		needsIssuesWriteForIssueEvents := reactionIncludesIssues && (hasIssuesEvent || hasIssueCommentEvent)
		needsIssuesWriteForPullRequestEvents := reactionIncludesPullRequests && hasPullRequestEvent
		needsIssuesWriteForReaction := needsIssuesWriteForIssueEvents || needsIssuesWriteForPullRequestEvents
		if needsIssuesWriteForReaction {
			permsMap[PermissionIssues] = PermissionWrite
		}
		// Reactions on pull requests and PR review comments require pull-requests: write.
		// issue_comment events also fire for PR comments (slash_command with events:[pull_request_comment]
		// compiles to issue_comment), so pull-requests: write is also needed when issue_comment is present.
		if reactionIncludesPullRequests && (hasPullRequestEvent || hasPullRequestReviewCommentEvent || hasIssueCommentEvent) {
			permsMap[PermissionPullRequests] = PermissionWrite
		}
		// Reactions on discussions use GraphQL discussion APIs.
		if reactionIncludesDiscussions && (hasDiscussionEvent || hasDiscussionCommentEvent) {
			permsMap[PermissionDiscussions] = PermissionWrite
		}
	}

	if hasStatusComment {
		// Status comments for issue and pull request related events use issue comment endpoints.
		if (statusCommentIncludesIssues && (hasIssuesEvent || hasIssueCommentEvent)) ||
			(statusCommentIncludesPullRequests && (hasPullRequestEvent || hasPullRequestReviewCommentEvent)) {
			permsMap[PermissionIssues] = PermissionWrite
		}
		// Status comments for discussions use discussion comment APIs and can be disabled via frontmatter.
		if statusCommentIncludesDiscussions && (hasDiscussionEvent || hasDiscussionCommentEvent) {
			permsMap[PermissionDiscussions] = PermissionWrite
		}
	}
}

func addBroadActivationInteractionPermissions(
	permsMap map[PermissionScope]PermissionLevel,
	hasReaction bool,
	reactionIncludesIssues bool,
	reactionIncludesPullRequests bool,
	reactionIncludesDiscussions bool,
	hasStatusComment bool,
	statusCommentIncludesIssues bool,
	statusCommentIncludesPullRequests bool,
	statusCommentIncludesDiscussions bool,
) {
	if !hasReaction && !hasStatusComment {
		return
	}

	needsIssuesWriteForReaction := hasReaction && (reactionIncludesIssues || reactionIncludesPullRequests)
	needsIssuesWriteForStatusComment := statusCommentIncludesIssues || statusCommentIncludesPullRequests
	if needsIssuesWriteForReaction || needsIssuesWriteForStatusComment {
		permsMap[PermissionIssues] = PermissionWrite
	}
	if hasReaction && reactionIncludesPullRequests {
		permsMap[PermissionPullRequests] = PermissionWrite
	}
	if (hasReaction && reactionIncludesDiscussions) || statusCommentIncludesDiscussions {
		permsMap[PermissionDiscussions] = PermissionWrite
	}
}

func shouldIncludeIssueReactions(data *WorkflowData) bool {
	if data == nil || data.ReactionIssues == nil {
		return true
	}
	return *data.ReactionIssues
}

func shouldIncludePullRequestReactions(data *WorkflowData) bool {
	if data == nil || data.ReactionPullRequests == nil {
		return true
	}
	return *data.ReactionPullRequests
}

func shouldIncludeDiscussionReactions(data *WorkflowData) bool {
	if data == nil || data.ReactionDiscussions == nil {
		return true
	}
	return *data.ReactionDiscussions
}

func shouldIncludeIssueStatusComments(data *WorkflowData) bool {
	if data == nil || data.StatusCommentIssues == nil {
		return true
	}
	return *data.StatusCommentIssues
}

func shouldIncludePullRequestStatusComments(data *WorkflowData) bool {
	if data == nil || data.StatusCommentPullRequests == nil {
		return true
	}
	return *data.StatusCommentPullRequests
}

func shouldIncludeDiscussionStatusComments(data *WorkflowData) bool {
	if data == nil || data.StatusCommentDiscussions == nil {
		return true
	}
	return *data.StatusCommentDiscussions
}

func activationEventSet(onSection string) (map[string]bool, bool) {
	events := make(map[string]bool)
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		compilerActivationJobLog.Printf("Failed to parse on section for activation permission scoping: %v", err)
		return events, false
	}

	onValue, hasOn := onData["on"]
	if !hasOn {
		compilerActivationJobLog.Print("No top-level on key found while parsing activation permission events")
		return events, false
	}

	switch v := onValue.(type) {
	case string:
		events[v] = true
	case []any:
		for _, item := range v {
			if eventName, ok := item.(string); ok {
				events[eventName] = true
			}
		}
	case map[string]any:
		for eventName := range v {
			if isActivationMetadataTriggerField(eventName) {
				continue
			}
			events[eventName] = true
		}
	default:
		compilerActivationJobLog.Printf("Unsupported on section type for activation permission scoping: %T", onValue)
		return events, false
	}

	return events, true
}

func isActivationMetadataTriggerField(eventName string) bool {
	_, isMetadataField := activationMetadataTriggerFields[eventName]
	return isMetadataField
}

// generatePromptInActivationJob generates the prompt creation steps and adds them to the activation job
// This creates the prompt.txt file that will be uploaded as an artifact and downloaded by the agent job
// beforeActivationJobs is the list of custom job names that run before (i.e., are dependencies of) activation.
// Passing nil or an empty slice means no custom jobs run before activation; expressions referencing any
// custom job will be filtered out of the substitution step to avoid actionlint errors.
func (c *Compiler) generatePromptInActivationJob(steps *[]string, data *WorkflowData, preActivationJobCreated bool, beforeActivationJobs []string) {
	compilerActivationJobLog.Print("Generating prompt steps in activation job")

	// Use a string builder to collect the YAML
	var yaml strings.Builder

	// Call the existing generatePrompt method to get all the prompt steps
	c.generatePrompt(&yaml, data, preActivationJobCreated, beforeActivationJobs)

	// Append the generated YAML content as a single string to steps
	yamlContent := yaml.String()
	*steps = append(*steps, yamlContent)

	compilerActivationJobLog.Print("Prompt generation steps added to activation job")
}

// generateResolveHostRepoStep generates a step that resolves the platform (host) repository
// for the activation job checkout using the job.workflow_* context fields.
//
// job.workflow_repository provides the owner/repo of the currently executing workflow file,
// correctly identifying the platform repo in all relay patterns (cross-repo workflow_call,
// event-driven relays like on: issue_comment, on: push, and cross-org scenarios).
//
// job.workflow_sha provides the immutable commit SHA of the workflow being executed, ensuring
// the activation checkout pins to the exact revision rather than a moving branch/tag ref.
func (c *Compiler) generateResolveHostRepoStep(data *WorkflowData) string {
	var step strings.Builder
	step.WriteString("      - name: Resolve host repo for activation checkout\n")
	step.WriteString("        id: resolve-host-repo\n")
	step.WriteString(fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
	step.WriteString("        env:\n")
	step.WriteString("          JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}\n")
	step.WriteString("          JOB_WORKFLOW_SHA: ${{ job.workflow_sha }}\n")
	step.WriteString("          JOB_WORKFLOW_REF: ${{ job.workflow_ref }}\n")
	step.WriteString("          JOB_WORKFLOW_FILE_PATH: ${{ job.workflow_file_path }}\n")
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	step.WriteString(generateGitHubScriptWithRequire("resolve_host_repo.cjs"))
	return step.String()
}

// generateCheckoutGitHubFolderForActivation generates the checkout step for .github and .agents folders
// specifically for the activation job. Unlike generateCheckoutGitHubFolder, this method doesn't skip
// the checkout when the agent job will have a full repository checkout, because the activation job
// runs before the agent job and needs independent access to workflow files for runtime imports during
// prompt generation.
func (c *Compiler) generateCheckoutGitHubFolderForActivation(data *WorkflowData) []string {
	// Check if action-tag is specified - if so, skip checkout
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				// action-tag is set, no checkout needed
				compilerActivationJobLog.Print("Skipping .github checkout in activation: action-tag specified")
				return nil
			}
		}
	}

	// Note: We don't check data.Permissions for contents read access here because
	// the activation job ALWAYS gets contents:read added to its permissions (see buildActivationJob
	// around line 720). The workflow's original permissions may not include contents:read,
	// but the activation job will always have it for GitHub API access and runtime imports.
	// The agent job uses only the user-specified permissions (no automatic contents:read augmentation).

	// For workflow_call triggers, checkout the callee (platform) repository using the target_repo
	// and target_ref outputs from the resolve-host-repo step. That step uses job.workflow_repository
	// and job.workflow_sha to identify the platform repo and pin to the exact commit,
	// correctly handling all relay patterns including cross-repo and cross-org scenarios.
	//
	// Skip when inlined-imports is enabled: content is embedded at compile time and no
	// runtime-import macros are used, so the callee's .md files are not needed at runtime.
	// In dev mode, actions/setup is referenced via a local workspace path (./actions/setup),
	// so it must be included in the sparse-checkout to preserve it for the post step.
	// In release/script/action modes, the action is in the runner cache and not the workspace.
	var extraPaths []string
	if c.actionMode.IsDev() {
		compilerActivationJobLog.Print("Dev mode: adding actions/setup to sparse-checkout to preserve local action post step")
		extraPaths = append(extraPaths, "actions/setup")
	}

	// Add engine-specific agent config directories to the sparse checkout.
	// .github and .agents are already included in GenerateGitHubFolderCheckoutStep's hardcoded list.
	// Root instruction files (AGENTS.md, CLAUDE.md, GEMINI.md) are excluded — they are not needed
	// during activation and are omitted to keep the shallow checkout minimal.
	defaultSparseCheckoutDirs := map[string]bool{".github": true, ".agents": true}
	registry := GetGlobalEngineRegistry()
	for _, folder := range registry.GetAllAgentManifestFolders() {
		if !defaultSparseCheckoutDirs[folder] {
			extraPaths = append(extraPaths, folder)
		}
	}
	compilerActivationJobLog.Printf("Adding %d engine-specific dirs to sparse-checkout: %v", len(extraPaths), extraPaths)

	cm := NewCheckoutManager(nil)
	activationToken := c.resolveActivationToken(data)
	if data != nil && hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding cross-repo-aware .github checkout for workflow_call trigger")
		cm.SetCrossRepoTargetRepo("${{ steps.resolve-host-repo.outputs.target_repo }}")
		cm.SetCrossRepoTargetRef("${{ steps.resolve-host-repo.outputs.target_ref }}")
		checkoutSteps := cm.GenerateGitHubFolderCheckoutStep(
			cm.GetCrossRepoTargetRepo(),
			cm.GetCrossRepoTargetRef(),
			activationToken,
			getActionPin,
			extraPaths...,
		)
		// When no custom token is configured, GITHUB_TOKEN is scoped to the calling
		// repository and cannot read a private callee repository in cross-repo invocations
		// (e.g. nbcnews/tvOS-App calling nbcnews/.github). Add an if: condition so the
		// checkout is only attempted for same-repo invocations where GITHUB_TOKEN works.
		// For cross-repo scenarios, users can enable the checkout by configuring
		// activation-github-token or activation-github-app in the workflow frontmatter.
		if activationToken == "${{ secrets.GITHUB_TOKEN }}" {
			compilerActivationJobLog.Print("No custom activation token — restricting cross-repo checkout to same-repo invocations")
			checkoutSteps = addSameRepoIfConditionToSteps(checkoutSteps)
		}
		return checkoutSteps
	}

	// For activation job, sparse checkout .github, .agents, and engine-specific config directories
	// (plus actions/setup in dev mode). Root instruction files are excluded as they are not needed
	// during activation. sparse-checkout-cone-mode: true ensures subdirectories are recursively included.
	compilerActivationJobLog.Print("Adding .github, .agents, and engine-specific dirs to sparse checkout for activation job")
	return cm.GenerateGitHubFolderCheckoutStep("", "", activationToken, getActionPin, extraPaths...)
}

// addSameRepoIfConditionToSteps injects an if: condition into each step that restricts
// execution to same-repo workflow_call invocations. This prevents checkout steps from
// failing when GITHUB_TOKEN cannot read a private callee repository in cross-repo scenarios.
func addSameRepoIfConditionToSteps(steps []string) []string {
	const sameRepoCondition = "steps.resolve-host-repo.outputs.target_repo == github.repository"
	result := make([]string, len(steps))
	for i, step := range steps {
		result[i] = injectIfConditionAfterName(step, sameRepoCondition)
	}
	return result
}

// injectIfConditionAfterName inserts an "if:" field immediately after the "- name:"
// line of a YAML step string. The field indentation is derived from the step's existing
// content so this remains stable if the step formatter changes indentation.
// Returns the step unchanged if a "- name:" line cannot be found, and is idempotent
// (does nothing if an "if:" field is already present).
func injectIfConditionAfterName(step, condition string) string {
	lines := strings.Split(step, "\n")

	// Find the "- name:" line
	nameLineIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- name:") {
			nameLineIdx = i
			break
		}
	}
	if nameLineIdx < 0 {
		compilerActivationJobLog.Printf("Warning: could not inject if-condition %q — step has no '- name:' line: %q", condition, step)
		return step
	}

	// Idempotency: don't inject if an "if:" field is already present
	for i := nameLineIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "if:") {
			return step
		}
	}

	// Derive the field indentation from the first non-empty line after "- name:"
	fieldIndent := ""
	for i := nameLineIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		fieldIndent = line[:len(line)-len(strings.TrimLeft(line, " "))]
		break
	}
	if fieldIndent == "" {
		// Fall back: indent = name-line indent + 2 spaces
		nameLine := lines[nameLineIdx]
		nameIndent := nameLine[:len(nameLine)-len(strings.TrimLeft(nameLine, " "))]
		fieldIndent = nameIndent + "  "
	}

	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:nameLineIdx+1]...)
	newLines = append(newLines, fieldIndent+"if: "+condition)
	newLines = append(newLines, lines[nameLineIdx+1:]...)
	return strings.Join(newLines, "\n")
}
