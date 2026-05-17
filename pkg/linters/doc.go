// Package linters is a namespace for gh-aw's custom Go analysis linters.
//
// The actual analyzers are implemented in subpackages — see the
// excessivefuncparams, largefunc, and osexitinlibrary subdirectories
// for analyzer entry points. This file exists so that the directory
// can host package-level documentation and an external test package
// that asserts the namespace's specification contract.
package linters
