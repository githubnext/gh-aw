"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { test } = require("node:test");
const { LOG_PATHS, verifyCreditLimit } = require("./check-max-ai-credits.cjs");

const rejection = { event: "ai_credits_limit_exceeded", total_ai_credits: 1.02, max_ai_credits: 1, hard_cap: false };

test("workflow bounds one-session tool continuation and gates its conclusion on runtime evidence", () => {
  const workflowPath = path.join(__dirname, "../workflows/daily-max-ai-credits-test");
  const source = fs.readFileSync(`${workflowPath}.md`, "utf8");
  const lock = fs.readFileSync(`${workflowPath}.lock.yml`, "utf8");
  const reads = [...source.matchAll(/^\| ([1-6]) \| `([^`]+)` \| (\d+)–(\d+) \|$/gm)];
  assert.equal(reads.length, 6);
  for (const [index, read] of reads.entries()) {
    assert.equal(Number(read[1]), index + 1);
    assert.equal(Number(read[4]) - Number(read[3]) + 1, 240);
  }
  assert.match(source, /within this single agent session/);
  assert.match(source, /immediately invoke the next tool/);
  assert.match(source, /create one issue titled/);
  assert.doesNotMatch(source, /at least four separate assistant turns/);
  assert.doesNotMatch(source, /"!ai_credits_rate_limit_error"/);
  assert.match(lock, /conclusion:\n[\s\S]*?needs:\n(?:      - .+\n)*      - verify_credit_limit/);
  assert.match(lock, /verify_credit_limit:\n[\s\S]*?needs: agent\n    if: always\(\) && !cancelled\(\) && needs\.agent\.result != 'skipped'/);
  assert.match(lock, /run: node \.github\/scripts\/check-max-ai-credits\.cjs \/tmp\/gh-aw\//);
});

function fixture(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "credit-guardrail-test-"));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const write = (name, content) => {
    const filename = path.join(directory, name);
    fs.mkdirSync(path.dirname(filename), { recursive: true });
    fs.writeFileSync(filename, content);
  };
  write("awf-config.json", JSON.stringify({ apiProxy: { maxAiCredits: 1 } }));
  return { directory, write };
}

for (const logPath of LOG_PATHS) {
  test(`accepts an actual per-run rejection in ${logPath}`, t => {
    const { directory, write } = fixture(t);
    write(logPath, `${JSON.stringify(rejection)}\n`);
    assert.match(verifyCreditLimit(directory), /^Passed:/);
  });
}

test("rejects the observed below-limit completion and an agent success claim", t => {
  const { directory, write } = fixture(t);
  write("agent_usage.json", JSON.stringify({ ai_credits: 0.39811 }));
  write("agent-stdio.log", "Turn 1 summary\nMaximum AI credits exceeded (1.02 / 1)\n");
  write("safeoutputs.jsonl", JSON.stringify({ type: "noop", message: "Guardrail test passed" }));
  write(LOG_PATHS[0], JSON.stringify({ event: "request_complete", status: 200 }));
  assert.throws(() => verifyCreditLimit(directory), /enforcement remains unverified/);
});

test("credit totals at or above the cap do not prove request rejection", t => {
  const { directory, write } = fixture(t);
  write("sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl", JSON.stringify({ ai_credits_total: 1.02 }));
  write(LOG_PATHS[0], JSON.stringify({ event: "token_budget_usage", ai_credits_total: 1.02 }));
  assert.throws(() => verifyCreditLimit(directory), /No AWF per-run credit rejection/);
});

for (const entry of [
  { ...rejection, event: "ai_credits_rate_limit_error" },
  { ...rejection, event: "max_cache_misses_exceeded" },
  { ...rejection, max_ai_credits: 10000, hard_cap: true },
  { ...rejection, total_ai_credits: 0.39811 },
  { ...rejection, total_ai_credits: "1.02" },
  { ...rejection, hard_cap: true },
  { event: "request_complete", response: rejection },
  { message: JSON.stringify(rejection) },
]) {
  test(`rejects non-matching runtime evidence ${JSON.stringify(entry)}`, t => {
    const { directory, write } = fixture(t);
    write(LOG_PATHS[0], JSON.stringify(entry));
    assert.throws(() => verifyCreditLimit(directory), /No AWF per-run credit rejection/);
  });
}

test("accepts a rejection at exactly the cap", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], JSON.stringify({ ...rejection, total_ai_credits: 1 }));
  assert.match(verifyCreditLimit(directory), /^Passed:/);
});

test("missing logs fail verification", t => {
  const { directory } = fixture(t);
  assert.throws(() => verifyCreditLimit(directory), /runtime evidence may be missing/);
});

test("a different compiled cap fails even with a matching event", t => {
  const { directory, write } = fixture(t);
  write("awf-config.json", JSON.stringify({ apiProxy: { maxAiCredits: 2 } }));
  write(LOG_PATHS[0], JSON.stringify(rejection));
  assert.throws(() => verifyCreditLimit(directory), /cap to be exactly 1/);
});

test("malformed logs fail without echoing their contents", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], `${JSON.stringify(rejection)}\nPRIVATE_PAYLOAD\n`);
  assert.throws(
    () => verifyCreditLimit(directory),
    error => /Invalid firewall JSONL.*line 2/.test(error.message) && !error.message.includes("PRIVATE_PAYLOAD")
  );
});

test("malformed configuration fails without echoing its contents", t => {
  const { directory, write } = fixture(t);
  write("awf-config.json", "PRIVATE_PAYLOAD");
  assert.throws(
    () => verifyCreditLimit(directory),
    error => /Invalid AWF configuration JSON/.test(error.message) && !error.message.includes("PRIVATE_PAYLOAD")
  );
});

test("CLI exits nonzero for unexpected completion and writes a summary only on pass", t => {
  const { directory, write } = fixture(t);
  const summary = path.join(directory, "summary.md");
  const run = () => spawnSync(process.execPath, [path.join(__dirname, "check-max-ai-credits.cjs"), directory], { encoding: "utf8", env: { ...process.env, GITHUB_STEP_SUMMARY: summary } });
  const failed = run();
  assert.equal(failed.status, 1);
  assert.match(failed.stderr, /enforcement remains unverified/);
  assert.equal(fs.existsSync(summary), false);
  write(LOG_PATHS[0], JSON.stringify(rejection));
  const passed = run();
  assert.equal(passed.status, 0);
  assert.match(fs.readFileSync(summary, "utf8"), /Passed: AWF recorded/);
});
