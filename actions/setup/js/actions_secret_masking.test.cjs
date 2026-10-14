import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { escapeWorkflowCommandValue, maskSecret, maskAuthorizationValue } = require("./actions_secret_masking.cjs");

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

  it("retains the real core receiver and masks falsy scalar secrets without using the shim marker", () => {
    const receivers = [];
    const setSecret = vi.fn(function () {
      receivers.push(this);
    });
    Object.defineProperty(setSecret, "__ghAwUnavailable", { value: false });
    global.core = { setSecret };
    maskSecret(0);
    maskSecret(false);
    for (const value of [null, undefined, ""]) maskSecret(value);
    expect(setSecret.mock.calls).toEqual([["0"], ["false"]]);
    expect(receivers).toEqual([global.core, global.core]);
  });

  it("emits an add-mask workflow command in GitHub Actions when core.setSecret is unavailable", () => {
    process.env.GITHUB_ACTIONS = "true";
    global.core = {};
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("line1\nline2%tail");

    expect(write).toHaveBeenCalledWith("::add-mask::line1%0Aline2%25tail\n");
  });

  it("uses add-mask instead of the shim setSecret placeholder in GitHub Actions", () => {
    process.env.GITHUB_ACTIONS = "true";
    const setSecret = vi.fn(() => {
      throw new Error("shim placeholder");
    });
    Object.defineProperty(setSecret, "__ghAwUnavailable", { value: true });
    global.core = { setSecret };
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("derived-secret");

    expect(setSecret).not.toHaveBeenCalled();
    expect(write).toHaveBeenCalledWith("::add-mask::derived-secret\n");
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

  it("masks Basic credentials in both encoded and decoded forms without masking the username", () => {
    const setSecret = vi.fn();
    global.core = { setSecret };
    const password = "synthetic-grafana-password";
    const decoded = `123456:${password}`;
    const encoded = Buffer.from(decoded).toString("base64");

    maskAuthorizationValue(`Basic ${encoded}`);

    expect(setSecret.mock.calls).toEqual([[`Basic ${encoded}`], [encoded], [decoded], [password]]);
    expect(setSecret).not.toHaveBeenCalledWith("123456");
  });

  it("masks bare Bearer credentials case-insensitively", () => {
    const setSecret = vi.fn();
    global.core = { setSecret };

    maskAuthorizationValue("bearer synthetic-bearer-token");

    expect(setSecret.mock.calls).toEqual([["bearer synthetic-bearer-token"], ["synthetic-bearer-token"]]);
  });

  it.each(["Basic not-base64!", "Basic bm8tY29sb24=", "custom-header-value"])("masks %s without inventing decoded credentials", value => {
    const setSecret = vi.fn();
    global.core = { setSecret };

    maskAuthorizationValue(value);

    expect(setSecret).toHaveBeenCalledWith(value);
    expect(setSecret).not.toHaveBeenCalledWith("");
    expect(setSecret).not.toHaveBeenCalledWith("no-colon");
  });
});
