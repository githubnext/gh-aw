// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { getActionInput } = req("./action_input_utils.cjs");

describe("getActionInput", () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = {
      INPUT_JOB_NAME: process.env.INPUT_JOB_NAME,
      "INPUT_JOB-NAME": process.env["INPUT_JOB-NAME"],
    };
    delete process.env.INPUT_JOB_NAME;
    delete process.env["INPUT_JOB-NAME"];
  });

  afterEach(() => {
    if (originalEnv.INPUT_JOB_NAME !== undefined) {
      process.env.INPUT_JOB_NAME = originalEnv.INPUT_JOB_NAME;
    } else {
      delete process.env.INPUT_JOB_NAME;
    }
    if (originalEnv["INPUT_JOB-NAME"] !== undefined) {
      process.env["INPUT_JOB-NAME"] = originalEnv["INPUT_JOB-NAME"];
    } else {
      delete process.env["INPUT_JOB-NAME"];
    }
  });

  it("returns the underscore form value when set", () => {
    process.env.INPUT_JOB_NAME = "agent";
    expect(getActionInput("JOB_NAME")).toBe("agent");
  });

  it("returns the hyphen form value when only the hyphen form is set", () => {
    process.env["INPUT_JOB-NAME"] = "agent";
    expect(getActionInput("JOB_NAME")).toBe("agent");
  });

  it("prefers the underscore form over the hyphen form", () => {
    process.env.INPUT_JOB_NAME = "underscore-value";
    process.env["INPUT_JOB-NAME"] = "hyphen-value";
    expect(getActionInput("JOB_NAME")).toBe("underscore-value");
  });

  it("trims whitespace from the returned value", () => {
    process.env.INPUT_JOB_NAME = "  agent  ";
    expect(getActionInput("JOB_NAME")).toBe("agent");
  });

  it("returns empty string when neither form is set", () => {
    expect(getActionInput("JOB_NAME")).toBe("");
  });
});
