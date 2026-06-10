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

  it("resolves 'copilot' provider alias to 'github-copilot' for AIC lookup", async () => {
    writeModelsFixture({
      "github-copilot": {
        models: {
          "claude-sonnet-4.6": {
            cost: {
              input: "0.000003",
              output: "0.000015",
              cache_read: "0.0000003",
            },
          },
        },
      },
    });

    const { computeInferenceAIC } = await import("./model_costs.cjs");

    // provider="copilot" should be treated as "github-copilot"
    const aicViaProvider = computeInferenceAIC({
      provider: "copilot",
      model: "claude-sonnet-4.6",
      inputTokens: 1000,
      outputTokens: 100,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    });
    expect(aicViaProvider).toBeGreaterThan(0);

    // model="copilot/claude-sonnet-4.6" (embedded provider prefix) should also resolve
    const aicViaEmbedded = computeInferenceAIC({
      provider: "",
      model: "copilot/claude-sonnet-4.6",
      inputTokens: 1000,
      outputTokens: 100,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    });
    expect(aicViaEmbedded).toBeGreaterThan(0);
    expect(aicViaEmbedded).toBeCloseTo(aicViaProvider, 6);
  });

  it("resolves 'github_models' provider alias to 'github-copilot' for AIC lookup", async () => {
    writeModelsFixture({
      "github-copilot": {
        models: {
          "gpt-5-mini": {
            cost: {
              input: "0.00000025",
              output: "0.000002",
            },
          },
        },
      },
    });

    const { computeInferenceAIC } = await import("./model_costs.cjs");

    // provider="github_models" (written by the AWF proxy for Copilot engine runs)
    // should be treated as "github-copilot" so AIC is computed and emitted
    const aic = computeInferenceAIC({
      provider: "github_models",
      model: "gpt-5-mini",
      inputTokens: 1000,
      outputTokens: 100,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
    });
    expect(aic).toBeGreaterThan(0);
  });
});
