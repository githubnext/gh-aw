import { describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { readJSONL as readEvaluations } from "./evaluate_outcomes.cjs";
import { readJSONL as readSpans } from "./emit_outcome_spans.cjs";

describe.each([
  { name: "evaluations", read: readEvaluations, values: [{ id: 1 }, { id: 2 }] },
  { name: "spans", read: readSpans, values: [{ id: 1 }, null, false, 0, { id: 2 }] },
])("$name JSONL reader", ({ read, values }) => {
  it("opens lazily, skips corrupt lines, and preserves existing value semantics", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-outcome-reader-"));
    const file = path.join(directory, "input.jsonl");
    fs.writeFileSync(file, '{"id":1}\nnot-json\nnull\nfalse\n0\n{"id":2}');
    const open = vi.spyOn(fs, "openSync");
    try {
      const records = read(file);
      expect(records[Symbol.iterator]()).toBe(records);
      expect(open).not.toHaveBeenCalled();
      expect([...records]).toEqual(values);
      expect(open).toHaveBeenCalledTimes(1);
    } finally {
      open.mockRestore();
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });

  it("closes the file immediately when consumption stops", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-outcome-reader-"));
    const file = path.join(directory, "input.jsonl");
    fs.writeFileSync(file, '{"id":1}\n{"id":2}\n');
    const close = vi.spyOn(fs, "closeSync");
    const parse = vi.spyOn(JSON, "parse");
    try {
      for (const record of read(file)) {
        expect(record).toEqual({ id: 1 });
        break;
      }
      expect(close).toHaveBeenCalledTimes(1);
      expect(parse).toHaveBeenCalledTimes(1);
    } finally {
      close.mockRestore();
      parse.mockRestore();
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });

  it("tolerates missing files but propagates other I/O failures", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-outcome-reader-"));
    const file = path.join(directory, "input.jsonl");
    try {
      expect([...read(file)]).toEqual([]);
      const error = Object.assign(new Error("Permission denied"), { code: "EACCES" });
      const open = vi.spyOn(fs, "openSync").mockImplementation(() => {
        throw error;
      });
      try {
        expect(() => [...read(file)]).toThrow(error);
      } finally {
        open.mockRestore();
      }
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });
});
