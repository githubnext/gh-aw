# linters Package

The `linters` package namespace contains custom static analysis linters used by `gh-aw` quality checks.

## Overview

This package currently provides custom Go analyzers in the following subpackages:

- `contextcancelnotdeferred` — reports context cancel functions that are called directly instead of deferred.
- `ctxbackground` — reports `context.Background()` calls inside functions that already receive a `context.Context` parameter.
- `errorfwrapv` — reports `fmt.Errorf` calls that format error arguments with `%v` instead of `%w`.
- `excessivefuncparams` — reports function declarations that exceed a configurable parameter-count threshold.
- `errormessage` — reports non-actionable error-message patterns in changed files.
- `errstringmatch` — reports `strings.Contains(err.Error(), "...")` patterns and recommends `errors.Is` / `errors.As`.
- `fileclosenotdeferred` — reports non-deferred file `Close()` calls that can leak resources.
- `execcommandwithoutcontext` — reports `exec.Command(...)` calls inside functions that already receive `context.Context` and should use `exec.CommandContext(...)`.
- `fmterrorfnoverbs` — reports `fmt.Errorf` calls whose format string contains no verbs, recommending `errors.New` instead.
- `fprintlnsprintf` — reports `fmt.Fprintln(..., fmt.Sprintf(...))` patterns and recommends direct formatting calls.
- `hardcodedfilepath` — reports hard-coded file path string literals that match known path constants or should be extracted into named constants; also annotates paths that appear in log/print calls.
- `httpnoctx` — reports HTTP client and package-level HTTP calls that do not accept a `context.Context`.
- `jsonmarshalignoredeerror` — reports `json.Marshal` and `json.Unmarshal` calls where the error return is discarded with `_`.
- `largefunc` — reports function bodies that exceed a configurable line-count threshold.
- `lenstringzero` — reports `len(s) == 0` / `len(s) != 0` comparisons on string values that should use `s == ""` / `s != ""`.
- `manualmutexunlock` — reports non-deferred mutex `Unlock()` calls that can lead to deadlocks on early returns or panics.
- `osexitinlibrary` — reports `os.Exit` calls in library packages (`pkg/*`) where process termination should be delegated to `cmd/*` entry points.
- `ossetenvlibrary` — reports `os.Setenv` calls in library packages (`pkg/*`) where side effects should be isolated.
- `panic-in-library-code` — reports `panic()` calls in library packages (`pkg/*`) where errors should be returned instead.
- `rawloginlib` — reports direct usage of the standard `log` package in library packages, where `pkg/logger` should be used.
- `regexpcompileinfunction` — reports `regexp.MustCompile` / `regexp.Compile` calls inside functions that should be package-level.
- `seenmapbool` — reports `map[string]bool` used as a set (values always `true`) that should use `map[string]struct{}` instead.
- `sortslice` — reports `sort.Slice` / `sort.SliceStable` calls that should use `slices.SortFunc` / `slices.SortStableFunc`.
- `ssljson` — validates `ssl.json` skill artifacts found in `.github/skills/` against the SSL spec (enum membership, graph integrity, transition targets, entry pointer validity).
- `strconvparseignorederror` — reports `strconv` parsing calls (`Atoi`, `ParseInt`, etc.) where the error return is discarded with `_`.
- `timeafterleak` — reports `time.After` calls used as the channel-receive expression in a `select` case inside a `for` or `range` loop that leak a timer channel on each iteration when another case fires first.
- `timesleepnocontext` — reports `time.Sleep` calls inside functions that already receive a `context.Context`, where a context-aware `select` should be used instead.
- `tolowerequalfold` — reports case-insensitive string comparisons using `strings.ToLower`/`ToUpper` that should use `strings.EqualFold`.
- `uncheckedtypeassertion` — reports single-value type assertions where unchecked panics are possible.
- `internal` — shared helper packages for analyzers (file checks and `nolint` handling).

## Public API

### Subpackages

| Subpackage | Description |
|------------|-------------|
| `contextcancelnotdeferred` | Custom `go/analysis` analyzer that flags context cancel functions called directly instead of deferred |
| `ctxbackground` | Custom `go/analysis` analyzer that flags `context.Background()` calls inside functions that already receive a context parameter |
| `errorfwrapv` | Custom `go/analysis` analyzer that flags `fmt.Errorf` calls that format error arguments with `%v` instead of `%w` |
| `excessivefuncparams` | Custom `go/analysis` analyzer that flags function declarations with too many positional parameters |
| `errormessage` | Custom `go/analysis` analyzer that flags non-actionable error message patterns in changed files |
| `errstringmatch` | Custom `go/analysis` analyzer that flags brittle `strings.Contains(err.Error(), "...")` checks |
| `execcommandwithoutcontext` | Custom `go/analysis` analyzer that flags `exec.Command(...)` calls that should use `exec.CommandContext(...)` in context-receiving functions |
| `fileclosenotdeferred` | Custom `go/analysis` analyzer that flags file `Close()` calls that are not deferred immediately |
| `fmterrorfnoverbs` | Custom `go/analysis` analyzer that flags `fmt.Errorf` calls with no format verbs, recommending `errors.New` |
| `fprintlnsprintf` | Custom `go/analysis` analyzer that flags `fmt.Fprintln(..., fmt.Sprintf(...))` patterns |
| `hardcodedfilepath` | Custom `go/analysis` analyzer that flags hard-coded file path string literals that match known path constants or should be extracted as named constants; annotates paths in log/print calls |
| `httpnoctx` | Custom `go/analysis` analyzer that flags HTTP calls that do not accept a `context.Context` |
| `jsonmarshalignoredeerror` | Custom `go/analysis` analyzer that flags `json.Marshal`/`json.Unmarshal` calls where the error return is discarded with `_` |
| `largefunc` | Custom `go/analysis` analyzer that flags large functions with actionable diagnostics |
| `lenstringzero` | Custom `go/analysis` analyzer that flags `len(s) == 0` / `len(s) != 0` on string values that should use `s == ""` / `s != ""` |
| `manualmutexunlock` | Custom `go/analysis` analyzer that flags mutex `Unlock()` calls that are not deferred |
| `osexitinlibrary` | Custom `go/analysis` analyzer that flags `os.Exit` usage in library packages |
| `ossetenvlibrary` | Custom `go/analysis` analyzer that flags `os.Setenv` usage in library packages |
| `panic-in-library-code` | Custom `go/analysis` analyzer that flags `panic()` usage in library packages |
| `rawloginlib` | Custom `go/analysis` analyzer that flags standard-library `log` package calls in library packages |
| `regexpcompileinfunction` | Custom `go/analysis` analyzer that flags regexp compilation inside function bodies |
| `seenmapbool` | Custom `go/analysis` analyzer that flags `map[string]bool` used as a set that should use `map[string]struct{}` |
| `sortslice` | Custom `go/analysis` analyzer that flags `sort.Slice` / `sort.SliceStable` calls that should use `slices.SortFunc` / `slices.SortStableFunc` |
| `ssljson` | Custom `go/analysis` analyzer that validates SSL JSON skill artifacts in `.github/skills/` |
| `strconvparseignorederror` | Custom `go/analysis` analyzer that flags `strconv` parsing calls where the error return is discarded with `_` |
| `timeafterleak` | Custom `go/analysis` analyzer that flags `time.After` in `select` cases inside loops that leak a timer channel on each iteration when another case fires first |
| `timesleepnocontext` | Custom `go/analysis` analyzer that flags `time.Sleep` calls in context-aware functions |
| `tolowerequalfold` | Custom `go/analysis` analyzer that flags case-insensitive comparisons via `strings.ToLower`/`ToUpper` that should use `strings.EqualFold` |
| `uncheckedtypeassertion` | Custom `go/analysis` analyzer that flags unchecked single-value type assertions |
| `internal` | Shared helper subpackages used by analyzers (`internal/filecheck`, `internal/nolint`) |

### Namespace exports

| Symbol | Description |
|---|---|
| `ErrorMessageAnalyzer` | Compatibility alias to `pkg/linters/errormessage.Analyzer` |

## Usage Examples

```go
import (
	"github.com/github/gh-aw/pkg/linters/ctxbackground"
	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
	"github.com/github/gh-aw/pkg/linters/errormessage"
	"github.com/github/gh-aw/pkg/linters/errstringmatch"
	"github.com/github/gh-aw/pkg/linters/execcommandwithoutcontext"
	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
	"github.com/github/gh-aw/pkg/linters/hardcodedfilepath"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/lenstringzero"
	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	panicinlibrarycode "github.com/github/gh-aw/pkg/linters/panic-in-library-code"
	"github.com/github/gh-aw/pkg/linters/rawloginlib"
	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
	"github.com/github/gh-aw/pkg/linters/sortslice"
	"github.com/github/gh-aw/pkg/linters/ssljson"
)

// Use with multichecker, singlechecker, or custom go/analysis driver.
_ = ctxbackground.Analyzer
_ = excessivefuncparams.Analyzer
_ = errormessage.Analyzer
_ = errstringmatch.Analyzer
_ = execcommandwithoutcontext.Analyzer
_ = fileclosenotdeferred.Analyzer
_ = hardcodedfilepath.Analyzer
_ = largefunc.Analyzer
_ = lenstringzero.Analyzer
_ = manualmutexunlock.Analyzer
_ = osexitinlibrary.Analyzer
_ = panicinlibrarycode.Analyzer
_ = rawloginlib.Analyzer
_ = regexpcompileinfunction.Analyzer
_ = sortslice.Analyzer
_ = ssljson.Analyzer
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/linters/contextcancelnotdeferred` — context-cancel-not-deferred analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ctxbackground` — context-background analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/errormessage` — error-message analyzer subpackage (also re-exported as `ErrorMessageAnalyzer`)
- `github.com/github/gh-aw/pkg/linters/errstringmatch` — err-string-match analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/execcommandwithoutcontext` — exec-command-without-context analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/excessivefuncparams` — excessive-func-params analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fileclosenotdeferred` — file-close-not-deferred analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fmterrorfnoverbs` — fmt-errorf-no-verbs analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fprintlnsprintf` — fprintln-sprintf analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/hardcodedfilepath` — hard-coded-file-path analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror` — json-marshal-ignored-error analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/largefunc` — large-func analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/lenstringzero` — len-string-zero analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/manualmutexunlock` — manual-mutex-unlock analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/osexitinlibrary` — os-exit-in-library analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ossetenvlibrary` — os-setenv-library analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/panic-in-library-code` — panic-in-library-code analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/rawloginlib` — raw-log-in-lib analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/regexpcompileinfunction` — regexp-compile-in-function analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/seenmapbool` — seen-map-bool analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/sortslice` — sort-slice analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ssljson` — ssl-json analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/strconvparseignorederror` — strconv-parse-ignored-error analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/timeafterleak` — time-after-leak analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/tolowerequalfold` — to-lower-equal-fold analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/uncheckedtypeassertion` — unchecked-type-assertion analyzer subpackage

**Transitive / Internal helpers**:
- `github.com/github/gh-aw/pkg/linters/internal/filecheck` — shared file-path filtering helpers used by multiple analyzers
- `github.com/github/gh-aw/pkg/linters/internal/nolint` — shared `//nolint` directive parsing helpers used by multiple analyzers

**External**:
- `golang.org/x/tools/go/analysis` — analyzer framework
- `golang.org/x/tools/go/analysis/passes/inspect` — AST inspection support
- `golang.org/x/tools/go/ast/inspector` — efficient AST traversal

## Design Notes

- The package is intentionally organized as a namespace (`pkg/linters/*`) so individual analyzers remain isolated and independently testable.
- CI currently enforces the `errstringmatch`, `manualmutexunlock`, `panicinlibrarycode`, `osexitinlibrary`, and `rawloginlib` analyzers via `.github/workflows/cgo.yml`.
- `excessivefuncparams` exposes a `-max-params` analyzer flag and defaults to `8` parameters (`DefaultMaxParams`).
- `largefunc` exposes a `-max-lines` analyzer flag, defaults to `60` lines (`DefaultMaxLines`), and skips `_test.go` files.
- `osexitinlibrary` helps enforce separation between library logic and process-level termination.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
