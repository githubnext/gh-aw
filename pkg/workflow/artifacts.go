package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var artifactsLog = logger.New("workflow:artifacts")

// ArtifactDownloadConfig holds configuration for building artifact download steps
type ArtifactDownloadConfig struct {
	ArtifactName     string // Name of the artifact to download (e.g., "agent-output", "prompt")
	ArtifactFilename string // Filename inside the artifact directory (e.g., "agent_output.json", "prompt.txt")
	DownloadPath     string // Path where artifact will be downloaded (e.g., "/tmp/gh-aw/safeoutputs/")
	SetupEnvStep     bool   // Whether to add environment variable setup step
	EnvVarName       string // Environment variable name to set (e.g., "GH_AW_AGENT_OUTPUT")
	StepName         string // Optional custom step name (defaults to "Download {artifact} artifact")
	IfCondition      string // Optional conditional expression for the step (e.g., "needs.agent.outputs.has_patch == 'true'")
}

// buildArtifactDownloadSteps creates steps to download a GitHub Actions artifact
// This is a generalized helper that can be used across different contexts (safe-outputs, safe-jobs, threat-detection)
func buildArtifactDownloadSteps(config ArtifactDownloadConfig) []string {
	artifactsLog.Printf("Building artifact download steps: artifact=%s, path=%s, setupEnv=%v",
		config.ArtifactName, config.DownloadPath, config.SetupEnvStep)

	var steps []string

	// Use provided step name or generate default
	stepName := config.StepName
	if stepName == "" {
		stepName = fmt.Sprintf("Download %s artifact", config.ArtifactName)
		artifactsLog.Printf("Using default step name: %s", stepName)
	}

	// Add step to download artifact
	steps = append(steps, fmt.Sprintf("      - name: %s\n", stepName))
	// Add conditional if specified
	if config.IfCondition != "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", config.IfCondition))
		artifactsLog.Printf("Added conditional: %s", config.IfCondition)
	}
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/download-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", config.ArtifactName))
	steps = append(steps, fmt.Sprintf("          path: %s\n", config.DownloadPath))

	// Add environment variable setup if requested
	if config.SetupEnvStep {
		artifactsLog.Printf("Adding environment variable setup step: %s=%s%s/%s",
			config.EnvVarName, config.DownloadPath, config.ArtifactName, config.ArtifactFilename)
		steps = append(steps, "      - name: Setup agent output environment variable\n")
		steps = append(steps, "        run: |\n")
		steps = append(steps, fmt.Sprintf("          mkdir -p %s\n", config.DownloadPath))
		steps = append(steps, fmt.Sprintf("          find \"%s\" -type f -print\n", config.DownloadPath))
		// artifacts are extracted to {download-path}/{artifact-name}/
		// The actual filename is specified in ArtifactFilename
		artifactPath := fmt.Sprintf("%s%s/%s", config.DownloadPath, config.ArtifactName, config.ArtifactFilename)
		steps = append(steps, fmt.Sprintf("          echo \"%s=%s\" >> \"$GITHUB_ENV\"\n", config.EnvVarName, artifactPath))
	}

	artifactsLog.Printf("Generated %d artifact download steps", len(steps))
	return steps
}

// ArtifactUploadConfig holds configuration for building artifact upload steps
type ArtifactUploadConfig struct {
	StepName       string   // Human-readable step name (e.g., "Upload Agent Stdio")
	ArtifactName   string   // Name of the artifact in GitHub Actions (e.g., "agent-stdio.log")
	UploadPaths    []string // Paths to upload (e.g., "/tmp/gh-aw/agent-stdio.log")
	IfNoFilesFound string   // What to do if files not found: "warn" or "ignore" (default: "warn")
}

// generateArtifactUpload creates a YAML step to upload a GitHub Actions artifact
// This is a generalized helper that eliminates duplication across different upload functions
func (c *Compiler) generateArtifactUpload(yaml *strings.Builder, config ArtifactUploadConfig) {
	artifactsLog.Printf("Generating artifact upload: step=%s, artifact=%s, paths=%v",
		config.StepName, config.ArtifactName, config.UploadPaths)

	// Record artifact upload for validation
	c.stepOrderTracker.RecordArtifactUpload(config.StepName, config.UploadPaths)

	// Determine if-no-files-found value (default to "warn")
	ifNoFilesFound := config.IfNoFilesFound
	if ifNoFilesFound == "" {
		ifNoFilesFound = "warn"
	}

	// Generate upload step YAML
	fmt.Fprintf(yaml, "      - name: %s\n", config.StepName)
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/upload-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", config.ArtifactName)

	// Write path (only single-path is supported)
	if len(config.UploadPaths) == 0 {
		panic(fmt.Sprintf("generateArtifactUpload: no upload paths specified for artifact %s", config.ArtifactName))
	}
	if len(config.UploadPaths) > 1 {
		panic(fmt.Sprintf("generateArtifactUpload: multiple paths not supported (got %d paths for artifact %s)", len(config.UploadPaths), config.ArtifactName))
	}
	fmt.Fprintf(yaml, "          path: %s\n", config.UploadPaths[0])

	fmt.Fprintf(yaml, "          if-no-files-found: %s\n", ifNoFilesFound)

	artifactsLog.Printf("Generated artifact upload step for %s", config.ArtifactName)
}
