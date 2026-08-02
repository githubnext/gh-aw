//go:build !integration

package workflow

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FormalDriftRecord is the conformance test representation of the DriftRecord entity
// defined in §6.5 of specs/awf-config-sources-spec.md. The json tags use snake_case
// to match the normative field names in the specification.
type FormalDriftRecord struct {
	PropertyPath    string `json:"property_path"`
	DriftCategory   string `json:"drift_category"`
	SuggestedAction string `json:"suggested_action"`
	DetectedAt      string `json:"detected_at"`
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

// formalDetectedAtISO8601Valid returns true when detectedAt is a valid ISO 8601 UTC timestamp
// (RFC 3339 / time.RFC3339). Covers T-DR-003: §6.5.1 detected_at format requirement.
func formalDetectedAtISO8601Valid(detectedAt string) bool {
	_, err := time.Parse(time.RFC3339, detectedAt)
	return err == nil
}

// formalDriftRecordRejectsAdditionalProperties returns true when the JSON input contains
// properties beyond the four required DriftRecord fields and is therefore rejected.
// Covers T-DR-005: §6.5.1 no additional properties requirement.
func formalDriftRecordRejectsAdditionalProperties(jsonInput string) bool {
	dec := json.NewDecoder(strings.NewReader(jsonInput))
	dec.DisallowUnknownFields()
	var record FormalDriftRecord
	return dec.Decode(&record) != nil
}

// formalCorrectionPREmbedsDriftRecords checks whether prBody contains a JSON array that can
// be parsed as a list of FormalDriftRecord objects.
// Covers T-DR-008: §6.5.3 corrective PR embeds full DriftRecord list as JSON.
func formalCorrectionPREmbedsDriftRecords(prBody string) (bool, []FormalDriftRecord) {
	start := strings.Index(prBody, "[")
	if start == -1 {
		return false, nil
	}
	end := strings.LastIndex(prBody, "]")
	if end == -1 || end < start {
		return false, nil
	}
	var records []FormalDriftRecord
	if err := json.Unmarshal([]byte(prBody[start:end+1]), &records); err != nil {
		return false, nil
	}
	return true, records
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

// TestFormal_P3_DriftRecordStructuralValidity covers T-DR-001 (required fields must be present)
// and T-DR-004 (suggested_action MUST NOT be empty).
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

// TestFormal_P4_DriftCategoryExhaustiveness covers T-DR-002 (drift_category enum values).
func TestFormal_P4_DriftCategoryExhaustiveness(t *testing.T) {
	assert.True(t, formalDriftCategoryExhaustiveness("missing_in_ghaw"))
	assert.True(t, formalDriftCategoryExhaustiveness("missing_in_schema"))
	assert.True(t, formalDriftCategoryExhaustiveness("spec_mismatch"))
	assert.False(t, formalDriftCategoryExhaustiveness("missing_in_gh_aw"))
	assert.False(t, formalDriftCategoryExhaustiveness("unknown"))
}

// TestFormal_TDR003_DetectedAtISO8601Format covers T-DR-003: detected_at MUST be a valid
// ISO 8601 UTC timestamp; non-conforming values MUST be rejected.
func TestFormal_TDR003_DetectedAtISO8601Format(t *testing.T) {
	assert.True(t, formalDetectedAtISO8601Valid("2026-06-08T00:00:00Z"), "RFC 3339 UTC timestamp is valid")
	assert.True(t, formalDetectedAtISO8601Valid("2026-01-01T12:34:56Z"), "RFC 3339 UTC timestamp is valid")
	assert.False(t, formalDetectedAtISO8601Valid("2026-06-08"), "date-only string is not a valid ISO 8601 UTC timestamp")
	assert.False(t, formalDetectedAtISO8601Valid("not-a-date"), "arbitrary string is not a valid ISO 8601 UTC timestamp")
	assert.False(t, formalDetectedAtISO8601Valid(""), "empty string is not a valid ISO 8601 UTC timestamp")
}

// TestFormal_TDR005_NoAdditionalProperties covers T-DR-005: DriftRecord objects MUST NOT
// include properties beyond the four required fields; additional properties MUST be rejected.
func TestFormal_TDR005_NoAdditionalProperties(t *testing.T) {
	// A JSON object with an extra field must be rejected.
	assert.True(t, formalDriftRecordRejectsAdditionalProperties(`{
		"property_path": "apiProxy.anthropicAutoCache",
		"drift_category": "spec_mismatch",
		"suggested_action": "fix",
		"detected_at": "2026-06-08T00:00:00Z",
		"extra_field": "disallowed"
	}`), "JSON with an extra field must be rejected")
	// A JSON object with only the four required fields must be accepted (decoder succeeds, so
	// formalDriftRecordRejectsAdditionalProperties returns false indicating no rejection).
	assert.False(t, formalDriftRecordRejectsAdditionalProperties(`{
		"property_path": "apiProxy.anthropicAutoCache",
		"drift_category": "spec_mismatch",
		"suggested_action": "fix",
		"detected_at": "2026-06-08T00:00:00Z"
	}`), "JSON with only required fields must be accepted")
}

// TestFormal_P5_SchemaOnlyPropertyFlaggedAsDrift covers T-DR-001 structural validity in the
// context of drift detection output.
func TestFormal_P5_SchemaOnlyPropertyFlaggedAsDrift(t *testing.T) {
	schema := []string{"apiProxy.anthropicAutoCache", "container.dockerHostPathPrefix"}
	covered := []string{"container.dockerHostPathPrefix"}

	drift := formalSchemaOnlyPropertyFlaggedAsDrift(schema, covered)
	assert.Len(t, drift, 1)
	assert.Equal(t, "apiProxy.anthropicAutoCache", drift[0].PropertyPath)
	assert.Equal(t, "missing_in_ghaw", drift[0].DriftCategory)
}

// TestFormal_P6_CorrectionPRForActionableDrift covers T-DR-006: when any DriftRecord has
// drift_category of missing_in_ghaw or spec_mismatch, a corrective PR MUST be opened.
func TestFormal_P6_CorrectionPRForActionableDrift(t *testing.T) {
	assert.True(t, formalCorrectionPRForActionableDrift("missing_in_ghaw"))
	assert.True(t, formalCorrectionPRForActionableDrift("spec_mismatch"))
	assert.False(t, formalCorrectionPRForActionableDrift("missing_in_schema"))
}

// TestFormal_P7_SLARemediationWindow covers T-DR-007: SLA escalation trigger timing.
func TestFormal_P7_SLARemediationWindow(t *testing.T) {
	detected := time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC) // Monday
	deadline := formalAddBusinessDays(detected, 5)

	assert.Equal(t, time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC), deadline, "5 business days should skip weekend")
	assert.True(t, formalSLARemediationWindow(detected, deadline))
	assert.False(t, formalSLARemediationWindow(detected, deadline.Add(time.Second)))
}

// TestFormal_P8_EscalationIssueStructure covers T-DR-007: escalation issue MUST be opened
// or updated with required fields when SLA is exceeded.
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

// TestFormal_TDR008_CorrectionPREmbedsDriftRecords covers T-DR-008: the corrective PR
// description MUST embed the full DriftRecord list as JSON.
func TestFormal_TDR008_CorrectionPREmbedsDriftRecords(t *testing.T) {
	records := []FormalDriftRecord{
		{
			PropertyPath:    "apiProxy.anthropicAutoCache",
			DriftCategory:   "missing_in_ghaw",
			SuggestedAction: "Add field to implementation",
			DetectedAt:      "2026-06-08T00:00:00Z",
		},
	}
	jsonBytes, err := json.Marshal(records)
	require.NoError(t, err)

	// A PR body that embeds the DriftRecord list as JSON must parse successfully.
	prBody := "## Corrective PR\n\nDrift detected:\n\n```json\n" + string(jsonBytes) + "\n```"
	ok, parsed := formalCorrectionPREmbedsDriftRecords(prBody)
	assert.True(t, ok, "PR body must contain a parseable JSON DriftRecord list")
	assert.Len(t, parsed, 1)
	assert.Equal(t, "apiProxy.anthropicAutoCache", parsed[0].PropertyPath)

	// A PR body with no JSON array must fail.
	ok, _ = formalCorrectionPREmbedsDriftRecords("## Corrective PR\n\nNo JSON here.")
	assert.False(t, ok, "PR body without a JSON array must fail")
}

func TestFormal_P9_SafeguardDegradedModeOnUnavailability(t *testing.T) {
	state := formalSafeguardDegradedModeOnUnavailability(false, true)
	assert.True(t, state.UseSnapshot)
	assert.True(t, state.WarningEmitted)
	assert.True(t, state.DestructiveOpsSuppressed)
	assert.True(t, state.DegradedMode)
}

// TestFormal_P10_DriftReportEmittedOnDetection covers T-DR-009 (empty list is valid output
// and MUST NOT trigger corrective PR) and T-DR-010 (Step 5 integration: output is a JSON
// array of zero or more DriftRecord objects).
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

	// T-DR-010: output MUST be serializable as a JSON array.
	jsonBytes, err := json.Marshal(empty)
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(jsonBytes), "empty DriftRecord list must serialize to a JSON empty array")
}
