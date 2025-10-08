package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseChangesetFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantType    string
		wantPackage string
	}{
		{
			name: "valid patch changeset",
			content: `---
"gh-aw": patch
---

Fix some bug in the code`,
			wantErr:     false,
			wantType:    "patch",
			wantPackage: "gh-aw",
		},
		{
			name: "valid minor changeset",
			content: `---
"gh-aw": minor
---

Add a new feature`,
			wantErr:     false,
			wantType:    "minor",
			wantPackage: "gh-aw",
		},
		{
			name: "valid major changeset",
			content: `---
"gh-aw": major
---

Breaking change`,
			wantErr:     false,
			wantType:    "major",
			wantPackage: "gh-aw",
		},
		{
			name: "missing frontmatter",
			content: `This is just text without frontmatter

No version info here`,
			wantErr: true,
		},
		{
			name: "invalid frontmatter",
			content: `---
invalid yaml: [
---

Some description`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the file
			result, err := parseChangesetFile(tmpFile)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("parseChangesetFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			// Validate results
			if result.BumpType != tt.wantType {
				t.Errorf("parseChangesetFile() BumpType = %v, want %v", result.BumpType, tt.wantType)
			}
			if result.Package != tt.wantPackage {
				t.Errorf("parseChangesetFile() Package = %v, want %v", result.Package, tt.wantPackage)
			}
		})
	}
}

func TestDetermineVersionBump(t *testing.T) {
	tests := []struct {
		name       string
		changesets []*ChangesetEntry
		want       string
	}{
		{
			name:       "empty changesets",
			changesets: []*ChangesetEntry{},
			want:       "",
		},
		{
			name: "single patch",
			changesets: []*ChangesetEntry{
				{BumpType: "patch"},
			},
			want: "patch",
		},
		{
			name: "single minor",
			changesets: []*ChangesetEntry{
				{BumpType: "minor"},
			},
			want: "minor",
		},
		{
			name: "single major",
			changesets: []*ChangesetEntry{
				{BumpType: "major"},
			},
			want: "major",
		},
		{
			name: "patch and minor - minor wins",
			changesets: []*ChangesetEntry{
				{BumpType: "patch"},
				{BumpType: "minor"},
			},
			want: "minor",
		},
		{
			name: "minor and major - major wins",
			changesets: []*ChangesetEntry{
				{BumpType: "minor"},
				{BumpType: "major"},
			},
			want: "major",
		},
		{
			name: "all types - major wins",
			changesets: []*ChangesetEntry{
				{BumpType: "patch"},
				{BumpType: "minor"},
				{BumpType: "major"},
			},
			want: "major",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineVersionBump(tt.changesets)
			if got != tt.want {
				t.Errorf("determineVersionBump() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		name     string
		current  *VersionInfo
		bumpType string
		want     *VersionInfo
	}{
		{
			name:     "patch bump",
			current:  &VersionInfo{Major: 1, Minor: 2, Patch: 3},
			bumpType: "patch",
			want:     &VersionInfo{Major: 1, Minor: 2, Patch: 4},
		},
		{
			name:     "minor bump resets patch",
			current:  &VersionInfo{Major: 1, Minor: 2, Patch: 3},
			bumpType: "minor",
			want:     &VersionInfo{Major: 1, Minor: 3, Patch: 0},
		},
		{
			name:     "major bump resets minor and patch",
			current:  &VersionInfo{Major: 1, Minor: 2, Patch: 3},
			bumpType: "major",
			want:     &VersionInfo{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:     "patch from zero",
			current:  &VersionInfo{Major: 0, Minor: 0, Patch: 0},
			bumpType: "patch",
			want:     &VersionInfo{Major: 0, Minor: 0, Patch: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bumpVersion(tt.current, tt.bumpType)
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("bumpVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name    string
		version *VersionInfo
		want    string
	}{
		{
			name:    "basic version",
			version: &VersionInfo{Major: 1, Minor: 2, Patch: 3},
			want:    "v1.2.3",
		},
		{
			name:    "zero version",
			version: &VersionInfo{Major: 0, Minor: 0, Patch: 0},
			want:    "v0.0.0",
		},
		{
			name:    "large version numbers",
			version: &VersionInfo{Major: 10, Minor: 20, Patch: 30},
			want:    "v10.20.30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.version)
			if got != tt.want {
				t.Errorf("formatVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFirstLine(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single line",
			text: "This is a single line",
			want: "This is a single line",
		},
		{
			name: "multiple lines",
			text: "First line\nSecond line\nThird line",
			want: "First line",
		},
		{
			name: "empty first line",
			text: "\n\nActual content",
			want: "Actual content",
		},
		{
			name: "whitespace first line",
			text: "  \n  \nActual content",
			want: "Actual content",
		},
		{
			name: "empty string",
			text: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstLine(tt.text)
			if got != tt.want {
				t.Errorf("extractFirstLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateChangelog(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		changesets    []*ChangesetEntry
		existingLog   string
		wantContains  []string
		wantNotExists bool
	}{
		{
			name:    "create new changelog",
			version: "v1.0.0",
			changesets: []*ChangesetEntry{
				{BumpType: "minor", Description: "Add new feature"},
				{BumpType: "patch", Description: "Fix bug"},
			},
			existingLog:   "",
			wantNotExists: true,
			wantContains: []string{
				"# Changelog",
				"## v1.0.0",
				"### Features",
				"Add new feature",
				"### Bug Fixes",
				"Fix bug",
			},
		},
		{
			name:    "append to existing changelog",
			version: "v1.1.0",
			changesets: []*ChangesetEntry{
				{BumpType: "minor", Description: "Another feature"},
			},
			existingLog: `# Changelog

All notable changes to this project will be documented in this file.

## v1.0.0 - 2024-01-01

### Features

- Initial release
`,
			wantContains: []string{
				"## v1.1.0",
				"Another feature",
				"## v1.0.0",
				"Initial release",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			origDir, _ := os.Getwd()
			defer func() { _ = os.Chdir(origDir) }()
			_ = os.Chdir(tmpDir)

			// Write existing changelog if provided
			if tt.existingLog != "" {
				if err := os.WriteFile("CHANGELOG.md", []byte(tt.existingLog), 0644); err != nil {
					t.Fatalf("Failed to write existing changelog: %v", err)
				}
			}

			// Update changelog
			if err := updateChangelog(tt.version, tt.changesets); err != nil {
				t.Fatalf("updateChangelog() error = %v", err)
			}

			// Read result
			content, err := os.ReadFile("CHANGELOG.md")
			if err != nil {
				t.Fatalf("Failed to read changelog: %v", err)
			}

			contentStr := string(content)

			// Check for expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(contentStr, want) {
					t.Errorf("CHANGELOG.md missing expected content: %q\nGot:\n%s", want, contentStr)
				}
			}

			// Verify version appears before older versions
			if tt.existingLog != "" && strings.Contains(tt.existingLog, "v1.0.0") {
				v110Index := strings.Index(contentStr, "v1.1.0")
				v100Index := strings.Index(contentStr, "v1.0.0")
				if v110Index == -1 || v100Index == -1 {
					t.Error("Could not find version markers in changelog")
				} else if v110Index > v100Index {
					t.Error("New version should appear before old version in changelog")
				}
			}
		})
	}
}
