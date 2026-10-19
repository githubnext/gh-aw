import { describe, it, expect } from "vitest";
import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";
import { createRequire } from "node:module";

const req = createRequire(import.meta.url);
const source = fs.readFileSync(new URL("./evaluate_outcomes.cjs", import.meta.url), "utf8");
const cache = "/tmp/gh-aw/cache-memory/outcome-collector";
const summaryPath = "/tmp/gh-aw/outcome-summary.json";
const manifestPath = `${cache}/run-1/safe-output-items.jsonl`;

function collectorHarness() {
  const files = new Map();
  const harness = {
    files,
    recentRuns: [{ databaseId: 1, workflowName: "worker", event: "push" }],
    items: [{ type: "create_pull_request", number: 42, timestamp: "2026-01-01T00:00:00Z" }],
    merged: false,
    apiError: false,
    listError: false,
    downloads: 0,
    apiCalls: [],
  };
  const fakeFS = {
    mkdirSync() {},
    existsSync: filename => files.has(filename),
    readFileSync(filename) {
      if (!files.has(filename)) throw Object.assign(new Error("not found"), { code: "ENOENT" });
      return files.get(filename);
    },
    writeFileSync: (filename, data) => files.set(filename, data),
    renameSync(from, to) {
      files.set(to, files.get(from));
      files.delete(from);
    },
    appendFileSync: (filename, data) => files.set(filename, (files.get(filename) || "") + data),
    copyFileSync: (from, to) => files.set(to, files.get(from)),
    unlinkSync: filename => files.delete(filename),
  };
  const execFileSync = (_command, args) => {
    if (args[0] === "run" && args[1] === "list") {
      if (harness.listError) throw new Error("listing unavailable");
      return JSON.stringify(harness.recentRuns);
    }
    if (args[0] === "run" && args[1] === "download") {
      harness.downloads++;
      files.set(path.join(args[args.indexOf("--dir") + 1], "safe-output-items.jsonl"), harness.items.map(item => JSON.stringify(item)).join("\n"));
      return "";
    }
    if (args[0] === "api") {
      harness.apiCalls.push(args[1]);
      if (harness.apiError) throw Object.assign(new Error("unavailable"), { stderr: "HTTP 503" });
      const response = args[1].endsWith("/pulls/42")
        ? { merged: harness.merged, state: harness.merged ? "closed" : "open", merged_at: harness.merged ? "2026-01-02T00:00:00Z" : null }
        : args[1].includes("/actions/runs/")
          ? { status: "completed", conclusion: "success" }
          : [];
      return JSON.stringify([response]);
    }
    throw new Error(`Unexpected command: ${args.join(" ")}`);
  };
  const testRequire = name => {
    if (name === "fs") return fakeFS;
    if (name === "child_process") return { execFileSync };
    if (name === "./shim.cjs") return {};
    if (name === "./jsonl_helpers.cjs") {
      return {
        readJsonlFile: function* (filename) {
          for (const line of fakeFS.readFileSync(filename).split("\n")) {
            if (line) yield JSON.parse(line);
          }
        },
      };
    }
    return req(name);
  };
  const context = {
    require: testRequire,
    module: { exports: {} },
    process: { env: { GITHUB_REPOSITORY: "acme/repo" } },
    console: { warn() {} },
    core: { info() {} },
  };
  vm.runInNewContext(source, context, { filename: "evaluate_outcomes.cjs" });
  return {
    ...harness,
    state: harness,
    run() {
      context.module.exports.main();
      return JSON.parse(files.get(summaryPath));
    },
    read: filename => JSON.parse(files.get(filename)),
  };
}

describe("outcome collector lifecycle", () => {
  it("revisits a pending PR outside discovery, then retires it after merge", () => {
    const h = collectorHarness();
    expect(h.run()).toMatchObject({ runs_checked: 1, pending: 1, accepted: 0, unresolved_runs: 1 });
    expect(h.files.has(manifestPath)).toBe(true);
    expect(h.read(`${cache}/seen-runs.json`)).toEqual({ version: 2, ids: [] });

    h.state.recentRuns = [];
    h.state.merged = true;
    expect(h.run()).toMatchObject({ runs_checked: 1, runs_revisited: 1, pending: 0, accepted: 1, unresolved_runs: 0 });
    expect(h.state.downloads).toBe(1);
    expect(h.files.has(manifestPath)).toBe(false);
    expect(h.read(`${cache}/pending-runs.json`)).toEqual([]);
    expect(h.read(`${cache}/seen-runs.json`)).toEqual({ version: 2, ids: [1] });
    expect(h.run()).toMatchObject({ runs_checked: 0, total_outcomes: 0 });
  });

  it("retries API failures instead of permanently marking the run seen", () => {
    const h = collectorHarness();
    h.state.items = [{ type: "dispatch_workflow", metadata: { run_id: 42 } }];
    h.state.apiError = true;
    expect(h.run()).toMatchObject({ errors: 1, unresolved_runs: 1 });
    h.state.apiError = false;
    h.state.recentRuns = [];
    expect(h.run()).toMatchObject({ accepted: 1, errors: 0, unresolved_runs: 0 });
  });

  it("keeps unknown evaluations visible across cycles", () => {
    const h = collectorHarness();
    h.state.items = [{ type: "create_discussion", number: 42 }];
    expect(h.run()).toMatchObject({ unknown: 1, unresolved_runs: 1 });
    h.state.recentRuns = [];
    expect(h.run()).toMatchObject({ unknown: 1, runs_revisited: 1, unresolved_runs: 1 });
  });

  it("retains a mixed run until every actionable item has resolved", () => {
    const h = collectorHarness();
    h.state.items.push({ type: "dispatch_workflow", metadata: { run_id: 99 } });
    expect(h.run()).toMatchObject({ total_outcomes: 2, accepted: 1, pending: 1, unresolved_runs: 1 });
    h.state.recentRuns = [];
    h.state.merged = true;
    expect(h.run()).toMatchObject({ total_outcomes: 2, accepted: 2, pending: 0, unresolved_runs: 0, runs_revisited: 1 });
    expect(h.state.downloads).toBe(1);
    expect(h.files.has(manifestPath)).toBe(false);
  });

  it("migrates legacy seen IDs by evaluating rediscovered runs again", () => {
    const h = collectorHarness();
    h.files.set(`${cache}/seen-runs.json`, "[1]");
    expect(h.run()).toMatchObject({ pending: 1, unresolved_runs: 1 });
  });

  it("does not reevaluate terminal runs from the discovery window", () => {
    const h = collectorHarness();
    h.state.merged = true;
    expect(h.run()).toMatchObject({ accepted: 1, unresolved_runs: 0 });
    expect(h.run()).toMatchObject({ runs_checked: 0, total_outcomes: 0 });
    expect(h.state.downloads).toBe(1);
  });

  it("does not turn listing failures into an empty success report", () => {
    const h = collectorHarness();
    h.state.listError = true;
    expect(() => h.run()).toThrow("Failed to list runs for outcome evaluation");
    expect(h.files.has(summaryPath)).toBe(false);
  });

  it("fails explicitly rather than resetting corrupt collector state", () => {
    const h = collectorHarness();
    h.files.set(`${cache}/pending-runs.json`, "not json");
    expect(() => h.run()).toThrow("Failed to read outcome state");
    expect(h.files.get(`${cache}/pending-runs.json`)).toBe("not json");
  });

  it.each([
    ["seen-runs.json", "null", "Invalid resolved outcome run cache"],
    ["seen-runs.json", '{"version":2,"ids":[-1]}', "Invalid resolved outcome run cache"],
    ["pending-runs.json", "[null]", "Invalid pending outcome run cache"],
  ])("rejects invalid persisted state in %s", (filename, content, message) => {
    const h = collectorHarness();
    h.files.set(`${cache}/${filename}`, content);
    expect(() => h.run()).toThrow(message);
    expect(h.files.get(`${cache}/${filename}`)).toBe(content);
  });
});
