// @ts-check
import { describe, it, expect } from "vitest";
const { AGENT_OUTPUT_FILENAME, TMP_GH_AW_PATH } = require("./constants.cjs");

describe("constants", () => {
  it("should export AGENT_OUTPUT_FILENAME", () => {
    expect(AGENT_OUTPUT_FILENAME).toBe("agent_output.json");
  });

  it("should export TMP_GH_AW_PATH", () => {
    expect(TMP_GH_AW_PATH).toBe("/tmp/gh-aw");
  });
});
