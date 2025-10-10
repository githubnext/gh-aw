package workflow

import (
	"fmt"
	"strings"
)

// UploadAssetsConfig holds configuration for publishing assets to an orphaned git branch
type UploadAssetsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	BranchName           string   `yaml:"branch,omitempty"`       // Branch name (default: "assets/${{ github.workflow }}")
	MaxSizeKB            int      `yaml:"max-size,omitempty"`     // Maximum file size in KB (default: 10240 = 10MB)
	AllowedExts          []string `yaml:"allowed-exts,omitempty"` // Allowed file extensions (default: common non-executable types)
}

// parseUploadAssetConfig handles upload-asset configuration
func (c *Compiler) parseUploadAssetConfig(outputMap map[string]any) *UploadAssetsConfig {
	if configData, exists := outputMap["upload-assets"]; exists {
		config := &UploadAssetsConfig{
			BranchName: "assets/${{ github.workflow }}", // Default branch name
			MaxSizeKB:  10240,                           // Default 10MB
			AllowedExts: []string{
				// Default set of extensions as specified in problem statement
				".png",
				".jpg",
				".jpeg",
			},
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse branch
			if branchName, exists := configMap["branch"]; exists {
				if branchNameStr, ok := branchName.(string); ok {
					config.BranchName = branchNameStr
				}
			}

			// Parse max-size
			if maxSize, exists := configMap["max-size"]; exists {
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

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		} else if configData == nil {
			// Handle null case: create config with defaults
			return config
		}

		return config
	}

	return nil
}

// buildUploadAssetsJob creates the publish_assets job
func (c *Compiler) buildUploadAssetsJob(data *WorkflowData, mainJobName string, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return nil, fmt.Errorf("safe-outputs.upload-asset configuration is required")
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

	// Step 2: Checkout repository
	steps = buildCheckoutRepository(steps, c)

	// Step 3: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Add step to download assets artifact if it exists
	steps = append(steps, "      - name: Download assets\n")
	steps = append(steps, "        continue-on-error: true\n") // Continue if no assets were uploaded
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: safe-outputs-assets\n")
	steps = append(steps, "          path: /tmp/gh-aw/safe-outputs/assets/\n")

	// list files
	steps = append(steps, "      - name: List downloaded asset files\n")
	steps = append(steps, "        continue-on-error: true\n") // Continue if no assets were uploaded
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Downloaded asset files:\"\n")
	steps = append(steps, "          ls -la /tmp/gh-aw/safe-outputs/assets/\n")

	// Step 4: Upload assets to orphaned branch using custom script
	steps = append(steps, "      - name: Upload Assets to Orphaned Branch\n")
	steps = append(steps, "        id: upload_assets\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_ASSETS_BRANCH: %q\n", data.SafeOutputs.UploadAssets.BranchName))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_ASSETS_MAX_SIZE_KB: %d\n", data.SafeOutputs.UploadAssets.MaxSizeKB))
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_ASSETS_ALLOWED_EXTS: %q\n", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ",")))

	// Pass the staged flag if it's set to true
	if c.trialMode || data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}
	if c.trialMode && c.trialApparentRepoSlug != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q\n", c.trialApparentRepoSlug))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.UploadAssets != nil {
		token = data.SafeOutputs.UploadAssets.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(uploadAssetsScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"published_count": "${{ steps.upload_assets.outputs.published_count }}",
		"branch_name":     "${{ steps.upload_assets.outputs.branch_name }}",
	}

	// Build the job condition using expression tree
	jobCondition := BuildSafeOutputType("upload-asset", data.SafeOutputs.UploadAssets.Min)

	// Set base permissions
	permissions := "permissions:\n      contents: write  # Required for creating orphaned branch and pushing assets"

	job := &Job{
		Name:           "upload_assets",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
