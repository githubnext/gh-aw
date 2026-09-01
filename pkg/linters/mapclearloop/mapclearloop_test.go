//go:build !integration

package mapclearloop_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/mapclearloop"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, mapclearloop.Analyzer, "mapclearloop")
}

func TestCoverageGateUsesBodyPosition(t *testing.T) {
	t.Parallel()
	if os.Getenv("MAPCLEARLOOP_COVERAGE_HELPER") == "1" {
		testdata := analysistest.TestData()
		analysistest.Run(t, testdata, mapclearloop.Analyzer, "mapclearloopcoverage")
		return
	}

	testdata := analysistest.TestData()
	source := filepath.Join(testdata, "src", "mapclearloopcoverage", "mapclearloopcoverage.go")
	profile := filepath.Join(t.TempDir(), "coverage.out")
	content := "mode: count\n" +
		source + ":5.1,5.20 1 1\n" +
		source + ":6.1,6.20 1 0\n"
	if err := os.WriteFile(profile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestCoverageGateUsesBodyPosition$")
	cmd.Env = append(os.Environ(),
		"MAPCLEARLOOP_COVERAGE_HELPER=1",
		"GH_AW_LINT_COVERAGE_PROFILE="+profile,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("coverage-gated analyzer failed: %v\n%s", err, output)
	}
}
