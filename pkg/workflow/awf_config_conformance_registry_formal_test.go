//go:build !integration

package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type formalConformanceRegistryRow struct {
	TestID      string
	Requirement string
	TestFile    string
}

func formalConformanceRegistryBaselineRows() []formalConformanceRegistryRow {
	return []formalConformanceRegistryRow{
		{TestID: "T-DR-001", Requirement: "§3.1 — required fields", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-002", Requirement: "§3.1 — drift_category enum", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-003", Requirement: "§3.1 — detected_at format", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-004", Requirement: "§3.1 — suggested_action non-empty", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-005", Requirement: "§3.1 — no additional properties", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-006", Requirement: "§7.5.1 — corrective PR trigger", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-007", Requirement: "§7.5.1 — SLA escalation trigger", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-008", Requirement: "§7.5.1 — corrective PR embeds records", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-009", Requirement: "§7.5.1 — empty list is valid", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-010", Requirement: "§7.2 Step 5 integration", TestFile: "pkg/workflow/awf_config_drift_test.go"},
		{TestID: "T-DR-SAFE-001", Requirement: "§8 item 1 — snapshot storage and freshness", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
		{TestID: "T-DR-SAFE-002", Requirement: "§8 item 2 — retrieval warning", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
		{TestID: "T-DR-SAFE-003", Requirement: "§8 item 3 — degraded-run safety", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
		{TestID: "T-DR-SAFE-004", Requirement: "§8 item 4 — scheduled persistence", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
	}
}

func formalConformanceRegistryParseSeriesID(id string, prefix string) (int, bool) {
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	numeric := strings.TrimPrefix(id, prefix)
	if len(numeric) < 3 {
		return 0, false
	}
	for _, r := range numeric {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	value, err := strconv.Atoi(numeric)
	if err != nil {
		return 0, false
	}
	return value, true
}

func formalConformanceRegistryParsePlainID(id string) (int, bool) {
	if strings.HasPrefix(id, "T-DR-SAFE-") {
		return 0, false
	}
	return formalConformanceRegistryParseSeriesID(id, "T-DR-")
}

func formalConformanceRegistryIsWellFormedFinalID(id string) bool {
	if strings.HasPrefix(id, "T-DR-SAFE-") {
		_, ok := formalConformanceRegistryParseSeriesID(id, "T-DR-SAFE-")
		return ok
	}
	_, ok := formalConformanceRegistryParsePlainID(id)
	return ok
}

func formalConformanceRegistryNextPlainID(rows []formalConformanceRegistryRow) string {
	max := 0
	for _, row := range rows {
		value, ok := formalConformanceRegistryParsePlainID(row.TestID)
		if !ok {
			continue
		}
		if value > max {
			max = value
		}
	}
	return fmt.Sprintf("T-DR-%03d", max+1)
}

func formalConformanceRegistryHasUniqueIDs(rows []formalConformanceRegistryRow) bool {
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := seen[row.TestID]; exists {
			return false
		}
		seen[row.TestID] = struct{}{}
	}
	return true
}

func formalConformanceRegistryHasRequirementReference(row formalConformanceRegistryRow) bool {
	return strings.TrimSpace(row.Requirement) != "" && strings.Contains(row.Requirement, "§")
}

func formalConformanceRegistryHasImplementationFile(row formalConformanceRegistryRow) bool {
	return strings.HasPrefix(row.TestFile, "pkg/workflow/") && strings.HasSuffix(row.TestFile, "_test.go")
}

func formalConformanceRegistryRouteTestFile(spansDriftOutputAndSchema bool) string {
	if spansDriftOutputAndSchema {
		return "pkg/workflow/awf_config_drift_test.go"
	}
	return "pkg/workflow/awf_config_safeguards_formal_test.go"
}

func formalConformanceRegistryHasSpecCrossReference(specIDs map[string]struct{}, id string) bool {
	_, ok := specIDs[id]
	return ok
}

func formalConformanceRegistrySeriesDisjoint(id string) bool {
	plain := strings.HasPrefix(id, "T-DR-") && !strings.HasPrefix(id, "T-DR-SAFE-")
	safe := strings.HasPrefix(id, "T-DR-SAFE-")
	return plain != safe
}

func TestFormalConformanceRegistry_P1_TestIDMonotonicity(t *testing.T) {
	next := formalConformanceRegistryNextPlainID(formalConformanceRegistryBaselineRows())
	assert.Equal(t, "T-DR-011", next)

	nextValue, ok := formalConformanceRegistryParsePlainID(next)
	require.True(t, ok)
	assert.Equal(t, 11, nextValue)
}

func TestFormalConformanceRegistry_P1_EmptyRegistryStartsAtOne(t *testing.T) {
	assert.Equal(t, "T-DR-001", formalConformanceRegistryNextPlainID(nil))
}

func TestFormalConformanceRegistry_P2_TestIDNoDuplicates(t *testing.T) {
	rows := formalConformanceRegistryBaselineRows()
	assert.True(t, formalConformanceRegistryHasUniqueIDs(rows))

	rows = append(rows, formalConformanceRegistryRow{TestID: "T-DR-010", Requirement: "§x", TestFile: "pkg/workflow/awf_config_drift_test.go"})
	assert.False(t, formalConformanceRegistryHasUniqueIDs(rows))
}

func TestFormalConformanceRegistry_P3_TestIDFormatWellFormed(t *testing.T) {
	valid := []string{"T-DR-001", "T-DR-010", "T-DR-1000", "T-DR-SAFE-001", "T-DR-SAFE-1234"}
	invalid := []string{"t-dr-001", "T-DR-01", "T-DR-ABC", "T-DRSAFE-001", "T-DR-SAFE-1", "T-DR-SAFE-01", "T-DR-SAFE-ABC"}

	for _, id := range valid {
		assert.True(t, formalConformanceRegistryIsWellFormedFinalID(id), id)
	}
	for _, id := range invalid {
		assert.False(t, formalConformanceRegistryIsWellFormedFinalID(id), id)
	}
}

func TestFormalConformanceRegistry_P4_PlaceholderIDRejectedAsFinal(t *testing.T) {
	assert.False(t, formalConformanceRegistryIsWellFormedFinalID("T-DR-NNN"))
}

func TestFormalConformanceRegistry_P5_RowHasRequirementReference(t *testing.T) {
	for _, row := range formalConformanceRegistryBaselineRows() {
		assert.True(t, formalConformanceRegistryHasRequirementReference(row), row.TestID)
	}
	assert.False(t, formalConformanceRegistryHasRequirementReference(formalConformanceRegistryRow{TestID: "T-DR-011", Requirement: "required fields", TestFile: "pkg/workflow/awf_config_drift_test.go"}))
}

func TestFormalConformanceRegistry_P6_RowHasImplementationFile(t *testing.T) {
	for _, row := range formalConformanceRegistryBaselineRows() {
		assert.True(t, formalConformanceRegistryHasImplementationFile(row), row.TestID)
	}
	assert.False(t, formalConformanceRegistryHasImplementationFile(formalConformanceRegistryRow{TestID: "T-DR-011", Requirement: "§3.1", TestFile: ""}))
}

func TestFormalConformanceRegistry_P7_SafeguardRowRoutingDecision(t *testing.T) {
	assert.Equal(t, "pkg/workflow/awf_config_safeguards_formal_test.go", formalConformanceRegistryRouteTestFile(false))
	assert.Equal(t, "pkg/workflow/awf_config_drift_test.go", formalConformanceRegistryRouteTestFile(true))
}

func TestFormalConformanceRegistry_P8_SpecCrossReferenceRequired(t *testing.T) {
	specIDs := map[string]struct{}{
		"T-DR-010":      {},
		"T-DR-SAFE-004": {},
	}
	assert.True(t, formalConformanceRegistryHasSpecCrossReference(specIDs, "T-DR-010"))
	assert.False(t, formalConformanceRegistryHasSpecCrossReference(specIDs, "T-DR-011"))
}

func TestFormalConformanceRegistry_P9_DriftSeriesVsSafeguardSeriesDisjoint(t *testing.T) {
	assert.True(t, formalConformanceRegistrySeriesDisjoint("T-DR-010"))
	assert.True(t, formalConformanceRegistrySeriesDisjoint("T-DR-SAFE-004"))
	assert.False(t, formalConformanceRegistrySeriesDisjoint("T-DRX-010"))
}

func TestFormalConformanceRegistry_EdgeCase_FourDigitRollover(t *testing.T) {
	rows := []formalConformanceRegistryRow{{TestID: "T-DR-999", Requirement: "§x", TestFile: "pkg/workflow/awf_config_drift_test.go"}}
	next := formalConformanceRegistryNextPlainID(rows)
	assert.Equal(t, "T-DR-1000", next)
	assert.True(t, formalConformanceRegistryIsWellFormedFinalID(next))
}

func TestFormalConformanceRegistry_EdgeCase_SafeguardOnlyRegistryDoesNotAffectPlainSeries(t *testing.T) {
	rows := []formalConformanceRegistryRow{
		{TestID: "T-DR-SAFE-001", Requirement: "§8", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
		{TestID: "T-DR-SAFE-004", Requirement: "§8", TestFile: "pkg/workflow/awf_config_safeguards_formal_test.go"},
	}
	assert.Equal(t, "T-DR-001", formalConformanceRegistryNextPlainID(rows))
}

func TestFormalConformanceRegistry_EdgeCase_MissingImplementationFileIsInvalid(t *testing.T) {
	row := formalConformanceRegistryRow{TestID: "T-DR-011", Requirement: "§3.1", TestFile: ""}
	assert.False(t, formalConformanceRegistryHasImplementationFile(row))
}
