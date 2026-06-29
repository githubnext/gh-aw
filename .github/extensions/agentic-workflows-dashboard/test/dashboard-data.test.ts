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
});
