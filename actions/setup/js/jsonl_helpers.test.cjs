import { describe, it, expect, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { Readable } from "node:stream";
import { execFileSync } from "node:child_process";
import { parseJsonlContent, parseJsonlLine, iterateLines, readLines, streamLines, iterateJsonl, readJsonlFile, streamJsonl } from "./jsonl_helpers.cjs";
import { ReadBuffer } from "./read_buffer.cjs";

async function collect(source) {
  const entries = [];
  for await (const entry of source) entries.push(entry);
  return entries;
}

describe("jsonl_helpers", () => {
  describe("parseJsonlContent", () => {
    it("returns parsed JSON entries and skips malformed lines", () => {
      const parsed = parseJsonlContent(['{"event":"token_steering"}', "not-json", "", "   ", '{"event":"request"}'].join("\n"));

      expect(parsed).toEqual([{ event: "token_steering" }, { event: "request" }]);
    });

    it("returns empty array for non-string or empty content", () => {
      expect(parseJsonlContent("")).toEqual([]);
      expect(parseJsonlContent(/** @type {any} */ null)).toEqual([]);
      expect(parseJsonlContent(/** @type {any} */ undefined)).toEqual([]);
    });

    it("supports optional line pre-filtering before JSON parsing", () => {
      const parsed = parseJsonlContent(['{"event":"token_steering"}', '{"event":"request"}', '{"event":"model_steering"}'].join("\n"), line => line.includes("steering"));

      expect(parsed).toEqual([{ event: "token_steering" }, { event: "model_steering" }]);
    });

    it("preserves every JSON value, whitespace, CRLF, BOM and unterminated records", async () => {
      const text = '\ufeff {"text":"你好 🚀"}\r\n[]\nnull\nfalse\n0\n-1.5e2\ntrue\n"hello"';
      const expected = [{ text: "你好 🚀" }, [], null, false, 0, -150, true, "hello"];
      expect([...iterateJsonl(text)]).toEqual(expected);
      // Single-byte chunks exercise split BOM and multi-byte code points.
      const chunks = Array.from(Buffer.from(text), byte => Buffer.from([byte]));
      expect(await collect(streamJsonl(chunks))).toEqual(expected);
    });

    it("does not invoke JSON.parse for obviously non-JSON lines", () => {
      const parse = vi.spyOn(JSON, "parse");
      try {
        expect([...iterateJsonl('DEBUG something\n[INFO] noise\n{not json}\nundefined\nnot json\nfalse alarm\n\n{"ok":true}')]).toEqual([{ ok: true }]);
        expect(parse).toHaveBeenCalledTimes(1);
      } finally {
        parse.mockRestore();
      }
    });

    it("reports corrupted lines with physical line numbers without input contents", async () => {
      const errors = [];
      const options = { onError: (error, line) => errors.push([error.message, line]) };
      const text = '\nnoise\n{"secret":"do-not-log",}\n{"ok":1}\n{"unfinished"';
      expect(await collect(streamJsonl([Buffer.from(text)], options))).toEqual([{ ok: 1 }]);
      expect(errors.map(entry => entry[1])).toEqual([2, 3, 5]);
      expect(JSON.stringify(errors)).not.toContain("do-not-log");
      expect(() => parseJsonlLine("noise")).toThrow(SyntaxError);
      expect(() => parseJsonlLine('{"broken"')).toThrow(SyntaxError);
    });

    it("supports filters and optional line limits without capping normal records", async () => {
      const errors = [];
      const options = {
        maxLineLength: 10,
        lineFilter: line => !line.includes("skip"),
        onError: (error, line) => errors.push([error.name, line]),
      };
      const text = '{"skip":true}\n{"long":1234}\n{"ok":1}';
      expect([...iterateJsonl(text, options)]).toEqual([{ ok: 1 }]);
      expect(errors).toEqual([["RangeError", 2]]);
      expect(await collect(streamJsonl([text], options))).toEqual([{ ok: 1 }]);
      expect(() => [...iterateJsonl("", { maxLineLength: 0 })]).toThrow(RangeError);
      await expect(collect(streamJsonl([], { maxLineLength: NaN }))).rejects.toThrow(RangeError);
    });

    it("propagates source, predicate and error callback failures", async () => {
      const failure = new Error("source failed");
      async function* source() {
        yield '{"ok":1}\n';
        throw failure;
      }
      const iterator = streamJsonl(source());
      expect((await iterator.next()).value).toEqual({ ok: 1 });
      await expect(iterator.next()).rejects.toBe(failure);
      expect(() => [
        ...iterateJsonl("{}", {
          lineFilter: () => {
            throw failure;
          },
        }),
      ]).toThrow(failure);
      expect(() => [
        ...iterateJsonl("invalid", {
          onError: () => {
            throw failure;
          },
        }),
      ]).toThrow(failure);
    });

    it("is lazy within a chunk and closes the source on early termination", async () => {
      let pulled = 0;
      let closed = false;
      async function* source() {
        try {
          pulled++;
          yield '{"id":1}\n{"id":2}\n';
          pulled++;
          yield '{"id":3}\n';
        } finally {
          closed = true;
        }
      }
      const parse = vi.spyOn(JSON, "parse");
      try {
        const iterator = streamJsonl(source());
        expect(pulled).toBe(0);
        expect((await iterator.next()).value).toEqual({ id: 1 });
        expect(pulled).toBe(1);
        expect(parse).toHaveBeenCalledTimes(1);
        await iterator.return();
        expect(closed).toBe(true);
        expect(pulled).toBe(1);
        expect(parse).toHaveBeenCalledTimes(1);
      } finally {
        parse.mockRestore();
      }
    });

    it("handles multi-megabyte valid and corrupt lines across chunk boundaries", async () => {
      const value = { text: "🚀".repeat(2 * 1024 * 1024) };
      const bytes = Buffer.from(`${JSON.stringify(value)}\r\n{"bad":${" ".repeat(3 * 1024 * 1024)}}x\n{"last":true}`);
      function* chunks() {
        for (let offset = 0; offset < bytes.length; offset += 65521) yield bytes.subarray(offset, offset + 65521);
      }
      expect(await collect(streamJsonl(chunks()))).toEqual([value, { last: true }]);
    });

    it("streams files lazily and closes file descriptors on break and callback errors", () => {
      const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-jsonl-"));
      const file = path.join(directory, "input.jsonl");
      const open = vi.spyOn(fs, "openSync");
      const close = vi.spyOn(fs, "closeSync");
      try {
        const iterator = readJsonlFile(file);
        expect(open).not.toHaveBeenCalled();
        const value = { text: "你好".repeat(1024 * 1024) };
        fs.writeFileSync(file, `${JSON.stringify(value)}\r\nnoise\n{"last":true}`);
        expect([...iterator]).toEqual([value, { last: true }]);
        expect(close).toHaveBeenCalledTimes(1);
        for (const entry of readJsonlFile(file)) {
          expect(entry).toEqual(value);
          break;
        }
        expect(close).toHaveBeenCalledTimes(2);
        expect(() => [
          ...readJsonlFile(file, {
            lineFilter: () => {
              throw new Error("filter failed");
            },
          }),
        ]).toThrow("filter failed");
        expect(close).toHaveBeenCalledTimes(3);
        expect(() => [...readJsonlFile(path.join(directory, "missing"))]).toThrow();
      } finally {
        open.mockRestore();
        close.mockRestore();
        fs.rmSync(directory, { recursive: true, force: true });
      }
    });

    it("bounds memory for a massive input under a 32 MB heap", () => {
      const helper = path.join(__dirname, "jsonl_helpers.cjs");
      const script = `
        const { streamJsonl, readJsonlFile } = require(${JSON.stringify(helper)});
        const fs = require("node:fs");
        const os = require("node:os");
        const path = require("node:path");
        const row = '{"id":1,"text":"' + "x".repeat(4096) + '"}\\n';
        async function* chunks() {
          for (let i = 0; i < 65536; i++) yield row;
        }
        (async () => {
          let count = 0;
          for await (const value of streamJsonl(chunks())) {
            if (value.id !== 1) throw new Error("bad record");
            count++;
          }
          const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-large-jsonl-"));
          const file = path.join(directory, "large.jsonl");
          try {
            const fd = fs.openSync(file, "w");
            try {
              const batch = row.repeat(16);
              for (let i = 0; i < 4096; i++) fs.writeSync(fd, batch);
            } finally {
              fs.closeSync(fd);
            }
            let fileCount = 0;
            for (const value of readJsonlFile(file)) {
              if (value.id !== 1) throw new Error("bad file record");
              fileCount++;
            }
            process.stdout.write(count + ":" + fileCount);
          } finally {
            fs.rmSync(directory, { recursive: true, force: true });
          }
        })().catch(error => { console.error(error); process.exitCode = 1; });
      `;
      expect(execFileSync(process.execPath, ["--max-old-space-size=32", "-e", script], { encoding: "utf8", timeout: 30000 })).toBe("65536:65536");
    }, 35000);

    it("frames raw lines consistently and destroys readable streams on break", async () => {
      expect([...iterateLines("a\r\n\nb")]).toEqual(["a", "", "b"]);
      const readable = Readable.from([Buffer.from('{"first":1}\n{"second":2}\n')]);
      for await (const value of streamJsonl(readable)) {
        expect(value).toEqual({ first: 1 });
        break;
      }
      expect(readable.destroyed).toBe(true);
      expect(await collect(streamLines(["a\r", "\n", Buffer.from("b")]))).toEqual(["a", "b"]);
      expect([...readLines(__filename)].length).toBeGreaterThan(1);
    });

    it("handles long JSON-RPC records and blank-line runs without recursion", () => {
      const reader = new ReadBuffer();
      reader.append(Buffer.from("\n".repeat(100000)));
      expect(reader.readMessage()).toBeNull();
      const value = { text: "🚀".repeat(1024 * 1024) };
      const bytes = Buffer.from(JSON.stringify(value) + "\nnoise\n{}\n");
      for (let offset = 0; offset < bytes.length; offset += 65521) reader.append(bytes.subarray(offset, offset + 65521));
      expect(reader.readMessage()).toEqual(value);
      expect(() => reader.readMessage()).toThrow(/Parse error/);
      expect(reader.readMessage()).toEqual({});
      expect(reader.readMessage()).toBeNull();
    });

    it("releases consumed JSON-RPC queue slots even when input never fully drains", () => {
      const reader = new ReadBuffer();
      reader.append(Buffer.from("{}\n"));
      for (let i = 0; i < 5000; i++) {
        reader.append(Buffer.from("{}\n"));
        expect(reader.readMessage()).toEqual({});
      }
      expect(reader._lines.length).toBeLessThan(1025);
      expect(reader.readMessage()).toEqual({});
      expect(reader.readMessage()).toBeNull();
    });
  });
});
