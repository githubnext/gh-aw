// Package linters is a namespace for gh-aw's custom Go analysis linters.
//
// The actual analyzers are implemented in subpackages. All 29 active analyzers:
//
//   - contextcancelnotdeferred — flags context cancel functions called directly instead of deferred
//   - ctxbackground — flags context.Background() inside functions that already receive a context
//   - errorfwrapv — flags fmt.Errorf calls that format error arguments with %v instead of %w
//   - errormessage — flags non-actionable error message patterns in changed files
//   - errstringmatch — flags brittle strings.Contains(err.Error(), "...") checks
//   - excessivefuncparams — flags function declarations with too many positional parameters
//   - execcommandwithoutcontext — flags exec.Command calls inside functions that already receive context.Context
//   - fileclosenotdeferred — flags file Close() calls that are not deferred
//   - fmterrorfnoverbs — flags fmt.Errorf calls with no format verbs, recommending errors.New
//   - fprintlnsprintf — flags fmt.Fprintln(..., fmt.Sprintf(...)) patterns
//   - httpnoctx — flags HTTP calls that do not accept a context.Context
//   - jsonmarshalignoredeerror — flags json.Marshal/Unmarshal calls where the error is discarded with _
//   - largefunc — flags function bodies that exceed a configurable line-count threshold
//   - lenstringzero — flags len(s) == 0 / len(s) != 0 on string values that should use s == "" / s != ""
//   - manualmutexunlock — flags non-deferred mutex Unlock() calls
//   - osexitinlibrary — flags os.Exit calls in library packages
//   - ossetenvlibrary — flags os.Setenv calls in library packages
//   - panic-in-library-code — flags panic() calls in library packages
//   - rawloginlib — flags direct usage of the standard log package in library packages
//   - regexpcompileinfunction — flags regexp.MustCompile/Compile calls inside functions
//   - seenmapbool — flags map[string]bool used as a set that should use map[string]struct{}
//   - sortslice — flags sort.Slice / sort.SliceStable calls that should use slices.SortFunc / slices.SortStableFunc
//   - ssljson — validates ssl.json skill artifacts in .github/skills/ against the SSL spec
//   - strconvparseignorederror — flags strconv parsing calls where the error is discarded with _
//   - timeafterleak — flags time.After in select cases inside loops that leak timer channels
//   - timesleepnocontext — flags time.Sleep calls in context-aware functions that should propagate cancellation
//   - tolowerequalfold — flags case-insensitive comparisons via ToLower/ToUpper that should use EqualFold
//   - uncheckedtypeassertion — flags unchecked single-value type assertions
//
// The package also exposes a compatibility alias (ErrorMessageAnalyzer) that
// points to the errormessage subpackage analyzer.
package linters
