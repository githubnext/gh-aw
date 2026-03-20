package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
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
	StepID           string // Optional step ID; when set, the env-setup step is gated on this step's success
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
	// Add step ID if specified (used to condition the env-setup step on download success)
	if config.StepID != "" {
		steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
		artifactsLog.Printf("Added step ID: %s", config.StepID)
	}
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
		artifactsLog.Printf("Adding environment variable setup step: find %s -name %s",
			config.DownloadPath, config.ArtifactFilename)
		steps = append(steps, "      - name: Setup agent output environment variable\n")
		// Only set the env var when the artifact was actually downloaded
		if config.StepID != "" {
			steps = append(steps, fmt.Sprintf("        if: steps.%s.outcome == 'success'\n", config.StepID))
			artifactsLog.Printf("Added env-setup conditional on step outcome: %s", config.StepID)
		}
		steps = append(steps, "        run: |\n")
		steps = append(steps, fmt.Sprintf("          mkdir -p %s\n", config.DownloadPath))
		steps = append(steps, fmt.Sprintf("          find \"%s\" -type f -print\n", config.DownloadPath))
		// Use find to dynamically locate the file regardless of internal artifact directory
		// structure. download-artifact may preserve the artifact's internal path hierarchy
		// (e.g. /tmp/gh-aw/ becomes {download-path}/gh-aw/ after extraction), so we cannot
		// rely on a hardcoded path.
		steps = append(steps, fmt.Sprintf("          FOUND_FILE=$(find \"%s\" -name \"%s\" -type f 2>/dev/null | head -1)\n", config.DownloadPath, config.ArtifactFilename))
		steps = append(steps, "          if [ -n \"$FOUND_FILE\" ]; then\n")
		steps = append(steps, fmt.Sprintf("            echo \"%s=$FOUND_FILE\" >> \"$GITHUB_ENV\"\n", config.EnvVarName))
		steps = append(steps, "          else\n")
		steps = append(steps, fmt.Sprintf("            echo \"Warning: %s not found under %s\" >&2\n", config.ArtifactFilename, config.DownloadPath))
		steps = append(steps, "          fi\n")
	}

	return steps
}
