// Package linters is a namespace for gh-aw's custom Go analysis linters.
//
// The actual analyzers are implemented in subpackages — see the
// excessivefuncparams, errormessage, largefunc, and osexitinlibrary
// subdirectories for analyzer entry points. The package also exposes
// a compatibility alias (ErrorMessageAnalyzer) that points to the
// errormessage subpackage analyzer.
package linters
