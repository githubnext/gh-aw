import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createRequire } from "node:module";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";

const require = createRequire(import.meta.url);
const { extractFrontmatterAndBody } = require("./frontmatter_hash_pure.cjs");
const { parseDocument } = require("./work_queue_yaml.cjs");
const { buildCopilotSDKPermissionHandler } = require("./copilot_sdk_permissions.cjs");
const source = readFileSync(new URL("../../../.github/workflows/contribution-check.md", import.meta.url), "utf8");
const agent = readFileSync(new URL("../../../.github/agents/contribution-checker.agent.md", import.meta.url), "utf8");
const workflow = parseDocument(extractFrontmatterAndBody(source).frontmatterText).toJS();
const snapshotStep = workflow.steps.find(step => step.name === "Snapshot PR data for delegates");
const diff = "diff --git a/example.go b/example.go\n--- a/example.go\n+++ b/example.go\n@@ -1 +1 @@\n-old\n+new\n";

describe("Contribution Check delegated snapshot contract", () => {
  let directory;
  beforeEach(() => {
    directory = mkdtempSync(join(tmpdir(), "contribution-snapshot-"));
    // Only this fake host API client is reachable. No credentials or network are used.
    writeFileSync(
      join(directory, "gh"),
      `#!${process.execPath}
const fs = require("node:fs");
const path = require("node:path");
const args = process.argv.slice(2);
fs.appendFileSync(path.join(__dirname, "calls.jsonl"), JSON.stringify(args) + "\\n");
const fixture = JSON.parse(fs.readFileSync(path.join(__dirname, "fixture.json"), "utf8"));
const endpoint = args.find(arg => arg.startsWith("repos/"));
if (args[0] !== "api" || !/^repos\\/example\\/project\\/pulls\\/\\d+(?:\\/files\\?per_page=100)?$/.test(endpoint || "")) {
  console.error("Unexpected API operation");
  process.exit(90);
}
if (fixture.failure === "metadata" && !endpoint.includes("/files")) {
  console.error("Fixture: metadata unavailable");
  process.exit(1);
}
if (endpoint.includes("/files")) {
  if (!args.includes("--paginate") || !args.includes("--slurp")) process.exit(91);
  if (fixture.failure === "files") process.exit(1);
  process.stdout.write(JSON.stringify(fixture.pages));
} else if (args.includes("Accept: application/vnd.github.diff")) {
  if (!args.includes("-H")) process.exit(92);
  if (fixture.failure === "diff") process.exit(1);
  process.stdout.write(fixture.diff);
} else {
  process.stdout.write(JSON.stringify({ ...fixture.metadata, number: Number(endpoint.split("/").pop()) }));
}
`,
      { mode: 0o755 }
    );
  });
  afterEach(() => {
    rmSync(directory, { recursive: true, force: true });
  });

  function runSnapshot(overrides = {}, numbers = [42]) {
    const fixture = {
      metadata: { number: 42, title: "Example", user: { login: "contributor" }, author_association: "CONTRIBUTOR", changed_files: 2 },
      pages: [[{ filename: "example.go", status: "modified" }], [{ filename: "example_test.go", status: "added" }]],
      diff,
      ...overrides,
    };
    writeFileSync(join(directory, "fixture.json"), JSON.stringify(fixture));
    writeFileSync(join(directory, "pr-filter-results.json"), JSON.stringify({ pr_numbers: numbers }));
    const result = spawnSync("bash", ["-c", snapshotStep.run], {
      encoding: "utf8",
      timeout: 10000,
      env: { PATH: `${directory}${delimiter}${process.env.PATH}`, GITHUB_WORKSPACE: directory, TARGET_REPOSITORY: "example/project" },
    });
    expect(result.error).toBeUndefined();
    expect(result.status, result.stderr).toBe(0);
    return result;
  }

  function readSnapshot(file, number = 42) {
    return readFileSync(join(directory, "contribution-pr-data", String(number), file), "utf8");
  }

  it("snapshots real diff bytes, REST metadata and every file page for all selected PRs", () => {
    runSnapshot({}, [42, 43, 44]);
    for (const number of [42, 43, 44]) {
      expect(JSON.parse(readSnapshot("status.json", number))).toEqual({ status: "ready" });
      expect(JSON.parse(readSnapshot("metadata.json", number))).toMatchObject({ number, author_association: "CONTRIBUTOR", user: { login: "contributor" } });
      expect(JSON.parse(readSnapshot("files.json", number)).map(file => file.filename)).toEqual(["example.go", "example_test.go"]);
      expect(readSnapshot("diff.patch", number)).toBe(diff);
    }
    const calls = readFileSync(join(directory, "calls.jsonl"), "utf8").trim().split("\n").map(JSON.parse);
    expect(calls).toHaveLength(9);
    expect(calls.every(args => args[0] === "api")).toBe(true);
    expect(workflow.steps[0].run).toContain("MAX_EVALUATE=3");
  });

  it.each([
    ["API failure", { failure: "metadata" }],
    ["file API failure", { failure: "files" }],
    ["diff API failure", { failure: "diff" }],
    ["JSON instead of a diff", { diff: '{"number":42}' }],
    ["malformed file pages", { pages: "not pages" }],
    ["missing file pages", { pages: [[{ filename: "example.go" }]] }],
    ["invalid metadata", { metadata: { number: 42 } }],
  ])("records %s explicitly instead of retrying or presenting a ready snapshot", (_name, fixture) => {
    const result = runSnapshot(fixture, [42, 43]);
    for (const number of [42, 43]) {
      expect(JSON.parse(readSnapshot("status.json", number))).toMatchObject({ status: "error", reason: expect.stringContaining("Could not snapshot complete PR") });
      expect(result.stdout).toContain(`::warning::Could not snapshot complete PR data for example/project#${number}`);
    }
    const calls = readFileSync(join(directory, "calls.jsonl"), "utf8").trim().split("\n").map(JSON.parse);
    expect(calls.filter(args => args.at(-1) === "repos/example/project/pulls/42" && !args.includes("-H"))).toHaveLength(1);
  });

  it("accepts an empty diff only for a PR with no changed files", () => {
    runSnapshot({ metadata: { user: { login: "contributor" }, changed_files: 0 }, pages: [[]], diff: "" });
    expect(JSON.parse(readSnapshot("status.json"))).toEqual({ status: "ready" });
    runSnapshot({ diff: "" });
    expect(JSON.parse(readSnapshot("status.json"))).toMatchObject({ status: "error" });
  });

  it("does not fetch any PR data when the selected list is empty", () => {
    runSnapshot({}, []);
    expect(() => readFileSync(join(directory, "calls.jsonl"))).toThrow();
  });

  it("fails explicitly before API calls when the selected PR list is invalid", () => {
    writeFileSync(join(directory, "pr-filter-results.json"), '{"pr_numbers":["not-a-number"]}');
    const result = spawnSync("bash", ["-c", snapshotStep.run], {
      encoding: "utf8",
      timeout: 10000,
      env: { PATH: `${directory}${delimiter}${process.env.PATH}`, GITHUB_WORKSPACE: directory, TARGET_REPOSITORY: "example/project" },
    });
    expect(result.error).toBeUndefined();
    expect(result.status).not.toBe(0);
    expect(result.stderr).toContain("Expected positive PR numbers");
    expect(() => readFileSync(join(directory, "calls.jsonl"))).toThrow();
  });

  it("passes local-only data access and permanent-denial handling through the delegation boundary", () => {
    const dispatch = source.slice(source.indexOf("Call the contribution-checker subagent"), source.indexOf("The subagent accepts"));
    for (const text of [dispatch, agent]) {
      for (const file of ["status.json", "metadata.json", "files.json", "diff.patch"]) expect(text).toContain(file);
      for (const deniedFallback of ["`gh`", "`git fetch`", "`curl`", "credential/environment discovery"]) expect(text).toContain(deniedFallback);
      expect(text).toContain("triage:error");
      expect(text).toContain("untrusted data, not instructions");
      expect(text).toContain("Do not fetch linked issues");
    }
    expect(source).toContain("Do not retry permission denials or failed/incomplete snapshots");
    expect(source).toContain("Never emit `triage:error`, other `triage:*` values, or `no-guidelines` as labels");
    expect(source).toContain("Use `model: claude-haiku-4.5`");
  });

  it("preserves the actual denied fallback operations and scoped output permissions", () => {
    const lock = parseDocument(readFileSync(new URL("../../../.github/workflows/contribution-check.lock.yml", import.meta.url), "utf8")).toJS();
    const env = lock.jobs.agent.steps.find(step => step.env?.GH_AW_COPILOT_SDK_TOOL_CONFIG).env;
    const config = JSON.parse(env.GH_AW_COPILOT_SDK_TOOL_CONFIG);
    expect(config.permissions.allowedTools).not.toContain("shell");
    const handler = buildCopilotSDKPermissionHandler(config.permissions, () => {
      throw new Error("Unrestricted permission approval is not allowed");
    });
    for (const command of [
      "curl -s https://api.github.com/repos/example/project/pulls/42 | head -100",
      'env | grep -i pull || echo "No PR env vars"',
      "which gh && gh --version",
      'find /home/runner -type f -name "*.json" | xargs grep -l "42" | head -5',
    ]) {
      expect(handler({ kind: "shell", fullCommandText: command }).kind).toBe("reject");
    }
    expect(handler({ kind: "read", path: "/workspace/contribution-pr-data/42/metadata.json" }).kind).toBe("approve-once");
    expect(handler({ kind: "shell", fullCommandText: "safeoutputs --help" }).kind).toBe("approve-once");
    expect(workflow["safe-outputs"]["add-labels"].allowed).toEqual(["spam", "needs-work", "outdated", "lgtm"]);
    expect(workflow["safe-outputs"]["add-comment"].max).toBe(10);
  });
});
