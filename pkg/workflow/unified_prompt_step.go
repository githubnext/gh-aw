package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

var unifiedPromptLog = logger.New("workflow:unified_prompt_step")

// PromptSection represents a section of prompt text to be appended
type PromptSection struct {
	// Content is the actual prompt text or a reference to a file
	Content string
	// IsFile indicates if Content is a filename (true) or inline text (false)
	IsFile bool
	// ShellCondition is an optional bash condition (without 'if' keyword) to wrap this section
	// Example: "${{ github.event_name == 'issue_comment' }}" becomes a shell condition
	ShellCondition string
	// EnvVars contains environment variables needed for expressions in this section
	EnvVars map[string]string
}

// removeConsecutiveEmptyLines removes consecutive empty lines, keeping only one
func removeConsecutiveEmptyLines(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	var result []string
	lastWasEmpty := false

	for _, line := range lines {
		isEmpty := strings.TrimSpace(line) == ""

		if isEmpty {
			// Only add if the last line wasn't empty
			if !lastWasEmpty {
				result = append(result, line)
				lastWasEmpty = true
			}
			// Skip consecutive empty lines
		} else {
			result = append(result, line)
			lastWasEmpty = false
		}
	}

	return strings.Join(result, "\n")
}

// collectPromptSections collects all prompt sections in the order they should be appended
func (c *Compiler) collectPromptSections(data *WorkflowData) []PromptSection {
	sections := collectBuiltinPromptSections(c, data)
	sections = append(sections, collectMemoryPromptSections(data)...)
	sections = append(sections, collectSafeOutputPromptSections(data)...)
	sections = append(sections, collectGitHubContextPromptSections(c, data)...)
	return sections
}

func collectBuiltinPromptSections(c *Compiler, data *WorkflowData) []PromptSection {
	sections := make([]PromptSection, 0, 4)
	if !isFeatureEnabled(constants.DisableXPIAPromptFeatureFlag, data) {
		unifiedPromptLog.Print("Adding XPIA section")
		sections = append(sections, PromptSection{Content: xpiaPromptFile, IsFile: true})
	} else {
		unifiedPromptLog.Print("XPIA section disabled by feature flag")
	}
	unifiedPromptLog.Print("Adding temp folder section")
	sections = append(sections, PromptSection{Content: tempFolderPromptFile, IsFile: true})
	unifiedPromptLog.Print("Adding markdown section")
	sections = append(sections, PromptSection{Content: markdownPromptFile, IsFile: true})
	if hasPlaywrightTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding playwright section")
		sections = append(sections, PromptSection{Content: playwrightPromptFile, IsFile: true})
	}
	if c.trialMode {
		unifiedPromptLog.Print("Adding trial mode section")
		sections = append(sections, PromptSection{
			Content: fmt.Sprintf("## Note\nThis workflow is running in directory $GITHUB_WORKSPACE, but that directory actually contains the contents of the repository '%s'.", c.trialLogicalRepoSlug),
			IsFile:  false,
		})
	}
	return sections
}

func collectMemoryPromptSections(data *WorkflowData) []PromptSection {
	var sections []PromptSection
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		unifiedPromptLog.Printf("Adding cache memory section: caches=%d", len(data.CacheMemoryConfig.Caches))
		if section := buildCacheMemoryPromptSection(data.CacheMemoryConfig); section != nil {
			sections = append(sections, *section)
		}
	}
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		unifiedPromptLog.Printf("Adding repo memory section: memories=%d", len(data.RepoMemoryConfig.Memories))
		if section := buildRepoMemoryPromptSection(data.RepoMemoryConfig); section != nil {
			sections = append(sections, *section)
		}
	}
	return sections
}

func collectSafeOutputPromptSections(data *WorkflowData) []PromptSection {
	var sections []PromptSection
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		unifiedPromptLog.Print("Adding safe outputs section")
		sections = append(sections, PromptSection{Content: safeOutputsPromptFile, IsFile: true})
		sections = append(sections, buildSafeOutputsSections(data.SafeOutputs)...)
	}
	if section := buildMCPCLIPromptSection(data); section != nil {
		unifiedPromptLog.Printf("Adding MCP CLI tools section: servers=%v", getMCPCLIServerNames(data))
		sections = append(sections, *section)
	}
	return sections
}

func collectGitHubContextPromptSections(c *Compiler, data *WorkflowData) []PromptSection {
	var sections []PromptSection
	if hasGitHubTool(data.ParsedTools) {
		if section := buildGitHubContextPromptSection(data); section != nil {
			sections = append(sections, *section)
		}
	}
	if section := buildGitHubToolUsePromptSection(data); section != nil {
		sections = append(sections, *section)
	}
	sections = append(sections, c.buildPRContextPromptSections(data)...)
	return sections
}

func buildGitHubContextPromptSection(data *WorkflowData) *PromptSection {
	unifiedPromptLog.Print("Adding GitHub context section")
	combinedPromptText := githubContextPromptText
	if checkoutsContent := buildCheckoutsPromptContent(data.CheckoutConfigs); checkoutsContent != "" {
		unifiedPromptLog.Printf("Injecting checkout list into GitHub context (%d checkouts)", len(data.CheckoutConfigs))
		const closeTag = "</github-context>"
		if idx := strings.LastIndex(combinedPromptText, closeTag); idx >= 0 {
			combinedPromptText = combinedPromptText[:idx] + checkoutsContent + combinedPromptText[idx:]
		} else {
			combinedPromptText += "\n" + checkoutsContent
		}
	}
	extractor := NewExpressionExtractor()
	expressionMappings, err := extractor.ExtractExpressions(combinedPromptText)
	if err != nil || len(expressionMappings) == 0 {
		return nil
	}
	envVars := make(map[string]string)
	for _, mapping := range expressionMappings {
		envVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
	}
	return &PromptSection{
		Content: extractor.ReplaceExpressionsWithEnvVars(combinedPromptText),
		IsFile:  false,
		EnvVars: envVars,
	}
}

func buildGitHubToolUsePromptSection(data *WorkflowData) *PromptSection {
	if isGitHubCLIModeEnabled(data) {
		unifiedPromptLog.Print("Adding cli-proxy tool-use guidance (gh CLI for reads, no GitHub MCP server)")
		cliProxyFile := cliProxyPromptFile
		if HasSafeOutputsEnabled(data.SafeOutputs) {
			cliProxyFile = cliProxyWithSafeOutputsPromptFile
		}
		return &PromptSection{Content: cliProxyFile, IsFile: true}
	}
	if !hasGitHubTool(data.ParsedTools) {
		return nil
	}
	unifiedPromptLog.Print("Adding GitHub MCP tool-use guidance")
	githubMCPFile := githubMCPToolsPromptFile
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		githubMCPFile = githubMCPToolsWithSafeOutputsPromptFile
	}
	return &PromptSection{Content: githubMCPFile, IsFile: true}
}

func (c *Compiler) buildPRContextPromptSections(data *WorkflowData) []PromptSection {
	if !c.hasCommentRelatedTriggers(data) || !c.shouldAddCheckoutStep(data) || !hasContentsReadAccess(data) {
		return nil
	}
	unifiedPromptLog.Print("Adding PR context section with condition")
	shellCondition := `[ "$GITHUB_EVENT_NAME" = "issue_comment" ] && [ -n "$GH_AW_IS_PR_COMMENT" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review_comment" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review" ]`
	envVars := map[string]string{"GH_AW_IS_PR_COMMENT": "${{ github.event.issue.pull_request && 'true' || '' }}"}
	sections := []PromptSection{{
		Content:        prContextPromptFile,
		IsFile:         true,
		ShellCondition: shellCondition,
		EnvVars:        envVars,
	}}
	if data.SafeOutputs != nil && data.SafeOutputs.PushToPullRequestBranch != nil {
		unifiedPromptLog.Print("Adding push-to-PR-branch tool preference guidance for PR comment context")
		sections = append(sections, PromptSection{
			Content:        prContextPushToPRBranchGuidanceFile,
			IsFile:         true,
			ShellCondition: shellCondition,
			EnvVars:        envVars,
		})
	}
	return sections
}

func hasContentsReadAccess(data *WorkflowData) bool {
	if data.CachedPermissions != nil {
		return data.CachedPermissions.HasContentsReadAccess()
	}
	return NewPermissionsParser(data.Permissions).HasContentsReadAccess()
}

// generateUnifiedPromptCreationStep generates a single workflow step (or multiple if needed) that creates
// the complete prompt file with built-in context instructions prepended to the user prompt content.
//
// This consolidates the prompt creation process:
// 1. Built-in context instructions (temp folder, playwright, safe outputs, etc.) - PREPENDED
// 2. User prompt content from markdown - APPENDED
//
// The function handles chunking for large content and ensures proper environment variable handling.
// Returns the combined expression mappings for use in the placeholder substitution step.
func (c *Compiler) generateUnifiedPromptCreationStep(yaml *strings.Builder, builtinSections []PromptSection, userPromptChunks []string, expressionMappings []*ExpressionMapping, data *WorkflowData) []*ExpressionMapping {
	unifiedPromptLog.Print("Generating unified prompt creation step")
	unifiedPromptLog.Printf("Built-in sections: %d, User prompt chunks: %d", len(builtinSections), len(userPromptChunks))

	delimiter := buildUnifiedPromptDelimiter(builtinSections, userPromptChunks)
	allEnvVars, allExpressionMappings := collectAllEnvVarsAndMappings(builtinSections, expressionMappings)
	writeUnifiedPromptStepHeader(yaml, allEnvVars, data)
	state := &promptStepWriterState{delimiter: delimiter, systemTagPending: len(builtinSections) > 0}
	writeBuiltinPromptSections(yaml, state, builtinSections)
	writeUserPromptChunks(yaml, state, userPromptChunks)
	closePromptHeredoc(yaml, state)
	yaml.WriteString("          } > \"$GH_AW_PROMPT\"\n")
	unifiedPromptLog.Print("Unified prompt creation step generated successfully")
	return allExpressionMappings
}

type promptStepWriterState struct {
	delimiter        string
	inHeredoc        bool
	systemTagPending bool
	userBlankRun     int
}

func buildUnifiedPromptDelimiter(builtinSections []PromptSection, userPromptChunks []string) string {
	var promptContentForHash strings.Builder
	for _, section := range builtinSections {
		promptContentForHash.WriteString(section.Content)
	}
	for _, chunk := range userPromptChunks {
		promptContentForHash.WriteString(chunk)
	}
	return GenerateHeredocDelimiterFromContent("PROMPT", promptContentForHash.String())
}

func collectAllEnvVarsAndMappings(builtinSections []PromptSection, expressionMappings []*ExpressionMapping) (map[string]string, []*ExpressionMapping) {
	allEnvVars := make(map[string]string)
	expressionMappingsMap := make(map[string]*ExpressionMapping)
	for _, section := range builtinSections {
		collectSectionEnvVars(section.EnvVars, allEnvVars, expressionMappingsMap)
	}
	for _, mapping := range expressionMappings {
		allEnvVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
		expressionMappingsMap[mapping.EnvVar] = mapping
	}
	return allEnvVars, sortedExpressionMappings(expressionMappingsMap)
}

func collectSectionEnvVars(sectionEnvVars, allEnvVars map[string]string, expressionMappingsMap map[string]*ExpressionMapping) {
	for key, value := range sectionEnvVars {
		if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
			allEnvVars[key] = value
			if _, exists := expressionMappingsMap[key]; !exists {
				expressionMappingsMap[key] = &ExpressionMapping{EnvVar: key, Content: strings.TrimSpace(value[4 : len(value)-3])}
			}
			continue
		}
		if _, exists := expressionMappingsMap[key]; !exists {
			expressionMappingsMap[key] = &ExpressionMapping{EnvVar: key, Content: fmt.Sprintf("'%s'", value)}
		}
	}
}

func sortedExpressionMappings(expressionMappingsMap map[string]*ExpressionMapping) []*ExpressionMapping {
	allExpressionMappings := make([]*ExpressionMapping, 0, len(expressionMappingsMap))
	for _, key := range sliceutil.SortedKeys(expressionMappingsMap) {
		allExpressionMappings = append(allExpressionMappings, expressionMappingsMap[key])
	}
	return allExpressionMappings
}

func writeUnifiedPromptStepHeader(yaml *strings.Builder, allEnvVars map[string]string, data *WorkflowData) {
	yaml.WriteString("      - name: Create prompt with built-in context\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	if data.SafeOutputs != nil {
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ runner.temp }}/gh-aw/safeoutputs/outputs.jsonl\n")
	}
	for _, key := range sliceutil.SortedKeys(allEnvVars) {
		fmt.Fprintf(yaml, "          %s: %s\n", key, allEnvVars[key])
	}
	yaml.WriteString("        # poutine:ignore untrusted_checkout_exec\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/create_prompt_first.sh\"\n")
	yaml.WriteString("          {\n")
}

func writeBuiltinPromptSections(yaml *strings.Builder, state *promptStepWriterState, builtinSections []PromptSection) {
	for i, section := range builtinSections {
		unifiedPromptLog.Printf("Writing built-in section %d/%d: hasCondition=%v, isFile=%v", i+1, len(builtinSections), section.ShellCondition != "", section.IsFile)
		if section.ShellCondition != "" {
			writeConditionalBuiltinPromptSection(yaml, state, section)
			continue
		}
		writeUnconditionalBuiltinPromptSection(yaml, state, section)
	}
	writeBuiltinPromptClosingTag(yaml, state, len(builtinSections) > 0)
}

func writeConditionalBuiltinPromptSection(yaml *strings.Builder, state *promptStepWriterState, section PromptSection) {
	closePromptHeredoc(yaml, state)
	writeSystemTagIfPending(yaml, state)
	fmt.Fprintf(yaml, "          if %s; then\n", section.ShellCondition)
	if section.IsFile {
		fmt.Fprintf(yaml, "            cat \"%s/%s\"\n", promptsDir, section.Content)
	} else {
		yaml.WriteString("            cat << '" + state.delimiter + "'\n")
		writeNormalizedLines(yaml, section.Content, "            ")
		yaml.WriteString("            " + state.delimiter + "\n")
	}
	yaml.WriteString("          fi\n")
}

func writeUnconditionalBuiltinPromptSection(yaml *strings.Builder, state *promptStepWriterState, section PromptSection) {
	if section.IsFile {
		closePromptHeredoc(yaml, state)
		writeSystemTagIfPending(yaml, state)
		fmt.Fprintf(yaml, "          cat \"%s/%s\"\n", promptsDir, section.Content)
		return
	}
	openPromptHeredoc(yaml, state)
	writeNormalizedLines(yaml, section.Content, "          ")
}

func writeBuiltinPromptClosingTag(yaml *strings.Builder, state *promptStepWriterState, hasBuiltinSections bool) {
	if !hasBuiltinSections {
		return
	}
	if state.inHeredoc {
		yaml.WriteString("          </system>\n")
		return
	}
	yaml.WriteString("          cat << '" + state.delimiter + "'\n")
	yaml.WriteString("          </system>\n")
	state.inHeredoc = true
}

func writeUserPromptChunks(yaml *strings.Builder, state *promptStepWriterState, userPromptChunks []string) {
	for chunkIdx, chunk := range userPromptChunks {
		unifiedPromptLog.Printf("Writing user prompt chunk %d/%d", chunkIdx+1, len(userPromptChunks))
		if strings.HasPrefix(chunk, "{{#runtime-import ") && strings.HasSuffix(chunk, "}}") {
			unifiedPromptLog.Print("Detected runtime-import macro, writing inline in heredoc")
			openPromptHeredoc(yaml, state)
			yaml.WriteString("          " + chunk + "\n")
			state.userBlankRun = 0
			continue
		}
		openPromptHeredoc(yaml, state)
		writeUserPromptChunkLines(yaml, state, chunk)
	}
}

func writeUserPromptChunkLines(yaml *strings.Builder, state *promptStepWriterState, chunk string) {
	lines := strings.SplitSeq(chunk, "\n")
	for line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			if state.userBlankRun >= maxConsecutiveBlankLines {
				continue
			}
			state.userBlankRun++
			yaml.WriteByte('\n')
			continue
		}
		state.userBlankRun = 0
		yaml.WriteString("          ")
		yaml.WriteString(trimmed)
		yaml.WriteByte('\n')
	}
}

func openPromptHeredoc(yaml *strings.Builder, state *promptStepWriterState) {
	if state.inHeredoc {
		return
	}
	yaml.WriteString("          cat << '" + state.delimiter + "'\n")
	state.inHeredoc = true
	if state.systemTagPending {
		yaml.WriteString("          <system>\n")
		state.systemTagPending = false
	}
}

func closePromptHeredoc(yaml *strings.Builder, state *promptStepWriterState) {
	if !state.inHeredoc {
		return
	}
	yaml.WriteString("          " + state.delimiter + "\n")
	state.inHeredoc = false
}

func writeSystemTagIfPending(yaml *strings.Builder, state *promptStepWriterState) {
	if !state.systemTagPending {
		return
	}
	yaml.WriteString("          cat << '" + state.delimiter + "'\n")
	yaml.WriteString("          <system>\n")
	yaml.WriteString("          " + state.delimiter + "\n")
	state.systemTagPending = false
}

func writeNormalizedLines(yaml *strings.Builder, content, prefix string) {
	cleanedContent := removeConsecutiveEmptyLines(stringutil.NormalizeLeadingWhitespace(content))
	for line := range strings.SplitSeq(cleanedContent, "\n") {
		yaml.WriteString(prefix + line + "\n")
	}
}

var safeOutputsPromptLog = logger.New("workflow:safe_outputs_prompt")

// toolWithMaxBudget formats a tool name with a per-call budget annotation when the
// configured maximum is greater than 1. This helps agents understand that multiple
// calls to the same tool are allowed for the current workflow.
//
// Returns "toolname" when max is nil or "1" (default single-call behavior).
// Returns "toolname(max:N)" when max > 1 so agents know the real action budget.
func toolWithMaxBudget(name string, max *string) string {
	if max == nil || *max == "1" {
		return name
	}
	return fmt.Sprintf("%s(max:%s)", name, *max)
}

// buildSafeOutputsSections returns the PromptSections that form the <safe-output-tools> block.
// The block contains:
//  1. An inline opening tag with a compact Tools list (dynamic, depends on which tools are enabled).
//     Any ${{ }} expressions in max: values are extracted to GH_AW_* env vars and replaced
//     with __GH_AW_*__ placeholders so they do not appear in the run: heredoc, avoiding the
//     GitHub Actions 21KB expression-size limit.
//  2. File references for tools that require multi-step instructions (create_pull_request,
//     push_to_pull_request_branch, auto-injected create_issue notice).
//  3. An inline closing tag.
//
// The static intro (gh CLI warning, temporary ID rules, noop note) lives in
// actions/setup/md/safe_outputs_prompt.md and is included by the caller before these sections.
func buildSafeOutputsSections(safeOutputs *SafeOutputsConfig) []PromptSection {
	if safeOutputs == nil {
		return nil
	}

	safeOutputsPromptLog.Print("Building safe outputs sections")
	tools := buildSafeOutputToolList(safeOutputs)
	if len(tools) == 0 {
		return nil
	}

	sections := []PromptSection{buildSafeOutputToolListSection(tools)}
	sections = append(sections, buildSafeOutputInstructionSections(safeOutputs)...)
	return append(sections, PromptSection{Content: "</safe-output-tools>", IsFile: false})
}

func buildSafeOutputToolList(safeOutputs *SafeOutputsConfig) []string {
	tools := collectIssueAndDiscussionSafeOutputTools(safeOutputs)
	tools = append(tools, collectPullRequestSafeOutputTools(safeOutputs)...)
	tools = append(tools, collectRepositorySafeOutputTools(safeOutputs)...)
	return appendSafeOutputCustomTools(safeOutputs, tools)
}

func collectIssueAndDiscussionSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.AddComments != nil {
		tools = append(tools, toolWithMaxBudget("add_comment", safeOutputs.AddComments.Max))
	}
	if safeOutputs.CreateIssues != nil {
		tools = append(tools, toolWithMaxBudget("create_issue", safeOutputs.CreateIssues.Max))
	}
	if safeOutputs.CloseIssues != nil {
		tools = append(tools, toolWithMaxBudget("close_issue", safeOutputs.CloseIssues.Max))
	}
	if safeOutputs.UpdateIssues != nil {
		tools = append(tools, toolWithMaxBudget("update_issue", safeOutputs.UpdateIssues.Max))
	}
	if safeOutputs.CreateDiscussions != nil {
		tools = append(tools, toolWithMaxBudget("create_discussion", safeOutputs.CreateDiscussions.Max))
	}
	if safeOutputs.UpdateDiscussions != nil {
		tools = append(tools, toolWithMaxBudget("update_discussion", safeOutputs.UpdateDiscussions.Max))
	}
	if safeOutputs.CloseDiscussions != nil {
		tools = append(tools, toolWithMaxBudget("close_discussion", safeOutputs.CloseDiscussions.Max))
	}
	if safeOutputs.CreateAgentSessions != nil {
		tools = append(tools, toolWithMaxBudget("create_agent_session", safeOutputs.CreateAgentSessions.Max))
	}
	return tools
}

func collectPullRequestSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	tools := collectPullRequestCoreSafeOutputTools(safeOutputs)
	tools = append(tools, collectPullRequestReviewSafeOutputTools(safeOutputs)...)
	tools = append(tools, collectMetadataSafeOutputTools(safeOutputs)...)
	return append(tools, collectProjectSafeOutputTools(safeOutputs)...)
}

func collectPullRequestCoreSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.CreatePullRequests != nil {
		tools = append(tools, toolWithMaxBudget("create_pull_request", safeOutputs.CreatePullRequests.Max))
	}
	if safeOutputs.ClosePullRequests != nil {
		tools = append(tools, toolWithMaxBudget("close_pull_request", safeOutputs.ClosePullRequests.Max))
	}
	if safeOutputs.UpdatePullRequests != nil {
		tools = append(tools, toolWithMaxBudget("update_pull_request", safeOutputs.UpdatePullRequests.Max))
	}
	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		tools = append(tools, toolWithMaxBudget("mark_pull_request_as_ready_for_review", safeOutputs.MarkPullRequestAsReadyForReview.Max))
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		tools = append(tools, toolWithMaxBudget("push_to_pull_request_branch", safeOutputs.PushToPullRequestBranch.Max))
	}
	return tools
}

func collectPullRequestReviewSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.DismissPullRequestReview != nil {
		tools = append(tools, toolWithMaxBudget("dismiss_pull_request_review", safeOutputs.DismissPullRequestReview.Max))
	}
	if safeOutputs.CreatePullRequestReviewComments != nil {
		tools = append(tools, toolWithMaxBudget("create_pull_request_review_comment", safeOutputs.CreatePullRequestReviewComments.Max))
	}
	if safeOutputs.SubmitPullRequestReview != nil {
		tools = append(tools, toolWithMaxBudget("submit_pull_request_review", safeOutputs.SubmitPullRequestReview.Max))
	}
	if safeOutputs.ReplyToPullRequestReviewComment != nil {
		tools = append(tools, toolWithMaxBudget("reply_to_pull_request_review_comment", safeOutputs.ReplyToPullRequestReviewComment.Max))
	}
	if safeOutputs.ResolvePullRequestReviewThread != nil {
		tools = append(tools, toolWithMaxBudget("resolve_pull_request_review_thread", safeOutputs.ResolvePullRequestReviewThread.Max))
	}
	return tools
}

func collectMetadataSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.AddLabels != nil {
		tools = append(tools, toolWithMaxBudget("add_labels", safeOutputs.AddLabels.Max))
	}
	if safeOutputs.RemoveLabels != nil {
		tools = append(tools, toolWithMaxBudget("remove_labels", safeOutputs.RemoveLabels.Max))
	}
	if safeOutputs.ReplaceLabel != nil {
		tools = append(tools, toolWithMaxBudget("replace_label", safeOutputs.ReplaceLabel.Max))
	}
	if safeOutputs.AddReviewer != nil {
		tools = append(tools, toolWithMaxBudget("add_reviewer", safeOutputs.AddReviewer.Max))
	}
	if safeOutputs.AssignMilestone != nil {
		tools = append(tools, toolWithMaxBudget("assign_milestone", safeOutputs.AssignMilestone.Max))
	}
	if safeOutputs.AssignToAgent != nil {
		tools = append(tools, toolWithMaxBudget("assign_to_agent", safeOutputs.AssignToAgent.Max))
	}
	if safeOutputs.AssignToUser != nil {
		tools = append(tools, toolWithMaxBudget("assign_to_user", safeOutputs.AssignToUser.Max))
	}
	if safeOutputs.UnassignFromUser != nil {
		tools = append(tools, toolWithMaxBudget("unassign_from_user", safeOutputs.UnassignFromUser.Max))
	}
	if safeOutputs.CreateCodeScanningAlerts != nil {
		tools = append(tools, toolWithMaxBudget("create_code_scanning_alert", safeOutputs.CreateCodeScanningAlerts.Max))
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		tools = append(tools, toolWithMaxBudget("autofix_code_scanning_alert", safeOutputs.AutofixCodeScanningAlert.Max))
	}
	if safeOutputs.CreateCheckRun != nil {
		tools = append(tools, toolWithMaxBudget("create_check_run", safeOutputs.CreateCheckRun.Max))
	}
	if safeOutputs.UploadAssets != nil {
		tools = append(tools, toolWithMaxBudget("upload_asset", safeOutputs.UploadAssets.Max))
	}
	if safeOutputs.UpdateRelease != nil {
		tools = append(tools, toolWithMaxBudget("update_release", safeOutputs.UpdateRelease.Max))
	}
	if safeOutputs.UpdateProjects != nil {
		tools = append(tools, toolWithMaxBudget("update_project", safeOutputs.UpdateProjects.Max))
	}
	if safeOutputs.HideComment != nil {
		tools = append(tools, toolWithMaxBudget("hide_comment", safeOutputs.HideComment.Max))
	}
	if safeOutputs.SetIssueType != nil {
		tools = append(tools, toolWithMaxBudget("set_issue_type", safeOutputs.SetIssueType.Max))
	}
	if safeOutputs.SetIssueField != nil {
		tools = append(tools, toolWithMaxBudget("set_issue_field", safeOutputs.SetIssueField.Max))
	}
	return tools
}

func collectProjectSafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.CreateProjects != nil {
		tools = append(tools, toolWithMaxBudget("create_project", safeOutputs.CreateProjects.Max))
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		tools = append(tools, toolWithMaxBudget("create_project_status_update", safeOutputs.CreateProjectStatusUpdates.Max))
	}
	if safeOutputs.LinkSubIssue != nil {
		tools = append(tools, toolWithMaxBudget("link_sub_issue", safeOutputs.LinkSubIssue.Max))
	}
	return tools
}

func collectRepositorySafeOutputTools(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	if safeOutputs.DispatchWorkflow != nil {
		tools = append(tools, toolWithMaxBudget("dispatch_workflow", safeOutputs.DispatchWorkflow.Max))
	}
	if safeOutputs.DispatchRepository != nil {
		tools = append(tools, "dispatch_repository")
	}
	if safeOutputs.CallWorkflow != nil {
		tools = append(tools, toolWithMaxBudget("call_workflow", safeOutputs.CallWorkflow.Max))
	}
	if safeOutputs.MissingTool != nil {
		tools = append(tools, toolWithMaxBudget("missing_tool", safeOutputs.MissingTool.Max))
	}
	if safeOutputs.MissingData != nil {
		tools = append(tools, toolWithMaxBudget("missing_data", safeOutputs.MissingData.Max))
	}
	if safeOutputs.NoOp != nil {
		tools = append(tools, toolWithMaxBudget("noop", safeOutputs.NoOp.Max))
	}
	return tools
}

func appendSafeOutputCustomTools(safeOutputs *SafeOutputsConfig, tools []string) []string {
	for _, jobName := range sliceutil.SortedKeys(safeOutputs.Jobs) {
		tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(jobName))
	}
	for _, scriptName := range sliceutil.SortedKeys(safeOutputs.Scripts) {
		tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(scriptName))
	}
	for _, actionName := range sliceutil.SortedKeys(safeOutputs.Actions) {
		tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(actionName))
	}
	return tools
}

func buildSafeOutputToolListSection(tools []string) PromptSection {
	toolsContent := "<safe-output-tools>\nTools: " + strings.Join(tools, ", ")
	envVars := make(map[string]string)
	extractor := NewExpressionExtractor()
	exprMappings, err := extractor.ExtractExpressions(toolsContent)
	if err == nil && len(exprMappings) > 0 {
		safeOutputsPromptLog.Printf("Extracted %d expression(s) from safe-output-tools block", len(exprMappings))
		toolsContent = extractor.ReplaceExpressionsWithEnvVars(toolsContent)
		for _, mapping := range exprMappings {
			envVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
		}
	}
	return PromptSection{Content: toolsContent, IsFile: false, EnvVars: envVars}
}

func buildSafeOutputInstructionSections(safeOutputs *SafeOutputsConfig) []PromptSection {
	var sections []PromptSection
	if safeOutputs.CreatePullRequests != nil {
		sections = append(sections, PromptSection{Content: safeOutputsCreatePRFile, IsFile: true})
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		sections = append(sections, PromptSection{Content: safeOutputsPushToBranchFile, IsFile: true})
	}
	if safeOutputs.CommentMemory != nil {
		sections = append(sections, PromptSection{Content: safeOutputsCommentMemoryFile, IsFile: true})
	}
	if safeOutputs.UploadAssets != nil {
		sections = append(sections, PromptSection{
			Content: "\nupload_asset: provide a file path; returns a URL; assets are published after the workflow completes (" + constants.SafeOutputsMCPServerID.String() + ").",
			IsFile:  false,
		})
	}
	if safeOutputs.CreateIssues != nil && safeOutputs.AutoInjectedCreateIssue {
		sections = append(sections, PromptSection{Content: safeOutputsAutoCreateIssueFile, IsFile: true})
	}
	return sections
}
