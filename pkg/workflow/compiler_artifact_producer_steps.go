package workflow

import (
	"path"
	"strings"

	"github.com/github/gh-aw/actions/setup"
)

const artifactProducerDirectory = "${{ runner.temp }}/gh-aw/artifact-producer/"

func inlineArtifactScript(script string) string {
	var result strings.Builder
	for line := range strings.SplitSeq(script, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			result.WriteString("          " + line + "\n")
		}
	}
	return result.String()
}

func artifactMetadataName(producer string) string {
	return "${{ steps.artifact-prefix.outputs.identity_prefix }}producer-" + producer
}

func artifactMetadataDirectory(producer string) string {
	return artifactProducerDirectory + producer
}

func (c *Compiler) artifactProducerPublishSteps(producer string) []string {
	steps := []string{
		"      - name: Record artifact producer attempt\n",
		"        shell: bash\n",
		"        env:\n",
		"          INVOCATION_ID: ${{ steps.artifact-prefix.outputs.identity_prefix }}\n",
		"          PRODUCER_JOB: " + producer + "\n",
		"          PRODUCER_ATTEMPT: ${{ github.run_attempt }}\n",
		"          METADATA_DIRECTORY: " + artifactMetadataDirectory(producer) + "\n",
		"        run: |\n",
		inlineArtifactScript(setup.ArtifactProducerScript),
		"      - name: Publish artifact producer attempt\n",
		"        uses: " + c.getActionPin("actions/upload-artifact") + "\n",
		"        with:\n",
		"          name: " + artifactMetadataName(producer) + "\n",
		"          path: " + path.Join(artifactMetadataDirectory(producer), "producer.json") + "\n",
		"          retention-days: 0\n",
		"          if-no-files-found: error\n",
	}
	if downloadArtifactSupportsPattern(c.getActionPin("actions/upload-artifact")) {
		steps = append(steps, "          overwrite: true\n")
	}
	return steps
}

func (c *Compiler) artifactProducerResolveSteps(producer string, required bool) []string {
	softFailure := ""
	if !required {
		softFailure = "        continue-on-error: true\n"
	}
	return []string{
		"      - name: Download " + producer + " producer metadata\n",
		"        id: artifact-metadata-" + producer + "\n",
		"        if: always()\n",
		softFailure,
		"        uses: " + c.getActionPin("actions/download-artifact") + "\n",
		"        with:\n",
		"          name: " + artifactMetadataName(producer) + "\n",
		"          path: " + artifactMetadataDirectory(producer) + "\n",
		"      - name: Resolve " + producer + " artifact prefix\n",
		"        id: artifact-source-" + producer + "\n",
		"        if: always() && steps.artifact-metadata-" + producer + ".outcome == 'success'\n",
		softFailure,
		"        shell: bash\n",
		"        env:\n",
		"          INPUTS_JSON: ${{ toJSON(inputs) }}\n",
		"          INVOCATION_ID: ${{ steps.artifact-prefix.outputs.identity_prefix }}\n",
		"          PRODUCER_JOB: " + producer + "\n",
		"          PRODUCER_RESULT: ${{ needs." + producer + ".result }}\n",
		"          METADATA_DIRECTORY: " + artifactMetadataDirectory(producer) + "\n",
		"        run: |\n",
		inlineArtifactScript(setup.ArtifactProducerResolverScript),
		inlineArtifactScript(setup.ArtifactPrefixScript),
	}
}
