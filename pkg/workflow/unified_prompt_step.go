package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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

// generateUnifiedPromptStep generates a single workflow step that appends all prompt sections.
// This consolidates what used to be multiple separate steps (temp folder, playwright, safe outputs,
// GitHub context, PR context, cache memory, repo memory) into one step.
func (c *Compiler) generateUnifiedPromptStep(yaml *strings.Builder, data *WorkflowData) {
	unifiedPromptLog.Print("Generating unified prompt step")

	// Get the heredoc delimiter for consistent usage
	delimiter := GenerateHeredocDelimiter("PROMPT")

	// Collect all prompt sections in order
	sections := c.collectPromptSections(data)

	if len(sections) == 0 {
		unifiedPromptLog.Print("No prompt sections to append, skipping unified step")
		return
	}

	unifiedPromptLog.Printf("Collected %d prompt sections", len(sections))

	// Collect all environment variables from all sections
	// Only include GitHub Actions expressions in the prompt creation step
	// Static values should only be in the substitution step
	allEnvVars := make(map[string]string)
	for _, section := range sections {
		for key, value := range section.EnvVars {
			// Only add GitHub Actions expressions to the prompt creation step
			// Static values (not wrapped in ${{ }}) are for the substitution step only
			if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
				allEnvVars[key] = value
			}
		}
	}

	// Generate the step
	yaml.WriteString("      - name: Create prompt with built-in context\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

	// Add all environment variables in sorted order for consistency
	var envKeys []string
	for key := range allEnvVars {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		fmt.Fprintf(yaml, "          %s: %s\n", key, allEnvVars[key])
	}

	yaml.WriteString("        run: |\n")

	// Track if we're inside a heredoc
	inHeredoc := false

	// Write each section's content
	for i, section := range sections {
		unifiedPromptLog.Printf("Writing section %d/%d: hasCondition=%v, isFile=%v",
			i+1, len(sections), section.ShellCondition != "", section.IsFile)

		if section.ShellCondition != "" {
			// Close heredoc if open, add conditional
			if inHeredoc {
				yaml.WriteString("          " + delimiter + "\n")
				inHeredoc = false
			}
			fmt.Fprintf(yaml, "          if %s; then\n", section.ShellCondition)

			if section.IsFile {
				// File reference inside conditional
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("            " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
			} else {
				// Inline content inside conditional - open heredoc, write content, close
				yaml.WriteString("            cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
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
				// Cat the file
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("          " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
			} else {
				// Inline content - open heredoc if not already open
				if !inHeredoc {
					yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
					inHeredoc = true
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

	// Close heredoc if still open
	if inHeredoc {
		yaml.WriteString("          " + delimiter + "\n")
	}

	unifiedPromptLog.Print("Unified prompt step generated successfully")
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

	// 4. Trial mode note (if in trial mode)
	if c.trialMode {
		unifiedPromptLog.Print("Adding trial mode section")
		trialContent := fmt.Sprintf("## Note\nThis workflow is running in directory $GITHUB_WORKSPACE, but that directory actually contains the contents of the repository '%s'.", c.trialLogicalRepoSlug)
		sections = append(sections, PromptSection{
			Content: trialContent,
			IsFile:  false,
		})
	}

	// 5. Cache memory instructions (if enabled)
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		unifiedPromptLog.Printf("Adding cache memory section: caches=%d", len(data.CacheMemoryConfig.Caches))
		section := buildCacheMemoryPromptSection(data.CacheMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	// 6. Repo memory instructions (if enabled)
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		unifiedPromptLog.Printf("Adding repo memory section: memories=%d", len(data.RepoMemoryConfig.Memories))
		section := buildRepoMemoryPromptSection(data.RepoMemoryConfig)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	// 7. Safe outputs instructions (if enabled)
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		unifiedPromptLog.Print("Adding safe outputs section")
		// Static intro from file (gh CLI warning, temporary ID rules, noop note)
		sections = append(sections, PromptSection{
			Content: safeOutputsPromptFile,
			IsFile:  true,
		})
		// Per-tool instructions (dynamic, depends on which tools are enabled)
		var perToolBuilder strings.Builder
		perToolBuilder.WriteString("<safe-output-tools>\n")
		generateSafeOutputsPerToolInstructions(&perToolBuilder, data.SafeOutputs)
		perToolBuilder.WriteString("</safe-output-tools>")
		sections = append(sections, PromptSection{
			Content: perToolBuilder.String(),
			IsFile:  false,
		})
	}

	// 8. GitHub context (if GitHub tool is enabled)
	if hasGitHubTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding GitHub context section")
		// Extract expressions from GitHub context prompt
		extractor := NewExpressionExtractor()
		expressionMappings, err := extractor.ExtractExpressions(githubContextPromptText)
		if err == nil && len(expressionMappings) > 0 {
			// Replace expressions with environment variable references
			modifiedPromptText := extractor.ReplaceExpressionsWithEnvVars(githubContextPromptText)

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

	// 9. PR context (if comment-related triggers and checkout is needed)
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)
	needsCheckout := c.shouldAddCheckoutStep(data)
	permParser := NewPermissionsParser(data.Permissions)
	hasContentsRead := permParser.HasContentsReadAccess()

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
	delimiter := GenerateHeredocDelimiter("PROMPT")

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
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}\n")
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

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          bash /opt/gh-aw/actions/create_prompt_first.sh\n")
	yaml.WriteString("          {\n")

	// Track if we're inside a heredoc
	inHeredoc := false

	// 1. Write built-in sections first (prepended), wrapped in <system> tags
	if len(builtinSections) > 0 {
		// Open system tag for built-in prompts
		yaml.WriteString("          cat << '" + delimiter + "'\n")
		yaml.WriteString("          <system>\n")
		yaml.WriteString("          " + delimiter + "\n")
	}

	for i, section := range builtinSections {
		unifiedPromptLog.Printf("Writing built-in section %d/%d: hasCondition=%v, isFile=%v",
			i+1, len(builtinSections), section.ShellCondition != "", section.IsFile)

		if section.ShellCondition != "" {
			// Close heredoc if open, add conditional
			if inHeredoc {
				yaml.WriteString("          " + delimiter + "\n")
				inHeredoc = false
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
				// Cat the file
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("          " + fmt.Sprintf("cat \"%s\"\n", promptPath))
			} else {
				// Inline content - open heredoc if not already open
				if !inHeredoc {
					yaml.WriteString("          cat << '" + delimiter + "'\n")
					inHeredoc = true
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

	// Close system tag for built-in prompts
	if len(builtinSections) > 0 {
		// Close heredoc if open
		if inHeredoc {
			yaml.WriteString("          " + delimiter + "\n")
			inHeredoc = false
		}
		yaml.WriteString("          cat << '" + delimiter + "'\n")
		yaml.WriteString("          </system>\n")
		yaml.WriteString("          " + delimiter + "\n")
	}

	// 2. Write user prompt chunks (appended after built-in sections)
	for chunkIdx, chunk := range userPromptChunks {
		unifiedPromptLog.Printf("Writing user prompt chunk %d/%d", chunkIdx+1, len(userPromptChunks))

		// Check if this chunk is a runtime-import macro
		if strings.HasPrefix(chunk, "{{#runtime-import ") && strings.HasSuffix(chunk, "}}") {
			// This is a runtime-import macro - write it using heredoc for safe escaping
			unifiedPromptLog.Print("Detected runtime-import macro, writing directly")

			// Close heredoc if open before writing runtime-import macro
			if inHeredoc {
				yaml.WriteString("          " + delimiter + "\n")
				inHeredoc = false
			}

			// Write the macro directly with proper indentation
			// Write the macro using a heredoc to avoid potential escaping issues
			yaml.WriteString("          cat << '" + delimiter + "'\n")
			yaml.WriteString("          " + chunk + "\n")
			yaml.WriteString("          " + delimiter + "\n")
			continue
		}

		// Regular chunk - close heredoc if open before starting new chunk
		if inHeredoc {
			yaml.WriteString("          " + delimiter + "\n")
			inHeredoc = false
		}

		// Each user prompt chunk is written as a separate heredoc
		yaml.WriteString("          cat << '" + delimiter + "'\n")

		lines := strings.SplitSeq(chunk, "\n")
		for line := range lines {
			yaml.WriteString("          ")
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
		yaml.WriteString("          " + delimiter + "\n")
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

// generateSafeOutputsPerToolInstructions appends per-tool usage instructions for each
// configured safe-output capability. It is called from collectPromptSections to inject
// tool-specific guidance inside the <safe-output-tools> XML block.
//
// The static intro (gh CLI warning, temporary ID rules, noop note) lives in
// actions/setup/md/safe_outputs_prompt.md and is included separately via a file reference.
func generateSafeOutputsPerToolInstructions(b *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	safeOutputsPromptLog.Print("Generating safe outputs per-tool instructions")

	// Build compact list of enabled tool names
	var tools []string
	if safeOutputs.AddComments != nil {
		tools = append(tools, "add_comment")
	}
	if safeOutputs.CreateIssues != nil {
		tools = append(tools, "create_issue")
	}
	if safeOutputs.CloseIssues != nil {
		tools = append(tools, "close_issue")
	}
	if safeOutputs.UpdateIssues != nil {
		tools = append(tools, "update_issue")
	}
	if safeOutputs.CreateDiscussions != nil {
		tools = append(tools, "create_discussion")
	}
	if safeOutputs.UpdateDiscussions != nil {
		tools = append(tools, "update_discussion")
	}
	if safeOutputs.CloseDiscussions != nil {
		tools = append(tools, "close_discussion")
	}
	if safeOutputs.CreateAgentSessions != nil {
		tools = append(tools, "create_agent_session")
	}
	if safeOutputs.CreatePullRequests != nil {
		tools = append(tools, "create_pull_request")
	}
	if safeOutputs.ClosePullRequests != nil {
		tools = append(tools, "close_pull_request")
	}
	if safeOutputs.UpdatePullRequests != nil {
		tools = append(tools, "update_pull_request")
	}
	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		tools = append(tools, "mark_pull_request_as_ready_for_review")
	}
	if safeOutputs.CreatePullRequestReviewComments != nil {
		tools = append(tools, "create_pull_request_review_comment")
	}
	if safeOutputs.SubmitPullRequestReview != nil {
		tools = append(tools, "submit_pull_request_review")
	}
	if safeOutputs.ReplyToPullRequestReviewComment != nil {
		tools = append(tools, "reply_to_pull_request_review_comment")
	}
	if safeOutputs.ResolvePullRequestReviewThread != nil {
		tools = append(tools, "resolve_pull_request_review_thread")
	}
	if safeOutputs.AddLabels != nil {
		tools = append(tools, "add_labels")
	}
	if safeOutputs.RemoveLabels != nil {
		tools = append(tools, "remove_labels")
	}
	if safeOutputs.AddReviewer != nil {
		tools = append(tools, "add_reviewer")
	}
	if safeOutputs.AssignMilestone != nil {
		tools = append(tools, "assign_milestone")
	}
	if safeOutputs.AssignToAgent != nil {
		tools = append(tools, "assign_to_agent")
	}
	if safeOutputs.AssignToUser != nil {
		tools = append(tools, "assign_to_user")
	}
	if safeOutputs.UnassignFromUser != nil {
		tools = append(tools, "unassign_from_user")
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		tools = append(tools, "push_to_pull_request_branch")
	}
	if safeOutputs.CreateCodeScanningAlerts != nil {
		tools = append(tools, "create_code_scanning_alert")
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		tools = append(tools, "autofix_code_scanning_alert")
	}
	if safeOutputs.UploadAssets != nil {
		tools = append(tools, "upload_asset")
	}
	if safeOutputs.UpdateRelease != nil {
		tools = append(tools, "update_release")
	}
	if safeOutputs.UpdateProjects != nil {
		tools = append(tools, "update_project")
	}
	if safeOutputs.CreateProjects != nil {
		tools = append(tools, "create_project")
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		tools = append(tools, "create_project_status_update")
	}
	if safeOutputs.LinkSubIssue != nil {
		tools = append(tools, "link_sub_issue")
	}
	if safeOutputs.HideComment != nil {
		tools = append(tools, "hide_comment")
	}
	if safeOutputs.DispatchWorkflow != nil {
		tools = append(tools, "dispatch_workflow")
	}
	if safeOutputs.MissingTool != nil {
		tools = append(tools, "missing_tool")
	}
	if safeOutputs.MissingData != nil {
		tools = append(tools, "missing_data")
	}

	if len(tools) == 0 {
		return
	}

	fmt.Fprintf(b, "Tools: %s\n\n", strings.Join(tools, ", "))

	// Special multi-step instructions for complex tools
	if safeOutputs.CreatePullRequests != nil {
		b.WriteString("**Creating a Pull Request**\n\n")
		b.WriteString("To create a pull request:\n")
		b.WriteString("1. Make any file changes directly in the working directory.\n")
		b.WriteString("2. If you haven't done so already, create a local branch using an appropriate unique name.\n")
		b.WriteString("3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Verify you haven't deleted or changed any files you didn't intend to.\n")
		b.WriteString("4. Do not push your changes. That will be done by the tool.\n")
		fmt.Fprintf(b, "5. Create the pull request with the create_pull_request tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		b.WriteString("**Pushing Changes to a Pull Request Branch**\n\n")
		b.WriteString("To push changes to the branch of a pull request:\n")
		b.WriteString("1. Make any file changes directly in the working directory.\n")
		b.WriteString("2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and verify you haven't deleted or changed any files you didn't intend to.\n")
		fmt.Fprintf(b, "3. Push the branch to the repo by using the push_to_pull_request_branch tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UploadAssets != nil {
		fmt.Fprintf(b, "upload_asset: provide a file path; returns a URL; assets are published after the workflow completes (%s).\n\n", constants.SafeOutputsMCPServerID)
	}

	// Auto-injected create_issue special notice
	if safeOutputs.CreateIssues != nil && safeOutputs.AutoInjectedCreateIssue {
		b.WriteString("**IMPORTANT**: Report your findings or results by creating a GitHub issue using the create_issue tool. If you have no meaningful results to report, call the noop tool instead.\n\n")
	}
}

var promptStepHelperLog = logger.New("workflow:prompt_step_helper")

// generateStaticPromptStep is a helper function that generates a workflow step
// for appending static prompt text to the prompt file. It encapsulates the common
// pattern used across multiple prompt generators (XPIA, temp folder, playwright, edit tool, etc.)
// to reduce code duplication and ensure consistency.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - description: The name of the workflow step (e.g., "Append XPIA security instructions to prompt")
//   - promptText: The static text content to append to the prompt (used for backward compatibility)
//   - shouldInclude: Whether to generate the step (false means skip generation entirely)
//
// Example usage:
//
//	generateStaticPromptStep(yaml,
//	    "Append XPIA security instructions to prompt",
//	    xpiaPromptText,
//	    data.SafetyPrompt)
//
// Deprecated: This function is kept for backward compatibility with inline prompts.
// Use generateStaticPromptStepFromFile for new code.
func generateStaticPromptStep(yaml *strings.Builder, description string, promptText string, shouldInclude bool) {
	promptStepHelperLog.Printf("Generating static prompt step: description=%s, shouldInclude=%t", description, shouldInclude)
	// Skip generation if guard condition is false
	if !shouldInclude {
		return
	}

	// Use the existing appendPromptStep helper with a renderer that writes the prompt text
	appendPromptStep(yaml,
		description,
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, promptText, indent)
		},
		"", // no condition
		"          ")
}
