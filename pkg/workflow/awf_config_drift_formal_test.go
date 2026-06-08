//go:build !integration

package workflow

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type FormalDriftRecord struct {
	PropertyPath    string
	DriftCategory   string
	SuggestedAction string
	DetectedAt      string
}

type formalEscalationIssue struct {
	Owner       string
	UnblockPlan []string
	RevisedETA  time.Time
}

type formalSafeguardState struct {
	UseSnapshot              bool
	WarningEmitted           bool
	DestructiveOpsSuppressed bool
	DegradedMode             bool
}

func formalDualSourceConsulted(normativeSpecConsulted, publishedSchemaConsulted bool) bool {
	return normativeSpecConsulted && publishedSchemaConsulted
}

func formalNoUndocumentedFieldGeneration(generatedFields []string, documentedFields map[string]struct{}) bool {
	for _, field := range generatedFields {
		if _, ok := documentedFields[field]; !ok {
			return false
		}
	}
	return true
}

func formalDriftRecordStructuralValidity(record FormalDriftRecord) bool {
	return record.PropertyPath != "" &&
		record.DriftCategory != "" &&
		record.SuggestedAction != "" &&
		record.DetectedAt != ""
}

func formalDriftCategoryExhaustiveness(category string) bool {
	return slices.Contains([]string{"missing_in_ghaw", "missing_in_schema", "spec_mismatch"}, category)
}

func formalSchemaOnlyPropertyFlaggedAsDrift(schemaProperties, implementationCoverage []string) []FormalDriftRecord {
	covered := map[string]struct{}{}
	for _, property := range implementationCoverage {
		covered[property] = struct{}{}
	}

	drift := make([]FormalDriftRecord, 0)
	for _, property := range schemaProperties {
		if _, ok := covered[property]; ok {
			continue
		}
		drift = append(drift, FormalDriftRecord{
			PropertyPath:    property,
			DriftCategory:   "missing_in_ghaw",
			SuggestedAction: "Add coverage for " + property,
			DetectedAt:      time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
	}
	return drift
}

func formalCorrectionPRForActionableDrift(category string) bool {
	return category == "missing_in_ghaw" || category == "spec_mismatch"
}

func formalAddBusinessDays(start time.Time, days int) time.Time {
	current := start.UTC()
	added := 0
	for added < days {
		current = current.AddDate(0, 0, 1)
		weekday := current.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}
		added++
	}
	return current
}

func formalSLARemediationWindow(detectedAt, now time.Time) bool {
	deadline := formalAddBusinessDays(detectedAt, 5)
	return !now.After(deadline)
}

func formalEscalationIssueStructure(issue formalEscalationIssue) bool {
	return issue.Owner != "" && len(issue.UnblockPlan) > 0 && !issue.RevisedETA.IsZero()
}

func formalSafeguardDegradedModeOnUnavailability(canonicalSourcesAvailable, hasLastKnownSnapshot bool) formalSafeguardState {
	if canonicalSourcesAvailable {
		return formalSafeguardState{}
	}
	return formalSafeguardState{
		UseSnapshot:              hasLastKnownSnapshot,
		WarningEmitted:           true,
		DestructiveOpsSuppressed: true,
		DegradedMode:             true,
	}
}

func formalDriftReportEmittedOnDetection(schemaProperties, implementationCoverage []string) []FormalDriftRecord {
	drift := formalSchemaOnlyPropertyFlaggedAsDrift(schemaProperties, implementationCoverage)
	if drift == nil {
		return []FormalDriftRecord{}
	}
	return drift
}

func TestFormal_P1_DualSourceConsultation(t *testing.T) {
	assert.True(t, formalDualSourceConsulted(true, true), "both normative spec and published schema must be consulted")
	assert.False(t, formalDualSourceConsulted(true, false), "single-source consultation is non-conformant")
	assert.False(t, formalDualSourceConsulted(false, true), "single-source consultation is non-conformant")
}

func TestFormal_P2_NoUndocumentedFieldGeneration(t *testing.T) {
	documented := map[string]struct{}{
		"apiProxy.anthropicAutoCache":    {},
		"container.dockerHostPathPrefix": {},
	}

	assert.True(t, formalNoUndocumentedFieldGeneration([]string{"apiProxy.anthropicAutoCache"}, documented))
	assert.False(t, formalNoUndocumentedFieldGeneration([]string{"apiProxy.undocumentedField"}, documented))
}

func TestFormal_P3_DriftRecordStructuralValidity(t *testing.T) {
	valid := FormalDriftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "Add coverage",
		DetectedAt:      "2026-06-08T00:00:00Z",
	}
	invalid := FormalDriftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "",
		DetectedAt:      "2026-06-08T00:00:00Z",
	}

	assert.True(t, formalDriftRecordStructuralValidity(valid))
	assert.False(t, formalDriftRecordStructuralValidity(invalid))
}

func TestFormal_P4_DriftCategoryExhaustiveness(t *testing.T) {
	assert.True(t, formalDriftCategoryExhaustiveness("missing_in_ghaw"))
	assert.True(t, formalDriftCategoryExhaustiveness("missing_in_schema"))
	assert.True(t, formalDriftCategoryExhaustiveness("spec_mismatch"))
	assert.False(t, formalDriftCategoryExhaustiveness("missing_in_gh_aw"))
	assert.False(t, formalDriftCategoryExhaustiveness("unknown"))
}

func TestFormal_P5_SchemaOnlyPropertyFlaggedAsDrift(t *testing.T) {
	schema := []string{"apiProxy.anthropicAutoCache", "container.dockerHostPathPrefix"}
	covered := []string{"container.dockerHostPathPrefix"}

	drift := formalSchemaOnlyPropertyFlaggedAsDrift(schema, covered)
	assert.Len(t, drift, 1)
	assert.Equal(t, "apiProxy.anthropicAutoCache", drift[0].PropertyPath)
	assert.Equal(t, "missing_in_ghaw", drift[0].DriftCategory)
}

func TestFormal_P6_CorrectionPRForActionableDrift(t *testing.T) {
	assert.True(t, formalCorrectionPRForActionableDrift("missing_in_ghaw"))
	assert.True(t, formalCorrectionPRForActionableDrift("spec_mismatch"))
	assert.False(t, formalCorrectionPRForActionableDrift("missing_in_schema"))
}

func TestFormal_P7_SLARemediationWindow(t *testing.T) {
	detected := time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC) // Monday
	deadline := formalAddBusinessDays(detected, 5)

	assert.Equal(t, time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC), deadline, "5 business days should skip weekend")
	assert.True(t, formalSLARemediationWindow(detected, deadline))
	assert.False(t, formalSLARemediationWindow(detected, deadline.Add(time.Second)))
}

func TestFormal_P8_EscalationIssueStructure(t *testing.T) {
	valid := formalEscalationIssue{
		Owner:       "@maintainer",
		UnblockPlan: []string{"reproduce drift", "ship corrective PR"},
		RevisedETA:  time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC),
	}
	invalid := formalEscalationIssue{Owner: "", UnblockPlan: nil, RevisedETA: time.Time{}}

	assert.True(t, formalEscalationIssueStructure(valid))
	assert.False(t, formalEscalationIssueStructure(invalid))
}

func TestFormal_P9_SafeguardDegradedModeOnUnavailability(t *testing.T) {
	state := formalSafeguardDegradedModeOnUnavailability(false, true)
	assert.True(t, state.UseSnapshot)
	assert.True(t, state.WarningEmitted)
	assert.True(t, state.DestructiveOpsSuppressed)
	assert.True(t, state.DegradedMode)
}

func TestFormal_P10_DriftReportEmittedOnDetection(t *testing.T) {
	drift := formalDriftReportEmittedOnDetection(
		[]string{"apiProxy.anthropicAutoCache"},
		[]string{},
	)
	assert.NotNil(t, drift)
	assert.Len(t, drift, 1)

	empty := formalDriftReportEmittedOnDetection(
		[]string{"container.dockerHostPathPrefix"},
		[]string{"container.dockerHostPathPrefix"},
	)
	assert.NotNil(t, empty)
	assert.Empty(t, empty)
}
