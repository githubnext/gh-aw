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
				contentLines := strings.Split(cleanedContent, "\n")
				for _, line := range contentLines {
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
				contentLines := strings.Split(cleanedContent, "\n")
				for _, line := range contentLines {
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
		var repoMemContent strings.Builder
		generateRepoMemoryPromptSection(&repoMemContent, data.RepoMemoryConfig)
		sections = append(sections, PromptSection{
			Content: repoMemContent.String(),
			IsFile:  false,
		})
	}

	// 7. Safe outputs instructions (if enabled)
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		unifiedPromptLog.Print("Adding safe outputs section")
		var safeOutputsBuilder strings.Builder
		safeOutputsBuilder.WriteString(`<safe-outputs>
<description>GitHub API Access Instructions</description>
<important>
The gh CLI is NOT authenticated. Do NOT use gh commands for GitHub operations.
</important>
<instructions>
To create or modify GitHub resources (issues, discussions, pull requests, etc.), you MUST call the appropriate safe output tool. Simply writing content will NOT work - the workflow requires actual tool calls.

Temporary IDs: Some safe output tools support a temporary ID field (usually named temporary_id) so you can reference newly-created items elsewhere in the SAME agent output (for example, using #aw_abc1 in a later body). 

**IMPORTANT - temporary_id format rules:**
- If you DON'T need to reference the item later, OMIT the temporary_id field entirely (it will be auto-generated if needed)
- If you DO need cross-references/chaining, you MUST match this EXACT validation regex: /^aw_[A-Za-z0-9]{3,8}$/i
- Format: aw_ prefix followed by 3 to 8 alphanumeric characters (A-Z, a-z, 0-9, case-insensitive)
- Valid alphanumeric characters: ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789
- INVALID examples: aw_ab (too short), aw_123456789 (too long), aw_test-id (contains hyphen), aw_id_123 (contains underscore)
- VALID examples: aw_abc, aw_abc1, aw_Test123, aw_A1B2C3D4, aw_12345678
- To generate valid IDs: use 3-8 random alphanumeric characters or omit the field to let the system auto-generate

Do NOT invent other aw_* formats — downstream steps will reject them with validation errors matching against /^aw_[A-Za-z0-9]{3,8}$/i.

Discover available tools from the safeoutputs MCP server.

**Critical**: Tool calls write structured data that downstream jobs process. Without tool calls, follow-up actions will be skipped.

**Note**: If you made no other safe output tool calls during this workflow execution, call the "noop" tool to provide a status message indicating completion or that no actions were needed.
`)
		generateSafeOutputsPromptSection(&safeOutputsBuilder, data.SafeOutputs)
		safeOutputsBuilder.WriteString("</instructions>\n</safe-outputs>")
		sections = append(sections, PromptSection{
			Content: safeOutputsBuilder.String(),
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

	// Track if we're inside a heredoc and whether we're writing the first content
	inHeredoc := false
	isFirstContent := true

	// 1. Write built-in sections first (prepended), wrapped in <system> tags
	if len(builtinSections) > 0 {
		// Open system tag for built-in prompts
		if isFirstContent {
			yaml.WriteString("          cat << '" + delimiter + "' > \"$GH_AW_PROMPT\"\n")
			isFirstContent = false
		} else {
			yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
		}
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
				if isFirstContent {
					yaml.WriteString("            " + fmt.Sprintf("cat \"%s\" > \"$GH_AW_PROMPT\"\n", promptPath))
					isFirstContent = false
				} else {
					yaml.WriteString("            " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
				}
			} else {
				// Inline content inside conditional - open heredoc, write content, close
				if isFirstContent {
					yaml.WriteString("            cat << '" + delimiter + "' > \"$GH_AW_PROMPT\"\n")
					isFirstContent = false
				} else {
					yaml.WriteString("            cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
				}
				normalizedContent := normalizeLeadingWhitespace(section.Content)
				cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
				contentLines := strings.Split(cleanedContent, "\n")
				for _, line := range contentLines {
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
				if isFirstContent {
					yaml.WriteString("          " + fmt.Sprintf("cat \"%s\" > \"$GH_AW_PROMPT\"\n", promptPath))
					isFirstContent = false
				} else {
					yaml.WriteString("          " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
				}
			} else {
				// Inline content - open heredoc if not already open
				if !inHeredoc {
					if isFirstContent {
						yaml.WriteString("          cat << '" + delimiter + "' > \"$GH_AW_PROMPT\"\n")
						isFirstContent = false
					} else {
						yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
					}
					inHeredoc = true
				}
				// Write content directly to open heredoc
				normalizedContent := normalizeLeadingWhitespace(section.Content)
				cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
				contentLines := strings.Split(cleanedContent, "\n")
				for _, line := range contentLines {
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
		yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
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
			if isFirstContent {
				yaml.WriteString("          cat << '" + delimiter + "' > \"$GH_AW_PROMPT\"\n")
				isFirstContent = false
			} else {
				yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
			}
			yaml.WriteString("          " + chunk + "\n")
			yaml.WriteString("          " + delimiter + "\n")
			continue
		}

		// Regular chunk - close heredoc if open before starting new chunk
		if inHeredoc {
			yaml.WriteString("          " + delimiter + "\n")
			inHeredoc = false
		}

		// Each user prompt chunk is written as a separate heredoc append
		if isFirstContent {
			yaml.WriteString("          cat << '" + delimiter + "' > \"$GH_AW_PROMPT\"\n")
			isFirstContent = false
		} else {
			yaml.WriteString("          cat << '" + delimiter + "' >> \"$GH_AW_PROMPT\"\n")
		}

		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
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

	unifiedPromptLog.Print("Unified prompt creation step generated successfully")

	// Return all expression mappings for use in the placeholder substitution step
	// This allows the substitution to happen AFTER runtime-import processing
	return allExpressionMappings
}

var safeOutputsPromptLog = logger.New("workflow:safe_outputs_prompt")

// generateSafeOutputsPromptSection appends per-tool usage instructions for each
// configured safe-output capability.  It is called from collectPromptSections to
// inject detailed guidance inside the <safe-outputs> XML block.
func generateSafeOutputsPromptSection(b *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	safeOutputsPromptLog.Print("Generating safe outputs prompt section")

	// Build heading that lists every enabled capability
	b.WriteString("\n---\n\n## ")
	written := false
	write := func(label string) {
		if written {
			b.WriteString(", ")
		}
		b.WriteString(label)
		written = true
	}

	if safeOutputs.AddComments != nil {
		write("Adding a Comment to an Issue or Pull Request")
	}
	if safeOutputs.CreateIssues != nil {
		write("Creating an Issue")
	}
	if safeOutputs.CloseIssues != nil {
		write("Closing an Issue")
	}
	if safeOutputs.UpdateIssues != nil {
		write("Updating Issues")
	}
	if safeOutputs.CreateDiscussions != nil {
		write("Creating a Discussion")
	}
	if safeOutputs.UpdateDiscussions != nil {
		write("Updating a Discussion")
	}
	if safeOutputs.CloseDiscussions != nil {
		write("Closing a Discussion")
	}
	if safeOutputs.CreateAgentSessions != nil {
		write("Creating an Agent Session")
	}
	if safeOutputs.CreatePullRequests != nil {
		write("Creating a Pull Request")
	}
	if safeOutputs.ClosePullRequests != nil {
		write("Closing a Pull Request")
	}
	if safeOutputs.UpdatePullRequests != nil {
		write("Updating a Pull Request")
	}
	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		write("Marking a Pull Request as Ready for Review")
	}
	if safeOutputs.CreatePullRequestReviewComments != nil {
		write("Creating a Pull Request Review Comment")
	}
	if safeOutputs.SubmitPullRequestReview != nil {
		write("Submitting a Pull Request Review")
	}
	if safeOutputs.ReplyToPullRequestReviewComment != nil {
		write("Replying to a Pull Request Review Comment")
	}
	if safeOutputs.ResolvePullRequestReviewThread != nil {
		write("Resolving a Pull Request Review Thread")
	}
	if safeOutputs.AddLabels != nil {
		write("Adding Labels to Issues or Pull Requests")
	}
	if safeOutputs.RemoveLabels != nil {
		write("Removing Labels from Issues or Pull Requests")
	}
	if safeOutputs.AddReviewer != nil {
		write("Adding a Reviewer to a Pull Request")
	}
	if safeOutputs.AssignMilestone != nil {
		write("Assigning a Milestone")
	}
	if safeOutputs.AssignToAgent != nil {
		write("Assigning to an Agent")
	}
	if safeOutputs.AssignToUser != nil {
		write("Assigning to a User")
	}
	if safeOutputs.UnassignFromUser != nil {
		write("Unassigning from a User")
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		write("Pushing Changes to Branch")
	}
	if safeOutputs.CreateCodeScanningAlerts != nil {
		write("Creating a Code Scanning Alert")
	}
	if safeOutputs.AutofixCodeScanningAlert != nil {
		write("Autofixing a Code Scanning Alert")
	}
	if safeOutputs.UploadAssets != nil {
		write("Uploading Assets")
	}
	if safeOutputs.UpdateRelease != nil {
		write("Updating a Release")
	}
	if safeOutputs.UpdateProjects != nil {
		write("Updating a Project")
	}
	if safeOutputs.CreateProjects != nil {
		write("Creating a Project")
	}
	if safeOutputs.CreateProjectStatusUpdates != nil {
		write("Creating a Project Status Update")
	}
	if safeOutputs.LinkSubIssue != nil {
		write("Linking a Sub-Issue")
	}
	if safeOutputs.HideComment != nil {
		write("Hiding a Comment")
	}
	if safeOutputs.DispatchWorkflow != nil {
		write("Dispatching a Workflow")
	}
	if safeOutputs.MissingTool != nil {
		write("Reporting Missing Tools or Functionality")
	}
	if safeOutputs.MissingData != nil {
		write("Reporting Missing Data")
	}

	if !written {
		// No specific capabilities listed – nothing more to add.
		return
	}

	b.WriteString("\n\n")
	fmt.Fprintf(b, "**IMPORTANT**: To perform the actions listed above, use the **%s** tools. Do NOT use `gh`, do NOT call the GitHub API directly. You do not have write access to the GitHub repository.\n\n", constants.SafeOutputsMCPServerID)

	if safeOutputs.AddComments != nil {
		b.WriteString("**Adding a Comment to an Issue or Pull Request**\n\n")
		fmt.Fprintf(b, "To add a comment to an issue or pull request, use the add_comment tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateIssues != nil {
		b.WriteString("**Creating an Issue**\n\n")
		fmt.Fprintf(b, "To create an issue, use the create_issue tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CloseIssues != nil {
		b.WriteString("**Closing an Issue**\n\n")
		fmt.Fprintf(b, "To close an issue, use the close_issue tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UpdateIssues != nil {
		b.WriteString("**Updating an Issue**\n\n")
		fmt.Fprintf(b, "To update an issue, use the update_issue tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateDiscussions != nil {
		b.WriteString("**Creating a Discussion**\n\n")
		fmt.Fprintf(b, "To create a discussion, use the create_discussion tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UpdateDiscussions != nil {
		b.WriteString("**Updating a Discussion**\n\n")
		fmt.Fprintf(b, "To update a discussion, use the update_discussion tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CloseDiscussions != nil {
		b.WriteString("**Closing a Discussion**\n\n")
		fmt.Fprintf(b, "To close a discussion, use the close_discussion tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateAgentSessions != nil {
		b.WriteString("**Creating an Agent Session**\n\n")
		fmt.Fprintf(b, "To create a GitHub Copilot agent session, use the create_agent_session tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreatePullRequests != nil {
		b.WriteString("**Creating a Pull Request**\n\n")
		b.WriteString("To create a pull request:\n")
		b.WriteString("1. Make any file changes directly in the working directory.\n")
		b.WriteString("2. If you haven't done so already, create a local branch using an appropriate unique name.\n")
		b.WriteString("3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Verify you haven't deleted or changed any files you didn't intend to.\n")
		b.WriteString("4. Do not push your changes. That will be done by the tool.\n")
		fmt.Fprintf(b, "5. Create the pull request with the create_pull_request tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.ClosePullRequests != nil {
		b.WriteString("**Closing a Pull Request**\n\n")
		fmt.Fprintf(b, "To close a pull request, use the close_pull_request tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UpdatePullRequests != nil {
		b.WriteString("**Updating a Pull Request**\n\n")
		fmt.Fprintf(b, "To update a pull request title or body, use the update_pull_request tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.MarkPullRequestAsReadyForReview != nil {
		b.WriteString("**Marking a Pull Request as Ready for Review**\n\n")
		fmt.Fprintf(b, "To mark a pull request as ready for review, use the mark_pull_request_as_ready_for_review tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreatePullRequestReviewComments != nil {
		b.WriteString("**Creating a Pull Request Review Comment**\n\n")
		fmt.Fprintf(b, "To create a pull request review comment, use the create_pull_request_review_comment tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.SubmitPullRequestReview != nil {
		b.WriteString("**Submitting a Pull Request Review**\n\n")
		fmt.Fprintf(b, "To submit a pull request review (APPROVE, REQUEST_CHANGES, or COMMENT), use the submit_pull_request_review tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.ReplyToPullRequestReviewComment != nil {
		b.WriteString("**Replying to a Pull Request Review Comment**\n\n")
		fmt.Fprintf(b, "To reply to an existing review comment on a pull request, use the reply_to_pull_request_review_comment tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.ResolvePullRequestReviewThread != nil {
		b.WriteString("**Resolving a Pull Request Review Thread**\n\n")
		fmt.Fprintf(b, "To resolve a review thread on a pull request, use the resolve_pull_request_review_thread tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AddLabels != nil {
		b.WriteString("**Adding Labels to Issues or Pull Requests**\n\n")
		fmt.Fprintf(b, "To add labels to an issue or pull request, use the add_labels tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.RemoveLabels != nil {
		b.WriteString("**Removing Labels from Issues or Pull Requests**\n\n")
		fmt.Fprintf(b, "To remove labels from an issue or pull request, use the remove_labels tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AddReviewer != nil {
		b.WriteString("**Adding a Reviewer to a Pull Request**\n\n")
		fmt.Fprintf(b, "To add a reviewer to a pull request, use the add_reviewer tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AssignMilestone != nil {
		b.WriteString("**Assigning a Milestone**\n\n")
		fmt.Fprintf(b, "To assign a milestone to an issue or pull request, use the assign_milestone tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AssignToAgent != nil {
		b.WriteString("**Assigning to an Agent**\n\n")
		fmt.Fprintf(b, "To assign an issue or pull request to a GitHub Copilot agent, use the assign_to_agent tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AssignToUser != nil {
		b.WriteString("**Assigning to a User**\n\n")
		fmt.Fprintf(b, "To assign an issue or pull request to a user, use the assign_to_user tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UnassignFromUser != nil {
		b.WriteString("**Unassigning from a User**\n\n")
		fmt.Fprintf(b, "To remove a user assignee from an issue or pull request, use the unassign_from_user tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		b.WriteString("**Pushing Changes to a Pull Request Branch**\n\n")
		b.WriteString("To push changes to the branch of a pull request:\n")
		b.WriteString("1. Make any file changes directly in the working directory.\n")
		b.WriteString("2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and verify you haven't deleted or changed any files you didn't intend to.\n")
		fmt.Fprintf(b, "3. Push the branch to the repo by using the push_to_pull_request_branch tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateCodeScanningAlerts != nil {
		b.WriteString("**Creating a Code Scanning Alert**\n\n")
		fmt.Fprintf(b, "To create a code scanning alert, use the create_code_scanning_alert tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.AutofixCodeScanningAlert != nil {
		b.WriteString("**Autofixing a Code Scanning Alert**\n\n")
		fmt.Fprintf(b, "To autofix a code scanning alert, use the autofix_code_scanning_alert tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UploadAssets != nil {
		b.WriteString("**Uploading Assets**\n\n")
		b.WriteString("To upload files as URL-addressable assets:\n")
		fmt.Fprintf(b, "1. Use the upload_asset tool from %s.\n", constants.SafeOutputsMCPServerID)
		b.WriteString("2. Provide the path to the file you want to upload.\n")
		b.WriteString("3. The tool will copy the file to a staging area and return a GitHub raw content URL.\n")
		b.WriteString("4. Assets are uploaded to an orphaned git branch after workflow completion.\n\n")
	}

	if safeOutputs.UpdateRelease != nil {
		b.WriteString("**Updating a Release**\n\n")
		fmt.Fprintf(b, "To update a GitHub release description, use the update_release tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.UpdateProjects != nil {
		b.WriteString("**Updating a Project**\n\n")
		fmt.Fprintf(b, "To create, add items to, or update a project board, use the update_project tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateProjects != nil {
		b.WriteString("**Creating a Project**\n\n")
		fmt.Fprintf(b, "To create a GitHub Projects V2 project, use the create_project tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.CreateProjectStatusUpdates != nil {
		b.WriteString("**Creating a Project Status Update**\n\n")
		fmt.Fprintf(b, "To create a project status update, use the create_project_status_update tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.LinkSubIssue != nil {
		b.WriteString("**Linking a Sub-Issue**\n\n")
		fmt.Fprintf(b, "To link an issue as a sub-issue of another issue, use the link_sub_issue tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.HideComment != nil {
		b.WriteString("**Hiding a Comment**\n\n")
		fmt.Fprintf(b, "To hide a comment, use the hide_comment tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.DispatchWorkflow != nil {
		b.WriteString("**Dispatching a Workflow**\n\n")
		fmt.Fprintf(b, "To dispatch a workflow_dispatch event to another workflow, use the dispatch_workflow tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.MissingTool != nil {
		b.WriteString("**Reporting Missing Tools or Functionality**\n\n")
		fmt.Fprintf(b, "To report a missing tool or capability, use the missing_tool tool from %s.\n\n", constants.SafeOutputsMCPServerID)
	}

	if safeOutputs.MissingData != nil {
		b.WriteString("**Reporting Missing Data**\n\n")
		fmt.Fprintf(b, "To report missing data required to achieve a goal, use the missing_data tool from %s.\n\n", constants.SafeOutputsMCPServerID)
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
