"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { test } = require("node:test");
const { spawnSync } = require("node:child_process");
const { CHECKS, networkCheck, fileRoundTrip, runChecks, verifyReport } = require("./firewall-policy-check.cjs");
const { buildCopilotSDKPermissionHandler } = require("../actions/setup/js/copilot_sdk_permissions.cjs");

const allowed = { exitCode: 0, connectStatus: 200, httpStatus: 200 };
const denied = { exitCode: 56, connectStatus: 403, httpStatus: 0 };

test("only an explicit CONNECT rejection passes the negative control", () => {
  assert.equal(networkCheck("denied-domain", denied, true).status, "PASS");
  for (const result of [
    { exitCode: 6, connectStatus: 0, httpStatus: 0 },
    { exitCode: 28, connectStatus: 0, httpStatus: 0 },
    { exitCode: 60, connectStatus: 200, httpStatus: 0 },
    { exitCode: null, connectStatus: null, httpStatus: null },
    { connectStatus: 403, httpStatus: 0 },
    { exitCode: 0, connectStatus: 403, httpStatus: 0 },
  ]) {
    assert.equal(networkCheck("denied-domain", result, true).status, "INCONCLUSIVE");
  }
});

test("origin responses, including errors, demonstrate unexpected access", () => {
  for (const result of [allowed, { exitCode: 22, connectStatus: 200, httpStatus: 403 }, { exitCode: 22, connectStatus: 0, httpStatus: 500 }]) {
    assert.equal(networkCheck("denied-domain", result, true).status, "FAIL");
  }
  assert.equal(networkCheck("allowed-api", denied).status, "FAIL");
  assert.equal(networkCheck("allowed-api", { exitCode: 28, connectStatus: 0, httpStatus: 0 }).status, "INCONCLUSIVE");
});

test("curl invocation is bounded and emits only HTTP metadata", async t => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "firewall-curl-fixture-"));
  const originalPath = process.env.PATH;
  t.after(() => {
    process.env.PATH = originalPath;
    fs.rmSync(directory, { recursive: true, force: true });
  });
  fs.writeFileSync(
    path.join(directory, "curl"),
    `#!${process.execPath}
const assert = require("node:assert/strict");
const args = process.argv.slice(2);
assert.deepEqual(args.slice(0, -1), ["--silent", "--show-error", "--fail", "--max-time", "10", "--output", "/dev/null", "--write-out", "%{http_connect} %{http_code}"]);
process.stderr.write("private remote diagnostic");
process.stdout.write(args.at(-1) === "https://example.com" ? "403 000" : "200 200");
process.exitCode = args.at(-1) === "https://example.com" ? 56 : 0;
`,
    { mode: 0o700 }
  );
  process.env.PATH = directory;
  const { report } = await fixture(t, { request: undefined });
  assert.equal(report.status, "PASS");
  assert.ok(!JSON.stringify(report).includes("private remote diagnostic"));
});

async function fixture(t, overrides = {}) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "firewall-policy-test-"));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const home = path.join(directory, "home");
  const agent = path.join(directory, "agent");
  const hostname = path.join(directory, "hostname");
  fs.mkdirSync(home);
  fs.mkdirSync(agent);
  fs.writeFileSync(hostname, "fixture-host\n");
  const urls = [];
  const report = await runChecks({
    home,
    agent,
    hostname,
    request: url => {
      urls.push(url);
      return url === "https://example.com" ? denied : allowed;
    },
    resolveDNS: async () => {},
    ...overrides,
  });
  return { directory, home, agent, urls, report };
}

test("fixed suite covers all eight assertions and cleans only owned markers", async t => {
  const { home, agent, urls, report } = await fixture(t);
  assert.deepEqual(urls, ["https://api.github.com", "https://github.com", "https://example.com"]);
  assert.deepEqual(
    report.checks.map(check => check.name),
    CHECKS
  );
  assert.equal(report.status, "PASS");
  verifyReport(report);
  assert.deepEqual(fs.readdirSync(home), []);
  assert.deepEqual(fs.readdirSync(agent), []);
  fs.writeFileSync(path.join(home, ".firewall-test-marker"), "existing");
  fileRoundTrip(home);
  assert.equal(fs.readFileSync(path.join(home, ".firewall-test-marker"), "utf8"), "existing");
});

test("DNS and transport failures cannot claim policy enforcement", async t => {
  const { report } = await fixture(t, {
    request: () => ({ exitCode: 28, connectStatus: 0, httpStatus: 0 }),
    resolveDNS: async () => {
      throw new Error("private diagnostic");
    },
  });
  assert.equal(report.status, "INCONCLUSIVE");
  assert.ok(!JSON.stringify(report).includes("private diagnostic"));
  assert.throws(() => verifyReport(report));
});

test("filesystem and localhost failures fail the suite without hiding remaining results", async t => {
  const { report } = await fixture(t, {
    hostname: path.join(os.tmpdir(), "nonexistent-firewall-hostname"),
    local: async () => {
      throw new Error("local failure");
    },
  });
  assert.equal(report.status, "FAIL");
  assert.equal(report.checks.length, 8);
  assert.equal(report.checks.find(check => check.name === "home-read").status, "FAIL");
  assert.equal(report.checks.find(check => check.name === "localhost").status, "FAIL");
  assert.throws(() => verifyReport(report));
});

test("verifier rejects missing, partial, duplicated, or inconsistent evidence", async t => {
  const { report } = await fixture(t);
  for (const invalid of [
    null,
    {},
    { ...report, checks: [] },
    { ...report, checks: report.checks.slice(0, 7) },
    { ...report, checks: report.checks.map(() => report.checks[0]) },
    { ...report, checks: report.checks.map(check => ({ ...check, status: "INCONCLUSIVE" })) },
    { ...report, checks: report.checks.map(check => (check.name === "denied-domain" ? { ...check, ...allowed } : check)) },
  ])
    assert.throws(() => verifyReport(invalid));
  const missing = spawnSync(process.execPath, [path.join(__dirname, "firewall-policy-check.cjs"), "--verify", path.join(os.tmpdir(), "missing-firewall-report.json")], { encoding: "utf8" });
  assert.equal(missing.status, 1);
  assert.match(missing.stderr, /failed or is incomplete/);
});

test("workflow keeps AWF, bounded execution, and an always-on evidence gate", () => {
  const source = fs.readFileSync(path.join(__dirname, "../.github/workflows/firewall-escape.md"), "utf8");
  assert.match(source, /sandbox:\s+agent:\s+id: awf/);
  assert.match(source, /network:\s+allowed:\s+- defaults\s+- node\s+- github.com\s+- api.github.com/);
  assert.match(source, /edit: false/);
  assert.match(source, /bash: \["node scripts\/firewall-policy-check.cjs:\*"\]/);
  assert.match(source, /post-steps:[\s\S]*?if: always\(\)[\s\S]*?run: node "\$\{RUNNER_TEMP\}\/gh-aw\/firewall-policy-check.cjs" --verify "\$FIREWALL_REPORT"/);
  assert.match(source, /node scripts\/firewall-policy-check.cjs \/tmp\/gh-aw\/agent\/firewall-policy-\$\{\{ github.run_id \}\}.json/);
  assert.match(source, /content-filtered model response[\s\S]*?inconclusive/);
  assert.doesNotMatch(source, /repo-memory:|cache-memory:|ADVANCED SANDBOX|80%|DNS tunneling/);
  assert.match(fs.readFileSync(path.join(__dirname, "../Makefile"), "utf8"), /node --test scripts\/firewall-policy-check.test.cjs/);
});

test("the SDK permits the workflow command but rejects unrelated execution and edits", () => {
  const lock = fs.readFileSync(path.join(__dirname, "../.github/workflows/firewall-escape.lock.yml"), "utf8");
  const config = JSON.parse(lock.match(/GH_AW_COPILOT_SDK_TOOL_CONFIG: '([^']+)'/)[1]);
  const handler = buildCopilotSDKPermissionHandler(config.permissions, () => ({ kind: "approve-once" }));
  assert.equal(handler({ kind: "shell", fullCommandText: "node scripts/firewall-policy-check.cjs /tmp/gh-aw/agent/firewall-policy-42.json" }).kind, "approve-once");
  assert.equal(handler({ kind: "shell", fullCommandText: "node -e 'process.exit(0)'" }).kind, "reject");
  assert.equal(handler({ kind: "write", path: "scripts/firewall-policy-check.cjs" }).kind, "reject");
});

test("host verification executes the staged trusted copy, not a changed checkout", async t => {
  const { directory, report } = await fixture(t);
  const source = fs.readFileSync(path.join(__dirname, "../.github/workflows/firewall-escape.md"), "utf8");
  const stage = source.match(/run: (install -m 0444[^\n]+)/)[1];
  const verify = source.match(/run: (node "\$\{RUNNER_TEMP\}\/gh-aw\/firewall-policy-check.cjs" --verify[^\n]+)/)[1];
  const workspace = path.join(directory, "workspace");
  const runtime = path.join(directory, "runtime");
  fs.mkdirSync(path.join(workspace, "scripts"), { recursive: true });
  fs.mkdirSync(path.join(runtime, "gh-aw"), { recursive: true });
  fs.copyFileSync(path.join(__dirname, "firewall-policy-check.cjs"), path.join(workspace, "scripts/firewall-policy-check.cjs"));
  const reportPath = path.join(directory, "report.json");
  fs.writeFileSync(reportPath, JSON.stringify(report));
  const options = { cwd: workspace, env: { PATH: process.env.PATH, RUNNER_TEMP: runtime, FIREWALL_REPORT: reportPath }, encoding: "utf8" };
  const staged = spawnSync("bash", ["-c", stage], options);
  assert.equal(staged.status, 0, staged.stderr);
  fs.writeFileSync(path.join(workspace, "scripts/firewall-policy-check.cjs"), 'throw new Error("agent-modified source must not execute");');
  const verified = spawnSync("bash", ["-c", verify], options);
  assert.equal(verified.status, 0, verified.stderr);
  assert.match(verified.stdout, /all eight bounded firewall assertions completed/);
  assert.match(lockFile(), /--mount "\$\{RUNNER_TEMP\}\/gh-aw:\$\{RUNNER_TEMP\}\/gh-aw:ro"/);
});

function lockFile() {
  return fs.readFileSync(path.join(__dirname, "../.github/workflows/firewall-escape.lock.yml"), "utf8");
}
