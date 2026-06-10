package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var labelObjectiveMappingLog = logger.New("github:label_objective_mapping")

// ObjectiveMapping defines how GitHub labels map to numeric objective values.
// This enables any label to be assigned a configurable numeric value, with one central definition place.
type ObjectiveMapping struct {
	// LabelToValue maps label names (case-insensitive) to numeric values.
	// Example: {"high-priority": 50, "copilot-opt": 50, "critical": 100, "p0": 100, "p1": 50}
	LabelToValue map[string]int `json:"label_to_value"`

	// MultiLabelLogic determines how multiple matching labels are combined:
	// "sum" = add all matching label values
	// "max" = take the highest value (default)
	// "first" = use the first match in priority order
	MultiLabelLogic string `json:"multi_label_logic"`

	// PriorityLabels defines evaluation order when logic is "first"
	// Used to establish precedence when multiple labels match
	PriorityLabels []string `json:"priority_labels,omitempty"`
}

// ComputeObjectiveValue calculates the numeric value for an issue based on its labels.
// Returns 0 if no labels match or if mapping is nil.
func (om *ObjectiveMapping) ComputeObjectiveValue(issueLabels []string) int {
	if om == nil || len(om.LabelToValue) == 0 {
		return 0
	}

	if len(issueLabels) == 0 {
		return 0
	}

	matchingValues := []int{}
	matchedLabels := []string{}

	for _, label := range issueLabels {
		normalizedLabel := strings.ToLower(strings.TrimSpace(label))
		if val, ok := om.LabelToValue[normalizedLabel]; ok {
			matchingValues = append(matchingValues, val)
			matchedLabels = append(matchedLabels, label)
		}
	}

	if len(matchingValues) == 0 {
		return 0
	}

	logic := om.MultiLabelLogic
	if logic == "" {
		logic = "max" // default
	}

	switch logic {
	case "sum":
		total := 0
		for _, v := range matchingValues {
			total += v
		}
		labelObjectiveMappingLog.Printf("Computed objective value via sum: labels=%v, value=%d", matchedLabels, total)
		return total

	case "first":
		// Return first issue label that's in priority_labels
		if len(om.PriorityLabels) > 0 {
			for _, issueLabel := range issueLabels {
				for _, priorityLabel := range om.PriorityLabels {
					if strings.EqualFold(issueLabel, priorityLabel) {
						normalizedIssue := strings.ToLower(strings.TrimSpace(issueLabel))
						if val, ok := om.LabelToValue[normalizedIssue]; ok {
							labelObjectiveMappingLog.Printf("Computed objective value via issue label priority: label=%s, value=%d", issueLabel, val)
							return val
						}
					}
				}
			}
		}
		// Fallback to first matching label
		result := matchingValues[0]
		labelObjectiveMappingLog.Printf("Computed objective value via first match: labels=%v, value=%d", matchedLabels, result)
		return result

	default: // "max"
		maxVal := matchingValues[0]
		for _, v := range matchingValues {
			if v > maxVal {
				maxVal = v
			}
		}
		labelObjectiveMappingLog.Printf("Computed objective value via max: labels=%v, value=%d", matchedLabels, maxVal)
		return maxVal
	}
}

// DefaultObjectiveMapping returns the built-in default label-to-value mapping.
func DefaultObjectiveMapping() *ObjectiveMapping {
	return &ObjectiveMapping{
		LabelToValue: map[string]int{
			"critical":        100,
			"p0":              100,
			"high-priority":   50,
			"copilot-opt":     50,
			"p1":              50,
			"security-fix":    75,
			"p2":              25,
			"medium-priority": 25,
			"performance":     30,
			"p3":              10,
			"low-priority":    10,
			"documentation":   5,
		},
		MultiLabelLogic: "max",
		PriorityLabels:  []string{"critical", "p0", "copilot-opt", "high-priority", "security-fix", "p1", "performance"},
	}
}

// LoadObjectiveMappingFromConfig loads the mapping from environment, config file, or defaults.
// Precedence:
// 1. OBJECTIVE_MAPPING_JSON environment variable
// 2. .github/objective-mapping.json file
// 3. Built-in defaults
func LoadObjectiveMappingFromConfig() *ObjectiveMapping {
	labelObjectiveMappingLog.Print("Loading objective mapping configuration")

	// Try loading from OBJECTIVE_MAPPING_JSON env var
	if mappingJSON := os.Getenv("OBJECTIVE_MAPPING_JSON"); mappingJSON != "" {
		labelObjectiveMappingLog.Print("Attempting to load from OBJECTIVE_MAPPING_JSON env var")
		var om ObjectiveMapping
		if err := json.Unmarshal([]byte(mappingJSON), &om); err == nil {
			labelObjectiveMappingLog.Printf("Loaded mapping from env var JSON: %d labels", len(om.LabelToValue))
			return &om
		} else {
			labelObjectiveMappingLog.Printf("Failed to parse OBJECTIVE_MAPPING_JSON as JSON: %v", err)
		}

		// If it's not valid JSON, treat it as a file path.
		if data, err := os.ReadFile(mappingJSON); err == nil {
			if err := json.Unmarshal(data, &om); err == nil {
				labelObjectiveMappingLog.Printf("Loaded mapping from env var file %q: %d labels", mappingJSON, len(om.LabelToValue))
				return &om
			}
			labelObjectiveMappingLog.Printf("Failed to parse OBJECTIVE_MAPPING_JSON file %q: %v", mappingJSON, err)
		} else {
			labelObjectiveMappingLog.Printf("Could not read OBJECTIVE_MAPPING_JSON as file %q: %v", mappingJSON, err)
		}
	}

	configPath := filepath.Join(".github", "objective-mapping.json")
	if data, err := os.ReadFile(configPath); err == nil {
		labelObjectiveMappingLog.Printf("Attempting to load from %s", configPath)
		var om ObjectiveMapping
		if err := json.Unmarshal(data, &om); err == nil {
			labelObjectiveMappingLog.Printf("Loaded mapping from config file: %d labels", len(om.LabelToValue))
			return &om
		}
		labelObjectiveMappingLog.Printf("Failed to parse config file: %v", err)
	}

	// Return default mapping
	defaults := DefaultObjectiveMapping()
	labelObjectiveMappingLog.Printf("Using default mapping: %d labels", len(defaults.LabelToValue))
	return defaults
}

// GetObjectiveLabels returns the subset of issue labels that have objective values.
// Also returns the labels in the order they appear in the issue's label list.
func (om *ObjectiveMapping) GetObjectiveLabels(issueLabels []string) []string {
	if om == nil || len(om.LabelToValue) == 0 {
		return []string{}
	}

	result := make([]string, 0)
	for _, label := range issueLabels {
		normalizedLabel := strings.ToLower(strings.TrimSpace(label))
		if _, ok := om.LabelToValue[normalizedLabel]; ok {
			result = append(result, label)
		}
	}

	return result
}

// MarshalJSON implements json.Marshaler to ensure consistent output.
func (om *ObjectiveMapping) MarshalJSON() ([]byte, error) {
	type Alias ObjectiveMapping
	return json.MarshalIndent(&struct {
		*Alias
	}{
		Alias: (*Alias)(om),
	}, "", "  ")
}

// String returns a human-readable summary of the mapping.
func (om *ObjectiveMapping) String() string {
	if om == nil {
		return "nil ObjectiveMapping"
	}
	return fmt.Sprintf("ObjectiveMapping{labels: %d, logic: %s, priorities: %d}",
		len(om.LabelToValue), om.MultiLabelLogic, len(om.PriorityLabels))
}

// ValidateLabelExists checks if a given label has a defined objective value.
func (om *ObjectiveMapping) ValidateLabelExists(label string) bool {
	if om == nil {
		return false
	}
	normalizedLabel := strings.ToLower(strings.TrimSpace(label))
	_, exists := om.LabelToValue[normalizedLabel]
	return exists
}

// GetAllLabels returns all labels defined in the mapping (sorted).
func (om *ObjectiveMapping) GetAllLabels() []string {
	if om == nil {
		return []string{}
	}
	var labels []string
	for label := range om.LabelToValue {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	return labels
}
