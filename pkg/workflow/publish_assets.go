package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var publishAssetsLog = logger.New("workflow:publish_assets")
var githubExpressionPattern = regexp.MustCompile(`(?s)^\$\{\{.*\}\}$`)

func isGitHubExpression(value string) bool {
	trimmed := strings.TrimSpace(value)
	return githubExpressionPattern.MatchString(trimmed)
}

func normalizeAllowedExtension(extension string) string {
	trimmed := strings.TrimSpace(extension)
	if trimmed == "" {
		return ""
	}
	if isGitHubExpression(trimmed) {
		return trimmed
	}
	normalized := strings.ToLower(trimmed)
	if !strings.HasPrefix(normalized, ".") {
		normalized = "." + normalized
	}
	return normalized
}

// UploadAssetsConfig holds configuration for publishing assets to an orphaned git branch
type UploadAssetsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	BranchName           string   `yaml:"branch,omitempty"`       // Branch name (default: "assets/${{ github.workflow }}")
	MaxSizeKB            int      `yaml:"max-size,omitempty"`     // Maximum file size in KB (default: 10240 = 10MB)
	AllowedExts          []string `yaml:"allowed-exts,omitempty"` // Allowed file extensions (default: common non-executable types)
}

// parseUploadAssetConfig handles upload-asset configuration
func (c *Compiler) parseUploadAssetConfig(outputMap map[string]any) *UploadAssetsConfig {
	configData, exists := outputMap["upload-asset"]
	if !exists {
		return nil
	}
	if b, ok := configData.(bool); ok && !b {
		publishAssetsLog.Print("upload-asset explicitly set to false, skipping")
		return nil
	}
	publishAssetsLog.Print("Parsing upload-asset configuration")
	config := defaultUploadAssetsConfig()
	configMap, ok := configData.(map[string]any)
	if !ok {
		if configData == nil {
			publishAssetsLog.Print("Using default upload-asset configuration")
		}
		return config
	}
	applyUploadAssetConfigMap(c, configMap, config)
	return config
}

func defaultUploadAssetsConfig() *UploadAssetsConfig {
	return &UploadAssetsConfig{
		BranchName:  "assets/${{ github.workflow }}",
		MaxSizeKB:   10240,
		AllowedExts: []string{".png", ".jpg", ".jpeg"},
	}
}

func applyUploadAssetConfigMap(c *Compiler, configMap map[string]any, config *UploadAssetsConfig) {
	if branchName, exists := configMap["branch"]; exists {
		if branchNameStr, ok := branchName.(string); ok {
			config.BranchName = branchNameStr
		}
	}
	if maxSize, exists := configMap["max-size"]; exists {
		if maxSizeInt, ok := typeutil.ParseIntValue(maxSize); ok && maxSizeInt > 0 {
			config.MaxSizeKB = maxSizeInt
		}
	}
	if allowedExts, exists := configMap["allowed-exts"]; exists {
		if extStrings := parseAllowedUploadAssetExtensions(allowedExts); len(extStrings) > 0 {
			config.AllowedExts = extStrings
		}
	}
	c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 0)
	publishAssetsLog.Printf("Parsed upload-asset config: branch=%s, max_size_kb=%d, allowed_exts=%d", config.BranchName, config.MaxSizeKB, len(config.AllowedExts))
}

func parseAllowedUploadAssetExtensions(value any) []string {
	allowedExtsArray, ok := value.([]any)
	if !ok {
		return nil
	}
	extStrings := make([]string, 0, len(allowedExtsArray))
	seen := make(map[string]struct{})
	for _, ext := range allowedExtsArray {
		extStr, ok := ext.(string)
		if !ok {
			continue
		}
		normalized := normalizeAllowedExtension(extStr)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		extStrings = append(extStrings, normalized)
	}
	return extStrings
}

// buildUploadAssetsJob creates the publish_assets job
func (c *Compiler) buildUploadAssetsJob(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) (*Job, error) {
	publishAssetsLog.Printf("Building upload_assets job: workflow=%s, main_job=%s, threat_detection=%v", data.Name, mainJobName, threatDetectionEnabled)
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return nil, errors.New("safe-outputs.upload-asset configuration is required")
	}
	preSteps := c.buildUploadAssetsPreSteps(data)
	customEnvVars := c.buildUploadAssetsEnvVars(data)
	outputs := map[string]string{
		"published_count": "${{ steps.upload_assets.outputs.published_count }}",
		"branch_name":     "${{ steps.upload_assets.outputs.branch_name }}",
	}
	postSteps := c.buildUploadAssetsPostSteps()
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "upload_assets",
		StepName:      "Push assets",
		StepID:        "upload_assets",
		ScriptName:    "upload_assets",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getUploadAssetsScript(),
		Permissions:   NewPermissionsContentsWrite(),
		Outputs:       outputs,
		Condition:     uploadAssetsJobCondition(data),
		PreSteps:      preSteps,
		PostSteps:     postSteps,
		Token:         data.SafeOutputs.UploadAssets.GitHubToken,
		Needs:         []string{mainJobName, string(constants.ActivationJobName)},
	})
}

func (c *Compiler) buildUploadAssetsPreSteps(data *WorkflowData) []string {
	var preSteps []string
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		preSteps = append(preSteps, c.generateCheckoutActionsFolder(data)...)
		publishTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		publishParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		preSteps = append(preSteps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, publishTraceID, publishParentSpanID)...)
	}
	preSteps = buildCheckoutRepository(preSteps, c, "", "")
	preSteps = append(preSteps, c.generateGitConfigurationSteps()...)
	preSteps = c.appendUploadAssetsDownloadSteps(preSteps, data)
	return preSteps
}

func (c *Compiler) appendUploadAssetsDownloadSteps(preSteps []string, data *WorkflowData) []string {
	assetsArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	preSteps = append(preSteps, "      - name: Download assets\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/download-artifact")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, fmt.Sprintf("          name: %ssafe-outputs-assets\n", assetsArtifactPrefix))
	preSteps = append(preSteps, fmt.Sprintf("          path: %s\n", constants.TmpGhAwAssetsDirSlash))
	preSteps = append(preSteps, "      - name: List downloaded asset files\n")
	preSteps = append(preSteps, "        continue-on-error: true\n")
	preSteps = append(preSteps, "        run: |\n")
	preSteps = append(preSteps, "          echo \"Downloaded asset files:\"\n")
	return append(preSteps, fmt.Sprintf("          find %s -maxdepth 1 -ls\n", constants.TmpGhAwAssetsDirSlash))
}

func (c *Compiler) buildUploadAssetsEnvVars(data *WorkflowData) []string {
	customEnvVars := []string{
		fmt.Sprintf("          GH_AW_ASSETS_DIR: %q\n", constants.TmpGhAwAssetsDir),
		fmt.Sprintf("          GH_AW_ASSETS_BRANCH: %q\n", data.SafeOutputs.UploadAssets.BranchName),
		fmt.Sprintf("          GH_AW_ASSETS_MAX_SIZE_KB: %d\n", data.SafeOutputs.UploadAssets.MaxSizeKB),
		fmt.Sprintf("          GH_AW_ASSETS_ALLOWED_EXTS: %q\n", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ",")),
	}
	return append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)
}

func uploadAssetsJobCondition(data *WorkflowData) ConditionNode {
	jobCondition := BuildSafeOutputType("upload_asset")
	if IsConditionalDetection(data.SafeOutputs) {
		jobCondition = BuildAnd(
			BuildAnd(BuildFunctionCall("always"), BuildSafeOutputType("upload_asset")),
			buildDetectionPassedCondition(),
		)
	}
	return jobCondition
}

func (c *Compiler) buildUploadAssetsPostSteps() []string {
	if !c.actionMode.IsDev() {
		return nil
	}
	publishAssetsLog.Print("Added restore actions folder step to upload_assets job (dev mode)")
	return []string{c.generateRestoreActionsSetupStep()}
}

// generateSafeOutputsAssetsArtifactUpload generates a step to upload safe-outputs assets as a separate artifact.
// This artifact is then downloaded by the upload_assets job to publish files to orphaned branches.
// In workflow_call context, the artifact name is prefixed to avoid clashes.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func generateSafeOutputsAssetsArtifactUpload(builder *strings.Builder, data *WorkflowData, pinAction func(string) string) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return
	}

	publishAssetsLog.Print("Generating safe-outputs assets artifact upload step")

	// In workflow_call context, apply the per-invocation prefix to avoid artifact name clashes.
	prefix := artifactPrefixExprForDownstreamJob(data)

	builder.WriteString("      # Upload safe-outputs assets for upload_assets job\n")
	builder.WriteString("      - name: Upload Safe Outputs Assets\n")
	builder.WriteString("        if: always()\n")
	fmt.Fprintf(builder, "        uses: %s\n", pinAction("actions/upload-artifact"))
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          name: %ssafe-outputs-assets\n", prefix)
	builder.WriteString("          path: ${{ runner.temp }}/gh-aw/safeoutputs/assets/\n")
	builder.WriteString("          retention-days: 1\n")
	builder.WriteString("          if-no-files-found: ignore\n")
}
