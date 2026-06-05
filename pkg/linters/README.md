# linters Package

The `linters` package namespace contains custom static analysis linters used by `gh-aw` quality checks.

## Overview

This package currently provides custom Go analyzers in the following subpackages:

- `contextcancelnotdeferred` — reports context cancel functions that are called directly instead of deferred.
- `ctxbackground` — reports `context.Background()` calls inside functions that already receive a `context.Context` parameter.
- `excessivefuncparams` — reports function declarations that exceed a configurable parameter-count threshold.
- `errormessage` — reports non-actionable error-message patterns in changed files.
- `errstringmatch` — reports `strings.Contains(err.Error(), "...")` patterns and recommends `errors.Is` / `errors.As`.
- `fileclosenotdeferred` — reports non-deferred file `Close()` calls that can leak resources.
- `fmterrorfnoverbs` — reports `fmt.Errorf` calls whose format string contains no verbs, recommending `errors.New` instead.
- `fprintlnsprintf` — reports `fmt.Fprintln(..., fmt.Sprintf(...))` patterns and recommends direct formatting calls.
- `jsonmarshalignoredeerror` — reports `json.Marshal` and `json.Unmarshal` calls where the error return is discarded with `_`.
- `largefunc` — reports function bodies that exceed a configurable line-count threshold.
- `manualmutexunlock` — reports non-deferred mutex `Unlock()` calls that can lead to deadlocks on early returns or panics.
- `osexitinlibrary` — reports `os.Exit` calls in library packages (`pkg/*`) where process termination should be delegated to `cmd/*` entry points.
- `ossetenvlibrary` — reports `os.Setenv` calls in library packages (`pkg/*`) where side effects should be isolated.
- `panic-in-library-code` — reports `panic()` calls in library packages (`pkg/*`) where errors should be returned instead.
- `rawloginlib` — reports direct usage of the standard `log` package in library packages, where `pkg/logger` should be used.
- `regexpcompileinfunction` — reports `regexp.MustCompile` / `regexp.Compile` calls inside functions that should be package-level.
- `seenmapbool` — reports `map[string]bool` used as a set (values always `true`) that should use `map[string]struct{}` instead.
- `ssljson` — validates `ssl.json` skill artifacts found in `.github/skills/` against the SSL spec (enum membership, graph integrity, transition targets, entry pointer validity).
- `strconvparseignorederror` — reports `strconv` parsing calls (`Atoi`, `ParseInt`, etc.) where the error return is discarded with `_`.
- `tolowerequalfold` — reports case-insensitive string comparisons using `strings.ToLower`/`ToUpper` that should use `strings.EqualFold`.
- `uncheckedtypeassertion` — reports single-value type assertions where unchecked panics are possible.
- `internal` — shared helper packages for analyzers (file checks and `nolint` handling).

## Public API

### Subpackages

| Subpackage | Description |
|------------|-------------|
| `contextcancelnotdeferred` | Custom `go/analysis` analyzer that flags context cancel functions called directly instead of deferred |
| `ctxbackground` | Custom `go/analysis` analyzer that flags `context.Background()` calls inside functions that already receive a context parameter |
| `excessivefuncparams` | Custom `go/analysis` analyzer that flags function declarations with too many positional parameters |
| `errormessage` | Custom `go/analysis` analyzer that flags non-actionable error message patterns in changed files |
| `errstringmatch` | Custom `go/analysis` analyzer that flags brittle `strings.Contains(err.Error(), "...")` checks |
| `fileclosenotdeferred` | Custom `go/analysis` analyzer that flags file `Close()` calls that are not deferred immediately |
| `fmterrorfnoverbs` | Custom `go/analysis` analyzer that flags `fmt.Errorf` calls with no format verbs, recommending `errors.New` |
| `fprintlnsprintf` | Custom `go/analysis` analyzer that flags `fmt.Fprintln(..., fmt.Sprintf(...))` patterns |
| `jsonmarshalignoredeerror` | Custom `go/analysis` analyzer that flags `json.Marshal`/`json.Unmarshal` calls where the error return is discarded with `_` |
| `largefunc` | Custom `go/analysis` analyzer that flags large functions with actionable diagnostics |
| `manualmutexunlock` | Custom `go/analysis` analyzer that flags mutex `Unlock()` calls that are not deferred |
| `osexitinlibrary` | Custom `go/analysis` analyzer that flags `os.Exit` usage in library packages |
| `ossetenvlibrary` | Custom `go/analysis` analyzer that flags `os.Setenv` usage in library packages |
| `panic-in-library-code` | Custom `go/analysis` analyzer that flags `panic()` usage in library packages |
| `rawloginlib` | Custom `go/analysis` analyzer that flags standard-library `log` package calls in library packages |
| `regexpcompileinfunction` | Custom `go/analysis` analyzer that flags regexp compilation inside function bodies |
| `seenmapbool` | Custom `go/analysis` analyzer that flags `map[string]bool` used as a set that should use `map[string]struct{}` |
| `ssljson` | Custom `go/analysis` analyzer that validates SSL JSON skill artifacts in `.github/skills/` |
| `strconvparseignorederror` | Custom `go/analysis` analyzer that flags `strconv` parsing calls where the error return is discarded with `_` |
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
	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
	"github.com/github/gh-aw/pkg/linters/largefunc"
	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
	"github.com/github/gh-aw/pkg/linters/osexitinlibrary"
	panicinlibrarycode "github.com/github/gh-aw/pkg/linters/panic-in-library-code"
	"github.com/github/gh-aw/pkg/linters/rawloginlib"
	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
	"github.com/github/gh-aw/pkg/linters/ssljson"
)

// Use with multichecker, singlechecker, or custom go/analysis driver.
_ = ctxbackground.Analyzer
_ = excessivefuncparams.Analyzer
_ = errormessage.Analyzer
_ = errstringmatch.Analyzer
_ = fileclosenotdeferred.Analyzer
_ = largefunc.Analyzer
_ = manualmutexunlock.Analyzer
_ = osexitinlibrary.Analyzer
_ = panicinlibrarycode.Analyzer
_ = rawloginlib.Analyzer
_ = regexpcompileinfunction.Analyzer
_ = ssljson.Analyzer
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/linters/contextcancelnotdeferred` — context-cancel-not-deferred analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ctxbackground` — context-background analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/errormessage` — error-message analyzer subpackage (also re-exported as `ErrorMessageAnalyzer`)
- `github.com/github/gh-aw/pkg/linters/errstringmatch` — err-string-match analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/excessivefuncparams` — excessive-func-params analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fileclosenotdeferred` — file-close-not-deferred analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fmterrorfnoverbs` — fmt-errorf-no-verbs analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/fprintlnsprintf` — fprintln-sprintf analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror` — json-marshal-ignored-error analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/largefunc` — large-func analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/manualmutexunlock` — manual-mutex-unlock analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/osexitinlibrary` — os-exit-in-library analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ossetenvlibrary` — os-setenv-library analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/panic-in-library-code` — panic-in-library-code analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/rawloginlib` — raw-log-in-lib analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/regexpcompileinfunction` — regexp-compile-in-function analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/seenmapbool` — seen-map-bool analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/ssljson` — ssl-json analyzer subpackage
- `github.com/github/gh-aw/pkg/linters/strconvparseignorederror` — strconv-parse-ignored-error analyzer subpackage
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
