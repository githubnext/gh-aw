import { describe, expect, it } from "vitest";

const { isOTLPPayload } = await import("./export_copilot_otel_traces.cjs");

describe("isOTLPPayload", () => {
  it("returns true when resourceSpans is an array", () => {
    expect(isOTLPPayload({ resourceSpans: [] })).toBe(true);
  });

  it("returns false for non-object and missing resourceSpans", () => {
    expect(isOTLPPayload(null)).toBe(false);
    expect(isOTLPPayload("not-an-object")).toBe(false);
    expect(isOTLPPayload({})).toBe(false);
    expect(isOTLPPayload({ resourceSpans: "not-an-array" })).toBe(false);
  });
});
