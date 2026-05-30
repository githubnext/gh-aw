//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
)

func TestExtractSafeOutputLabels_IncludesLabelCommand(t *testing.T) {
	data := &workflow.WorkflowData{
		SafeOutputs: &workflow.SafeOutputsConfig{
			CreateIssues: &workflow.CreateIssuesConfig{
				Labels:        []string{"bug"},
				AllowedLabels: []string{"triage"},
			},
			AddLabels: &workflow.AddLabelsConfig{
				Allowed: []string{"automation"},
			},
		},
		LabelCommand: []string{"deploy"},
	}

	got := extractSafeOutputLabels(data)
	want := []string{"bug", "triage", "automation", "deploy"}

	if len(got) != len(want) {
		t.Fatalf("expected %d labels, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected labels %v, got %v", want, got)
		}
	}
}

func TestExtractSafeOutputLabels_DeduplicatesAcrossSources(t *testing.T) {
	data := &workflow.WorkflowData{
		SafeOutputs: &workflow.SafeOutputsConfig{
			CreatePullRequests: &workflow.CreatePullRequestsConfig{
				Labels: []string{"deploy"},
			},
		},
		LabelCommand: []string{"deploy"},
	}

	got := extractSafeOutputLabels(data)
	if len(got) != 1 || got[0] != "deploy" {
		t.Fatalf("expected deduplicated labels [deploy], got %v", got)
	}
}
