//go:build !integration

package workflow

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const formalEscalationTitlePrefix = "[Schema Drift SLA]"

type formalEscalationIssue struct {
	Labels            []string
	Title             string
	DriftDetectedOn   string
	SourceWorkflowRun string
	Owner             string
	UnblockPlan       []string
	RevisedETA        string
	WaiverRationale   string
}

func formalHasLabelPair(issue formalEscalationIssue) bool {
	return slices.Contains(issue.Labels, "workflow") && slices.Contains(issue.Labels, "bug")
}

func formalHasTitlePrefix(issue formalEscalationIssue) bool {
	return strings.HasPrefix(issue.Title, formalEscalationTitlePrefix)
}

func formalTemplateFieldsComplete(issue formalEscalationIssue) bool {
	return strings.TrimSpace(issue.DriftDetectedOn) != "" &&
		strings.TrimSpace(issue.SourceWorkflowRun) != "" &&
		strings.TrimSpace(issue.Owner) != "" &&
		formalHasUnblockPlan(issue.UnblockPlan) &&
		strings.TrimSpace(issue.RevisedETA) != ""
}

func formalHasUnblockPlan(steps []string) bool {
	for _, step := range steps {
		if strings.TrimSpace(step) != "" {
			return true
		}
	}
	return false
}

func formalIsValidEscalationIssue(issue formalEscalationIssue) bool {
	return formalHasLabelPair(issue) &&
		formalHasTitlePrefix(issue) &&
		formalTemplateFieldsComplete(issue)
}

func formalValidEscalationIssue() formalEscalationIssue {
	return formalEscalationIssue{
		Labels:            []string{"workflow", "bug"},
		Title:             "[Schema Drift SLA] Config source mismatch",
		DriftDetectedOn:   "2026-08-21",
		SourceWorkflowRun: "https://github.com/github/gh-aw/actions/runs/123",
		Owner:             "@maintainer",
		UnblockPlan:       []string{"Update the config source mapping"},
		RevisedETA:        "2026-08-28",
	}
}

func TestFormalEscalation_LabelPairComplete(t *testing.T) {
	issue := formalValidEscalationIssue()
	assert.True(t, formalHasLabelPair(issue))

	issue.Labels = []string{"workflow"}
	assert.False(t, formalHasLabelPair(issue))

	issue.Labels = []string{"bug"}
	assert.False(t, formalHasLabelPair(issue))

	issue.Labels = nil
	assert.False(t, formalHasLabelPair(issue))
}

func TestFormalEscalation_TitlePrefixed(t *testing.T) {
	issue := formalValidEscalationIssue()
	assert.True(t, formalHasTitlePrefix(issue))

	issue.Title = "Config mismatch [Schema Drift SLA]"
	assert.False(t, formalHasTitlePrefix(issue))

	issue.Title = "Schema Drift SLA: Config mismatch"
	assert.False(t, formalHasTitlePrefix(issue))
}

func TestFormalEscalation_TemplateFieldsComplete(t *testing.T) {
	valid := formalValidEscalationIssue()
	assert.True(t, formalTemplateFieldsComplete(valid))

	tests := map[string]func(*formalEscalationIssue){
		"drift detected on":   func(issue *formalEscalationIssue) { issue.DriftDetectedOn = "" },
		"source workflow run": func(issue *formalEscalationIssue) { issue.SourceWorkflowRun = "" },
		"owner":               func(issue *formalEscalationIssue) { issue.Owner = "" },
		"unblock plan":        func(issue *formalEscalationIssue) { issue.UnblockPlan = nil },
		"revised ETA":         func(issue *formalEscalationIssue) { issue.RevisedETA = "" },
		"whitespace only":     func(issue *formalEscalationIssue) { issue.Owner = " " },
		"blank unblock plan":  func(issue *formalEscalationIssue) { issue.UnblockPlan = []string{" "} },
	}
	for name, removeField := range tests {
		t.Run(name, func(t *testing.T) {
			issue := valid
			removeField(&issue)
			assert.False(t, formalTemplateFieldsComplete(issue))
		})
	}
}

func TestFormalEscalation_WaiverRationaleOptional(t *testing.T) {
	withoutWaiver := formalValidEscalationIssue()
	withWaiver := withoutWaiver
	withWaiver.WaiverRationale = "Awaiting an upstream schema release"

	assert.Equal(t, formalTemplateFieldsComplete(withoutWaiver), formalTemplateFieldsComplete(withWaiver))
	assert.True(t, formalTemplateFieldsComplete(withoutWaiver))
}

func TestFormalEscalation_PartialLabelSetRejected(t *testing.T) {
	for _, labels := range [][]string{{"workflow"}, {"bug"}} {
		issue := formalValidEscalationIssue()
		issue.Labels = labels
		assert.False(t, formalIsValidEscalationIssue(issue))
	}
}

func TestFormalEscalation_TitlePrefixExactBoundary(t *testing.T) {
	issue := formalValidEscalationIssue()
	issue.Title = formalEscalationTitlePrefix
	assert.True(t, formalHasTitlePrefix(issue))

	issue.Title = "[Schema Drift SLA"
	assert.False(t, formalHasTitlePrefix(issue))
}

func TestFormalEscalation_OverallValidityGate(t *testing.T) {
	tests := map[string]struct {
		mutate func(*formalEscalationIssue)
		valid  bool
	}{
		"valid": {
			mutate: func(*formalEscalationIssue) {},
			valid:  true,
		},
		"missing label": {
			mutate: func(issue *formalEscalationIssue) { issue.Labels = []string{"workflow"} },
		},
		"missing title prefix": {
			mutate: func(issue *formalEscalationIssue) { issue.Title = "Config source mismatch" },
		},
		"missing owner": {
			mutate: func(issue *formalEscalationIssue) { issue.Owner = "" },
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			issue := formalValidEscalationIssue()
			test.mutate(&issue)
			assert.Equal(t, test.valid, formalIsValidEscalationIssue(issue))
		})
	}
}
