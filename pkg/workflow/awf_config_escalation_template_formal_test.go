//go:build !integration

package workflow

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Formal predicates P17-P19 for specs/awf-config-sources-spec.md Section 7.4.1
// (Drift SLA tracking, CR-06): escalation issue label pair, title prefix, and
// minimum required template fields.

const (
	formalEscalationLabelWorkflow = "workflow"
	formalEscalationLabelBug      = "bug"
	formalEscalationTitlePrefixed = "[Schema Drift SLA]"
)

// formalEscalationTemplate models the minimum required fields of the Section
// 7.4.1 escalation template. WaiverRationale is explicitly optional ("if any").
type formalEscalationTemplate struct {
	DriftDetectedOn   time.Time
	SourceWorkflowRun string
	Owner             string
	UnblockPlan       []string
	RevisedETA        time.Time
	WaiverRationale   string
}

// formalEscalationIssueRecord models a full escalation issue (P17-P19 composite).
type formalEscalationIssueRecord struct {
	Labels   []string
	Title    string
	Template formalEscalationTemplate
}

// formalEscalationLabelPairComplete (P17): escalation issues carry both the
// `workflow` and `bug` labels.
func formalEscalationLabelPairComplete(labels []string) bool {
	return slices.Contains(labels, formalEscalationLabelWorkflow) &&
		slices.Contains(labels, formalEscalationLabelBug)
}

// formalEscalationTitlePrefix (P18): escalation issue titles begin with the
// `[Schema Drift SLA]` prefix.
func formalEscalationTitlePrefix(title string) bool {
	return strings.HasPrefix(title, formalEscalationTitlePrefixed)
}

// formalEscalationTemplateFieldsComplete (P19): every required template field is
// populated; the waiver rationale remains optional.
func formalEscalationTemplateFieldsComplete(template formalEscalationTemplate) bool {
	return !template.DriftDetectedOn.IsZero() &&
		template.SourceWorkflowRun != "" &&
		template.Owner != "" &&
		len(template.UnblockPlan) > 0 &&
		!template.RevisedETA.IsZero()
}

// formalEscalationIssueValid is the P17-P19 composite guard conjunction.
func formalEscalationIssueValid(issue formalEscalationIssueRecord) bool {
	return formalEscalationLabelPairComplete(issue.Labels) &&
		formalEscalationTitlePrefix(issue.Title) &&
		formalEscalationTemplateFieldsComplete(issue.Template)
}

func formalValidEscalationTemplate() formalEscalationTemplate {
	return formalEscalationTemplate{
		DriftDetectedOn:   time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC),
		SourceWorkflowRun: "https://github.com/github/gh-aw/actions/runs/31955892199",
		Owner:             "@on-call",
		UnblockPlan:       []string{"Reconcile canonical schema", "Open corrective PR"},
		RevisedETA:        time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
	}
}

func TestFormalP17_EscalationLabelPairComplete(t *testing.T) {
	assert.True(t, formalEscalationLabelPairComplete([]string{"workflow", "bug"}))
	assert.True(t, formalEscalationLabelPairComplete([]string{"bug", "automation", "workflow"}))
	assert.False(t, formalEscalationLabelPairComplete(nil))
}

func TestFormalP17_PartialLabelSetRejected(t *testing.T) {
	assert.False(t, formalEscalationLabelPairComplete([]string{"workflow"}))
	assert.False(t, formalEscalationLabelPairComplete([]string{"bug"}))
	assert.False(t, formalEscalationLabelPairComplete([]string{"workflows", "bugs"}))
}

func TestFormalP18_EscalationTitlePrefix(t *testing.T) {
	assert.True(t, formalEscalationTitlePrefix("[Schema Drift SLA] apiProxy.anthropicAutoCache unresolved"))
	assert.False(t, formalEscalationTitlePrefix("Schema Drift SLA: unresolved drift"))
	assert.False(t, formalEscalationTitlePrefix("drift [Schema Drift SLA] unresolved"))
}

func TestFormalP18_ShortTitleRejected(t *testing.T) {
	assert.False(t, formalEscalationTitlePrefix(""))
	assert.False(t, formalEscalationTitlePrefix("[Schema Drift"))
	assert.True(t, formalEscalationTitlePrefix(formalEscalationTitlePrefixed))
}

func TestFormalP19_EscalationTemplateFieldsComplete(t *testing.T) {
	template := formalValidEscalationTemplate()
	assert.True(t, formalEscalationTemplateFieldsComplete(template))

	missingRun := template
	missingRun.SourceWorkflowRun = ""
	assert.False(t, formalEscalationTemplateFieldsComplete(missingRun))

	missingOwner := template
	missingOwner.Owner = ""
	assert.False(t, formalEscalationTemplateFieldsComplete(missingOwner))

	missingDetectedOn := template
	missingDetectedOn.DriftDetectedOn = time.Time{}
	assert.False(t, formalEscalationTemplateFieldsComplete(missingDetectedOn))

	missingETA := template
	missingETA.RevisedETA = time.Time{}
	assert.False(t, formalEscalationTemplateFieldsComplete(missingETA))
}

func TestFormalP19_WaiverRationaleOptional(t *testing.T) {
	template := formalValidEscalationTemplate()
	assert.Empty(t, template.WaiverRationale)
	assert.True(t, formalEscalationTemplateFieldsComplete(template))

	withWaiver := template
	withWaiver.WaiverRationale = "Upstream outage; waiting on canonical source restore."
	assert.True(t, formalEscalationTemplateFieldsComplete(withWaiver))
}

func TestFormalP19_EmptyUnblockPlanRejected(t *testing.T) {
	template := formalValidEscalationTemplate()

	template.UnblockPlan = []string{}
	assert.False(t, formalEscalationTemplateFieldsComplete(template))

	template.UnblockPlan = nil
	assert.False(t, formalEscalationTemplateFieldsComplete(template))
}

func TestFormalEscalationIssueBuild_TableDriven(t *testing.T) {
	validIssue := formalEscalationIssueRecord{
		Labels:   []string{formalEscalationLabelWorkflow, formalEscalationLabelBug},
		Title:    formalEscalationTitlePrefixed + " apiProxy.anthropicAutoCache unresolved",
		Template: formalValidEscalationTemplate(),
	}

	missingLabel := validIssue
	missingLabel.Labels = []string{formalEscalationLabelWorkflow}

	badTitle := validIssue
	badTitle.Title = "Schema Drift SLA escalation"

	incompleteTemplate := validIssue
	incompleteTemplate.Template.Owner = ""

	tests := []struct {
		name  string
		issue formalEscalationIssueRecord
		want  bool
	}{
		{name: "all predicates satisfied", issue: validIssue, want: true},
		{name: "missing bug label", issue: missingLabel, want: false},
		{name: "missing title prefix", issue: badTitle, want: false},
		{name: "missing template owner", issue: incompleteTemplate, want: false},
		{name: "empty issue", issue: formalEscalationIssueRecord{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formalEscalationIssueValid(tt.issue))
		})
	}
}
