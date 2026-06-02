//go:build !integration

package cli

import "testing"

func TestAppendRepositoryPackageWorkflowSpecs_UsesResolvedRefWhenVersionOmitted(t *testing.T) {
	repoSpec := &RepoSpec{
		RepoSlug: "owner/repo",
	}
	pkg := &resolvedRepositoryPackage{
		ResolvedRef:        "v1.2.3",
		InstallationSource: []string{"workflows/review.md"},
	}

	specs := appendRepositoryPackageWorkflowSpecs(nil, repoSpec, pkg)
	if len(specs) != 1 {
		t.Fatalf("expected 1 workflow spec, got %d", len(specs))
	}
	if specs[0].Version != "v1.2.3" {
		t.Fatalf("expected resolved version v1.2.3, got %q", specs[0].Version)
	}
}
