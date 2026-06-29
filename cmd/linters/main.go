// Command linters runs the gh-aw custom analysis linters.
//
// Usage:
//
//	linters [flags] [packages]
//
// Flags common to all linters are listed by running:
//
//	linters -help
//
// Each linter may also expose its own flags, e.g.:
//
//	linters -largefunc.max-lines=80 ./...
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/github/gh-aw/pkg/linters/contextcancelnotdeferred"
	"github.com/github/gh-aw/pkg/linters/ctxbackground"
	"github.com/github/gh-aw/pkg/linters/deferinloop"
	"github.com/github/gh-aw/pkg/linters/errorfwrapv"
	"github.com/github/gh-aw/pkg/linters/errormessage"
	"github.com/github/gh-aw/pkg/linters/errortypeassertion"
	"github.com/github/gh-aw/pkg/linters/errstringmatch"
	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
	"github.com/github/gh-aw/pkg/linters/execcommandwithoutcontext"
	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
	"github.com/github/gh-aw/pkg/linters/fmterrorfnoverbs"
	"github.com/github/gh-aw/pkg/linters/fprintlnsprintf"
	"github.com/github/gh-aw/pkg/linters/hardcodedfilepath"
	"github.com/github/gh-aw/pkg/linters/httpnoctx"
	"github.com/github/gh-aw/pkg/linters/httpstatuscode"
	"github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/lenstringsplit"
	"github.com/github/gh-aw/pkg/linters/lenstringzero"
	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	"github.com/github/gh-aw/pkg/linters/osgetenvlibrary"
	"github.com/github/gh-aw/pkg/linters/ossetenvlibrary"
	panicinlibrarycode "github.com/github/gh-aw/pkg/linters/panic-in-library-code"
	"github.com/github/gh-aw/pkg/linters/rawloginlib"
	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
	"github.com/github/gh-aw/pkg/linters/seenmapbool"
	"github.com/github/gh-aw/pkg/linters/sortslice"
	"github.com/github/gh-aw/pkg/linters/sprintferrdot"
	"github.com/github/gh-aw/pkg/linters/sprintferrorsnew"
	"github.com/github/gh-aw/pkg/linters/ssljson"
	"github.com/github/gh-aw/pkg/linters/strconvparseignorederror"
	"github.com/github/gh-aw/pkg/linters/stringreplaceminusone"
	"github.com/github/gh-aw/pkg/linters/timeafterleak"
	"github.com/github/gh-aw/pkg/linters/timesleepnocontext"
	"github.com/github/gh-aw/pkg/linters/tolowerequalfold"
	"github.com/github/gh-aw/pkg/linters/uncheckedtypeassertion"
	"github.com/github/gh-aw/pkg/linters/wgdonenotdeferred"
)

func main() {
	multichecker.Main(
		contextcancelnotdeferred.Analyzer,
		ctxbackground.Analyzer,
		deferinloop.Analyzer,
		errormessage.Analyzer,
		errortypeassertion.Analyzer,
		fprintlnsprintf.Analyzer,
		errstringmatch.Analyzer,
		errorfwrapv.Analyzer,
		execcommandwithoutcontext.Analyzer,
		excessivefuncparams.Analyzer,
		fileclosenotdeferred.Analyzer,
		fmterrorfnoverbs.Analyzer,
		hardcodedfilepath.Analyzer,
		httpnoctx.Analyzer,
		httpstatuscode.Analyzer,
		largefunc.Analyzer,
		manualmutexunlock.Analyzer,
		osexitinlibrary.Analyzer,
		osgetenvlibrary.Analyzer,
		ossetenvlibrary.Analyzer,
		panicinlibrarycode.Analyzer,
		rawloginlib.Analyzer,
		regexpcompileinfunction.Analyzer,
		ssljson.Analyzer,
		seenmapbool.Analyzer,
		sortslice.Analyzer,
		sprintferrdot.Analyzer,
		sprintferrorsnew.Analyzer,
		strconvparseignorederror.Analyzer,
		stringreplaceminusone.Analyzer,
		jsonmarshalignoredeerror.Analyzer,
		lenstringzero.Analyzer,
		lenstringsplit.Analyzer,
		timeafterleak.Analyzer,
		timesleepnocontext.Analyzer,
		tolowerequalfold.Analyzer,
		uncheckedtypeassertion.Analyzer,
		wgdonenotdeferred.Analyzer,
	)
}
