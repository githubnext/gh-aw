import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { dedupRolloutBucket, parseCreateIssueTitleDedupRolloutPercent, resolveDeduplicateByTitle } = require("./issue_title_dedup.cjs");

describe("issue_title_dedup", () => {
  it("defaults rollout percent to 50 when value is unset", () => {
    expect(parseCreateIssueTitleDedupRolloutPercent(undefined)).toBe(50);
    expect(parseCreateIssueTitleDedupRolloutPercent("")).toBe(50);
  });

  it("falls back to default rollout percent for invalid values", () => {
    expect(parseCreateIssueTitleDedupRolloutPercent("abc")).toBe(50);
    expect(parseCreateIssueTitleDedupRolloutPercent(-1)).toBe(50);
    expect(parseCreateIssueTitleDedupRolloutPercent(101)).toBe(50);
  });

  it("uses bucket < percent threshold for rollout when config is omitted", () => {
    const seed = "example-workflow-seed";
    const bucket = dedupRolloutBucket(seed);
    const resolved = resolveDeduplicateByTitle(undefined, seed, 50);

    expect(resolved.maxDistance).toBe(0);
    expect(resolved.enabled).toBe(bucket < 50);
  });

  it("maps empty seed to out-of-rollout bucket", () => {
    expect(dedupRolloutBucket("")).toBe(100);
  });

  it("respects explicit false and bypasses rollout logic", () => {
    const resolved = resolveDeduplicateByTitle(false, "example-workflow-seed", 100);
    expect(resolved).toEqual({ enabled: false, maxDistance: 0 });
  });
});
