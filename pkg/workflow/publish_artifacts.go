package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
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
		if v, ok := typeutil.ParseIntValue(maxUploads); ok && v > 0 {
			config.MaxUploads = v
		}
	}

	// Parse default-retention-days.
	if retDays, exists := configMap["default-retention-days"]; exists {
		if v, ok := typeutil.ParseIntValue(retDays); ok && v > 0 {
			config.DefaultRetentionDays = v
		}
	}

	// Parse max-retention-days.
	if maxRetDays, exists := configMap["max-retention-days"]; exists {
		if v, ok := typeutil.ParseIntValue(maxRetDays); ok && v > 0 {
			config.MaxRetentionDays = v
		}
	}

	// Parse max-size-bytes.
	if maxBytes, exists := configMap["max-size-bytes"]; exists {
		if v, ok := typeutil.ParseIntValue(maxBytes); ok && v > 0 {
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

// generateSafeOutputsArtifactStagingUpload generates a step in the main agent job that uploads
// the artifact staging directory so the safe_outputs job can download it for inline processing.
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
