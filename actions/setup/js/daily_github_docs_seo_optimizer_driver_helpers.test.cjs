import { describe, expect, it, vi } from "vitest";

import {
  REQUEST_COUNT,
  buildFinalReportingPrompt,
  buildIsolatedPermissionConfig,
  parseJSONFromCopilotOutput,
  runDailyGitHubDocsSEOOptimizerDriver,
  validateRequestsPayload,
} from "../../../.github/drivers/daily_github_docs_seo_optimizer_driver.ts";

describe("daily_github_docs_seo_optimizer_driver.ts", () => {
  it("parses fenced JSON output", () => {
    expect(parseJSONFromCopilotOutput('```json\n{"ok":true}\n```', "test")).toEqual({ ok: true });
  });

  it("validates exactly ten distinct requests", () => {
    const requests = Array.from({ length: REQUEST_COUNT }, (_, index) => `request ${index + 1}`);
    expect(validateRequestsPayload({ requests })).toEqual(requests);
    expect(() => validateRequestsPayload({ requests: [...requests.slice(0, 9), "request 1"] })).toThrow(/distinct/);
  });

  it("builds a final prompt that embeds the structured dataset", () => {
    const prompt = buildFinalReportingPrompt("workflow prompt", {
      request_count: 1,
      requests: ["request 1"],
      evaluations: [],
    });

    expect(prompt).toContain("workflow prompt");
    expect(prompt).toContain("Driver-Supplied Baseline Dataset");
    expect(prompt).toContain('"request_count": 1');
  });

  it("uses isolated sessions for data collection and workflow permissions for final reporting", async () => {
    const requests = Array.from({ length: REQUEST_COUNT }, (_, index) => `request ${index + 1}`);
    const runWithCopilotSDKImpl = vi.fn().mockResolvedValueOnce({
      exitCode: 0,
      output: JSON.stringify({ requests }),
    });

    for (const request of requests) {
      runWithCopilotSDKImpl.mockResolvedValueOnce({
        exitCode: 0,
        output: JSON.stringify({
          request,
          options: [
            { rank: 1, name: "Option 1", reason: "reason 1" },
            { rank: 2, name: "Option 2", reason: "reason 2" },
            { rank: 3, name: "Option 3", reason: "reason 3" },
          ],
          documentation_pages: [],
        }),
      });
    }

    runWithCopilotSDKImpl.mockResolvedValueOnce({
      exitCode: 0,
      output: "final report",
      hasOutput: true,
      durationMs: 1,
    });

    const result = await runDailyGitHubDocsSEOOptimizerDriver({
      env: {
        GH_AW_PROMPT: "/tmp/workflow-prompt.txt",
        COPILOT_SDK_URI: "http://127.0.0.1:1234",
        COPILOT_CONNECTION_TOKEN: "token",
        GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON: JSON.stringify({
          model: "gpt-5.4",
          providers: [{ name: "copilot", type: "copilot" }],
          models: [{ id: "gpt-5.4", provider: "copilot" }],
        }),
        GH_AW_COPILOT_SDK_SERVER_ARGS: JSON.stringify(["--allow-tool", "shell(git)", "--allow-tool", "safeoutputs_create_issue"]),
      },
      fsModule: {
        readFileSync: vi.fn(() => "workflow prompt"),
      },
      parseMultiProviderJsonImpl: vi.fn(value => JSON.parse(value)),
      parsePermissionConfigImpl: vi.fn(() => ({ allowedTools: ["shell(git)", "safeoutputs_create_issue"] })),
      applyModelFallbackImpl: vi.fn(() => "gpt-5.4"),
      runWithCopilotSDKImpl,
    });

    expect(result.output).toBe("final report");
    expect(runWithCopilotSDKImpl).toHaveBeenCalledTimes(REQUEST_COUNT + 2);

    const isolatedPermissionConfig = buildIsolatedPermissionConfig();
    expect(runWithCopilotSDKImpl.mock.calls[0][0].permissionConfig).toEqual(isolatedPermissionConfig);
    expect(runWithCopilotSDKImpl.mock.calls[1][0].permissionConfig).toEqual(isolatedPermissionConfig);
    expect(runWithCopilotSDKImpl.mock.calls[REQUEST_COUNT + 1][0].permissionConfig).toEqual({
      allowedTools: ["shell(git)", "safeoutputs_create_issue"],
    });

    const finalPrompt = runWithCopilotSDKImpl.mock.calls[REQUEST_COUNT + 1][0].prompt;
    expect(finalPrompt).toContain("workflow prompt");
    expect(finalPrompt).toContain("Driver-Supplied Baseline Dataset");
    expect(finalPrompt).toContain('"request_count": 10');
  });
});
