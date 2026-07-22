package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var initLog = logger.New("cli:init")

// InitOptions contains all configuration options for repository initialization
type InitOptions struct {
	Ctx              context.Context
	Verbose          bool
	Engine           string
	NoGitattributes  bool
	Skill            bool
	Agent            bool
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

	ctx := initRepositoryContext(opts)
	copilotArtifactsEnabled := opts.Engine == "" || opts.Engine == "copilot"

	showInitRepositoryBanner()
	if err := runInitRepositoryPreflight(opts); err != nil {
		return err
	}
	if err := ensureInitRepositoryGitSetup(opts); err != nil {
		return err
	}
	if err := ensureCopilotInitArtifacts(opts, copilotArtifactsEnabled); err != nil {
		return err
	}
	if err := cleanupSetupAgenticWorkflowsAgent(opts.Verbose); err != nil {
		return err
	}
	if err := ensureInitRepositoryMCP(ctx, opts, copilotArtifactsEnabled); err != nil {
		return err
	}
	if err := ensureInitRepositoryCodespaces(opts); err != nil {
		return err
	}
	if err := ensureInitRepositoryVSCodeSettings(opts.Verbose); err != nil {
		return err
	}
	ensureInitRepositoryShellCompletions(opts)
	ensureInitRepositoryMaintenanceWorkflow(ctx, opts.Verbose)
	initLog.Print("Repository initialization completed successfully")
	if err := createInitRepositoryPRIfRequested(opts); err != nil {
		return err
	}
	displayInitRepositorySuccess(opts.CodespaceRepos)
	return nil
}

func initRepositoryContext(opts InitOptions) context.Context {
	if opts.Ctx != nil {
		return opts.Ctx
	}
	return context.Background()
}

func showInitRepositoryBanner() {
	console.ShowWelcomeBanner("This tool will initialize your repository for GitHub Agentic Workflows.")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting up repository..."))
	fmt.Fprintln(os.Stderr, "")
}

func runInitRepositoryPreflight(opts InitOptions) error {
	if opts.CreatePR {
		if err := PreflightCheckForCreatePR(opts.Verbose); err != nil {
			return err
		}
	}
	if !isGitRepo() {
		initLog.Print("Not in a git repository, initialization failed")
		return errors.New("not in a git repository")
	}
	initLog.Print("Verified git repository")
	return nil
}

func ensureInitRepositoryGitSetup(opts InitOptions) error {
	if _, err := ensureGHESRepoConfig(opts.Verbose); err != nil {
		initLog.Printf("Failed to configure GHES repo config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to configure GHES repo config: %v", err)))
	}
	if opts.NoGitattributes {
		initLog.Print("Skipping .gitattributes configuration")
		return nil
	}
	initLog.Print("Configuring .gitattributes")
	updated, err := ensureGitAttributes()
	if err != nil {
		initLog.Printf("Failed to configure .gitattributes: %v", err)
		return fmt.Errorf("failed to configure .gitattributes: %w", err)
	}
	if updated && opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .gitattributes"))
	}
	return nil
}

func ensureCopilotInitArtifacts(opts InitOptions, copilotArtifactsEnabled bool) error {
	if !copilotArtifactsEnabled {
		initLog.Printf("Skipping Copilot dispatcher skill for engine: %s", opts.Engine)
		return nil
	}
	if err := ensureInitRepositoryDispatcherSkill(opts); err != nil {
		return err
	}
	if err := ensureInitRepositoryAgent(opts); err != nil {
		return err
	}
	warnOnInitRepositoryCleanupError(deleteLegacyAgentFiles(opts.Verbose), "Failed to delete legacy agent files")
	warnOnInitRepositoryCleanupError(deleteAgenticWorkflowDesignerSkillDir(opts.Verbose), "Failed to delete legacy agentic-workflow-designer skill directory")
	return nil
}

func ensureInitRepositoryDispatcherSkill(opts InitOptions) error {
	if !opts.Skill {
		initLog.Print("Skipping agentic workflows dispatcher skill")
		return nil
	}
	initLog.Print("Writing agentic workflows dispatcher skill")
	if err := ensureAgenticWorkflowsDispatcher(opts.Verbose, false, true); err != nil {
		initLog.Printf("Failed to write dispatcher skill: %v", err)
		return fmt.Errorf("failed to write dispatcher skill: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created dispatcher skill"))
	}
	return nil
}

func ensureInitRepositoryAgent(opts InitOptions) error {
	if !opts.Agent {
		initLog.Print("Skipping agentic workflows custom agent")
		return nil
	}
	initLog.Print("Writing agentic workflows custom agent")
	if err := ensureAgenticWorkflowsAgent(opts.Verbose, true); err != nil {
		initLog.Printf("Failed to write agentic workflows custom agent: %v", err)
		return fmt.Errorf("failed to write agentic workflows custom agent: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created agentic workflows custom agent"))
	}
	return nil
}

func warnOnInitRepositoryCleanupError(err error, message string) {
	if err == nil {
		return
	}
	initLog.Printf("%s: %v", message, err)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: %s: %v", message, err)))
}

func cleanupSetupAgenticWorkflowsAgent(verbose bool) error {
	initLog.Print("Cleaning up setup agentic workflows agent")
	if err := deleteSetupAgenticWorkflowsAgent(verbose); err != nil {
		initLog.Printf("Failed to delete setup agentic workflows agent: %v", err)
		return fmt.Errorf("failed to delete setup agentic workflows agent: %w", err)
	}
	return nil
}

func ensureInitRepositoryMCP(ctx context.Context, opts InitOptions, copilotArtifactsEnabled bool) error {
	if !opts.MCP || !copilotArtifactsEnabled {
		return nil
	}
	initLog.Print("Configuring GitHub Copilot Agent MCP integration")
	actionMode := workflow.DetectActionMode(GetVersion())
	initLog.Printf("Using action mode for copilot-setup-steps.yml: %s", actionMode)
	if err := ensureCopilotSetupSteps(ctx, opts.Verbose, actionMode, GetVersion()); err != nil {
		initLog.Printf("Failed to create copilot-setup-steps.yml: %v", err)
		return fmt.Errorf("failed to create copilot-setup-steps.yml: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created .github/workflows/copilot-setup-steps.yml"))
	}
	if err := ensureMCPConfig(opts.Verbose); err != nil {
		initLog.Printf("Failed to create MCP config: %v", err)
		return fmt.Errorf("failed to create MCP config: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .github/mcp.json"))
	}
	return nil
}

func ensureInitRepositoryCodespaces(opts InitOptions) error {
	if !opts.CodespaceEnabled {
		return nil
	}
	initLog.Printf("Configuring GitHub Codespaces devcontainer with additional repos: %v", opts.CodespaceRepos)
	if err := ensureDevcontainerConfig(opts.Verbose, opts.CodespaceRepos); err != nil {
		initLog.Printf("Failed to configure devcontainer: %v", err)
		return fmt.Errorf("failed to configure devcontainer: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured .devcontainer/devcontainer.json"))
	}
	return nil
}

func ensureInitRepositoryVSCodeSettings(verbose bool) error {
	initLog.Print("Configuring VSCode settings")
	if err := ensureVSCodeSettings(verbose); err != nil {
		initLog.Printf("Failed to update VSCode settings: %v", err)
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated .vscode/settings.json"))
	}
	return nil
}

func ensureInitRepositoryShellCompletions(opts InitOptions) {
	if !opts.Completions {
		return
	}
	initLog.Print("Installing shell completions")
	fmt.Fprintln(os.Stderr, "")
	if err := InstallShellCompletion(opts.Verbose, opts.RootCmd); err != nil {
		initLog.Printf("Shell completion installation failed: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Shell completion installation encountered an issue: %v", err)))
	}
	fmt.Fprintln(os.Stderr, "")
}

func ensureInitRepositoryMaintenanceWorkflow(ctx context.Context, verbose bool) {
	initLog.Print("Checking for workflows with expires field to generate maintenance workflow")
	if err := ensureMaintenanceWorkflow(ctx, verbose); err != nil {
		initLog.Printf("Failed to generate maintenance workflow: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate maintenance workflow: %v", err)))
	}
}

func createInitRepositoryPRIfRequested(opts InitOptions) error {
	if !opts.CreatePR {
		return nil
	}
	initLog.Print("Create PR enabled - preparing to create branch, commit, push, and create PR")
	fmt.Fprintln(os.Stderr, "")
	prBody := "This PR initializes the repository for agentic workflows by:\n" +
		"- Configuring .gitattributes\n" +
		"- Creating GitHub Copilot custom instructions\n" +
		"- Setting up workflow prompts and skills"
	if _, err := CreatePRWithChanges("init-agentic-workflows", "chore: initialize agentic workflows", "Initialize agentic workflows", prBody, opts.Verbose); err != nil {
		return err
	}
	return nil
}

func displayInitRepositorySuccess(codespaceRepos []string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows!"))
	fmt.Fprintln(os.Stderr, "")
	if len(codespaceRepos) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("GitHub Codespaces devcontainer configured"))
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To create a workflow, see https://github.github.com/gh-aw/setup/creating-workflows"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or add an example workflow, see https://github.com/githubnext/agentics"))
	fmt.Fprintln(os.Stderr, "")
}

// ensureMaintenanceWorkflow checks existing workflows for expires field and generates/updates
// the maintenance workflow file if any workflows use it
func ensureMaintenanceWorkflow(ctx context.Context, verbose bool) error {
	initLog.Print("Checking for workflows with expires field")

	// Find git root
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Determine the workflows directory
	workflowsDir := filepath.Join(gitRoot, constants.GetWorkflowDir())
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

	if err := workflow.GenerateMaintenanceWorkflow(ctx, workflow.GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      workflowsDir,
		Version:          GetVersion(),
		ActionMode:       compiler.GetActionMode(),
		ActionTag:        compiler.GetActionTag(),
		RepoConfig:       repoConfig,
		RepoSlug:         compiler.GetRepositorySlug(),
	}); err != nil {
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
//  1. GITHUB_SERVER_URL, GITHUB_ENTERPRISE_HOST, GITHUB_HOST, GH_HOST environment variables
//  2. The hostname extracted from the git origin remote URL
func detectGHESDeployment() string {
	// Check env vars in unified priority order (mirrors GetGitHubHost):
	// GITHUB_SERVER_URL > GITHUB_ENTERPRISE_HOST > GITHUB_HOST > GH_HOST
	for _, envVar := range []string{"GITHUB_SERVER_URL", "GITHUB_ENTERPRISE_HOST", "GITHUB_HOST", "GH_HOST"} {
		rawValue := os.Getenv(envVar) //nolint:osgetenvlibrary
		if rawValue == "" {
			continue
		}
		host := strings.TrimPrefix(rawValue, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimSuffix(host, "/")
		if isGHESHost(host) {
			initLog.Printf("Detected GHES deployment from %s: %s", envVar, host)
			return host
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

	if mkdirErr := fileutil.EnsureParentDir(configPath, constants.DirPermPublic); mkdirErr != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", workflow.RepoConfigFileName, mkdirErr)
	}

	if writeErr := os.WriteFile(configPath, data, constants.FilePermPublic); writeErr != nil {
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
