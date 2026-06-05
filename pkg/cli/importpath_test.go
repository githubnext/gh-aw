//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveImportPath_ImportsOpts characterises the imports.go call-site preset:
// workflowspec paths are returned unchanged, "/" prefix paths become repo-relative
// strings, and relative paths are cleaned and forward-slash normalised.
func TestResolveImportPath_ImportsOpts(t *testing.T) {
	tmpDir := t.TempDir()
	workflowDir := filepath.ToSlash(filepath.Join(tmpDir, "workflows"))

	opts := importPathResolverOpts{
		WorkflowSpecPassthrough: true,
		RepoRelativeAbsolute:    true,
		NormalizeSlash:          true,
	}

	tests := []struct {
		name       string
		importPath string
		expected   string
	}{
		{
			name:       "workflowspec with @ is returned unchanged",
			importPath: "owner/repo/path/file.md@abc123",
			expected:   "owner/repo/path/file.md@abc123",
		},
		{
			name:       "workflowspec without @ is returned unchanged",
			importPath: "owner/repo/path/file.md",
			expected:   "owner/repo/path/file.md",
		},
		{
			name:       "absolute path strips leading /",
			importPath: "/shared/file.md",
			expected:   "shared/file.md",
		},
		{
			name:       "relative path is resolved against baseDir",
			importPath: "shared/file.md",
			expected:   filepath.ToSlash(filepath.Clean(filepath.Join(workflowDir, "shared/file.md"))),
		},
		{
			name:       "relative path with .. is cleaned",
			importPath: "../other/file.md",
			expected:   filepath.ToSlash(filepath.Clean(filepath.Join(workflowDir, "../other/file.md"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveImportPath(tt.importPath, workflowDir, opts)
			assert.Equal(t, tt.expected, result,
				"resolveImportPath(%q, %q, importsOpts) = %q", tt.importPath, workflowDir, result)
		})
	}
}

// TestResolveImportPath_RunPushOpts characterises the run_push.go call-site preset:
// section refs are stripped, workflowspec paths return "", and relative paths are
// joined with baseDir.
func TestResolveImportPath_RunPushOpts(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "workflows")
	require.NoError(t, os.MkdirAll(baseDir, 0755))

	opts := importPathResolverOpts{
		StripSectionRef:  true,
		WorkflowSpecSkip: true,
	}

	tests := []struct {
		name       string
		importPath string
		expected   string
	}{
		{
			name:       "relative path",
			importPath: "shared/file.md",
			expected:   filepath.Join(baseDir, "shared/file.md"),
		},
		{
			name:       "section ref is stripped",
			importPath: "shared/file.md#section",
			expected:   filepath.Join(baseDir, "shared/file.md"),
		},
		{
			name:       "workflowspec with @ returns empty",
			importPath: "owner/repo/path/file.md@abc123",
			expected:   "",
		},
		{
			name:       "workflowspec without @ returns empty",
			importPath: "owner/repo/path/file.md",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveImportPath(tt.importPath, baseDir, opts)
			assert.Equal(t, tt.expected, result,
				"resolveImportPath(%q, %q, runPushOpts) = %q", tt.importPath, baseDir, result)
		})
	}
}
