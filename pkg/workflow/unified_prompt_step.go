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
	var sections []PromptSection

	sections = appendBasePromptSections(sections, data)
	sections = c.appendOptionalContextPromptSections(sections, data)
	sections = appendSafeOutputPromptSections(sections, data)
	sections = appendGitHubContextPromptSection(sections, data)
	sections = appendGitHubToolGuidanceSection(sections, data)
	return c.appendPRContextPromptSections(sections, data)
}

func appendBasePromptSections(sections []PromptSection, data *WorkflowData) []PromptSection {
	if !isFeatureEnabled(constants.DisableXPIAPromptFeatureFlag, data) {
		unifiedPromptLog.Print("Adding XPIA section")
		sections = append(sections, PromptSection{
			Content: xpiaPromptFile,
			IsFile:  true,
		})
	} else {
		unifiedPromptLog.Print("XPIA section disabled by feature flag")
	}

	unifiedPromptLog.Print("Adding temp folder section")
	sections = append(sections, PromptSection{
		Content: tempFolderPromptFile,
		IsFile:  true,
	})

	unifiedPromptLog.Print("Adding markdown section")
	sections = append(sections, PromptSection{
		Content: markdownPromptFile,
		IsFile:  true,
	})

	if hasPlaywrightTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding playwright section")
		sections = append(sections, PromptSection{
			Content: playwrightPromptFile,
			IsFile:  true,
		})
	}
	return sections
}

func (c *Compiler) appendOptionalContextPromptSections(sections []PromptSection, data *WorkflowData) []PromptSection {
	if c.trialMode {
		unifiedPromptLog.Print("Adding trial mode section")
		trialContent := fmt.Sprintf("## Note\nThis workflow is running in directory $GITHUB_WORKSPACE, but that directory actually contains the contents of the repository '%s'.", c.trialLogicalRepoSlug)
		sections = append(sections, PromptSection{
			Content: trialContent,
			IsFile:  false,
		})
	}

	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		unifiedPromptLog.Printf("Adding cache memory section: caches=%d", len(data.CacheMemoryConfig.Caches))
		section := buildCacheMemoryPromptSection(data.CacheMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		unifiedPromptLog.Printf("Adding repo memory section: memories=%d", len(data.RepoMemoryConfig.Memories))
		section := buildRepoMemoryPromptSection(data.RepoMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}
	if section := buildMCPCLIPromptSection(data); section != nil {
		unifiedPromptLog.Printf("Adding MCP CLI tools section: servers=%v", getMCPCLIServerNames(data))
		sections = append(sections, *section)
	}
	return sections
}

func appendSafeOutputPromptSections(sections []PromptSection, data *WorkflowData) []PromptSection {
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		unifiedPromptLog.Print("Adding safe outputs section")
		sections = append(sections, PromptSection{
			Content: safeOutputsPromptFile,
			IsFile:  true,
		})
		sections = append(sections, buildSafeOutputsSections(data.SafeOutputs)...)
	}
	return sections
}

func appendGitHubContextPromptSection(sections []PromptSection, data *WorkflowData) []PromptSection {
	if !hasGitHubTool(data.ParsedTools) {
		return sections
	}
	unifiedPromptLog.Print("Adding GitHub context section")

	combinedPromptText := githubContextPromptText
	if checkoutsContent := buildCheckoutsPromptContent(data.CheckoutConfigs); checkoutsContent != "" {
		unifiedPromptLog.Printf("Injecting checkout list into GitHub context (%d checkouts)", len(data.CheckoutConfigs))
		combinedPromptText = insertBeforeClosingTag(combinedPromptText, "</github-context>", checkoutsContent)
	}

	extractor := NewExpressionExtractor()
	expressionMappings, err := extractor.ExtractExpressions(combinedPromptText)
	if err == nil && len(expressionMappings) > 0 {
		sections = append(sections, buildPromptSectionWithExpressionEnv(extractor, combinedPromptText, expressionMappings))
	}
	return sections
}

func insertBeforeClosingTag(content, closeTag, inserted string) string {
	if idx := strings.LastIndex(content, closeTag); idx >= 0 {
		return content[:idx] + inserted + content[idx:]
	}
	return content + "\n" + inserted
}

func buildPromptSectionWithExpressionEnv(extractor *ExpressionExtractor, content string, mappings []*ExpressionMapping) PromptSection {
	envVars := make(map[string]string)
	for _, mapping := range mappings {
		envVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
	}
	return PromptSection{
		Content: extractor.ReplaceExpressionsWithEnvVars(content),
		IsFile:  false,
		EnvVars: envVars,
	}
}

func appendGitHubToolGuidanceSection(sections []PromptSection, data *WorkflowData) []PromptSection {
	if isGitHubCLIModeEnabled(data) {
		unifiedPromptLog.Print("Adding cli-proxy tool-use guidance (gh CLI for reads, no GitHub MCP server)")
		cliProxyFile := cliProxyPromptFile
		if HasSafeOutputsEnabled(data.SafeOutputs) {
			cliProxyFile = cliProxyWithSafeOutputsPromptFile
		}
		sections = append(sections, PromptSection{
			Content: cliProxyFile,
			IsFile:  true,
		})
	} else if hasGitHubTool(data.ParsedTools) {
		// GitHub MCP tool-use guidance: clarifies that the MCP server is read-only and
		// directs the model to use it for GitHub reads. When safe-outputs is also enabled,
		// the guidance explicitly separates reads (GitHub MCP) from writes (safeoutputs) so
		// the model is never steered away from the available read tools.
		unifiedPromptLog.Print("Adding GitHub MCP tool-use guidance")
		githubMCPFile := githubMCPToolsPromptFile
		if HasSafeOutputsEnabled(data.SafeOutputs) {
			githubMCPFile = githubMCPToolsWithSafeOutputsPromptFile
		}
		sections = append(sections, PromptSection{
			Content: githubMCPFile,
			IsFile:  true,
		})
	}
	return sections
}

func (c *Compiler) appendPRContextPromptSections(sections []PromptSection, data *WorkflowData) []PromptSection {
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)
	needsCheckout := c.shouldAddCheckoutStep(data)
	var hasContentsRead bool
	if data.CachedPermissions != nil {
		hasContentsRead = data.CachedPermissions.HasContentsReadAccess()
	} else {
		hasContentsRead = NewPermissionsParser(data.Permissions).HasContentsReadAccess()
	}

	if hasCommentTriggers && needsCheckout && hasContentsRead {
		unifiedPromptLog.Print("Adding PR context section with condition")
		// Use shell condition for PR comment detection
		// This checks for issue_comment, pull_request_review_comment, or pull_request_review events
		// For issue_comment, we also need to check if it's on a PR (github.event.issue.pull_request != null)
		// However, for simplicity in the unified step, we'll add an environment variable to check this
		shellCondition := `[ "$GITHUB_EVENT_NAME" = "issue_comment" ] && [ -n "$GH_AW_IS_PR_COMMENT" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review_comment" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review" ]`

		// Add environment variable to check if issue_comment is on a PR
		envVars := map[string]string{
			"GH_AW_IS_PR_COMMENT": "${{ github.event.issue.pull_request && 'true' || '' }}",
		}

		sections = append(sections, PromptSection{
			Content:        prContextPromptFile,
			IsFile:         true,
			ShellCondition: shellCondition,
			EnvVars:        envVars,
		})

		// When push_to_pull_request_branch is configured, add guidance to prefer it over
		// create_pull_request when the workflow was triggered by a PR comment.
		if data.SafeOutputs != nil && data.SafeOutputs.PushToPullRequestBranch != nil {
			unifiedPromptLog.Print("Adding push-to-PR-branch tool preference guidance for PR comment context")
			sections = append(sections, PromptSection{
				Content:        prContextPushToPRBranchGuidanceFile,
				IsFile:         true,
				ShellCondition: shellCondition,
				EnvVars:        envVars,
			})
		}
	}

	return sections
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

	delimiter := unifiedPromptDelimiter(builtinSections, userPromptChunks)
	allEnvVars, allExpressionMappings := collectUnifiedPromptEnvAndMappings(builtinSections, expressionMappings)

	writeUnifiedPromptStepHeader(yaml, allEnvVars, data)
	inHeredoc := writeBuiltinPromptSections(yaml, builtinSections, delimiter)
	inHeredoc = closeBuiltinSystemTag(yaml, delimiter, builtinSections, inHeredoc)
	inHeredoc = writeUserPromptChunks(yaml, userPromptChunks, delimiter, inHeredoc)
	closeUnifiedPromptHeredoc(yaml, delimiter, inHeredoc)

	unifiedPromptLog.Print("Unified prompt creation step generated successfully")
	return allExpressionMappings
}

func unifiedPromptDelimiter(builtinSections []PromptSection, userPromptChunks []string) string {
	var promptContentForHash strings.Builder
	for _, section := range builtinSections {
		promptContentForHash.WriteString(section.Content)
	}
	for _, chunk := range userPromptChunks {
		promptContentForHash.WriteString(chunk)
	}
	return GenerateHeredocDelimiterFromContent("PROMPT", promptContentForHash.String())
}

func collectUnifiedPromptEnvAndMappings(
	builtinSections []PromptSection,
	expressionMappings []*ExpressionMapping,
) (map[string]string, []*ExpressionMapping) {
	allEnvVars := make(map[string]string)
	expressionMappingsMap := make(map[string]*ExpressionMapping)
	for _, section := range builtinSections {
		addPromptSectionEnvMappings(section, allEnvVars, expressionMappingsMap)
	}
	for _, mapping := range expressionMappings {
		allEnvVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
		expressionMappingsMap[mapping.EnvVar] = mapping
	}
	return allEnvVars, sortedExpressionMappings(expressionMappingsMap)
}

func addPromptSectionEnvMappings(section PromptSection, allEnvVars map[string]string, mappings map[string]*ExpressionMapping) {
	for key, value := range section.EnvVars {
		if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
			allEnvVars[key] = value
			if _, exists := mappings[key]; !exists {
				mappings[key] = &ExpressionMapping{EnvVar: key, Content: strings.TrimSpace(value[4 : len(value)-3])}
			}
			continue
		}
		if _, exists := mappings[key]; !exists {
			mappings[key] = &ExpressionMapping{EnvVar: key, Content: fmt.Sprintf("'%s'", value)}
		}
	}
}

func sortedExpressionMappings(mappings map[string]*ExpressionMapping) []*ExpressionMapping {
	result := make([]*ExpressionMapping, 0, len(mappings))
	for _, key := range sliceutil.SortedKeys(mappings) {
		result = append(result, mappings[key])
	}
	return result
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

func writeBuiltinPromptSections(yaml *strings.Builder, builtinSections []PromptSection, delimiter string) bool {
	inHeredoc := false
	systemTagPending := len(builtinSections) > 0
	for i, section := range builtinSections {
		unifiedPromptLog.Printf("Writing built-in section %d/%d: hasCondition=%v, isFile=%v",
			i+1, len(builtinSections), section.ShellCondition != "", section.IsFile)
		inHeredoc, systemTagPending = writeBuiltinPromptSection(yaml, section, delimiter, inHeredoc, systemTagPending)
	}
	return inHeredoc
}

func writeBuiltinPromptSection(yaml *strings.Builder, section PromptSection, delimiter string, inHeredoc, systemTagPending bool) (bool, bool) {
	if section.ShellCondition != "" {
		return writeConditionalPromptSection(yaml, section, delimiter, inHeredoc, systemTagPending)
	}
	return writeUnconditionalPromptSection(yaml, section, delimiter, inHeredoc, systemTagPending)
}

func writeConditionalPromptSection(yaml *strings.Builder, section PromptSection, delimiter string, inHeredoc, systemTagPending bool) (bool, bool) {
	if inHeredoc {
		yaml.WriteString("          " + delimiter + "\n")
		inHeredoc = false
	}
	if systemTagPending {
		writeSystemTagBlock(yaml, delimiter, "          ")
		systemTagPending = false
	}
	fmt.Fprintf(yaml, "          if %s; then\n", section.ShellCondition)
	if section.IsFile {
		yaml.WriteString("            " + fmt.Sprintf("cat \"%s/%s\"\n", promptsDir, section.Content))
	} else {
		yaml.WriteString("            cat << '" + delimiter + "'\n")
		writeInlinePromptContent(yaml, section.Content, "            ")
		yaml.WriteString("            " + delimiter + "\n")
	}
	yaml.WriteString("          fi\n")
	return inHeredoc, systemTagPending
}

func writeUnconditionalPromptSection(yaml *strings.Builder, section PromptSection, delimiter string, inHeredoc, systemTagPending bool) (bool, bool) {
	if section.IsFile {
		if inHeredoc {
			yaml.WriteString("          " + delimiter + "\n")
			inHeredoc = false
		}
		if systemTagPending {
			writeSystemTagBlock(yaml, delimiter, "          ")
			systemTagPending = false
		}
		yaml.WriteString("          " + fmt.Sprintf("cat \"%s/%s\"\n", promptsDir, section.Content))
		return inHeredoc, systemTagPending
	}
	if !inHeredoc {
		yaml.WriteString("          cat << '" + delimiter + "'\n")
		inHeredoc = true
		if systemTagPending {
			yaml.WriteString("          <system>\n")
			systemTagPending = false
		}
	}
	writeInlinePromptContent(yaml, section.Content, "          ")
	return inHeredoc, systemTagPending
}

func writeSystemTagBlock(yaml *strings.Builder, delimiter, indent string) {
	yaml.WriteString(indent + "cat << '" + delimiter + "'\n")
	yaml.WriteString(indent + "<system>\n")
	yaml.WriteString(indent + delimiter + "\n")
}

func writeInlinePromptContent(yaml *strings.Builder, content, indent string) {
	normalizedContent := stringutil.NormalizeLeadingWhitespace(content)
	cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
	for line := range strings.SplitSeq(cleanedContent, "\n") {
		yaml.WriteString(indent + line + "\n")
	}
}

func closeBuiltinSystemTag(yaml *strings.Builder, delimiter string, builtinSections []PromptSection, inHeredoc bool) bool {
	if len(builtinSections) == 0 {
		return inHeredoc
	}
	if inHeredoc {
		yaml.WriteString("          </system>\n")
		return inHeredoc
	}
	yaml.WriteString("          cat << '" + delimiter + "'\n")
	yaml.WriteString("          </system>\n")
	return true
}

func writeUserPromptChunks(yaml *strings.Builder, userPromptChunks []string, delimiter string, inHeredoc bool) bool {
	userBlankRun := 0
	for chunkIdx, chunk := range userPromptChunks {
		unifiedPromptLog.Printf("Writing user prompt chunk %d/%d", chunkIdx+1, len(userPromptChunks))
		inHeredoc, userBlankRun = writeUserPromptChunk(yaml, chunk, delimiter, inHeredoc, userBlankRun)
	}
	return inHeredoc
}

func writeUserPromptChunk(yaml *strings.Builder, chunk, delimiter string, inHeredoc bool, userBlankRun int) (bool, int) {
	if strings.HasPrefix(chunk, "{{#runtime-import ") && strings.HasSuffix(chunk, "}}") {
		unifiedPromptLog.Print("Detected runtime-import macro, writing inline in heredoc")
		if !inHeredoc {
			yaml.WriteString("          cat << '" + delimiter + "'\n")
			inHeredoc = true
		}
		yaml.WriteString("          " + chunk + "\n")
		return inHeredoc, 0
	}
	if !inHeredoc {
		yaml.WriteString("          cat << '" + delimiter + "'\n")
		inHeredoc = true
	}
	for line := range strings.SplitSeq(chunk, "\n") {
		userBlankRun = writeUserPromptLine(yaml, line, userBlankRun)
	}
	return inHeredoc, userBlankRun
}

func writeUserPromptLine(yaml *strings.Builder, line string, userBlankRun int) int {
	trimmed := strings.TrimRight(line, " \t")
	if trimmed == "" {
		if userBlankRun >= maxConsecutiveBlankLines {
			return userBlankRun
		}
		yaml.WriteByte('\n')
		return userBlankRun + 1
	}
	yaml.WriteString("          ")
	yaml.WriteString(trimmed)
	yaml.WriteByte('\n')
	return 0
}

func closeUnifiedPromptHeredoc(yaml *strings.Builder, delimiter string, inHeredoc bool) {
	if inHeredoc {
		yaml.WriteString("          " + delimiter + "\n")
	}
	yaml.WriteString("          } > \"$GH_AW_PROMPT\"\n")
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

	tools := buildSafeOutputToolNames(safeOutputs)
	if len(tools) == 0 {
		return nil
	}

	var sections []PromptSection
	sections = append(sections, buildSafeOutputToolsOpeningSection(tools))
	sections = appendSafeOutputInstructionSections(sections, safeOutputs)
	sections = append(sections, PromptSection{Content: "</safe-output-tools>", IsFile: false})
	return sections
}

func buildSafeOutputToolNames(safeOutputs *SafeOutputsConfig) []string {
	var tools []string
	tools = appendCoreSafeOutputToolNames(tools, safeOutputs)
	tools = appendReviewSafeOutputToolNames(tools, safeOutputs)
	tools = appendProjectSafeOutputToolNames(tools, safeOutputs)
	return appendCustomSafeOutputToolNames(tools, safeOutputs)
}

func appendCoreSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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

func appendReviewSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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

func appendProjectSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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
	if safeOutputs.PushToPullRequestBranch != nil {
		tools = append(tools, toolWithMaxBudget("push_to_pull_request_branch", safeOutputs.PushToPullRequestBranch.Max))
	}
	return appendAdvancedSafeOutputToolNames(tools, safeOutputs)
}

func appendAdvancedSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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
	if safeOutputs.CreateProjects != nil {
		tools = append(tools, toolWithMaxBudget("create_project", safeOutputs.CreateProjects.Max))
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		tools = append(tools, toolWithMaxBudget("create_project_status_update", safeOutputs.CreateProjectStatusUpdates.Max))
	}
	if safeOutputs.LinkSubIssue != nil {
		tools = append(tools, toolWithMaxBudget("link_sub_issue", safeOutputs.LinkSubIssue.Max))
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
	return appendWorkflowSafeOutputToolNames(tools, safeOutputs)
}

func appendWorkflowSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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

func appendCustomSafeOutputToolNames(tools []string, safeOutputs *SafeOutputsConfig) []string {
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

func buildSafeOutputToolsOpeningSection(tools []string) PromptSection {
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

func appendSafeOutputInstructionSections(sections []PromptSection, safeOutputs *SafeOutputsConfig) []PromptSection {
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
	// Auto-injected create_issue special notice
	if safeOutputs.CreateIssues != nil && safeOutputs.AutoInjectedCreateIssue {
		sections = append(sections, PromptSection{Content: safeOutputsAutoCreateIssueFile, IsFile: true})
	}
	return sections
}
