package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWorkflows(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		staged       bool
		verbose      bool
		workflowDir  string
		wantErr      bool
	}{
		{
			name:         "invalid absolute workflow dir",
			workflowName: "",
			staged:       false,
			verbose:      false,
			workflowDir:  "/absolute/path",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test specifically checks the workflow-dir validation
			// which happens before git checks, so we can test it without git setup
			err := UpdateWorkflows(tt.workflowName, tt.staged, tt.verbose, tt.workflowDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateWorkflows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				expectedMsg := "workflow-dir must be a relative path"
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Errorf("UpdateWorkflows() error = %v, expected to contain %q", err, expectedMsg)
				}
			}
		})
	}
}

func TestCheckPackageForUpdates(t *testing.T) {
	tests := []struct {
		name          string
		pkg           Package
		verbose       bool
		wantHasUpdate bool
		wantErr       bool
	}{
		{
			name: "package with no commit SHA",
			pkg: Package{
				Name:      "test/package",
				CommitSHA: "",
			},
			verbose:       false,
			wantHasUpdate: true,
			wantErr:       false,
		},
		{
			name: "package with empty commit SHA",
			pkg: Package{
				Name:      "test/package",
				CommitSHA: "",
			},
			verbose:       true,
			wantHasUpdate: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasUpdate, err := checkPackageForUpdates(tt.pkg, tt.verbose)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkPackageForUpdates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if hasUpdate != tt.wantHasUpdate {
				t.Errorf("checkPackageForUpdates() hasUpdate = %v, want %v", hasUpdate, tt.wantHasUpdate)
			}
		})
	}
}

func TestFilterPackagesByWorkflow(t *testing.T) {
	packages := []Package{
		{
			Name:      "test/package1",
			Workflows: []string{"workflow1", "workflow2"},
		},
		{
			Name:      "test/package2",
			Workflows: []string{"workflow3"},
		},
		{
			Name:      "test/package3",
			Workflows: []string{"workflow1", "workflow4"},
		},
	}

	tests := []struct {
		name         string
		packages     []Package
		workflowName string
		wantCount    int
	}{
		{
			name:         "filter by workflow1",
			packages:     packages,
			workflowName: "workflow1",
			wantCount:    2, // package1 and package3 contain workflow1
		},
		{
			name:         "filter by workflow3",
			packages:     packages,
			workflowName: "workflow3",
			wantCount:    1, // only package2 contains workflow3
		},
		{
			name:         "filter by nonexistent workflow",
			packages:     packages,
			workflowName: "nonexistent",
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterPackagesByWorkflow(tt.packages, tt.workflowName)

			if len(filtered) != tt.wantCount {
				t.Errorf("filterPackagesByWorkflow() returned %d packages, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestIsLocalPackage(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name        string
		packagePath string
		want        bool
	}{
		{
			name:        "local package path",
			packagePath: ".aw/packages/test/repo",
			want:        true,
		},
		{
			name:        "local package with subdirs",
			packagePath: "project/.aw/packages/test/repo",
			want:        true,
		},
		{
			name:        "global package path",
			packagePath: filepath.Join(homeDir, ".aw/packages/test/repo"),
			want:        false,
		},
		{
			name:        "other path",
			packagePath: "/some/other/path",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalPackage(tt.packagePath); got != tt.want {
				t.Errorf("isLocalPackage() = %v, want %v", got, tt.want)
			}
		})
	}
}
