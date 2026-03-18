package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var publishAssetsLog = logger.New("workflow:publish_assets")

// UploadAssetsConfig holds configuration for uploading assets as GitHub Actions artifacts
type UploadAssetsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	MaxSizeKB            int      `yaml:"max-size,omitempty"`     // Maximum file size in KB (default: 10240 = 10MB)
	AllowedExts          []string `yaml:"allowed-exts,omitempty"` // Allowed file extensions (default: common image types)
}

// parseUploadAssetConfig handles upload-asset configuration
func (c *Compiler) parseUploadAssetConfig(outputMap map[string]any) *UploadAssetsConfig {
	if configData, exists := outputMap["upload-asset"]; exists {
		publishAssetsLog.Print("Parsing upload-asset configuration")
		config := &UploadAssetsConfig{
			MaxSizeKB: 10240, // Default 10MB
			AllowedExts: []string{
				// Default set of extensions as specified in problem statement
				".png",
				".jpg",
				".jpeg",
			},
		}

		if configMap, ok := configData.(map[string]any); ok {
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

			// Parse common base fields with default max of 0 (no limit)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 0)
			publishAssetsLog.Printf("Parsed upload-asset config: max_size_kb=%d, allowed_exts=%d", config.MaxSizeKB, len(config.AllowedExts))
		} else if configData == nil {
			// Handle null case: create config with defaults
			publishAssetsLog.Print("Using default upload-asset configuration")
			return config
		}

		return config
	}

	return nil
}

// buildUploadAssetsJob creates the upload_assets job that uploads staged asset files
// as unzipped GitHub Actions artifacts (archive:false) and outputs the URL map.
func (c *Compiler) buildUploadAssetsJob(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) (*Job, error) {
	publishAssetsLog.Printf("Building upload_assets job: workflow=%s, main_job=%s, threat_detection=%v", data.Name, mainJobName, threatDetectionEnabled)

	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return nil, errors.New("safe-outputs.upload-asset configuration is required")
	}

	var preSteps []string

	// Add setup step to copy scripts
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		preSteps = append(preSteps, c.generateCheckoutActionsFolder(data)...)

		// Upload assets job doesn't need project support
		preSteps = append(preSteps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)
	}

	// Download assets artifact so upload_assets.cjs can read the staged files.
	// In workflow_call context, use the per-invocation prefix from the agent job.
	assetsArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	preSteps = append(preSteps, "      - name: Download assets\n")
	preSteps = append(preSteps, "        continue-on-error: true\n") // Continue if no assets were uploaded
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, fmt.Sprintf("          name: %ssafe-outputs-assets\n", assetsArtifactPrefix))
	preSteps = append(preSteps, "          path: /tmp/gh-aw/safeoutputs/assets/\n")

	// List downloaded files for debugging
	preSteps = append(preSteps, "      - name: List downloaded asset files\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, "        run: |\n")
	preSteps = append(preSteps, "          echo \"Downloaded asset files:\"\n")
	preSteps = append(preSteps, "          find /tmp/gh-aw/safeoutputs/assets/ -maxdepth 1 -ls\n")

	// Environment variables needed by upload_assets.cjs
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSETS_MAX_SIZE_KB: %d\n", data.SafeOutputs.UploadAssets.MaxSizeKB))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSETS_ALLOWED_EXTS: %q\n", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ",")))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	// Job outputs: asset URL map (temporaryId -> artifactUrl) consumed by safe_outputs job
	outputs := map[string]string{
		"upload_count":  "${{ steps.upload_assets.outputs.upload_count }}",
		"asset_url_map": "${{ steps.upload_assets.outputs.asset_url_map }}",
	}

	// Build the job condition using expression tree
	jobCondition := BuildSafeOutputType("upload_asset")

	// Build job dependencies — detection is now inline in the agent job
	needs := []string{mainJobName}

	// Use the shared builder function to create the job.
	// The job no longer needs contents:write permission since it uploads artifacts,
	// not git content. Use actions:write for artifact creation.
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "upload_assets",
		StepName:      "Upload assets",
		StepID:        "upload_assets",
		ScriptName:    "upload_assets",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getUploadAssetsScript(),
		Permissions:   NewPermissionsActionsWrite(),
		Outputs:       outputs,
		Condition:     jobCondition,
		PreSteps:      preSteps,
		Token:         data.SafeOutputs.UploadAssets.GitHubToken,
		Needs:         needs,
	})
}

// generateSafeOutputsAssetsArtifactUpload generates a step to upload safe-outputs assets as a separate artifact
// This artifact is then downloaded by the upload_assets job to publish files to orphaned branches.
// In workflow_call context, the artifact name is prefixed to avoid clashes.
func generateSafeOutputsAssetsArtifactUpload(builder *strings.Builder, data *WorkflowData) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return
	}

	publishAssetsLog.Print("Generating safe-outputs assets artifact upload step")

	// In workflow_call context, apply the per-invocation prefix to avoid artifact name clashes.
	prefix := artifactPrefixExprForDownstreamJob(data)

	builder.WriteString("      # Upload safe-outputs assets for upload_assets job\n")
	builder.WriteString("      - name: Upload Safe Outputs Assets\n")
	builder.WriteString("        if: always()\n")
	fmt.Fprintf(builder, "        uses: %s\n", GetActionPin("actions/upload-artifact"))
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          name: %ssafe-outputs-assets\n", prefix)
	builder.WriteString("          path: /tmp/gh-aw/safeoutputs/assets/\n")
	builder.WriteString("          retention-days: 1\n")
	builder.WriteString("          if-no-files-found: ignore\n")
}
