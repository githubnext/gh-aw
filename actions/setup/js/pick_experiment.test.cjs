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
  exportVariable: vi.fn(),
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
    it("breaks ties randomly for two-variant experiment when counts are equal", () => {
      const state = { counts: {} };
      // Math.floor(0 * 2) = 0 → tied[0] = "A"
      vi.spyOn(Math, "random").mockReturnValueOnce(0);
      expect(pickVariant("f", ["A", "B"], state)).toBe("A");

      // Math.floor(0.5 * 2) = 1 → tied[1] = "B"
      vi.spyOn(Math, "random").mockReturnValueOnce(0.5);
      expect(pickVariant("f", ["A", "B"], state)).toBe("B");

      vi.restoreAllMocks();
    });

    it("selects the least-used variant", () => {
      const state = { counts: { f: { A: 3, B: 1 } } };
      expect(pickVariant("f", ["A", "B"], state)).toBe("B");
    });

    it("handles three variants in round-robin fashion", () => {
      const state = { counts: { f: { A: 2, B: 2, C: 1 } } };
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("C");
    });

    it("randomly selects from all tied variants when all counts are equal", () => {
      const state = { counts: { f: { A: 1, B: 1, C: 1 } } };
      // All three variants are tied; verify the random index is respected.
      // Math.floor(0   * 3) = 0 → tied[0] = "A"
      // Math.floor(0.4 * 3) = 1 → tied[1] = "B"  (0.4*3=1.2)
      // Math.floor(0.7 * 3) = 2 → tied[2] = "C"  (0.7*3=2.1)
      vi.spyOn(Math, "random").mockReturnValueOnce(0).mockReturnValueOnce(0.4).mockReturnValueOnce(0.7);
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("A");
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("B");
      expect(pickVariant("f", ["A", "B", "C"], state)).toBe("C");

      vi.restoreAllMocks();
    });

    it("handles unknown experiment name (no counts yet) by picking randomly", () => {
      const state = { counts: {} };
      // Both variants are tied with zero counts; verify the random index is respected.
      // Math.floor(0   * 2) = 0 → tied[0] = "X"
      // Math.floor(0.5 * 2) = 1 → tied[1] = "Y"
      vi.spyOn(Math, "random").mockReturnValueOnce(0);
      expect(pickVariant("new", ["X", "Y"], state)).toBe("X");

      vi.spyOn(Math, "random").mockReturnValueOnce(0.5);
      expect(pickVariant("new", ["X", "Y"], state)).toBe("Y");

      vi.restoreAllMocks();
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
      expect(state).toEqual({ counts: {}, runs: [] });
    });

    it("returns empty state on invalid JSON", () => {
      const file = path.join(tmpDir, "bad.json");
      fs.writeFileSync(file, "not valid json");
      const state = loadState(file);
      expect(state).toEqual({ counts: {}, runs: [] });
    });

    it("round-trips state through save and load", () => {
      const file = path.join(tmpDir, "state.json");
      const orig = { counts: { f: { A: 3, B: 1 } }, runs: [] };
      saveState(file, orig);
      const loaded = loadState(file);
      expect(loaded).toEqual(orig);
    });

    it("initialises runs to [] when loading legacy state without runs field", () => {
      const file = path.join(tmpDir, "state.json");
      fs.writeFileSync(file, JSON.stringify({ counts: { f: { A: 1 } } }), "utf8");
      const loaded = loadState(file);
      expect(loaded.runs).toEqual([]);
    });

    it("preserves existing runs array when loading state", () => {
      const file = path.join(tmpDir, "state.json");
      const runs = [{ run_id: "123", timestamp: "2026-01-01T00:00:00.000Z", assignments: { f: "A" } }];
      fs.writeFileSync(file, JSON.stringify({ counts: { f: { A: 1 } }, runs }), "utf8");
      const loaded = loadState(file);
      expect(loaded.runs).toEqual(runs);
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

      // Force Math.random → 0 so the first tied variant ("A") is selected.
      vi.spyOn(Math, "random").mockReturnValue(0);

      await main();

      // Individual output per experiment
      expect(mockCore.setOutput).toHaveBeenCalledWith("feature1", "A");
      // Combined JSON output
      expect(mockCore.setOutput).toHaveBeenCalledWith("experiments", JSON.stringify({ feature1: "A" }));
      expect(mockCore.setFailed).not.toHaveBeenCalled();

      vi.restoreAllMocks();
    });

    it("persists state between calls to simulate multi-run balance", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        feat: ["X", "Y"],
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      // Force Math.random → 0 so the first tied variant ("X") is selected on the first run.
      vi.spyOn(Math, "random").mockReturnValue(0);

      // First run → X
      await main();
      const firstCall = mockCore.setOutput.mock.calls.find(c => c[0] === "feat");
      expect(firstCall?.[1]).toBe("X");

      vi.restoreAllMocks();
      vi.clearAllMocks();

      // Second run → Y (state persisted from first call; Y has the lower count)
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

      // Force Math.random → 0 so the first tied variant is chosen for each experiment.
      vi.spyOn(Math, "random").mockReturnValue(0);

      await main();

      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(fs.existsSync(assignmentsFile)).toBe(true);
      const assignments = JSON.parse(fs.readFileSync(assignmentsFile, "utf8"));
      expect(assignments).toEqual({ feature1: "A", style: "concise" });

      vi.restoreAllMocks();
    });

    it("overwrites assignments.json on successive runs reflecting the current variant", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      // Force Math.random → 0 so the first tied variant ("X") is chosen on the first run.
      vi.spyOn(Math, "random").mockReturnValue(0);

      // First run → X
      await main();
      const assignmentsFile = path.join(tmpDir, "assignments.json");
      expect(JSON.parse(fs.readFileSync(assignmentsFile, "utf8"))).toEqual({ feat: "X" });

      vi.restoreAllMocks();
      vi.clearAllMocks();

      // Second run → Y (Y has the lower count after first run recorded X)
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

      // Force Math.random → 0 so the first tied variant ("concise") is chosen.
      vi.spyOn(Math, "random").mockReturnValue(0);

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("style", "concise");
      expect(mockCore.setFailed).not.toHaveBeenCalled();

      vi.restoreAllMocks();
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

    it("includes description as blockquote in step summary when description field is set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], description: "Test the new style feature" },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("> Test the new style feature");
    });

    it("wraps experiment assignments in a details section", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"] },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("<details>");
      expect(rawCall).toContain("<summary>🧪 Experiment Assignments</summary>");
      expect(rawCall).toContain("</details>");
    });

    it("includes tracking issue link in step summary when issue field is set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], issue: 42 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      process.env.GITHUB_REPOSITORY = "myorg/myrepo";

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("[#42](https://github.com/myorg/myrepo/issues/42)");
    });

    it("includes both description and issue link when both fields are set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], description: "My experiment", issue: 99 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      process.env.GITHUB_REPOSITORY = "owner/repo";

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("> My experiment");
      expect(rawCall).toContain("[#99](https://github.com/owner/repo/issues/99)");
    });

    it("does not include description or issue extras for legacy bare-array experiments", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        feature1: ["A", "B"],
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).not.toContain("Tracking issue");
      expect(rawCall).not.toContain("> ");
    });

    it("renders issue number without link when GITHUB_REPOSITORY is not set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], issue: 7 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("Tracking issue: #7");
      expect(rawCall).not.toContain("https://github.com");
    });

    it("renders hypothesis in step summary when hypothesis field is set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], hypothesis: "H0: no change. H1: A is faster." },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("**Hypothesis:** H0: no change. H1: A is faster.");
    });

    it("renders guardrail metrics in step summary when guardrail_metrics field is set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: {
          variants: ["A", "B"],
          guardrail_metrics: [
            { name: "success_rate", threshold: ">=0.95" },
            { name: "empty_output_rate", threshold: "==0" },
          ],
        },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("**Guardrail metrics:**");
      expect(rawCall).toContain("`success_rate` >=0.95");
      expect(rawCall).toContain("`empty_output_rate` ==0");
    });

    it("renders progress bar in step summary when min_samples field is set", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate state to simulate 10 runs for variant A, 5 for B.
      const state = { counts: { style: { A: 10, B: 5 } } };
      const fs2 = require("fs");
      fs2.writeFileSync(stateFile, JSON.stringify(state, null, 2));
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], min_samples: 20 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("📊 Sampling Progress");
      expect(rawCall).toContain("(target: 20 per variant)");
      // Variant A has 11 runs (10 preloaded + 1 picked), variant B has 5 runs.
      expect(rawCall).toContain("/20");
    });

    it("shows ready-for-analysis flag when all variants reach min_samples", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate state so both variants are already at min_samples.
      const state = { counts: { style: { A: 25, B: 25 } } };
      const fs2 = require("fs");
      fs2.writeFileSync(stateFile, JSON.stringify(state, null, 2));
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], min_samples: 25 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).toContain("✅ Ready for analysis");
    });

    it("does not render a progress bar when min_samples is 0", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], min_samples: 0 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).not.toContain("📊 Sampling Progress");
    });

    it("does not render a progress bar when min_samples is negative", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({
        style: { variants: ["A", "B"], min_samples: -5 },
      });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.GITHUB_REPOSITORY;

      await main();

      const rawCall = mockCore.summary.addRaw.mock.calls[0]?.[0] ?? "";
      expect(rawCall).not.toContain("📊 Sampling Progress");
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

  // ── per-run metadata (state.runs) ─────────────────────────────────────────

  describe("per-run metadata", () => {
    it("appends a run record to state.runs after picking variants", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate counts so Y is the deterministic least-used pick (avoids random tie-break).
      fs.writeFileSync(stateFile, JSON.stringify({ counts: { feat: { X: 1, Y: 0 } }, runs: [] }), "utf8");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      process.env.GITHUB_RUN_ID = "42";

      await main();

      const state = loadState(stateFile);
      expect(state.runs).toHaveLength(1);
      expect(state.runs[0].run_id).toBe("42");
      expect(state.runs[0].assignments).toEqual({ feat: "Y" });
      expect(typeof state.runs[0].timestamp).toBe("string");
    });

    it("accumulates run records across multiple runs", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      process.env.GITHUB_RUN_ID = "1";

      await main();
      vi.clearAllMocks();

      process.env.GITHUB_RUN_ID = "2";
      await main();

      const state = loadState(stateFile);
      expect(state.runs).toHaveLength(2);
      expect(state.runs[0].run_id).toBe("1");
      expect(state.runs[1].run_id).toBe("2");
    });

    it("does not append a run record when no experiments are assigned", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = "{}";
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      const stateFile = path.join(tmpDir, "state.json");
      // state.json is not written when no experiments are declared
      expect(fs.existsSync(stateFile)).toBe(false);
    });

    it("prunes runs to last MAX_RUN_HISTORY when run history exceeds the cap", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate with 513 fake runs (above MAX_RUN_HISTORY = 512).
      const existingRuns = Array.from({ length: 513 }, (_, i) => ({
        run_id: String(i),
        timestamp: "2026-01-01T00:00:00.000Z",
        assignments: { feat: "X" },
      }));
      fs.writeFileSync(stateFile, JSON.stringify({ counts: { feat: { X: 513, Y: 0 } }, runs: existingRuns }), "utf8");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      const state = loadState(stateFile);
      // 513 existing + 1 new = 514, pruned to last 512.
      expect(state.runs).toHaveLength(512);
      // The most recent run is always last.
      expect(state.runs[state.runs.length - 1].assignments).toEqual({ feat: "Y" });
    });
  });

  // ── OTEL resource attributes ──────────────────────────────────────────────

  describe("OTEL resource attributes", () => {
    it("exports OTEL_RESOURCE_ATTRIBUTES with experiment assignments", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate so Y is the deterministic least-used pick.
      fs.writeFileSync(stateFile, JSON.stringify({ counts: { feat: { X: 1, Y: 0 } }, runs: [] }), "utf8");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      delete process.env.OTEL_RESOURCE_ATTRIBUTES;

      await main();

      expect(mockCore.exportVariable).toHaveBeenCalledWith("OTEL_RESOURCE_ATTRIBUTES", "experiment.feat=Y");
    });

    it("appends to existing OTEL_RESOURCE_ATTRIBUTES", async () => {
      const stateFile = path.join(tmpDir, "state.json");
      // Pre-populate so Y is the deterministic least-used pick.
      fs.writeFileSync(stateFile, JSON.stringify({ counts: { feat: { X: 1, Y: 0 } }, runs: [] }), "utf8");
      process.env.GH_AW_EXPERIMENT_SPEC = JSON.stringify({ feat: ["X", "Y"] });
      process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
      process.env.OTEL_RESOURCE_ATTRIBUTES = "service.name=myservice";

      await main();

      expect(mockCore.exportVariable).toHaveBeenCalledWith("OTEL_RESOURCE_ATTRIBUTES", "service.name=myservice,experiment.feat=Y");
    });

    it("does not export OTEL_RESOURCE_ATTRIBUTES when no experiments are assigned", async () => {
      process.env.GH_AW_EXPERIMENT_SPEC = "{}";
      process.env.GH_AW_EXPERIMENT_STATE_FILE = path.join(tmpDir, "state.json");
      process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;

      await main();

      expect(mockCore.exportVariable).not.toHaveBeenCalled();
    });
  });
});
