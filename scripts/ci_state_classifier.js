#!/usr/bin/env node
"use strict";

const fs = require("fs");
const { execFileSync } = require("child_process");
const assert = require("assert");

const FAIL_VALUES = new Set(["FAILURE", "ERROR", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED"]);
const PENDING_VALUES = new Set(["PENDING", "QUEUED", "IN_PROGRESS", "REQUESTED", "WAITING"]);
const POLICY_PATTERNS = [/\bcla\b/i, /license\/cla/i, /code[- ]?owners/i, /\bdco\b/i, /\bpolicy\b/i, /\bcompliance\b/i, /signed[- ]off/i];

function norm(value) {
  if (value === null || value === undefined) {
    return "";
  }
  return String(value).trim().toUpperCase();
}

function toChecks(raw) {
  if (!Array.isArray(raw)) {
    throw new Error("Expected status rollup array");
  }
  return raw.map(item => {
    const name = item.name || item.context || item.__typename || "unknown";
    const conclusion = norm(item.conclusion ?? item.state);
    const status = norm(item.status ?? item.state);
    return { name: String(name), conclusion, status };
  });
}

function isPolicy(check) {
  return POLICY_PATTERNS.some(pattern => pattern.test(check.name));
}

function classify(checks) {
  if (checks.length === 0) {
    return "no_checks";
  }

  const failing = checks.filter(check => FAIL_VALUES.has(check.conclusion) || FAIL_VALUES.has(check.status));
  if (failing.length > 0) {
    return failing.every(isPolicy) ? "policy_blocked" : "failed";
  }

  const pending = checks.filter(check => PENDING_VALUES.has(check.status) || PENDING_VALUES.has(check.conclusion));
  if (pending.length > 0) {
    return pending.every(isPolicy) ? "policy_blocked" : "pending";
  }

  return "passed";
}

function loadStatusRollup(path) {
  const payload = JSON.parse(fs.readFileSync(path, "utf8"));
  if (Array.isArray(payload)) {
    return payload;
  }
  if (payload && Array.isArray(payload.statusCheckRollup)) {
    return payload.statusCheckRollup;
  }
  throw new Error("Expected JSON list or object with statusCheckRollup list");
}

function fetchStatusRollup(repo, pr) {
  const output = execFileSync("gh", ["pr", "view", "--repo", repo, String(pr), "--json", "statusCheckRollup,url"], { encoding: "utf8" });
  const payload = JSON.parse(output);
  if (!payload || !Array.isArray(payload.statusCheckRollup)) {
    return [];
  }
  return payload.statusCheckRollup;
}

function report(repo, pr, checks) {
  return {
    repo,
    pr,
    classification: classify(checks),
    check_count: checks.length,
    checks: checks.map(check => ({
      name: check.name,
      conclusion: check.conclusion,
      status: check.status,
      is_policy_check: isPolicy(check),
    })),
  };
}

function selfTest() {
  assert.equal(classify([]), "no_checks");
  assert.equal(classify([{ name: "unit", conclusion: "SUCCESS", status: "COMPLETED" }]), "passed");
  assert.equal(classify([{ name: "build", conclusion: "FAILURE", status: "COMPLETED" }]), "failed");
  assert.equal(classify([{ name: "license/cla", conclusion: "", status: "QUEUED" }]), "policy_blocked");
  assert.equal(classify([{ name: "tests", conclusion: "", status: "IN_PROGRESS" }]), "pending");
  process.stdout.write(`${JSON.stringify({ self_test: "ok" })}\n`);
}

function parseArgs(argv) {
  const args = {
    repo: "",
    pr: null,
    input: "",
    selfTest: false,
    pretty: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--repo") {
      index += 1;
      args.repo = argv[index] || "";
      continue;
    }
    if (token === "--pr") {
      index += 1;
      const value = argv[index];
      if (!value || Number.isNaN(Number(value))) {
        throw new Error("--pr requires an integer");
      }
      args.pr = Number(value);
      continue;
    }
    if (token === "--input") {
      index += 1;
      args.input = argv[index] || "";
      continue;
    }
    if (token === "--self-test") {
      args.selfTest = true;
      continue;
    }
    if (token === "--pretty") {
      args.pretty = true;
      continue;
    }
    if (token === "--help" || token === "-h") {
      process.stdout.write(
        [
          "Usage:",
          "  scripts/ci_state_classifier.js --repo owner/repo --pr 123",
          "  scripts/ci_state_classifier.js --input status.json [--repo owner/repo] [--pr 123]",
          "  scripts/ci_state_classifier.js --self-test",
          "",
          "Outputs one classification: passed, failed, pending, no_checks, policy_blocked",
        ].join("\n") + "\n"
      );
      process.exit(0);
    }
    throw new Error(`Unknown argument: ${token}`);
  }

  return args;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.selfTest) {
    selfTest();
    return;
  }

  let raw = [];
  if (args.input) {
    raw = loadStatusRollup(args.input);
  } else {
    if (!args.repo || args.pr === null) {
      throw new Error("--repo and --pr are required unless --input or --self-test is used");
    }
    raw = fetchStatusRollup(args.repo, args.pr);
  }

  const checks = toChecks(raw);
  const output = report(args.repo || "fixture", args.pr, checks);
  if (args.pretty) {
    process.stdout.write(`${JSON.stringify(output, null, 2)}\n`);
    return;
  }
  process.stdout.write(`${JSON.stringify(output)}\n`);
}

try {
  main();
} catch (error) {
  process.stderr.write(`${error.message}\n`);
  process.exit(1);
}
