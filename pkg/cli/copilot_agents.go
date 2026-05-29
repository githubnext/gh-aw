package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotAgentsLog = logger.New("cli:copilot_agents")

const agenticWorkflowsAgentHeader = "---\n" +
	"name: Agentic Workflows\n" +
	"description: Minimal file index for GitHub Agentic Workflows tasks in this repository.\n" +
	"---\n\n" +
	"# Agentic Workflows\n\n" +
	"Read only the files you need:\n"

// ensureAgenticWorkflowsDispatcher ensures that .github/skills/agentic-workflows/SKILL.md contains the dispatcher skill
func ensureAgenticWorkflowsDispatcher(verbose bool, skipInstructions bool) error {
	copilotAgentsLog.Print("Ensuring agentic workflows dispatcher skill")

	if skipInstructions {
		copilotAgentsLog.Print("Skipping skill creation: instructions disabled")
		return nil
	}

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	targetDir := filepath.Join(gitRoot, ".github", "skills", "agentic-workflows")
	targetPath := filepath.Join(targetDir, "SKILL.md")

	// Ensure the target directory exists
	if err := os.MkdirAll(targetDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create .github/skills/agentic-workflows directory: %w", err)
	}

	// Download the skill file from GitHub
	skillContent, err := downloadSkillFileFromGitHub(verbose)
	if err != nil {
		copilotAgentsLog.Printf("Failed to download skill file from GitHub: %v", err)
		return fmt.Errorf("failed to download skill file from GitHub: %w", err)
	}

	// Check if the file already exists and matches the downloaded content
	existingContent := ""
	if content, err := os.ReadFile(targetPath); err == nil {
		existingContent = string(content)
	}

	// Check if content matches the downloaded template
	expectedContent := strings.TrimSpace(skillContent)
	if strings.TrimSpace(existingContent) == expectedContent {
		copilotAgentsLog.Printf("Dispatcher skill is up-to-date: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dispatcher skill is up-to-date: "+targetPath))
		}
		return nil
	}

	// Skill files are committed repository instructions, so keep them world-readable.
	if err := os.WriteFile(targetPath, []byte(skillContent), constants.FilePermPublic); err != nil {
		copilotAgentsLog.Printf("Failed to write dispatcher skill: %s, error: %v", targetPath, err)
		return fmt.Errorf("failed to write dispatcher skill: %w", err)
	}

	if existingContent == "" {
		copilotAgentsLog.Printf("Created dispatcher skill: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher skill: "+targetPath))
		}
	} else {
		copilotAgentsLog.Printf("Updated dispatcher skill: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated dispatcher skill: "+targetPath))
		}
	}

	return nil
}

// ensureAgenticWorkflowsAgent ensures that .github/agents/agentic-workflows.md contains the custom agent.
func ensureAgenticWorkflowsAgent(verbose bool) error {
	copilotAgentsLog.Print("Ensuring agentic workflows custom agent")

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(gitRoot, ".github", "agents")
	targetPath := filepath.Join(targetDir, "agentic-workflows.md")

	if err := os.MkdirAll(targetDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create .github/agents directory: %w", err)
	}

	existingContent := ""
	if content, err := os.ReadFile(targetPath); err == nil {
		existingContent = string(content)
	}

	agenticWorkflowsAgentContent, err := buildAgenticWorkflowsAgentContent(gitRoot)
	if err != nil {
		return err
	}

	expectedContent := strings.TrimSpace(agenticWorkflowsAgentContent)
	if strings.TrimSpace(existingContent) == expectedContent {
		copilotAgentsLog.Printf("Agentic Workflows custom agent is up-to-date: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Agentic Workflows custom agent is up-to-date: "+targetPath))
		}
		return nil
	}

	if err := os.WriteFile(targetPath, []byte(agenticWorkflowsAgentContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write Agentic Workflows custom agent: %w", err)
	}

	if existingContent == "" {
		copilotAgentsLog.Printf("Created Agentic Workflows custom agent: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created Agentic Workflows custom agent: "+targetPath))
		}
	} else {
		copilotAgentsLog.Printf("Updated Agentic Workflows custom agent: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated Agentic Workflows custom agent: "+targetPath))
		}
	}

	return nil
}

func buildAgenticWorkflowsAgentContent(gitRoot string) (string, error) {
	lines := []string{
		strings.TrimRight(agenticWorkflowsAgentHeader, "\n"),
		formatAgenticWorkflowsAgentEntry(".github/skills/agentic-workflows/SKILL.md", "router skill for workflow create, debug, and upgrade tasks"),
	}

	promptPaths, err := filepath.Glob(filepath.Join(gitRoot, ".github", "aw", "*.md"))
	if err != nil {
		return "", fmt.Errorf("failed to list .github/aw prompts: %w", err)
	}

	for _, promptPath := range promptPaths {
		purpose, err := summarizeAgenticWorkflowPrompt(promptPath)
		if err != nil {
			return "", fmt.Errorf("failed to summarize prompt %s: %w", promptPath, err)
		}

		relPath, err := filepath.Rel(gitRoot, promptPath)
		if err != nil {
			return "", fmt.Errorf("failed to compute relative path for %s: %w", promptPath, err)
		}

		lines = append(lines, formatAgenticWorkflowsAgentEntry(filepath.ToSlash(relPath), purpose))
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func formatAgenticWorkflowsAgentEntry(path, purpose string) string {
	return fmt.Sprintf("- `%s` — %s.", path, strings.TrimSuffix(purpose, "."))
}

func summarizeAgenticWorkflowPrompt(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	summary := extractPromptSummary(string(content))
	if summary == "" {
		summary = humanizePromptFilename(filepath.Base(path))
	}

	return strings.TrimSuffix(summary, "."), nil
}

func extractPromptSummary(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}

	start := 0
	description := ""
	if strings.TrimSpace(lines[0]) == "---" {
		start = 1
		for ; start < len(lines); start++ {
			line := strings.TrimSpace(lines[start])
			if line == "---" {
				start++
				break
			}
			if strings.HasPrefix(line, "description:") {
				description = cleanPromptSummary(strings.TrimSpace(strings.TrimPrefix(line, "description:")))
			}
		}
	}

	for ; start < len(lines); start++ {
		line := strings.TrimSpace(lines[start])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			return cleanPromptSummary(strings.TrimSpace(strings.TrimPrefix(line, "# ")))
		}
		break
	}

	if description != "" && len(description) <= 80 && !strings.ContainsAny(description, "*`#") {
		return description
	}

	return ""
}

func cleanPromptSummary(summary string) string {
	summary = strings.Trim(summary, `"'`)
	summary = strings.TrimSpace(summary)
	summary = strings.TrimPrefix(summary, "RECOMMENDED: ")
	summary = strings.TrimPrefix(summary, "✅ GOOD - ")
	summary = strings.TrimPrefix(summary, "✅ ")
	summary = strings.TrimSpace(summary)

	if idx := strings.Index(summary, " - "); idx > 0 {
		summary = summary[:idx]
	}

	if idx := strings.Index(summary, ","); idx > 0 && len(summary) > 80 {
		summary = summary[:idx]
	}

	return summary
}

func humanizePromptFilename(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	switch base {
	case "agentic-chat":
		return "draft task descriptions"
	case "asciicharts":
		return "render ASCII charts"
	case "cli-commands":
		return "reference gh aw CLI commands"
	case "context":
		return "use sanitized context text"
	case "github-agentic-workflows":
		return "reference workflow instructions"
	case "github-mcp-server":
		return "use the GitHub MCP server"
	case "llms":
		return "discover LLM API endpoints"
	case "pr-reviewer":
		return "design PR reviewer workflows"
	case "safe-outputs":
		return "configure safe outputs"
	case "serena-tool":
		return "use the Serena tool"
	case "test-coverage":
		return "add test coverage guidance"
	case "test-expression":
		return "test expressions"
	case "token-optimization":
		return "reduce token usage"
	case "visual-regression":
		return "run visual regression tests"
	}

	summary := strings.ReplaceAll(base, "-", " ")
	if summary == "" {
		return ""
	}
	return strings.ToUpper(summary[:1]) + summary[1:]
}

// cleanupOldPromptFile removes an old prompt file from .github/prompts/ if it exists
func cleanupOldPromptFile(promptFileName string, verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	oldPath := filepath.Join(gitRoot, ".github", "prompts", promptFileName)

	// Check if the old file exists and remove it
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old prompt file: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed old prompt file: "+oldPath))
		}
	}

	return nil
}

// deleteSetupAgenticWorkflowsAgent deletes the setup-agentic-workflows.agent.md file if it exists
func deleteSetupAgenticWorkflowsAgent(verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	agentPath := filepath.Join(gitRoot, ".github", "agents", "setup-agentic-workflows.agent.md")

	// Check if the file exists and remove it
	if _, err := os.Stat(agentPath); err == nil {
		if err := os.Remove(agentPath); err != nil {
			return fmt.Errorf("failed to remove setup-agentic-workflows agent: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Removed setup-agentic-workflows agent: %s\n", agentPath)
		}
	}

	// Also clean up the old prompt file if it exists
	return cleanupOldPromptFile("setup-agentic-workflows.prompt.md", verbose)
}

// deleteOldTemplateFiles deletes old template files that are no longer bundled in the binary
func deleteOldTemplateFiles(verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	// All template files that were previously bundled
	// Now that we download the agent file on demand, all files should be removed
	templateFiles := []string{
		"agentic-workflows.agent.md",
		"create-agentic-workflow.md",
		"create-shared-agentic-workflow.md",
		"debug-agentic-workflow.md",
		"github-agentic-workflows.md",
		"serena-tool.md",
		"update-agentic-workflow.md",
		"upgrade-agentic-workflows.md",
	}

	templatesDir := filepath.Join(gitRoot, "pkg", "cli", "templates")

	// Check if templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to clean up
		return nil
	}

	removedCount := 0
	for _, file := range templateFiles {
		path := filepath.Join(templatesDir, file)
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove old template file %s: %w", file, err)
			}
			removedCount++
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed old template file: "+path))
			}
		}
	}

	// If any files were removed, try to remove the directory if it's now empty
	if removedCount > 0 {
		entries, err := os.ReadDir(templatesDir)
		if err == nil && len(entries) == 0 {
			if err := os.Remove(templatesDir); err != nil {
				return fmt.Errorf("failed to remove empty templates directory: %w", err)
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed empty templates directory: "+templatesDir))
			}
		}
	}

	return nil
}

// deleteLegacyAgentFiles deletes legacy workflow-specific agent files from .github/agents/.
func deleteLegacyAgentFiles(verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	// Map of subdirectory to list of files to delete
	filesToDelete := map[string][]string{
		"agents": {
			"agentic-workflows.agent.md",
			"create-agentic-workflow.agent.md",
			"debug-agentic-workflow.agent.md",
			"create-shared-agentic-workflow.agent.md",
			"create-shared-agentic-workflow.md",
			"create-agentic-workflow.md",
			"setup-agentic-workflows.md",
			"update-agentic-workflows.md",
			"upgrade-agentic-workflows.md",
		},
		"aw": {
			"upgrade-agentic-workflow.md", // singular form (typo/duplicate)
		},
	}

	for subdir, files := range filesToDelete {
		for _, file := range files {
			path := filepath.Join(gitRoot, ".github", subdir, file)
			if _, err := os.Stat(path); err == nil {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove old %s file %s: %w", subdir, file, err)
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "Removed old %s file: %s\n", subdir, path)
				}
			}
		}
	}

	return nil
}
