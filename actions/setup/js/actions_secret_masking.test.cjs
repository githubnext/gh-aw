import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { escapeWorkflowCommandValue, maskSecret } = require("./actions_secret_masking.cjs");

describe("actions_secret_masking.cjs", () => {
  let originalCore;
  let originalGithubActions;

  beforeEach(() => {
    originalCore = global.core;
    originalGithubActions = process.env.GITHUB_ACTIONS;
  });

  afterEach(() => {
    if (originalCore === undefined) {
      delete global.core;
    } else {
      global.core = originalCore;
    }
    if (originalGithubActions === undefined) {
      delete process.env.GITHUB_ACTIONS;
    } else {
      process.env.GITHUB_ACTIONS = originalGithubActions;
    }
    vi.restoreAllMocks();
  });

  it("uses core.setSecret when a real implementation is available", () => {
    const setSecret = vi.fn();
    global.core = { setSecret };

    maskSecret("derived-secret");

    expect(setSecret).toHaveBeenCalledWith("derived-secret");
  });

  it("emits an add-mask workflow command in GitHub Actions when core.setSecret is unavailable", () => {
    process.env.GITHUB_ACTIONS = "true";
    global.core = {};
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("line1\nline2%tail");

    expect(write).toHaveBeenCalledWith("::add-mask::line1%0Aline2%25tail\n");
  });

  it("does nothing outside GitHub Actions when core.setSecret is unavailable", () => {
    delete process.env.GITHUB_ACTIONS;
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("derived-secret");

    expect(write).not.toHaveBeenCalled();
  });

  it("escapes workflow command payloads", () => {
    expect(escapeWorkflowCommandValue("a%b\rc\n")).toBe("a%25b%0Dc%0A");
  });
});
