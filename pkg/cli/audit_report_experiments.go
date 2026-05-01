// This file provides command-line interface functionality for gh-aw.
// This file (audit_report_experiments.go) parses the experiment artifact uploaded by the
// activation job and exposes the A/B experiment assignment data for display in the
// audit and logs commands.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// ExperimentData represents the A/B experiment assignments for a single workflow run.
type ExperimentData struct {
	// Assignments maps each experiment name to the variant selected for this run.
	// e.g. {"caveman": "yes", "style": "concise"}
	Assignments map[string]string `json:"assignments"`

	// CumulativeCounts maps each experiment name to a per-variant invocation counter.
	// e.g. {"caveman": {"yes": 3, "no": 2}}
	CumulativeCounts map[string]map[string]int `json:"cumulative_counts,omitempty"`
}

// experimentStateJSON matches the shape of the state.json written by pick_experiment.cjs:
// { "counts": { "<name>": { "<variant>": <count> } } }
type experimentStateJSON struct {
	Counts map[string]map[string]int `json:"counts"`
}

// findExperimentStatePath returns the first existing state.json path inside the experiment
// artifact directory. The file may be flattened to the run root or nested inside the
// artifact subdirectory.
func findExperimentStatePath(logsPath string) string {
	candidates := []string{
		filepath.Join(logsPath, "state.json"),
		filepath.Join(logsPath, constants.ExperimentArtifactName, "state.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// extractExperimentData reads state.json from the experiment artifact directory under
// logsPath and returns a populated ExperimentData or nil when no experiment artifact
// is present.
//
// The state.json only contains the cumulative counters; it does not record which variant
// was chosen for *this* run. The selected variant is derived by applying the same
// least-used selection rule (lowest count wins; ties broken by the sorted variant order).
func extractExperimentData(logsPath string) *ExperimentData {
	if logsPath == "" {
		return nil
	}

	statePath := findExperimentStatePath(logsPath)
	if statePath == "" {
		return nil
	}

	raw, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}

	var state experimentStateJSON
	if err := json.Unmarshal(raw, &state); err != nil || len(state.Counts) == 0 {
		return nil
	}

	// Derive this-run assignments: the variant selected on the most-recent run is
	// the one with the maximum count (ties resolved by sorted order).
	assignments := make(map[string]string, len(state.Counts))
	names := make([]string, 0, len(state.Counts))
	for name := range state.Counts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		variantCounts := state.Counts[name]
		selected := deriveLastSelectedVariant(variantCounts)
		assignments[name] = selected
	}

	return &ExperimentData{
		Assignments:      assignments,
		CumulativeCounts: state.Counts,
	}
}

// formatExperimentLabel returns a compact, human-readable label summarising the
// experiment assignments for a single run. It is used in the Overview section of
// the audit report to surface experiment context alongside the run header.
//
// Examples:
//
//	one experiment:  "style=concise"
//	two experiments: "caveman=yes, style=concise"
//	nil/empty:       ""
func formatExperimentLabel(exp *ExperimentData) string {
	if exp == nil || len(exp.Assignments) == 0 {
		return ""
	}

	names := make([]string, 0, len(exp.Assignments))
	for name := range exp.Assignments {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+exp.Assignments[name])
	}
	return strings.Join(parts, ", ")
}

// experimentMatchesFilter reports whether exp satisfies the given experiment/variant
// filter pair. Rules:
//   - If experimentName is empty, every run passes (no filter active).
//   - If experimentName is set but exp is nil or lacks that experiment, the run fails.
//   - If variant is also set, the assigned variant must equal variant.
func experimentMatchesFilter(exp *ExperimentData, experimentName, variant string) bool {
	if experimentName == "" {
		return true
	}
	if exp == nil {
		return false
	}
	assigned, ok := exp.Assignments[experimentName]
	if !ok {
		return false
	}
	if variant != "" && assigned != variant {
		return false
	}
	return true
}

// formatExperimentSkipMessage returns the informational message emitted when a run
// is skipped because its experiment data does not satisfy the active filter.
func formatExperimentSkipMessage(runID int64, experimentName, variant string) string {
	if variant != "" {
		return fmt.Sprintf("Run %d skipped: experiment %q not assigned variant %q", runID, experimentName, variant)
	}
	return fmt.Sprintf("Run %d skipped: experiment %q not assigned (not found in run artifacts)", runID, experimentName)
}

// deriveLastSelectedVariant returns the variant selected on the last run based on the
// highest count. Ties are broken by sorted order.
func deriveLastSelectedVariant(variantCounts map[string]int) string {
	if len(variantCounts) == 0 {
		return ""
	}

	variants := make([]string, 0, len(variantCounts))
	for v := range variantCounts {
		variants = append(variants, v)
	}
	sort.Strings(variants)

	selected := variants[0]
	maxCount := variantCounts[selected]
	for _, v := range variants[1:] {
		if variantCounts[v] > maxCount {
			maxCount = variantCounts[v]
			selected = v
		}
	}
	return selected
}
