package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#50FA7B")).
			MarginBottom(1)

	serverNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#BD93F9"))

	typeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD"))

	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5555")).
			Padding(1).
			Margin(1)
)

// inspectMCPServer connects to an MCP server and queries its capabilities
func inspectMCPServer(config parser.MCPServerConfig, toolFilter string, verbose bool, useActionsSecrets bool) error {
	fmt.Printf("%s %s (%s)\n",
		serverNameStyle.Render("📡 "+config.Name),
		typeStyle.Render(config.Type),
		typeStyle.Render(buildConnectionString(config)))

	// Validate secrets/environment variables
	if err := validateServerSecrets(config, verbose, useActionsSecrets); err != nil {
		fmt.Print(errorBoxStyle.Render(fmt.Sprintf("❌ Secret validation failed: %s", err)))
		return nil // Don't return error, just show validation failure
	}

	// Connect to the server
	info, err := connectToMCPServer(config, verbose)
	if err != nil {
		fmt.Print(errorBoxStyle.Render(fmt.Sprintf("❌ Connection failed: %s", err)))
		return nil // Don't return error, just show connection failure
	}

	if verbose {
		fmt.Println(console.FormatSuccessMessage("✅ Successfully connected to MCP server"))
	}

	// Display server capabilities
	displayServerCapabilities(info, toolFilter)

	return nil
}

// buildConnectionString creates a display string for the connection details
func buildConnectionString(config parser.MCPServerConfig) string {
	switch config.Type {
	case "stdio":
		if config.Container != "" {
			return fmt.Sprintf("docker: %s", config.Container)
		}
		if len(config.Args) > 0 {
			return fmt.Sprintf("cmd: %s %s", config.Command, strings.Join(config.Args, " "))
		}
		return fmt.Sprintf("cmd: %s", config.Command)
	case "http":
		return config.URL
	default:
		return config.Type
	}
}

// validateServerSecrets checks if required environment variables/secrets are available
func validateServerSecrets(config parser.MCPServerConfig, verbose bool, useActionsSecrets bool) error {
	// Extract secrets from the config
	requiredSecrets := extractSecretsFromConfig(config)

	// Special case: Check for GH_AW_GITHUB_TOKEN when GitHub tool is in remote mode
	if config.Name == "github" && config.Type == "http" {
		// GitHub remote mode requires GH_AW_GITHUB_TOKEN secret
		// Check if a custom token is already specified in the env
		hasCustomToken := false
		for _, value := range config.Env {
			if strings.Contains(value, "secrets.") && !strings.Contains(value, "GH_AW_GITHUB_TOKEN") {
				// Custom token specified, no need to check GH_AW_GITHUB_TOKEN
				hasCustomToken = true
				break
			}
		}

		if !hasCustomToken {
			// Add GH_AW_GITHUB_TOKEN to required secrets if not already present
			alreadyPresent := false
			for _, secret := range requiredSecrets {
				if secret.Name == "GH_AW_GITHUB_TOKEN" {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				requiredSecrets = append(requiredSecrets, SecretInfo{
					Name:   "GH_AW_GITHUB_TOKEN",
					EnvKey: "GITHUB_TOKEN",
				})
			}
		}
	}

	if len(requiredSecrets) == 0 {
		// No secrets required, proceed with normal env var validation
		for key, value := range config.Env {
			// Check if value contains variable references
			if strings.Contains(value, "${") {
				// Extract variable name (simplified parsing)
				if strings.Contains(value, "secrets.") {
					// This should have been caught by extractSecretsFromConfig
					continue
				}
				if strings.Contains(value, "GH_TOKEN") || strings.Contains(value, "GITHUB_TOKEN") || strings.Contains(value, "GITHUB_PERSONAL_ACCESS_TOKEN") {
					if token, err := parser.GetGitHubToken(); err != nil {
						return fmt.Errorf("GitHub token not found in environment (set GH_TOKEN or GITHUB_TOKEN)")
					} else {
						config.Env[key] = token
					}
				}
				// Handle our placeholder for GitHub token requirement
				if strings.Contains(value, "GITHUB_TOKEN_REQUIRED") {
					if token, err := parser.GetGitHubToken(); err != nil {
						return fmt.Errorf("GitHub token required but not available: %w", err)
					} else {
						config.Env[key] = token
					}
				}
			} else {
				// For direct environment variable values (not containing ${}),
				// check if they represent actual token values
				if value == "" {
					return fmt.Errorf("environment variable '%s' has empty value", key)
				}
				// If value contains "GITHUB_TOKEN_REQUIRED", treat it as needing validation
				if strings.Contains(value, "GITHUB_TOKEN_REQUIRED") {
					if token, err := parser.GetGitHubToken(); err != nil {
						return fmt.Errorf("GitHub token required but not available: %w", err)
					} else {
						config.Env[key] = token
					}
				} else {
					// Automatically try to get GitHub token for GitHub-related environment variables
					if key == "GITHUB_PERSONAL_ACCESS_TOKEN" || key == "GITHUB_TOKEN" || key == "GH_TOKEN" {
						if actualValue := os.Getenv(key); actualValue == "" {
							// Try to automatically get the GitHub token
							if token, err := parser.GetGitHubToken(); err == nil {
								config.Env[key] = token
							} else {
								return fmt.Errorf("GitHub token required for '%s' but not available: %w", key, err)
							}
						}
					} else {
						// For backward compatibility: check if environment variable with this name exists
						// This preserves the original behavior for existing tests
						if actualValue := os.Getenv(key); actualValue == "" {
							return fmt.Errorf("environment variable '%s' not set", key)
						}
					}
				}
			}
		}
		return nil
	}

	// Check availability of required secrets
	secretsStatus := checkSecretsAvailability(requiredSecrets, useActionsSecrets)

	// Separate secrets by availability
	var availableSecrets []SecretInfo
	var missingSecrets []SecretInfo

	for _, secret := range secretsStatus {
		if secret.Available {
			availableSecrets = append(availableSecrets, secret)
		} else {
			missingSecrets = append(missingSecrets, secret)
		}
	}

	// Display information about secrets
	if verbose {
		if len(availableSecrets) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d available secret(s):", len(availableSecrets))))
			for _, secret := range availableSecrets {
				source := "environment"
				if secret.Source == "actions" {
					source = "GitHub Actions"
				}
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ %s (from %s)", secret.Name, source)))
			}
		}
	}

	// Warn about missing secrets
	if len(missingSecrets) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  %d required secret(s) not found:", len(missingSecrets))))
		for _, secret := range missingSecrets {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  ✗ %s", secret.Name)))
		}
	}

	return nil
}

// connectToMCPServer establishes a connection to the MCP server and queries its capabilities
func connectToMCPServer(config parser.MCPServerConfig, verbose bool) (*parser.MCPServerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch config.Type {
	case "stdio":
		return connectStdioMCPServer(ctx, config, verbose)
	case "docker":
		// Docker MCP servers are treated as stdio servers that run via docker command
		return connectStdioMCPServer(ctx, config, verbose)
	case "http":
		return connectHTTPMCPServer(ctx, config, verbose)
	default:
		return nil, fmt.Errorf("unsupported MCP server type: %s", config.Type)
	}
}

// connectStdioMCPServer connects to a stdio-based MCP server using the Go SDK
func connectStdioMCPServer(ctx context.Context, config parser.MCPServerConfig, verbose bool) (*parser.MCPServerInfo, error) {
	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Starting stdio MCP server: %s %s", config.Command, strings.Join(config.Args, " "))))
	}

	// Validate the command exists
	if config.Command != "" {
		if _, err := exec.LookPath(config.Command); err != nil {
			return nil, fmt.Errorf("command not found: %s", config.Command)
		}
	}

	// Create the command for the MCP server
	var cmd *exec.Cmd
	if config.Container != "" {
		// Docker container mode
		args := append([]string{"run", "--rm", "-i"}, config.Args...)
		cmd = exec.Command("docker", args...)
	} else {
		// Direct command mode
		cmd = exec.Command(config.Command, config.Args...)
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range config.Env {
		// Resolve environment variable references
		resolvedValue := os.ExpandEnv(value)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, resolvedValue))
	}

	// Create MCP client and connect
	client := mcp.NewClient(&mcp.Implementation{Name: "gh-aw-inspector", Version: "1.0.0"}, nil)
	transport := &mcp.CommandTransport{Command: cmd}

	// Create a timeout context for connection
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	if verbose {
		fmt.Println(console.FormatSuccessMessage("Successfully connected to MCP server"))
	}

	// Query server capabilities
	info := &parser.MCPServerInfo{
		Config:    config,
		Connected: true,
		Tools:     []*mcp.Tool{},
		Resources: []*mcp.Resource{},
		Roots:     []*mcp.Root{},
	}

	// List tools
	listToolsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	toolsResult, err := session.ListTools(listToolsCtx, &mcp.ListToolsParams{})
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to list tools: %v", err)))
		}
	} else {
		info.Tools = append(info.Tools, toolsResult.Tools...)
	}

	// List resources
	listResourcesCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resourcesResult, err := session.ListResources(listResourcesCtx, &mcp.ListResourcesParams{})
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to list resources: %v", err)))
		}
	} else {
		info.Resources = append(info.Resources, resourcesResult.Resources...)
	}

	// Note: Roots are not directly available via MCP protocol in the current spec,
	// so we'll keep an empty list or try to infer from resources
	for _, resource := range info.Resources {
		// Simple heuristic: extract root URIs from resources
		if strings.Contains(resource.URI, "://") {
			parts := strings.SplitN(resource.URI, "://", 2)
			if len(parts) == 2 {
				rootURI := parts[0] + "://"
				// Check if we already have this root
				found := false
				for _, root := range info.Roots {
					if root.URI == rootURI {
						found = true
						break
					}
				}
				if !found {
					info.Roots = append(info.Roots, &mcp.Root{
						URI:  rootURI,
						Name: parts[0],
					})
				}
			}
		}
	}

	return info, nil
}

// connectHTTPMCPServer connects to an HTTP-based MCP server using the Go SDK
func connectHTTPMCPServer(ctx context.Context, config parser.MCPServerConfig, verbose bool) (*parser.MCPServerInfo, error) {
	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Connecting to HTTP MCP server: %s", config.URL)))
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{Name: "gh-aw-inspector", Version: "1.0.0"}, nil)

	// Create streamable client transport for HTTP
	transport := &mcp.StreamableClientTransport{
		Endpoint: config.URL,
	}

	// Create a timeout context for connection
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HTTP MCP server: %w", err)
	}
	defer session.Close()

	if verbose {
		fmt.Println(console.FormatSuccessMessage("Successfully connected to HTTP MCP server"))
	}

	// Query server capabilities
	info := &parser.MCPServerInfo{
		Config:    config,
		Connected: true,
		Tools:     []*mcp.Tool{},
		Resources: []*mcp.Resource{},
		Roots:     []*mcp.Root{},
	}

	// List tools
	listToolsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	toolsResult, err := session.ListTools(listToolsCtx, &mcp.ListToolsParams{})
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to list tools: %v", err)))
		}
	} else {
		info.Tools = append(info.Tools, toolsResult.Tools...)
	}

	// List resources
	listResourcesCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resourcesResult, err := session.ListResources(listResourcesCtx, &mcp.ListResourcesParams{})
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to list resources: %v", err)))
		}
	} else {
		info.Resources = append(info.Resources, resourcesResult.Resources...)
	}

	// Extract root URIs from resources (simple heuristic)
	for _, resource := range info.Resources {
		if strings.Contains(resource.URI, "://") {
			parts := strings.SplitN(resource.URI, "://", 2)
			if len(parts) == 2 {
				rootURI := parts[0] + "://"
				// Check if we already have this root
				found := false
				for _, root := range info.Roots {
					if root.URI == rootURI {
						found = true
						break
					}
				}
				if !found {
					info.Roots = append(info.Roots, &mcp.Root{
						URI:  rootURI,
						Name: parts[0],
					})
				}
			}
		}
	}

	return info, nil
}

// displayServerCapabilities shows the server's tools, resources, and roots in formatted tables
func displayServerCapabilities(info *parser.MCPServerInfo, toolFilter string) {
	// Display tools with allowed/not allowed status
	if len(info.Tools) > 0 {
		// If a specific tool is requested, show detailed information
		if toolFilter != "" {
			displayDetailedToolInfo(info, toolFilter)
		} else {
			fmt.Printf("\n%s\n", headerStyle.Render("🛠️  Tool Access Status"))

			// Create a map for quick lookup of allowed tools
			allowedMap := make(map[string]bool)
			for _, allowed := range info.Config.Allowed {
				allowedMap[allowed] = true
			}

			headers := []string{"Tool Name", "Allow", "Description"}
			rows := make([][]string, 0, len(info.Tools))

			for _, tool := range info.Tools {
				description := tool.Description
				if len(description) > 50 {
					description = description[:47] + "..."
				}

				// Determine status
				status := "🚫"
				if len(info.Config.Allowed) == 0 {
					// If no allowed list is specified, assume all tools are allowed
					status = "✅"
				} else if allowedMap[tool.Name] {
					status = "✅"
				}

				rows = append(rows, []string{tool.Name, status, description})
			}

			table := console.RenderTable(console.TableConfig{
				Headers: headers,
				Rows:    rows,
			})
			fmt.Print(table)

			// Display summary
			allowedCount := 0
			for _, tool := range info.Tools {
				if len(info.Config.Allowed) == 0 || allowedMap[tool.Name] {
					allowedCount++
				}
			}
			fmt.Printf("\n📊 Summary: %d allowed, %d not allowed out of %d total tools\n",
				allowedCount, len(info.Tools)-allowedCount, len(info.Tools))

			// Add helpful hint about how to allow tools in workflow frontmatter
			displayToolAllowanceHint(info)
		}

	} else {
		if toolFilter != "" {
			fmt.Printf("\n%s\n", console.FormatWarningMessage(fmt.Sprintf("Tool '%s' not found", toolFilter)))
		} else {
			fmt.Printf("\n%s\n", console.FormatWarningMessage("No tools available"))
		}
	}

	// Display resources (skip if showing specific tool details)
	if toolFilter == "" && len(info.Resources) > 0 {
		fmt.Printf("\n%s\n", headerStyle.Render("📚 Available Resources"))

		headers := []string{"URI", "Name", "Description", "MIME Type"}
		rows := make([][]string, 0, len(info.Resources))

		for _, resource := range info.Resources {
			description := resource.Description
			if len(description) > 40 {
				description = description[:37] + "..."
			}

			mimeType := resource.MIMEType
			if mimeType == "" {
				mimeType = "N/A"
			}

			rows = append(rows, []string{resource.URI, resource.Name, description, mimeType})
		}

		table := console.RenderTable(console.TableConfig{
			Headers: headers,
			Rows:    rows,
		})
		fmt.Print(table)
	} else if toolFilter == "" {
		fmt.Printf("\n%s\n", console.FormatWarningMessage("No resources available"))
	}

	// Display roots (skip if showing specific tool details)
	if toolFilter == "" && len(info.Roots) > 0 {
		fmt.Printf("\n%s\n", headerStyle.Render("🌳 Available Roots"))

		headers := []string{"URI", "Name"}
		rows := make([][]string, 0, len(info.Roots))

		for _, root := range info.Roots {
			rows = append(rows, []string{root.URI, root.Name})
		}

		table := console.RenderTable(console.TableConfig{
			Headers: headers,
			Rows:    rows,
		})
		fmt.Print(table)
	} else if toolFilter == "" {
		fmt.Printf("\n%s\n", console.FormatWarningMessage("No roots available"))
	}

	fmt.Println()
}

// displayDetailedToolInfo shows detailed information about a specific tool
func displayDetailedToolInfo(info *parser.MCPServerInfo, toolName string) {
	// Find the specific tool
	var foundTool *mcp.Tool
	for _, tool := range info.Tools {
		if tool.Name == toolName {
			foundTool = tool
			break
		}
	}

	if foundTool == nil {
		fmt.Printf("\n%s\n", console.FormatWarningMessage(fmt.Sprintf("Tool '%s' not found", toolName)))
		fmt.Printf("Available tools: ")
		toolNames := make([]string, len(info.Tools))
		for i, tool := range info.Tools {
			toolNames[i] = tool.Name
		}
		fmt.Printf("%s\n", strings.Join(toolNames, ", "))
		return
	}

	// Check if tool is allowed
	isAllowed := len(info.Config.Allowed) == 0 // Default to allowed if no allowlist
	for _, allowed := range info.Config.Allowed {
		if allowed == toolName {
			isAllowed = true
			break
		}
	}

	fmt.Printf("\n%s\n", headerStyle.Render(fmt.Sprintf("🛠️  Tool Details: %s", foundTool.Name)))

	// Display basic information
	fmt.Printf("📋 **Name:** %s\n", foundTool.Name)

	// Show title if available and different from name
	if foundTool.Title != "" && foundTool.Title != foundTool.Name {
		fmt.Printf("📄 **Title:** %s\n", foundTool.Title)
	}
	if foundTool.Annotations != nil && foundTool.Annotations.Title != "" && foundTool.Annotations.Title != foundTool.Name && foundTool.Annotations.Title != foundTool.Title {
		fmt.Printf("📄 **Annotation Title:** %s\n", foundTool.Annotations.Title)
	}

	fmt.Printf("📝 **Description:** %s\n", foundTool.Description)

	// Display allowance status
	if isAllowed {
		fmt.Printf("✅ **Status:** Allowed\n")
	} else {
		fmt.Printf("🚫 **Status:** Not allowed (add to 'allowed' list in workflow frontmatter)\n")
	}

	// Display annotations if available
	if foundTool.Annotations != nil {
		fmt.Printf("\n%s\n", headerStyle.Render("⚙️  Tool Attributes"))

		if foundTool.Annotations.ReadOnlyHint {
			fmt.Printf("🔒 **Read-only:** This tool does not modify its environment\n")
		} else {
			fmt.Printf("🔓 **Modifies environment:** This tool can make changes\n")
		}

		if foundTool.Annotations.IdempotentHint {
			fmt.Printf("🔄 **Idempotent:** Calling with same arguments has no additional effect\n")
		}

		if foundTool.Annotations.DestructiveHint != nil {
			if *foundTool.Annotations.DestructiveHint {
				fmt.Printf("⚠️  **Destructive:** May perform destructive updates\n")
			} else {
				fmt.Printf("➕ **Additive:** Performs only additive updates\n")
			}
		}

		if foundTool.Annotations.OpenWorldHint != nil {
			if *foundTool.Annotations.OpenWorldHint {
				fmt.Printf("🌐 **Open world:** Interacts with external entities\n")
			} else {
				fmt.Printf("🏠 **Closed world:** Domain of interaction is closed\n")
			}
		}
	}

	// Display input schema
	if foundTool.InputSchema != nil {
		fmt.Printf("\n%s\n", headerStyle.Render("📥 Input Schema"))
		if schemaJSON, err := json.MarshalIndent(foundTool.InputSchema, "", "  "); err == nil {
			fmt.Printf("```json\n%s\n```\n", string(schemaJSON))
		} else {
			fmt.Printf("Error displaying input schema: %v\n", err)
		}
	} else {
		fmt.Printf("\n%s\n", console.FormatInfoMessage("📥 No input schema defined"))
	}

	// Display output schema
	if foundTool.OutputSchema != nil {
		fmt.Printf("\n%s\n", headerStyle.Render("📤 Output Schema"))
		if schemaJSON, err := json.MarshalIndent(foundTool.OutputSchema, "", "  "); err == nil {
			fmt.Printf("```json\n%s\n```\n", string(schemaJSON))
		} else {
			fmt.Printf("Error displaying output schema: %v\n", err)
		}
	} else {
		fmt.Printf("\n%s\n", console.FormatInfoMessage("📤 No output schema defined"))
	}

	fmt.Println()
}

// displayToolAllowanceHint shows helpful information about how to allow tools in workflow frontmatter
func displayToolAllowanceHint(info *parser.MCPServerInfo) {
	// Create a map for quick lookup of allowed tools
	allowedMap := make(map[string]bool)
	for _, allowed := range info.Config.Allowed {
		allowedMap[allowed] = true
	}

	// Count blocked tools and collect their names
	var blockedTools []string
	for _, tool := range info.Tools {
		if len(info.Config.Allowed) > 0 && !allowedMap[tool.Name] {
			blockedTools = append(blockedTools, tool.Name)
		}
	}

	if len(blockedTools) > 0 {
		fmt.Printf("\n%s\n", console.FormatInfoMessage("💡 To allow blocked tools, add them to your workflow frontmatter:"))

		// Show the frontmatter syntax example
		fmt.Printf("\n")
		fmt.Printf("```yaml\n")
		fmt.Printf("tools:\n")
		fmt.Printf("  %s:\n", info.Config.Name)
		fmt.Printf("    allowed:\n")

		// Add currently allowed tools first (if any)
		for _, allowed := range info.Config.Allowed {
			fmt.Printf("      - %s\n", allowed)
		}

		// Show first few blocked tools as examples (limit to 3 for readability)
		exampleCount := len(blockedTools)
		if exampleCount > 3 {
			exampleCount = 3
		}

		for i := 0; i < exampleCount; i++ {
			fmt.Printf("      - %s\n", blockedTools[i])
		}

		if len(blockedTools) > 3 {
			fmt.Printf("      # ... and %d more tools\n", len(blockedTools)-3)
		}

		fmt.Printf("```\n")

		if len(blockedTools) > 3 {
			fmt.Printf("\n%s\n", console.FormatInfoMessage(fmt.Sprintf("📋 All blocked tools: %s", strings.Join(blockedTools, ", "))))
		}
	} else if len(info.Config.Allowed) == 0 {
		// No explicit allowed list - all tools are allowed by default
		fmt.Printf("\n%s\n", console.FormatInfoMessage("💡 All tools are currently allowed (no 'allowed' list specified)"))
		if len(info.Tools) > 0 {
			fmt.Printf("\n%s\n", console.FormatInfoMessage("To restrict tools, add an 'allowed' list to your workflow frontmatter:"))
			fmt.Printf("\n")
			fmt.Printf("```yaml\n")
			fmt.Printf("tools:\n")
			fmt.Printf("  %s:\n", info.Config.Name)
			fmt.Printf("    allowed:\n")
			fmt.Printf("      - %s  # Allow only specific tools\n", info.Tools[0].Name)
			if len(info.Tools) > 1 {
				fmt.Printf("      - %s\n", info.Tools[1].Name)
			}
			fmt.Printf("```\n")
		}
	} else {
		// All tools are explicitly allowed
		fmt.Printf("\n%s\n", console.FormatSuccessMessage("✅ All available tools are explicitly allowed in your workflow"))
	}

	fmt.Printf("\n%s\n", console.FormatInfoMessage("📖 For more information, see: https://github.com/githubnext/gh-aw/blob/main/docs/tools.md"))
}
