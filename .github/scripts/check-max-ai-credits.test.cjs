"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { test } = require("node:test");
const { LOG_PATHS, USAGE_PATH, verifyCreditLimit: verify } = require("./check-max-ai-credits.cjs");
const expected = { runId: "38556022048", runAttempt: "1", repository: "github/gh-aw" };
const verifyCreditLimit = directory => verify(directory, expected);
const captured = require("./fixtures/max-ai-credits-rejection-38556022048.json");

// Contract fixture, not a captured producer event.
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
  write("aw_info.json", JSON.stringify(captured.info));
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
  assert.throws(() => verifyCreditLimit(directory), /Runtime evidence may be missing/);
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
  const run = () =>
    spawnSync(process.execPath, [path.join(__dirname, "check-max-ai-credits.cjs"), directory], {
      encoding: "utf8",
      env: { ...process.env, GITHUB_STEP_SUMMARY: summary, GITHUB_RUN_ID: expected.runId, GITHUB_RUN_ATTEMPT: expected.runAttempt, GITHUB_REPOSITORY: expected.repository },
    });
  const failed = run();
  assert.equal(failed.status, 1);
  assert.match(failed.stderr, /enforcement remains unverified/);
  assert.equal(fs.existsSync(summary), false);
  write(LOG_PATHS[0], JSON.stringify(rejection));
  const passed = run();
  assert.equal(passed.status, 0);
  assert.match(fs.readFileSync(summary, "utf8"), /Passed: AWF recorded/);
});

test("captured genuine Codex rejection remains unverified without a structured AWF rejection", t => {
  const { directory, write } = fixture(t);
  write("awf-config.json", JSON.stringify(captured.config));
  write(
    LOG_PATHS.find(name => name.includes("token-tracker-audit")),
    captured.tracker.map(entry => JSON.stringify(entry)).join("\n")
  );
  write(USAGE_PATH, JSON.stringify(captured.usage));
  write("agent-stdio.log", JSON.stringify(captured.terminalError));
  assert.throws(
    () => verifyCreditLimit(directory),
    error =>
      /Inspected 2 records in 1 candidate firewall files/.test(error.message) &&
      /check AWF rejection-event emission and artifact collection/.test(error.message) &&
      /Last AWF token_usage total: 1.021966 \(at\/above cap 1\)/.test(error.message) &&
      /Codex terminal API error reports credit rejection \(1.021966 \/ 1\)/.test(error.message) &&
      /enforcement remains unverified/.test(error.message)
  );
});

for (const info of [{ ...captured.info, run_id: 38242557753 }, { ...captured.info, run_attempt: "2" }, { ...captured.info, repository: "another/repository" }, {}, null]) {
  test(`rejects wrong or missing artifact identity ${JSON.stringify(info)}`, t => {
    const { directory, write } = fixture(t);
    write("aw_info.json", JSON.stringify(info));
    write(LOG_PATHS[0], JSON.stringify(rejection));
    assert.throws(() => verifyCreditLimit(directory), /artifact repository\/run\/attempt does not match/);
  });
}

for (const identity of [{ run_id: 38242557753 }, { run_attempt: 2 }, { repository: "another/repository" }]) {
  test(`rejects explicit wrong-run rejection even after qualifying evidence ${JSON.stringify(identity)}`, t => {
    const { directory, write } = fixture(t);
    write(LOG_PATHS[0], [rejection, { ...rejection, ...identity }].map(entry => JSON.stringify(entry)).join("\n"));
    assert.throws(() => verifyCreditLimit(directory), /Firewall rejection repository\/run\/attempt does not match/);
  });
}

test("verification requires an independently supplied expected run identity", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], JSON.stringify(rejection));
  assert.throws(() => verify(directory, {}), /Expected repository, run ID and run attempt are required/);
  fs.rmSync(path.join(directory, "aw_info.json"));
  assert.throws(() => verifyCreditLimit(directory), /Missing aw_info.json/);
  write("aw_info.json", "PRIVATE_PAYLOAD");
  assert.throws(
    () => verifyCreditLimit(directory),
    error => /Invalid JSON in aw_info.json/.test(error.message) && !error.message.includes("PRIVATE_PAYLOAD")
  );
});

test("malformed tracker or usage evidence fails even after a valid rejection", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], JSON.stringify(rejection));
  const trackerPath = LOG_PATHS.find(name => name.includes("token-tracker-audit"));
  write(trackerPath, "PRIVATE_PAYLOAD");
  assert.throws(() => verifyCreditLimit(directory), /Invalid firewall JSONL.*token-tracker-audit/);
  write(trackerPath, "");
  write(USAGE_PATH, "PRIVATE_PAYLOAD");
  assert.throws(() => verifyCreditLimit(directory), /Invalid firewall JSONL.*token-usage/);
});

for (const entry of [
  { type: "error", message: "unexpected status 403 Forbidden: Access denied." },
  { type: "item.completed", item: { type: "command_execution", aggregated_output: JSON.stringify(captured.terminalError) } },
  { type: "error", message: `Quoted previous-run log: ${captured.terminalError.message}` },
  { type: "report_incomplete", message: captured.terminalError.message },
  { type: "error", message: "unexpected status 403 Forbidden: Maximum AI credits exceeded (1.02 / 2)." },
]) {
  test(`never passes on non-authoritative agent evidence ${entry.type}: ${entry.message || "tool output"}`, t => {
    const { directory, write } = fixture(t);
    write("agent-stdio.log", JSON.stringify(entry));
    assert.throws(() => verifyCreditLimit(directory), /enforcement remains unverified/);
    if (entry !== undefined && entry.message !== "unexpected status 403 Forbidden: Maximum AI credits exceeded (1.02 / 2).") {
      assert.throws(
        () => verifyCreditLimit(directory),
        error => !error.message.includes("Codex terminal API error reports")
      );
    }
  });
}

test("diagnostics omit untrusted payloads and do not mistake below-cap usage for missing usage", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], JSON.stringify({ event: "request_complete", message: "PRIVATE_PAYLOAD" }));
  write(USAGE_PATH, JSON.stringify({ event: "token_usage", ai_credits_total: 0.39811 }));
  write("agent-stdio.log", JSON.stringify({ ...captured.terminalError, message: `${captured.terminalError.message} PRIVATE_PAYLOAD` }));
  assert.throws(
    () => verifyCreditLimit(directory),
    error => /below cap 1/.test(error.message) && !error.message.includes("PRIVATE_PAYLOAD")
  );
});

test("invalid structured rejection fields are diagnosed without becoming passing evidence", t => {
  const { directory, write } = fixture(t);
  write(LOG_PATHS[0], JSON.stringify({ ...rejection, max_ai_credits: 2 }));
  assert.throws(() => verifyCreditLimit(directory), /1 credit rejection records had invalid cap, usage or hard_cap fields/);
});
