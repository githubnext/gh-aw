import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createRequire } from "node:module";

const req = createRequire(import.meta.url);
const childProcess = req("node:child_process");
const scriptPath = req.resolve("./prepare_objective_impact_outcomes.cjs");
let root;
let exec;
let prepareIssueOutcomes;

beforeEach(() => {
  root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-impact-outcomes-"));
  exec = vi.spyOn(childProcess, "execFileSync");
  delete req.cache[scriptPath];
  ({ prepareIssueOutcomes } = req(scriptPath));
});

afterEach(() => {
  vi.restoreAllMocks();
  delete req.cache[scriptPath];
  fs.rmSync(root, { recursive: true, force: true });
});

function writeRuns(runs) {
  fs.writeFileSync(path.join(root, "workflow-logs.json"), JSON.stringify({ runs }));
}

function report(items, runId = 1) {
  return JSON.stringify({ run_id: runId, workflow: "producer", items });
}

describe("CLI-backed impact report outcomes", () => {
  it("uses gh aw outcomes and preserves its evidence and objective attribution without reclassification", () => {
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
    expect(exec).toHaveBeenCalledExactlyOnceWith("gh", ["aw", "outcomes", "1", "--repo", "acme/repo", "--output", path.join(root, "safe-output-runs"), "--json"], expect.objectContaining({ encoding: "utf8" }));
    const rows = fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8").trim().split("\n").map(JSON.parse);
    expect(rows).toHaveLength(2);
    expect(rows[0]).toMatchObject({ run_id: 1, workflow_name: "producer", workflow_aic: 4, number: 42, outcome_status: "pending", signal: "open", detail: "created successfully", objective_value: 3, zero_touch: false });
    expect(rows[1]).toMatchObject({ outcome_status: "lifecycle_close", signal: "bot_closed" });
    expect(JSON.parse(fs.readFileSync(path.join(root, "safe-output-issue-summary.json"), "utf8"))).toMatchObject({ total_issue_outcomes: 2, accepted_count: 0, pending_count: 1, lifecycle_count: 1 });
  });

  it("supports GitHub API and gh run list inputs and deduplicates source run IDs", () => {
    writeRuns([{ id: 1 }, { databaseId: 2 }, { run_id: 1 }]);
    exec.mockReturnValueOnce(report([])).mockReturnValueOnce(report([], 2));
    prepareIssueOutcomes(root, "acme/repo");
    expect(exec).toHaveBeenCalledTimes(2);
    expect(fs.readFileSync(path.join(root, "safe-output-issue-evaluations.jsonl"), "utf8")).toBe("");
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
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("Failed to evaluate outcomes for run 1");
    expect(fs.existsSync(path.join(root, "safe-output-issue-summary.json"))).toBe(false);
  });

  it.each(["invalid JSON", '{"run_id":1}', '{"run_id":2,"items":[]}', '{"run_id":1,"items":[null]}'])("rejects malformed CLI output: %s", output => {
    writeRuns([{ run_id: 1 }]);
    exec.mockReturnValue(output);
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow(/Invalid gh aw outcomes/);
  });

  it("rejects missing or invalid inputs rather than publishing no outcomes", () => {
    expect(() => prepareIssueOutcomes(root, "")).toThrow("GITHUB_REPOSITORY is required");
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("Failed to read impact report input");
    writeRuns([{ run_id: -1 }]);
    expect(() => prepareIssueOutcomes(root, "acme/repo")).toThrow("invalid run ID");
    expect(exec).not.toHaveBeenCalled();
  });
});
