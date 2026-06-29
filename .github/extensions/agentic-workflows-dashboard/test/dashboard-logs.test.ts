import { describe, expect, it } from "vitest";

import { DEFAULT_LOG_TIMEOUT_MINUTES, REPORT_WINDOWS } from "../src/dashboard-config.js";
import { buildLogsArgs, continuationToLogsOptions, logsArgsToOptions, normalizeLogsCommandArgs, normalizeLogsOptions } from "../src/dashboard-logs.js";

describe("dashboard logs helpers", () => {
  it("defaults logs timeouts in minutes", () => {
    const options = normalizeLogsOptions({});

    expect(options.timeout).toBe(DEFAULT_LOG_TIMEOUT_MINUTES);
    expect(buildLogsArgs(options)).toEqual(expect.arrayContaining(["--timeout", String(DEFAULT_LOG_TIMEOUT_MINUTES)]));
  });

  it("accepts a window object in normalizeLogsOptions without falling back to default", () => {
    const windowObj = REPORT_WINDOWS["3d"];
    const options = normalizeLogsOptions({ window: windowObj });

    expect(options.window.id).toBe("3d");
    expect(options.startDate).toBe(windowObj.startDate);
  });

  it("preserves fallback filters when continuing logs pages", () => {
    const initial = normalizeLogsOptions({ window: "3d", count: 5, timeout: 2, engine: "claude", workflowName: "ci-doctor" });
    const next = continuationToLogsOptions({ before_run_id: 90 }, initial);

    expect(next).toMatchObject({
      count: 5,
      timeout: 2,
      engine: "claude",
      workflowName: "ci-doctor",
      beforeRunID: 90,
      startDate: "-3d",
    });
  });

  it("parses logs command args into continuation-ready options", () => {
    const options = logsArgsToOptions(["logs", "ci-doctor", "--json", "-c", "5", "--engine", "claude", "--timeout", "3", "--before-run-id", "99"], { window: "7d" });

    expect(options).toMatchObject({
      workflowName: "ci-doctor",
      count: 5,
      engine: "claude",
      timeout: 3,
      beforeRunID: 99,
      startDate: "-1w",
    });
  });

  it("injects the selected report window and minute timeout when missing", () => {
    expect(normalizeLogsCommandArgs(["logs", "--json"], "3d", 4)).toEqual(["logs", "--json", "--start-date", "-3d", "--timeout", "4", "--artifacts", "usage"]);
  });
});
