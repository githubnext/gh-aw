# ADR-43939: Use astutil.IsPkgSelector for Stdlib Package Detection in Linters

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

Five custom linters (`errstringmatch`, `fprintlnsprintf`, `osexitinlibrary`, `fileclosenotdeferred`, `contextcancelnotdeferred`) identified stdlib package calls using syntactic identifier name comparison: `ident.Name == "os"`, `ident.Name == "fmt"`, etc. This approach only inspects the AST identifier name, not the resolved import path, which produces two classes of defects:

1. **False negatives**: aliased imports (e.g., `import xos "os"`) bypass the guard entirely because `xos != "os"`, causing the linter to miss real violations.
2. **False positives**: local variables that shadow the package name (e.g., `var os fakeOS`) satisfy the string match even though `os` no longer refers to the stdlib package. Three of these linters are CI-enforced, making false positives build-blockers.

### Decision

We will replace every syntactic `ident.Name == "<pkg>"` guard in these five linters with `astutil.IsPkgSelector(pass, sel, "<pkg>")`, which resolves the selector's receiver through `pass.TypesInfo` to the actual imported package path. This requires threading `*analysis.Pass` into the affected private helper functions (`isFileOpenCall`, `isContextWithCancelCall`, `brittleErrStringFuncName`, `isFmtFunc`). Test data cases for aliased imports and local shadowing are added per linter to lock in the corrected behaviour.

### Alternatives Considered

#### Alternative 1: Retain Syntactic Matching and Document Limitations

Continue using `ident.Name == "<pkg>"` and add a comment acknowledging that aliased imports are not detected and shadowed variables may produce false positives. This was rejected because the false-positive case is a concrete build-blocker for three CI-enforced linters — documenting a known defect does not eliminate the build failures it causes. The false-negative case also undermines the purpose of the linters, which is to catch real coding mistakes regardless of import style.

#### Alternative 2: Resolve Aliases Manually via import-spec Iteration

Iterate over the file's `ast.ImportSpec` nodes to build a map from local identifier to canonical package path, then check the local identifier's canonical path. This would fix false negatives for aliased imports. It was rejected because it reimplements logic that `pass.TypesInfo` already provides correctly, adds per-file bookkeeping to every linter run function, and still requires a custom lookup at each call site — more code and more opportunity for error than delegating to `astutil.IsPkgSelector`.

### Consequences

#### Positive
- False-positive build failures caused by shadowed package names are eliminated for all three CI-enforced linters.
- Aliased stdlib imports (e.g., `import ctx "context"`) are now correctly detected and flagged.
- The fix uses the type system (`pass.TypesInfo`) as the authoritative source for package identity, consistent with how Go analysis tools are intended to work.
- New testdata files provide regression coverage for both the alias case and the shadowing case.

#### Negative
- All five affected helper functions now require `*analysis.Pass` as an additional parameter, making their signatures more verbose and coupling them to the analysis pass context.
- Callers cannot invoke the helpers outside an `analysis.Pass` run (e.g., in a standalone unit test that constructs AST nodes without a full type-checker pass), which reduces testability in isolation.

#### Neutral
- No change to the diagnostic messages or suggested fixes produced by these linters; only the package-identity predicate changes.
- The pattern established here (`astutil.IsPkgSelector` instead of `ident.Name`) can serve as a guide for any future linter that needs to identify stdlib calls.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
