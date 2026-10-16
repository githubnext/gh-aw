import { describe, it, expect, vi } from "vitest";

const { resolveDetectionContinueOnError, main } = require("./resolve_detection_continue_on_error.cjs");

describe("runtime detection enforcement default", () => {
  it.each([undefined, "", " ", "true", "TRUE", " true "])("preserves warning mode for %j", value => {
    expect(resolveDetectionContinueOnError(value)).toBe(true);
  });

  it.each(["false", "FALSE", " false "])("preserves blocking mode for %j", value => {
    expect(resolveDetectionContinueOnError(value)).toBe(false);
  });

  it.each(["yes", "0", "1", "null", "[]", "{}", "${{ false }}"])("rejects invalid effective values %j", value => {
    expect(() => resolveDetectionContinueOnError(value)).toThrow('GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR must be "true" or "false"');
  });

  it("exports false without a truthiness fallback", () => {
    const old = process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR;
    const oldCore = global.core;
    try {
      global.core = { setOutput: vi.fn() };
      process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR = "false";
      main();
      expect(global.core.setOutput).toHaveBeenCalledWith("continue_on_error", "false");
    } finally {
      global.core = oldCore;
      if (old === undefined) delete process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR;
      else process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR = old;
    }
  });
});
