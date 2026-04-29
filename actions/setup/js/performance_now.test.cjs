// @ts-check
import { describe, it, expect, vi, afterEach } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const perfHooks = req("perf_hooks");

describe("performance_now.cjs", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("exports nowMs as a function", () => {
    const { nowMs } = req("./performance_now.cjs");
    expect(typeof nowMs).toBe("function");
  });

  it("returns a number", () => {
    const { nowMs } = req("./performance_now.cjs");
    expect(typeof nowMs()).toBe("number");
  });

  it("returns a value within the current performance-based epoch range", () => {
    const { nowMs } = req("./performance_now.cjs");
    const before = perfHooks.performance.timeOrigin + perfHooks.performance.now();
    const result = nowMs();
    const after = perfHooks.performance.timeOrigin + perfHooks.performance.now();
    expect(result).toBeGreaterThanOrEqual(before);
    expect(result).toBeLessThanOrEqual(after);
  });

  it("returns an increasing value on successive calls", () => {
    const { nowMs } = req("./performance_now.cjs");
    const first = nowMs();
    const second = nowMs();
    expect(second).toBeGreaterThanOrEqual(first);
  });

  it("uses performance.timeOrigin + performance.now()", () => {
    vi.spyOn(perfHooks.performance, "timeOrigin", "get").mockReturnValue(1000);
    vi.spyOn(perfHooks.performance, "now").mockReturnValue(500.25);

    // Re-require the module to use the mocked values
    const { nowMs } = req("./performance_now.cjs");
    const result = nowMs();
    expect(result).toBe(1500.25);
  });

  it("returns a floating-point value when performance.now() has sub-ms precision", () => {
    vi.spyOn(perfHooks.performance, "timeOrigin", "get").mockReturnValue(1_700_000_000_000);
    vi.spyOn(perfHooks.performance, "now").mockReturnValue(123.456);

    const { nowMs } = req("./performance_now.cjs");
    expect(nowMs()).toBeCloseTo(1_700_000_000_123.456, 2);
  });

  it("returns epoch-scale milliseconds (greater than year 2020)", () => {
    const { nowMs } = req("./performance_now.cjs");
    // Jan 1 2020 in ms
    expect(nowMs()).toBeGreaterThan(1_577_836_800_000);
  });

  it("returns epoch-scale milliseconds (less than year 2100)", () => {
    const { nowMs } = req("./performance_now.cjs");
    // Jan 1 2100 in ms
    expect(nowMs()).toBeLessThan(4_102_444_800_000);
  });

  it("handles zero performance.now() value", () => {
    const origin = 1_700_000_000_000;
    vi.spyOn(perfHooks.performance, "timeOrigin", "get").mockReturnValue(origin);
    vi.spyOn(perfHooks.performance, "now").mockReturnValue(0);

    const { nowMs } = req("./performance_now.cjs");
    expect(nowMs()).toBe(origin);
  });
});
