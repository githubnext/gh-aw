package workflow

import (
	"fmt"
	"strings"
)

// PublishAssetsConfig holds configuration for publishing assets to an orphaned git branch
type PublishAssetsConfig struct {
	BranchName  string   `yaml:"branch,omitempty"`   // Branch name (default: "assets/${{ github.workflow }}")
	MaxSizeKB   int      `yaml:"max-size-kb,omitempty"`   // Maximum file size in KB (default: 10240 = 10MB)
	AllowedExts []string `yaml:"allowed-exts,omitempty"`  // Allowed file extensions (default: common non-executable types)
	GitHubToken string   `yaml:"github-token,omitempty"`  // GitHub token for this specific output type
}

// parsePublishAssetsConfig handles publish-assets configuration
func (c *Compiler) parsePublishAssetsConfig(outputMap map[string]any) *PublishAssetsConfig {
	if configData, exists := outputMap["publish-assets"]; exists {
		config := &PublishAssetsConfig{
			BranchName:  "assets/${{ github.workflow }}", // Default branch name
			MaxSizeKB:   10240,                           // Default 10MB
			AllowedExts: getDefaultAllowedExtensions(),   // Default safe extensions
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse branch
			if branchName, exists := configMap["branch"]; exists {
				if branchNameStr, ok := branchName.(string); ok {
					config.BranchName = branchNameStr
				}
			}

			// Parse max-size-kb
			if maxSize, exists := configMap["max-size-kb"]; exists {
				if maxSizeInt, ok := parseIntValue(maxSize); ok && maxSizeInt > 0 {
					config.MaxSizeKB = maxSizeInt
				}
			}

			// Parse allowed-exts
			if allowedExts, exists := configMap["allowed-exts"]; exists {
				if allowedExtsArray, ok := allowedExts.([]any); ok {
					var extStrings []string
					for _, ext := range allowedExtsArray {
						if extStr, ok := ext.(string); ok {
							extStrings = append(extStrings, extStr)
						}
					}
					if len(extStrings) > 0 {
						config.AllowedExts = extStrings
					}
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					config.GitHubToken = githubTokenStr
				}
			}
		} else if configData == nil {
			// Handle null case: create config with defaults
			return config
		}

		return config
	}

	return nil
}

// getDefaultAllowedExtensions returns a list of safe file extensions for asset publishing
func getDefaultAllowedExtensions() []string {
	return []string{
		// Images
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".bmp", ".ico",
		// Documents
		".pdf", ".txt", ".md", ".rtf", ".doc", ".docx",
		// Data formats
		".json", ".yaml", ".yml", ".xml", ".csv", ".tsv",
		// Web assets
		".css", ".js", ".html", ".htm",
		// Archives (non-executable)
		".zip", ".tar", ".gz", ".bz2",
		// Audio/Video
		".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".ogg",
		// Fonts
		".ttf", ".otf", ".woff", ".woff2", ".eot",
		// Other safe formats
		".log", ".conf", ".cfg", ".ini",
	}
}

// buildPublishAssetsJob creates the publish_assets job
func (c *Compiler) buildPublishAssetsJob(data *WorkflowData, mainJobName string, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.PublishAssets == nil {
		return nil, fmt.Errorf("safe-outputs.publish-asset configuration is required")
	}

	var steps []string

	// Add permission checks if no task job was created but permission checks are needed
	if !taskJobCreated && c.needsRoleCheck(data, frontmatter) {
		// Add team member check step
		steps = append(steps, "      - name: Check team membership for workflow\n")
		steps = append(steps, "        id: check-team-member\n")
		steps = append(steps, "        uses: actions/github-script@v8\n")

		// Add environment variables for permission check
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_REQUIRED_ROLES: %s\n", strings.Join(data.Roles, ",")))

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Generate the JavaScript code for the permission check
		scriptContent := c.generateRoleCheckScript(data.Roles)
		scriptLines := strings.Split(scriptContent, "\n")
		for _, line := range scriptLines {
			if strings.TrimSpace(line) != "" {
				steps = append(steps, fmt.Sprintf("            %s\n", line))
			}
		}
	}

	// Add step to download safe outputs from the main job
	steps = append(steps, "      - name: Download safe outputs\n")
	steps = append(steps, "        uses: actions/download-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", OutputArtifactName))
	steps = append(steps, "          path: /tmp/safe-outputs/\n")

	// Add step to download assets artifact if it exists
	steps = append(steps, "      - name: Download assets\n")
	steps = append(steps, "        uses: actions/download-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: safe-outputs-assets\n")
	steps = append(steps, "          path: /tmp/safe-outputs/assets/\n")
	steps = append(steps, "        continue-on-error: true\n") // Continue if no assets were uploaded

	steps = append(steps, "      - name: Publish Assets to Orphaned Branch\n")
	steps = append(steps, "        id: publish_assets\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_ASSETS_BRANCH: %q\n", data.SafeOutputs.PublishAssets.BranchName))

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.PublishAssets != nil {
		token = data.SafeOutputs.PublishAssets.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(publishAssetsScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"published_count": "${{ steps.publish_assets.outputs.published_count }}",
		"branch_name":     "${{ steps.publish_assets.outputs.branch_name }}",
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = commandConditionStr
	} else {
		jobCondition = "" // No conditional execution
	}

	// Set base permissions
	permissions := "permissions:\n      contents: write  # Required for creating orphaned branch and pushing assets"

	// Add actions: write permission if team member checks are present for command workflows
	_, hasExplicitRoles := frontmatter["roles"]
	requiresWorkflowCancellation := data.Command != "" ||
		(!taskJobCreated && c.needsRoleCheck(data, frontmatter) && hasExplicitRoles)

	if requiresWorkflowCancellation {
		permissions = "permissions:\n      actions: write  # Required for github.rest.actions.cancelWorkflowRun()\n      contents: write  # Required for creating orphaned branch and pushing assets"
	}

	job := &Job{
		Name:           "publish_assets",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
