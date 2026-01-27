package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var projectSafeOutputsLog = logger.New("workflow:project_safe_outputs")

// applyProjectSafeOutputs checks for a project field in the frontmatter and automatically
// configures safe-outputs for project tracking when present. This provides the same
// project tracking behavior that campaign orchestrators have.
//
// When a project field is detected:
// - Automatically adds update-project safe-output if not already configured
// - Automatically adds create-project-status-update safe-output if not already configured
// - Applies project-specific settings (max-updates, github-token, etc.)
func (c *Compiler) applyProjectSafeOutputs(frontmatter map[string]any, existingSafeOutputs *SafeOutputsConfig) *SafeOutputsConfig {
	projectSafeOutputsLog.Print("Checking for project field in frontmatter")

	// Check if project field exists
	projectData, hasProject := frontmatter["project"]
	if !hasProject || projectData == nil {
		projectSafeOutputsLog.Print("No project field found in frontmatter")
		return existingSafeOutputs
	}

	projectSafeOutputsLog.Print("Project field found, parsing configuration")

	// Parse project configuration
	var projectConfig *ProjectConfig
	if projectMap, ok := projectData.(map[string]any); ok {
		projectConfig = c.parseProjectConfig(projectMap)
	} else if projectStr, ok := projectData.(string); ok {
		// Simple string format: just a URL
		projectConfig = &ProjectConfig{
			URL: projectStr,
		}
	} else {
		projectSafeOutputsLog.Print("Invalid project field format, skipping")
		return existingSafeOutputs
	}

	if projectConfig == nil || projectConfig.URL == "" {
		projectSafeOutputsLog.Print("No valid project URL found, skipping")
		return existingSafeOutputs
	}

	projectSafeOutputsLog.Printf("Project URL configured: %s", projectConfig.URL)

	// Create or update SafeOutputsConfig
	safeOutputs := existingSafeOutputs
	if safeOutputs == nil {
		safeOutputs = &SafeOutputsConfig{}
		projectSafeOutputsLog.Print("Created new SafeOutputsConfig for project tracking")
	}

	// Apply defaults if not specified
	maxUpdates := projectConfig.MaxUpdates
	if maxUpdates == 0 {
		maxUpdates = 100 // Default for project updates (same as campaign orchestrators)
	}

	maxStatusUpdates := projectConfig.MaxStatusUpdates
	if maxStatusUpdates == 0 {
		maxStatusUpdates = 1 // Default for status updates
	}

	// Configure update-project if not already configured
	if safeOutputs.UpdateProjects == nil {
		projectSafeOutputsLog.Printf("Adding update-project safe-output (max: %d)", maxUpdates)
		safeOutputs.UpdateProjects = &UpdateProjectConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Max: maxUpdates,
			},
			GitHubToken: projectConfig.GitHubToken,
		}
	} else {
		projectSafeOutputsLog.Print("update-project already configured, preserving existing configuration")
	}

	// Configure create-project-status-update if not already configured
	if safeOutputs.CreateProjectStatusUpdates == nil {
		projectSafeOutputsLog.Printf("Adding create-project-status-update safe-output (max: %d)", maxStatusUpdates)
		safeOutputs.CreateProjectStatusUpdates = &CreateProjectStatusUpdateConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Max: maxStatusUpdates,
			},
			GitHubToken: projectConfig.GitHubToken,
		}
	} else {
		projectSafeOutputsLog.Print("create-project-status-update already configured, preserving existing configuration")
	}

	return safeOutputs
}

// parseProjectConfig parses project configuration from a map
func (c *Compiler) parseProjectConfig(projectMap map[string]any) *ProjectConfig {
	config := &ProjectConfig{}

	// Parse URL (required)
	if url, exists := projectMap["url"]; exists {
		if urlStr, ok := url.(string); ok {
			config.URL = urlStr
		}
	}

	// Parse scope (optional)
	if scope, exists := projectMap["scope"]; exists {
		if scopeList, ok := scope.([]any); ok {
			for _, item := range scopeList {
				if scopeStr, ok := item.(string); ok {
					config.Scope = append(config.Scope, scopeStr)
				}
			}
		}
	}

	// Parse max-updates (optional)
	if maxUpdates, exists := projectMap["max-updates"]; exists {
		switch v := maxUpdates.(type) {
		case int:
			config.MaxUpdates = v
		case float64:
			config.MaxUpdates = int(v)
		}
	}

	// Parse max-status-updates (optional)
	if maxStatusUpdates, exists := projectMap["max-status-updates"]; exists {
		switch v := maxStatusUpdates.(type) {
		case int:
			config.MaxStatusUpdates = v
		case float64:
			config.MaxStatusUpdates = int(v)
		}
	}

	// Parse github-token (optional)
	if token, exists := projectMap["github-token"]; exists {
		if tokenStr, ok := token.(string); ok {
			config.GitHubToken = tokenStr
		}
	}

	// Parse do-not-downgrade-done-items (optional)
	if doNotDowngrade, exists := projectMap["do-not-downgrade-done-items"]; exists {
		if doNotDowngradeBool, ok := doNotDowngrade.(bool); ok {
			config.DoNotDowngradeDoneItems = &doNotDowngradeBool
		}
	}

	return config
}
