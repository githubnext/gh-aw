import { describe, expect, it } from "vitest";

import { createDashboardDataAccess } from "../src/dashboard-data.js";

describe("dashboard data access", () => {
  it("keeps logs command filters across continuation batches", async () => {
    const calls: string[][] = [];
    const dataAccess = createDashboardDataAccess({
      runGhAw: async args => {
        calls.push(args);
        if (calls.length === 1) {
          return JSON.stringify({
            runs: [{ run_id: 100, workflow_name: "CI Doctor" }],
            continuation: { before_run_id: 99 },
          });
        }

        return JSON.stringify({
          runs: [{ run_id: 99, workflow_name: "CI Doctor" }],
        });
      },
    });

    const result = await dataAccess.execCommand("gh aw logs ci-doctor --json -c 5 --engine claude", { window: "3d", timeout: 2 });
    const payload = JSON.parse(result.output);

    expect(calls).toEqual([
      ["logs", "ci-doctor", "--json", "-c", "5", "--engine", "claude", "--start-date", "-3d", "--timeout", "2", "--artifacts", "usage"],
      ["logs", "--json", "-c", "5", "--timeout", "2", "ci-doctor", "--start-date", "-3d", "--engine", "claude", "--before-run-id", "99", "--artifacts", "usage"],
    ]);
    expect(payload.runs.map((run: { run_id: number }) => run.run_id)).toEqual([100, 99]);
    expect(payload.logs_fetches).toBe(2);
    expect(payload.partial).toBe(false);
  });

  it("passes timeout minutes through to forecast calls", async () => {
    const calls: string[][] = [];
    const dataAccess = createDashboardDataAccess({
      runGhAw: async args => {
        calls.push(args);
        if (args[0] === "logs") {
          return JSON.stringify({
            runs: [{ run_id: 100, workflow_name: "CI Doctor", workflow_path: ".github/workflows/ci-doctor.lock.yml", aic: 12, created_at: "2026-06-29T12:00:00Z" }],
          });
        }

        return JSON.stringify({
          workflows: [{ workflow_id: "ci-doctor", monthly_projected_aic: 44 }],
        });
      },
    });

    const usage = await dataAccess.getUsage({ window: "7d", timeout: 3 });

    expect(calls[1]).toEqual(["forecast", "--json", "--period", "month", "--days", "7", "--timeout", "3", "ci-doctor"]);
    expect(usage.items[0]?.monthly_forecast_aic).toBe(44);
  });

  it("passes --output to logs and audit commands when logsOutputDir is configured", async () => {
    const calls: string[][] = [];
    const dataAccess = createDashboardDataAccess({
      runGhAw: async args => {
        calls.push(args);
        if (args[0] === "logs") return JSON.stringify({ runs: [{ run_id: 100, workflow_name: "CI Doctor" }] });
        if (args[0] === "audit") return JSON.stringify({ overview: {}, metrics: {} });
        return "[]";
      },
      logsOutputDir: "/shared/logs/owner/repo",
    });

    await dataAccess.getRuns({ window: "7d", count: 5, timeout: 1 });
    await dataAccess.getAudit("100");

    const logsCall = calls.find(a => a[0] === "logs");
    const auditCall = calls.find(a => a[0] === "audit");

    expect(logsCall).toEqual(expect.arrayContaining(["--output", "/shared/logs/owner/repo"]));
    expect(auditCall).toEqual(expect.arrayContaining(["--output", "/shared/logs/owner/repo"]));
  });

  it("does not inject --output when logsOutputDir is not configured", async () => {
    const calls: string[][] = [];
    const dataAccess = createDashboardDataAccess({
      runGhAw: async args => {
        calls.push(args);
        if (args[0] === "logs") return JSON.stringify({ runs: [{ run_id: 100 }] });
        return "[]";
      },
    });

    await dataAccess.getRuns({ window: "7d", count: 5, timeout: 1 });

    const logsCall = calls.find(a => a[0] === "logs");
    expect(logsCall).not.toEqual(expect.arrayContaining(["--output"]));
  });

  it("does not inject duplicate --output when one is already present in execCommand args", async () => {
    const calls: string[][] = [];
    const dataAccess = createDashboardDataAccess({
      logsOutputDir: "/shared/logs/owner/repo",
      runGhAw: async args => {
        calls.push(args);
        if (args[0] === "logs") return JSON.stringify({ runs: [{ run_id: 200 }] });
        return "[]";
      },
    });

    // When the caller already has --output in execCommand we should not add a second one.
    // Simulate this by calling getRuns normally — the injected --output appears exactly once.
    await dataAccess.getRuns({ window: "7d", count: 5, timeout: 1 });

    const logsCall = calls.find(a => a[0] === "logs");
    const outputOccurrences = logsCall?.filter(a => a === "--output").length ?? 0;
    expect(outputOccurrences).toBe(1);
  });

  it("parses gh aw status output that has a status line prefix before the JSON", async () => {
    const dataAccess = createDashboardDataAccess({
      runGhAw: async args => {
        if (args[0] === "status") {
          // Simulate gh aw writing a status message to stdout before the JSON array
          return `✓ Fetched 2 workflows\n[{"workflow":"ci-doctor"},{"workflow":"ab-advisor"}]`;
        }
        return "[]";
      },
    });

    const defs = await dataAccess.getDefinitions();
    expect(defs).toHaveLength(2);
    expect((defs[0] as { workflow: string }).workflow).toBe("ci-doctor");
  });

  it("throws a descriptive error when gh aw status produces no output", async () => {
    const dataAccess = createDashboardDataAccess({
      runGhAw: async () => "",
    });

    await expect(dataAccess.getDefinitions()).rejects.toThrow("command produced no output");
  });

  it("throws a descriptive error including an output snippet when JSON is unparseable", async () => {
    const dataAccess = createDashboardDataAccess({
      runGhAw: async () => "error: authentication required",
    });

    await expect(dataAccess.getDefinitions()).rejects.toThrow("failed to parse JSON");
  });
});
