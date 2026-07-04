// @ts-check

import { describe, it, expect } from "vitest";

describe("merge_awf_model_multipliers.cjs compatibility shim", () => {
  it("re-exports merge_frontmatter_models API and legacy writer alias", async () => {
    const legacy = await import("./merge_awf_model_multipliers.cjs");

    expect(legacy.writeMergedModelsJSON).toEqual(expect.any(Function));
    expect(legacy.writeMergedModelMultipliersJSON).toEqual(expect.any(Function));
    expect(legacy.mergeModelCosts).toEqual(expect.any(Function));
  });
});
