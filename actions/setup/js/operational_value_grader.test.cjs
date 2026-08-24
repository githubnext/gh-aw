// @ts-check

const { executeOperationalValueEvaluator, buildRunSubject } = require("./operational_value_grader.cjs");

const TEST_ENV = {
  PATH: process.env.PATH,
  HOME: process.env.HOME,
  TMPDIR: process.env.TMPDIR,
  GITHUB_RUN_ID: "12345",
  GITHUB_RUN_ATTEMPT: "2",
  GITHUB_REPOSITORY: "github/gh-aw",
  GITHUB_WORKFLOW: "Example",
  GITHUB_REF: "refs/heads/main",
  GITHUB_SHA: "0123456789abcdef",
  GITHUB_EVENT_NAME: "schedule",
};

function operationalValueEvaluator(output, baseline = { mode: "baseline-comparable", value: 0.25 }) {
  return `#!/usr/bin/env bash
set -euo pipefail
case \${1:-} in
--definition)
cat <<'DEFINITION'
${JSON.stringify({ schemaVersion: 4, grader: "operational-value", baseline })}
DEFINITION
;;
--grade-run)
cat >/dev/null
cat <<'RESULT'
${JSON.stringify(output)}
RESULT
;;
*) exit 1 ;;
esac
`;
}

describe("operational_value_grader", () => {
  it("builds a stable workflow-run subject", () => {
    expect(buildRunSubject(TEST_ENV)).toEqual({
      id: "12345",
      attempt: 2,
      repository: "github/gh-aw",
      workflow: "Example",
      ref: "refs/heads/main",
      sha: "0123456789abcdef",
      eventName: "schedule",
      createdAt: null,
    });
  });

  it("returns absolute value with a secondary baseline delta", () => {
    const output = executeOperationalValueEvaluator(
      operationalValueEvaluator({
        value: 0.75,
        opportunityKey: "schedule:2026-08-23",
        case: { key: "schedule:2026-08-23" },
        evidenceCutoff: "2026-08-23T12:00:00Z",
        maturesAt: "2026-08-30T12:00:00Z",
        provenance: [{ repository: "github/gh-aw", kind: "git-commit", ref: "abc123" }],
      }),
      { digest: "abc" },
      { evidenceAt: "2026-08-24T12:00:00Z", env: TEST_ENV }
    );

    expect(output.value).toBe(0.75);
    expect(output.baselineValue).toBe(0.25);
    expect(output.deltaFromBaseline).toBe(0.5);
    expect(output.observation.subject).toEqual({
      type: "workflow-run",
      runId: "12345",
      attempt: 2,
      repository: "github/gh-aw",
      workflow: "Example",
      ref: "refs/heads/main",
      sha: "0123456789abcdef",
      eventName: "schedule",
      createdAt: null,
    });
    expect(output.observation.mature).toBe(false);
  });

  it("caps evidence at maturation and marks mature observations", () => {
    const output = executeOperationalValueEvaluator(
      operationalValueEvaluator(
        {
          value: 1,
          opportunityKey: "issue:42",
          case: { issue: 42 },
          evidenceCutoff: "2026-08-30T12:00:00Z",
          maturesAt: "2026-08-30T12:00:00Z",
          provenance: [{ repository: "github/gh-aw", kind: "issue", ref: "42" }],
        },
        { mode: "attainment-only", value: null }
      ),
      {},
      { evidenceAt: "2026-09-01T12:00:00Z", env: TEST_ENV }
    );

    expect(output.observation.mature).toBe(true);
    expect(output.observation.evidenceCutoff).toBe("2026-08-30T12:00:00Z");
    expect(output.baselineValue).toBeNull();
    expect(output.deltaFromBaseline).toBeNull();
  });

  it("rejects invalid values and uncapped evidence", () => {
    expect(() =>
      executeOperationalValueEvaluator(
        operationalValueEvaluator({
          value: 2,
          opportunityKey: "issue:42",
          case: { issue: 42 },
          evidenceCutoff: "2026-09-01T12:00:00Z",
          maturesAt: "2026-08-30T12:00:00Z",
          provenance: [],
        }),
        {},
        { evidenceAt: "2026-09-01T12:00:00Z", env: TEST_ENV }
      )
    ).toThrow("result.value must be null or a finite number in [0,1]");
  });

  it("rejects invalid Bash", () => {
    expect(() => executeOperationalValueEvaluator("#!/usr/bin/env bash\nif", {}, { evidenceAt: "2026-08-24T12:00:00Z", env: TEST_ENV })).toThrow("invalid Bash syntax");
  });

  it("rejects an invalid frozen baseline", () => {
    expect(() =>
      executeOperationalValueEvaluator(
        operationalValueEvaluator(
          {
            value: 1,
            opportunityKey: "issue:42",
            case: { issue: 42 },
            evidenceCutoff: "2026-08-24T12:00:00Z",
            maturesAt: "2026-08-30T12:00:00Z",
            provenance: [{ repository: "github/gh-aw", kind: "issue", ref: "42" }],
          },
          { mode: "baseline-comparable", value: 2 }
        ),
        {},
        { evidenceAt: "2026-08-24T12:00:00Z", env: TEST_ENV }
      )
    ).toThrow("baseline value in [0,1]");
  });
});
