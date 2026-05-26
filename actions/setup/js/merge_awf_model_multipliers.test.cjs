import { afterEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { main } from "./merge_awf_model_multipliers.cjs";

describe("merge_awf_model_multipliers.cjs", () => {
  /** @type {string[]} */
  const tempDirs = [];

  afterEach(() => {
    for (const tempDir of tempDirs) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    tempDirs.length = 0;
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
});
