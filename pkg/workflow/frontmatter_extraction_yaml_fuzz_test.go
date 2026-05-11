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

		mustContain("  # labels: # Label filtering applied via job conditions")
		mustContain("# - " + topAQuoted + " # Label filtering applied via job conditions")
		mustContain("# - " + topBQuoted + " # Label filtering applied via job conditions")
		mustContain("          # - " + nestedAQuoted)
		mustContain("          # - " + nestedBQuoted)
		if strings.Contains(result, "          # - "+nestedAQuoted+" # Label filtering applied via job conditions") {
			t.Fatalf("nested labels item should not be marked as top-level label filtering:\n%s", result)
		}
		if strings.Contains(result, "          # - "+nestedBQuoted+" # Label filtering applied via job conditions") {
			t.Fatalf("nested labels item should not be marked as top-level label filtering:\n%s", result)
		}

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

		if !strings.Contains(result, "          # - "+nestedAQuoted) {
			t.Fatalf("expected nested labels item to remain in on.steps output:\n%s", result)
		}
		if !strings.Contains(result, "          # - "+nestedBQuoted) {
			t.Fatalf("expected nested labels item to remain in on.steps output:\n%s", result)
		}
		if strings.Contains(result, "Label filtering applied via job conditions") {
			t.Fatalf("unexpected top-level label filter annotation without on.labels:\n%s", result)
		}
	})
}
