# ADR-30029: In-Process Statistical Analysis for the Experiments Analyze Command

**Date**: 2026-05-04
**Status**: Draft
**Deciders**: Unknown (generated from PR #30029 diff)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw experiments analyze` command displays variant assignment counts from experiment state stored in git branches, but provided no statistical interpretation of that data. §11 of the experiments specification defines concrete requirements (R-STAT-005 through R-STAT-009) for readiness gating, balance testing, multi-variant correction, and guardrail thresholds. The CLI needed to implement these requirements without adding an external statistical runtime dependency, and had to work both locally (reading `.github/workflows/*.md` frontmatter) and remotely (fetching the same files via the GitHub API). The Go standard library's `math` package provides the building blocks (`math.Erfc`) needed to approximate chi-square p-values without an external stats library.

### Decision

We will implement A/B experiment statistical analysis as a self-contained, in-process layer within the CLI (`experiments_analyze_statistics.go`), using Go's standard `math` package with a Wilson-Hilferty normal approximation for chi-square p-values rather than importing an external statistics library. Experiment configuration (hypothesis, `analysis_type`, `min_samples`, guardrail thresholds) is loaded best-effort from workflow `.md` frontmatter — locally by scanning `.github/workflows/`, remotely via the GitHub Contents API — and the analysis falls back to safe defaults (`min_samples=20`, equal expected proportions) when configuration is absent or unparseable. The analysis produces a EXTEND / READY_FOR_ANALYSIS recommendation and is surfaced both in human-readable stderr output and in the `--json` `analyses` field.

### Alternatives Considered

#### Alternative 1: Depend on an external Go statistics library (e.g., `gonum/stat`)

`gonum/stat` provides exact chi-square CDF computation and a broad suite of statistical tests. It was not chosen because it would add a large transitive dependency to a CLI tool whose primary job is git and GitHub orchestration, not statistical computing. The Wilson-Hilferty approximation is accurate to within 1% for the chi-square values and degrees of freedom encountered in typical A/B experiments (variants rarely exceed 10, chi2 rarely exceeds 100), making a full library disproportionate to the use case.

#### Alternative 2: Defer statistical evaluation to an external tool or service

The analyze command could emit raw counts and direct users to upload them to an external analytics platform (BigQuery, R, Python) for interpretation. This was not chosen because it produces no actionable signal at the point of use — developers running `gh aw experiments analyze` need to know immediately whether a variant has reached `min_samples` and whether the assignment distribution is suspicious, not after a separate pipeline run.

#### Alternative 3: Implement only the EXTEND / READY_FOR_ANALYSIS gate, skip balance testing

Implementing only the readiness gate (R-STAT-007) would have been simpler. It was not chosen because the chi-square balance test (detecting skewed traffic assignment before outcome analysis) and Bonferroni correction (avoiding false positives in multi-variant experiments) are specified requirements (§11.1, §11.3) that address real experimental validity risks independently of the readiness gate.

### Consequences

#### Positive
- No new runtime dependencies; the CLI remains a single static binary.
- Statistical analysis is available offline and in air-gapped environments where an external analytics service is unreachable.
- The self-contained statistics layer is independently unit-testable (26 tests covering degenerate inputs, monotonicity, and JSON serialization).
- Config loading degrades gracefully: the command remains fully functional even when the workflow `.md` file is absent.

#### Negative
- The Wilson-Hilferty approximation degrades for extreme chi-square values (>100) or df=1 with extreme chi2; exact computation is not provided.
- Guardrail pass/fail evaluation is explicitly deferred (R-STAT-009) because outcome metric data is not stored in `state.json`; users see thresholds but not verdicts.
- The `workflowFileCandidates` function currently returns only one candidate (the experiment name as-is), making remote config lookup fragile when the workflow filename differs from the sanitized experiment name.

#### Neutral
- Statistical output is written to stderr (not stdout) to avoid interfering with piped or scripted use of `--json` output.
- The `analyses` field is added to `ExperimentDetails` only for the `analyze` subcommand; it is absent from `list` output.
- Config loading for remote repositories performs one GitHub API call per candidate filename and stops at the first successful match.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Statistical Computation

1. Implementations **MUST** compute the EXTEND / READY_FOR_ANALYSIS recommendation by comparing each variant's run count against `min_samples`; EXTEND **MUST** be issued when any variant is below `min_samples` (R-STAT-007, R-STAT-008).
2. Implementations **MUST** apply the default `min_samples` of 20 when the experiment configuration does not declare an explicit value.
3. Implementations **MUST** compute a chi-square goodness-of-fit balance test against expected allocation proportions (equal split or declared weights) and report χ², degrees of freedom, p-value, and a balanced/unbalanced verdict at the 0.05 significance level.
4. Implementations **MUST** apply Bonferroni correction (`α_adjusted = 0.05 / (K − 1)`) for experiments with K ≥ 3 variants and **MUST NOT** report a Bonferroni alpha for experiments with fewer than 3 variants (§11.3 / R-STAT-005).
5. Implementations **MUST NOT** perform guardrail pass/fail evaluation from `state.json` data alone; they **MUST** display declared guardrail thresholds and note that pass/fail evaluation requires per-run outcome metric data (R-STAT-009).

### Configuration Loading

1. Implementations **MUST** load experiment configuration best-effort: a failure to locate or parse the workflow `.md` file **MUST NOT** cause the analyze command to return an error; it **MUST** fall back to defaults.
2. Implementations **MUST** use local `.github/workflows/` scanning when no `--repo` override is provided, and **MUST** use the GitHub Contents API when `--repo` is specified.
3. Implementations **MUST** validate that locally resolved workflow file paths are within `.github/workflows/` before reading them, and **MUST NOT** read files outside that directory.
4. Implementations **SHOULD** apply expected proportions from declared variant `weight:` values when all observed variant names have a corresponding weight entry; otherwise they **MUST** fall back to equal proportions.

### Output

1. Implementations **MUST** include the `analyses` field in `--json` output for the `analyze` subcommand.
2. Implementations **MUST** write human-readable statistical output to stderr, not stdout, to preserve the integrity of piped `--json` output.
3. Implementations **MUST** list variants in alphabetical order in both human-readable and JSON output to ensure deterministic results.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25294596855) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
