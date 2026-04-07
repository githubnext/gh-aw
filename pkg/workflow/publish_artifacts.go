package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var publishArtifactsLog = logger.New("workflow:publish_artifacts")

// defaultArtifactMaxUploads is the default maximum number of upload_artifact tool calls allowed per run.
const defaultArtifactMaxUploads = 1

// defaultArtifactRetentionDays is the default artifact retention period in days.
const defaultArtifactRetentionDays = 7

// defaultArtifactMaxRetentionDays is the default maximum retention cap in days.
const defaultArtifactMaxRetentionDays = 30

// defaultArtifactMaxSizeBytes is the default maximum total upload size (100 MB).
const defaultArtifactMaxSizeBytes int64 = 104857600

// artifactStagingDir is the path where the model stages files to be uploaded as artifacts.
// Use the shell-variable form only inside `run:` blocks; for `with: path:` inputs use
// artifactStagingDirExpr which uses the GitHub Actions expression syntax.
const artifactStagingDir = "${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/"

// artifactStagingDirExpr is the GitHub Actions expression form of artifactStagingDir.
// `actions/upload-artifact` and `actions/download-artifact` do not expand shell variables
// in their `path:` inputs, so we must use ${{ runner.temp }} here.
const artifactStagingDirExpr = "${{ runner.temp }}/gh-aw/safeoutputs/upload-artifacts/"

// artifactSlotDir is the per-slot directory used by the handler to organise staged files.
// Use the shell-variable form only inside `run:` blocks; for `with: path:` inputs use
// artifactSlotDirExpr which uses the GitHub Actions expression syntax.
const artifactSlotDir = "${RUNNER_TEMP}/gh-aw/upload-artifacts/"

// artifactSlotDirExpr is the GitHub Actions expression form of artifactSlotDir.
const artifactSlotDirExpr = "${{ runner.temp }}/gh-aw/upload-artifacts/"

// SafeOutputsUploadArtifactStagingArtifactName is the artifact that carries the staging directory
// from the main agent job to the upload_artifact job.
const SafeOutputsUploadArtifactStagingArtifactName = "safe-outputs-upload-artifacts"

// ArtifactFiltersConfig holds include/exclude glob patterns for artifact file selection.
type ArtifactFiltersConfig struct {
	Include []string `yaml:"include,omitempty"` // Glob patterns for files to include
	Exclude []string `yaml:"exclude,omitempty"` // Glob patterns for files to exclude
}

// ArtifactDefaultsConfig holds default request settings applied when the model does not
// specify a value explicitly.
type ArtifactDefaultsConfig struct {
	SkipArchive bool   `yaml:"skip-archive,omitempty"` // Default value for skip_archive
	IfNoFiles   string `yaml:"if-no-files,omitempty"`  // Behaviour when no files match: "error" or "ignore"
}

// ArtifactAllowConfig holds policy settings for optional behaviours that must be explicitly
// opted-in to by the workflow author.
type ArtifactAllowConfig struct {
	SkipArchive bool `yaml:"skip-archive,omitempty"` // Allow skip_archive: true in model requests
}

// UploadArtifactConfig holds configuration for the upload-artifact safe output type.
type UploadArtifactConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	MaxUploads           int                     `yaml:"max-uploads,omitempty"`            // Max upload_artifact tool calls allowed (default: 1)
	DefaultRetentionDays int                     `yaml:"default-retention-days,omitempty"` // Default retention period (default: 7 days)
	MaxRetentionDays     int                     `yaml:"max-retention-days,omitempty"`     // Maximum retention cap (default: 30 days)
	MaxSizeBytes         int64                   `yaml:"max-size-bytes,omitempty"`         // Max total bytes per upload (default: 100 MB)
	AllowedPaths         []string                `yaml:"allowed-paths,omitempty"`          // Glob patterns restricting which paths the model may upload
	Filters              *ArtifactFiltersConfig  `yaml:"filters,omitempty"`                // Default include/exclude filters applied on top of allowed-paths
	Defaults             *ArtifactDefaultsConfig `yaml:"defaults,omitempty"`               // Default values injected when the model omits a field
	Allow                *ArtifactAllowConfig    `yaml:"allow,omitempty"`                  // Opt-in behaviours
}

// parseUploadArtifactConfig parses the upload-artifact key from the safe-outputs map.
func (c *Compiler) parseUploadArtifactConfig(outputMap map[string]any) *UploadArtifactConfig {
	configData, exists := outputMap["upload-artifact"]
	if !exists {
		return nil
	}

	// Explicit false disables upload-artifact (e.g. when passed via import-inputs).
	if b, ok := configData.(bool); ok && !b {
		publishArtifactsLog.Print("upload-artifact explicitly set to false, skipping")
		return nil
	}

	publishArtifactsLog.Print("Parsing upload-artifact configuration")
	config := &UploadArtifactConfig{
		MaxUploads:           defaultArtifactMaxUploads,
		DefaultRetentionDays: defaultArtifactRetentionDays,
		MaxRetentionDays:     defaultArtifactMaxRetentionDays,
		MaxSizeBytes:         defaultArtifactMaxSizeBytes,
	}

	configMap, ok := configData.(map[string]any)
	if !ok {
		// No config map (e.g. upload-artifact: true) – use defaults.
		publishArtifactsLog.Print("upload-artifact enabled with default configuration")
		return config
	}

	// Parse max-uploads.
	if maxUploads, exists := configMap["max-uploads"]; exists {
		if v, ok := parseIntValue(maxUploads); ok && v > 0 {
			config.MaxUploads = v
		}
	}

	// Parse default-retention-days.
	if retDays, exists := configMap["default-retention-days"]; exists {
		if v, ok := parseIntValue(retDays); ok && v > 0 {
			config.DefaultRetentionDays = v
		}
	}

	// Parse max-retention-days.
	if maxRetDays, exists := configMap["max-retention-days"]; exists {
		if v, ok := parseIntValue(maxRetDays); ok && v > 0 {
			config.MaxRetentionDays = v
		}
	}

	// Parse max-size-bytes.
	if maxBytes, exists := configMap["max-size-bytes"]; exists {
		if v, ok := parseIntValue(maxBytes); ok && v > 0 {
			config.MaxSizeBytes = int64(v)
		}
	}

	// Parse allowed-paths.
	if allowedPaths, exists := configMap["allowed-paths"]; exists {
		if arr, ok := allowedPaths.([]any); ok {
			for _, p := range arr {
				if s, ok := p.(string); ok && s != "" {
					config.AllowedPaths = append(config.AllowedPaths, s)
				}
			}
		}
	}

	// Parse filters.
	if filtersData, exists := configMap["filters"]; exists {
		if filtersMap, ok := filtersData.(map[string]any); ok {
			filters := &ArtifactFiltersConfig{}
			if inc, ok := filtersMap["include"].([]any); ok {
				for _, v := range inc {
					if s, ok := v.(string); ok {
						filters.Include = append(filters.Include, s)
					}
				}
			}
			if exc, ok := filtersMap["exclude"].([]any); ok {
				for _, v := range exc {
					if s, ok := v.(string); ok {
						filters.Exclude = append(filters.Exclude, s)
					}
				}
			}
			if len(filters.Include) > 0 || len(filters.Exclude) > 0 {
				config.Filters = filters
			}
		}
	}

	// Parse defaults.
	if defaultsData, exists := configMap["defaults"]; exists {
		if defaultsMap, ok := defaultsData.(map[string]any); ok {
			defaults := &ArtifactDefaultsConfig{}
			if skipArchive, ok := defaultsMap["skip-archive"].(bool); ok {
				defaults.SkipArchive = skipArchive
			}
			if ifNoFiles, ok := defaultsMap["if-no-files"].(string); ok && ifNoFiles != "" {
				defaults.IfNoFiles = ifNoFiles
			}
			config.Defaults = defaults
		}
	}

	// Parse allow.
	if allowData, exists := configMap["allow"]; exists {
		if allowMap, ok := allowData.(map[string]any); ok {
			allow := &ArtifactAllowConfig{}
			if skipArchive, ok := allowMap["skip-archive"].(bool); ok {
				allow.SkipArchive = skipArchive
			}
			config.Allow = allow
		}
	}

	// Parse common base fields (max, github-token, staged).
	c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 0)

	publishArtifactsLog.Printf("Parsed upload-artifact config: max_uploads=%d, default_retention=%d, max_retention=%d, max_size_bytes=%d",
		config.MaxUploads, config.DefaultRetentionDays, config.MaxRetentionDays, config.MaxSizeBytes)
	return config
}

// buildUploadArtifactJob creates the upload_artifact standalone job.
//
// Architecture:
//  1. The model stages files to artifactStagingDir during its run.
//  2. The main agent job uploads that directory as a GitHub Actions staging artifact.
//  3. This job downloads the staging artifact, validates each upload_artifact request,
//     copies approved files into per-slot directories, and then uploads each slot using
//     actions/upload-artifact with a conditional step per MaxUploads slot.
//  4. A temporary artifact ID is returned for each slot via job outputs.
func (c *Compiler) buildUploadArtifactJob(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) (*Job, error) {
	publishArtifactsLog.Printf("Building upload_artifact job: workflow=%s, main_job=%s, threat_detection=%v",
		data.Name, mainJobName, threatDetectionEnabled)

	if data.SafeOutputs == nil || data.SafeOutputs.UploadArtifact == nil {
		return nil, errors.New("safe-outputs.upload-artifact configuration is required")
	}

	cfg := data.SafeOutputs.UploadArtifact

	var preSteps []string

	// Add setup step so scripts are available at SetupActionDestination.
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		preSteps = append(preSteps, c.generateCheckoutActionsFolder(data)...)
		publishTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		preSteps = append(preSteps, c.generateSetupStep(setupActionRef, SetupActionDestination, false, publishTraceID)...)
	}

	// Download the staging artifact that contains the files staged by the model.
	// The agent output artifact (carrying upload_artifact NDJSON records) is NOT added here
	// because buildCustomActionStep / buildGitHubScriptStep already prepends that step
	// automatically to every safe-output job.
	artifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	stagingArtifactName := artifactPrefix + SafeOutputsUploadArtifactStagingArtifactName
	preSteps = append(preSteps,
		"      - name: Download upload-artifact staging\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", stagingArtifactName),
		fmt.Sprintf("          path: %s\n", artifactStagingDirExpr),
	)

	// Build custom environment variables consumed by upload_artifact.cjs.
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_MAX_UPLOADS: %d\n", cfg.MaxUploads))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_DEFAULT_RETENTION_DAYS: %d\n", cfg.DefaultRetentionDays))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_MAX_RETENTION_DAYS: %d\n", cfg.MaxRetentionDays))
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_MAX_SIZE_BYTES: %d\n", cfg.MaxSizeBytes))

	if len(cfg.AllowedPaths) > 0 {
		allowedPathsJSON := marshalStringSliceJSON(cfg.AllowedPaths)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_ALLOWED_PATHS: %q\n", allowedPathsJSON))
	}

	if cfg.Allow != nil && cfg.Allow.SkipArchive {
		customEnvVars = append(customEnvVars, "          GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE: \"true\"\n")
	}
	if cfg.Defaults != nil {
		if cfg.Defaults.SkipArchive {
			customEnvVars = append(customEnvVars, "          GH_AW_ARTIFACT_DEFAULT_SKIP_ARCHIVE: \"true\"\n")
		}
		if cfg.Defaults.IfNoFiles != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_DEFAULT_IF_NO_FILES: %q\n", cfg.Defaults.IfNoFiles))
		}
	}
	if cfg.Filters != nil {
		if len(cfg.Filters.Include) > 0 {
			filtersIncJSON := marshalStringSliceJSON(cfg.Filters.Include)
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_FILTERS_INCLUDE: %q\n", filtersIncJSON))
		}
		if len(cfg.Filters.Exclude) > 0 {
			filtersExcJSON := marshalStringSliceJSON(cfg.Filters.Exclude)
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ARTIFACT_FILTERS_EXCLUDE: %q\n", filtersExcJSON))
		}
	}

	// Add standard env vars (run ID, repo, etc.).
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, "")...)

	// Build conditional actions/upload-artifact steps – one per MaxUploads slot.
	// The handler sets slot_N_enabled=true and outputs the slot name / retention when
	// the Nth upload_artifact request was successfully validated and staged.
	var postSteps []string
	for i := range cfg.MaxUploads {
		slotDir := fmt.Sprintf("%sslot_%d/", artifactSlotDirExpr, i)
		postSteps = append(postSteps,
			fmt.Sprintf("      - name: Upload artifact slot %d\n", i),
			fmt.Sprintf("        if: steps.upload_artifacts.outputs.slot_%d_enabled == 'true'\n", i),
			fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")),
			"        with:\n",
			fmt.Sprintf("          name: ${{ steps.upload_artifacts.outputs.slot_%d_name }}\n", i),
			fmt.Sprintf("          path: %s\n", slotDir),
			fmt.Sprintf("          retention-days: ${{ steps.upload_artifacts.outputs.slot_%d_retention_days }}\n", i),
			"          if-no-files-found: ignore\n",
		)
	}

	// In dev mode, restore the actions/setup folder so the post-step cleanup succeeds.
	if c.actionMode.IsDev() {
		postSteps = append(postSteps, c.generateRestoreActionsSetupStep())
		publishArtifactsLog.Print("Added restore actions folder step to upload_artifact job (dev mode)")
	}

	jobCondition := BuildSafeOutputType("upload_artifact")
	needs := []string{mainJobName, string(constants.ActivationJobName)}

	// Collect job outputs for all slots so downstream jobs can reference them.
	outputs := map[string]string{
		"artifact_count": "${{ steps.upload_artifacts.outputs.artifact_count }}",
	}
	for i := range cfg.MaxUploads {
		outputs[fmt.Sprintf("slot_%d_tmp_id", i)] = fmt.Sprintf("${{ steps.upload_artifacts.outputs.slot_%d_tmp_id }}", i)
	}

	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "upload_artifact",
		StepName:      "Upload artifacts",
		StepID:        "upload_artifacts",
		ScriptName:    "upload_artifact",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        "",
		Permissions:   NewPermissions(),
		Outputs:       outputs,
		Condition:     jobCondition,
		PreSteps:      preSteps,
		PostSteps:     postSteps,
		Token:         cfg.GitHubToken,
		Needs:         needs,
	})
}

// generateSafeOutputsArtifactStagingUpload generates a step in the main agent job that uploads
// the artifact staging directory so the upload_artifact job can download it.
// This step only appears when upload-artifact is configured in safe-outputs.
func generateSafeOutputsArtifactStagingUpload(builder *strings.Builder, data *WorkflowData) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadArtifact == nil {
		return
	}

	publishArtifactsLog.Print("Generating safe-outputs artifact staging upload step")

	prefix := artifactPrefixExprForDownstreamJob(data)

	builder.WriteString("      # Upload safe-outputs upload-artifact staging for the upload_artifact job\n")
	builder.WriteString("      - name: Upload Upload-Artifact Staging\n")
	builder.WriteString("        if: always()\n")
	fmt.Fprintf(builder, "        uses: %s\n", GetActionPin("actions/upload-artifact"))
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          name: %s%s\n", prefix, SafeOutputsUploadArtifactStagingArtifactName)
	fmt.Fprintf(builder, "          path: %s\n", artifactStagingDirExpr)
	builder.WriteString("          retention-days: 1\n")
	builder.WriteString("          if-no-files-found: ignore\n")
}

// marshalStringSliceJSON serialises a []string to a compact JSON array string.
// This is used to pass multi-value config fields as environment variables.
func marshalStringSliceJSON(values []string) string {
	data, err := json.Marshal(values)
	if err != nil {
		// Should never happen for plain string slices.
		return "[]"
	}
	return string(data)
}
