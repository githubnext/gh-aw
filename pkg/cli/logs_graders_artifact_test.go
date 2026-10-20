//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsDownloadsSeparateGraderJobArtifact(t *testing.T) {
	for _, prefix := range []string{"", "a1b2c3-"} {
		t.Run(prefix, func(t *testing.T) {
			calls := installSessionTestGH(t, map[string]map[string]string{
				prefix + "graders": customGraderArtifactFixture(t),
			})
			dir := t.TempDir()
			registerCustomGraderFixture(t, dir, prefix)
			sets := applyGradersArtifact([]string{"usage"}, true)
			filter := ResolveArtifactFilter(sets)
			assert.Contains(t, filter, constants.GradersArtifactName.String())
			assert.Contains(t, findMissingFilterEntries(filter, dir), constants.GradersArtifactName.String(),
				"cached usage/graders is not the separate graders artifact")
			require.NoError(t, downloadRunArtifacts(context.Background(), downloadArtifactsOptions{
				runID: 123, outputDir: dir, owner: "owner", repo: "repo", artifactFilter: filter,
			}))
			results := extractGradersData(dir)
			require.NotNil(t, results)
			require.Len(t, results.Results, 2)
			assert.Equal(t, "custom-count", results.Results[0].ID)
			assert.InDelta(t, 3, *results.Results[0].Value, 1e-9)
			assert.FileExists(t, filepath.Join(dir, prefix+"graders", constants.GraderResultsFilename.String()))
			content, err := os.ReadFile(calls)
			require.NoError(t, err)
			assert.Contains(t, string(content), "--name "+prefix+"graders")
		})
	}
}
