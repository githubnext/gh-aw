package cli

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotAgentsLog = logger.New("cli:copilot_agents")

const agenticWorkflowsSkillFileListPlaceholder = "{{AW_FILE_LIST}}"
const ghAWMarkdownFilesAPIURL = "https://api.github.com/repos/github/gh-aw/contents/.github/aw?ref=main"

//go:embed data/agentic_workflows_agent.md
var agenticWorkflowsAgentTemplate string

//go:embed data/agentic_workflows_skill.md
var agenticWorkflowsSkillTemplate string

//go:embed data/agentic_workflows_fallback_aw_files.json
var agenticWorkflowsFallbackAWFiles string

//go:embed data/agentic_workflow_designer_skill.md
var agenticWorkflowDesignerSkillTemplate string

var listAgenticWorkflowsMarkdownFiles = fetchAgenticWorkflowsMarkdownFiles

// ensureAgenticWorkflowsDispatcher ensures that .github/skills/agentic-workflows/SKILL.md
// exists and contains the routing instructions loaded by the Agentic Workflows agent.
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

	if err := fileutil.EnsureParentDir(targetPath, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create .github/skills/agentic-workflows directory: %w", err)
	}

	skillContent, err := buildAgenticWorkflowsSkillContent()
	if err != nil {
		copilotAgentsLog.Printf("Failed to build dispatcher skill: %v", err)
		return fmt.Errorf("failed to build dispatcher skill: %w", err)
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

// ensureAgenticWorkflowDesignerSkill ensures that
// .github/skills/agentic-workflow-designer/SKILL.md exists and matches the
// bundled workflow designer skill content.
func ensureAgenticWorkflowDesignerSkill(verbose bool, skipInstructions bool) error {
	copilotAgentsLog.Print("Ensuring agentic workflow designer skill")

	if skipInstructions {
		copilotAgentsLog.Print("Skipping skill creation: instructions disabled")
		return nil
	}

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	targetDir := filepath.Join(gitRoot, ".github", "skills", "agentic-workflow-designer")
	targetPath := filepath.Join(targetDir, "SKILL.md")

	if err := fileutil.EnsureParentDir(targetPath, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create .github/skills/agentic-workflow-designer directory: %w", err)
	}

	existingContent := ""
	if content, err := os.ReadFile(targetPath); err == nil {
		existingContent = string(content)
	}

	expectedContent := strings.TrimSpace(agenticWorkflowDesignerSkillTemplate)
	if strings.TrimSpace(existingContent) == expectedContent {
		copilotAgentsLog.Printf("Agentic workflow designer skill is up-to-date: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Agentic workflow designer skill is up-to-date: "+targetPath))
		}
		return nil
	}

	if err := os.WriteFile(targetPath, []byte(agenticWorkflowDesignerSkillTemplate), constants.FilePermPublic); err != nil {
		copilotAgentsLog.Printf("Failed to write agentic workflow designer skill: %s, error: %v", targetPath, err)
		return fmt.Errorf("failed to write agentic workflow designer skill: %w", err)
	}

	if existingContent == "" {
		copilotAgentsLog.Printf("Created agentic workflow designer skill: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created agentic workflow designer skill: "+targetPath))
		}
	} else {
		copilotAgentsLog.Printf("Updated agentic workflow designer skill: %s", targetPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated agentic workflow designer skill: "+targetPath))
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

	if err := fileutil.EnsureParentDir(targetPath, constants.DirPermPublic); err != nil {
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
	return agenticWorkflowsAgentTemplate, nil
}

func buildAgenticWorkflowsSkillContent() (string, error) {
	awFiles, err := listAgenticWorkflowsMarkdownFiles(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch .github/aw markdown file list from github/gh-aw: %v. Falling back to embedded list.", err)))
		awFiles = embeddedFallbackAWMarkdownFiles()
	}
	sort.Strings(awFiles)
	if len(awFiles) == 0 {
		return "", errors.New("no .github/aw markdown files available from remote or embedded fallback")
	}

	var fileList strings.Builder
	for _, file := range awFiles {
		fmt.Fprintf(&fileList, "- `.github/aw/%s`\n", file)
	}

	if !strings.Contains(agenticWorkflowsSkillTemplate, agenticWorkflowsSkillFileListPlaceholder) {
		return "", fmt.Errorf("agentic workflows skill template is missing %s placeholder", agenticWorkflowsSkillFileListPlaceholder)
	}

	return strings.Replace(agenticWorkflowsSkillTemplate, agenticWorkflowsSkillFileListPlaceholder, fileList.String(), 1), nil
}

type gitHubRepositoryContentEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func fetchAgenticWorkflowsMarkdownFiles(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ghAWMarkdownFilesAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build github API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gh-aw")

	client := &http.Client{Timeout: constants.DefaultHTTPClientTimeout}
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("github API request timed out after %s: %w", constants.DefaultHTTPClientTimeout, err)
		}
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %s", resp.Status)
	}

	var entries []gitHubRepositoryContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode github API response: %w", err)
	}

	awFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "file" || !strings.HasSuffix(entry.Name, ".md") {
			continue
		}
		awFiles = append(awFiles, entry.Name)
	}

	if len(awFiles) == 0 {
		return nil, errors.New("github API returned no markdown files")
	}

	sort.Strings(awFiles)
	return awFiles, nil
}

func embeddedFallbackAWMarkdownFiles() []string {
	var awFiles []string
	if err := json.Unmarshal([]byte(agenticWorkflowsFallbackAWFiles), &awFiles); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse embedded .github/aw fallback markdown file list: %v", err)))
		return nil
	}
	sort.Strings(awFiles)
	return awFiles
}

// cleanupOldPromptFile removes an old prompt file from .github/prompts/ if it exists
func cleanupOldPromptFile(promptFileName string, verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return nil // Not in a git repository, skip
	}

	oldPath := filepath.Join(gitRoot, ".github", "prompts", promptFileName)

	// Check if the old file exists and remove it
	if fileutil.FileExists(oldPath) {
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
	if fileutil.FileExists(agentPath) {
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
		if fileutil.FileExists(path) {
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
			if fileutil.FileExists(path) {
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
