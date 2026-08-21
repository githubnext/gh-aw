package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func artifactTestJob(name string, needs []string, steps string) *Job {
	return &Job{Name: name, Needs: needs, Steps: []string{steps}}
}

func TestArtifactManagerAnalyzeJobsAndInventory(t *testing.T) {
	manager := NewArtifactManager()
	jobs := map[string]*Job{
		"producer": artifactTestJob("producer", nil, `      - name: Upload reports
        uses: actions/upload-artifact@sha
        with:
          name: reports
          path: |
            /tmp/report.json
            /tmp/report.txt
      - name: Upload logs
        uses: actions/upload-artifact@sha
        with:
          name: logs
          path: /tmp/logs
`),
		"consumer": artifactTestJob("consumer", []string{"producer"}, `      - name: Download outputs
        uses: actions/download-artifact@sha
        with:
          pattern: "{reports,logs}"
          path: /tmp/downloads
          merge-multiple: true
`),
	}

	require.NoError(t, manager.AnalyzeJobs(jobs))

	assert.Equal(t, []ArtifactInventoryEntry{
		{
			Name:         "logs",
			Created:      true,
			Downloaded:   true,
			CreatedIn:    []string{"producer"},
			DownloadedIn: []string{"consumer"},
		},
		{
			Name:         "reports",
			Created:      true,
			Downloaded:   true,
			CreatedIn:    []string{"producer"},
			DownloadedIn: []string{"consumer"},
		},
	}, manager.Inventory())
	assert.Equal(t, []string{"/tmp/report.json", "/tmp/report.txt"}, manager.uploads["producer"][0].Paths)
	assert.True(t, manager.downloads["consumer"][0].MergeMultiple)
}

func TestArtifactManagerTracksDownloadOnlyArtifact(t *testing.T) {
	manager := NewArtifactManager()
	jobs := map[string]*Job{
		"consumer": artifactTestJob("consumer", nil, `      - uses: actions/download-artifact@sha
        with:
          name: external-result
`),
	}

	require.NoError(t, manager.AnalyzeJobs(jobs))
	assert.Equal(t, []ArtifactInventoryEntry{{
		Name:         "external-result",
		Downloaded:   true,
		DownloadedIn: []string{"consumer"},
	}}, manager.Inventory())
}

func TestArtifactManagerIgnoresUnrelatedYAMLLikeValues(t *testing.T) {
	manager := NewArtifactManager()
	jobs := map[string]*Job{
		"producer": artifactTestJob("producer", nil, `      - name: Configure
        env:
          WORKFLOW_REF: owner/repo/path with colon: value
        run: echo "$WORKFLOW_REF"
      - uses: actions/upload-artifact@sha
        with:
          name: result
          path: result.json
`),
	}

	require.NoError(t, manager.AnalyzeJobs(jobs))
	assert.Equal(t, "result", manager.Inventory()[0].Name)
}

func TestArtifactManagerRejectsUploadNameClashes(t *testing.T) {
	manager := NewArtifactManager()
	jobs := map[string]*Job{
		"first": artifactTestJob("first", nil, `      - uses: actions/upload-artifact@sha
        with:
          name: shared
          path: first.txt
`),
		"second": artifactTestJob("second", nil, `      - uses: actions/upload-artifact@sha
        with:
          name: shared
          path: second.txt
`),
	}

	err := manager.AnalyzeJobs(jobs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `artifact name clash: "shared"`)
	assert.Contains(t, err.Error(), `"first" and "second"`)
}

func TestArtifactManagerRejectsMalformedUploads(t *testing.T) {
	tests := []struct {
		name      string
		step      string
		wantError string
	}{
		{
			name: "invalid static name",
			step: `      - uses: actions/upload-artifact@sha
        with:
          name: invalid/name
          path: output.txt
`,
			wantError: "has an invalid name",
		},
		{
			name: "missing path",
			step: `      - uses: actions/upload-artifact@sha
        with:
          name: output
`,
			wantError: "must have at least one path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewArtifactManager()
			err := manager.AnalyzeJobs(map[string]*Job{
				"producer": artifactTestJob("producer", nil, tt.step),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestGenerateWorkflowHeaderIncludesArtifactInventory(t *testing.T) {
	compiler := NewCompiler()
	require.NoError(t, compiler.artifactManager.AnalyzeJobs(map[string]*Job{
		"producer": artifactTestJob("producer", nil, `      - uses: actions/upload-artifact@sha
        with:
          name: result
          path: result.json
`),
		"consumer": artifactTestJob("consumer", []string{"producer"}, `      - uses: actions/download-artifact@sha
        with:
          name: result
`),
	}))

	var header strings.Builder
	data := &WorkflowData{RawFrontmatter: map[string]any{}}
	require.NoError(t, compiler.generateWorkflowHeader(&header, data, "hash", "body", nil, nil))

	output := header.String()
	assert.Contains(t, output, `# Artifacts:
#   - name: "result"
#     created: true
#     downloaded: true
#     created-in: producer
#     downloaded-in: consumer`)

	manifest, err := ExtractGHAWManifestFromLockFile(output)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, compiler.artifactManager.Inventory(), manifest.Artifacts)
}
