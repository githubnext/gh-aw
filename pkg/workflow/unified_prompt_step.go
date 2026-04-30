package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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

// normalizeLeadingWhitespace removes consistent leading whitespace from all lines
// This handles content that was generated with indentation for heredocs
func normalizeLeadingWhitespace(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Find minimum leading whitespace (excluding empty lines)
	minLeadingSpaces := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
		if minLeadingSpaces == -1 || leadingSpaces < minLeadingSpaces {
			minLeadingSpaces = leadingSpaces
		}
	}

	// If no content or no leading spaces, return as-is
	if minLeadingSpaces <= 0 {
		return content
	}

	// Remove the minimum leading whitespace from all lines
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if strings.TrimSpace(line) == "" {
			// Keep empty lines as empty
			result.WriteString("")
		} else if len(line) >= minLeadingSpaces {
			// Remove leading whitespace
			result.WriteString(line[minLeadingSpaces:])
		} else {
			result.WriteString(line)
		}
	}

	return result.String()
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

	// 0. XPia instructions (unless disabled by feature flag)
	if !isFeatureEnabled(constants.DisableXPIAPromptFeatureFlag, data) {
		unifiedPromptLog.Print("Adding XPIA section")
		sections = append(sections, PromptSection{
			Content: xpiaPromptFile,
			IsFile:  true,
		})
	} else {
		unifiedPromptLog.Print("XPIA section disabled by feature flag")
	}

	// 1. Temporary folder instructions (always included)
	unifiedPromptLog.Print("Adding temp folder section")
	sections = append(sections, PromptSection{
		Content: tempFolderPromptFile,
		IsFile:  true,
	})

	// 2. Markdown generation instructions (always included)
	unifiedPromptLog.Print("Adding markdown section")
	sections = append(sections, PromptSection{
		Content: markdownPromptFile,
		IsFile:  true,
	})

	// 3. Playwright instructions (if playwright tool is enabled)
	if hasPlaywrightTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding playwright section")
		sections = append(sections, PromptSection{
			Content: playwrightPromptFile,
			IsFile:  true,
		})
	}

	// 4. Agentic Workflows MCP guide (if agentic-workflows tool is enabled)
	if hasAgenticWorkflowsTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding agentic-workflows guide section")
		sections = append(sections, PromptSection{
			Content: agenticWorkflowsGuideFile,
			IsFile:  true,
		})
	}

	// 5. Trial mode note (if in trial mode)
	if c.trialMode {
		unifiedPromptLog.Print("Adding trial mode section")
		trialContent := fmt.Sprintf("## Note\nThis workflow is running in directory $GITHUB_WORKSPACE, but that directory actually contains the contents of the repository '%s'.", c.trialLogicalRepoSlug)
		sections = append(sections, PromptSection{
			Content: trialContent,
			IsFile:  false,
		})
	}

	// 6. Cache memory instructions (if enabled)
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		unifiedPromptLog.Printf("Adding cache memory section: caches=%d", len(data.CacheMemoryConfig.Caches))
		section := buildCacheMemoryPromptSection(data.CacheMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	// 7. Repo memory instructions (if enabled)
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		unifiedPromptLog.Printf("Adding repo memory section: memories=%d", len(data.RepoMemoryConfig.Memories))
		section := buildRepoMemoryPromptSection(data.RepoMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	// 8. Safe outputs instructions (if enabled)
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		unifiedPromptLog.Print("Adding safe outputs section")
		// Static intro from file (gh CLI warning, temporary ID rules, noop note)
		sections = append(sections, PromptSection{
			Content: safeOutputsPromptFile,
			IsFile:  true,
		})
		// Per-tool sections: opening tag + tools list (inline), tool instruction files, closing tag
		sections = append(sections, buildSafeOutputsSections(data.SafeOutputs)...)
	}

	// 8a. MCP CLI tools instructions (if any MCP servers are mounted as CLIs)
	if section := buildMCPCLIPromptSection(data); section != nil {
		unifiedPromptLog.Printf("Adding MCP CLI tools section: servers=%v", getMCPCLIServerNames(data))
		sections = append(sections, *section)
	}

	// 9. GitHub context (if GitHub tool is enabled)
	if hasGitHubTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding GitHub context section")

		// Build the combined prompt text: base github context + optional checkout list.
		// The checkout list may contain ${{ github.repository }} which must go through
		// the expression extractor so the placeholder substitution step can resolve it.
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

		// Extract expressions from the combined content (includes any new expressions
		// introduced by the checkout list, e.g. ${{ github.repository }}).
		extractor := NewExpressionExtractor()
		expressionMappings, err := extractor.ExtractExpressions(combinedPromptText)
		if err == nil && len(expressionMappings) > 0 {
			modifiedPromptText := extractor.ReplaceExpressionsWithEnvVars(combinedPromptText)

			// Build environment variables map
			envVars := make(map[string]string)
			for _, mapping := range expressionMappings {
				envVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
			}

			sections = append(sections, PromptSection{
				Content: modifiedPromptText,
				IsFile:  false,
				EnvVars: envVars,
			})
		}
	}

	// 10. GitHub tool-use guidance: directs the model to the correct mechanism for
	// GitHub reads (and writes when safe-outputs is also enabled).
	// When GitHub mode is gh-proxy, the agent uses the pre-authenticated gh CLI for reads
	// instead of a GitHub MCP server (which is not registered). Otherwise, the GitHub
	// MCP server is used for reads.
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

	// 11. PR context (if comment-related triggers and checkout is needed)
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

	// Get the heredoc delimiter for consistent usage
	delimiter := GenerateHeredocDelimiterFromSeed("PROMPT", data.FrontmatterHash)

	// Collect all environment variables from built-in sections and user prompt expressions
	allEnvVars := make(map[string]string)

	// Also collect all expression mappings for the substitution step (using a map to avoid duplicates)
	expressionMappingsMap := make(map[string]*ExpressionMapping)

	// Add environment variables and expression mappings from built-in sections
	for _, section := range builtinSections {
		for key, value := range section.EnvVars {
			// Extract the GitHub expression from the value (e.g., "${{ github.repository }}" -> "github.repository")
			// This is needed for the substitution step
			if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
				content := strings.TrimSpace(value[4 : len(value)-3])
				// Add to both allEnvVars (for prompt creation step) and expressionMappingsMap (for substitution step)
				allEnvVars[key] = value
				// Only add if not already present (user prompt expressions take precedence)
				if _, exists := expressionMappingsMap[key]; !exists {
					expressionMappingsMap[key] = &ExpressionMapping{
						EnvVar:  key,
						Content: content,
					}
				}
			} else {
				// For static values (not GitHub Actions expressions), only add to expressionMappingsMap
				// This ensures they're only available in the substitution step, not the prompt creation step
				if _, exists := expressionMappingsMap[key]; !exists {
					expressionMappingsMap[key] = &ExpressionMapping{
						EnvVar:  key,
						Content: fmt.Sprintf("'%s'", value), // Wrap in quotes for substitution
					}
				}
			}
		}
	}

	// Add environment variables from user prompt expressions (these override built-in ones)
	for _, mapping := range expressionMappings {
		allEnvVars[mapping.EnvVar] = fmt.Sprintf("${{ %s }}", mapping.Content)
		expressionMappingsMap[mapping.EnvVar] = mapping
	}

	// Convert map back to slice for the substitution step
	allExpressionMappings := make([]*ExpressionMapping, 0, len(expressionMappingsMap))

	// Sort the keys to ensure stable output
	sortedKeys := make([]string, 0, len(expressionMappingsMap))
	for key := range expressionMappingsMap {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Add mappings in sorted order
	for _, key := range sortedKeys {
		allExpressionMappings = append(allExpressionMappings, expressionMappingsMap[key])
	}

	// Generate the step with all environment variables
	yaml.WriteString("      - name: Create prompt with built-in context\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

	if data.SafeOutputs != nil {
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ runner.temp }}/gh-aw/safeoutputs/outputs.jsonl\n")
	}

	// Add all environment variables in sorted order for consistency
	var envKeys []string
	for key := range allEnvVars {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		fmt.Fprintf(yaml, "          %s: %s\n", key, allEnvVars[key])
	}

	yaml.WriteString("        # poutine:ignore untrusted_checkout_exec\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/create_prompt_first.sh\"\n")
	yaml.WriteString("          {\n")

	// Track if we're inside a heredoc
	inHeredoc := false

	// 1. Write built-in sections first (prepended), wrapped in <system> tags.
	// The <system> opening tag is deferred: it is written either as the first line
	// of the first inline section's heredoc, or in its own block just before the
	// first file or conditional section. This allows the opening tag to share a
	// heredoc block with adjacent inline content, reducing the total number of blocks.
	systemTagPending := len(builtinSections) > 0

	for i, section := range builtinSections {
		unifiedPromptLog.Printf("Writing built-in section %d/%d: hasCondition=%v, isFile=%v",
			i+1, len(builtinSections), section.ShellCondition != "", section.IsFile)

		if section.ShellCondition != "" {
			// Close heredoc if open, add conditional
			if inHeredoc {
				yaml.WriteString("          " + delimiter + "\n")
				inHeredoc = false
			}
			// Write <system> before conditional if still pending
			if systemTagPending {
				yaml.WriteString("          cat << '" + delimiter + "'\n")
				yaml.WriteString("          <system>\n")
				yaml.WriteString("          " + delimiter + "\n")
				systemTagPending = false
			}
			fmt.Fprintf(yaml, "          if %s; then\n", section.ShellCondition)

			if section.IsFile {
				// File reference inside conditional
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("            " + fmt.Sprintf("cat \"%s\"\n", promptPath))
			} else {
				// Inline content inside conditional - open heredoc, write content, close
				yaml.WriteString("            cat << '" + delimiter + "'\n")
				normalizedContent := normalizeLeadingWhitespace(section.Content)
				cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
				contentLines := strings.SplitSeq(cleanedContent, "\n")
				for line := range contentLines {
					yaml.WriteString("            " + line + "\n")
				}
				yaml.WriteString("            " + delimiter + "\n")
			}

			yaml.WriteString("          fi\n")
		} else {
			// Unconditional section
			if section.IsFile {
				// Close heredoc if open
				if inHeredoc {
					yaml.WriteString("          " + delimiter + "\n")
					inHeredoc = false
				}
				// Write <system> before file if still pending
				if systemTagPending {
					yaml.WriteString("          cat << '" + delimiter + "'\n")
					yaml.WriteString("          <system>\n")
					yaml.WriteString("          " + delimiter + "\n")
					systemTagPending = false
				}
				// Cat the file
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("          " + fmt.Sprintf("cat \"%s\"\n", promptPath))
			} else {
				// Inline content - open heredoc if not already open
				if !inHeredoc {
					yaml.WriteString("          cat << '" + delimiter + "'\n")
					inHeredoc = true
					// Write <system> as first line when opening the heredoc
					if systemTagPending {
						yaml.WriteString("          <system>\n")
						systemTagPending = false
					}
				}
				// Write content directly to open heredoc
				normalizedContent := normalizeLeadingWhitespace(section.Content)
				cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
				contentLines := strings.SplitSeq(cleanedContent, "\n")
				for line := range contentLines {
					yaml.WriteString("          " + line + "\n")
				}
			}
		}
	}

	// Close </system> tag after all built-in sections.
	// Merge with the open heredoc (if any) to minimise the total number of cat/heredoc
	// blocks, which reduces the number of lines that change in the diff when the user
	// prompt changes (each block boundary contributes two delimiter lines).
	if len(builtinSections) > 0 {
		if inHeredoc {
			// Append </system> to the still-open heredoc and keep it open for
			// the user content that follows.
			yaml.WriteString("          </system>\n")
		} else {
			// No heredoc is open: start a new one for </system> and keep it
			// open so the subsequent user content lands in the same block.
			yaml.WriteString("          cat << '" + delimiter + "'\n")
			yaml.WriteString("          </system>\n")
			inHeredoc = true
		}
	}

	// 2. Write user prompt chunks (appended after built-in sections).
	// All chunks are written into the same heredoc block (opened above or here)
	// to minimise the number of delimiter lines in the compiled lock file.
	for chunkIdx, chunk := range userPromptChunks {
		unifiedPromptLog.Printf("Writing user prompt chunk %d/%d", chunkIdx+1, len(userPromptChunks))

		// Check if this chunk is a runtime-import macro
		if strings.HasPrefix(chunk, "{{#runtime-import ") && strings.HasSuffix(chunk, "}}") {
			// Runtime-import macros are plain text lines processed by the
			// interpolate-prompt step; they can live in the same heredoc block
			// as surrounding content.
			unifiedPromptLog.Print("Detected runtime-import macro, writing inline in heredoc")

			if !inHeredoc {
				yaml.WriteString("          cat << '" + delimiter + "'\n")
				inHeredoc = true
			}
			yaml.WriteString("          " + chunk + "\n")
			continue
		}

		// Regular chunk: write to the current heredoc (or open one).
		if !inHeredoc {
			yaml.WriteString("          cat << '" + delimiter + "'\n")
			inHeredoc = true
		}

		lines := strings.SplitSeq(chunk, "\n")
		for line := range lines {
			yaml.WriteString("          ")
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}

	// Close heredoc if still open
	if inHeredoc {
		yaml.WriteString("          " + delimiter + "\n")
	}
	yaml.WriteString("          } > \"$GH_AW_PROMPT\"\n")

	unifiedPromptLog.Print("Unified prompt creation step generated successfully")

	// Return all expression mappings for use in the placeholder substitution step
	// This allows the substitution to happen AFTER runtime-import processing
	return allExpressionMappings
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

	// Build compact list of enabled tool names, annotated with max budget when > 1.
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
	if safeOutputs.AddLabels != nil {
		tools = append(tools, toolWithMaxBudget("add_labels", safeOutputs.AddLabels.Max))
	}
	if safeOutputs.RemoveLabels != nil {
		tools = append(tools, toolWithMaxBudget("remove_labels", safeOutputs.RemoveLabels.Max))
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
	if safeOutputs.CreateCodeScanningAlerts != nil {
		tools = append(tools, toolWithMaxBudget("create_code_scanning_alert", safeOutputs.CreateCodeScanningAlerts.Max))
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		tools = append(tools, toolWithMaxBudget("autofix_code_scanning_alert", safeOutputs.AutofixCodeScanningAlert.Max))
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
	if safeOutputs.DispatchWorkflow != nil {
		tools = append(tools, toolWithMaxBudget("dispatch_workflow", safeOutputs.DispatchWorkflow.Max))
	}
	if safeOutputs.DispatchRepository != nil {
		// dispatch_repository uses per-tool max values (map-of-tools pattern); no top-level max.
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
	// noop is always included: it is auto-injected by extractSafeOutputsConfig and
	// must always appear in the tools list so agents can signal no-op completion.
	if safeOutputs.NoOp != nil {
		tools = append(tools, toolWithMaxBudget("noop", safeOutputs.NoOp.Max))
	}

	// Add custom job tools from SafeOutputs.Jobs (sorted for deterministic output).
	if len(safeOutputs.Jobs) > 0 {
		jobNames := make([]string, 0, len(safeOutputs.Jobs))
		for jobName := range safeOutputs.Jobs {
			jobNames = append(jobNames, jobName)
		}
		sort.Strings(jobNames)
		for _, jobName := range jobNames {
			tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(jobName))
		}
	}

	// Add custom script tools from SafeOutputs.Scripts (sorted for deterministic output).
	if len(safeOutputs.Scripts) > 0 {
		scriptNames := make([]string, 0, len(safeOutputs.Scripts))
		for scriptName := range safeOutputs.Scripts {
			scriptNames = append(scriptNames, scriptName)
		}
		sort.Strings(scriptNames)
		for _, scriptName := range scriptNames {
			tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(scriptName))
		}
	}

	// Add custom action tools from SafeOutputs.Actions (sorted for deterministic output).
	if len(safeOutputs.Actions) > 0 {
		actionNames := make([]string, 0, len(safeOutputs.Actions))
		for actionName := range safeOutputs.Actions {
			actionNames = append(actionNames, actionName)
		}
		sort.Strings(actionNames)
		for _, actionName := range actionNames {
			tools = append(tools, stringutil.NormalizeSafeOutputIdentifier(actionName))
		}
	}

	if len(tools) == 0 {
		return nil
	}

	var sections []PromptSection

	// Build the inline opening: XML tag + compact tools list.
	// Extract any ${{ }} expressions from max: values so they do not appear in the
	// run: heredoc (which is subject to GitHub Actions' 21KB expression-size limit).
	// Expressions are replaced with __GH_AW_...__  placeholders and added to EnvVars
	// so the placeholder substitution step can resolve them at runtime.
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

	// Inline opening: XML tag + compact tools list (with placeholders for any expressions)
	sections = append(sections, PromptSection{
		Content: toolsContent,
		IsFile:  false,
		EnvVars: envVars,
	})

	// File sections for tools with multi-step instructions
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

	// Inline closing tag
	sections = append(sections, PromptSection{
		Content: "</safe-output-tools>",
		IsFile:  false,
	})

	return sections
}
