//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGHAWManifest(t *testing.T) {
	tests := []struct {
		name                string
		secretNames         []string
		actionRefs          []string
		containers          []GHAWManifestContainer
		wantVersion         int
		wantSecrets         []string
		wantActionRepos     []string
		wantContainerImages []string
	}{
		{
			name:        "empty inputs",
			secretNames: nil,
			actionRefs:  nil,
			wantVersion: 1,
			wantSecrets: []string{},
		},
		{
			name:        "secrets prefix stripped to plain name",
			secretNames: []string{"GITHUB_TOKEN", "MY_SECRET"},
			wantVersion: 1,
			wantSecrets: []string{"GITHUB_TOKEN", "MY_SECRET"},
		},
		{
			name:        "secrets.NAME prefix is stripped on input",
			secretNames: []string{"secrets.GITHUB_TOKEN", "GITHUB_TOKEN"},
			wantVersion: 1,
			wantSecrets: []string{"GITHUB_TOKEN"},
		},
		{
			name:        "secrets are sorted and deduplicated",
			secretNames: []string{"Z_SECRET", "A_SECRET", "Z_SECRET"},
			wantVersion: 1,
			wantSecrets: []string{"A_SECRET", "Z_SECRET"},
		},
		{
			name: "action refs with SHA and comment",
			actionRefs: []string{
				"actions/checkout@abc1234def5678 # v4",
				"docker://alpine:3.14", // no @ separator; skipped
			},
			wantVersion:     1,
			wantSecrets:     []string{},
			wantActionRepos: []string{"actions/checkout"},
		},
		{
			name: "action refs without comment use sha as version",
			actionRefs: []string{
				"actions/checkout@v4",
			},
			wantVersion:     1,
			wantSecrets:     []string{},
			wantActionRepos: []string{"actions/checkout"},
		},
		{
			name: "duplicate action refs are deduplicated",
			actionRefs: []string{
				"actions/checkout@abc123 # v4",
				"actions/checkout@abc123 # v4",
			},
			wantVersion:     1,
			wantSecrets:     []string{},
			wantActionRepos: []string{"actions/checkout"},
		},
		{
			name: "containers are sorted and deduplicated",
			containers: []GHAWManifestContainer{
				{Image: "node:lts-alpine"},
				{Image: "alpine:3.14"},
				{Image: "node:lts-alpine"}, // duplicate
			},
			wantVersion:         1,
			wantSecrets:         []string{},
			wantContainerImages: []string{"alpine:3.14", "node:lts-alpine"},
		},
		{
			name: "container with digest retained",
			containers: []GHAWManifestContainer{
				{
					Image:       "node:lts-alpine",
					Digest:      "sha256:abc123",
					PinnedImage: "node:lts-alpine@sha256:abc123",
				},
			},
			wantVersion:         1,
			wantSecrets:         []string{},
			wantContainerImages: []string{"node:lts-alpine"},
		},
		{
			name:                "nil containers produces empty containers field",
			containers:          nil,
			wantVersion:         1,
			wantSecrets:         []string{},
			wantContainerImages: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewGHAWManifest(tt.secretNames, tt.actionRefs, tt.containers)
			require.NotNil(t, m, "manifest should not be nil")
			assert.Equal(t, tt.wantVersion, m.Version, "manifest version")
			if tt.wantSecrets != nil {
				assert.Equal(t, tt.wantSecrets, m.Secrets, "manifest secrets")
			}
			if tt.wantActionRepos != nil {
				repos := make([]string, len(m.Actions))
				for i, a := range m.Actions {
					repos[i] = a.Repo
				}
				assert.Equal(t, tt.wantActionRepos, repos, "action repos")
			}
			if tt.wantContainerImages != nil {
				images := make([]string, len(m.Containers))
				for i, c := range m.Containers {
					images[i] = c.Image
				}
				assert.Equal(t, tt.wantContainerImages, images, "container images")
			}
		})
	}
}

func TestNewGHAWManifestContainerDigest(t *testing.T) {
	containers := []GHAWManifestContainer{
		{
			Image:       "node:lts-alpine",
			Digest:      "sha256:abc123",
			PinnedImage: "node:lts-alpine@sha256:abc123",
		},
		{
			Image: "alpine:3.14", // no digest
		},
	}
	m := NewGHAWManifest(nil, nil, containers)
	require.Len(t, m.Containers, 2, "should have two containers")

	// Sorted: alpine before node
	assert.Equal(t, "alpine:3.14", m.Containers[0].Image, "first container image")
	assert.Empty(t, m.Containers[0].Digest, "alpine digest should be empty")
	assert.Empty(t, m.Containers[0].PinnedImage, "alpine pinned_image should be empty")

	assert.Equal(t, "node:lts-alpine", m.Containers[1].Image, "second container image")
	assert.Equal(t, "sha256:abc123", m.Containers[1].Digest, "node digest")
	assert.Equal(t, "node:lts-alpine@sha256:abc123", m.Containers[1].PinnedImage, "node pinned_image")

	// JSON serialization: digest fields present only when non-empty (omitempty)
	jsonStr, err := m.ToJSON()
	require.NoError(t, err, "ToJSON should not fail")
	assert.Contains(t, jsonStr, `"containers"`, "containers key in JSON")
	assert.Contains(t, jsonStr, `"node:lts-alpine"`, "node image in JSON")
	assert.Contains(t, jsonStr, `"sha256:abc123"`, "node digest in JSON")
	assert.Contains(t, jsonStr, `"node:lts-alpine@sha256:abc123"`, "pinned_image in JSON")
	// alpine has no digest/pinned_image — omitempty must suppress them
	assert.NotContains(t, jsonStr, `"digest":""`, "empty digest must be omitted")
	assert.NotContains(t, jsonStr, `"pinned_image":""`, "empty pinned_image must be omitted")
}

func TestGHAWManifestToJSON(t *testing.T) {
	m := &GHAWManifest{
		Version: 1,
		Secrets: []string{"GITHUB_TOKEN", "MY_SECRET"},
		Actions: []GHAWManifestAction{
			{Repo: "actions/checkout", SHA: "abc123", Version: "v4"},
		},
	}

	json, err := m.ToJSON()
	require.NoError(t, err, "ToJSON should not fail")
	assert.Contains(t, json, `"version":1`, "version in JSON")
	assert.Contains(t, json, `"GITHUB_TOKEN"`, "GITHUB_TOKEN in JSON")
	assert.Contains(t, json, `"MY_SECRET"`, "MY_SECRET in JSON")
	assert.Contains(t, json, `"actions/checkout"`, "action repo in JSON")
	assert.Contains(t, json, `"abc123"`, "action SHA in JSON")
	assert.Contains(t, json, `"v4"`, "action version in JSON")
}

func TestExtractGHAWManifestFromLockFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantNil     bool
		wantErr     bool
		wantVersion int
		wantSecrets []string
	}{
		{
			name:    "no manifest line returns nil",
			content: "# gh-aw-metadata: {}\nsome: yaml",
			wantNil: true,
		},
		{
			name:        "manifest extracted successfully",
			content:     `# gh-aw-manifest: {"version":1,"secrets":["GITHUB_TOKEN"],"actions":[]}`,
			wantVersion: 1,
			wantSecrets: []string{"GITHUB_TOKEN"},
		},
		{
			name:        "manifest with leading spaces in comment",
			content:     `#  gh-aw-manifest: {"version":1,"secrets":[],"actions":[]}`,
			wantVersion: 1,
			wantSecrets: []string{},
		},
		{
			name:    "invalid JSON returns error",
			content: "# gh-aw-manifest: {invalid json}",
			wantErr: true,
		},
		{
			name: "manifest embedded in multi-line header",
			content: `# gh-aw-metadata: {"schema_version":"v3","frontmatter_hash":"abc"}
# gh-aw-manifest: {"version":1,"secrets":["FOO"],"actions":[]}
name: my-workflow`,
			wantVersion: 1,
			wantSecrets: []string{"FOO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ExtractGHAWManifestFromLockFile(tt.content)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}
			require.NoError(t, err, "unexpected error")
			if tt.wantNil {
				assert.Nil(t, m, "expected nil manifest")
				return
			}
			require.NotNil(t, m, "manifest should not be nil")
			assert.Equal(t, tt.wantVersion, m.Version, "manifest version")
			assert.Equal(t, tt.wantSecrets, m.Secrets, "manifest secrets")
		})
	}
}

func TestNormalizeSecretName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GITHUB_TOKEN", "GITHUB_TOKEN"},
		{"secrets.GITHUB_TOKEN", "GITHUB_TOKEN"},
		{"MY_SECRET", "MY_SECRET"},
		{"secrets.MY_SECRET", "MY_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeSecretName(tt.input), "normalized secret name")
		})
	}
}

func TestParseActionRefs(t *testing.T) {
	tests := []struct {
		name    string
		refs    []string
		wantLen int
		check   func(t *testing.T, actions []GHAWManifestAction)
	}{
		{
			name:    "empty refs",
			refs:    nil,
			wantLen: 0,
		},
		{
			name: "ref with SHA and version comment",
			refs: []string{"actions/checkout@abc1234 # v4"},
			check: func(t *testing.T, actions []GHAWManifestAction) {
				require.Len(t, actions, 1, "expected 1 action")
				assert.Equal(t, "actions/checkout", actions[0].Repo, "repo")
				assert.Equal(t, "abc1234", actions[0].SHA, "sha")
				assert.Equal(t, "v4", actions[0].Version, "version")
			},
		},
		{
			name: "ref without comment uses sha as version",
			refs: []string{"actions/checkout@v4"},
			check: func(t *testing.T, actions []GHAWManifestAction) {
				require.Len(t, actions, 1, "expected 1 action")
				assert.Equal(t, "actions/checkout", actions[0].Repo, "repo")
				assert.Equal(t, "v4", actions[0].SHA, "sha")
				assert.Equal(t, "v4", actions[0].Version, "version (same as sha when no comment)")
			},
		},
		{
			name: "ref without @ is skipped",
			refs: []string{"actions/checkout"},
			check: func(t *testing.T, actions []GHAWManifestAction) {
				assert.Empty(t, actions, "action without @ should be skipped")
			},
		},
		{
			name: "duplicate refs deduplicated",
			refs: []string{
				"actions/checkout@abc123 # v4",
				"actions/checkout@abc123 # v4",
			},
			check: func(t *testing.T, actions []GHAWManifestAction) {
				assert.Len(t, actions, 1, "duplicates should be removed")
			},
		},
		{
			name: "actions sorted by repo then sha",
			refs: []string{
				"z-org/z-action@sha2",
				"a-org/a-action@sha1",
			},
			check: func(t *testing.T, actions []GHAWManifestAction) {
				require.Len(t, actions, 2, "expected 2 actions")
				assert.Equal(t, "a-org/a-action", actions[0].Repo, "first action repo")
				assert.Equal(t, "z-org/z-action", actions[1].Repo, "second action repo")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := parseActionRefs(tt.refs)
			if tt.wantLen > 0 {
				assert.Len(t, actions, tt.wantLen, "action count")
			}
			if tt.check != nil {
				tt.check(t, actions)
			}
		})
	}
}
