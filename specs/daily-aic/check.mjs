#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const directory = dirname(fileURLToPath(import.meta.url));
const root = resolve(directory, "../..");
const jarInput = process.env.TLA2TOOLS_JAR;
const java = process.env.JAVA_BIN || "java";
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const jarHash = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88";
if (!jarInput) throw new Error("Set TLA2TOOLS_JAR to the official v1.7.4 tla2tools.jar.");
const jar = resolve(jarInput);
if (digest(readFileSync(jar)) !== jarHash) throw new Error("TLA+ tools checksum mismatch.");
const version = spawnSync(java, ["-version"], { encoding: "utf8", timeout: 10000 });
if (version.error || version.status !== 0) throw new Error(`Java unavailable: ${version.error || version.stderr}`);

const measured = [
  "specs/daily-aic/DailyAIC.tla",
  "specs/daily-aic/DailyAIC.cfg",
  "specs/daily-aic/Aggregation.cfg",
  "specs/daily-aic/check.mjs",
  "docs/src/content/docs/specs/ai-credits-specification.md",
  "actions/setup/js/check_daily_aic_workflow_guardrail.cjs",
  "actions/setup/js/daily_aic_scan.cjs",
  "actions/setup/js/daily_aic_api_budget.cjs",
  "actions/setup/js/daily_aic_cache_helpers.cjs",
  "actions/setup/js/daily_aic_component_coverage.cjs",
  "actions/setup/js/daily_aic_workflow_helpers.cjs",
  "actions/setup/js/daily_aic_repo_memory_ledger.cjs",
  "actions/setup/js/restore_aic_scan_cache.cjs",
  "actions/setup/js/artifact_client.cjs",
];
const hashes = () => Object.fromEntries(measured.map(file => [file, digest(readFileSync(join(root, file)))]));
const sourceHashes = hashes();
const revision = spawnSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8", timeout: 10000 });
if (revision.error || revision.status !== 0) throw new Error("Cannot identify the source revision.");
const output = process.argv[2] ? resolve(process.argv[2]) : mkdtempSync(join(tmpdir(), "daily-aic-formal-"));
mkdirSync(output, { recursive: true });
const reportPath = join(output, "report.json");
writeFileSync(reportPath, JSON.stringify({ status: "incomplete", sourceHashes }, null, 2) + "\n");
const model = readFileSync(join(directory, "DailyAIC.tla"), "utf8");
const baseline = readFileSync(join(directory, "DailyAIC.cfg"), "utf8");
const aggregate = readFileSync(join(directory, "Aggregation.cfg"), "utf8");
writeFileSync(join(output, "DailyAIC.tla"), model);

const sanyArgs = ["-cp", jar, "tla2sany.SANY", join(output, "DailyAIC.tla")];
const sany = spawnSync(java, sanyArgs, { cwd: output, encoding: "utf8", timeout: 30000 });
const sanyLog = `${sany.stdout || ""}${sany.stderr || ""}`;
writeFileSync(join(output, "sany.log"), sanyLog);
if (sany.error || sany.status !== 0 || !sanyLog.includes("Semantic processing of module DailyAIC") || /Semantic errors|Fatal errors|\*\*\* Errors/.test(sanyLog)) {
  throw new Error(`TLA+ parsing/semantic validation failed; see ${join(output, "sany.log")}.`);
}

function check(name, config, invariant = null) {
  const configPath = join(output, `${name}.cfg`);
  const logPath = join(output, `${name}.log`);
  writeFileSync(configPath, config);
  const args = ["-Xmx1g", "-cp", jar, "tlc2.TLC", "-workers", "2", "-seed", "1", "-metadir", join(output, `${name}-states`), "-config", configPath, join(output, "DailyAIC.tla")];
  const result = spawnSync(java, args, { cwd: output, encoding: "utf8", timeout: 120000, maxBuffer: 16 * 1024 * 1024 });
  const log = `${result.stdout || ""}${result.stderr || ""}`;
  writeFileSync(logPath, log);
  const stateCounts = [...log.matchAll(/([\d,]+) states? generated, ([\d,]+) distinct states? found, ([\d,]+) states? left on queue/g)].at(-1);
  const states = stateCounts
    ? {
        generated: Number(stateCounts[1].replaceAll(",", "")),
        distinct: Number(stateCounts[2].replaceAll(",", "")),
        queued: Number(stateCounts[3].replaceAll(",", "")),
      }
    : null;
  const expected = invariant
    ? result.status === 12 && log.includes(`Invariant ${invariant} is violated.`) && log.includes("The behavior up to this point is:")
    : result.status === 0 && log.includes("Model checking completed. No error has been found.") && states?.queued === 0 && states.distinct > 0;
  if (result.error || !expected) throw new Error(`Unexpected TLC result for ${name}; see ${logPath}.`);
  console.log(`${name}: ${invariant ? `expected ${invariant} counterexample` : "exhaustive safety/liveness PASS"} (${states?.distinct ?? "unknown"} distinct states)`);
  return { name, command: [java, ...args], configHash: digest(config), exit: result.status, expectedInvariant: invariant, states, log: logPath };
}

const results = [check("policy", baseline), check("aggregation", aggregate)];
const controls = {
  partial_history: "CompleteWindowRequired",
  estimate_quota: "NoForbiddenEstimates",
  reuse_estimate: "RecordedCacheOnly",
  trust_issue_branch: "CacheProducerTrusted",
  double_count: "TotalsReconcile",
  resurrect_detection_summary: "RecordedEvidence",
};
for (const [fault, invariant] of Object.entries(controls)) {
  results.push(check(`broken-${fault}`, baseline.replace('Fault = "none"', `Fault = "${fault}"`), invariant));
}
const accountingWitnesses = [
  "raw",
  "agent_jsonl",
  "agent_json",
  "detection_jsonl",
  "response_credits",
  "explicit_credits",
  "priced_tokens",
  "retained_prior_attempt",
  "component_skipped",
  "runner_not_assigned",
  "execution_not_started",
  "empty_detection",
  "failed_agent_empty",
  "legacy_startup_failure",
  "legacy_sample_replay",
  "failed_agent_missing",
  "failed_evals",
  "explicit_zero",
  "missing_usage",
  "missing_component",
  "unreadable_preferred",
  "malformed",
  "unknown_pricing",
  "stale_artifact",
  "unverified_rerun",
  "conflicting_request",
  "all_jobs_skipped",
  "no_billable_jobs",
];
const policyWitnesses = [
  "estimated",
  "cache_hit",
  "history_fallback",
  "retry_recovery",
  "ledger",
  "ledger_empty",
  "issues_cache",
  "issue_comment_cache",
  "estimated_reinspection",
  "rerun_reinspection",
  "missing_maximum",
  "quota_stop",
  "permission_stop",
  "under_budget",
  "exceeded",
  "disabled",
  "skipped",
  "partial_snapshot",
  "mixed_sources",
];
for (const witness of [...accountingWitnesses, ...policyWitnesses]) {
  const template = ["partial_snapshot", "mixed_sources"].includes(witness) ? aggregate : baseline;
  const config = `${template.replace('Witness = "none"', `Witness = "${witness}"`)}\nINVARIANT WitnessNotReached\n`;
  results.push(check(`witness-${witness}`, config, "WitnessNotReached"));
}
if (JSON.stringify(hashes()) !== JSON.stringify(sourceHashes)) throw new Error("Measured sources changed during validation; rerun the checker.");
const finalRevision = spawnSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8", timeout: 10000 });
if (finalRevision.error || finalRevision.status !== 0 || finalRevision.stdout !== revision.stdout) throw new Error("Source revision changed during validation; rerun the checker.");
writeFileSync(
  reportPath,
  JSON.stringify(
    {
      status: "passed",
      revisionAnchor: revision.stdout.trim(),
      sourceHashes,
      javaVersion: version.stderr.trim(),
      jarHash,
      sany: { command: [java, ...sanyArgs], exit: sany.status },
      results,
    },
    null,
    2
  ) + "\n"
);
console.log(`Validated ${results.length} cases; evidence: ${join(output, "report.json")}`);
