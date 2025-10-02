package parser

import (
	"testing"
)

func TestParseImportSpec(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOrg     string
		wantRepo    string
		wantVersion string
		wantPath    string
		wantErr     bool
	}{
		{
			name:        "valid import with single path segment",
			input:       "microsoft/genaiscript v1.5 agentics/engine.md",
			wantOrg:     "microsoft",
			wantRepo:    "genaiscript",
			wantVersion: "v1.5",
			wantPath:    "agentics/engine.md",
			wantErr:     false,
		},
		{
			name:        "valid import with nested path",
			input:       "github/copilot v2.0.0 workflows/shared/config.md",
			wantOrg:     "github",
			wantRepo:    "copilot",
			wantVersion: "v2.0.0",
			wantPath:    "workflows/shared/config.md",
			wantErr:     false,
		},
		{
			name:        "valid import with branch name",
			input:       "githubnext/gh-aw main README.md",
			wantOrg:     "githubnext",
			wantRepo:    "gh-aw",
			wantVersion: "main",
			wantPath:    "README.md",
			wantErr:     false,
		},
		{
			name:        "valid import with commit SHA",
			input:       "example/repo abc123def456 path/to/file.md",
			wantOrg:     "example",
			wantRepo:    "repo",
			wantVersion: "abc123def456",
			wantPath:    "path/to/file.md",
			wantErr:     false,
		},
		{
			name:    "invalid: missing path",
			input:   "microsoft/genaiscript v1.5",
			wantErr: true,
		},
		{
			name:    "invalid: missing version and path",
			input:   "microsoft/genaiscript",
			wantErr: true,
		},
		{
			name:    "invalid: no org slash",
			input:   "microsoft-genaiscript v1.5 path.md",
			wantErr: true,
		},
		{
			name:    "invalid: empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid: only whitespace",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ParseImportSpec(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseImportSpec() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseImportSpec() unexpected error: %v", err)
			}

			if spec.Org != tt.wantOrg {
				t.Errorf("ParseImportSpec() org = %v, want %v", spec.Org, tt.wantOrg)
			}
			if spec.Repo != tt.wantRepo {
				t.Errorf("ParseImportSpec() repo = %v, want %v", spec.Repo, tt.wantRepo)
			}
			if spec.Version != tt.wantVersion {
				t.Errorf("ParseImportSpec() version = %v, want %v", spec.Version, tt.wantVersion)
			}
			if spec.Path != tt.wantPath {
				t.Errorf("ParseImportSpec() path = %v, want %v", spec.Path, tt.wantPath)
			}
		})
	}
}

func TestImportSpec_RepoSlug(t *testing.T) {
	spec := &ImportSpec{
		Org:     "microsoft",
		Repo:    "genaiscript",
		Version: "v1.5",
		Path:    "agentics/engine.md",
	}

	expected := "microsoft/genaiscript"
	if spec.RepoSlug() != expected {
		t.Errorf("RepoSlug() = %v, want %v", spec.RepoSlug(), expected)
	}
}

func TestImportSpec_String(t *testing.T) {
	spec := &ImportSpec{
		Org:     "microsoft",
		Repo:    "genaiscript",
		Version: "v1.5",
		Path:    "agentics/engine.md",
	}

	expected := "microsoft/genaiscript v1.5 agentics/engine.md"
	if spec.String() != expected {
		t.Errorf("String() = %v, want %v", spec.String(), expected)
	}
}

func TestImportSpec_ValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid relative path",
			path:    "agentics/engine.md",
			wantErr: false,
		},
		{
			name:    "invalid absolute path",
			path:    "/agentics/engine.md",
			wantErr: true,
		},
		{
			name:    "invalid path with ..",
			path:    "../agentics/engine.md",
			wantErr: true,
		},
		{
			name:    "valid path with subdirs",
			path:    "workflows/shared/config.md",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ImportSpec{
				Org:     "test",
				Repo:    "repo",
				Version: "v1.0",
				Path:    tt.path,
			}

			err := spec.ValidatePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportSpec_IsVersionTag(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "semver tag v1.5",
			version: "v1.5",
			want:    true,
		},
		{
			name:    "semver tag v2.0.0",
			version: "v2.0.0",
			want:    true,
		},
		{
			name:    "branch name main",
			version: "main",
			want:    false,
		},
		{
			name:    "commit SHA",
			version: "abc123def456",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ImportSpec{
				Org:     "test",
				Repo:    "repo",
				Version: tt.version,
				Path:    "path.md",
			}

			if got := spec.IsVersionTag(); got != tt.want {
				t.Errorf("IsVersionTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseImports(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantCount int
		wantErr   bool
	}{
		{
			name: "valid array of imports",
			input: []interface{}{
				"microsoft/genaiscript v1.5 agentics/engine.md",
				"github/copilot v2.0.0 workflows/shared/config.md",
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "nil input",
			input:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "empty array",
			input:     []interface{}{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:    "invalid: not an array",
			input:   "microsoft/genaiscript v1.5 agentics/engine.md",
			wantErr: true,
		},
		{
			name: "invalid: array with non-string",
			input: []interface{}{
				"microsoft/genaiscript v1.5 agentics/engine.md",
				123,
			},
			wantErr: true,
		},
		{
			name: "invalid: array with invalid import spec",
			input: []interface{}{
				"microsoft/genaiscript v1.5 agentics/engine.md",
				"invalid-spec",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports, err := ParseImports(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseImports() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseImports() unexpected error: %v", err)
			}

			if len(imports) != tt.wantCount {
				t.Errorf("ParseImports() count = %v, want %v", len(imports), tt.wantCount)
			}
		})
	}
}

func TestImportSpec_GetLocalCachePath(t *testing.T) {
	spec := &ImportSpec{
		Org:     "microsoft",
		Repo:    "genaiscript",
		Version: "v1.5",
		Path:    "agentics/engine.md",
	}

	importsDir := "/home/user/.aw/imports"
	expected := "/home/user/.aw/imports/microsoft/genaiscript/v1.5"

	got := spec.GetLocalCachePath(importsDir)
	if got != expected {
		t.Errorf("GetLocalCachePath() = %v, want %v", got, expected)
	}
}
