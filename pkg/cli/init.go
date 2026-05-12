package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var initLog = logger.New("cli:init")

// InitOptions contains all configuration options for repository initialization
type InitOptions struct {
	Verbose          bool
	MCP              bool
	CodespaceRepos   []string
	CodespaceEnabled bool
	Completions      bool
	CreatePR         bool
	RootCmd          CommandProvider
}

// InitRepository initializes the repository for agentic workflows
func InitRepository(opts InitOptions) error {
	initLog.Print("Starting repository initialization for agentic workflows")

	// Show welcome banner for interactive mode
	console.ShowWelcomeBanner("This tool will initialize your repository for GitHub Agentic Workflows.")

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting up repository..."))
	fmt.Fprintln(os.Stderr, "")

	// If --create-pull-request is enabled, run pre-flight checks before doing any work
	if opts.CreatePR {
		if err := PreflightCheckForCreatePR(opts.Verbose); err != nil {
			return err
		}
	}

	// Ensure we're in a git repository
	if !isGitRepo() {
		initLog.Print("Not in a git repository, initialization failed")
		return errors.New("not in a git repository")
	}
	initLog.Print("Verified git repository")

	// Auto-detect GHES deployment and configure aw.json ghes: true when needed.
	if _, err := ensureGHESRepoConfig(opts.Verbose); err != nil {
		initLog.Printf("Failed to configure GHES repo config: %v", err)
		// Non-fatal: continue with the rest of init
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to configure GHES repo config: %v", err)))
	}

	// Configure .gitattributes
	initLog.Print("Configuring .gitattributes")
	if updated, err := ensureGitAttributes(); err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	} else if updated && opts.Verbose {
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

		// Create .github/mcp.json
		if err := ensureMCPConfig(opts.Verbose); err != nil {
			initLog.Printf("Failed to create MCP config: %v", err)
			return fmt.Errorf("failed to create MCP config: %w", err)
		}
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/mcp.json"))
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

		prBody := "This PR initializes the repository for agentic workflows by:\n" +
			"- Configuring .gitattributes\n" +
			"- Creating GitHub Copilot custom instructions\n" +
			"- Setting up workflow prompts and agents"
		if _, err := CreatePRWithChanges("init-agentic-workflows", "chore: initialize agentic workflows", "Initialize agentic workflows", prBody, opts.Verbose); err != nil {
			return err
		}
	}

	// Display success message with next steps
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if len(opts.CodespaceRepos) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Codespaces devcontainer configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, see https://github.github.com/gh-aw/setup/creating-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add an example workflow, see https://github.com/githubnext/agentics"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// ensureMaintenanceWorkflow checks existing workflows for expires field and generates/updates
// the maintenance workflow file if any workflows use it
func ensureMaintenanceWorkflow(verbose bool) error {
	initLog.Print("Checking for workflows with expires field")

	// Find git root
	gitRoot, err := gitutil.FindGitRoot()
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

	// Load repo-level configuration (optional; errors are non-fatal during init).
	repoConfig, err := workflow.LoadRepoConfig(gitRoot)
	if err != nil {
		initLog.Printf("Failed to load repo config, using defaults: %v", err)
		repoConfig = nil
	}

	if err := workflow.GenerateMaintenanceWorkflow(workflowDataList, workflowsDir, GetVersion(), compiler.GetActionMode(), compiler.GetActionTag(), verbose, repoConfig, compiler.GetRepositorySlug()); err != nil {
		return fmt.Errorf("failed to generate maintenance workflow: %w", err)
	}

	if verbose && len(workflowDataList) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Generated/updated maintenance workflow"))
	}

	return nil
}

// isGHESHost returns true when the given host is a GitHub Enterprise Server instance,
// i.e. it is neither the public github.com nor a GitHub Enterprise Cloud tenant
// (which uses the *.ghe.com domain).
func isGHESHost(host string) bool {
	// Strip optional port (e.g. "ghes.example.com:8080" → "ghes.example.com")
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	if host == "" {
		return false
	}
	if host == "github.com" {
		return false
	}
	// GitHub Enterprise Cloud tenants end with .ghe.com — not GHES
	if strings.HasSuffix(host, ".ghe.com") {
		return false
	}
	return true
}

// detectGHESDeployment returns the GHES host if the current repository's git
// remote points to a GitHub Enterprise Server instance, or "" if it does not.
// Detection uses the following sources in priority order:
//  1. GITHUB_SERVER_URL environment variable (set automatically inside GitHub Actions)
//  2. GH_HOST environment variable (set by the gh CLI)
//  3. The hostname extracted from the git origin remote URL
func detectGHESDeployment() string {
	// Check GITHUB_SERVER_URL first (set inside GitHub Actions runners)
	if serverURL := os.Getenv("GITHUB_SERVER_URL"); serverURL != "" {
		// serverURL is like "https://ghes.example.com", extract just the host.
		host := serverURL
		for _, scheme := range []string{"https://", "http://"} {
			host = strings.TrimPrefix(host, scheme)
		}
		host = strings.TrimSuffix(host, "/")
		if isGHESHost(host) {
			initLog.Printf("Detected GHES deployment from GITHUB_SERVER_URL: %s", host)
			return host
		}
	}

	// Check GH_HOST (set when using the gh CLI against an enterprise instance)
	if ghHost := os.Getenv("GH_HOST"); ghHost != "" {
		if isGHESHost(ghHost) {
			initLog.Printf("Detected GHES deployment from GH_HOST: %s", ghHost)
			return ghHost
		}
	}

	// Fall back to detecting the host from the git origin remote
	host := getHostFromOriginRemote()
	if isGHESHost(host) {
		initLog.Printf("Detected GHES deployment from git remote: %s", host)
		return host
	}

	return ""
}

// ensureGHESRepoConfig writes or updates .github/workflows/aw.json to set
// "ghes": true when running on a GHES deployment.  The function is a no-op
// if GHES is not detected or if "ghes": true is already present.
// Returns (updated bool, err).
func ensureGHESRepoConfig(verbose bool) (bool, error) {
	ghesHost := detectGHESDeployment()
	if ghesHost == "" {
		initLog.Print("No GHES deployment detected, skipping aw.json ghes configuration")
		return false, nil
	}

	initLog.Printf("GHES deployment detected (%s): configuring aw.json ghes: true", ghesHost)

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return false, fmt.Errorf("failed to find git root: %w", err)
	}

	configPath := filepath.Join(gitRoot, workflow.RepoConfigFileName)

	// Read existing content or start with an empty document.
	var doc map[string]any
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
			return false, fmt.Errorf("failed to parse %s: %w", workflow.RepoConfigFileName, jsonErr)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return false, fmt.Errorf("failed to read %s: %w", workflow.RepoConfigFileName, readErr)
	}

	if doc == nil {
		doc = make(map[string]any)
	}

	// Nothing to do if ghes is already true.
	if existing, ok := doc["ghes"].(bool); ok && existing {
		initLog.Print("aw.json already has ghes: true, nothing to update")
		return false, nil
	}

	doc["ghes"] = true

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, fmt.Errorf("failed to serialise %s: %w", workflow.RepoConfigFileName, err)
	}
	data = append(data, '\n')

	// Ensure the parent directory exists.
	if mkdirErr := os.MkdirAll(filepath.Dir(configPath), 0755); mkdirErr != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", workflow.RepoConfigFileName, mkdirErr)
	}

	if writeErr := os.WriteFile(configPath, data, 0644); writeErr != nil {
		return false, fmt.Errorf("failed to write %s: %w", workflow.RepoConfigFileName, writeErr)
	}

	initLog.Printf("Wrote ghes: true to %s", configPath)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(
			fmt.Sprintf("Configured %s with ghes: true (GHES deployment detected: %s)", workflow.RepoConfigFileName, ghesHost),
		))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("GHES deployment detected (%s): set ghes: true in %s for artifact compatibility", ghesHost, workflow.RepoConfigFileName),
		))
	}
	return true, nil
}
