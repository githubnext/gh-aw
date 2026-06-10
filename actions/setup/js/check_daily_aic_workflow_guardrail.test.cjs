import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let exports;

describe("check_daily_aic_workflow_guardrail", () => {
  beforeEach(async () => {
    vi.resetModules();
    process.env.GITHUB_EVENT_NAME = "";
    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = "";
    const mod = await import("./check_daily_aic_workflow_guardrail.cjs");
    exports = mod.default || mod;
  });

  afterEach(() => {
    delete process.env.GITHUB_EVENT_NAME;
    delete process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT;
  });

  it("skips workflow_call, repository_dispatch, and workflow_dispatch with aw_context", () => {
    process.env.GITHUB_EVENT_NAME = "workflow_call";
    expect(exports.shouldSkipDailyAICGuardrail()).toBe(true);

    process.env.GITHUB_EVENT_NAME = "repository_dispatch";
    expect(exports.shouldSkipDailyAICGuardrail()).toBe(true);

    process.env.GITHUB_EVENT_NAME = "workflow_dispatch";
    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = '{"item_number":123}';
    expect(exports.shouldSkipDailyAICGuardrail()).toBe(true);

    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = "";
    expect(exports.shouldSkipDailyAICGuardrail()).toBe(false);
  });

  it("matches usage artifacts only", () => {
    expect(exports.matchesGuardrailArtifactName("usage")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("prefix-usage")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("agent")).toBe(false);
    expect(exports.matchesGuardrailArtifactName("detection")).toBe(false);
    expect(exports.matchesGuardrailArtifactName("activation")).toBe(false);
  });

  it("sums AI Credits across multiple JSONL files and usage attributes", () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "daily-guardrail-token-usage-"));
    const nestedDir = path.join(tmpDir, "nested");
    fs.mkdirSync(nestedDir);
    const filePathA = path.join(tmpDir, "token-usage-a.jsonl");
    const filePathB = path.join(nestedDir, "token-usage-b.jsonl");
    fs.writeFileSync(filePathA, [JSON.stringify({ model: "gpt-5.5", aic: 1.25 }), JSON.stringify({ model: "gpt-5.5", aic: 0.75 })].join("\n"), "utf8");
    fs.writeFileSync(
      filePathB,
      [
        JSON.stringify({ usage: { aic: 1.5 } }),
        JSON.stringify({ usage: { aic: 0.5 } }),
        JSON.stringify({ aic: 9, usage: { aic: 0.25 } }),
        JSON.stringify({ ai_credits: 8, usage: { ai_credits: 0.1 } }),
        JSON.stringify({ aiCredits: 7, usage: { aiCredits: 0.15 } }),
        JSON.stringify({ aiCredits: 0.2, usage: { aiCredits: "" } }),
        JSON.stringify({ aic: 0.3, usage: { aic: "" } }),
      ].join("\n"),
      "utf8"
    );

    expect(exports.sumAICFromUsageJSONLFiles(exports.findJSONLFiles(tmpDir))).toBe(5);
  });

  it("computes aggregate AIC statistics for prior runs", () => {
    expect(exports.calculateDailyAICStats([{ aic: 100 }, { aic: 200 }, { aic: 300 }])).toEqual({
      count: 3,
      total: 600,
      average: 200,
      min: 100,
      max: 300,
      stddev: 100,
    });
  });

  it("caps inspection when GitHub API rate limit headroom is low", () => {
    expect(exports.computeMaxInspectableRuns(110)).toBe(0);
    expect(exports.computeMaxInspectableRuns(120)).toBeGreaterThan(0);
  });

  it("formats structured daily AI Credits log messages", () => {
    const message = exports.formatDailyGuardrailLogMessage("Resolved current workflow AI Credits guardrail context", {
      currentRunId: 123,
      workflowId: 456,
      currentAIC: 789,
    });
    const prefix = "[daily-workflow-aic] Resolved current workflow AI Credits guardrail context: ";
    expect(message).toContain(prefix);
    expect(JSON.parse(message.slice(prefix.length))).toEqual({
      currentRunId: 123,
      workflowId: 456,
      currentAIC: 789,
    });
    expect(exports.formatDailyGuardrailLogMessage("Completed AI Credits inspection window")).toBe("[daily-workflow-aic] Completed AI Credits inspection window");
  });

  it("renders a daily AI Credits details summary with stats and prior runs", () => {
    const markdown = exports.renderDailyAICSummary(
      "Nightly triage",
      "copilot-swe-agent[bot]",
      1_500_000,
      [
        {
          id: 11,
          html_url: "https://example.test/runs/11",
          created_at: "2026-05-31T10:00:00Z",
          conclusion: "success",
          aic: 1_200_000,
        },
        {
          id: 10,
          html_url: "https://example.test/runs/10",
          created_at: "2026-05-31T09:00:00Z",
          conclusion: "failure",
          aic: 300_000,
        },
      ],
      {
        remaining: 4321,
        limit: 5000,
        used: 679,
        reset: "2026-05-31T12:00:00.000Z",
      },
      {
        candidateRunsCount: 5,
        inspectedRunsCount: 2,
        truncatedByRateLimit: true,
      }
    );

    expect(markdown).toContain("| 24h total AIC | 1.5M |");
    expect(markdown).toContain("| Threshold | 1.5M |");
    expect(markdown).toContain("| Avg AIC / run | 750K |");
    expect(markdown).toContain("| Std dev AIC | 636.4K |");
    expect(markdown).toContain("| [#11](https://example.test/runs/11) | 2026-05-31T10:00:00Z | success | 1.2M |");
    expect(markdown).toContain("Stopped early to preserve GitHub API rate limit headroom");
    expect(markdown).not.toContain("Guardrail issue:");
  });

  it("main() does not fail the step when GitHub API calls throw", async () => {
    // Simulate a scenario where the GitHub API throws during workflow run lookup.
    // The step should catch the error and NOT rethrow it, keeping daily_effective_workflow_exceeded at "false".
    const coreOutputs = {};
    const coreWarnings = [];
    const mockCore = {
      setOutput: (key, value) => {
        coreOutputs[key] = value;
      },
      info: () => {},
      warning: msg => coreWarnings.push(msg),
    };

    const mockGithub = {
      rest: {
        rateLimit: {
          get: async () => {
            throw new Error("API rate limit exceeded");
          },
        },
        actions: {
          getWorkflowRun: async () => {
            throw new Error("Network error");
          },
          listWorkflowRuns: async () => {
            throw new Error("Unexpected error");
          },
        },
      },
    };

    const mockContext = {
      repo: { owner: "test-owner", repo: "test-repo" },
      runId: 42,
    };

    // Inject globals so the module can use them
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;

    process.env.GH_AW_MAX_DAILY_AI_CREDITS = "1000000";
    process.env.GH_AW_GITHUB_TOKEN = "fake-token";

    try {
      // Should resolve without throwing even though the API calls throw
      await expect(exports.main()).resolves.toBeUndefined();
      // The default "false" output must be set
      expect(coreOutputs["daily_effective_workflow_exceeded"]).toBe("false");
      // A warning must be emitted describing the error
      expect(coreWarnings.some(w => /unexpected error.*skipped/i.test(w))).toBe(true);
    } finally {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.GH_AW_MAX_DAILY_AI_CREDITS;
      delete process.env.GH_AW_GITHUB_TOKEN;
    }
  });

  it("main() logs rate limit consumption delta when guardrail runs without candidate runs", async () => {
    // Verify that fetchAndLogRateLimit is called at the start and end of the guardrail
    // and that a consumption-delta diagnostic log is emitted.
    const coreInfos = [];
    const coreOutputs = {};
    const mockCore = {
      setOutput: (key, value) => {
        coreOutputs[key] = value;
      },
      info: msg => coreInfos.push(msg),
      warning: () => {},
      summary: {
        addDetails: function () {
          return this;
        },
        write: async () => {},
      },
    };

    let rateLimitCallCount = 0;
    const mockGithub = {
      rest: {
        rateLimit: {
          get: async () => {
            rateLimitCallCount += 1;
            const remaining = rateLimitCallCount === 1 ? 4995 : 5000;
            return {
              data: {
                resources: {
                  core: { limit: 5000, remaining, used: 5000 - remaining, reset: Math.floor(Date.now() / 1000) + 3600 },
                },
              },
              headers: {},
            };
          },
        },
        actions: {
          getWorkflowRun: async () => ({
            data: {
              workflow_id: 777,
              actor: { login: "octocat" },
              triggering_actor: { login: "octocat" },
            },
            headers: {},
          }),
          listWorkflowRuns: async () => ({
            data: { workflow_runs: [] },
            headers: {},
          }),
        },
      },
    };

    const mockContext = {
      repo: { owner: "test-owner", repo: "test-repo" },
      runId: 99,
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;

    process.env.GH_AW_MAX_DAILY_AI_CREDITS = "50000";
    process.env.GH_AW_GITHUB_TOKEN = "fake-token";

    try {
      await expect(exports.main()).resolves.toBeUndefined();

      // A consumption-delta log must have been emitted.
      const consumptionLog = coreInfos.find(msg => msg.includes("rate limit consumed by daily AIC guardrail"));
      expect(consumptionLog).toBeDefined();
      expect(consumptionLog).toContain("rateLimitBeforeInspection");
      expect(consumptionLog).toContain("rateLimitAfterInspection");
      expect(consumptionLog).toContain("consumed");
      const detailsPrefix = "[daily-workflow-aic] GitHub API rate limit consumed by daily AIC guardrail: ";
      const details = JSON.parse(consumptionLog.slice(detailsPrefix.length));
      expect(details).toMatchObject({
        rateLimitBeforeInspection: 4995,
        rateLimitAfterInspection: 5000,
        consumed: 0,
      });

      // fetchAndLogRateLimit must have been called at least twice (start + end).
      expect(rateLimitCallCount).toBe(2);
    } finally {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.GH_AW_MAX_DAILY_AI_CREDITS;
      delete process.env.GH_AW_GITHUB_TOKEN;
    }
  });
});
