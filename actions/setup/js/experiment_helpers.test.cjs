import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";

const { readExperimentAssignments, EXPERIMENT_ASSIGNMENTS_PATH } = await import("./experiment_helpers.cjs");

describe("readExperimentAssignments", () => {
  let readFileSpy;
  const savedStateDir = process.env.GH_AW_EXPERIMENT_STATE_DIR;

  beforeEach(() => {
    delete process.env.GH_AW_EXPERIMENT_STATE_DIR;
    readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
  });

  afterEach(() => {
    readFileSpy.mockRestore();
    if (savedStateDir !== undefined) {
      process.env.GH_AW_EXPERIMENT_STATE_DIR = savedStateDir;
    } else {
      delete process.env.GH_AW_EXPERIMENT_STATE_DIR;
    }
  });

  it("returns null when the assignments file does not exist", () => {
    expect(readExperimentAssignments()).toBeNull();
  });

  it("returns null when the assignments file contains invalid JSON", () => {
    readFileSpy.mockReturnValue("not-valid-json");
    expect(readExperimentAssignments()).toBeNull();
  });

  it("returns null when the assignments file contains a non-object value", () => {
    readFileSpy.mockReturnValue(JSON.stringify(["A", "B"]));
    expect(readExperimentAssignments()).toBeNull();
  });

  it("returns the parsed assignments object when the file is valid", () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === EXPERIMENT_ASSIGNMENTS_PATH) {
        return JSON.stringify({ caveman: "yes", style: "detailed" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readExperimentAssignments()).toEqual({ caveman: "yes", style: "detailed" });
  });

  it("reads from GH_AW_EXPERIMENT_STATE_DIR/assignments.json when env var is set", () => {
    process.env.GH_AW_EXPERIMENT_STATE_DIR = "/custom/experiments";
    readFileSpy.mockImplementation(filePath => {
      if (filePath === "/custom/experiments/assignments.json") {
        return JSON.stringify({ feature: "on" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readExperimentAssignments()).toEqual({ feature: "on" });
  });

  it("falls back to EXPERIMENT_ASSIGNMENTS_PATH when GH_AW_EXPERIMENT_STATE_DIR is not set", () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === EXPERIMENT_ASSIGNMENTS_PATH) {
        return JSON.stringify({ mode: "fast" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    expect(readExperimentAssignments()).toEqual({ mode: "fast" });
  });
});
