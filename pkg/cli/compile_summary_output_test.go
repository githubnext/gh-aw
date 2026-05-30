//go:build !integration

package cli

import (
"bytes"
"os"
"regexp"
"strings"
"testing"
)

var compilationSummaryTests = []struct {
name                string
stats               *CompilationStats
showAll             bool
expectedInOutput    []string
notExpectedInOutput []string
}{
{
name: "multiple failed workflows with FailureDetails",
stats: &CompilationStats{Total: 5, Errors: 4, Warnings: 1, FailureDetails: []WorkflowFailure{
{Path: ".github/workflows/test1.md", ErrorCount: 1, ErrorMessages: []string{"test1.md:5:1: error: invalid engine value 'copiliot'"}},
{Path: ".github/workflows/test2.md", ErrorCount: 2, ErrorMessages: []string{"test2.md:10:1: error: network.allowed requires strict mode"}},
{Path: ".github/workflows/test3.md", ErrorCount: 2, ErrorMessages: []string{"test3.md:3:1: error: deprecated field usage", "test3.md:4:1: error: event filter is invalid"}},
}},
expectedInOutput: []string{
"Compiled 5 workflow(s): 4 error(s) across 3 failed workflow(s), 1 warning(s)",
"Failed workflows:", "✗ test1.md", "✗ test2.md", "✗ test3.md",
"🔴 CRITICAL (fix first):", "🟠 HIGH PRIORITY:", "🟡 MEDIUM PRIORITY:", "🔵 LOW PRIORITY:",
"💡 Recovery plan:", "Use a supported engine name in frontmatter",
"Either enable strict mode for the workflow or remove the unsupported network configuration.",
},
},
{
name: "single failed workflow with FailureDetails",
stats: &CompilationStats{Total: 1, Errors: 2, FailureDetails: []WorkflowFailure{
{Path: ".github/workflows/workflow-single.md", ErrorCount: 2, ErrorMessages: []string{"workflow-single.md:1:1: error: invalid engine value 'copiliot'", "workflow-single.md:2:1: error: runtime version conflict"}},
}},
expectedInOutput: []string{
"Compiled 1 workflow(s): 2 error(s) across 1 failed workflow(s), 0 warning(s)",
"Failed workflows:", "✗ workflow-single.md",
"workflow-single.md (2 error(s)):", "invalid engine value 'copiliot'", "runtime version conflict",
},
},
{
name:    "show-all prints every prioritized error",
showAll: true,
stats: &CompilationStats{Total: 1, Errors: 6, FailureDetails: []WorkflowFailure{
{Path: ".github/workflows/workflow-all.md", ErrorCount: 6, ErrorMessages: []string{
"workflow-all.md:1:1: error: invalid engine value 'copiliot'",
"workflow-all.md:2:1: error: network.allowed requires strict mode",
"workflow-all.md:3:1: error: tools.github config invalid",
"workflow-all.md:4:1: error: runtime version conflict",
"workflow-all.md:5:1: error: event filter is invalid",
"workflow-all.md:6:1: error: deprecated field usage",
}},
}},
expectedInOutput: []string{
"workflow-all.md (6 error(s)):",
"1. workflow-all.md:1:1: error: invalid engine value 'copiliot'",
"6. workflow-all.md:6:1: error: deprecated field usage",
},
notExpectedInOutput: []string{"Run 'gh aw compile --show-all'"},
},
{
name: "backward compatibility with FailedWorkflows",
stats: &CompilationStats{Total: 3, Errors: 2, FailedWorkflows: []string{"old-workflow1.md", "old-workflow2.md"}},
expectedInOutput: []string{
"Compiled 3 workflow(s): 2 error(s) across 2 failed workflow(s), 0 warning(s)",
"Failed workflows:", "✗ old-workflow1.md", "✗ old-workflow2.md",
},
},
{
name:  "successful compilation without failures",
stats: &CompilationStats{Total: 5},
expectedInOutput:    []string{"Compiled 5 workflow(s): 0 error(s), 0 warning(s)"},
notExpectedInOutput: []string{"Failed workflows:", "✗"},
},
}

func captureCompilationSummaryOutput(t *testing.T, stats *CompilationStats, showAll bool) string {
t.Helper()
oldStderr := os.Stderr
r, w, _ := os.Pipe()
os.Stderr = w
printCompilationSummary(stats, showAll)
w.Close()
os.Stderr = oldStderr
var buf bytes.Buffer
buf.ReadFrom(r)
return buf.String()
}

func checkMultipleFailureHeadings(t *testing.T, output string) {
t.Helper()
assertHeadingContainsMessage(t, output, "🔴 CRITICAL \(fix first\)", "invalid engine value 'copiliot'")
assertHeadingContainsMessage(t, output, "🟠 HIGH PRIORITY", "network.allowed requires strict mode")
assertHeadingContainsMessage(t, output, "🟡 MEDIUM PRIORITY", "event filter is invalid")
assertHeadingContainsMessage(t, output, "🔵 LOW PRIORITY", "deprecated field usage")
}

// TestPrintCompilationSummaryWithFailedWorkflows tests that printCompilationSummary
// displays a clear list of failed workflow IDs before showing detailed error messages
func TestPrintCompilationSummaryWithFailedWorkflows(t *testing.T) {
for _, tt := range compilationSummaryTests {
t.Run(tt.name, func(t *testing.T) {
output := captureCompilationSummaryOutput(t, tt.stats, tt.showAll)
for _, expected := range tt.expectedInOutput {
if !strings.Contains(output, expected) {
t.Errorf("Expected output to contain %q, but it didn't.
Full output:
%s", expected, output)
}
}
for _, notExpected := range tt.notExpectedInOutput {
if strings.Contains(output, notExpected) {
t.Errorf("Expected output to NOT contain %q, but it did.
Full output:
%s", notExpected, output)
}
}
if tt.name == "multiple failed workflows with FailureDetails" {
checkMultipleFailureHeadings(t, output)
}
})
}
}

func assertHeadingContainsMessage(t *testing.T, output string, heading string, message string) {
t.Helper()
pattern := `(?s)` + heading + `:.*?` + regexp.QuoteMeta(message)
matched, err := regexp.MatchString(pattern, output)
if err != nil {
t.Fatalf("Failed to compile regex pattern %q: %v", pattern, err)
}
if !matched {
t.Fatalf("Expected heading %q to contain message %q.
Full output:
%s", heading, message, output)
}
}
