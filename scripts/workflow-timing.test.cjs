"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { test } = require("node:test");

const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;

function block(source, key) {
  const match = source.match(new RegExp(`^( +)${key}: \\|\\n((?:\\1  .*\\n|\\n)+)`, "m"));
  assert.ok(match, `Missing ${key} block`);
  return match[2].replace(new RegExp(`^${match[1]}  `, "gm"), "");
}

for (const workflow of ["cgo", "CWI", "license-check"]) {
  const source = fs.readFileSync(path.join(__dirname, `../.github/workflows/${workflow}.yml`), "utf8");
  const timing = source.slice(source.indexOf("  summarize-timing:"));
  const script = new AsyncFunction("require", "github", "context", "core", block(timing, "script"));

  function harness(t, result) {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "workflow-timing-"));
    t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
    const summary = path.join(directory, "summary.md");
    const warnings = [];
    const calls = [];
    const context = { repo: { owner: "example", repo: "repository" }, runId: 123 };
    const listJobs = () => {};
    const github = {
      rest: { actions: { listJobsForWorkflowRun: listJobs } },
      paginate: async (method, parameters) => {
        assert.equal(method, listJobs);
        calls.push(parameters);
        if (result instanceof Error) throw result;
        return result;
      },
    };
    const run = () =>
      script(
        name => {
          assert.equal(name, "fs");
          return {
            appendFileSync: (_filename, content) => fs.appendFileSync(summary, content),
          };
        },
        github,
        context,
        { warning: message => warnings.push(message) }
      );
    return { run, warnings, calls, summary };
  }

  test(`${workflow}: summarizes failed jobs and steps without hiding their conclusions`, async t => {
    const fixture = harness(t, [
      {
        name: "Build | Windows",
        html_url: "https://github.com/example/repository/actions/runs/123/job/456",
        conclusion: "failure",
        started_at: "2026-10-13T10:00:00Z",
        completed_at: "2026-10-13T10:01:05Z",
        steps: [{ name: "Compile\nCLI", conclusion: "failure" }],
      },
      { name: "Summarize workflow timing", steps: [] },
    ]);
    await fixture.run();
    assert.deepEqual(fixture.calls, [{ owner: "example", repo: "repository", run_id: 123, per_page: 100 }]);
    const summary = fs.readFileSync(fixture.summary, "utf8");
    assert.match(summary, /Build \\\| Windows/);
    assert.match(summary, /\| job \| failure \| 1m 5s \|/);
    assert.match(summary, /Compile CLI \| step \| failure \| n\/a/);
    assert.doesNotMatch(summary, /Summarize workflow timing/);
    assert.deepEqual(fixture.warnings, []);
  });

  for (const status of [403, 429]) {
    test(`${workflow}: reports rate-limit ${status} as unavailable timing, not an empty success table`, async t => {
      const error = Object.assign(new Error("API rate limit exceeded for installation."), { status });
      const fixture = harness(t, error);
      await fixture.run();
      assert.equal(fixture.calls.length, 1);
      assert.equal(fixture.warnings.length, 1);
      assert.match(fixture.warnings[0], /timing.*unavailable.*rate limit/i);
      const summary = fs.readFileSync(fixture.summary, "utf8");
      assert.match(summary, /unavailable.*rate limit/i);
      assert.match(summary, /Build and test results are unchanged/);
      assert.doesNotMatch(summary, /\| Name \|/);
    });
  }

  for (const [status, message] of [
    [401, "Bad credentials"],
    [403, "Resource not accessible by integration"],
    [404, "Not Found"],
    [500, "Internal Server Error"],
    [500, "API rate limit exceeded"],
  ]) {
    test(`${workflow}: propagates non-quota API error ${status}: ${message}`, async t => {
      const error = Object.assign(new Error(message), { status });
      const fixture = harness(t, error);
      await assert.rejects(fixture.run, candidate => candidate === error);
      assert.deepEqual(fixture.warnings, []);
      assert.equal(fs.existsSync(fixture.summary), false);
    });
  }

  if (workflow === "license-check") continue;

  test(`${workflow}: checkout-free cache cleanup targets the explicit repository`, t => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "workflow-cache-"));
    t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
    const gh = path.join(directory, "gh");
    fs.writeFileSync(
      gh,
      `#!/bin/bash
set -eu
case "$*" in
  *'--repo example/repository'*) ;;
  *) echo 'missing explicit repository' >&2; exit 1 ;;
esac
printf '%s\\n' "$*" >> "$CALL_LOG"
if [ "$2" = list ]; then printf '%s' "$CACHE_ID"; fi
`,
      { mode: 0o755 }
    );
    for (const cacheId of ["456", ""]) {
      const callLog = path.join(directory, `calls-${cacheId || "empty"}`);
      const result = spawnSync("bash", ["-e", "-c", block(timing, "run")], {
        cwd: directory,
        env: {
          PATH: `${directory}:${process.env.PATH}`,
          GITHUB_REPOSITORY: "example/repository",
          GITHUB_REF: "refs/pull/1/merge",
          CACHE_KEY: "checkout-example",
          CACHE_ID: cacheId,
          CALL_LOG: callLog,
        },
        encoding: "utf8",
      });
      assert.equal(result.status, 0, result.stderr);
      const calls = fs.readFileSync(callLog, "utf8").trim().split("\n");
      assert.equal(calls.length, cacheId ? 2 : 1);
      assert.match(calls[0], /--ref refs\/pull\/1\/merge/);
      assert.match(calls[0], /--key checkout-example/);
      if (cacheId) assert.match(calls[1], /cache delete 456/);
    }
    assert.match(timing, /continue-on-error: true/);
    assert.doesNotMatch(timing.slice(timing.indexOf("- name: Summarize job")), /continue-on-error/);
  });
}
