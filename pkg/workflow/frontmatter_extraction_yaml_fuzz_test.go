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
	f.Fuzz(func(t *testing.T, topA, topB, nestedA, nestedB string) {
		topAQuoted := strconv.Quote(topA)
		topBQuoted := strconv.Quote(topB)
		nestedAQuoted := strconv.Quote(nestedA)
		nestedBQuoted := strconv.Quote(nestedB)

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

		assertContains := func(expected string) {
			if !strings.Contains(result, expected) {
				t.Fatalf("expected %q in result:\n%s", expected, result)
			}
		}

		assertContains("  # labels: # Label filtering applied via job conditions")
		assertContains("# - " + topAQuoted + " # Label filtering applied via job conditions")
		assertContains("# - " + topBQuoted + " # Label filtering applied via job conditions")
		assertContains("          # - " + nestedAQuoted)
		assertContains("          # - " + nestedBQuoted)
		if strings.Contains(result, "          # - "+nestedAQuoted+" # Label filtering applied via job conditions") {
			t.Fatalf("nested labels item should not be marked as top-level label filtering:\n%s", result)
		}
		if strings.Contains(result, "          # - "+nestedBQuoted+" # Label filtering applied via job conditions") {
			t.Fatalf("nested labels item should not be marked as top-level label filtering:\n%s", result)
		}

		expectedLabelFilterAnnotations := len([]string{topAQuoted, topBQuoted}) + 1 // labels key + top-level items
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
		if got := strings.Count(result, "Label filtering applied via job conditions"); got != 0 {
			t.Fatalf("unexpected top-level label filter annotations without on.labels (got %d):\n%s", got, result)
		}
	})
}
