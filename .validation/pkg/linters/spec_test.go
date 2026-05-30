//go:build !integration

package linters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters"
	"github.com/github/gh-aw/pkg/linters/errormessage"
	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	"github.com/github/gh-aw/pkg/linters/ssljson"
)

// TestSpec tests derive from pkg/linters/README.md. They enforce the documented
// public surface of the linters namespace (Analyzer entry points and default
// thresholds for each subpackage) without coupling to analyzer internals.

// TestSpec_PublicAPI_ExcessiveFuncParamsAnalyzer validates that the
// excessivefuncparams subpackage exposes an Analyzer entry point per the README.
func TestSpec_PublicAPI_ExcessiveFuncParamsAnalyzer(t *testing.T) {
	a := excessivefuncparams.Analyzer
	require.NotNil(t, a, "excessivefuncparams.Analyzer must be a non-nil *analysis.Analyzer")
	assert.IsType(t, (*analysis.Analyzer)(nil), a, "Analyzer should be *analysis.Analyzer for go/analysis drivers")
	assert.NotEmpty(t, a.Name, "Analyzer.Name should be set so go/analysis drivers can identify it")
	assert.NotNil(t, a.Run, "Analyzer.Run must be wired so the analyzer is executable")
}

// TestSpec_PublicAPI_LargeFuncAnalyzer validates that the largefunc subpackage
// exposes an Analyzer entry point per the README.
func TestSpec_PublicAPI_LargeFuncAnalyzer(t *testing.T) {
	a := largefunc.Analyzer
	require.NotNil(t, a, "largefunc.Analyzer must be a non-nil *analysis.Analyzer")
	assert.IsType(t, (*analysis.Analyzer)(nil), a, "Analyzer should be *analysis.Analyzer for go/analysis drivers")
	assert.NotEmpty(t, a.Name, "Analyzer.Name should be set so go/analysis drivers can identify it")
	assert.NotNil(t, a.Run, "Analyzer.Run must be wired so the analyzer is executable")
}

// TestSpec_PublicAPI_OsExitInLibraryAnalyzer validates that the osexitinlibrary
// subpackage exposes an Analyzer entry point per the README.
func TestSpec_PublicAPI_OsExitInLibraryAnalyzer(t *testing.T) {
	a := osexitinlibrary.Analyzer
	require.NotNil(t, a, "osexitinlibrary.Analyzer must be a non-nil *analysis.Analyzer")
	assert.IsType(t, (*analysis.Analyzer)(nil), a, "Analyzer should be *analysis.Analyzer for go/analysis drivers")
	assert.NotEmpty(t, a.Name, "Analyzer.Name should be set so go/analysis drivers can identify it")
	assert.NotNil(t, a.Run, "Analyzer.Run must be wired so the analyzer is executable")
}

// TestSpec_PublicAPI_ErrorMessageAnalyzer validates that the errormessage
// subpackage exposes an Analyzer entry point per the README Subpackages table.
func TestSpec_PublicAPI_ErrorMessageAnalyzer(t *testing.T) {
	a := errormessage.Analyzer
	require.NotNil(t, a, "errormessage.Analyzer must be a non-nil *analysis.Analyzer")
	assert.IsType(t, (*analysis.Analyzer)(nil), a, "Analyzer should be *analysis.Analyzer for go/analysis drivers")
	assert.NotEmpty(t, a.Name, "Analyzer.Name should be set so go/analysis drivers can identify it")
	assert.NotNil(t, a.Run, "Analyzer.Run must be wired so the analyzer is executable")
}

// TestSpec_PublicAPI_SSLJsonAnalyzer validates that the ssljson subpackage
// exposes an Analyzer entry point per the README Subpackages table.
func TestSpec_PublicAPI_SSLJsonAnalyzer(t *testing.T) {
	a := ssljson.Analyzer
	require.NotNil(t, a, "ssljson.Analyzer must be a non-nil *analysis.Analyzer")
	assert.IsType(t, (*analysis.Analyzer)(nil), a, "Analyzer should be *analysis.Analyzer for go/analysis drivers")
	assert.NotEmpty(t, a.Name, "Analyzer.Name should be set so go/analysis drivers can identify it")
	assert.NotNil(t, a.Run, "Analyzer.Run must be wired so the analyzer is executable")
}

// TestSpec_NamespaceExports_ErrorMessageAnalyzer validates the documented
// namespace-level compatibility alias `ErrorMessageAnalyzer` referenced in the
// README "Namespace exports" table.
// Spec: "ErrorMessageAnalyzer | Compatibility alias to pkg/linters/errormessage.Analyzer"
func TestSpec_NamespaceExports_ErrorMessageAnalyzer(t *testing.T) {
	require.NotNil(t, linters.ErrorMessageAnalyzer,
		"linters.ErrorMessageAnalyzer must be a non-nil compatibility alias per the README")
	assert.Same(t, errormessage.Analyzer, linters.ErrorMessageAnalyzer,
		"linters.ErrorMessageAnalyzer should be the same *analysis.Analyzer as errormessage.Analyzer")
}

// TestSpec_Constants_DefaultMaxParams validates the documented default
// "8 parameters" threshold for the excessivefuncparams analyzer.
// Spec: "excessivefuncparams ... defaults to 8 parameters (DefaultMaxParams)."
func TestSpec_Constants_DefaultMaxParams(t *testing.T) {
	assert.Equal(t, 8, excessivefuncparams.DefaultMaxParams,
		"DefaultMaxParams should match the documented default of 8")
}

// TestSpec_Constants_DefaultMaxLines validates the documented default
// "60 lines" threshold for the largefunc analyzer.
// Spec: "largefunc ... defaults to 60 lines (DefaultMaxLines)."
func TestSpec_Constants_DefaultMaxLines(t *testing.T) {
	assert.Equal(t, 60, largefunc.DefaultMaxLines,
		"DefaultMaxLines should match the documented default of 60")
}

// TestSpec_DesignDecision_MaxParamsFlag validates the documented "-max-params"
// analyzer flag for excessivefuncparams.
// Spec: "excessivefuncparams exposes a -max-params analyzer flag"
func TestSpec_DesignDecision_MaxParamsFlag(t *testing.T) {
	flag := excessivefuncparams.Analyzer.Flags.Lookup("max-params")
	require.NotNil(t, flag, "excessivefuncparams should expose a -max-params flag per the spec")
}

// TestSpec_DesignDecision_MaxLinesFlag validates the documented "-max-lines"
// analyzer flag for largefunc.
// Spec: "largefunc exposes a -max-lines analyzer flag"
func TestSpec_DesignDecision_MaxLinesFlag(t *testing.T) {
	flag := largefunc.Analyzer.Flags.Lookup("max-lines")
	require.NotNil(t, flag, "largefunc should expose a -max-lines flag per the spec")
}

// TestSpec_UsageExample_AnalyzersUsable validates the documented usage pattern:
// each Analyzer can be referenced (e.g. passed to multichecker/singlechecker).
// Spec usage example imports all five documented subpackages:
//
//	_ = excessivefuncparams.Analyzer
//	_ = errormessage.Analyzer
//	_ = largefunc.Analyzer
//	_ = osexitinlibrary.Analyzer
//	_ = ssljson.Analyzer
func TestSpec_UsageExample_AnalyzersUsable(t *testing.T) {
	analyzers := []*analysis.Analyzer{
		excessivefuncparams.Analyzer,
		errormessage.Analyzer,
		largefunc.Analyzer,
		osexitinlibrary.Analyzer,
		ssljson.Analyzer,
	}
	for _, a := range analyzers {
		assert.NotNil(t, a, "each documented Analyzer should be usable in a multichecker/singlechecker slice")
	}
}

// TestSpec_DesignDecision_UniqueAnalyzerNames validates that each documented
// subpackage exposes a distinct Analyzer.Name so they can coexist in a single
// go/analysis driver (multichecker) without conflict.
// Spec: "intentionally organized as a namespace ... so individual analyzers
// remain isolated and independently testable."
func TestSpec_DesignDecision_UniqueAnalyzerNames(t *testing.T) {
	names := map[string]bool{
		excessivefuncparams.Analyzer.Name: true,
		errormessage.Analyzer.Name:        true,
		largefunc.Analyzer.Name:           true,
		osexitinlibrary.Analyzer.Name:     true,
		ssljson.Analyzer.Name:             true,
	}
	assert.Len(t, names, 5, "each documented subpackage should expose a distinct Analyzer.Name")
}
