package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildArtifactDownloadSteps_OptionalDownload verifies that artifact downloads use a
// pattern-based lookup instead of a name-based one. actions/download-artifact fails with
// "Artifact not found for name: <name>" for name-based downloads, which surfaced in every
// downstream job whenever the agent job died before uploading its artifact and masked the
// real upstream failure. A pattern that matches nothing is a silent no-op instead.
func TestBuildArtifactDownloadSteps_OptionalDownload(t *testing.T) {
	steps := strings.Join(buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: "agent",
		DownloadPath: "/tmp/gh-aw/",
	}, getActionPin), "")

	assert.Contains(t, steps, "          pattern: agent\n", "download should be pattern-based so a missing artifact is not an error")
	assert.Contains(t, steps, "          merge-multiple: true\n", "merge-multiple keeps the extraction layout identical to name-based downloads")
	assert.NotContains(t, steps, "          name: agent\n", "name-based downloads fail hard when the artifact is missing")
	assert.Contains(t, steps, "        continue-on-error: true\n")
}

// TestBuildArtifactDownloadSteps_MissingArtifactWarning verifies that the env-setup step only
// exports the agent output path when the file actually exists, and otherwise emits an explicit
// annotation pointing at the upstream agent job failure.
func TestBuildArtifactDownloadSteps_MissingArtifactWarning(t *testing.T) {
	steps := strings.Join(buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName:     "agent",
		ArtifactFilename: "agent_output.json",
		DownloadPath:     "/tmp/gh-aw/",
		SetupEnvStep:     true,
		EnvVarName:       "GH_AW_AGENT_OUTPUT",
		StepID:           "download-agent-output",
	}, getActionPin), "")

	assert.Contains(t, steps, `if [ -f "/tmp/gh-aw/agent_output.json" ]; then`)
	assert.Contains(t, steps, `echo "GH_AW_AGENT_OUTPUT=/tmp/gh-aw/agent_output.json" >> "$GITHUB_OUTPUT"`)
	assert.Contains(t, steps, "::warning title=Upstream agent job failure::")
	assert.Contains(t, steps, "\n          fi\n", "the existence check must be closed")

	// The warning must not be emitted when the artifact is present.
	warningIdx := strings.Index(steps, "::warning title=Upstream agent job failure::")
	elseIdx := strings.Index(steps, "\n          else\n")
	assert.NotEqual(t, -1, elseIdx, "warning should live in the else branch")
	assert.Less(t, elseIdx, warningIdx, "warning should be emitted only when the artifact is missing")
}
