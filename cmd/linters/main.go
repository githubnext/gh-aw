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

func main() {
	multichecker.Main(
		contextcancelnotdeferred.Analyzer,
		ctxbackground.Analyzer,
		errormessage.Analyzer,
		fprintlnsprintf.Analyzer,
		errstringmatch.Analyzer,
		execcommandwithoutcontext.Analyzer,
		excessivefuncparams.Analyzer,
		fileclosenotdeferred.Analyzer,
		fmterrorfnoverbs.Analyzer,
		largefunc.Analyzer,
		manualmutexunlock.Analyzer,
		osexitinlibrary.Analyzer,
		ossetenvlibrary.Analyzer,
		panicinlibrarycode.Analyzer,
		rawloginlib.Analyzer,
		regexpcompileinfunction.Analyzer,
		ssljson.Analyzer,
		seenmapbool.Analyzer,
		sortslice.Analyzer,
		strconvparseignorederror.Analyzer,
		jsonmarshalignoredeerror.Analyzer,
		lenstringzero.Analyzer,
		tolowerequalfold.Analyzer,
		uncheckedtypeassertion.Analyzer,
	)
}
