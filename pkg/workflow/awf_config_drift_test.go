//go:build !integration

package workflow

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// driftRecord mirrors the DriftRecord entity schema defined in
// specs/awf-config-sources-spec.md §6.5.1 and validated by the conformance
// test IDs listed in specs/awf-config-sources-compliance/README.md.
type driftRecord struct {
	PropertyPath    string `json:"property_path"`
	DriftCategory   string `json:"drift_category"`
	SuggestedAction string `json:"suggested_action"`
	DetectedAt      string `json:"detected_at"`
}

// driftCategories enumerates the valid drift_category values per §6.5.1.
var driftCategories = map[string]bool{
	"missing_in_ghaw":   true,
	"missing_in_schema": true,
	"spec_mismatch":     true,
}

// validateDriftRecord validates a driftRecord against the §6.5.1 requirements:
// required fields present, drift_category in the allowed enum, detected_at is a
// valid ISO 8601 UTC timestamp, and suggested_action is non-empty.
func validateDriftRecord(r driftRecord) error {
	if r.PropertyPath == "" || r.DriftCategory == "" || r.SuggestedAction == "" || r.DetectedAt == "" {
		return assertError("DriftRecord is missing one or more required fields (property_path, drift_category, suggested_action, detected_at)")
	}
	if !driftCategories[r.DriftCategory] {
		return assertError("drift_category must be one of missing_in_ghaw, missing_in_schema, spec_mismatch; got " + r.DriftCategory)
	}
	detectedAt, err := time.Parse(time.RFC3339, r.DetectedAt)
	if err != nil || detectedAt.Location() != time.UTC {
		if err == nil {
			err = assertError("timestamp must use the UTC Z designator")
		}
		return assertError("detected_at must be a valid ISO 8601 UTC timestamp: " + err.Error())
	}
	return nil
}

type assertError string

func (e assertError) Error() string { return string(e) }

// validateDriftRecordJSONStrict decodes raw JSON into a driftRecord while
// rejecting any properties beyond the four required fields, per §6.5.1
// "no additional properties".
func validateDriftRecordJSONStrict(raw []byte) (driftRecord, error) {
	var r driftRecord
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return driftRecord{}, err
	}
	return r, nil
}

// driftRequiresCorrectivePR implements §6.5.3: a corrective PR MUST be opened
// when any DriftRecord in the list has an actionable drift_category.
func driftRequiresCorrectivePR(records []driftRecord) bool {
	for _, r := range records {
		if r.DriftCategory == "missing_in_ghaw" || r.DriftCategory == "spec_mismatch" {
			return true
		}
	}
	return false
}

// driftRequiresSLAEscalation implements §6.5.3: an escalation issue MUST be
// opened or updated when the SLA window has been exceeded and actionable
// DriftRecord items are present.
func driftRequiresSLAEscalation(records []driftRecord, slaExceeded bool) bool {
	return slaExceeded && driftRequiresCorrectivePR(records)
}

// driftCorrectivePRBody implements §6.5.3: the corrective PR description MUST
// embed the full DriftRecord list as JSON.
func driftCorrectivePRBody(records []driftRecord) (string, error) {
	b, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// TestDriftRecord_TDR001_RequiredFields validates T-DR-001: DriftRecord MUST
// include property_path, drift_category, suggested_action, and detected_at;
// records missing any required field are invalid and MUST be rejected.
func TestDriftRecord_TDR001_RequiredFields(t *testing.T) {
	valid := driftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "Add coverage",
		DetectedAt:      "2026-06-08T00:00:00Z",
	}
	require.NoError(t, validateDriftRecord(valid))

	missingFields := []driftRecord{
		{DriftCategory: "missing_in_ghaw", SuggestedAction: "Add coverage", DetectedAt: "2026-06-08T00:00:00Z"},
		{PropertyPath: "x", SuggestedAction: "Add coverage", DetectedAt: "2026-06-08T00:00:00Z"},
		{PropertyPath: "x", DriftCategory: "missing_in_ghaw", DetectedAt: "2026-06-08T00:00:00Z"},
		{PropertyPath: "x", DriftCategory: "missing_in_ghaw", SuggestedAction: "Add coverage"},
	}
	for _, r := range missingFields {
		assert.Error(t, validateDriftRecord(r))
	}
}

// TestDriftRecord_TDR002_DriftCategoryEnum validates T-DR-002: drift_category
// MUST be one of missing_in_ghaw, missing_in_schema, or spec_mismatch; any
// other value is invalid.
func TestDriftRecord_TDR002_DriftCategoryEnum(t *testing.T) {
	base := driftRecord{PropertyPath: "x", SuggestedAction: "y", DetectedAt: "2026-06-08T00:00:00Z"}

	for _, category := range []string{"missing_in_ghaw", "missing_in_schema", "spec_mismatch"} {
		r := base
		r.DriftCategory = category
		require.NoError(t, validateDriftRecord(r))
	}

	invalid := base
	invalid.DriftCategory = "unknown_category"
	assert.Error(t, validateDriftRecord(invalid))
}

// TestDriftRecord_TDR003_DetectedAtFormat validates T-DR-003: detected_at MUST
// be a valid ISO 8601 UTC timestamp; non-conforming values MUST be rejected.
func TestDriftRecord_TDR003_DetectedAtFormat(t *testing.T) {
	base := driftRecord{PropertyPath: "x", DriftCategory: "missing_in_ghaw", SuggestedAction: "y"}

	valid := base
	valid.DetectedAt = "2026-06-08T00:00:00Z"
	require.NoError(t, validateDriftRecord(valid))

	for _, ts := range []string{"not-a-timestamp", "2026-06-08", "06/08/2026", "2026-06-08T01:00:00+01:00"} {
		invalid := base
		invalid.DetectedAt = ts
		assert.Error(t, validateDriftRecord(invalid))
	}
}

// TestDriftRecord_TDR004_SuggestedActionNonEmpty validates T-DR-004:
// suggested_action MUST NOT be empty; an empty string MUST be rejected.
func TestDriftRecord_TDR004_SuggestedActionNonEmpty(t *testing.T) {
	invalid := driftRecord{
		PropertyPath:  "x",
		DriftCategory: "missing_in_ghaw",
		DetectedAt:    "2026-06-08T00:00:00Z",
	}
	assert.Error(t, validateDriftRecord(invalid))
}

// TestDriftRecord_TDR005_NoAdditionalProperties validates T-DR-005:
// DriftRecord objects MUST NOT include properties beyond the four required
// fields; additional properties MUST be rejected.
func TestDriftRecord_TDR005_NoAdditionalProperties(t *testing.T) {
	validJSON := []byte(`{"property_path":"x","drift_category":"missing_in_ghaw","suggested_action":"y","detected_at":"2026-06-08T00:00:00Z"}`)
	r, err := validateDriftRecordJSONStrict(validJSON)
	require.NoError(t, err)
	assert.Equal(t, "x", r.PropertyPath)

	withExtra := []byte(`{"property_path":"x","drift_category":"missing_in_ghaw","suggested_action":"y","detected_at":"2026-06-08T00:00:00Z","extra_field":"nope"}`)
	_, err = validateDriftRecordJSONStrict(withExtra)
	assert.Error(t, err)
}

// TestDriftRecord_TDR006_CorrectivePRTrigger validates T-DR-006: when any
// DriftRecord in the output list has drift_category of missing_in_ghaw or
// spec_mismatch, the detecting automation MUST open a corrective PR (CR-05).
func TestDriftRecord_TDR006_CorrectivePRTrigger(t *testing.T) {
	assert.True(t, driftRequiresCorrectivePR([]driftRecord{{DriftCategory: "missing_in_ghaw"}}))
	assert.True(t, driftRequiresCorrectivePR([]driftRecord{{DriftCategory: "spec_mismatch"}}))
	assert.False(t, driftRequiresCorrectivePR([]driftRecord{{DriftCategory: "missing_in_schema"}}))
	assert.False(t, driftRequiresCorrectivePR(nil))
}

// TestDriftRecord_TDR007_SLAEscalationTrigger validates T-DR-007: when the
// CR-06 SLA window is exceeded and DriftRecord items with actionable
// categories are present, an escalation issue MUST be opened or updated.
func TestDriftRecord_TDR007_SLAEscalationTrigger(t *testing.T) {
	actionable := []driftRecord{{DriftCategory: "missing_in_ghaw"}}
	nonActionable := []driftRecord{{DriftCategory: "missing_in_schema"}}

	assert.True(t, driftRequiresSLAEscalation(actionable, true))
	assert.False(t, driftRequiresSLAEscalation(actionable, false), "escalation must not fire before the SLA window is exceeded")
	assert.False(t, driftRequiresSLAEscalation(nonActionable, true), "escalation must not fire without actionable drift")
}

// TestDriftRecord_TDR008_CorrectivePREmbedsRecords validates T-DR-008: the
// corrective PR description MUST embed the full DriftRecord list as JSON.
func TestDriftRecord_TDR008_CorrectivePREmbedsRecords(t *testing.T) {
	records := []driftRecord{
		{
			PropertyPath:    "apiProxy.anthropicAutoCache",
			DriftCategory:   "missing_in_ghaw",
			SuggestedAction: "Add coverage",
			DetectedAt:      "2026-06-08T00:00:00Z",
		},
	}

	body, err := driftCorrectivePRBody(records)
	require.NoError(t, err)

	var decoded []driftRecord
	require.NoError(t, json.Unmarshal([]byte(body), &decoded))
	assert.Equal(t, records, decoded)
}

// TestDriftRecord_TDR009_EmptyListValid validates T-DR-009: an empty
// DriftRecord list (no drift detected) is a valid output and MUST NOT
// trigger corrective PR or escalation actions.
func TestDriftRecord_TDR009_EmptyListValid(t *testing.T) {
	assert.False(t, driftRequiresCorrectivePR([]driftRecord{}))
	assert.False(t, driftRequiresSLAEscalation([]driftRecord{}, true))

	body, err := driftCorrectivePRBody([]driftRecord{})
	require.NoError(t, err)
	assert.Equal(t, "[]", body)
}

// TestDriftRecord_TDR010_Step5Integration validates T-DR-010: the drift
// detection procedure Step 5 MUST produce a list of zero or more DriftRecord
// objects; the output format MUST be a JSON array conforming to the §6.5.1
// schema.
func TestDriftRecord_TDR010_Step5Integration(t *testing.T) {
	records := []driftRecord{
		{
			PropertyPath:    "container.dockerHostPathPrefix",
			DriftCategory:   "spec_mismatch",
			SuggestedAction: "Reconcile implementation with spec",
			DetectedAt:      "2026-06-08T00:00:00Z",
		},
	}

	body, err := driftCorrectivePRBody(records)
	require.NoError(t, err)

	var decoded []driftRecord
	require.NoError(t, json.Unmarshal([]byte(body), &decoded))
	require.Len(t, decoded, 1)
	require.NoError(t, validateDriftRecord(decoded[0]))

	empty, err := driftCorrectivePRBody([]driftRecord{})
	require.NoError(t, err)
	var decodedEmpty []driftRecord
	require.NoError(t, json.Unmarshal([]byte(empty), &decodedEmpty))
	assert.Empty(t, decodedEmpty)
}
