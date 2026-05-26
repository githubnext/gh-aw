import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { normalizeOutcome } = req("./evaluate_outcomes.cjs");

describe("evaluate_outcomes.cjs", () => {
  it("maps existence-only fallback to weak unknown evidence", () => {
    expect(normalizeOutcome("unknown", "object still exists")).toEqual({
      outcome_status: "unknown",
      evidence_strength: "weak",
      signal: "target_exists_only",
    });
  });

  it("maps merged outcomes to strong accepted evidence", () => {
    expect(normalizeOutcome("accepted", "merged")).toEqual({
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "merged",
    });
  });
});
