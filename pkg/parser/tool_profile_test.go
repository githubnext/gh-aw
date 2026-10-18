package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportsWithToolProfile(t *testing.T) {
	profile := BuiltinToolProfilePathPrefix + "test-ecosystem.md"
	RegisterBuiltinVirtualFile(profile, []byte("---\nruntimes:\n  go: {}\n---\n"))
	for _, tc := range []struct {
		name    string
		fm      map[string]any
		paths   []string
		wantErr string
	}{
		{"absent", nil, nil, ""},
		{"imports only", map[string]any{"imports": []any{"shared.md"}}, []string{"shared.md"}, ""},
		{"profile only", map[string]any{"tools": map[string]any{"profile": "test-ecosystem"}}, []string{profile}, ""},
		{"after imports", map[string]any{"imports": []any{"shared.md"}, "tools": map[string]any{"profile": "test-ecosystem"}}, []string{"shared.md", profile}, ""},
		{"object imports", map[string]any{"imports": map[string]any{"aw": []any{"shared.md"}}, "tools": map[string]any{"profile": "test-ecosystem"}}, []string{"shared.md", profile}, ""},
		{"unknown", map[string]any{"tools": map[string]any{"profile": "unknown"}}, nil, "unknown ecosystem profile"},
		{"empty", map[string]any{"tools": map[string]any{"profile": ""}}, nil, "must name a supported ecosystem"},
		{"wrong type", map[string]any{"tools": map[string]any{"profile": []any{"go"}}}, nil, "must name a supported ecosystem"},
		{"traversal", map[string]any{"tools": map[string]any{"profile": "../go"}}, nil, "must name a supported ecosystem"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			specs, err := importsWithToolProfile(tc.fm)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			var paths []string
			for _, spec := range specs {
				paths = append(paths, spec.Path)
			}
			require.Equal(t, tc.paths, paths)
		})
	}
}

func TestToolProfileNestedResolution(t *testing.T) {
	profile := BuiltinToolProfilePathPrefix + "go.md"
	item := importQueueItem{
		importPath: "org/repo/shared.md@main",
		remoteOrigin: &remoteImportOrigin{
			Owner: "org", Repo: "repo", Ref: "main",
		},
	}
	resolved, origin, err := resolveNestedImportPathAndOrigin(item, profile)
	require.NoError(t, err)
	require.Equal(t, profile, resolved)
	require.Nil(t, origin)
	require.Equal(t, profile, canonicalizeNestedImportPath(profile, "/nested", "/base", nil, profile))
}
