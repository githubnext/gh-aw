# ADR-67636: Resolve Linter Function Aliases by Type-Object Identity With Conservative Mutability Rejection

**Date**: 2026-10-10
**Status**: Draft
**Deciders**: pelikhan (PR author); gh-aw maintainers (review pending)

---

### Context

The custom Go analyzers in `pkg/linters` match package-qualified calls such as `reflect.DeepEqual(a, b)`
through the shared helper `astutil.PackageCall`. Authors can trivially evade those analyzers by binding
the function to a local variable (`eq := reflect.DeepEqual; eq(a, b)`), so the helper was extended to
follow local function-value aliases. The previous resolution worked syntactically — it matched on the
selector identifier name and the imported package name — which is unsound in two directions: it can
mis-resolve renamed/dot imports and method values, and it reports a diagnostic for an alias identifier
even when the variable is later reassigned to an unrelated function. Because analyzers in this repository
gate CI, a false positive is more costly than a missed call, and the helper has no flow-sensitive
(SSA/control-flow) information available to decide which value an alias holds at a given call site.

### Decision

We will resolve aliases using **type-object identity** rather than syntax: `packageFunction` reads the
resolved `*types.Func` from `pass.TypesInfo.Uses`, verifies that it belongs to the imported package and
has no receiver, and returns the canonical package path and function name. Alias chains are followed
recursively through declaration initializers, with a `seen` set of `types.Object` to terminate cycles.
We will pair this with a **conservative mutability rejection**: an alias is only resolved when it is a
local variable with a declaration initializer and no reassignment (`AssignStmt`, `RangeStmt` binding) and
no address escape (`&alias`). Package-scope aliases are never resolved. Calls that occur before a later
write are also left unresolved, because the analyzer makes no flow-sensitive claims.

### Alternatives Considered

#### Alternative 1: Keep syntactic name matching and accept the false positives

The previous implementation already asserted the original issue fixture and required no new type
plumbing. It was rejected because it resolves on identifier spelling rather than the resolved symbol:
renamed imports, dot imports, method values, and shadowed identifiers can be mismatched, and any alias
that is later reassigned yields a diagnostic for a call that never reaches the target function. In a
CI-gating linter, those false positives force `nolint` noise onto unrelated code.

#### Alternative 2: Flow-sensitive analysis over SSA (`buildssa`) to resolve aliases per call site

Using `golang.org/x/tools/go/analysis/passes/buildssa` would let the analyzer resolve the exact value an
alias holds at each call, so calls before a reassignment could still be flagged precisely. It was rejected
as disproportionate for the recovered precision: it adds an SSA dependency and per-package build cost to a
helper shared by every analyzer in `pkg/linters`, and the aliasing patterns it would additionally catch
(reassigned function variables) are rare and adversarial rather than accidental. The conservative rule
gives soundness for the realistic cases at near-zero cost.

#### Alternative 3: Resolve only direct `pkg.Func` selectors and drop alias support entirely

Reverting alias support would eliminate all false-positive risk. It was rejected because the originating
issue (#66436) is specifically about alias-based evasion of `reflect.DeepEqual` detection, so dropping
support reintroduces the gap the analyzer exists to close.

### Consequences

#### Positive

- Resolution is correct under renamed imports, dot imports, parenthesized expressions, and shadowing,
  because the decision is made on the `*types.Func` object rather than on source identifiers.
- Method values are excluded by the receiver check, removing a class of spurious matches.
- No false positives from reassigned aliases: a mutable or address-escaping alias is simply unresolved.
- The rule is shared by every analyzer that calls `astutil.PackageCall`, so all of them improve at once.

#### Negative

- True positives are missed whenever an alias is reassigned anywhere in the package, even for calls that
  provably precede the write — a deliberate soundness-for-recall trade.
- Package-level function aliases are never resolved, leaving a known evasion path open.
- The mutability scan walks all files in the pass for each alias, adding analysis cost relative to a
  single-scope lookup.

#### Neutral

- Diagnostic text, CI enforcement status, and coverage gating are unchanged; only which call sites are
  reported changes.
- The alias-resolution boundaries are now documented in `pkg/linters/README.md`, making the conservative
  behaviour an explicit contract rather than an implementation detail.
- Regression fixtures were expanded (unrelated functions, shadowing, method values, alias-specific
  `nolint` suppression) to pin the new boundaries.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
