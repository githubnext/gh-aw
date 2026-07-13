//go:build !integration

package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func FuzzCommentOutProcessedFieldsInOnSectionTopLevelLabels(f *testing.F) {
	f.Add("bug", "enhancement", "triage", "needs-info")
	f.Add("panel-review", "can't-repro", "nested-1", "nested-2")
	f.Add("", "", "", "")

	compiler := NewCompiler()
	f.Fuzz(func(t *testing.T, topLevelLabelA, topLevelLabelB, nestedLabelA, nestedLabelB string) {
		topAQuoted := strconv.Quote(topLevelLabelA)
		topBQuoted := strconv.Quote(topLevelLabelB)
		nestedAQuoted := strconv.Quote(nestedLabelA)
		nestedBQuoted := strconv.Quote(nestedLabelB)

		yamlStr := fmt.Sprintf(`on:
  issues:
    types: [labeled]
  labels:
    - %s
    - %s
  steps:
    - name: Nested labels in step input
      uses: actions/github-script@v8
      with:
        labels:
          - %s
          - %s
        script: |
          core.info('label')
`, topAQuoted, topBQuoted, nestedAQuoted, nestedBQuoted)

		result := compiler.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})

		mustContain := func(expected string) {
			if !strings.Contains(result, expected) {
				t.Fatalf("expected %q in result:\n%s", expected, result)
			}
		}

		// Commented-out blocks in the on: section are flattened to the block's base
		// indentation (indent 2 here) so yamllint's comments-indentation rule does
		// not fire on the deeper-nested step content.
		mustContain("  # labels: # Label filtering applied via job conditions")
		mustContain("# - " + topAQuoted + " # Label filtering applied via job conditions")
		mustContain("# - " + topBQuoted + " # Label filtering applied via job conditions")
		mustContain("  # - " + nestedAQuoted)
		mustContain("  # - " + nestedBQuoted)
		// Because commented blocks are flattened to indent 2, nested step-label items
		// are no longer distinguishable from top-level label items by indentation.
		// The annotation-count invariant below is what guarantees the nested items are
		// not mislabeled as top-level label filtering (any such mislabel would push the
		// count above the expected value).

		expectedTopLevelLabelItems := 2
		expectedLabelFilterAnnotations := expectedTopLevelLabelItems + 1 // labels key + top-level items
		if got := strings.Count(result, "Label filtering applied via job conditions"); got != expectedLabelFilterAnnotations {
			t.Fatalf("expected %d label-filter annotations (labels key + top-level items), got %d:\n%s", expectedLabelFilterAnnotations, got, result)
		}
	})
}

func FuzzCommentOutProcessedFieldsInOnSectionNoTopLevelLabels(f *testing.F) {
	f.Add("triage", "needs-info")
	f.Add("", "")

	compiler := NewCompiler()
	f.Fuzz(func(t *testing.T, nestedA, nestedB string) {
		nestedAQuoted := strconv.Quote(nestedA)
		nestedBQuoted := strconv.Quote(nestedB)

		yamlStr := fmt.Sprintf(`on:
  issues:
    types: [labeled]
  steps:
    - name: Nested labels in step input
      uses: actions/github-script@v8
      with:
        labels:
          - %s
          - %s
        script: |
          core.info('label')
`, nestedAQuoted, nestedBQuoted)

		result := compiler.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})

		// Commented on.steps content is flattened to the block base indentation (2).
		if !strings.Contains(result, "  # - "+nestedAQuoted) {
			t.Fatalf("expected nested labels item to remain in on.steps output:\n%s", result)
		}
		if !strings.Contains(result, "  # - "+nestedBQuoted) {
			t.Fatalf("expected nested labels item to remain in on.steps output:\n%s", result)
		}
		if strings.Contains(result, "Label filtering applied via job conditions") {
			t.Fatalf("unexpected top-level label filter annotation without on.labels:\n%s", result)
		}
	})
}
