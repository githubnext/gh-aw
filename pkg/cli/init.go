package cli

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var initLog = logger.New("cli:init")

// InitRepositoryDefault initializes the repository for agentic workflows with default settings.
// This is used when no flags are provided and always configures for Copilot with MCP support.
func InitRepositoryDefault(verbose bool, rootCmd CommandProvider) error {
	initLog.Print("Starting repository initialization")

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Initializing repository for GitHub Agentic Workflows..."))
	fmt.Fprintln(os.Stderr, "")

	// Initialize repository with basic settings
	if err := initializeBasicRepository(verbose); err != nil {
		return err
	}

	// Always configure Copilot MCP
	initLog.Print("Configuring GitHub Copilot Agent MCP integration")

	// Detect action mode for setup steps generation
	actionMode := workflow.DetectActionMode(GetVersion())
	initLog.Printf("Using action mode for copilot-setup-steps.yml: %s", actionMode)

	// Create copilot-setup-steps.yml
	if err := ensureCopilotSetupSteps(verbose, actionMode, GetVersion()); err != nil {
		initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
		return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))

	// Create .vscode/mcp.json
	if err := ensureMCPConfig(verbose); err != nil {
		initLog.Printf("Failed to create MCP config: %v", err)
		return fmt.Errorf("failed to create MCP config: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .vscode/mcp.json"))

	// Configure VSCode settings
	initLog.Print("Configuring VSCode settings")
	if err := ensureVSCodeSettings(verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))

	// Display success message
	initLog.Print("Repository initialization completed successfully")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select agentic-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// initializeBasicRepository sets up the basic repository structure
func initializeBasicRepository(verbose bool) error {
	// Configure .gitattributes
	initLog.Print("Configuring .gitattributes")
	if err := ensureGitAttributes(); err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
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

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	return nil
}

// InitOptions contains all configuration options for repository initialization
type InitOptions struct {
	Verbose          bool
	MCP              bool
	CodespaceRepos   []string
	CodespaceEnabled bool
	Completions      bool
	Push             bool
	CreatePR         bool
	RootCmd          CommandProvider
}

// InitRepository initializes the repository for agentic workflows
func InitRepository(opts InitOptions) error {
	initLog.Print("Starting repository initialization for agentic workflows")

	// If --push or --create-pull-request is enabled, ensure git status is clean before starting
	if opts.Push || opts.CreatePR {
		if opts.CreatePR {
			initLog.Print("Checking for clean working directory (--create-pull-request enabled)")
		} else {
			initLog.Print("Checking for clean working directory (--push enabled)")
		}
		if err := checkCleanWorkingDirectory(opts.Verbose); err != nil {
			initLog.Printf("Git status check failed: %v", err)
			if opts.CreatePR {
				return fmt.Errorf("--create-pull-request requires a clean working directory: %w", err)
			}
			return fmt.Errorf("--push requires a clean working directory: %w", err)
		}
	}

	// If creating a PR, check GitHub CLI is available
	if opts.CreatePR {
		if !isGHCLIAvailable() {
			return fmt.Errorf("GitHub CLI (gh) is required for PR creation but not available")
		}
	}

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
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}

	// Write dispatcher agent
	initLog.Print("Writing agentic workflows dispatcher agent")
	if err := ensureAgenticWorkflowsDispatcher(opts.Verbose, false); err != nil {
		initLog.Printf("Failed to write dispatcher agent: %v", err)
		return fmt.Errorf("failed to write dispatcher agent: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher agent"))
	}

	// Delete existing setup agentic workflows agent if it exists
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(opts.Verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}

	// Configure MCP if requested
	if opts.MCP {
		initLog.Print("Configuring GitHub Copilot Agent MCP integration")

		// Detect action mode for setup steps generation
		actionMode := workflow.DetectActionMode(GetVersion())
		initLog.Printf("Using action mode for copilot-setup-steps.yml: %s", actionMode)

		// Create copilot-setup-steps.yml
		if err := ensureCopilotSetupSteps(opts.Verbose, actionMode, GetVersion()); err != nil {
			initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
			return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))
		}

		// Create .vscode/mcp.json
		if err := ensureMCPConfig(opts.Verbose); err != nil {
			initLog.Printf("Failed to create MCP config: %v", err)
			return fmt.Errorf("failed to create MCP config: %w", err)
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .vscode/mcp.json"))
		}
	}

	// Configure Codespaces if requested
	if opts.CodespaceEnabled {
		initLog.Printf("Configuring GitHub Codespaces devcontainer with additional repos: %v", opts.CodespaceRepos)

		// Create or update .devcontainer/devcontainer.json
		if err := ensureDevcontainerConfig(opts.Verbose, opts.CodespaceRepos); err != nil {
			initLog.Printf("Failed to configure devcontainer: %v", err)
			return fmt.Errorf("failed to configure devcontainer: %w", err)
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .devcontainer/devcontainer.json"))
		}
	}

	// Configure VSCode settings
	initLog.Print("Configuring VSCode settings")

	// Update .vscode/settings.json
	if err := ensureVSCodeSettings(opts.Verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))
	}

	// Install shell completions if requested
	if opts.Completions {
		initLog.Print("Installing shell completions")
		fmt.Fprintln(os.Stderr, "")

		if err := InstallShellCompletion(opts.Verbose, opts.RootCmd); err != nil {
			initLog.Printf("Shell completion installation failed: %v", err)
			// Don't fail init if completion installation has issues
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Shell completion installation encountered an issue: %v", err)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Generate/update maintenance workflow if any workflows use expires field
	initLog.Print("Checking for workflows with expires field to generate maintenance workflow")
	if err := ensureMaintenanceWorkflow(opts.Verbose); err != nil {
		initLog.Printf("Failed to generate maintenance workflow: %v", err)
		// Don't fail init if maintenance workflow generation has issues
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
	}

	initLog.Print("Repository initialization completed successfully")

	// If --create-pull-request is enabled, create branch, commit, push, and create PR
	if opts.CreatePR {
		initLog.Print("Create PR enabled - preparing to create branch, commit, push, and create PR")
		fmt.Fprintln(os.Stderr, "")

		// Get current branch for restoration later
		currentBranch, err := getCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Create temporary branch
		branchName := fmt.Sprintf("init-agentic-workflows-%d", rand.Intn(9000)+1000)
		if err := createAndSwitchBranch(branchName, opts.Verbose); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}

		// Commit changes
		commitMessage := "chore: initialize agentic workflows"
		if err := commitChanges(commitMessage, opts.Verbose); err != nil {
			// Switch back to original branch before returning error
			_ = switchBranch(currentBranch, opts.Verbose)
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		// Push branch
		if err := pushBranch(branchName, opts.Verbose); err != nil {
			// Switch back to original branch before returning error
			_ = switchBranch(currentBranch, opts.Verbose)
			return fmt.Errorf("failed to push branch %s: %w", branchName, err)
		}

		// Create PR
		prTitle := "Initialize agentic workflows"
		prBody := "This PR initializes the repository for agentic workflows by:\n" +
			"- Configuring .gitattributes\n" +
			"- Creating GitHub Copilot custom instructions\n" +
			"- Setting up workflow prompts and agents"
		if _, _, err := createPR(branchName, prTitle, prBody, opts.Verbose); err != nil {
			// Switch back to original branch before returning error
			_ = switchBranch(currentBranch, opts.Verbose)
			return fmt.Errorf("failed to create PR: %w", err)
		}

		// Switch back to original branch
		if err := switchBranch(currentBranch, opts.Verbose); err != nil {
			return fmt.Errorf("failed to switch back to branch %s: %w", currentBranch, err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created PR for initialization"))
	} else if opts.Push {
		// If --push is enabled, commit and push changes
		initLog.Print("Push enabled - preparing to commit and push changes")
		fmt.Fprintln(os.Stderr, "")

		// Check if we're on the default branch
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking current branch..."))
		if err := checkOnDefaultBranch(opts.Verbose); err != nil {
			initLog.Printf("Default branch check failed: %v", err)
			return fmt.Errorf("cannot push: %w", err)
		}

		// Confirm with user (skip in CI)
		if err := confirmPushOperation(opts.Verbose); err != nil {
			initLog.Printf("Push operation not confirmed: %v", err)
			return fmt.Errorf("push operation cancelled: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Preparing to commit and push changes..."))

		// Use the helper function to orchestrate the full workflow
		commitMessage := "chore: initialize agentic workflows"
		if err := commitAndPushChanges(commitMessage, opts.Verbose); err != nil {
			// Check if it's the "no changes" case
			hasChanges, checkErr := hasChangesToCommit()
			if checkErr == nil && !hasChanges {
				initLog.Print("No changes to commit")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes to commit"))
			} else {
				return err
			}
		} else {
			// Print success messages based on whether remote exists
			fmt.Fprintln(os.Stderr, "")
			if hasRemote() {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes pushed to remote"))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Changes committed locally (no remote configured)"))
			}
		}
	}

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if opts.MCP {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Copilot Agent MCP integration configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	if len(opts.CodespaceRepos) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Codespaces devcontainer configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, launch Copilot CLI: npx @github/copilot"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then type /agent and select agentic-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add workflows from the catalog: "+string(constants.CLIExtensionPrefix)+" add <workflow-name>"))
	fmt.Fprintln(os.Stderr, "")

	return nil
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

	// Filter out README.md files
	files = filterWorkflowFiles(files)

	// Create a compiler to parse workflows (version and action mode auto-detected)
	compiler := workflow.NewCompiler()
	initLog.Printf("Action mode detected for maintenance workflow: %s", compiler.GetActionMode())

	// Parse all workflows to collect WorkflowData
	var workflowDataList []*workflow.WorkflowData
	for _, file := range files {
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
