package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var interactiveLog = logger.New("cli:interactive")

// commonWorkflowNames contains common workflow name patterns for autocomplete suggestions
var commonWorkflowNames = []string{
	"issue-triage",
	"pr-review-helper",
	"code-quality-check",
	"security-scan",
	"daily-report",
	"weekly-summary",
	"release-notes",
	"bug-reporter",
	"dependency-update",
	"documentation-check",
}

// InteractiveWorkflowBuilder collects user input to build an agentic workflow
type InteractiveWorkflowBuilder struct {
	WorkflowName  string
	Trigger       string
	Engine        string
	Tools         []string
	SafeOutputs   []string
	Intent        string
	NetworkAccess string
	CustomDomains []string
}

// CreateWorkflowInteractively prompts the user to build a workflow interactively
func CreateWorkflowInteractively(ctx context.Context, workflowName string, verbose bool, force bool) error {
	interactiveLog.Printf("Starting interactive workflow creation: workflowName=%s, force=%v", workflowName, force)

	// Assert this function is not running in automated unit tests
	if os.Getenv("GO_TEST_MODE") == "true" || os.Getenv("CI") != "" {
		return errors.New("interactive workflow creation cannot be used in automated tests or CI environments")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting interactive workflow creation..."))
	}

	builder := &InteractiveWorkflowBuilder{
		WorkflowName: workflowName,
	}

	// If using default workflow name, prompt for a better one
	if workflowName == "my-workflow" {
		if err := builder.promptForWorkflowName(); err != nil {
			return fmt.Errorf("failed to get workflow name: %w", err)
		}
	}

	// Run through the interactive prompts organized by groups
	if err := builder.promptForConfiguration(); err != nil {
		return fmt.Errorf("failed to get workflow configuration: %w", err)
	}

	// Generate the workflow
	if err := builder.generateWorkflow(force); err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Compile the workflow
	if err := builder.compileWorkflow(ctx, verbose); err != nil {
		return fmt.Errorf("failed to compile workflow: %w", err)
	}

	return nil
}

// promptForWorkflowName asks the user for a workflow name
func (b *InteractiveWorkflowBuilder) promptForWorkflowName() error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("What should we call this workflow?").
				Description("Enter a descriptive name for your workflow (e.g., 'issue-triage', 'code-review-helper')").
				Suggestions(commonWorkflowNames).
				Value(&b.WorkflowName).
				Validate(ValidateWorkflowName),
		),
	).WithAccessible(console.IsAccessibleMode())

	return form.Run()
}

// promptForConfiguration organizes all prompts into logical groups with titles and descriptions
func (b *InteractiveWorkflowBuilder) promptForConfiguration() error {
	// Prepare trigger options
	triggerOptions := []huh.Option[string]{
		huh.NewOption("Manual trigger (workflow_dispatch)", "workflow_dispatch"),
		huh.NewOption("Issue opened or reopened", "issues"),
		huh.NewOption("Pull request opened or synchronized", "pull_request"),
		huh.NewOption("Push to main branch", "push"),
		huh.NewOption("Issue comment created", "issue_comment"),
		huh.NewOption("Schedule (daily, scattered execution time)", "schedule_daily"),
		huh.NewOption("Schedule (weekly on Monday, scattered execution time)", "schedule_weekly"),
		huh.NewOption("Command trigger (/bot-name)", "command"),
	}

	// Prepare engine options
	engineOptions := []huh.Option[string]{
		huh.NewOption("copilot - GitHub Copilot CLI", "copilot"),
		huh.NewOption("claude - Anthropic Claude Code coding agent", "claude"),
		huh.NewOption("codex - OpenAI Codex engine", "codex"),
		huh.NewOption("custom - Custom engine configuration", "custom"),
	}

	// Prepare tool options
	toolOptions := []huh.Option[string]{
		huh.NewOption("github - GitHub API tools (issues, PRs, comments, repos)", "github"),
		huh.NewOption("edit - File editing tools", "edit"),
		huh.NewOption("bash - Shell command tools", "bash"),
		huh.NewOption("web-fetch - Web content fetching tools", "web-fetch"),
		huh.NewOption("web-search - Web search tools", "web-search"),
		huh.NewOption("playwright - Browser automation tools", "playwright"),
		huh.NewOption("serena - Serena code analysis tool", "serena"),
	}

	// Prepare safe output options programmatically from safe_outputs_tools.json
	outputOptions := buildSafeOutputOptions()

	// Pre-detect network access based on repo contents
	detectedNetworks := detectNetworkFromRepo()

	// Prepare network options
	networkOptions := []huh.Option[string]{
		huh.NewOption("defaults - Basic infrastructure only", "defaults"),
		huh.NewOption("ecosystem - Common development ecosystems (Python, Node.js, Go, etc.)", "ecosystem"),
	}
	if len(detectedNetworks) > 0 {
		// Build a custom option that reflects what was auto-detected
		label := "detected - Auto-detected ecosystems: " + strings.Join(detectedNetworks, ", ")
		networkOptions = append([]huh.Option[string]{huh.NewOption(label, strings.Join(append([]string{"defaults"}, detectedNetworks...), ","))}, networkOptions...)
	}

	// Set default network access
	b.NetworkAccess = "defaults"
	if len(detectedNetworks) > 0 {
		b.NetworkAccess = strings.Join(append([]string{"defaults"}, detectedNetworks...), ",")
	}

	// Variables to hold multi-select results
	var selectedTools []string
	var selectedOutputs []string

	// Create form with organized groups
	form := huh.NewForm(
		// Group 1: Basic Configuration
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("When should this workflow run?").
				Description("Choose the GitHub event that triggers this workflow").
				Options(triggerOptions...).
				Height(8).
				Value(&b.Trigger),
			huh.NewSelect[string]().
				Title("Which AI engine should process this workflow?").
				Description("The AI engine interprets instructions and executes tasks using available tools").
				Options(engineOptions...).
				Value(&b.Engine),
		).
			Title("Basic Configuration").
			Description("Let's start with the fundamentals of your workflow"),

		// Group 2: Capabilities
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which tools should the AI have access to?").
				Description("Tools enable the AI to interact with code, APIs, and external systems").
				Options(toolOptions...).
				Height(8).
				Value(&selectedTools),
			huh.NewMultiSelect[string]().
				Title("What outputs should the AI be able to create?").
				Description("Safe outputs allow the AI to create GitHub resources after human approval").
				Options(outputOptions...).
				Height(10).
				Value(&selectedOutputs),
		).
			Title("Capabilities").
			Description("Select the tools and outputs your workflow needs"),

		// Group 3: Network & Security
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What network access does the workflow need?").
				Description("Network access controls which external domains the workflow can reach").
				Options(networkOptions...).
				Value(&b.NetworkAccess),
		).
			Title("Network & Security").
			Description("Configure network access and security settings"),

		// Group 4: Instructions
		huh.NewGroup(
			huh.NewText().
				Title("Describe what this workflow should do:").
				Description("Provide clear, detailed instructions for the AI to follow when executing this workflow").
				Value(&b.Intent).
				Validate(ValidateWorkflowIntent),
		).
			Title("Instructions").
			Description("Describe what you want this workflow to accomplish"),
	).WithAccessible(console.IsAccessibleMode())

	if err := form.Run(); err != nil {
		return err
	}

	// Store the multi-select results
	b.Tools = selectedTools
	b.SafeOutputs = selectedOutputs

	return nil
}

// generateWorkflow creates the markdown workflow file based on user selections
func (b *InteractiveWorkflowBuilder) generateWorkflow(force bool) error {
	interactiveLog.Printf("Generating workflow file: name=%s, engine=%s, trigger=%s", b.WorkflowName, b.Engine, b.Trigger)

	// Get current working directory for .github/workflows
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	githubWorkflowsDir := filepath.Join(workingDir, constants.GetWorkflowDir())
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Construct the destination file path
	destFile := filepath.Join(githubWorkflowsDir, b.WorkflowName+".md")

	// Check if destination file already exists
	if _, err := os.Stat(destFile); err == nil && !force {
		var overwrite bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Workflow file '%s' already exists. Overwrite?", filepath.Base(destFile))).
					Affirmative("Yes, overwrite").
					Negative("No, cancel").
					Value(&overwrite),
			),
		).WithAccessible(console.IsAccessibleMode())

		if err := confirmForm.Run(); err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if !overwrite {
			return errors.New("workflow creation cancelled")
		}
	}

	// Generate workflow content
	content := b.generateWorkflowContent()

	// Write the workflow to file with owner-only read/write permissions (0600) for security best practices
	if err := os.WriteFile(destFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", destFile, err)
	}

	interactiveLog.Printf("Workflow file created successfully: %s", destFile)
	fmt.Fprintf(os.Stderr, "Created new workflow: %s\n", destFile)
	return nil
}

// generateWorkflowContent creates the workflow markdown content
func (b *InteractiveWorkflowBuilder) generateWorkflowContent() string {
	var content strings.Builder

	// Write frontmatter
	content.WriteString("---\n")

	// Add trigger configuration
	content.WriteString(b.generateTriggerConfig())

	// Add permissions
	content.WriteString(b.generatePermissionsConfig())

	// Add engine configuration
	fmt.Fprintf(&content, "engine: %s\n", b.Engine)

	// Add network configuration
	content.WriteString(b.generateNetworkConfig())

	// Add tools configuration
	if len(b.Tools) > 0 {
		content.WriteString(b.generateToolsConfig())
	}

	// Add safe outputs configuration
	if len(b.SafeOutputs) > 0 {
		content.WriteString(b.generateSafeOutputsConfig())
	}

	content.WriteString("---\n\n")

	// Add workflow title and content
	fmt.Fprintf(&content, "# %s\n\n", b.WorkflowName)

	if b.Intent != "" {
		fmt.Fprintf(&content, "%s\n\n", b.Intent)
	}

	// Add TODO sections for customization
	content.WriteString("<!--\n")
	content.WriteString("## TODO: Customize this workflow\n\n")
	content.WriteString("The workflow has been generated based on your selections. Consider adding:\n\n")
	content.WriteString("- [ ] More specific instructions for the AI\n")
	content.WriteString("- [ ] Error handling requirements\n")
	content.WriteString("- [ ] Output format specifications\n")
	content.WriteString("- [ ] Integration with other workflows\n")
	content.WriteString("- [ ] Testing and validation steps\n\n")

	content.WriteString("## Configuration Summary\n\n")
	fmt.Fprintf(&content, "- **Trigger**: %s\n", b.describeTrigger())
	fmt.Fprintf(&content, "- **AI Engine**: %s\n", b.Engine)

	if len(b.Tools) > 0 {
		fmt.Fprintf(&content, "- **Tools**: %s\n", strings.Join(b.Tools, ", "))
	}

	if len(b.SafeOutputs) > 0 {
		fmt.Fprintf(&content, "- **Safe Outputs**: %s\n", strings.Join(b.SafeOutputs, ", "))
	}

	fmt.Fprintf(&content, "- **Network Access**: %s\n", b.NetworkAccess)

	content.WriteString("\n## Next Steps\n\n")
	content.WriteString("1. Review and customize the workflow content above\n")
	content.WriteString("2. Remove TODO sections when ready\n")
	fmt.Fprintf(&content, "3. Run `%s compile` to generate the GitHub Actions workflow\n", string(constants.CLIExtensionPrefix))
	content.WriteString("4. Test the workflow with a manual trigger or appropriate event\n")
	content.WriteString("-->\n")

	return content.String()
}

// Helper methods for generating configuration sections

func (b *InteractiveWorkflowBuilder) generateTriggerConfig() string {
	switch b.Trigger {
	case "workflow_dispatch":
		return "on:\n  workflow_dispatch:\n"
	case "issues":
		return "on:\n  issues:\n    types: [opened, reopened]\n"
	case "pull_request":
		return "on:\n  pull_request:\n    types: [opened, synchronize]\n"
	case "push":
		return "on:\n  push:\n    branches: [main]\n"
	case "issue_comment":
		return "on:\n  issue_comment:\n    types: [created]\n"
	case "schedule_daily":
		return "on:\n  schedule: daily\n"
	case "schedule_weekly":
		return "on:\n  schedule: weekly on monday\n"
	case "command":
		return "on:\n  command:\n    name: bot-name  # TODO: Replace with your bot name\n"
	default:
		return "on:\n  workflow_dispatch:\n"
	}
}

func (b *InteractiveWorkflowBuilder) generatePermissionsConfig() string {
	// Compute read permissions needed by the AI agent for data access.
	// Write permissions are NEVER set here — they are always handled automatically
	// by the safe-outputs job via workflow.ComputePermissionsForSafeOutputs().
	perms := workflow.NewPermissions()
	perms.Set(workflow.PermissionContents, workflow.PermissionRead)

	if slices.Contains(b.Tools, "github") {
		// Default toolsets: context, repos, issues, pull_requests
		// repos → contents: read (already set)
		// issues → issues: read
		// pull_requests → pull-requests: read
		perms.Set(workflow.PermissionIssues, workflow.PermissionRead)
		perms.Set(workflow.PermissionPullRequests, workflow.PermissionRead)
	}

	// Include read permissions needed by the safe-outputs job (e.g. contents: read
	// is already present; actions: read for autofix scanning alerts).
	// Write permissions from ComputePermissionsForSafeOutputs are handled by the
	// safe-outputs job automatically and must not appear in the main workflow block.
	safeOutputsPerms := workflow.ComputePermissionsForSafeOutputs(workflow.SafeOutputsConfigFromKeys(b.SafeOutputs))
	for _, scope := range workflow.GetAllPermissionScopes() {
		if level, exists := safeOutputsPerms.Get(scope); exists && level == workflow.PermissionRead {
			perms.Set(scope, workflow.PermissionRead)
		}
	}

	return perms.RenderToYAML() + "\n"
}

func (b *InteractiveWorkflowBuilder) generateNetworkConfig() string {
	switch b.NetworkAccess {
	case "ecosystem":
		return "network:\n  allowed:\n    - defaults\n    - python\n    - node\n    - go\n    - java\n"
	default:
		// Handle comma-separated networks (e.g. "defaults,node,python")
		parts := strings.Split(b.NetworkAccess, ",")
		if len(parts) == 1 {
			return fmt.Sprintf("network: %s\n", parts[0])
		}
		var cfg strings.Builder
		cfg.WriteString("network:\n  allowed:\n")
		for _, p := range parts {
			fmt.Fprintf(&cfg, "    - %s\n", strings.TrimSpace(p))
		}
		return cfg.String()
	}
}

func (b *InteractiveWorkflowBuilder) generateToolsConfig() string {
	if len(b.Tools) == 0 {
		return ""
	}

	var config strings.Builder
	config.WriteString("tools:\n")

	// Add standard tools
	for _, tool := range b.Tools {
		switch tool {
		case "github":
			// Use default toolsets (context, repos, issues, pull_requests)
			// which matches the DefaultGitHubToolsets constant.
			config.WriteString("  github:\n    toolsets: [default]\n")
		case "bash":
			config.WriteString("  bash:\n")
		default:
			fmt.Fprintf(&config, "  %s:\n", tool)
		}
	}

	return config.String()
}

func (b *InteractiveWorkflowBuilder) generateSafeOutputsConfig() string {
	if len(b.SafeOutputs) == 0 {
		return ""
	}

	var config strings.Builder
	config.WriteString("safe-outputs:\n")

	for _, output := range b.SafeOutputs {
		fmt.Fprintf(&config, "  %s:\n", output)
	}

	return config.String()
}

func (b *InteractiveWorkflowBuilder) describeTrigger() string {
	switch b.Trigger {
	case "workflow_dispatch":
		return "Manual trigger"
	case "issues":
		return "Issue opened or reopened"
	case "pull_request":
		return "Pull request opened or synchronized"
	case "push":
		return "Push to main branch"
	case "issue_comment":
		return "Issue comment created"
	case "schedule_daily":
		return "Daily schedule (fuzzy, scattered time)"
	case "schedule_weekly":
		return "Weekly schedule (Monday, fuzzy scattered time)"
	case "command":
		return "Command trigger (/bot-name)"
	case "custom":
		return "Custom trigger (TODO: configure)"
	default:
		return "Unknown trigger"
	}
}

// compileWorkflow automatically compiles the generated workflow
func (b *InteractiveWorkflowBuilder) compileWorkflow(ctx context.Context, verbose bool) error {
	interactiveLog.Printf("Starting workflow compilation: name=%s, verbose=%v", b.WorkflowName, verbose)

	// Create spinner for compilation progress
	spinner := console.NewSpinner("Compiling your workflow...")
	spinner.Start()

	// Use the existing compile functionality
	config := CompileConfig{
		MarkdownFiles:        []string{b.WorkflowName},
		Verbose:              verbose,
		EngineOverride:       "",
		Validate:             true,
		Watch:                false,
		WorkflowDir:          "",
		SkipInstructions:     false,
		NoEmit:               false,
		Purge:                false,
		TrialMode:            false,
		TrialLogicalRepoSlug: "",
	}

	_, err := CompileWorkflows(ctx, config)

	if err != nil {
		spinner.Stop()
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Compilation failed: %v", err)))
		return err
	}

	// Stop spinner with success message
	spinner.StopWithMessage("✓ Workflow compiled successfully!")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("You can now find your compiled workflow at .github/workflows/%s.lock.yml", b.WorkflowName)))

	return nil
}

// buildSafeOutputOptions loads safe output tool options from safe_outputs_tools.json
// and returns them as huh form options.
func buildSafeOutputOptions() []huh.Option[string] {
	tools := workflow.GetSafeOutputToolOptions()

	options := make([]huh.Option[string], 0, len(tools))
	for _, t := range tools {
		// Truncate long descriptions so option labels remain readable
		desc := t.Description
		if idx := strings.IndexByte(desc, '.'); idx > 0 && idx < 120 {
			desc = desc[:idx]
		} else if len(desc) > 120 {
			desc = desc[:120] + "…"
		}
		label := fmt.Sprintf("%s - %s", t.Key, desc)
		options = append(options, huh.NewOption(label, t.Key))
	}
	return options
}

// repoLanguageMarkers maps well-known ecosystem indicator files to their network bucket name.
var repoLanguageMarkers = []struct {
	file   string
	bucket string
}{
	{"go.mod", "go"},
	{"package.json", "node"},
	{"requirements.txt", "python"},
	{"pyproject.toml", "python"},
	{"Pipfile", "python"},
	{"Gemfile", "ruby"},
	{"Cargo.toml", "rust"},
	{"pom.xml", "java"},
	{"build.gradle", "java"},
	{"*.csproj", "dotnet"},
	{"*.fsproj", "dotnet"},
}

// detectNetworkFromRepo scans the current working directory for ecosystem indicator
// files and returns a deduplicated, sorted list of network bucket names to add.
func detectNetworkFromRepo() []string {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	for _, m := range repoLanguageMarkers {
		var found bool
		if strings.ContainsAny(m.file, "*?[") {
			// Glob pattern
			matches, err := filepath.Glob(filepath.Join(cwd, m.file))
			found = err == nil && len(matches) > 0
		} else {
			_, err := os.Stat(filepath.Join(cwd, m.file))
			found = err == nil
		}
		if found && !seen[m.bucket] {
			seen[m.bucket] = true
		}
	}

	buckets := make([]string, 0, len(seen))
	for b := range seen {
		buckets = append(buckets, b)
	}
	sort.Strings(buckets)
	return buckets
}
