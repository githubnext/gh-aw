//go:build !integration

package cli

import "testing"

func TestAppendRepositoryPackageWorkflowSpecs_PropagatesResolvedRef(t *testing.T) {
	repoSpec := &RepoSpec{
		RepoSlug: "owner/repo",
	}
	pkg := &resolvedRepositoryPackage{
		ResolvedRef:        "v1.2.3",
		InstallationSource: []string{"workflows/review.md"},
		SkillFiles: []resolvedPackageSkillFile{
			{SourcePath: "skills/review/SKILL.md", SkillName: "review"},
		},
		AgentFiles: []string{"agents/review-agent.md"},
	}

	specs := appendRepositoryPackageWorkflowSpecs(nil, repoSpec, pkg)
	if len(specs) != 3 {
		t.Fatalf("expected 3 workflow specs, got %d", len(specs))
	}
	for i, spec := range specs {
		if spec.Version != "v1.2.3" {
			t.Fatalf("expected resolved version v1.2.3 for spec %d, got %q", i, spec.Version)
		}
	}
}
