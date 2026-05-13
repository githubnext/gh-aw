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

	"github.com/github/gh-aw/pkg/linters/largefunc"
)

func main() {
	multichecker.Main(
		largefunc.Analyzer,
	)
}
