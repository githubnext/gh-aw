//go:build !integration

package linters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters"
	"github.com/github/gh-aw/pkg/linters/contextcancelnotdeferred"
	"github.com/github/gh-aw/pkg/linters/ctxbackground"
	"github.com/github/gh-aw/pkg/linters/errormessage"
	"github.com/github/gh-aw/pkg/linters/errstringmatch"
	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
	"github.com/github/gh-aw/pkg/linters/execcommandwithoutcontext"
	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
	"github.com/github/gh-aw/pkg/linters/fmterrorfnoverbs"
	"github.com/github/gh-aw/pkg/linters/fprintlnsprintf"
	"github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/lenstringzero"
	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	"github.com/github/gh-aw/pkg/linters/ossetenvlibrary"
	panicinlibrarycode "github.com/github/gh-aw/pkg/linters/panic-in-library-code"
	"github.com/github/gh-aw/pkg/linters/rawloginlib"
	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
	"github.com/github/gh-aw/pkg/linters/seenmapbool"
	"github.com/github/gh-aw/pkg/linters/sortslice"
	"github.com/github/gh-aw/pkg/linters/ssljson"
	"github.com/github/gh-aw/pkg/linters/strconvparseignorederror"
	"github.com/github/gh-aw/pkg/linters/tolowerequalfold"
	"github.com/github/gh-aw/pkg/linters/uncheckedtypeassertion"
)

// TestSpec tests derive from pkg/linters/README.md. They enforce the documented
// public surface of the linters namespace (the Analyzer entry point exposed by
// each documented subpackage and the documented default thresholds) without
// coupling to analyzer internals.

// docAnalyzer pairs a README "Subpackages" table label with the Analyzer value
// that subpackage is documented to expose.
type docAnalyzer struct {
	label    string
	analyzer *analysis.Analyzer
}

// documentedAnalyzers returns the analyzer subpackages documented in the README
// "Public API > Subpackages" table. The README documents 24 analyzer
// subpackages (the non-analyzer `internal` helper subpackage is excluded because
// it exposes no Analyzer).
//
// Spec (README "Public API > Subpackages"):
//
//	contextcancelnotdeferred, ctxbackground, excessivefuncparams, errormessage,
//	errstringmatch, execcommandwithoutcontext, fileclosenotdeferred, fmterrorfnoverbs, fprintlnsprintf,
//	jsonmarshalignoredeerror, largefunc, lenstringzero, manualmutexunlock,
//	osexitinlibrary, ossetenvlibrary, panic-in-library-code, rawloginlib,
//	regexpcompileinfunction, seenmapbool, sortslice, ssljson,
//	strconvparseignorederror, tolowerequalfold, uncheckedtypeassertion
func documentedAnalyzers() []docAnalyzer {
	return []docAnalyzer{
		{"contextcancelnotdeferred", contextcancelnotdeferred.Analyzer},
		{"ctxbackground", ctxbackground.Analyzer},
		{"excessivefuncparams", excessivefuncparams.Analyzer},
		{"errormessage", errormessage.Analyzer},
		{"errstringmatch", errstringmatch.Analyzer},
		{"execcommandwithoutcontext", execcommandwithoutcontext.Analyzer},
		{"fileclosenotdeferred", fileclosenotdeferred.Analyzer},
		{"fmterrorfnoverbs", fmterrorfnoverbs.Analyzer},
		{"fprintlnsprintf", fprintlnsprintf.Analyzer},
		{"jsonmarshalignoredeerror", jsonmarshalignoredeerror.Analyzer},
		{"largefunc", largefunc.Analyzer},
		{"lenstringzero", lenstringzero.Analyzer},
		{"manualmutexunlock", manualmutexunlock.Analyzer},
		{"osexitinlibrary", osexitinlibrary.Analyzer},
		{"ossetenvlibrary", ossetenvlibrary.Analyzer},
		{"panic-in-library-code", panicinlibrarycode.Analyzer},
		{"rawloginlib", rawloginlib.Analyzer},
		{"regexpcompileinfunction", regexpcompileinfunction.Analyzer},
		{"seenmapbool", seenmapbool.Analyzer},
		{"sortslice", sortslice.Analyzer},
		{"ssljson", ssljson.Analyzer},
		{"strconvparseignorederror", strconvparseignorederror.Analyzer},
		{"tolowerequalfold", tolowerequalfold.Analyzer},
		{"uncheckedtypeassertion", uncheckedtypeassertion.Analyzer},
	}
}

// TestSpec_PublicAPI_SubpackageAnalyzers validates that every analyzer
// subpackage documented in the README "Subpackages" table exposes a non-nil
// `Analyzer` entry point of type *analysis.Analyzer with its Name and Run wired,
// so each can be consumed by a go/analysis driver (multichecker/singlechecker).
func TestSpec_PublicAPI_SubpackageAnalyzers(t *testing.T) {
	for _, d := range documentedAnalyzers() {
		t.Run(d.label, func(t *testing.T) {
			require.NotNil(t, d.analyzer, "%s must expose a non-nil *analysis.Analyzer per the README Subpackages table", d.label)
			assert.IsType(t, (*analysis.Analyzer)(nil), d.analyzer, "%s.Analyzer should be *analysis.Analyzer for go/analysis drivers", d.label)
			assert.NotEmpty(t, d.analyzer.Name, "%s.Analyzer.Name should be set so go/analysis drivers can identify it", d.label)
			assert.NotNil(t, d.analyzer.Run, "%s.Analyzer.Run must be wired so the analyzer is executable", d.label)
		})
	}
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
// each documented Analyzer can be referenced (e.g. passed to a
// multichecker/singlechecker slice). The README "Usage Examples" block assigns
// `_ = <subpackage>.Analyzer` for the documented analyzers; this test exercises
// the same pattern across all documented subpackages.
func TestSpec_UsageExample_AnalyzersUsable(t *testing.T) {
	for _, d := range documentedAnalyzers() {
		assert.NotNil(t, d.analyzer, "documented Analyzer %q should be usable in a multichecker/singlechecker slice", d.label)
	}
}

// TestSpec_DesignDecision_UniqueAnalyzerNames validates that each documented
// subpackage exposes a distinct Analyzer.Name so they can coexist in a single
// go/analysis driver (multichecker) without conflict.
// Spec: "intentionally organized as a namespace ... so individual analyzers
// remain isolated and independently testable."
func TestSpec_DesignDecision_UniqueAnalyzerNames(t *testing.T) {
	documented := documentedAnalyzers()
	names := make(map[string]bool, len(documented))
	for _, d := range documented {
		names[d.analyzer.Name] = true
	}
	assert.Len(t, names, len(documented),
		"each documented subpackage should expose a distinct Analyzer.Name")
}
