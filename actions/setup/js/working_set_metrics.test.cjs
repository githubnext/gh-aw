import { describe, it, expect } from "vitest";

const { calculateWorkingSetFromEntries, calculateWorkingSetFromJSONL } = require("./working_set_metrics.cjs");

describe("working_set_metrics", () => {
  describe("calculateWorkingSetFromEntries", () => {
    it("returns unavailable for no entries", () => {
      const { workingSet, ignoredRecords } = calculateWorkingSetFromEntries([]);
      expect(workingSet).toEqual({ measurement_state: "unavailable", cumulative_input_tokens: 0, peak_input_tokens: 0, rebuild_excess_tokens: 0, invocations: 0 });
      expect(ignoredRecords).toBe(0);
    });

    it("computes rebuild factor for valid entries", () => {
      const { workingSet } = calculateWorkingSetFromEntries([{ input_tokens: 100 }, { input_tokens: 200 }, { input_tokens: 300 }]);
      expect(workingSet.measurement_state).toBe("measured");
      expect(workingSet.cumulative_input_tokens).toBe(600);
      expect(workingSet.peak_input_tokens).toBe(300);
      expect(workingSet.rebuild_excess_tokens).toBe(300);
      expect(workingSet.invocations).toBe(3);
      expect(workingSet.rebuild_factor).toBe(2);
    });

    it("has rebuild factor 1 for a single entry", () => {
      const { workingSet } = calculateWorkingSetFromEntries([{ input_tokens: 50 }]);
      expect(workingSet.rebuild_factor).toBe(1);
      expect(workingSet.rebuild_excess_tokens).toBe(0);
    });

    it("ignores invalid records and marks partial", () => {
      const entries = [{ input_tokens: 10 }, { input_tokens: -1 }, { input_tokens: 1.5 }, { input_tokens: "5" }, null, {}];
      const { workingSet, ignoredRecords } = calculateWorkingSetFromEntries(entries);
      expect(ignoredRecords).toBe(5);
      expect(workingSet.measurement_state).toBe("partial");
      expect(workingSet.invocations).toBe(1);
    });

    it("ignores unsafe integers", () => {
      const { ignoredRecords } = calculateWorkingSetFromEntries([{ input_tokens: Number.MAX_SAFE_INTEGER + 1 }]);
      expect(ignoredRecords).toBe(1);
    });

    it("stays unavailable when all tokens are zero", () => {
      const { workingSet } = calculateWorkingSetFromEntries([{ input_tokens: 0 }, { input_tokens: 0 }]);
      expect(workingSet.measurement_state).toBe("unavailable");
      expect(workingSet.rebuild_factor).toBeUndefined();
      expect(workingSet.invocations).toBe(2);
    });

    it("honors initial ignored record count", () => {
      const { workingSet, ignoredRecords } = calculateWorkingSetFromEntries([{ input_tokens: 10 }], 2);
      expect(ignoredRecords).toBe(2);
      expect(workingSet.measurement_state).toBe("partial");
    });
  });

  describe("calculateWorkingSetFromJSONL", () => {
    it("parses valid JSONL", () => {
      const { workingSet, ignoredRecords } = calculateWorkingSetFromJSONL('{"input_tokens":100}\n{"input_tokens":300}\n');
      expect(workingSet.measurement_state).toBe("measured");
      expect(workingSet.rebuild_factor).toBe(4 / 3);
      expect(ignoredRecords).toBe(0);
    });

    it("counts malformed lines as ignored and marks partial", () => {
      const { workingSet, ignoredRecords } = calculateWorkingSetFromJSONL('{"input_tokens":100}\nnot json\n{"input_tokens":100}\n');
      expect(ignoredRecords).toBe(1);
      expect(workingSet.measurement_state).toBe("partial");
      expect(workingSet.invocations).toBe(2);
    });

    it("handles empty content", () => {
      const { workingSet } = calculateWorkingSetFromJSONL("");
      expect(workingSet.measurement_state).toBe("unavailable");
    });
  });
});
