package workflow

import "fmt"

// ArtifactDownloadConfig holds configuration for building artifact download steps
type ArtifactDownloadConfig struct {
	ArtifactName string // Name of the artifact to download (e.g., "agent_output.json", "prompt.txt")
	DownloadPath string // Path where artifact will be downloaded (e.g., "/tmp/gh-aw/safe-outputs/")
	SetupEnvStep bool   // Whether to add environment variable setup step
	EnvVarName   string // Environment variable name to set (e.g., "GH_AW_AGENT_OUTPUT")
	StepName     string // Optional custom step name (defaults to "Download {artifact} artifact")
}

// buildArtifactDownloadSteps creates steps to download a GitHub Actions artifact
// This is a generalized helper that can be used across different contexts (safe-outputs, safe-jobs, threat-detection)
func buildArtifactDownloadSteps(config ArtifactDownloadConfig) []string {
	var steps []string

	// Use provided step name or generate default
	stepName := config.StepName
	if stepName == "" {
		stepName = fmt.Sprintf("Download %s artifact", config.ArtifactName)
	}

	// Add step to download artifact
	steps = append(steps, fmt.Sprintf("      - name: %s\n", stepName))
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", config.ArtifactName))
	steps = append(steps, fmt.Sprintf("          path: %s\n", config.DownloadPath))

	// Add environment variable setup if requested
	if config.SetupEnvStep {
		steps = append(steps, "      - name: Setup agent output environment variable\n")
		steps = append(steps, "        run: |\n")
		steps = append(steps, fmt.Sprintf("          mkdir -p %s\n", config.DownloadPath))
		steps = append(steps, fmt.Sprintf("          find %s -type f -print\n", config.DownloadPath))
		// Configure environment variable to point to downloaded artifact file
		artifactPath := fmt.Sprintf("%s%s", config.DownloadPath, config.ArtifactName)
		steps = append(steps, fmt.Sprintf("          echo \"%s=%s\" >> $GITHUB_ENV\n", config.EnvVarName, artifactPath))
	}

	return steps
}
