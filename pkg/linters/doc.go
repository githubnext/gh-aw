// Package linters is a namespace for gh-aw's custom Go analysis linters.
//
// The actual analyzers are implemented in subpackages. All 21 active analyzers:
//
//   - contextcancelnotdeferred — flags context cancel functions called directly instead of deferred
//   - ctxbackground — flags context.Background() inside functions that already receive a context
//   - errormessage — flags non-actionable error message patterns in changed files
//   - errstringmatch — flags brittle strings.Contains(err.Error(), "...") checks
//   - excessivefuncparams — flags function declarations with too many positional parameters
//   - fileclosenotdeferred — flags file Close() calls that are not deferred
//   - fmterrorfnoverbs — flags fmt.Errorf calls with no format verbs, recommending errors.New
//   - fprintlnsprintf — flags fmt.Fprintln(..., fmt.Sprintf(...)) patterns
//   - jsonmarshalignoredeerror — flags json.Marshal/Unmarshal calls where the error is discarded with _
//   - largefunc — flags function bodies that exceed a configurable line-count threshold
//   - manualmutexunlock — flags non-deferred mutex Unlock() calls
//   - osexitinlibrary — flags os.Exit calls in library packages
//   - ossetenvlibrary — flags os.Setenv calls in library packages
//   - panic-in-library-code — flags panic() calls in library packages
//   - rawloginlib — flags direct usage of the standard log package in library packages
//   - regexpcompileinfunction — flags regexp.MustCompile/Compile calls inside functions
//   - seenmapbool — flags map[string]bool used as a set that should use map[string]struct{}
//   - ssljson — validates ssl.json skill artifacts in .github/skills/ against the SSL spec
//   - strconvparseignorederror — flags strconv parsing calls where the error is discarded with _
//   - tolowerequalfold — flags case-insensitive comparisons via ToLower/ToUpper that should use EqualFold
//   - uncheckedtypeassertion — flags unchecked single-value type assertions
//
// The package also exposes a compatibility alias (ErrorMessageAnalyzer) that
// points to the errormessage subpackage analyzer.
package linters
