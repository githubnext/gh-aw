package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var publishArtifactsLog = logger.New("workflow:publish_artifacts")

// defaultArtifactMaxUploads is the default maximum number of upload_artifact tool calls allowed per run.
const defaultArtifactMaxUploads = 1

// defaultArtifactMaxSizeBytes is the default maximum total upload size (100 MB).
const defaultArtifactMaxSizeBytes int64 = 104857600

// artifactStagingDirExpr is the GitHub Actions expression form of the staging directory.
// `actions/upload-artifact` and `actions/download-artifact` do not expand shell variables
// in their `path:` inputs, so we must use ${{ runner.temp }} here.
const artifactStagingDirExpr = "${{ runner.temp }}/gh-aw/safeoutputs/upload-artifacts/"

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
	IfNoFiles string `yaml:"if-no-files,omitempty"` // Behaviour when no files match: "error" or "ignore"
}

// UploadArtifactConfig holds configuration for the upload-artifact safe output type.
type UploadArtifactConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	MaxUploads           int                     `yaml:"max-uploads,omitempty"`    // Max upload_artifact tool calls allowed (default: 1)
	RetentionDays        *string                 `yaml:"retention-days,omitempty"` // Fixed retention period in days (templatable int; agent cannot override)
	SkipArchive          *string                 `yaml:"skip-archive,omitempty"`   // Fixed skip-archive flag (templatable bool; agent cannot override)
	MaxSizeBytes         int64                   `yaml:"max-size-bytes,omitempty"` // Max total bytes per upload (default: 100 MB)
	AllowedPaths         []string                `yaml:"allowed-paths,omitempty"`  // Glob patterns restricting which paths the model may upload
	Filters              *ArtifactFiltersConfig  `yaml:"filters,omitempty"`        // Default include/exclude filters applied on top of allowed-paths
	Defaults             *ArtifactDefaultsConfig `yaml:"defaults,omitempty"`       // Default values injected when the model omits a field
}

// parseUploadArtifactConfig parses the upload-artifact key from the safe-outputs map.
func (c *Compiler) parseUploadArtifactConfig(outputMap map[string]any) *UploadArtifactConfig {
	configData, exists := outputMap["upload-artifact"]
	if !exists {
		return nil
	}
	if b, ok := configData.(bool); ok && !b {
		publishArtifactsLog.Print("upload-artifact explicitly set to false, skipping")
		return nil
	}
	publishArtifactsLog.Print("Parsing upload-artifact configuration")
	config := defaultUploadArtifactConfig()
	configMap, ok := configData.(map[string]any)
	if !ok {
		publishArtifactsLog.Print("upload-artifact enabled with default configuration")
		return config
	}
	applyUploadArtifactConfigMap(c, configMap, config)
	return config
}

func defaultUploadArtifactConfig() *UploadArtifactConfig {
	return &UploadArtifactConfig{
		MaxUploads:   defaultArtifactMaxUploads,
		MaxSizeBytes: defaultArtifactMaxSizeBytes,
	}
}

func applyUploadArtifactConfigMap(c *Compiler, configMap map[string]any, config *UploadArtifactConfig) {
	if maxUploads, exists := configMap["max-uploads"]; exists {
		if v, ok := typeutil.ParseIntValue(maxUploads); ok && v > 0 {
			config.MaxUploads = v
		}
	}
	config.RetentionDays = parseTemplatableIntString(configMap, "retention-days")
	config.SkipArchive = parseTemplatableBoolString(configMap, "skip-archive")
	if maxBytes, exists := configMap["max-size-bytes"]; exists {
		if v, ok := typeutil.ParseIntValue(maxBytes); ok && v > 0 {
			config.MaxSizeBytes = int64(v)
		}
	}
	config.AllowedPaths = parseStringList(configMap["allowed-paths"])
	config.Filters = parseArtifactFilters(configMap["filters"])
	config.Defaults = parseArtifactDefaults(configMap["defaults"])
	c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 0)
	publishArtifactsLog.Printf("Parsed upload-artifact config: max_uploads=%d, retention_days=%v, skip_archive=%v, max_size_bytes=%d",
		config.MaxUploads, config.RetentionDays, config.SkipArchive, config.MaxSizeBytes)
}

func parseTemplatableIntString(configMap map[string]any, field string) *string {
	if err := preprocessIntFieldAsString(configMap, field, publishArtifactsLog); err != nil {
		publishArtifactsLog.Printf("Warning: %v", err)
	}
	if value, exists := configMap[field]; exists {
		if s, ok := value.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

func parseTemplatableBoolString(configMap map[string]any, field string) *string {
	if err := preprocessBoolFieldAsString(configMap, field, publishArtifactsLog); err != nil {
		publishArtifactsLog.Printf("Warning: %v", err)
	}
	if value, exists := configMap[field]; exists {
		if s, ok := value.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

func parseStringList(value any) []string {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	var stringsList []string
	for _, p := range arr {
		if s, ok := p.(string); ok && s != "" {
			stringsList = append(stringsList, s)
		}
	}
	return stringsList
}

func parseArtifactFilters(value any) *ArtifactFiltersConfig {
	filtersMap, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	filters := &ArtifactFiltersConfig{
		Include: parseStringListAllowEmpty(filtersMap["include"]),
		Exclude: parseStringListAllowEmpty(filtersMap["exclude"]),
	}
	if len(filters.Include) > 0 || len(filters.Exclude) > 0 {
		return filters
	}
	return nil
}

func parseStringListAllowEmpty(value any) []string {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	var stringsList []string
	for _, v := range arr {
		if s, ok := v.(string); ok {
			stringsList = append(stringsList, s)
		}
	}
	return stringsList
}

func parseArtifactDefaults(value any) *ArtifactDefaultsConfig {
	defaultsMap, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	defaults := &ArtifactDefaultsConfig{}
	if ifNoFiles, ok := defaultsMap["if-no-files"].(string); ok && ifNoFiles != "" {
		defaults.IfNoFiles = ifNoFiles
	}
	if defaults.IfNoFiles != "" {
		return defaults
	}
	return nil
}

// generateSafeOutputsArtifactStagingUpload generates a step in the main agent job that uploads
// the artifact staging directory so the safe_outputs job can download it for inline processing.
// This step only appears when upload-artifact is configured in safe-outputs.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func generateSafeOutputsArtifactStagingUpload(builder *strings.Builder, data *WorkflowData, pinAction func(string) string) {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadArtifact == nil {
		return
	}

	publishArtifactsLog.Print("Generating safe-outputs artifact staging upload step")

	prefix := artifactPrefixExprForDownstreamJob(data)

	builder.WriteString("      # Upload safe-outputs upload-artifact staging for the upload_artifact job\n")
	builder.WriteString("      - name: Upload upload-artifact staging\n")
	builder.WriteString("        if: always()\n")
	fmt.Fprintf(builder, "        uses: %s\n", pinAction("actions/upload-artifact"))
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          name: %s%s\n", prefix, SafeOutputsUploadArtifactStagingArtifactName)
	fmt.Fprintf(builder, "          path: %s\n", artifactStagingDirExpr)
	builder.WriteString("          retention-days: 1\n")
	builder.WriteString("          if-no-files-found: ignore\n")
}
