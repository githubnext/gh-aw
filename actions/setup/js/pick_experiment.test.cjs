// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock @actions/core globals
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

global.core = mockCore;

const { pickVariant, pickVariantWeighted, loadState, saveState, recordVariant, isWithinDateWindow, normalizeConfig, main } = await import("./pick_experiment.cjs");

describe("pick_experiment", () => {
  /** @type {string} */
  let tmpDir;

  beforeEach(() => {
    vi.clearAllMocks();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "pick_experiment_test_"));
  });

  // ── pickVariant ────────────────────────────────────────────────────────────

  describe("pickVariant", () => {
    it("selects the first variant when counts are equal", () => {
      const state = { counts: {} };
      expect(pickVariant("f", ["A", "B"], state)).toBe("A");
    });

    it("selects the least-used variant", () => {
      const state = { counts: { f: { A: 3, B: 1 } } };
      expect(pickVariant("f", ["A", "B"], state)).toBe("B");
    });

    it("handles three variants in round-robin fashion", () => {
      const state = { counts: { f: { A: 2, B: 2, C: 1 } } };
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("C");
    });

    it("returns the first variant when all counts are equal (tie-break by order)", () => {
      const state = { counts: { f: { A: 1, B: 1, C: 1 } } };
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("A");
    });

    it("handles unknown experiment name (no counts yet)", () => {
      const state = { counts: {} };
      expect(pickVariant("new", ["X", "Y"], state)).toBe("X");
    });
  });

  // ── recordVariant ──────────────────────────────────────────────────────────

  describe("recordVariant", () => {
    it("increments the variant counter", () => {
      const state = { counts: {} };
      recordVariant("f", "A", state);
      expect(state.counts["f"]["A"]).toBe(1);
    });

    it("accumulates counts across multiple calls", () => {
      const state = { counts: {} };
      recordVariant("f", "A", state);
      recordVariant("f", "A", state);
      recordVariant("f", "B", state);
      expect(state.counts["f"]["A"]).toBe(2);
      expect(state.counts["f"]["B"]).toBe(1);
    });
  });

  // ── loadState / saveState ──────────────────────────────────────────────────

  describe("loadState", () => {
    it("returns empty state when file does not exist", () => {
      const state = loadState(path.join(tmpDir, "nonexistent.json"));
      expect(state).toEqual({ counts: {} });
    });

    it("returns empty state on invalid JSON", () => {
      const file = path.join(tmpDir, "bad.json");
      fs.writeFileSync(file, "not valid json");
      const state = loadState(file);
      expect(state).toEqual({ counts: {} });
    });

    it("round-trips state through save and load", () => {
      const file = path.join(tmpDir, "state.json");
      const orig = { counts: { f: { A: 3, B: 1 } } };
      saveState(file, orig);
      const loaded = loadState(file);
      expect(loaded).toEqual(orig);
    });
  });

  // ── statistical balance ────────────────────────────────────────────────────

  describe("statistical balance", () => {
    it("distributes two variants evenly across 10 runs", () => {
      const state = { counts: {} };
      const selections = [];
      for (let i = 0; i < 10; i++) {
        const v = pickVariant("f", ["A", "B"], state);
        recordVariant("f", v, state);
        selections.push(v);
      }
      const countA = selections.filter(v => v === "A").length;
      const countB = selections.filter(v => v === "B").length;
      expect(countA).toBe(5);
      expect(countB).toBe(5);
    });

    it("distributes three variants evenly across 9 runs", () => {
      const state = { counts: {} };
      const selections = [];
      for (let i = 0; i < 9; i++) {
        const v = pickVariant("f", ["A", "B", "C"], state);
        recordVariant("f", v, state);
        selections.push(v);
      }
      const countA = selections.filter(v => v === "A").length;
      const countB = selections.filter(v => v === "B").length;
      const countC = selections.filter(v => v === "C").length;
      expect(countA).toBe(3);
      expect(countB).toBe(3);
      expect(countC).toBe(3);
    });
  });

  // ── main ───────────────────────────────────────────────────────────────────

  describe("main", () => {
    it("sets step outputs for each experiment and a combined JSON output", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        feature1: ["A", "B"],
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      // Individual output per experiment
      expect(mockCore.setOutput).toHaveBeenCalledWith("feature1", "A");
      // Combined JSON output
      expect(mockCore.setOutput).toHaveBeenCalledWith("experiments", JSON.stringify({ feature1: "A" }));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("persists state between calls to simulate multi-run balance", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        feat: ["X", "Y"],
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      // First run → X
      await main();
      const firstCall = mockCore.setOutput.mock.calls.find(c => c[0] === "feat");
      expect(firstCall?.[1]).toBe("X");

      vi.clearAllMocks();

      // Second run → Y (state persisted from first call)
      await main();
      const secondCall = mockCore.setOutput.mock.calls.find(c => c[0] === "feat");
      expect(secondCall?.[1]).toBe("Y");
    });

    it("does nothing when spec is empty", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = "{}";
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      // When no experiments are declared, the function returns early and emits no outputs.
      expect(mockCore.setOutput).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("writes assignments.json alongside state.json after picking variants", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        feature1: ["A", "B"],
        style: ["concise", "detailed"],
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(fs.existsSync(assignmentsFile)).toBe(true);
      const assignments = JSON.parse(fs.readFileSync(assignmentsFile, "utf8"));
      expect(assignments).toEqual({ feature1: "A", style: "concise" });
    });

    it("overwrites assignments.json on successive runs reflecting the current variant", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      // First run → X
      await main();
      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(JSON.parse(fs.readFileSync(assignmentsFile, "utf8"))).toEqual({ feat: "X" });

      vi.clearAllMocks();

      // Second run → Y
      await main();
      expect(JSON.parse(fs.readFileSync(assignmentsFile, "utf8"))).toEqual({ feat: "Y" });
    });

    it("does not write assignments.json when spec is empty", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = "{}";
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(fs.existsSync(assignmentsFile)).toBe(false);
    });

    it("does not write assignments.json when all experiments have fewer than 2 variants", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ exp1: ["only-one"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      // All experiments are skipped (< 2 variants), so no assignments are written.
      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(fs.existsSync(assignmentsFile)).toBe(false);
    });

    it("calls setFailed on invalid JSON spec", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = "not-json";
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      expect(mockCore.setFailed).toHaveBeenCalled();
    });

    it("accepts new object-form spec and picks variant", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["concise", "verbose"] },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("style", "concise");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("uses control variant when today is before start_date", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Use a far-future start_date to ensure we're always before it.
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["concise", "verbose"], start_date: "2099-01-01" },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      // Should use control variant (first: concise) without recording a count.
      expect(mockCore.setOutput).toHaveBeenCalledWith("style", "concise");
      // Counter for 'style' should NOT have been incremented.
      const state = loadState(stateFile);
      expect(state.counts["style"]).toBeUndefined();
    });

    it("uses control variant when today is after end_date", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["concise", "verbose"], end_date: "2000-01-01" },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("style", "concise");
    });
  });

  // ── pickVariantWeighted ────────────────────────────────────────────────────

  describe("pickVariantWeighted", () => {
    it("always selects the only non-zero-weight variant when one weight is 100", () => {
      // With weight [0, 100] the second variant must always be selected.
      for (let i = 0; i < 20; i++) {
        expect(pickVariantWeighted(["A", "B"], [0, 100])).toBe("B");
      }
    });

    it("always selects the only non-zero-weight variant when one weight is 0", () => {
      for (let i = 0; i < 20; i++) {
        expect(pickVariantWeighted(["A", "B"], [100, 0])).toBe("A");
      }
    });

    it("falls back to first variant when all weights are zero", () => {
      expect(pickVariantWeighted(["A", "B"], [0, 0])).toBe("A");
    });

    it("distributes variants proportionally across many runs", () => {
      const counts = { A: 0, B: 0 };
      const N = 1000;
      for (let i = 0; i < N; i++) {
        const v = pickVariantWeighted(["A", "B"], [70, 30]);
        counts[v]++;
      }
      // With weights 70:30 we expect ~70% A and ~30% B.  Allow 10% absolute tolerance.
      expect(counts["A"] / N).toBeCloseTo(0.7, 1);
      expect(counts["B"] / N).toBeCloseTo(0.3, 1);
    });
  });

  // ── isWithinDateWindow ────────────────────────────────────────────────────

  describe("isWithinDateWindow", () => {
    it("returns true when no dates are specified", () => {
      expect(isWithinDateWindow(undefined, undefined, "2026-06-01")).toBe(true);
    });

    it("returns true when today equals start_date", () => {
      expect(isWithinDateWindow("2026-06-01", undefined, "2026-06-01")).toBe(true);
    });

    it("returns false when today is before start_date", () => {
      expect(isWithinDateWindow("2026-06-01", undefined, "2026-05-31")).toBe(false);
    });

    it("returns true when today equals end_date", () => {
      expect(isWithinDateWindow(undefined, "2026-06-30", "2026-06-30")).toBe(true);
    });

    it("returns false when today is after end_date", () => {
      expect(isWithinDateWindow(undefined, "2026-06-30", "2026-07-01")).toBe(false);
    });

    it("returns true when today is within [start_date, end_date]", () => {
      expect(isWithinDateWindow("2026-05-01", "2026-06-30", "2026-06-01")).toBe(true);
    });

    it("returns false when today is before the window", () => {
      expect(isWithinDateWindow("2026-05-01", "2026-06-30", "2026-04-30")).toBe(false);
    });

    it("returns false when today is after the window", () => {
      expect(isWithinDateWindow("2026-05-01", "2026-06-30", "2026-07-01")).toBe(false);
    });
  });

  // ── normalizeConfig ───────────────────────────────────────────────────────

  describe("normalizeConfig", () => {
    it("wraps a bare array in a variants object", () => {
      expect(normalizeConfig(["A", "B"])).toEqual({ variants: ["A", "B"] });
    });

    it("passes through an object-form config unchanged", () => {
      const cfg = { variants: ["A", "B"], weight: [70, 30] };
      expect(normalizeConfig(cfg)).toBe(cfg);
    });
  });
});
