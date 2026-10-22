// @ts-check

// Replays the Copilot startup model/endpoint mismatch from run 38105198559
// (workflow routing-test-pr67487-mismatch) through the real harness, detector,
// collector and failure handler. Every stage after the harness reads the files
// the harness wrote, so a renamed or dropped record field, a moved record path,
// or a harness that spawns the CLI fails this test. The fixture is shared with
// pkg/cli/model_endpoint_mismatch_replay_test.go.

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { spawnSync } from "child_process";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { fileURLToPath } from "url";

const require = createRequire(import.meta.url);
const fixtureDir = fileURLToPath(new URL("../../../pkg/cli/testdata/model_endpoint_mismatch_startup/", import.meta.url));
const fixtureReflectPath = path.join(fixtureDir, "awf-reflect.json");
const fixtureRecord = JSON.parse(fs.readFileSync(path.join(fixtureDir, "model-endpoint-mismatch.json"), "utf8"));
const fixtureSessionEvent = JSON.parse(fs.readFileSync(path.join(fixtureDir, "aw_session.jsonl"), "utf8").trim());
const harnessDir = path.dirname(require.resolve("./copilot_harness.cjs"));

/**
 * Run the real copilot_harness.cjs with /reflect stubbed to the fixture and a stub CLI that records each spawn.
 * Mirrors the compiled agent step: stdout and stderr go to agent-stdio.log and the exit code to agent_execution_exit_code.txt.
 * @param {string} rootDir
 * @param {string} wireAPI
 */
function runHarness(rootDir, wireAPI) {
  const preloadPath = path.join(rootDir, "reflect-preload.cjs");
  const stubPath = path.join(rootDir, "copilot-stub.cjs");
  const callsPath = path.join(rootDir, "copilot-stub-calls.jsonl");
  const configPath = path.join(rootDir, "awf-config.json");
  fs.writeFileSync(configPath, JSON.stringify({ network: { allowDomains: [] } }));
  fs.writeFileSync(
    preloadPath,
    `const fs = require("fs");
const reflect = require(${JSON.stringify(require.resolve("./awf_reflect.cjs"))});
reflect.fetchAWFReflect = async () => ({ ok: true, reflectData: JSON.parse(fs.readFileSync(${JSON.stringify(fixtureReflectPath)}, "utf8")) });`
  );
  fs.writeFileSync(stubPath, `require("fs").appendFileSync(${JSON.stringify(callsPath)}, JSON.stringify(process.argv.slice(2)) + "\\n");`);
  const result = spawnSync(process.execPath, ["--require", preloadPath, "copilot_harness.cjs", process.execPath, stubPath], {
    cwd: harnessDir,
    env: {
      ...process.env,
      GH_AW_HARNESS_INITIAL_DELAY_MS: "1",
      GH_AW_HARNESS_MAX_DELAY_MS: "1",
      GH_AW_TMP_DIR: rootDir,
      GH_AW_AWF_CONFIG_PATH: configPath,
      GH_AW_SAFE_OUTPUTS: path.join(rootDir, "safe-outputs.jsonl"),
      GITHUB_OUTPUT: path.join(rootDir, "harness-github-output"),
      GITHUB_STEP_SUMMARY: path.join(rootDir, "harness-step-summary"),
      AWF_REFLECT_ENABLED: "1",
      GH_AW_MODEL_ROUTING: "0",
      COPILOT_MODEL: "gpt-5.6-luna",
      COPILOT_PROVIDER_WIRE_API: wireAPI,
    },
    encoding: "utf8",
    timeout: 15000,
  });
  fs.writeFileSync(path.join(rootDir, "agent-stdio.log"), `${result.stdout}${result.stderr}`);
  fs.writeFileSync(path.join(rootDir, "agent_execution_exit_code.txt"), String(result.status));
  const calls = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\n").filter(Boolean) : [];
  return { result, calls };
}

/**
 * Run the real detector against the harness output and return its GITHUB_OUTPUT lines as a map.
 * @param {string} rootDir
 */
async function runDetector(rootDir) {
  const outputFile = path.join(rootDir, "detector-github-output");
  vi.stubEnv("GITHUB_OUTPUT", outputFile);
  vi.stubEnv("GH_AW_ENGINE_ID", "copilot");
  vi.stubEnv("GH_AW_ENGINE_INTERNAL_LOGS_DIR", "");
  const stderr = vi.spyOn(process.stderr, "write").mockReturnValue(true);
  try {
    await require("./detect_agent_errors.cjs").main({ rootDir });
  } finally {
    stderr.mockRestore();
  }
  return Object.fromEntries(
    fs
      .readFileSync(outputFile, "utf8")
      .split("\n")
      .filter(line => line.includes("="))
      .map(line => [line.slice(0, line.indexOf("=")), line.slice(line.indexOf("=") + 1)])
  );
}

/**
 * Run the real failure handler with the real templates and a mocked GitHub API.
 * @param {string} rootDir
 * @param {Record<string, string>} detectorOutputs
 * @param {number | null} exitCode
 * @param {unknown} agentOutput
 */
async function runFailureHandler(rootDir, detectorOutputs, exitCode, agentOutput) {
  const agentOutputPath = path.join(rootDir, "agent_output.json");
  fs.writeFileSync(agentOutputPath, JSON.stringify(agentOutput));
  vi.stubEnv("GH_AW_AGENT_OUTPUT", agentOutputPath);
  vi.stubEnv("GH_AW_UNIFIED_SESSION", "");
  vi.stubEnv("GH_AW_AGENT_CONCLUSION", exitCode === 0 ? "success" : "failure");
  vi.stubEnv("GH_AW_FAILURE_REPORT_AS_ISSUE", "true");
  vi.stubEnv("GH_AW_MODEL_ENDPOINT_MISMATCH_ERROR", detectorOutputs.model_endpoint_mismatch_error);
  const create = vi.fn().mockResolvedValue({ data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_101" } });
  const createComment = vi.fn().mockResolvedValue({ data: { id: 1001 } });
  global.github = {
    rest: {
      search: { issuesAndPullRequests: vi.fn(async () => ({ data: { total_count: 0, items: [] } })) },
      issues: { create, createComment },
      pulls: { get: vi.fn() },
      actions: { listJobsForWorkflowRun: vi.fn() },
    },
    paginate: vi.fn(async () => [{ name: "agent", conclusion: "failure", steps: [{ name: "Setup agent", conclusion: "failure" }] }]),
    graphql: vi.fn(),
  };
  vi.resetModules();
  await require("./handle_agent_failure.cjs").main();
  return { create, createComment };
}

describe("Copilot startup model/endpoint mismatch replay (run 38105198559)", () => {
  /** @type {string} */
  let rootDir;

  beforeEach(() => {
    rootDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-endpoint-mismatch-replay-"));
    const promptsDir = path.join(rootDir, "gh-aw", "prompts");
    fs.mkdirSync(promptsDir, { recursive: true });
    fs.cpSync(fileURLToPath(new URL("../md", import.meta.url)), promptsDir, { recursive: true });
    vi.stubEnv("RUNNER_TEMP", rootDir);
    vi.stubEnv("GITHUB_WORKSPACE", rootDir);
    vi.stubEnv("GH_AW_WORKFLOW_NAME", "routing-test-pr67487-mismatch");
    vi.stubEnv("GH_AW_WORKFLOW_ID", "routing-test-pr67487-mismatch");
    vi.stubEnv("GH_AW_RUN_URL", "https://github.com/owner/repo/actions/runs/38105198559");
    vi.stubEnv("GITHUB_HEAD_REF", "feature/model-endpoint-mismatch-replay");
    vi.stubEnv("GITHUB_REF_NAME", "feature/model-endpoint-mismatch-replay");
    global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn(), setFailed: vi.fn() };
    global.context = { repo: { owner: "owner", repo: "repo" } };
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    delete global.core;
    delete global.github;
    delete global.context;
    fs.rmSync(rootDir, { recursive: true, force: true });
  });

  it("stops before any model request and carries the harness record to the failure issue", async () => {
    // Check 1: the harness stops before any model request.
    const { result, calls } = runHarness(rootDir, "completions");
    expect(result.status).toBe(1);
    expect(calls).toHaveLength(0);
    expect(result.stderr).toContain(`[copilot-harness] ${fixtureRecord.detail} ${fixtureRecord.fix}`);
    expect(result.stderr).not.toMatch(/attempt \d+:/);
    expect(result.stderr).not.toMatch(/retry|--continue/i);

    // Check 2: the record the harness wrote equals the run's record, and the later stages classify it.
    const recordPath = path.join(rootDir, "agent", "model-endpoint-mismatch.json");
    expect(JSON.parse(fs.readFileSync(recordPath, "utf8"))).toEqual(fixtureRecord);

    const detectorOutputs = await runDetector(rootDir);
    expect(detectorOutputs.model_endpoint_mismatch_error).toBe("true");

    const { writeUnifiedSession } = require("./unified_session.cjs");
    const events = writeUnifiedSession({ rootDir, engine: "copilot", warn() {} });
    const mismatches = events.filter(event => event.type === "model_endpoint.mismatch");
    expect(mismatches).toEqual([fixtureSessionEvent]);
    const execution = events.find(event => event.type === "agent.execution");
    expect(execution?.data.categories).toContain("model_endpoint_mismatch_error");
    expect(execution?.data.exitCode).toBe(1);

    // Check 3: the failure issue states the cause and the fix.
    const agentOutput = {
      items: [{ type: "report_incomplete", reason: "engine_driver_failure", failureCause: "engine_outage", driverExitCode: 1, retryCount: 0, details: "Failure classification: engine_outage\nLast engine error type: unknown" }],
      errors: [],
      collectorEmptyOutputCause: "engine_driver_failure",
      collectorFailureCause: "engine_outage",
      collectorDriverExitCode: 1,
      collectorRetryCount: 0,
    };
    const { create } = await runFailureHandler(rootDir, detectorOutputs, result.status, agentOutput);
    expect(create).toHaveBeenCalledOnce();
    const { title, body } = create.mock.calls[0][0];
    expect(title).toContain("model/endpoint mismatch");
    expect(body).toContain("**Model/Endpoint Mismatch**");
    for (const line of [
      "Phase: startup",
      "Configured model: gpt-5.6-luna",
      "Resolved model: gpt-5.6-luna",
      "Wire API: completions (source: override)",
      "Supported endpoints: /responses, ws:/responses",
      `Cause: ${fixtureRecord.detail}`,
      `Fix: ${fixtureRecord.fix}`,
    ]) {
      expect(body).toContain(line);
    }
    expect(body).toContain("failure_categories: model_endpoint_mismatch");
    expect(body).not.toContain("engine_outage");
  });

  it("starts the CLI once and reports no mismatch when the wire API matches /reflect", async () => {
    // Check 4: negative control with the same fixture and environment.
    const { result, calls } = runHarness(rootDir, "responses");
    expect(result.status).toBe(0);
    expect(calls).toHaveLength(1);
    expect(result.stderr).toContain("COPILOT_PROVIDER_WIRE_API already set to responses — source=override (verified against AWF /reflect)");
    expect(fs.existsSync(path.join(rootDir, "agent", "model-endpoint-mismatch.json"))).toBe(false);

    const detectorOutputs = await runDetector(rootDir);
    expect(detectorOutputs.model_endpoint_mismatch_error).toBe("false");

    const { writeUnifiedSession } = require("./unified_session.cjs");
    const events = writeUnifiedSession({ rootDir, engine: "copilot", warn() {} });
    expect(events.filter(event => event.type === "model_endpoint.mismatch")).toEqual([]);

    const { create, createComment } = await runFailureHandler(rootDir, detectorOutputs, result.status, { items: [{ type: "noop", message: "Nothing to do" }], errors: [] });
    expect(create).not.toHaveBeenCalled();
    expect(createComment).not.toHaveBeenCalled();
  });
});
