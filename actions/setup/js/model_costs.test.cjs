// @ts-check

import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, describe, expect, it } from "vitest";

const tmpDirs = [];

afterEach(async () => {
  delete process.env.GH_AW_MODELS_JSON_PATH;
  const { _resetModelCostsCache } = await import("./model_costs.cjs");
  _resetModelCostsCache();
  for (const dir of tmpDirs.splice(0)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});

function writeModelsFixture(providers) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-model-costs-"));
  tmpDirs.push(dir);
  const file = path.join(dir, "models.json");
  fs.writeFileSync(file, JSON.stringify({ providers }, null, 2));
  process.env.GH_AW_MODELS_JSON_PATH = file;
}

describe("model_costs.cjs", () => {
  it("computes inference AIC using provider-specific pricing", async () => {
    writeModelsFixture({
      anthropic: {
        models: {
          "claude-sonnet-4.6": {
            cost: {
              input: "0.000003",
              output: "0.000015",
              cache_read: "0.0000003",
              cache_write: "0.00000375",
              reasoning: "0.000015",
            },
          },
        },
      },
    });

    const { computeInferenceAIC } = await import("./model_costs.cjs");
    const aic = computeInferenceAIC({
      provider: "anthropic",
      model: "claude-sonnet-4.6-20250514",
      inputTokens: 1000,
      outputTokens: 200,
      cacheReadTokens: 400,
      cacheWriteTokens: 50,
      reasoningTokens: 25,
    });

    expect(aic).toBeCloseTo(0.54825, 6);
  });

  it("formats AI Credits for footer display", async () => {
    const { formatAIC } = await import("./model_costs.cjs");
    expect(formatAIC(0.125)).toBe("0.125");
    expect(formatAIC(1.25)).toBe("1.25");
    expect(formatAIC(12.5)).toBe("12.5");
  });
});
