---
private: true
emoji: "🔐"
name: PureLock
description: Daily workflow that locks down one uncovered pure Go function with a maximum-coverage, parallel-safe testify test suite
on:
  schedule: daily
  workflow_dispatch:
  skip-if-match: 'is:pr is:open in:title "[purelock]"'
permissions:
  contents: read
  actions: read
  pull-requests: read
engine:
  id: copilot
strict: true
timeout-minutes: 35
max-turns: 60
max-daily-ai-credits: 10000
network:
  allowed:
    - defaults
    - go
tools:
  cache-memory:
    retention-days: 60
    allowed-extensions: [".json"]
  bash: ["*"]
  edit:
imports:
  - shared/mcp/serena-go.md
  - shared/otlp.md
if: needs.purelock_precompute.outputs.has_candidates == 'true'
jobs:
  purelock_precompute:
    runs-on: ubuntu-latest
    needs: [activation]
    timeout-minutes: 45
    permissions:
      contents: read
      actions: read
    outputs:
      has_candidates: ${{ steps.scan.outputs.has_candidates }}
      candidate_count: ${{ steps.scan.outputs.candidate_count }}
      coverage_source: ${{ steps.coverage.outputs.coverage_source }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.1
        with:
          persist-credentials: false
      - name: Setup Go
        uses: actions/setup-go@v7.0.0
        with:
          go-version-file: go.mod
          cache: true
      - name: Collect per-function coverage
        id: coverage
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          DEFAULT_BRANCH: ${{ github.event.repository.default_branch }}
        run: |
          set -euo pipefail
          BUNDLE=/tmp/purelock
          mkdir -p "$BUNDLE" "$BUNDLE/ci-coverage"
          COVERAGE_SOURCE=none

          # Prefer coverage already computed by CI over re-running the suite.
          RUN_ID=$(gh run list --workflow ci.yml --branch "${DEFAULT_BRANCH:-main}" \
            --status success --limit 1 --json databaseId --jq '.[0].databaseId' 2>/dev/null || true)
          if [ -n "$RUN_ID" ] && gh run download "$RUN_ID" --pattern 'ci-integration-coverage-*' \
              --dir "$BUNDLE/ci-coverage" >/dev/null 2>&1; then
            PROFILES=$(find "$BUNDLE/ci-coverage" -type f -name 'coverage-integration-*.out' | sort)
            if [ -n "$PROFILES" ]; then
              {
                echo "mode: atomic"
                for profile in $PROFILES; do
                  tail -n +2 "$profile"
                done
              } > "$BUNDLE/merged.out"
              COVERAGE_SOURCE="ci-run-$RUN_ID"
            fi
          fi

          if [ "$COVERAGE_SOURCE" = "none" ]; then
            echo "No CI coverage artifacts available; computing coverage locally."
            go test ./pkg/... -count=1 -covermode=atomic -timeout=25m \
              -coverprofile="$BUNDLE/merged.out" > "$BUNDLE/go-test.log" 2>&1 || true
            if [ -s "$BUNDLE/merged.out" ]; then
              COVERAGE_SOURCE=local
            fi
          fi

          if [ -s "$BUNDLE/merged.out" ]; then
            go tool cover -func="$BUNDLE/merged.out" > "$BUNDLE/func-coverage.txt"
            tail -n 1 "$BUNDLE/func-coverage.txt"
          else
            : > "$BUNDLE/func-coverage.txt"
          fi
          echo "coverage_source=$COVERAGE_SOURCE" >> "$GITHUB_OUTPUT"
      - name: Run pure-function static analysis
        id: scan
        run: |
          set -euo pipefail
          BUNDLE=/tmp/purelock
          go run .github/scripts/purelock/purity_scan.go \
            -cover "$BUNDLE/func-coverage.txt" \
            -out "$BUNDLE/candidates.json" \
            -summary "$BUNDLE/candidates.md" \
            -limit 40 \
            -max-coverage 95 \
            ./pkg/... | tee "$BUNDLE/scan.log"
          COUNT=$(jq '.candidates | length' "$BUNDLE/candidates.json")
          echo "candidate_count=$COUNT" >> "$GITHUB_OUTPUT"
          if [ "$COUNT" -gt 0 ]; then
            echo "has_candidates=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_candidates=false" >> "$GITHUB_OUTPUT"
          fi
          cat "$BUNDLE/candidates.md" >> "$GITHUB_STEP_SUMMARY"
      - name: Upload PureLock bundle
        uses: actions/upload-artifact@v7.0.1
        with:
          name: purelock-bundle-${{ github.run_id }}
          path: |
            /tmp/purelock/candidates.json
            /tmp/purelock/candidates.md
            /tmp/purelock/func-coverage.txt
          if-no-files-found: error
          retention-days: 3
steps:
  - name: Setup Go
    uses: actions/setup-go@v7.0.0
    with:
      go-version-file: go.mod
      cache: true
  - name: Download PureLock bundle
    uses: actions/download-artifact@v8.0.1
    with:
      name: purelock-bundle-${{ github.run_id }}
      path: /tmp/gh-aw/purelock
safe-outputs:
  create-pull-request:
    title-prefix: "[purelock] "
    labels: [automation, testing, coverage]
    draft: true
    expires: 5d
    if-no-changes: ignore
    protected-files: blocked
    allowed-files:
      - "**/*_test.go"
      - "**/testdata/fuzz/**"
    max-patch-files: 4
  noop:
sandbox:
  agent:
    sudo: false
evals:
  - id: candidate_selected
    question: Did the agent select one pure function from the precomputed candidate list, skipping functions already recorded in cache memory?
  - id: coverage_measured
    question: Did the agent measure package coverage before and after adding tests and confirm the target function's coverage increased?
  - id: pr_created_or_noop
    question: Did the agent create a draft pull request containing only test files, or call noop when coverage could not be improved?
---

# PureLock 🔐

Lock down exactly **one** pure Go function per run with the smallest possible test suite that reaches the highest possible coverage.

The `purelock_precompute` job already did every expensive, deterministic step: it merged coverage profiles, type-checked `./pkg/...` with `go/packages`, ran a fixed-point side-effect analysis, and ranked the pure functions where coverage is weakest. Spend your budget writing tests, not exploring the repository.

## Precomputed inputs

- `/tmp/gh-aw/purelock/candidates.json` — ranked pure-function candidates. Each entry has `package`, `file`, `line`, `name`, `receiver`, `signature`, `complexity`, `coverage_pct`, `has_test_file`, `fuzz_friendly`, `score`, and `purity_notes`.
- `/tmp/gh-aw/purelock/candidates.md` — the same list as a compact table.
- `/tmp/gh-aw/purelock/func-coverage.txt` — `go tool cover -func` output used to rank the list.
- Coverage source for this run: `${{ needs.purelock_precompute.outputs.coverage_source }}`.

Treat `candidates.json` as the **complete** working set. Do not scan the repository for other functions.

## 1. Select one function

1. Read `/tmp/gh-aw/cache-memory/purelock/state.json` when it exists. Shape:
   `{"processed":[{"key":"pkg/x/y.go:120:FuncName","date":"YYYY-MM-DD","outcome":"pr|noop"}]}`.
2. Walk `candidates.json` in order (already sorted by score) and pick the first candidate whose `key` (`file:line:name`) is absent from `processed`, or was processed more than 60 days ago.
3. If every candidate was processed recently, call `noop` explaining that the current candidate set is exhausted, and still update cache memory.
4. Work on that single function only.

## 2. Confirm purity before writing tests

The static analysis is conservative but not infallible. Confirm the selection with Serena before trusting it:

- Read the function body and verify it has no I/O, no clock or randomness, no global reads or writes, and no mutation of its arguments.
- Use `find_referencing_symbols` to see how callers use it — real call sites are the best source of realistic inputs and edge cases.
- If the function turns out to be impure, record it in cache memory with `"outcome":"noop"`, then move to the next candidate in the list (at most three attempts per run).

## 3. Measure the baseline

Run the package suite once to establish the baseline:

```bash
go test ./<package-dir>/ -count=1 -covermode=atomic -coverprofile=/tmp/purelock/before.out
go tool cover -func=/tmp/purelock/before.out | grep '<file>:<line>:'
```

Record both the target function coverage and the package total. These two numbers are the acceptance gate.

## 4. Write a maximum-coverage suite

Optimize for **high coverage and high assertion density with the fewest possible tests**. Prefer one table-driven test over many small ones.

Requirements:

- Place tests in `<file>_test.go` next to the source when that file exists, otherwise create it.
- Use `testify`: `require` for preconditions and anything that must abort the case, `assert` for independent value checks.
- One table-driven `Test<FuncName>` with named subtests covering every branch the analyzer counted in `complexity`: happy path, each error path, boundary values, empty and zero values, and Unicode or overflow inputs where the types allow.
- Assert every observable output — all return values, error identity via `require.ErrorIs` or `require.ErrorContains`, and exact string or struct equality via `assert.Equal`.
- Call `t.Parallel()` in the top-level test **and** in every subtest; pure functions are always safe to parallelize.
- Deterministic only: no sleeps, no clock, no network, no filesystem, no shared mutable globals.
- Do not modify production code, existing tests, or unrelated files.

### Testify lint rules

- Never use `assert.True(t, a == b)` — use `assert.Equal`.
- Never use `assert.Nil` for errors — use `require.NoError` or `assert.NoError`.
- Never use `assert.Equal(t, len(x), n)` — use `assert.Len`.
- Never ignore a returned error in a test.
- Always give subtests descriptive names that state the behavior, not the input.

## 5. Escalate to fuzzing when coverage stalls

Re-measure after writing the table test:

```bash
go test ./<package-dir>/ -count=1 -covermode=atomic -coverprofile=/tmp/purelock/after.out
go tool cover -func=/tmp/purelock/after.out | grep '<file>:<line>:'
```

If the target function is still below 100% **and** the candidate has `"fuzz_friendly": true`, add a `Fuzz<FuncName>` target in the same file:

- Seed the corpus with `f.Add(...)` using the table cases plus the uncovered edge inputs.
- Assert invariants that must hold for every input (no panic, idempotence, round-trip, or an output-range property), not specific values.
- Verify with `go test ./<package-dir>/ -run '^$' -fuzz 'Fuzz<FuncName>' -fuzztime=30s` and remove the fuzz target if it cannot close the gap.
- Commit any minimized corpus files the run produces under `testdata/fuzz/`.

If the function is not fuzz friendly and coverage is still short, extend the table instead; document the residual uncovered lines in the pull request body.

## 6. Validate before shipping

All of these must pass:

1. `gofmt -l <changed files>` reports nothing.
2. `go vet ./<package-dir>/`.
3. `go test ./<package-dir>/ -race -count=1`.
4. Target function coverage strictly increased versus the baseline.
5. Package total coverage strictly increased versus the baseline.
6. `git diff --name-only` lists only `*_test.go` files and `testdata/fuzz/**`.

If any check fails, revert your edits and call `noop` with the reason.

## 7. Report and remember

On success, create one draft pull request titled `Lock down <FuncName> with a pure-function test suite`. In the body include:

- the function, file, and signature
- why it is pure, quoting the analyzer's `purity_notes` and your Serena verification
- coverage before and after, for the function and the package
- test count, subtest count, and assertion count
- whether fuzzing was needed, and any residual uncovered lines

Always — on pull request, noop, or exhausted list — write `/tmp/gh-aw/cache-memory/purelock/state.json` with the processed entry appended, deduplicated by `key`, keeping the newest date. This is what cycles the workflow through every pure function in the repository.
