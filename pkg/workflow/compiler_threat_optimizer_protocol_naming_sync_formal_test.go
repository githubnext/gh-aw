//go:build !integration

package workflow

import (
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormal_ComplianceReadmeNormTestNamesStaySynced enforces the naming
// linkage between the Section 6.4/6.6 norm tables in
// specs/compiler-threat-detection-compliance/README.md (T-CTR-024..040) and
// the test function names in pkg/workflow/compiler_threat_optimizer_protocol_test.go.
//
// If a norm test in that file is renamed, removed, or its T-CTR-* ID changes
// without a corresponding update to the compliance README norm tables (or
// vice versa), this test fails. This closes the drift risk noted in the SPDD
// review: the compliance README's norm tables must stay in sync if
// pkg/workflow/compiler_threat_optimizer_protocol_test.go test names change.
func TestFormal_ComplianceReadmeNormTestNamesStaySynced(t *testing.T) {
	testIDsFromTestNames := normTestIDsFromOptimizerProtocolTestNames(t)
	testIDsFromReadme := normTestIDsFromComplianceReadme(t)

	require.NotEmpty(t, testIDsFromTestNames, "expected at least one T-CTR-* norm test function name")
	require.NotEmpty(t, testIDsFromReadme, "expected at least one T-CTR-* norm table row in the compliance README")

	for id := range testIDsFromReadme {
		assert.Truef(t, testIDsFromTestNames[id],
			"compliance README norm table references %s but no test function in "+
				"pkg/workflow/compiler_threat_optimizer_protocol_test.go covers it; "+
				"update the test name or the README table together", id)
	}

	for id := range testIDsFromTestNames {
		assert.Truef(t, testIDsFromReadme[id],
			"pkg/workflow/compiler_threat_optimizer_protocol_test.go has a test function covering %s "+
				"but the compliance README Section 6.4/6.6 norm tables do not list it; "+
				"update the README table or the test name together", id)
	}
}

// normTestIDsFromOptimizerProtocolTestNames parses
// compiler_threat_optimizer_protocol_test.go's function declarations and
// extracts every T-CTR-NNN ID encoded in a test function name, expanding
// "NNNThroughMMM" ranges (e.g. TCTR026Through028 -> T-CTR-026, T-CTR-027,
// T-CTR-028).
func normTestIDsFromOptimizerProtocolTestNames(t *testing.T) map[string]bool {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller must return a valid file path")

	sourcePath := filepath.Join(filepath.Dir(thisFile), "compiler_threat_optimizer_protocol_test.go")
	file, err := parser.ParseFile(gotoken.NewFileSet(), sourcePath, nil, parser.SkipObjectResolution)
	require.NoError(t, err, "failed to parse %s", sourcePath)

	idPattern := regexp.MustCompile(`TCTR(\d{3})(?:Through(\d{3}))?`)
	ids := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || !isTestFuncName(fn.Name.Name) {
			return true
		}
		for _, match := range idPattern.FindAllStringSubmatch(fn.Name.Name, -1) {
			start, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			end := start
			if match[2] != "" {
				end, err = strconv.Atoi(match[2])
				require.NoError(t, err)
			}
			require.LessOrEqualf(t, start, end,
				"malformed T-CTR range in test function name %q: start must not exceed end", fn.Name.Name)
			for n := start; n <= end; n++ {
				ids[formalCTRTestIDString(n)] = true
			}
		}
		return true
	})

	return ids
}

// isTestFuncName reports whether name is a top-level Go test function name
// (starts with "Test"). Function names without a TCTR-* ID simply yield no
// matches from idPattern and contribute no IDs.
func isTestFuncName(name string) bool {
	return len(name) >= 4 && name[:4] == "Test"
}

// formalCTRTestIDString formats a numeric test ID as a T-CTR-NNN string with
// zero-padding matching the README table format.
func formalCTRTestIDString(n int) string {
	return "T-CTR-" + formalZeroPad3(n)
}

func formalZeroPad3(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// normTestIDsFromComplianceReadme extracts every T-CTR-NNN ID listed in the
// Section 6.4 and Section 6.6 norm tables of
// specs/compiler-threat-detection-compliance/README.md.
func normTestIDsFromComplianceReadme(t *testing.T) map[string]bool {
	t.Helper()

	section := formalSpecSection(t, filepath.Join("compiler-threat-detection-compliance", "README.md"),
		"## Section 6.4 False-Positive Handling Norms", "The Section 6 norm tests are implemented in")

	idPattern := regexp.MustCompile(`\| (T-CTR-\d{3}) \|`)
	ids := make(map[string]bool)
	for _, match := range idPattern.FindAllStringSubmatch(section, -1) {
		ids[match[1]] = true
	}
	return ids
}

// TestFormal_ComplianceReadmeNormTestNamesStaySynced_IDsSorted is a sanity
// check that the extracted ID sets are well-formed (non-empty, zero-padded)
// so the primary sync test above cannot silently pass on an empty set.
func TestFormal_ComplianceReadmeNormTestNamesStaySynced_IDsSorted(t *testing.T) {
	ids := normTestIDsFromComplianceReadme(t)
	var sorted []string
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	require.NotEmpty(t, sorted)
	for _, id := range sorted {
		assert.Regexp(t, `^T-CTR-\d{3}$`, id)
	}
}
