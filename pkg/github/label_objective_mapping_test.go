package github

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectiveMapping_ComputeObjectiveValue_Max(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical":      100,
			"high-priority": 50,
			"medium":        25,
		},
		MultiLabelLogic: "max",
	}

	tests := []struct {
		name     string
		labels   []string
		expected int
	}{
		{"no labels", []string{}, 0},
		{"no matches", []string{"unknown"}, 0},
		{"single match", []string{"high-priority"}, 50},
		{"multiple matches - max wins", []string{"medium", "high-priority"}, 50},
		{"all matches - highest wins", []string{"critical", "high-priority", "medium"}, 100},
		{"case insensitive", []string{"Critical", "HIGH-PRIORITY"}, 100},
		{"whitespace trimmed", []string{" high-priority ", "  medium  "}, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.ComputeObjectiveValue(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestObjectiveMapping_ComputeObjectiveValue_Sum(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical":      100,
			"high-priority": 50,
			"medium":        25,
		},
		MultiLabelLogic: "sum",
	}

	tests := []struct {
		name     string
		labels   []string
		expected int
	}{
		{"no labels", []string{}, 0},
		{"no matches", []string{"unknown"}, 0},
		{"single match", []string{"high-priority"}, 50},
		{"multiple matches - sum", []string{"medium", "high-priority"}, 75},
		{"all matches - sum all", []string{"critical", "high-priority", "medium"}, 175},
		{"duplicate match", []string{"high-priority", "high-priority"}, 100}, // sum counts both
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.ComputeObjectiveValue(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestObjectiveMapping_ComputeObjectiveValue_First(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical":      100,
			"high-priority": 50,
			"medium":        25,
			"low":           10,
		},
		MultiLabelLogic: "first",
		PriorityLabels:  []string{"critical", "high-priority", "medium"},
	}

	tests := []struct {
		name     string
		labels   []string
		expected int
	}{
		{"no labels", []string{}, 0},
		{"critical first in priority", []string{"low", "critical", "high-priority"}, 100},
		{"high-priority first in issue labels", []string{"high-priority", "critical"}, 50},
		{"no match in priority, fallback to first", []string{"medium"}, 25},
		{"no priority match, use any", []string{"low"}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.ComputeObjectiveValue(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestObjectiveMapping_ComputeObjectiveValue_Nil(t *testing.T) {
	var mapping *ObjectiveMapping
	assert.Equal(t, 0, mapping.ComputeObjectiveValue([]string{"any"}))
}

func TestObjectiveMapping_ComputeObjectiveValue_Empty(t *testing.T) {
	mapping := &ObjectiveMapping{}
	assert.Equal(t, 0, mapping.ComputeObjectiveValue([]string{"any"}))
}

func TestObjectiveMapping_GetObjectiveLabels(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical":      100,
			"high-priority": 50,
		},
	}

	tests := []struct {
		name     string
		labels   []string
		expected []string
	}{
		{"no labels", []string{}, []string{}},
		{"no matches", []string{"unknown", "other"}, []string{}},
		{"single match", []string{"critical", "unknown"}, []string{"critical"}},
		{"multiple matches", []string{"unknown", "critical", "other", "high-priority"}, []string{"critical", "high-priority"}},
		{"case preserved in output", []string{"CRITICAL", "High-Priority"}, []string{"CRITICAL", "High-Priority"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.GetObjectiveLabels(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultObjectiveMapping(t *testing.T) {
	mapping := DefaultObjectiveMapping()
	require.NotNil(t, mapping)
	assert.NotEmpty(t, mapping.LabelToValue)
	assert.Equal(t, "max", mapping.MultiLabelLogic)
	assert.NotEmpty(t, mapping.PriorityLabels)

	// Verify some expected labels
	assert.Equal(t, 100, mapping.LabelToValue["critical"])
	assert.Equal(t, 100, mapping.LabelToValue["p0"])
	assert.Equal(t, 50, mapping.LabelToValue["high-priority"])
	assert.Equal(t, 50, mapping.LabelToValue["copilot-opt"])
}

func TestObjectiveMapping_MarshalJSON(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical": 100,
			"high":     50,
		},
		MultiLabelLogic: "max",
		PriorityLabels:  []string{"critical"},
	}

	data, err := json.Marshal(mapping)
	require.NoError(t, err)

	var unmarshaled ObjectiveMapping
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, mapping.LabelToValue, unmarshaled.LabelToValue)
	assert.Equal(t, mapping.MultiLabelLogic, unmarshaled.MultiLabelLogic)
	assert.Equal(t, mapping.PriorityLabels, unmarshaled.PriorityLabels)
}

func TestObjectiveMapping_ValidateLabelExists(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical": 100,
			"high":     50,
		},
	}

	assert.True(t, mapping.ValidateLabelExists("critical"))
	assert.True(t, mapping.ValidateLabelExists("CRITICAL"))
	assert.True(t, mapping.ValidateLabelExists(" critical "))
	assert.False(t, mapping.ValidateLabelExists("unknown"))
	assert.False(t, (*ObjectiveMapping)(nil).ValidateLabelExists("any"))
}

func TestObjectiveMapping_GetAllLabels(t *testing.T) {
	mapping := &ObjectiveMapping{
		LabelToValue: map[string]int{
			"zebra":  10,
			"apple":  20,
			"banana": 15,
		},
	}

	labels := mapping.GetAllLabels()
	assert.Equal(t, []string{"apple", "banana", "zebra"}, labels) // sorted
}

func TestObjectiveMapping_String(t *testing.T) {
	mapping := DefaultObjectiveMapping()
	str := mapping.String()
	assert.Contains(t, str, "ObjectiveMapping")
	assert.Contains(t, str, "max")

	var nilMapping *ObjectiveMapping
	assert.Equal(t, "nil ObjectiveMapping", nilMapping.String())
}

func TestLoadObjectiveMappingFromConfig_EnvVar(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("OBJECTIVE_MAPPING_JSON")
	defer os.Setenv("OBJECTIVE_MAPPING_JSON", originalEnv)

	// Set test env var
	testMapping := `{"label_to_value": {"test-label": 42}, "multi_label_logic": "sum"}`
	os.Setenv("OBJECTIVE_MAPPING_JSON", testMapping)

	mapping := LoadObjectiveMappingFromConfig()
	require.NotNil(t, mapping)
	assert.Equal(t, 42, mapping.LabelToValue["test-label"])
	assert.Equal(t, "sum", mapping.MultiLabelLogic)
}

func TestLoadObjectiveMappingFromConfig_Default(t *testing.T) {
	// Clear env to ensure fallback to default
	originalEnv := os.Getenv("OBJECTIVE_MAPPING_JSON")
	defer os.Setenv("OBJECTIVE_MAPPING_JSON", originalEnv)
	os.Setenv("OBJECTIVE_MAPPING_JSON", "")

	mapping := LoadObjectiveMappingFromConfig()
	require.NotNil(t, mapping)
	assert.NotEmpty(t, mapping.LabelToValue)
	assert.Equal(t, "max", mapping.MultiLabelLogic)
}

func TestLoadObjectiveMappingFromConfig_GitHubPathPreferred(t *testing.T) {
	originalEnv := os.Getenv("OBJECTIVE_MAPPING_JSON")
	defer os.Setenv("OBJECTIVE_MAPPING_JSON", originalEnv)
	os.Setenv("OBJECTIVE_MAPPING_JSON", "")

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(originalWD))
	}()

	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, ".github"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".github", "objective-mapping.json"), []byte(`{"label_to_value": {"github-path": 99}, "multi_label_logic": "max"}`), 0o644))
	require.NoError(t, os.Chdir(tempDir))

	mapping := LoadObjectiveMappingFromConfig()
	require.NotNil(t, mapping)
	assert.Equal(t, 99, mapping.LabelToValue["github-path"])
	assert.Equal(t, "max", mapping.MultiLabelLogic)
}

func TestLoadObjectiveMappingFromConfig_IgnoresLegacyPath(t *testing.T) {
	originalEnv := os.Getenv("OBJECTIVE_MAPPING_JSON")
	defer os.Setenv("OBJECTIVE_MAPPING_JSON", originalEnv)
	os.Setenv("OBJECTIVE_MAPPING_JSON", "")

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(originalWD))
	}()

	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, ".gh-aw"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".gh-aw", "objective-mapping.json"), []byte(`{"label_to_value": {"legacy-path": 77}, "multi_label_logic": "sum"}`), 0o644))
	require.NoError(t, os.Chdir(tempDir))

	mapping := LoadObjectiveMappingFromConfig()
	require.NotNil(t, mapping)
	assert.NotContains(t, mapping.LabelToValue, "legacy-path")
	assert.Equal(t, "max", mapping.MultiLabelLogic)
}

func TestObjectiveMapping_RealWorldScenario(t *testing.T) {
	// Test realistic scenario from impact efficiency report
	mapping := DefaultObjectiveMapping()

	// Scenario 1: high-priority issue (like in issue #38040)
	labels1 := []string{"high-priority"}
	value1 := mapping.ComputeObjectiveValue(labels1)
	assert.Equal(t, 50, value1)

	// Scenario 2: critical security fix with multiple labels
	labels2 := []string{"security-fix", "bug", "high-priority"}
	value2 := mapping.ComputeObjectiveValue(labels2)
	assert.Equal(t, 75, value2) // max: security-fix=75

	// Scenario 3: P0 critical issue
	labels3 := []string{"p0", "critical-bug"}
	value3 := mapping.ComputeObjectiveValue(labels3)
	assert.Equal(t, 100, value3) // p0=100

	// Scenario 4: low-priority work
	labels4 := []string{"documentation", "low-priority"}
	value4 := mapping.ComputeObjectiveValue(labels4)
	assert.Equal(t, 10, value4) // max: low-priority=10

	// Scenario 5: no objective labels
	labels5 := []string{"type:bug", "component:cli"}
	value5 := mapping.ComputeObjectiveValue(labels5)
	assert.Equal(t, 0, value5)
}
