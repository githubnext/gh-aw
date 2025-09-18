package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
)

// InteractiveWorkflowBuilder collects user input to build an agentic workflow
type InteractiveWorkflowBuilder struct {
	WorkflowName  string
	Trigger       string
	Engine        string
	Tools         []string
	MCPTools      []string
	SafeOutputs   []string
	Intent        string
	NetworkAccess string
	CustomDomains []string
}

// CreateWorkflowInteractively prompts the user to build a workflow interactively
func CreateWorkflowInteractively(workflowName string, verbose bool, force bool) error {
	if verbose {
		fmt.Println(console.FormatInfoMessage("Starting interactive workflow creation..."))
	}

	builder := &InteractiveWorkflowBuilder{
		WorkflowName: workflowName,
	}

	// Run through the interactive prompts
	if err := builder.promptForTrigger(); err != nil {
		return fmt.Errorf("failed to get trigger selection: %w", err)
	}

	if err := builder.promptForEngine(); err != nil {
		return fmt.Errorf("failed to get engine selection: %w", err)
	}

	if err := builder.promptForTools(); err != nil {
		return fmt.Errorf("failed to get tools selection: %w", err)
	}

	if err := builder.promptForMCPTools(verbose); err != nil {
		return fmt.Errorf("failed to get MCP tools selection: %w", err)
	}

	if err := builder.promptForSafeOutputs(); err != nil {
		return fmt.Errorf("failed to get safe outputs selection: %w", err)
	}

	if err := builder.promptForNetworkAccess(); err != nil {
		return fmt.Errorf("failed to get network access selection: %w", err)
	}

	if err := builder.promptForIntent(); err != nil {
		return fmt.Errorf("failed to get workflow intent: %w", err)
	}

	// Generate the workflow
	if err := builder.generateWorkflow(verbose, force); err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Compile the workflow
	if err := builder.compileWorkflow(verbose); err != nil {
		return fmt.Errorf("failed to compile workflow: %w", err)
	}

	return nil
}

// promptForTrigger asks the user to select when the workflow should run
func (b *InteractiveWorkflowBuilder) promptForTrigger() error {
	triggerOptions := []string{
		"Manual trigger (workflow_dispatch)",
		"Issue opened or reopened",
		"Pull request opened or synchronized",
		"Push to main branch",
		"Issue comment created",
		"Schedule (daily at 9 AM UTC)",
		"Schedule (weekly on Monday at 9 AM UTC)",
		"Command trigger (/bot-name)",
		"Custom trigger",
	}

	prompt := &survey.Select{
		Message: "When should this workflow run?",
		Options: triggerOptions,
		Help:    "Select the event that should trigger your agentic workflow",
	}

	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Map selection to YAML config
	switch selected {
	case "Manual trigger (workflow_dispatch)":
		b.Trigger = "workflow_dispatch"
	case "Issue opened or reopened":
		b.Trigger = "issues"
	case "Pull request opened or synchronized":
		b.Trigger = "pull_request"
	case "Push to main branch":
		b.Trigger = "push"
	case "Issue comment created":
		b.Trigger = "issue_comment"
	case "Schedule (daily at 9 AM UTC)":
		b.Trigger = "schedule_daily"
	case "Schedule (weekly on Monday at 9 AM UTC)":
		b.Trigger = "schedule_weekly"
	case "Command trigger (/bot-name)":
		b.Trigger = "command"
	case "Custom trigger":
		b.Trigger = "custom"
	}

	return nil
}

// promptForEngine asks the user to select the AI engine
func (b *InteractiveWorkflowBuilder) promptForEngine() error {
	engineOptions := []string{
		"claude (default) - Claude coding agent",
		"codex - GitHub Codex engine",
		"custom - Custom engine configuration",
	}

	prompt := &survey.Select{
		Message: "Which AI engine should process this workflow?",
		Options: engineOptions,
		Help:    "Claude is recommended for most use cases",
	}

	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract engine ID
	switch {
	case strings.HasPrefix(selected, "claude"):
		b.Engine = "claude"
	case strings.HasPrefix(selected, "codex"):
		b.Engine = "codex"
	case strings.HasPrefix(selected, "custom"):
		b.Engine = "custom"
	}

	return nil
}

// promptForTools asks the user to select which tools the AI can use
func (b *InteractiveWorkflowBuilder) promptForTools() error {
	toolOptions := []string{
		"github - GitHub API tools (issues, PRs, comments)",
		"edit - File editing tools",
		"bash - Shell command tools",
		"web-fetch - Web content fetching tools",
		"web-search - Web search tools",
		"playwright - Browser automation tools",
	}

	prompt := &survey.MultiSelect{
		Message: "Which tools should the AI have access to?",
		Options: toolOptions,
		Help:    "Select all tools that your workflow might need. You can always modify these later.",
	}

	var selected []string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract tool names
	b.Tools = make([]string, 0, len(selected))
	for _, tool := range selected {
		toolName := strings.Split(tool, " ")[0]
		b.Tools = append(b.Tools, toolName)
	}

	return nil
}

// promptForMCPTools asks the user to select MCP tools from the registry
func (b *InteractiveWorkflowBuilder) promptForMCPTools(verbose bool) error {
	// Ask if they want to add MCP tools
	var wantsMCP bool
	mcpPrompt := &survey.Confirm{
		Message: "Do you want to add MCP (Model Context Protocol) tools?",
		Help:    "MCP tools provide additional capabilities like database access, API integrations, etc.",
		Default: false,
	}

	if err := survey.AskOne(mcpPrompt, &wantsMCP); err != nil {
		return err
	}

	if !wantsMCP {
		return nil
	}

	// Fetch available MCP servers
	if verbose {
		fmt.Println(console.FormatInfoMessage("Fetching MCP servers from registry..."))
	}

	registryClient := NewMCPRegistryClient("") // Use default registry
	servers, err := registryClient.SearchServers("")
	if err != nil {
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to fetch MCP servers: %v", err)))
		fmt.Println(console.FormatInfoMessage("Skipping MCP tools selection - you can add them manually later"))
		return nil // Don't fail the whole process
	}

	if len(servers) == 0 {
		fmt.Println(console.FormatWarningMessage("No MCP servers found in registry"))
		return nil
	}

	// Create options list - limit to first 20 for better UX
	maxOptions := 20
	if len(servers) > maxOptions {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Showing first %d out of %d available MCP servers", maxOptions, len(servers))))
		}
		servers = servers[:maxOptions]
	}

	// Create options list
	options := make([]string, 0, len(servers))
	for _, server := range servers {
		name := server.Name
		if name == "" {
			continue // Skip servers without names
		}

		desc := server.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		if desc != "" {
			options = append(options, fmt.Sprintf("%s - %s", name, desc))
		} else {
			options = append(options, name)
		}
	}

	if len(options) == 0 {
		fmt.Println(console.FormatWarningMessage("No valid MCP servers found"))
		return nil
	}

	// Sort options for better UX
	sort.Strings(options)

	prompt := &survey.MultiSelect{
		Message:  "Select MCP tools to include:",
		Options:  options,
		Help:     "Choose external tools and services your workflow needs",
		PageSize: 10,
	}

	var selected []string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract server names
	b.MCPTools = make([]string, 0, len(selected))
	for _, option := range selected {
		serverName := strings.Split(option, " ")[0]
		b.MCPTools = append(b.MCPTools, serverName)
	}

	return nil
}

// promptForSafeOutputs asks the user to select safe output options
func (b *InteractiveWorkflowBuilder) promptForSafeOutputs() error {
	outputOptions := []string{
		"create-issue - Create GitHub issues",
		"add-comment - Add comments to issues/PRs",
		"create-pull-request - Create pull requests",
		"update-issue - Update existing issues",
		"create-discussion - Create repository discussions",
	}

	prompt := &survey.MultiSelect{
		Message: "What outputs should the AI be able to create?",
		Options: outputOptions,
		Help:    "Safe outputs provide secure ways for AI to interact with GitHub. Select what your workflow needs to do.",
	}

	var selected []string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Extract output names
	b.SafeOutputs = make([]string, 0, len(selected))
	for _, output := range selected {
		outputName := strings.Split(output, " ")[0]
		b.SafeOutputs = append(b.SafeOutputs, outputName)
	}

	return nil
}

// promptForNetworkAccess asks about network access requirements
func (b *InteractiveWorkflowBuilder) promptForNetworkAccess() error {
	networkOptions := []string{
		"defaults - Basic infrastructure only",
		"ecosystem - Common development ecosystems (Python, Node.js, Go, etc.)",
		"custom - Specify custom domains",
		"none - No network access",
	}

	prompt := &survey.Select{
		Message: "What network access does the workflow need?",
		Options: networkOptions,
		Help:    "Network permissions control what external sites the AI can access",
		Default: "defaults",
	}

	var selected string
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(selected, "defaults"):
		b.NetworkAccess = "defaults"
	case strings.HasPrefix(selected, "ecosystem"):
		b.NetworkAccess = "ecosystem"
	case strings.HasPrefix(selected, "custom"):
		b.NetworkAccess = "custom"
		return b.promptForCustomDomains()
	case strings.HasPrefix(selected, "none"):
		b.NetworkAccess = "none"
	}

	return nil
}

// promptForCustomDomains asks for specific domains when custom network access is selected
func (b *InteractiveWorkflowBuilder) promptForCustomDomains() error {
	prompt := &survey.Input{
		Message: "Enter comma-separated list of allowed domains:",
		Help:    "Example: api.github.com, example.com, *.trusted-domain.com",
	}

	var domains string
	if err := survey.AskOne(prompt, &domains); err != nil {
		return err
	}

	if domains != "" {
		b.CustomDomains = strings.Split(domains, ",")
		for i, domain := range b.CustomDomains {
			b.CustomDomains[i] = strings.TrimSpace(domain)
		}
	}

	return nil
}

// promptForIntent asks the user to describe what the workflow should do
func (b *InteractiveWorkflowBuilder) promptForIntent() error {
	prompt := &survey.Multiline{
		Message: "Describe what this workflow should do:",
		Help:    "Provide a clear description of the workflow's purpose and what the AI should accomplish. This will be the main prompt for the AI.",
	}

	return survey.AskOne(prompt, &b.Intent)
}

// generateWorkflow creates the markdown workflow file based on user selections
func (b *InteractiveWorkflowBuilder) generateWorkflow(verbose bool, force bool) error {
	// Get current working directory for .github/workflows
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	githubWorkflowsDir := filepath.Join(workingDir, ".github", "workflows")
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Construct the destination file path
	destFile := filepath.Join(githubWorkflowsDir, b.WorkflowName+".md")

	// Check if destination file already exists
	if _, err := os.Stat(destFile); err == nil && !force {
		return fmt.Errorf("workflow file '%s' already exists. Use --force to overwrite", destFile)
	}

	// Generate workflow content
	content := b.generateWorkflowContent()

	// Write the workflow to file
	if err := os.WriteFile(destFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", destFile, err)
	}

	fmt.Printf("Created new workflow: %s\n", destFile)
	return nil
}

// GenerateWorkflowContent creates the workflow markdown content (exported for testing)
func (b *InteractiveWorkflowBuilder) GenerateWorkflowContent() string {
	return b.generateWorkflowContent()
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
	content.WriteString(fmt.Sprintf("engine: %s\n", b.Engine))

	// Add network configuration if needed
	if b.NetworkAccess != "defaults" {
		content.WriteString(b.generateNetworkConfig())
	}

	// Add tools configuration
	if len(b.Tools) > 0 || len(b.MCPTools) > 0 {
		content.WriteString(b.generateToolsConfig())
	}

	// Add safe outputs configuration
	if len(b.SafeOutputs) > 0 {
		content.WriteString(b.generateSafeOutputsConfig())
	}

	content.WriteString("---\n\n")

	// Add workflow title and content
	content.WriteString(fmt.Sprintf("# %s\n\n", b.WorkflowName))

	if b.Intent != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", b.Intent))
	}

	// Add TODO sections for customization
	content.WriteString("## TODO: Customize this workflow\n\n")
	content.WriteString("The workflow has been generated based on your selections. Consider adding:\n\n")
	content.WriteString("- [ ] More specific instructions for the AI\n")
	content.WriteString("- [ ] Error handling requirements\n")
	content.WriteString("- [ ] Output format specifications\n")
	content.WriteString("- [ ] Integration with other workflows\n")
	content.WriteString("- [ ] Testing and validation steps\n\n")

	content.WriteString("## Configuration Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Trigger**: %s\n", b.describeTrigger()))
	content.WriteString(fmt.Sprintf("- **AI Engine**: %s\n", b.Engine))

	if len(b.Tools) > 0 {
		content.WriteString(fmt.Sprintf("- **Tools**: %s\n", strings.Join(b.Tools, ", ")))
	}

	if len(b.MCPTools) > 0 {
		content.WriteString(fmt.Sprintf("- **MCP Tools**: %s\n", strings.Join(b.MCPTools, ", ")))
	}

	if len(b.SafeOutputs) > 0 {
		content.WriteString(fmt.Sprintf("- **Safe Outputs**: %s\n", strings.Join(b.SafeOutputs, ", ")))
	}

	content.WriteString(fmt.Sprintf("- **Network Access**: %s\n", b.NetworkAccess))

	content.WriteString("\n## Next Steps\n\n")
	content.WriteString("1. Review and customize the workflow content above\n")
	content.WriteString("2. Remove TODO sections when ready\n")
	content.WriteString(fmt.Sprintf("3. Run `%s compile` to generate the GitHub Actions workflow\n", constants.CLIExtensionPrefix))
	content.WriteString("4. Test the workflow with a manual trigger or appropriate event\n")

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
		return "on:\n  schedule:\n    - cron: \"0 9 * * *\"  # Daily at 9 AM UTC\n"
	case "schedule_weekly":
		return "on:\n  schedule:\n    - cron: \"0 9 * * 1\"  # Weekly on Monday at 9 AM UTC\n"
	case "command":
		return "on:\n  command:\n    name: bot-name  # TODO: Replace with your bot name\n"
	case "custom":
		return "# TODO: Add your custom trigger configuration\non:\n  workflow_dispatch:\n"
	default:
		return "on:\n  workflow_dispatch:\n"
	}
}

func (b *InteractiveWorkflowBuilder) generatePermissionsConfig() string {
	permissions := []string{"contents: read"}

	// Add permissions based on safe outputs
	for _, output := range b.SafeOutputs {
		switch output {
		case "create-issue", "update-issue", "add-comment":
			if !containsString(permissions, "issues: write") {
				permissions = append(permissions, "issues: write")
			}
		case "create-pull-request":
			if !containsString(permissions, "pull-requests: write") {
				permissions = append(permissions, "pull-requests: write")
			}
		case "create-discussion":
			if !containsString(permissions, "discussions: write") {
				permissions = append(permissions, "discussions: write")
			}
		}
	}

	// Always add actions: read for safe outputs
	if len(b.SafeOutputs) > 0 && !containsString(permissions, "actions: read") {
		permissions = append(permissions, "actions: read")
	}

	var config strings.Builder
	config.WriteString("permissions:\n")
	for _, perm := range permissions {
		config.WriteString(fmt.Sprintf("  %s\n", perm))
	}

	return config.String()
}

func (b *InteractiveWorkflowBuilder) generateNetworkConfig() string {
	switch b.NetworkAccess {
	case "ecosystem":
		return "network:\n  allowed:\n    - defaults\n    - python\n    - node\n    - go\n    - java\n"
	case "custom":
		if len(b.CustomDomains) > 0 {
			var config strings.Builder
			config.WriteString("network:\n  allowed:\n")
			for _, domain := range b.CustomDomains {
				config.WriteString(fmt.Sprintf("    - \"%s\"\n", domain))
			}
			return config.String()
		}
		return "network:\n  allowed: []  # TODO: Add your custom domains\n"
	case "none":
		return "network: {}\n"
	default:
		return ""
	}
}

func (b *InteractiveWorkflowBuilder) generateToolsConfig() string {
	if len(b.Tools) == 0 && len(b.MCPTools) == 0 {
		return ""
	}

	var config strings.Builder
	config.WriteString("tools:\n")

	// Add standard tools
	for _, tool := range b.Tools {
		switch tool {
		case "github":
			config.WriteString("  github:\n    allowed:\n      - get_issue\n      - add_issue_comment\n      - create_issue\n")
		case "bash":
			config.WriteString("  bash:\n")
		default:
			config.WriteString(fmt.Sprintf("  %s:\n", tool))
		}
	}

	// Add MCP tools
	for _, mcpTool := range b.MCPTools {
		config.WriteString(fmt.Sprintf("  %s:\n    # TODO: Configure MCP server settings\n", mcpTool))
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
		config.WriteString(fmt.Sprintf("  %s:\n", output))
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
		return "Daily schedule (9 AM UTC)"
	case "schedule_weekly":
		return "Weekly schedule (Monday 9 AM UTC)"
	case "command":
		return "Command trigger (/bot-name)"
	case "custom":
		return "Custom trigger (TODO: configure)"
	default:
		return "Unknown trigger"
	}
}

// compileWorkflow automatically compiles the generated workflow
func (b *InteractiveWorkflowBuilder) compileWorkflow(verbose bool) error {
	fmt.Println(console.FormatInfoMessage("Compiling the generated workflow..."))

	// Use the existing compile functionality
	return CompileWorkflows([]string{b.WorkflowName}, verbose, "", true, false, "", false, false, false)
}

// Helper function to check if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
