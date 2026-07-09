// Package linters is a namespace for gh-aw's custom Go analysis linters.
//
// All 44 active analyzers:
//
//   - appendbytestring — flags append(b, []byte(s)...) calls where s is a string that can be simplified to append(b, s...)
//   - bytescomparestring — flags string(a) == string(b) and string(a) != string(b) comparisons where a and b are []byte values that should use bytes.Equal instead
//   - contextcancelnotdeferred — flags context cancel functions called directly instead of deferred
//   - ctxbackground — flags context.Background() inside functions that already receive a context
//   - deferinloop — flags defer statements placed directly inside for or range loop bodies
//   - errorfwrapv — flags fmt.Errorf calls that format error arguments with %v instead of %w
//   - errormessage — flags non-actionable error message patterns in changed files
//   - errortypeassertion — flags type assertions from error to concrete types and recommends errors.As
//   - errstringmatch — flags brittle strings.Contains(err.Error(), "...") checks
//   - excessivefuncparams — flags function declarations with too many positional parameters
//   - execcommandwithoutcontext — flags exec.Command calls inside functions that already receive context.Context
//   - fileclosenotdeferred — flags file Close() calls that are not deferred
//   - fmterrorfnoverbs — flags fmt.Errorf calls with no format verbs, recommending errors.New
//   - fprintlnsprintf — flags fmt.Fprintln(..., fmt.Sprintf(...)) patterns
//   - hardcodedfilepath — flags hard-coded file path string literals that match known path constants or should be extracted as named constants
//   - httpnoctx — flags HTTP calls that do not accept a context.Context
//   - httprespbodyclose — flags HTTP response bodies that are not closed
//   - httpstatuscode — flags HTTP status code anti-patterns
//   - jsonmarshalignoredeerror — flags json.Marshal/Unmarshal calls where the error is discarded with _
//   - largefunc — flags function bodies that exceed a configurable line-count threshold
//   - lenstringsplit — flags len(strings.Split(s, sep)) with a non-empty separator that should use strings.Count(s, sep)+1
//   - lenstringzero — flags len(s) == 0 / len(s) != 0 on string values that should use s == "" / s != ""
//   - manualmutexunlock — flags non-deferred mutex Unlock() calls
//   - osexitinlibrary — flags os.Exit calls in library packages
//   - osgetenvlibrary — flags os.Getenv calls in library packages
//   - ossetenvlibrary — flags os.Setenv calls in library packages
//   - panic-in-library-code — flags panic() calls in library packages
//   - rawloginlib — flags direct usage of the standard log package in library packages
//   - regexpcompileinfunction — flags regexp.MustCompile/Compile calls inside functions
//   - seenmapbool — flags map[string]bool used as a set that should use map[string]struct{}
//   - sortslice — flags sort.Slice / sort.SliceStable calls that should use slices.SortFunc / slices.SortStableFunc
//   - sprintferrdot — flags redundant .Error() calls on error values passed to fmt format functions
//   - sprintferrorsnew — flags errors.New(fmt.Sprintf(...)) calls that should use fmt.Errorf instead
//   - sprintfint — flags fmt.Sprintf calls that format integers that should use strconv.Itoa
//   - ssljson — validates ssl.json skill artifacts in .github/skills/ against the SSL spec
//   - strconvparseignorederror — flags strconv parsing calls where the error is discarded with _
//   - stringreplaceminusone — flags strings.Replace calls with n=-1 that should use strings.ReplaceAll
//   - stringsindexcontains — flags strings.Index(s, substr) comparisons that should use strings.Contains
//   - timeafterleak — flags time.After in select cases inside loops that leak timer channels
//   - timesleepnocontext — flags time.Sleep calls in context-aware functions that should propagate cancellation
//   - tolowerequalfold — flags case-insensitive comparisons via ToLower/ToUpper that should use EqualFold
//   - uncheckedtypeassertion — flags unchecked single-value type assertions
//   - wgdonenotdeferred — flags non-deferred sync.WaitGroup.Done() calls
//   - writebytestring — flags w.Write([]byte(s)) calls where s is a string that can be replaced with io.WriteString(w, s)
//
// The package also exposes a compatibility alias (ErrorMessageAnalyzer) that
// points to the errormessage subpackage analyzer.
package linters
