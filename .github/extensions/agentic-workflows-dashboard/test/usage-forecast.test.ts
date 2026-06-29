import { describe, expect, it } from "vitest";

import { applyForecastToUsageSummary, buildUsageSummary, forecastDaysForWindow, getForecastMonthlyAIC, normalizeWorkflowID } from "../src/usage-forecast.js";

describe("usage forecast helpers", () => {
  it("normalizes workflow ids from workflow paths", () => {
    expect(normalizeWorkflowID(".github/workflows/ci-doctor.lock.yml")).toBe("ci-doctor");
    expect(normalizeWorkflowID("daily-planner.md")).toBe("daily-planner");
  });

  it("maps dashboard windows onto supported forecast history windows", () => {
    expect(forecastDaysForWindow({ id: "3d" })).toBe(7);
    expect(forecastDaysForWindow({ id: "7d" })).toBe(7);
    expect(forecastDaysForWindow({ id: "1mo" })).toBe(30);
  });

  it("prefers monte carlo p50 monthly forecast when available", () => {
    expect(getForecastMonthlyAIC({ monthly_monte_carlo: { p50_projected_aic: 123.4 }, monthly_projected_aic: 100 })).toBe(123.4);
    expect(getForecastMonthlyAIC({ monthly_projected_aic: 98.2 })).toBe(98.2);
  });

  it("merges logs usage with forecast command output by workflow id", () => {
    const summary = applyForecastToUsageSummary(
      buildUsageSummary(
        [
          {
            workflow_name: "CI Doctor",
            workflow_path: ".github/workflows/ci-doctor.lock.yml",
            aic: 10,
            created_at: "2026-06-28T12:00:00Z",
          },
          {
            workflow_name: "CI Doctor",
            workflow_path: ".github/workflows/ci-doctor.lock.yml",
            aic: 5,
            created_at: "2026-06-29T12:00:00Z",
          },
        ],
        { id: "7d", days: 7 }
      ),
      [
        {
          workflow_id: "ci-doctor",
          monthly_monte_carlo: { p50_projected_aic: 77.7 },
        },
      ]
    );

    expect(summary).toHaveLength(1);
    expect(summary[0]).toMatchObject({
      workflow_id: "ci-doctor",
      workflow_name: "CI Doctor",
      run_count: 2,
      total_aic: 15,
      cost_per_run: 7.5,
      daily_aic: 15 / 7,
      monthly_forecast_aic: 77.7,
      last_run_at: "2026-06-29T12:00:00Z",
    });
  });

  it("can build summary with forecast data in a single call", () => {
    const summary = buildUsageSummary(
      [
        {
          workflow_name: "CI Doctor",
          workflow_path: ".github/workflows/ci-doctor.lock.yml",
          aic: 10,
          created_at: "2026-06-28T12:00:00Z",
        },
        {
          workflow_name: "CI Doctor",
          workflow_path: ".github/workflows/ci-doctor.lock.yml",
          aic: 5,
          created_at: "2026-06-29T12:00:00Z",
        },
      ],
      { id: "7d", days: 7 },
      [
        {
          workflow_id: "ci-doctor",
          monthly_monte_carlo: { p50_projected_aic: 77.7 },
        },
      ]
    );

    expect(summary).toHaveLength(1);
    expect(summary[0]).toMatchObject({
      workflow_id: "ci-doctor",
      workflow_name: "CI Doctor",
      run_count: 2,
      total_aic: 15,
      cost_per_run: 7.5,
      daily_aic: 15 / 7,
      monthly_forecast_aic: 77.7,
      last_run_at: "2026-06-29T12:00:00Z",
    });
  });
});
