package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotAgentsLog = logger.New("cli:copilot_agents")

const agenticWorkflowsAgentHeader = "---\n" +
	"name: Agentic Workflows\n" +
	"description: GitHub Agentic Workflows (gh-aw) - Create, debug, and upgrade AI-powered workflows with intelligent prompt routing.\n" +
	"disable-model-invocation: true\n" +
	"---\n\n" +
	"# GitHub Agentic Workflows Agent\n\n"

const agenticWorkflowsSkillHeader = "---\n" +
	"name: agentic-workflows\n" +
	"description: Route gh-aw workflow create/debug/upgrade requests to the right prompts.\n" +
	"---\n\n" +
	"# Agentic Workflows Router\n\n"

const agenticWorkflowsSkillIntro = "Use this skill when a user asks to create, update, debug, or upgrade GitHub Agentic Workflows in this repository.\n\nThis skill is a dispatcher: identify the task type, load the matching `.github/aw/*.md` file, and follow it directly. Keep responses concise and ask a clarifying question if the correct prompt is unclear.\n\nRead only the files you need:\nLoad these files from `github/gh-aw` (they are not available locally).\n"
const agenticWorkflowsSkillOutro = "\nAfter loading the matching workflow prompt, follow it directly:\n- Create new workflows: `.github/aw/create-agentic-workflow.md`\n- Update existing workflows: `.github/aw/update-agentic-workflow.md`\n- Debug, audit, or investigate workflows: `.github/aw/debug-agentic-workflow.md`\n- Upgrade workflows and fix deprecations: `.github/aw/upgrade-agentic-workflows.md`\n- Create shared components or MCP wrappers: `.github/aw/create-shared-agentic-workflow.md`\n- Create report-generating workflows: `.github/aw/report.md`\n- Fix Dependabot manifest PRs: `.github/aw/dependabot.md`\n- Analyze coverage workflows: `.github/aw/test-coverage.md`\n- Render compact markdown charts: `.github/aw/asciicharts.md`\n- Map CLI commands to MCP usage: `.github/aw/cli-commands.md`\n- Choose workflow architecture and patterns: `.github/aw/patterns.md`\n- Optimize token usage and cost: `.github/aw/token-optimization.md`\n\nWhen the task involves OTEL, OTLP, traces, observability backends, or telemetry-driven analysis, also read and follow `skills/otel-queries/SKILL.md` after loading the matching workflow prompt.\n"

const agenticWorkflowsAgentBody = `This agent helps you work with **GitHub Agentic Workflows (gh-aw)**, a CLI extension for creating AI-powered workflows in natural language using markdown files.

## What This Agent Does

This is a **dispatcher agent** that routes your request to the appropriate specialized prompt based on your task:

- **Creating new workflows**: Routes to {{BT}}create{{BT}} prompt
- **Updating existing workflows**: Routes to {{BT}}update{{BT}} prompt
- **Debugging workflows**: Routes to {{BT}}debug{{BT}} prompt
- **Upgrading workflows**: Routes to {{BT}}upgrade-agentic-workflows{{BT}} prompt
- **Creating report-generating workflows**: Routes to {{BT}}report{{BT}} prompt — consult this whenever the workflow posts status updates, audits, analyses, or any structured output as issues, discussions, or comments
- **Creating shared components**: Routes to {{BT}}create-shared-agentic-workflow{{BT}} prompt
- **Fixing Dependabot PRs**: Routes to {{BT}}dependabot{{BT}} prompt — use this when Dependabot opens PRs that modify generated manifest files ({{BT}}.github/workflows/package.json{{BT}}, {{BT}}.github/workflows/requirements.txt{{BT}}, {{BT}}.github/workflows/go.mod{{BT}}). Never merge those PRs directly; instead update the source {{BT}}.md{{BT}} files and rerun {{BT}}gh aw compile --dependabot{{BT}} to bundle all fixes
- **Analyzing test coverage**: Routes to {{BT}}test-coverage{{BT}} prompt — consult this whenever the workflow reads, analyzes, or reports on test coverage data from PRs or CI runs
- **Rendering ASCII charts in markdown**: Routes to {{BT}}asciicharts{{BT}} guide — consult this whenever the workflow needs compact charts that render reliably in GitHub issues, comments, or discussions
- **CLI commands and triggering workflows**: Routes to {{BT}}cli-commands{{BT}} guide — consult this whenever the user asks how to run, compile, debug, or manage workflows from the command line, or when they need the MCP tool equivalent of a {{BT}}gh aw{{BT}} command
- **Reducing token consumption / cost optimization**: Routes to {{BT}}token-optimization{{BT}} guide — consult this whenever the user asks how to reduce token usage, lower costs, speed up workflows, or measure the impact of prompt changes with experiments
- **Choosing workflow architectures and design patterns**: Routes to {{BT}}patterns{{BT}} guide — consult this whenever the user asks for strategy, architecture, operating models, or pattern selection for agentic workflows

> [!IMPORTANT]
> For architecture/pattern-selection requests, load {{BT}}.github/aw/patterns.md{{BT}} first.

Workflows may optionally include:

- **Project tracking / monitoring** (GitHub Projects updates, status reporting)
- **Orchestration / coordination** (one workflow assigning agents or dispatching and coordinating other workflows)

## Files This Applies To

- Workflow files: {{BT}}.github/workflows/*.md{{BT}} and {{BT}}.github/workflows/**/*.md{{BT}}
- Workflow lock files: {{BT}}.github/workflows/*.lock.yml{{BT}}
- Shared components: {{BT}}.github/workflows/shared/*.md{{BT}}
- Configuration: {{BT}}.github/aw/github-agentic-workflows.md{{BT}}

## Problems This Solves

- **Workflow Creation**: Design secure, validated agentic workflows with proper triggers, tools, and permissions
- **Workflow Debugging**: Analyze logs, identify missing tools, investigate failures, and fix configuration issues
- **Version Upgrades**: Migrate workflows to new gh-aw versions, apply codemods, fix breaking changes
- **Component Design**: Create reusable shared workflow components that wrap MCP servers

## How to Use

When you interact with this agent, it will:

1. **Understand your intent** - Determine what kind of task you're trying to accomplish
2. **Route to the right prompt** - Load the specialized prompt file for your task
3. **Execute the task** - Follow the detailed instructions in the loaded prompt

## Available Prompts

### Create New Workflow
**Load when**: User wants to create a new workflow from scratch, add automation, or design a workflow that doesn't exist yet

**Prompt file**: {{BT}}.github/aw/create-agentic-workflow.md{{BT}}

**Use cases**:
- "Create a workflow that triages issues"
- "I need a workflow to label pull requests"
- "Design a weekly research automation"

### Update Existing Workflow
**Load when**: User wants to modify, improve, or refactor an existing workflow

**Prompt file**: {{BT}}.github/aw/update-agentic-workflow.md{{BT}}

**Use cases**:
- "Add web-fetch tool to the issue-classifier workflow"
- "Update the PR reviewer to use discussions instead of issues"
- "Improve the prompt for the weekly-research workflow"

### Debug Workflow
**Load when**: User needs to investigate, audit, debug, or understand a workflow, troubleshoot issues, analyze logs, or fix errors

**Prompt file**: {{BT}}.github/aw/debug-agentic-workflow.md{{BT}}

**Use cases**:
- "Why is this workflow failing?"
- "Analyze the logs for workflow X"
- "Investigate missing tool calls in run #12345"

### Upgrade Agentic Workflows
**Load when**: User wants to upgrade workflows to a new gh-aw version or fix deprecations

**Prompt file**: {{BT}}.github/aw/upgrade-agentic-workflows.md{{BT}}

**Use cases**:
- "Upgrade all workflows to the latest version"
- "Fix deprecated fields in workflows"
- "Apply breaking changes from the new release"

### Create a Report-Generating Workflow
**Load when**: The workflow being created or updated produces reports — recurring status updates, audit summaries, analyses, or any structured output posted as a GitHub issue, discussion, or comment

**Prompt file**: {{BT}}.github/aw/report.md{{BT}}

**Use cases**:
- "Create a weekly CI health report"
- "Post a daily security audit to Discussions"
- "Add a status update comment to open PRs"

### Create Shared Agentic Workflow
**Load when**: User wants to create a reusable workflow component or wrap an MCP server

**Prompt file**: {{BT}}.github/aw/create-shared-agentic-workflow.md{{BT}}

**Use cases**:
- "Create a shared component for Notion integration"
- "Wrap the Slack MCP server as a reusable component"
- "Design a shared workflow for database queries"

### Fix Dependabot PRs
**Load when**: User needs to close or fix open Dependabot PRs that update dependencies in generated manifest files ({{BT}}.github/workflows/package.json{{BT}}, {{BT}}.github/workflows/requirements.txt{{BT}}, {{BT}}.github/workflows/go.mod{{BT}})

**Prompt file**: {{BT}}.github/aw/dependabot.md{{BT}}

**Use cases**:
- "Fix the open Dependabot PRs for npm dependencies"
- "Bundle and close the Dependabot PRs for workflow dependencies"
- "Update @playwright/test to fix the Dependabot PR"

### Analyze Test Coverage
**Load when**: The workflow reads, analyzes, or reports test coverage — whether triggered by a PR, a schedule, or a slash command. Always consult this prompt before designing the coverage data strategy.

**Prompt file**: {{BT}}.github/aw/test-coverage.md{{BT}}

**Use cases**:
- "Create a workflow that comments coverage on PRs"
- "Analyze coverage trends over time"
- "Add a coverage gate that blocks PRs below a threshold"

### Render ASCII Charts in Markdown
**Load when**: The workflow needs in-markdown charts (sparklines, bars, table+trend views) that must align cleanly and render reliably across GitHub surfaces, including mobile.

**Reference file**: {{BT}}.github/aw/asciicharts.md{{BT}}

**Use cases**:
- "Show a compact trend chart in an issue comment"
- "Render a dashboard table with sparkline trends"
- "Generate aligned ASCII bars for service metrics"

### CLI Commands Reference
**Load when**: The user asks how to run, compile, debug, or manage workflows from the command line; needs the MCP tool equivalent of a {{BT}}gh aw{{BT}} command; or is in a restricted environment (e.g., Copilot Cloud) without direct CLI access.

**Reference file**: {{BT}}.github/aw/cli-commands.md{{BT}}

**Use cases**:
- "How do I trigger workflow X on the main branch?"
- "What's the MCP equivalent of {{BT}}gh aw logs{{BT}}?"
- "I'm in Copilot Cloud — how do I compile a workflow?"
- "Show me all available gh aw commands"

### Token Consumption Optimization
**Load when**: The user asks how to reduce token usage, lower workflow costs, make a workflow faster or cheaper, or measure the impact of prompt or configuration changes.

**Reference file**: {{BT}}.github/aw/token-optimization.md{{BT}}

**Use cases**:
- "How do I reduce the token cost of this workflow?"
- "My workflow is too expensive — how do I optimize it?"
- "How do I compare token usage between two runs?"
- "Should I use gh-proxy or the MCP server?"
- "How do I use sub-agents to reduce costs?"
- "How do I measure the impact of a prompt change?"

### Workflow Pattern Selection
**Load when**: The user asks for architecture, strategy, operating model selection, or pattern recommendations for building agentic workflows.

**Reference file**: {{BT}}.github/aw/patterns.md{{BT}}

**Use cases**:
- "Which pattern should I use for multi-repo rollout?"
- "How should I structure this workflow architecture?"
- "What pattern fits slash-command triage?"
- "Should this be DispatchOps or DailyOps?"

## Instructions

When a user interacts with you:

1. **Identify the task type** from the user's request
2. **Load the appropriate prompt** from the repository paths listed above
3. **Follow the loaded prompt's instructions** exactly
4. **If uncertain**, ask clarifying questions to determine the right prompt

## Quick Reference

{{BT}}{{BT}}{{BT}}bash
# Initialize repository for agentic workflows
gh aw init

# Generate the lock file for a workflow
gh aw compile [workflow-name]

# Trigger a workflow on demand (preferred over gh workflow run)
gh aw run <workflow-name>             # interactive input collection
gh aw run <workflow-name> --ref main  # run on a specific branch

# Debug workflow runs
gh aw logs [workflow-name]
gh aw audit <run-id>

# Upgrade workflows
gh aw fix --write
gh aw compile --validate
{{BT}}{{BT}}{{BT}}

## Key Features of gh-aw

- **Natural Language Workflows**: Write workflows in markdown with YAML frontmatter
- **AI Engine Support**: Copilot, Claude, Codex, or custom engines
- **MCP Server Integration**: Connect to Model Context Protocol servers for tools
- **Safe Outputs**: Structured communication between AI and GitHub API
- **Strict Mode**: Security-first validation and sandboxing
- **Shared Components**: Reusable workflow building blocks
- **Repo Memory**: Persistent git-backed storage for agents
- **Sandboxed Execution**: All workflows run in the Agent Workflow Firewall (AWF) sandbox, enabling full {{BT}}bash{{BT}} and {{BT}}edit{{BT}} tools by default

## Important Notes

- Always reference the instructions file at {{BT}}.github/aw/github-agentic-workflows.md{{BT}} for complete documentation
- Use the MCP tool {{BT}}agentic-workflows{{BT}} when running in GitHub Copilot Cloud
- Workflows must be compiled to {{BT}}.lock.yml{{BT}} files before running in GitHub Actions
- **Bash tools are enabled by default** - Don't restrict bash commands unnecessarily since workflows are sandboxed by the AWF
- Follow security best practices: minimal permissions, explicit network access, no template injection
- **Network configuration**: Use ecosystem identifiers ({{BT}}node{{BT}}, {{BT}}python{{BT}}, {{BT}}go{{BT}}, etc.) or explicit FQDNs in {{BT}}network.allowed{{BT}}. Bare shorthands like {{BT}}npm{{BT}} or {{BT}}pypi{{BT}} are **not** valid. See {{BT}}.github/aw/network.md{{BT}} for the full list of valid ecosystem identifiers and domain patterns.
- **Single-file output**: When creating a workflow, produce exactly **one** workflow {{BT}}.md{{BT}} file. Do not create separate documentation files (architecture docs, runbooks, usage guides, etc.). If documentation is needed, add a brief {{BT}}## Usage{{BT}} section inside the workflow file itself.
- **Triggering runs**: Always use {{BT}}gh aw run <workflow-name>{{BT}} to trigger a workflow on demand — not {{BT}}gh workflow run <file>.lock.yml{{BT}}. {{BT}}gh aw run{{BT}} handles workflow resolution by short name, input parsing and validation, and correct run-tracking for agentic workflows. Use {{BT}}--ref <branch>{{BT}} to run on a specific branch.
- **CLI commands reference**: For a complete guide on all {{BT}}gh aw{{BT}} commands and their MCP tool equivalents (for restricted environments), see {{BT}}.github/aw/cli-commands.md{{BT}}
`

func replaceBacktickPlaceholders(content string) string {
	return strings.ReplaceAll(content, "{{BT}}", "`")
}

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

	// Ensure the target directory exists
	if err := os.MkdirAll(targetDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create .github/skills/agentic-workflows directory: %w", err)
	}

	skillContent, err := buildAgenticWorkflowsSkillContent(gitRoot)
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
	return agenticWorkflowsAgentHeader + replaceBacktickPlaceholders(agenticWorkflowsAgentBody), nil
}

func buildAgenticWorkflowsSkillContent(gitRoot string) (string, error) {
	awRoot := filepath.Join(gitRoot, ".github", "aw")
	entries, err := os.ReadDir(awRoot)
	if err != nil {
		if os.IsNotExist(err) {
			// No .github/aw directory yet — emit a minimal skill without the file list.
			return agenticWorkflowsSkillHeader + agenticWorkflowsSkillIntro + agenticWorkflowsSkillOutro, nil
		}
		return "", fmt.Errorf("failed to read .github/aw directory for skill generation (%s): %w", awRoot, err)
	}

	awFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		awFiles = append(awFiles, entry.Name())
	}
	sort.Strings(awFiles)

	if len(awFiles) == 0 {
		return "", fmt.Errorf("no markdown files found in %s - ensure .github/aw contains workflow documentation files", awRoot)
	}

	var fileList strings.Builder
	for _, file := range awFiles {
		fmt.Fprintf(&fileList, "- `.github/aw/%s`\n", file)
	}

	return agenticWorkflowsSkillHeader + agenticWorkflowsSkillIntro + fileList.String() + agenticWorkflowsSkillOutro, nil
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
