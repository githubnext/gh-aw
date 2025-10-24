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
func (c *Compiler) buildUploadAssetsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return nil, fmt.Errorf("safe-outputs.upload-asset configuration is required")
	}

	var steps []string

	// Permission checks are now handled by the separate check_membership job
	// which is always created when needed (when activation job is created)

	// Step 1: Checkout repository
	steps = buildCheckoutRepository(steps, c)

	// Step 2: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Step 3: Download assets artifact if it exists
	steps = append(steps, "      - name: Download assets\n")
	steps = append(steps, "        continue-on-error: true\n") // Continue if no assets were uploaded
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact", "v5")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: safe-outputs-assets\n")
	steps = append(steps, "          path: /tmp/gh-aw/safeoutputs/assets/\n")

	// Step 4: List files
	steps = append(steps, "      - name: List downloaded asset files\n")
	steps = append(steps, "        continue-on-error: true\n") // Continue if no assets were uploaded
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Downloaded asset files:\"\n")
	steps = append(steps, "          ls -la /tmp/gh-aw/safeoutputs/assets/\n")

	// Build custom environment variables specific to upload-assets
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSETS_BRANCH: %q\n", data.SafeOutputs.UploadAssets.BranchName))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSETS_MAX_SIZE_KB: %d\n", data.SafeOutputs.UploadAssets.MaxSizeKB))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSETS_ALLOWED_EXTS: %q\n", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ",")))

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // No target repo for upload assets
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.UploadAssets != nil {
		token = data.SafeOutputs.UploadAssets.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Upload Assets to Orphaned Branch",
		StepID:        "upload_assets",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        uploadAssetsScript,
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Create outputs for the job
	outputs := map[string]string{
		"published_count": "${{ steps.upload_assets.outputs.published_count }}",
		"branch_name":     "${{ steps.upload_assets.outputs.branch_name }}",
	}

	// Build the job condition using expression tree
	jobCondition := BuildSafeOutputType("upload_asset", data.SafeOutputs.UploadAssets.Min)

	job := &Job{
		Name:           "upload_assets",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
