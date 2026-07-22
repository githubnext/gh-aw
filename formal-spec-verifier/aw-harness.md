# Formal Notes: aw-harness.md

**Last formalized**: 2026-07-22-15-57-11
**Notation**: TLA+ / F* / Z3
**Issue**: (pending)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `SingleSessionInvariant` | createAgentSession() is called exactly once per harness invocation |
| P2 | `PromptAtomicity` | prompt.txt contents are passed as a single atomic message; never split |
| P3 | `ProviderRegistrationFirst` | Extension 1 (provider setup) must register ≥1 provider before session begins |
| P4 | `ExitCodePartition` | exit ∈ {0,1,2}; 0 iff session completes without error |
| P5 | `BudgetHardGate` | When cumTokens ≥ budget.critical%, session MUST abort; exit code 1 |
| P6 | `BudgetSoftGate` | When cumTokens ≥ budget.warn%, steer message MUST be injected |
| P7 | `DisposeAlwaysCalled` | session.dispose() is called on both success and failure paths |
| P8 | `CliProxyAlwaysOn` | cli-proxy:false in frontmatter MUST be overridden to true; warning emitted |
| P9 | `GhProxyAlwaysOn` | tools.github.mode:remote MUST be overridden to gh-proxy; warning emitted |
| P10 | `NoAmbientContext` | No AGENTS.md or skills dir auto-loaded unless in imports: |
| P11 | `UserExtensionAfterBuiltins` | User extensions registered only after all 5 built-in extensions |
| P12 | `BuiltinExtensionFatal` | Failure of built-in extension (§8.2–§8.5) → exit code 2 |
| P13 | `UserExtensionNonFatal` | Failure of user extension without extensions-required:true → warning + continue |
| P14 | `JsonlToStderr` | All JSONL events and diagnostics go to stderr, not stdout |
| P15 | `StepSummaryWritten` | When GITHUB_STEP_SUMMARY set, observability ext writes GFM summary |

## Key Invariants

- Single session: exactly one AgentSession per invocation
- Budget enforcement: soft warn at warn%, hard abort at critical%
- Extension ordering: provider-setup first, user extensions last
- Proxy features: gh-proxy and cli-proxy always active regardless of frontmatter
- No credential logging: API keys never appear in JSONL or stderr
- Repair loop bounded: max 3 consecutive repair attempts per session

## Edge Cases Identified

- No provider credentials in environment → exit code 2 (not 1)
- Pi SDK fails to load → exit code 2, not code 1
- User extension throws during init, extensions-required:true → exit code 2
- User extension throws during event handler → WARN + continue session
- Budget exhausted mid-turn → preserve completed-turn artifacts, discard in-flight
- cli-proxy:false in frontmatter → silently overridden, warning to stderr

## Notes for Future Runs

- Implementation is aspirational (as of 2026-06-21 aw_harness.cjs not found)
- Five extensions: provider-setup, cost-tracker, steering, repair, observability
- Context provenance file at /tmp/gh-aw/context-provenance.jsonl is interesting sub-spec
- Pi SDK version pinning (§10.9) could be formalized further
- Repair loop max-3 constraint (§8.6.3) is a liveness/termination property worth expanding
