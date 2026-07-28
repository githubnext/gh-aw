//go:build !integration

package awfconfig

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DriftRecord struct {
	PropertyPath    string `json:"property_path"`
	DriftCategory   string `json:"drift_category"`
	SuggestedAction string `json:"suggested_action"`
	DetectedAt      string `json:"detected_at"`
}

type EscalationIssue struct {
	Owner string
}

type SafeguardResult struct {
	DestructiveActionsSkipped bool
	DegradedMode              bool
}

func dualSourceConsultation(consultedSources []string) bool {
	hasSpec := slices.Contains(consultedSources, "docs/awf-config-spec.md")
	hasSchema := slices.Contains(consultedSources, "docs/awf-config.schema.json")
	return hasSpec && hasSchema
}

func noUndocumentedFields(generatedFields []string, schemaAndSpecFields map[string]struct{}) bool {
	for _, field := range generatedFields {
		if _, ok := schemaAndSpecFields[field]; !ok {
			return false
		}
	}
	return true
}

func shouldRunDriftDetection(schemaChanged, scheduledRun, explicitRequest bool) bool {
	return schemaChanged || scheduledRun || explicitRequest
}

func isValidDriftCategory(category string) bool {
	return slices.Contains([]string{"missing_in_ghaw", "missing_in_schema", "spec_mismatch"}, category)
}

func hasRequiredDriftRecordFields(record DriftRecord) bool {
	return record.PropertyPath != "" &&
		record.DriftCategory != "" &&
		record.SuggestedAction != "" &&
		record.DetectedAt != ""
}

func requiresCorrectionPR(category string) bool {
	return category == "missing_in_ghaw" || category == "spec_mismatch"
}

func addBusinessDays(start time.Time, days int) time.Time {
	current := start.UTC()
	added := 0
	for added < days {
		current = current.AddDate(0, 0, 1)
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			continue
		}
		added++
	}
	return current
}

func withinSLARemediationWindow(detectedAt, now time.Time) bool {
	deadline := addBusinessDays(detectedAt, 5)
	return !now.After(deadline)
}

func requiresEscalationIssue(detectedAt, now time.Time) bool {
	slaDeadline := addBusinessDays(detectedAt, 5)
	escalationDeadline := slaDeadline.AddDate(0, 0, 1)
	return now.After(escalationDeadline)
}

func hasEscalationOwner(issue EscalationIssue) bool {
	return issue.Owner != ""
}

func degradedModeOnSourceFailure(canonicalSourcesAvailable bool) SafeguardResult {
	if canonicalSourcesAvailable {
		return SafeguardResult{}
	}
	return SafeguardResult{
		DestructiveActionsSkipped: true,
		DegradedMode:              true,
	}
}

func schemaPropertyCoverage(schemaProperties, ghawReferences []string) bool {
	refs := map[string]struct{}{}
	for _, ref := range ghawReferences {
		refs[ref] = struct{}{}
	}

	for _, property := range schemaProperties {
		if _, ok := refs[property]; !ok {
			return false
		}
	}
	return true
}

func isValidPropertyPathDotNotation(path string) bool {
	if path == "" || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return false
	}
	return !strings.Contains(path, "..")
}

func hasNoAdditionalProperties(serialized []byte) bool {
	var payload map[string]any
	if err := json.Unmarshal(serialized, &payload); err != nil {
		return false
	}

	allowed := map[string]struct{}{
		"property_path":    {},
		"drift_category":   {},
		"suggested_action": {},
		"detected_at":      {},
	}

	for key := range payload {
		if _, ok := allowed[key]; !ok {
			return false
		}
	}
	return len(payload) == len(allowed)
}

func TestDualSourceConsultation(t *testing.T) {
	assert.True(t, dualSourceConsultation([]string{"docs/awf-config-spec.md", "docs/awf-config.schema.json"}))
	assert.False(t, dualSourceConsultation([]string{"docs/awf-config-spec.md"}))
}

func TestNoUndocumentedFieldsGenerated(t *testing.T) {
	documented := map[string]struct{}{
		"apiProxy.anthropicAutoCache":    {},
		"container.dockerHostPathPrefix": {},
	}

	assert.True(t, noUndocumentedFields([]string{"apiProxy.anthropicAutoCache"}, documented))
	assert.False(t, noUndocumentedFields([]string{"apiProxy.unknown"}, documented))
}

func TestDriftTriggerConditions(t *testing.T) {
	assert.True(t, shouldRunDriftDetection(true, false, false))
	assert.True(t, shouldRunDriftDetection(false, true, false))
	assert.True(t, shouldRunDriftDetection(false, false, true))
	assert.False(t, shouldRunDriftDetection(false, false, false))
}

func TestDriftCategoryValues(t *testing.T) {
	assert.True(t, isValidDriftCategory("missing_in_ghaw"))
	assert.True(t, isValidDriftCategory("missing_in_schema"))
	assert.True(t, isValidDriftCategory("spec_mismatch"))
	assert.False(t, isValidDriftCategory("unknown"))
}

func TestDriftRecordRequiredFields(t *testing.T) {
	valid := DriftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "Add coverage",
		DetectedAt:      "2026-07-27T16:00:00Z",
	}
	invalid := DriftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "",
		DetectedAt:      "2026-07-27T16:00:00Z",
	}

	assert.True(t, hasRequiredDriftRecordFields(valid))
	assert.False(t, hasRequiredDriftRecordFields(invalid))
}

func TestCorrectionPRRequired(t *testing.T) {
	assert.True(t, requiresCorrectionPR("missing_in_ghaw"))
	assert.True(t, requiresCorrectionPR("spec_mismatch"))
	assert.False(t, requiresCorrectionPR("missing_in_schema"))
}

func TestSLARemediationWindow(t *testing.T) {
	detectedAt := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC) // Monday
	deadline := addBusinessDays(detectedAt, 5)

	assert.Equal(t, time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), deadline)
	assert.True(t, withinSLARemediationWindow(detectedAt, deadline))
	assert.False(t, withinSLARemediationWindow(detectedAt, deadline.Add(time.Second)))
}

func TestEscalationIssueOnSLABreach(t *testing.T) {
	detectedAt := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)
	slaDeadline := addBusinessDays(detectedAt, 5)

	assert.False(t, requiresEscalationIssue(detectedAt, slaDeadline))
	assert.True(t, requiresEscalationIssue(detectedAt, slaDeadline.AddDate(0, 0, 1).Add(time.Second)))
}

func TestEscalationOwnerAssigned(t *testing.T) {
	assert.True(t, hasEscalationOwner(EscalationIssue{Owner: "@maintainer"}))
	assert.False(t, hasEscalationOwner(EscalationIssue{Owner: ""}))
}

func TestDegradedModeOnCanonicalSourceFailure(t *testing.T) {
	failed := degradedModeOnSourceFailure(false)
	ok := degradedModeOnSourceFailure(true)

	assert.True(t, failed.DestructiveActionsSkipped)
	assert.True(t, failed.DegradedMode)
	assert.False(t, ok.DestructiveActionsSkipped)
	assert.False(t, ok.DegradedMode)
}

func TestSchemaPropertyFullCoverage(t *testing.T) {
	schemaProperties := []string{"apiProxy.anthropicAutoCache", "container.dockerHostPathPrefix"}
	completeRefs := []string{"apiProxy.anthropicAutoCache", "container.dockerHostPathPrefix"}
	incompleteRefs := []string{"container.dockerHostPathPrefix"}

	assert.True(t, schemaPropertyCoverage(schemaProperties, completeRefs))
	assert.False(t, schemaPropertyCoverage(schemaProperties, incompleteRefs))
}

func TestDriftRecordPropertyPathFormat(t *testing.T) {
	assert.True(t, isValidPropertyPathDotNotation("apiProxy.anthropicAutoCache"))
	assert.True(t, isValidPropertyPathDotNotation("container.dockerHostPathPrefix"))
	assert.False(t, isValidPropertyPathDotNotation(""))
	assert.False(t, isValidPropertyPathDotNotation(".apiProxy.anthropicAutoCache"))
	assert.False(t, isValidPropertyPathDotNotation("apiProxy.anthropicAutoCache."))
	assert.False(t, isValidPropertyPathDotNotation("apiProxy..anthropicAutoCache"))
}

func TestDriftRecordNoExtraFields(t *testing.T) {
	record := DriftRecord{
		PropertyPath:    "apiProxy.anthropicAutoCache",
		DriftCategory:   "missing_in_ghaw",
		SuggestedAction: "Add coverage",
		DetectedAt:      "2026-07-27T16:00:00Z",
	}
	serialized, err := json.Marshal(record)
	require.NoError(t, err)
	assert.True(t, hasNoAdditionalProperties(serialized))

	extra := []byte(`{"property_path":"apiProxy.anthropicAutoCache","drift_category":"missing_in_ghaw","suggested_action":"Add coverage","detected_at":"2026-07-27T16:00:00Z","extra":true}`)
	assert.False(t, hasNoAdditionalProperties(extra))
}
