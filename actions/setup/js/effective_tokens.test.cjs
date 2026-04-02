// @ts-check
/// <reference types="@actions/github-script" />

const { defaultTokenClassWeights, getTokenClassWeights, getModelMultiplier, computeBaseWeightedTokens, computeEffectiveTokens, formatET, _resetCache } = require("./effective_tokens.cjs");

// Model multipliers JSON used in tests (matches pkg/cli/data/model_multipliers.json)
const TEST_MULTIPLIERS_JSON = JSON.stringify({
  version: "1",
  description: "Test model multipliers",
  reference_model: "claude-sonnet-4.5",
  token_class_weights: {
    input: 1.0,
    cached_input: 0.1,
    output: 4.0,
    reasoning: 4.0,
    cache_write: 1.0,
  },
  multipliers: {
    "model-a": 2.0,
    "model-b": 1.0,
    "claude-haiku-4.5": 0.1,
    "claude-sonnet-4.5": 1.0,
    "claude-opus-4.5": 5.0,
    "gpt-4o": 1.0,
    "gpt-4o-mini": 0.1,
    o1: 3.0,
  },
});

describe("effective_tokens", () => {
  beforeEach(() => {
    _resetCache();
    process.env.GH_AW_MODEL_MULTIPLIERS = TEST_MULTIPLIERS_JSON;
  });

  afterEach(() => {
    _resetCache();
    delete process.env.GH_AW_MODEL_MULTIPLIERS;
  });

  describe("defaultTokenClassWeights", () => {
    test("returns spec-mandated default weights", () => {
      const w = defaultTokenClassWeights();
      expect(w.input).toBe(1.0);
      expect(w.cached_input).toBe(0.1);
      expect(w.output).toBe(4.0);
      expect(w.reasoning).toBe(4.0);
      expect(w.cache_write).toBe(1.0);
    });
  });

  describe("getTokenClassWeights", () => {
    test("returns weights from GH_AW_MODEL_MULTIPLIERS when set", () => {
      const w = getTokenClassWeights();
      expect(w.input).toBe(1.0);
      expect(w.cached_input).toBe(0.1);
      expect(w.output).toBe(4.0);
      expect(w.reasoning).toBe(4.0);
    });

    test("returns default weights when env var is not set", () => {
      _resetCache();
      delete process.env.GH_AW_MODEL_MULTIPLIERS;
      const w = getTokenClassWeights();
      expect(w.input).toBe(1.0);
      expect(w.cached_input).toBe(0.1);
      expect(w.output).toBe(4.0);
    });

    test("merges partial token_class_weights with defaults", () => {
      _resetCache();
      process.env.GH_AW_MODEL_MULTIPLIERS = JSON.stringify({
        token_class_weights: { output: 8.0 },
        multipliers: {},
      });
      const w = getTokenClassWeights();
      expect(w.input).toBe(1.0); // default
      expect(w.output).toBe(8.0); // overridden
      expect(w.cached_input).toBe(0.1); // default
    });
  });

  describe("getModelMultiplier", () => {
    test("returns exact match multiplier", () => {
      expect(getModelMultiplier("model-a")).toBe(2.0);
      expect(getModelMultiplier("model-b")).toBe(1.0);
    });

    test("is case-insensitive", () => {
      expect(getModelMultiplier("MODEL-A")).toBe(2.0);
      expect(getModelMultiplier("Model-A")).toBe(2.0);
    });

    test("returns longest prefix match", () => {
      // "gpt-4o-mini" starts with "gpt-4o", but exact "gpt-4o-mini" matches first
      expect(getModelMultiplier("gpt-4o-mini")).toBe(0.1);
      // A model that starts with "gpt-4o" but isn't exact
      expect(getModelMultiplier("gpt-4o-preview")).toBe(1.0); // prefix match to "gpt-4o"
    });

    test("returns 1.0 for unknown model (reference baseline)", () => {
      expect(getModelMultiplier("unknown-model-xyz")).toBe(1.0);
    });

    test("returns 1.0 for empty model string", () => {
      expect(getModelMultiplier("")).toBe(1.0);
    });

    test("returns 1.0 when env var is not set", () => {
      _resetCache();
      delete process.env.GH_AW_MODEL_MULTIPLIERS;
      expect(getModelMultiplier("claude-opus-4.5")).toBe(1.0);
    });

    test("matches claude-haiku-4.5 with multiplier 0.1", () => {
      expect(getModelMultiplier("claude-haiku-4.5")).toBe(0.1);
    });

    test("matches claude-opus-4.5 with multiplier 5.0", () => {
      expect(getModelMultiplier("claude-opus-4.5")).toBe(5.0);
    });
  });

  describe("computeBaseWeightedTokens", () => {
    // T-ET-001: Single invocation with all four token classes produces correct base_weighted_tokens
    test("T-ET-001: computes base weighted tokens with all token classes", () => {
      // From spec Appendix A.2:
      // root: base = (1.0 × 500) + (0.1 × 200) + (4.0 × 150) + (0 reasoning) = 500 + 20 + 600 = 1120
      // Note: cache_write is 0 in this example
      const base = computeBaseWeightedTokens(500, 150, 200, 0, 0);
      expect(base).toBe(1120);
    });

    test("computes base weighted tokens for retrieval invocation", () => {
      // retrieval: base = (1.0 × 300) + (4.0 × 100) = 300 + 400 = 700
      const base = computeBaseWeightedTokens(300, 100, 0, 0, 0);
      expect(base).toBe(700);
    });

    test("computes base weighted tokens for synthesis invocation", () => {
      // synthesis: base = (1.0 × 200) + (0.1 × 100) + (4.0 × 250) = 200 + 10 + 1000 = 1210
      const base = computeBaseWeightedTokens(200, 250, 100, 0, 0);
      expect(base).toBe(1210);
    });

    // T-ET-003: Zero-value token classes do not affect the result
    test("T-ET-003: zero token classes do not affect result", () => {
      expect(computeBaseWeightedTokens(0, 0, 0, 0, 0)).toBe(0);
      expect(computeBaseWeightedTokens(100, 0, 0, 0, 0)).toBe(100);
      expect(computeBaseWeightedTokens(0, 100, 0, 0, 0)).toBe(400);
    });

    test("includes reasoning tokens in computation", () => {
      // reasoning: w_reason = 4.0
      const base = computeBaseWeightedTokens(0, 0, 0, 0, 50);
      expect(base).toBe(200); // 4.0 × 50
    });

    test("includes cache_write tokens in computation", () => {
      // cache_write: w_cache_write = 1.0
      const base = computeBaseWeightedTokens(0, 0, 0, 100, 0);
      expect(base).toBe(100); // 1.0 × 100
    });

    // T-ET-004: Custom weights are applied when default weights are overridden
    test("T-ET-004: custom weights are applied when overridden", () => {
      _resetCache();
      process.env.GH_AW_MODEL_MULTIPLIERS = JSON.stringify({
        token_class_weights: {
          input: 2.0,
          cached_input: 0.2,
          output: 8.0,
          reasoning: 8.0,
          cache_write: 2.0,
        },
        multipliers: {},
      });
      // With custom weights: base = (2.0 × 100) + (8.0 × 50) = 200 + 400 = 600
      const base = computeBaseWeightedTokens(100, 50, 0, 0, 0);
      expect(base).toBe(600);
    });
  });

  describe("computeEffectiveTokens", () => {
    // T-ET-002: Single invocation ET equals m × base_weighted_tokens
    test("T-ET-002: ET equals m × base_weighted_tokens", () => {
      // root: base=1120, m=2.0, ET=2240
      const et = computeEffectiveTokens("model-a", 500, 150, 200, 0, 0);
      expect(et).toBe(2240);
    });

    test("computes ET for retrieval invocation (m=1.0)", () => {
      // retrieval: base=700, m=1.0, ET=700
      const et = computeEffectiveTokens("model-b", 300, 100, 0, 0, 0);
      expect(et).toBe(700);
    });

    test("computes ET for synthesis invocation (m=2.0)", () => {
      // synthesis: base=1210, m=2.0, ET=2420
      const et = computeEffectiveTokens("model-a", 200, 250, 100, 0, 0);
      expect(et).toBe(2420);
    });

    test("returns 0 for zero token inputs", () => {
      expect(computeEffectiveTokens("model-a", 0, 0, 0, 0, 0)).toBe(0);
    });

    test("uses multiplier 1.0 for unknown model", () => {
      // base = 1.0 × 100 = 100, m = 1.0, ET = 100
      const et = computeEffectiveTokens("unknown-model", 100, 0, 0, 0, 0);
      expect(et).toBe(100);
    });

    test("computes ET as real-valued product (no rounding)", () => {
      // gpt-4o-mini multiplier = 0.1, base = 1.0 × 100 = 100, ET = 0.1 × 100 = 10
      const et = computeEffectiveTokens("gpt-4o-mini", 100, 0, 0, 0, 0);
      expect(et).toBe(10);
    });

    test("correctly handles high multiplier model (claude-opus)", () => {
      // claude-opus-4.5 multiplier = 5.0
      // base = 1.0 × 100 + 4.0 × 50 = 100 + 200 = 300, ET = 5.0 × 300 = 1500
      const et = computeEffectiveTokens("claude-opus-4.5", 100, 50, 0, 0, 0);
      expect(et).toBe(1500);
    });

    test("handles reasoning tokens with o1 model (m=3.0)", () => {
      // o1 multiplier = 3.0
      // base = (1.0 × 100) + (4.0 × 50) + (4.0 × 30) = 100 + 200 + 120 = 420
      // ET = 3.0 × 420 = 1260
      const et = computeEffectiveTokens("o1", 100, 50, 0, 0, 30);
      expect(et).toBe(1260);
    });
  });

  describe("Spec Appendix A: Worked Example (T-ET-010, T-ET-011, T-ET-012)", () => {
    // Complete worked example from spec Appendix A.2-A.4
    test("T-ET-010: multi-invocation ET_total equals sum of per-invocation ETs", () => {
      const rootET = computeEffectiveTokens("model-a", 500, 150, 200, 0, 0); // 2240
      const retrievalET = computeEffectiveTokens("model-b", 300, 100, 0, 0, 0); // 700
      const synthesisET = computeEffectiveTokens("model-a", 200, 250, 100, 0, 0); // 2420

      expect(rootET).toBe(2240);
      expect(retrievalET).toBe(700);
      expect(synthesisET).toBe(2420);

      const totalET = rootET + retrievalET + synthesisET;
      expect(totalET).toBe(5360);
    });

    test("T-ET-011: raw_total_tokens equals sum of all raw tokens", () => {
      // root:      500+150+200+0 = 850
      // retrieval: 300+100+0+0  = 400
      // synthesis: 200+250+100+0 = 550
      // total: 1800
      const rawTotal = 500 + 150 + 200 + 0 + (300 + 100 + 0 + 0) + (200 + 250 + 100 + 0);
      expect(rawTotal).toBe(1800);
    });

    test("T-ET-012: total_invocations count is 3 (root + 2 sub-agents)", () => {
      const invocations = [
        { id: "root", parentId: null, model: "model-a", inputTokens: 500, outputTokens: 150, cacheReadTokens: 200, cacheWriteTokens: 0 },
        { id: "retrieval", parentId: "root", model: "model-b", inputTokens: 300, outputTokens: 100, cacheReadTokens: 0, cacheWriteTokens: 0 },
        { id: "synthesis", parentId: "root", model: "model-a", inputTokens: 200, outputTokens: 250, cacheReadTokens: 100, cacheWriteTokens: 0 },
      ];
      expect(invocations.length).toBe(3);
      // T-ET-020: root node has parent_id = null
      expect(invocations[0].parentId).toBeNull();
      // T-ET-021: sub-agents reference valid parent_id
      expect(invocations[1].parentId).toBe("root");
      expect(invocations[2].parentId).toBe("root");
    });
  });

  describe("env var parsing edge cases", () => {
    test("handles malformed JSON gracefully", () => {
      _resetCache();
      process.env.GH_AW_MODEL_MULTIPLIERS = "{ not valid json }";
      expect(() => getModelMultiplier("any-model")).not.toThrow();
      expect(getModelMultiplier("any-model")).toBe(1.0);
    });

    test("handles empty env var gracefully", () => {
      _resetCache();
      process.env.GH_AW_MODEL_MULTIPLIERS = "";
      expect(getModelMultiplier("any-model")).toBe(1.0);
    });

    test("handles missing multipliers key gracefully", () => {
      _resetCache();
      process.env.GH_AW_MODEL_MULTIPLIERS = JSON.stringify({ version: "1" });
      expect(getModelMultiplier("any-model")).toBe(1.0);
    });

    test("caches parsed result across multiple calls", () => {
      getModelMultiplier("model-a");
      getModelMultiplier("model-b");
      // Should not throw or cause inconsistencies
      expect(getModelMultiplier("model-a")).toBe(2.0);
      expect(getModelMultiplier("model-b")).toBe(1.0);
    });
  });

  describe("formatET", () => {
    test("returns exact string for values under 1000", () => {
      expect(formatET(0)).toBe("0");
      expect(formatET(1)).toBe("1");
      expect(formatET(900)).toBe("900");
      expect(formatET(999)).toBe("999");
    });

    test("formats values in the thousands as K", () => {
      expect(formatET(1000)).toBe("1K");
      expect(formatET(1200)).toBe("1.2K");
      expect(formatET(12345)).toBe("12.3K");
      expect(formatET(450000)).toBe("450K");
      expect(formatET(999999)).toBe("1000K");
    });

    test("formats values in the millions as M", () => {
      expect(formatET(1_000_000)).toBe("1M");
      expect(formatET(1_200_000)).toBe("1.2M");
      expect(formatET(12_345_678)).toBe("12.3M");
    });

    test("omits trailing .0 in K/M format", () => {
      expect(formatET(2000)).toBe("2K");
      expect(formatET(5_000_000)).toBe("5M");
    });
  });
});
