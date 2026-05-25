//go:build !integration

package cli

import "testing"

func TestBuildSourceStringWithCommitSHA_ManifestSource(t *testing.T) {
	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug:    "owner/repo",
			Version:     "v1.2.3",
			PackagePath: "packages/repo-assist",
		},
		WorkflowPath:           "workflows/triage.md",
		FromRepositoryManifest: true,
	}

	got := buildSourceStringWithCommitSHA(workflow, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	want := "owner/repo/packages/repo-assist@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildSourceStringWithCommitSHA_ManifestSourceRoot(t *testing.T) {
	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "owner/repo",
			Version:  "v1.2.3",
		},
		WorkflowPath:           "workflows/triage.md",
		FromRepositoryManifest: true,
	}

	got := buildSourceStringWithCommitSHA(workflow, "")
	want := "owner/repo@v1.2.3"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParseManifestSourceSpec(t *testing.T) {
	tests := []struct {
		source       string
		wantManifest bool
		wantRepo     string
		wantPackage  string
		wantVersion  string
	}{
		{source: "owner/repo@v1.0.0", wantManifest: true, wantRepo: "owner/repo", wantVersion: "v1.0.0"},
		{source: "owner/repo/packages/repo-assist@main", wantManifest: true, wantRepo: "owner/repo", wantPackage: "packages/repo-assist", wantVersion: "main"},
		{source: "owner/repo/workflows/triage.md@v1.0.0", wantManifest: false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			spec, isManifest, err := parseManifestSourceSpec(tt.source)
			if err != nil {
				t.Fatalf("parseManifestSourceSpec returned error: %v", err)
			}
			if isManifest != tt.wantManifest {
				t.Fatalf("expected manifest=%v, got %v", tt.wantManifest, isManifest)
			}
			if !tt.wantManifest {
				return
			}
			if spec.RepoSlug != tt.wantRepo || spec.PackagePath != tt.wantPackage || spec.Version != tt.wantVersion {
				t.Fatalf("unexpected repo spec: %+v", spec)
			}
		})
	}
}
