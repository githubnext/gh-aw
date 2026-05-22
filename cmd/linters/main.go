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

	"github.com/github/gh-aw/pkg/linters/ctxbackground"
	"github.com/github/gh-aw/pkg/linters/errormessage"
	"github.com/github/gh-aw/pkg/linters/errstringmatch"
	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	"github.com/github/gh-aw/pkg/linters/rawloginlib"
	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
	"github.com/github/gh-aw/pkg/linters/ssljson"
)

func main() {
	multichecker.Main(
		ctxbackground.Analyzer,
		errormessage.Analyzer,
		errstringmatch.Analyzer,
		excessivefuncparams.Analyzer,
		fileclosenotdeferred.Analyzer,
		largefunc.Analyzer,
		manualmutexunlock.Analyzer,
		osexitinlibrary.Analyzer,
		rawloginlib.Analyzer,
		regexpcompileinfunction.Analyzer,
		ssljson.Analyzer,
	)
}
