// Package osexitinlibrary implements a Go analysis linter that flags
// os.Exit calls in library (pkg/) packages.
package osexitinlibrary

import (
	"fmt"

	"github.com/github/gh-aw/pkg/linters/internal/librarycall"
)

// restriction bans os.Exit outside of cmd/ entry-points.
var restriction = librarycall.Restriction{
	Linter:  "osexitinlibrary",
	PkgPath: "os",
	Funcs:   []string{"Exit"},
	Message: func(funcName, pkgPath string) string {
		return fmt.Sprintf("os.%s called in library package %s; move process termination to a cmd/ entry-point", funcName, pkgPath)
	},
}

// Analyzer is the os-exit-in-library analysis pass.
var Analyzer = restriction.Analyzer("reports os.Exit calls inside library packages where they bypass deferred cleanup and prevent testing")
