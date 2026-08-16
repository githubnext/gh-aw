// Package logfatallibrary implements a Go analysis linter that flags
// log.Fatal, log.Fatalf, and log.Fatalln calls in library (pkg/) packages.
// These functions call os.Exit(1) internally, which bypasses deferred cleanup
// and makes the package untestable in isolation.
package logfatallibrary

import (
	"fmt"

	"github.com/github/gh-aw/pkg/linters/internal/librarycall"
)

// restriction bans the log functions that call os.Exit(1) internally.
var restriction = librarycall.Restriction{
	Linter:  "logfatallibrary",
	PkgPath: "log",
	Funcs:   []string{"Fatal", "Fatalf", "Fatalln"},
	Message: func(funcName, pkgPath string) string {
		return fmt.Sprintf("log.%s called in library package %s; use error returns instead to avoid implicit os.Exit", funcName, pkgPath)
	},
}

// Analyzer is the log-fatal-in-library analysis pass.
var Analyzer = restriction.Analyzer("reports log.Fatal, log.Fatalf, and log.Fatalln calls inside library packages where they implicitly call os.Exit and bypass deferred cleanup")
