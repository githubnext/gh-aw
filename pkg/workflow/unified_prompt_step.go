package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
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

	// Collect all prompt sections in order
	sections := c.collectPromptSections(data)

	if len(sections) == 0 {
		unifiedPromptLog.Print("No prompt sections to append, skipping unified step")
		return
	}

	unifiedPromptLog.Printf("Collected %d prompt sections", len(sections))

	// Collect all environment variables from all sections
	allEnvVars := make(map[string]string)
	for _, section := range sections {
		for key, value := range section.EnvVars {
			allEnvVars[key] = value
		}
	}

	// Generate the step
	yaml.WriteString("      - name: Append context instructions to prompt\n")
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
				yaml.WriteString("          PROMPT_EOF\n")
				inHeredoc = false
			}
			fmt.Fprintf(yaml, "          if %s; then\n", section.ShellCondition)

			if section.IsFile {
				// File reference inside conditional
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("            " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
			} else {
				// Inline content inside conditional - open heredoc, write content, close
				yaml.WriteString("            cat << 'PROMPT_EOF' >> \"$GH_AW_PROMPT\"\n")
				normalizedContent := normalizeLeadingWhitespace(section.Content)
				cleanedContent := removeConsecutiveEmptyLines(normalizedContent)
				contentLines := strings.Split(cleanedContent, "\n")
				for _, line := range contentLines {
					yaml.WriteString("            " + line + "\n")
				}
				yaml.WriteString("            PROMPT_EOF\n")
			}

			yaml.WriteString("          fi\n")
		} else {
			// Unconditional section
			if section.IsFile {
				// Close heredoc if open
				if inHeredoc {
					yaml.WriteString("          PROMPT_EOF\n")
					inHeredoc = false
				}
				// Cat the file
				promptPath := fmt.Sprintf("%s/%s", promptsDir, section.Content)
				yaml.WriteString("          " + fmt.Sprintf("cat \"%s\" >> \"$GH_AW_PROMPT\"\n", promptPath))
			} else {
				// Inline content - open heredoc if not already open
				if !inHeredoc {
					yaml.WriteString("          cat << 'PROMPT_EOF' >> \"$GH_AW_PROMPT\"\n")
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
		yaml.WriteString("          PROMPT_EOF\n")
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

	// 1. Temporary folder instructions (always included)
	unifiedPromptLog.Print("Adding temp folder section")
	sections = append(sections, PromptSection{
		Content: tempFolderPromptFile,
		IsFile:  true,
	})

	// 2. Playwright instructions (if playwright tool is enabled)
	if hasPlaywrightTool(data.ParsedTools) {
		unifiedPromptLog.Print("Adding playwright section")
		sections = append(sections, PromptSection{
			Content: playwrightPromptFile,
			IsFile:  true,
		})
	}

	// 3. Trial mode note (if in trial mode)
	if c.trialMode {
		unifiedPromptLog.Print("Adding trial mode section")
		trialContent := fmt.Sprintf("## Note\nThis workflow is running in directory $GITHUB_WORKSPACE, but that directory actually contains the contents of the repository '%s'.", c.trialLogicalRepoSlug)
		sections = append(sections, PromptSection{
			Content: trialContent,
			IsFile:  false,
		})
	}

	// 4. Cache memory instructions (if enabled)
	if data.CacheMemoryConfig != nil && len(data.CacheMemoryConfig.Caches) > 0 {
		unifiedPromptLog.Printf("Adding cache memory section: caches=%d", len(data.CacheMemoryConfig.Caches))
		var cacheContent strings.Builder
		generateCacheMemoryPromptSection(&cacheContent, data.CacheMemoryConfig)
		sections = append(sections, PromptSection{
			Content: cacheContent.String(),
			IsFile:  false,
		})
	}

	// 5. Repo memory instructions (if enabled)
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		unifiedPromptLog.Printf("Adding repo memory section: memories=%d", len(data.RepoMemoryConfig.Memories))
		var repoMemContent strings.Builder
		generateRepoMemoryPromptSection(&repoMemContent, data.RepoMemoryConfig)
		sections = append(sections, PromptSection{
			Content: repoMemContent.String(),
			IsFile:  false,
		})
	}

	// 6. Safe outputs instructions (if enabled)
	if HasSafeOutputsEnabled(data.SafeOutputs) {
		enabledTools := GetEnabledSafeOutputToolNames(data.SafeOutputs)
		if len(enabledTools) > 0 {
			unifiedPromptLog.Printf("Adding safe outputs section: tools=%d", len(enabledTools))
			toolsList := strings.Join(enabledTools, ", ")
			safeOutputsContent := fmt.Sprintf(`<safe-outputs>
<description>GitHub API Access Instructions</description>
<important>
The gh CLI is NOT authenticated. Do NOT use gh commands for GitHub operations.
</important>
<instructions>
To create or modify GitHub resources (issues, discussions, pull requests, etc.), you MUST call the appropriate safe output tool. Simply writing content will NOT work - the workflow requires actual tool calls.

**Available tools**: %s

**Critical**: Tool calls write structured data that downstream jobs process. Without tool calls, follow-up actions will be skipped.
</instructions>
</safe-outputs>`, toolsList)
			sections = append(sections, PromptSection{
				Content: safeOutputsContent,
				IsFile:  false,
			})
		}
	}

	// 7. GitHub context (if GitHub tool is enabled)
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

	// 8. PR context (if comment-related triggers and checkout is needed)
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
		shellCondition := `[ "$GITHUB_EVENT_NAME" = "issue_comment" -a -n "$GH_AW_IS_PR_COMMENT" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review_comment" ] || [ "$GITHUB_EVENT_NAME" = "pull_request_review" ]`

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
