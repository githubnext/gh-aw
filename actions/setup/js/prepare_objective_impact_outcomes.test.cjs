import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";

const req = createRequire(import.meta.url);
const childProcess = req("node:child_process");
const scriptPath = req.resolve("./prepare_objective_impact_outcomes.cjs");
let root;
let exec;
let prepareIssueOutcomes;

beforeEach(() => {
  vi.stubEnv("GH_AW_OUTCOME_GH_TIMEOUT_MS", "300000");
  root = fs.mkdtempSync(path.join(path.dirname(scriptPath), ".impact-outcomes-test-"));
  exec = vi.spyOn(childProcess, "execFileSync");
  delete req.cache[scriptPath];
  ({ prepareIssueOutcomes } = req(scriptPath));
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  delete req.cache[scriptPath];
  fs.rmSync(root, { recursive: true, force: true });
});

function writeRuns(runs) {
  fs.writeFileSync(path.join(root, "workflow-logs.json"), JSON.stringify({ runs }));
}

function report(items, runId = 1) {
  return JSON.stringify({ outcomes: [{ run_id: runId, workflow: "producer", items }] });
}

describe("CLI-backed impact report outcomes", () => {
  it("uses gh aw logs --outcomes and preserves its evidence and objective attribution without reclassification", () => {
    writeRuns([{ run_id: 1, workflow_name: "old-name", aic: 4, url: "https://github.com/acme/repo/actions/runs/1" }]);
    exec.mockReturnValue(
      report([
        {
          type: "create_issue",
          object_number: 42,
          object_url: "https://github.com/acme/repo/issues/42",
          repo: "acme/repo",
          outcome_status: "pending",
          evidence_strength: "medium",
          signal: "open",
          detail: "created successfully",
          objective_value: 3,
          zero_touch: false,
        },
        { type: "close_issue", object_number: 43, outcome_status: "lifecycle_close", evidence_strength: "strong", signal: "bot_closed" },
        { type: "add_comment", outcome_status: "accepted", evidence_strength: "medium", signal: "engaged" },
      ])
    );
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).toHaveBeenCalledExactlyOnceWith(
      "gh",
      ["aw", "logs", "--stdin", "--outcomes", "--repo", "acme/repo", "--output", path.join(root, "safe-output-runs"), "--timeout", "5", "--json"],
      expect.objectContaining({ encoding: "utf8", input: "1\n" })
    );
    const rows = fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8").trim().split("\n").map(JSON.parse);
    expect(rows).toHaveLength(2);
    expect(rows[0]).toMatchObject({ run_id: 1, workflow_name: "producer", workflow_aic: 4, number: 42, outcome_status: "pending", signal: "open", detail: "created successfully", objective_value: 3, zero_touch: false });
    expect(rows[1]).toMatchObject({ outcome_status: "lifecycle_close", signal: "bot_closed" });
    expect(JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-summary.json"), "utf8"))).toMatchObject({ total_issue_outcomes: 2, accepted_count: 0, pending_count: 1, lifecycle_count: 1 });
  });

  it("supports GitHub API and gh run list inputs and deduplicates source run IDs", () => {
    writeRuns([{ id: 1 }, { databaseId: 2 }, { run_id: 1 }]);
    exec.mockReturnValue(
      JSON.stringify({
        outcomes: [
          { run_id: 2, items: [{ type: "close_issue", outcome_status: "accepted" }] },
          { run_id: 1, items: [{ type: "create_issue", outcome_status: "pending" }] },
        ],
      })
    );
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).toHaveBeenCalledExactlyOnceWith(
      "gh",
      ["aw", "logs", "--stdin", "--outcomes", "--repo", "acme/repo", "--output", path.join(root, "safe-output-runs"), "--timeout", "10", "--json"],
      expect.objectContaining({ input: "1\n2\n", timeout: 600000, maxBuffer: 64 * 1024 * 1024 })
    );
    const rows = fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8").trim().split("\n").map(JSON.parse);
    expect(rows.map(row => row.run_id)).toEqual([1, 2]);
  });

  it("reevaluates source runs on later reports instead of caching outcome classifications", () => {
    writeRuns([{ run_id: 1 }]);
    exec.mockReturnValueOnce(report([{ type: "create_issue", outcome_status: "pending", signal: "open" }]));
    prepareIssueOutcomes(root, "acme/repo");
    exec.mockReturnValueOnce(report([{ type: "create_issue", outcome_status: "accepted", signal: "completed" }]));
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).toHaveBeenCalledTimes(2);
    expect(JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8"))).toMatchObject({ outcome_status: "accepted", signal: "completed" });
  });

  it("surfaces CLI failures without publishing a successful empty report", () => {
    writeRuns([{ run_id: 1 }]);
    exec.mockImplementation(() => {
      throw new Error("unavailable");
    });
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("Failed to evaluate outcomes for selected runs");
    expect(fs.existsSync(path.join(root, "safe-output-issue-summary.json"))).toBe(false);
  });

  it("retains available observations and explicit missing evidence in a partial summary", () => {
    writeRuns([{ run_id: 1 }, { run_id: 2, workflow_name: "missing", url: "https://github.com/acme/repo/actions/runs/2" }]);
    exec.mockReturnValue(
      JSON.stringify({
        outcomes: [
          { run_id: 1, items: [{ type: "create_issue", outcome_status: "accepted" }] },
          { run_id: 2, items: [], eval_error: "safe-output manifest unavailable; outcome coverage is unknown" },
        ],
      })
    );
    prepareIssueOutcomes(root, "acme/repo");
    expect(JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8"))).toMatchObject({ run_id: 1, outcome_status: "accepted" });
    expect(JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-summary.json"), "utf8"))).toMatchObject({
      partial: true,
      selected_run_count: 2,
      evaluated_run_count: 1,
      unavailable_run_count: 1,
      total_issue_outcomes: 1,
      accepted_count: 1,
      run_errors: [{ run_id: 2, workflow_name: "missing", workflow_run_url: "https://github.com/acme/repo/actions/runs/2", eval_error: "safe-output manifest unavailable; outcome coverage is unknown" }],
    });
  });

  it.each([
    ["all unavailable", "safe-output manifest unavailable; outcome coverage is unknown", true, 0, 1],
    ["verified empty", undefined, false, 1, 0],
  ])("distinguishes %s runs from no published issue outcomes", (_label, evalError, partial, evaluated, unavailable) => {
    writeRuns([{ run_id: 1 }]);
    exec.mockReturnValue(JSON.stringify({ outcomes: [{ run_id: 1, items: [], eval_error: evalError }] }));
    prepareIssueOutcomes(root, "acme/repo");
    expect(fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8")).toBe("");
    const summary = JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-summary.json"), "utf8"));
    expect(summary).toMatchObject({ partial, selected_run_count: 1, evaluated_run_count: evaluated, unavailable_run_count: unavailable, total_issue_outcomes: 0 });
    expect(summary.run_errors).toHaveLength(unavailable);
    if (evalError) expect(summary.run_errors[0]).toMatchObject({ run_id: 1, eval_error: evalError });
  });

  it.each([
    "invalid JSON",
    '{"run_id":1}',
    '{"outcomes":[{"run_id":2,"items":[]}]}',
    '{"outcomes":[{"run_id":1,"items":[null]}]}',
    '{"outcomes":[]}',
    '{"outcomes":[{"run_id":1,"items":[]},{"run_id":2,"items":[]}]}',
    '{"outcomes":[{"run_id":1,"items":[],"eval_error":true}]}',
    '{"outcomes":[{"run_id":1,"eval_error":"missing manifest"}]}',
    '{"outcomes":[{"run_id":1,"items":[{"type":"create_issue","outcome_status":"accepted"}],"eval_error":"missing manifest"}]}',
    '{"outcomes":[null]}',
    '{"outcomes":[{"run_id":1,"items":[]},{"run_id":1,"items":[]}]}',
  ])("rejects malformed CLI output: %s", output => {
    writeRuns([{ run_id: 1 }]);
    exec.mockReturnValue(output);
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow(/Invalid gh aw logs --outcomes/);
    expect(fs.existsSync(path.join(root, "safe-output-issue-summary.json"))).toBe(false);
  });

  it("does not invoke the CLI for an empty source run list", () => {
    writeRuns([]);
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).not.toHaveBeenCalled();
    expect(fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8")).toBe("");
  });

  it("clamps the total batch timeout to the subprocess limit", () => {
    vi.stubEnv("GH_AW_OUTCOME_GH_TIMEOUT_MS", "2147483647");
    writeRuns([{ run_id: 1 }, { run_id: 2 }]);
    exec.mockReturnValue(
      JSON.stringify({
        outcomes: [
          { run_id: 1, items: [] },
          { run_id: 2, items: [] },
        ],
      })
    );
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).toHaveBeenCalledWith("gh", expect.any(Array), expect.objectContaining({ timeout: 2147483647 }));
  });

  it("validates every source ID before issuing API requests", () => {
    writeRuns([{ run_id: 1 }, { run_id: -2 }]);
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("invalid run ID");
    expect(exec).not.toHaveBeenCalled();
  });

  it("rejects missing or invalid inputs rather than publishing no outcomes", () => {
    expect(() => prepareIssueOutcomes(root, "")).toThrow("GITHUB_REPOSITORY is required");
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("Failed to read impact report input");
    writeRuns([{ run_id: -1 }]);
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("invalid run ID");
    expect(exec).not.toHaveBeenCalled();
  });
});
