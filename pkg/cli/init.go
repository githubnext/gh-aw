package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var initLog = logger.New("cli:init")

// InitRepository initializes the repository for agentic workflows
func InitRepository(verbose bool, mcp bool, campaign bool, tokens bool, engine string, codespaceRepos []string, codespaceEnabled bool, completions bool, rootCmd CommandProvider) error {
	initLog.Print("Starting repository initialization for agentic workflows")

	// Ensure we're in a git repository
	if !isGitRepo() {
		initLog.Print("Not in a git repository, initialization failed")
		return fmt.Errorf("not in a git repository")
	}
	initLog.Print("Verified git repository")

	// Configure .gitattributes
	initLog.Print("Configuring .gitattributes")
	if err := ensureGitAttributes(); err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}

	// Ensure .github/aw/logs/.gitignore exists
	initLog.Print("Ensuring .github/aw/logs/.gitignore exists")
	if err := ensureLogsGitignore(); err != nil {
		initLog.Printf("Failed to ensure logs .gitignore: %v", err)
		return fmt.Errorf("failed to ensure logs .gitignore: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .github/aw/logs/.gitignore"))
	}

	// Write copilot instructions
	initLog.Print("Writing GitHub Copilot instructions")
	if err := ensureCopilotInstructions(verbose, false); err != nil {
		initLog.Printf("Failed to write copilot instructions: %v", err)
		return fmt.Errorf("failed to write copilot instructions: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created GitHub Copilot instructions"))
	}

	// Write dispatcher agent
	initLog.Print("Writing agentic workflows dispatcher agent")
	if err := ensureAgenticWorkflowsDispatcher(verbose, false); err != nil {
		initLog.Printf("Failed to write dispatcher agent: %v", err)
		return fmt.Errorf("failed to write dispatcher agent: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher agent"))
	}

	// Write create workflow prompt
	initLog.Print("Writing create workflow prompt")
	if err := ensureCreateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create workflow prompt: %v", err)
		return fmt.Errorf("failed to write create workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created create workflow prompt"))
	}

	// Write update workflow prompt (new)
	initLog.Print("Writing update workflow prompt")
	if err := ensureUpdateWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write update workflow prompt: %v", err)
		return fmt.Errorf("failed to write update workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created update workflow prompt"))
	}

	// Write create shared agentic workflow prompt
	initLog.Print("Writing create shared agentic workflow prompt")
	if err := ensureCreateSharedAgenticWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write create shared workflow prompt: %v", err)
		return fmt.Errorf("failed to write create shared workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created shared workflow creation prompt"))
	}

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	// Write debug workflow prompt
	initLog.Print("Writing debug workflow prompt")
	if err := ensureDebugWorkflowPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write debug workflow prompt: %v", err)
		return fmt.Errorf("failed to write debug workflow prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created debug workflow prompt"))
	}

	// Write upgrade agentic workflows prompt
	initLog.Print("Writing upgrade agentic workflows prompt")
	if err := ensureUpgradeAgenticWorkflowsPrompt(verbose, false); err != nil {
		initLog.Printf("Failed to write upgrade workflows prompt: %v", err)
		return fmt.Errorf("failed to write upgrade workflows prompt: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created upgrade workflows prompt"))
	}

	// Write campaign dispatcher agent if requested
	if campaign {
		initLog.Print("Writing campaign dispatcher agent")
		if err := ensureAgenticCampaignsDispatcher(verbose, false); err != nil {
			initLog.Printf("Failed to write campaign dispatcher agent: %v", err)
			return fmt.Errorf("failed to write campaign dispatcher agent: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign dispatcher agent"))
		}

		// Write campaign instruction files
		initLog.Print("Writing campaign instruction files")
		campaignEnsureFuncs := []struct {
			fn   func(bool, bool) error
			name string
		}{
			{ensureCampaignCreationInstructions, "campaign creation instructions"},
			{ensureCampaignOrchestratorInstructions, "campaign orchestrator instructions"},
			{ensureCampaignProjectUpdateInstructions, "campaign project update instructions"},
			{ensureCampaignWorkflowExecution, "campaign workflow execution"},
			{ensureCampaignClosingInstructions, "campaign closing instructions"},
			{ensureCampaignGeneratorInstructions, "campaign generator instructions"},
		}

		for _, item := range campaignEnsureFuncs {
			if err := item.fn(verbose, false); err != nil {
				initLog.Printf("Failed to write %s: %v", item.name, err)
				return fmt.Errorf("failed to write %s: %w", item.name, err)
			}
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created campaign instruction files"))
		}

		// Add campaign-generator workflow from gh-aw repository
		initLog.Print("Adding campaign-generator workflow")
		if err := addCampaignGeneratorWorkflow(verbose); err != nil {
			initLog.Printf("Failed to add campaign-generator workflow: %v", err)
			return fmt.Errorf("failed to add campaign-generator workflow: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added campaign-generator workflow"))
		}
	}

	// Configure MCP if requested
	if mcp {
		initLog.Print("Configuring GitHub Copilot Agent MCP integration")

		// Create copilot-setup-steps.yml
		if err := ensureCopilotSetupSteps(verbose); err != nil {
			initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
			return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))
		}

		// Create .vscode/mcp.json
		if err := ensureMCPConfig(verbose); err != nil {
			initLog.Printf("Failed to create MCP config: %v", err)
			return fmt.Errorf("failed to create MCP config: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .vscode/mcp.json"))
		}
	}

	// Configure Codespaces if requested
	if codespaceEnabled {
		initLog.Printf("Configuring GitHub Codespaces devcontainer with additional repos: %v", codespaceRepos)

		// Create or update .devcontainer/devcontainer.json
		if err := ensureDevcontainerConfig(verbose, codespaceRepos); err != nil {
			initLog.Printf("Failed to configure devcontainer: %v", err)
			return fmt.Errorf("failed to configure devcontainer: %w", err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .devcontainer/devcontainer.json"))
		}
	}

	// Configure VSCode settings
	initLog.Print("Configuring VSCode settings")

	// Update .vscode/settings.json
	if err := ensureVSCodeSettings(verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))
	}

	// Validate tokens if requested
	if tokens {
		initLog.Print("Validating repository secrets for agentic workflows")
		fmt.Fprintln(os.Stderr, "")

		// Run token bootstrap validation
		if err := runTokensBootstrap(engine, "", ""); err != nil {
			initLog.Printf("Token validation failed: %v", err)
			// Don't fail init if token validation has issues
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Token validation encountered an issue: %v", err)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Install shell completions if requested
	if completions {
		initLog.Print("Installing shell completions")
		fmt.Fprintln(os.Stderr, "")

		if err := InstallShellCompletion(verbose, rootCmd); err != nil {
			initLog.Printf("Shell completion installation failed: %v", err)
			// Don't fail init if completion installation has issues
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Shell completion installation encountered an issue: %v", err)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Generate/update maintenance workflow if any workflows use expires field
	initLog.Print("Checking for workflows with expires field to generate maintenance workflow")
	if err := ensureMaintenanceWorkflow(verbose); err != nil {
		initLog.Printf("Failed to generate maintenance workflow: %v", err)
		// Don't fail init if maintenance workflow generation has issues
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
	}

	initLog.Print("Repository initialization completed successfully")

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if mcp {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	if len(codespaceRepos) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Codespaces devcontainer configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	if tokens {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To configure missing secrets, use: gh aw secret set <secret-name> --owner <owner> --repo <repo>"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select agentic-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// addCampaignGeneratorWorkflow generates and compiles the campaign-generator workflow
func addCampaignGeneratorWorkflow(verbose bool) error {
	initLog.Print("Generating campaign-generator workflow")

	// Get the git root directory
	gitRoot, err := findGitRoot()
	if err != nil {
		initLog.Printf("Failed to find git root: %v", err)
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// The runnable artifact is the compiled lock file, which must be in .github/workflows.
	// Keep the markdown source in .github/aw to match other gh-aw prompts.
	workflowsDir := filepath.Join(gitRoot, ".github", "workflows")
	awDir := filepath.Join(gitRoot, ".github", "aw")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		initLog.Printf("Failed to create workflows directory: %v", err)
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}
	if err := os.MkdirAll(awDir, 0755); err != nil {
		initLog.Printf("Failed to create .github/aw directory: %v", err)
		return fmt.Errorf("failed to create .github/aw directory: %w", err)
	}

	// Build the campaign-generator workflow
	data := campaign.BuildCampaignGenerator()
	workflowPath := filepath.Join(awDir, "campaign-generator.md")

	// Render the workflow to markdown
	content := renderCampaignGeneratorMarkdown(data)

	// Write markdown file with restrictive permissions
	if err := os.WriteFile(workflowPath, []byte(content), 0600); err != nil {
		initLog.Printf("Failed to write campaign-generator.md: %v", err)
		return fmt.Errorf("failed to write campaign-generator.md: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Created campaign-generator workflow: %s\n", workflowPath)
	}

	// Compile to lock file using the standard compiler.
	// campaign-generator.md lives in .github/aw, but MarkdownToLockFile is
	// intentionally mapped to emit the runnable lock file into .github/workflows.
	compiler := workflow.NewCompiler(verbose, "", GetVersion())
	if err := CompileWorkflowWithValidation(compiler, workflowPath, verbose, false, false, false, false, false); err != nil {
		initLog.Printf("Failed to compile campaign-generator: %v", err)
		return fmt.Errorf("failed to compile campaign-generator: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Compiled campaign-generator workflow\n")
	}

	initLog.Print("Campaign-generator workflow generated successfully")
	return nil
}

// renderCampaignGeneratorMarkdown converts WorkflowData to markdown format for campaign-generator
func renderCampaignGeneratorMarkdown(data *workflow.WorkflowData) string {
	var b strings.Builder

	b.WriteString("---\n")
	if strings.TrimSpace(data.Name) != "" {
		fmt.Fprintf(&b, "name: %q\n", data.Name)
	}
	if strings.TrimSpace(data.Description) != "" {
		fmt.Fprintf(&b, "description: %q\n", data.Description)
	}
	if strings.TrimSpace(data.On) != "" {
		b.WriteString(strings.TrimSuffix(data.On, "\n"))
		b.WriteString("\n")
	}
	if strings.TrimSpace(data.Permissions) != "" {
		b.WriteString(strings.TrimSuffix(data.Permissions, "\n"))
		b.WriteString("\n")
	}

	// Engine configuration
	engineID := "copilot"
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	}
	fmt.Fprintf(&b, "engine: %s\n", engineID)

	// Tools
	if len(data.Tools) > 0 {
		b.WriteString("tools:\n")
		if gh, ok := data.Tools["github"].(map[string]any); ok {
			b.WriteString("  github:\n")
			if toolsets, ok := gh["toolsets"].([]any); ok {
				b.WriteString("    toolsets: [")
				for i, ts := range toolsets {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%v", ts)
				}
				b.WriteString("]\n")
			}
		}
	}

	// Safe outputs
	if data.SafeOutputs != nil {
		b.WriteString("safe-outputs:\n")

		if data.SafeOutputs.AddComments != nil {
			b.WriteString("  add-comment:\n")
			b.WriteString("    max: 10\n")
		}

		if data.SafeOutputs.UpdateIssues != nil {
			b.WriteString("  update-issue:\n")
		}

		if data.SafeOutputs.AssignToAgent != nil {
			b.WriteString("  assign-to-agent:\n")
		}

		if data.SafeOutputs.CreateProjects != nil {
			b.WriteString("  create-project:\n")
			b.WriteString("    max: 1\n")
			if data.SafeOutputs.CreateProjects.GitHubToken != "" {
				fmt.Fprintf(&b, "    github-token: \"%s\"\n", data.SafeOutputs.CreateProjects.GitHubToken)
			}
		}

		if data.SafeOutputs.UpdateProjects != nil {
			b.WriteString("  update-project:\n")
			b.WriteString("    max: 10\n")
			if data.SafeOutputs.UpdateProjects.GitHubToken != "" {
				fmt.Fprintf(&b, "    github-token: \"%s\"\n", data.SafeOutputs.UpdateProjects.GitHubToken)
			}
			if len(data.SafeOutputs.UpdateProjects.Views) > 0 {
				b.WriteString("    views:\n")
				for _, view := range data.SafeOutputs.UpdateProjects.Views {
					fmt.Fprintf(&b, "      - name: \"%s\"\n", view.Name)
					fmt.Fprintf(&b, "        layout: \"%s\"\n", view.Layout)
					fmt.Fprintf(&b, "        filter: \"%s\"\n", view.Filter)
				}
			}
		}

		if data.SafeOutputs.Messages != nil {
			b.WriteString("  messages:\n")
			if data.SafeOutputs.Messages.Footer != "" {
				fmt.Fprintf(&b, "    footer: \"%s\"\n", data.SafeOutputs.Messages.Footer)
			}
			if data.SafeOutputs.Messages.RunStarted != "" {
				fmt.Fprintf(&b, "    run-started: \"%s\"\n", data.SafeOutputs.Messages.RunStarted)
			}
			if data.SafeOutputs.Messages.RunSuccess != "" {
				fmt.Fprintf(&b, "    run-success: \"%s\"\n", data.SafeOutputs.Messages.RunSuccess)
			}
			if data.SafeOutputs.Messages.RunFailure != "" {
				fmt.Fprintf(&b, "    run-failure: \"%s\"\n", data.SafeOutputs.Messages.RunFailure)
			}
		}
	}

	if strings.TrimSpace(data.TimeoutMinutes) != "" {
		fmt.Fprintf(&b, "timeout-minutes: %s\n", data.TimeoutMinutes)
	}

	b.WriteString("---\n\n")

	// Write the prompt/body
	if strings.TrimSpace(data.MarkdownContent) != "" {
		b.WriteString(data.MarkdownContent)
	}

	return b.String()
}

// ensureMaintenanceWorkflow checks existing workflows for expires field and generates/updates
// the maintenance workflow file if any workflows use it
func ensureMaintenanceWorkflow(verbose bool) error {
	initLog.Print("Checking for workflows with expires field")

	// Find git root
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Determine the workflows directory
	workflowsDir := filepath.Join(gitRoot, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		// No workflows directory yet, skip maintenance workflow generation
		initLog.Print("No workflows directory found, skipping maintenance workflow generation")
		return nil
	}

	// Find all workflow markdown files
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find workflow files: %w", err)
	}

	// Create a compiler to parse workflows
	compiler := workflow.NewCompiler(false, "", GetVersion())

	// Parse all workflows to collect WorkflowData
	var workflowDataList []*workflow.WorkflowData
	for _, file := range files {
		// Skip campaign specs and generated files
		if strings.HasSuffix(file, ".campaign.md") || strings.HasSuffix(file, ".campaign.g.md") {
			continue
		}

		initLog.Printf("Parsing workflow: %s", file)
		workflowData, err := compiler.ParseWorkflowFile(file)
		if err != nil {
			// Ignore parse errors - workflows might be incomplete during init
			initLog.Printf("Skipping workflow %s due to parse error: %v", file, err)
			continue
		}

		workflowDataList = append(workflowDataList, workflowData)
	}

	// Always call GenerateMaintenanceWorkflow even with empty list
	// This allows it to delete existing maintenance workflow if no workflows have expires
	initLog.Printf("Generating maintenance workflow for %d workflows", len(workflowDataList))
	if err := workflow.GenerateMaintenanceWorkflow(workflowDataList, workflowsDir, GetVersion(), compiler.GetActionMode(), verbose); err != nil {
		return fmt.Errorf("failed to generate maintenance workflow: %w", err)
	}

	if verbose && len(workflowDataList) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Generated/updated maintenance workflow"))
	}

	return nil
}
