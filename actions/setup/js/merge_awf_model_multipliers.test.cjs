import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { main, normalizeAliasRows, renderModelAliasSummary } from "./merge_awf_model_multipliers.cjs";

describe("merge_awf_model_multipliers.cjs", () => {
  /** @type {string[]} */
  const tempDirs = [];
  /** @type {string | undefined} */
  let originalStepSummary;

  beforeEach(() => {
    originalStepSummary = process.env.GITHUB_STEP_SUMMARY;
  });

  afterEach(() => {
    for (const tempDir of tempDirs) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    tempDirs.length = 0;
    if (originalStepSummary === undefined) {
      delete process.env.GITHUB_STEP_SUMMARY;
    } else {
      process.env.GITHUB_STEP_SUMMARY = originalStepSummary;
    }
  });

  /**
   * @returns {{ runnerTemp: string, configPath: string, multipliersPath: string }}
   */
  function setupTempFiles() {
    const runnerTemp = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-merge-"));
    const ghAwDir = path.join(runnerTemp, "gh-aw");
    fs.mkdirSync(ghAwDir, { recursive: true });

    const multipliersRoot = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-multipliers-"));
    tempDirs.push(runnerTemp, multipliersRoot);

    return {
      runnerTemp,
      configPath: path.join(ghAwDir, "awf-config.json"),
      multipliersPath: path.join(multipliersRoot, "model_multipliers.json"),
    };
  }

  it("merges normalized multipliers into apiProxy.modelMultipliers", () => {
    const { runnerTemp, configPath, multipliersPath } = setupTempFiles();
    fs.writeFileSync(configPath, JSON.stringify({ apiProxy: { enabled: true } }), "utf8");
    fs.writeFileSync(
      multipliersPath,
      JSON.stringify({
        multipliers: {
          "claude-sonnet-4.6": 1.5,
          "gpt-3.5-turbo": 0,
          "copilot-fast": "2.0",
          bool: true,
        },
      }),
      "utf8"
    );

    main({ runnerTemp, multipliersPath });

    const parsed = JSON.parse(fs.readFileSync(configPath, "utf8"));
    expect(parsed.apiProxy.enabled).toBe(true);
    expect(parsed.apiProxy.modelMultipliers).toEqual({ "claude-sonnet-4.6": 1.5 });
  });

  it("removes modelMultipliers when normalized set is empty", () => {
    const { runnerTemp, configPath, multipliersPath } = setupTempFiles();
    fs.writeFileSync(configPath, JSON.stringify({ apiProxy: { modelMultipliers: { keep: 2 } } }), "utf8");
    fs.writeFileSync(
      multipliersPath,
      JSON.stringify({
        multipliers: {
          invalid: "x",
        },
      }),
      "utf8"
    );

    main({ runnerTemp, multipliersPath });

    const parsed = JSON.parse(fs.readFileSync(configPath, "utf8"));
    expect(parsed.apiProxy.modelMultipliers).toBeUndefined();
  });

  it("warns and leaves config unchanged when multipliers JSON is invalid", () => {
    const { runnerTemp, configPath, multipliersPath } = setupTempFiles();
    const before = JSON.stringify({ apiProxy: { enabled: true } });
    fs.writeFileSync(configPath, before, "utf8");
    fs.writeFileSync(multipliersPath, "{not-json", "utf8");

    /** @type {string[]} */
    const warnings = [];
    main({ runnerTemp, multipliersPath, warn: message => warnings.push(message) });

    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("failed to parse model multipliers file");
    expect(fs.readFileSync(configPath, "utf8")).toBe(before);
  });

  it("renders model aliases to step summary as a details table", () => {
    const { runnerTemp, configPath, multipliersPath } = setupTempFiles();
    const stepSummaryPath = path.join(runnerTemp, "step_summary.md");
    process.env.GITHUB_STEP_SUMMARY = stepSummaryPath;

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        apiProxy: {
          enabled: true,
          models: {
            "": ["claude-sonnet-4.6", "gpt-5.5"],
            sonnet: ["claude-sonnet-4.6"],
          },
        },
      }),
      "utf8"
    );
    fs.writeFileSync(multipliersPath, JSON.stringify({ multipliers: { "claude-sonnet-4.6": 1.5 } }), "utf8");

    main({ runnerTemp, multipliersPath });

    const summary = fs.readFileSync(stepSummaryPath, "utf8");
    expect(summary).toContain("<details>");
    expect(summary).toContain("<summary>AWF model aliases (2)</summary>");
    expect(summary).toContain("| Alias | Resolution order |");
    expect(summary).toContain("| (default) | `claude-sonnet-4.6` → `gpt-5.5` |");
    expect(summary).toContain("| `sonnet` | `claude-sonnet-4.6` |");
    expect(summary).toContain("</details>");
  });

  it("normalizes aliases and escapes table values", () => {
    const rows = normalizeAliasRows({
      sonnet: [" claude|sonnet ", "", 42, " "],
      "": ["gpt-5.5"],
      invalid: "not-array",
    });
    const summary = renderModelAliasSummary(rows);

    expect(rows).toEqual([
      { alias: "sonnet", targets: ["claude|sonnet"] },
      { alias: "", targets: ["gpt-5.5"] },
    ]);
    expect(summary).not.toContain("42");
    expect(summary).toContain("| `sonnet` | `claude\\|sonnet` |");
    expect(summary).toContain("| (default) | `gpt-5.5` |");
  });

  it("returns no alias rows for non-object inputs", () => {
    expect(normalizeAliasRows(null)).toEqual([]);
    expect(normalizeAliasRows(undefined)).toEqual([]);
    expect(normalizeAliasRows([])).toEqual([]);
    expect(normalizeAliasRows("x")).toEqual([]);
    expect(normalizeAliasRows(123)).toEqual([]);
    expect(normalizeAliasRows(() => "x")).toEqual([]);
  });

  it("warns when summary write fails", () => {
    const { runnerTemp, configPath, multipliersPath } = setupTempFiles();
    process.env.GITHUB_STEP_SUMMARY = runnerTemp;
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        apiProxy: { models: { alias: ["claude-sonnet-4.6"] } },
      }),
      "utf8"
    );
    fs.writeFileSync(multipliersPath, JSON.stringify({ multipliers: { "claude-sonnet-4.6": 1.5 } }), "utf8");

    /** @type {string[]} */
    const warnings = [];
    main({ runnerTemp, multipliersPath, warn: message => warnings.push(message) });

    expect(warnings.some(message => message.includes("failed to write AWF model alias summary"))).toBe(true);
  });
});
