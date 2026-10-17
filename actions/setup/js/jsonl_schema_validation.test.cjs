import { describe, it, expect, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { iterateJsonl, iterateJsonlEvents, readJsonlFile, readJsonlFileEvents, streamJsonl, streamJsonlEvents, JsonlParser } from "./jsonl_helpers.cjs";

async function collect(source) {
  const values = [];
  for await (const value of source) values.push(value);
  return values;
}

describe("JSONL schema validation", () => {
  const schema = { type: "object", required: ["id"], properties: { id: { type: "integer" }, text: { type: "string" } }, additionalProperties: false };

  it("emits schema failures with physical positions and original raw records", () => {
    const text = '{"id":1}\r\n{"id":"private-value"}\n{broken}\n\n{"id":2}';
    const onError = vi.fn();
    const events = [...iterateJsonlEvents(text, { schema, onError })];
    expect(events.map(event => event.type)).toEqual(["json", "invalid", "invalid", "invalid", "json"]);
    expect(events[1]).toMatchObject({ reason: "schema", lineNumber: 2, validationError: { path: "id", message: "must be a integer" } });
    expect(events.map(event => event.rawLine).join("")).toBe(text);
    expect(events[1].error.message).not.toContain("private-value");
    expect(events[1].validationError).not.toHaveProperty("received");
    expect(onError).toHaveBeenCalledTimes(2);
    expect([...iterateJsonl(text, { schema })]).toEqual([{ id: 1 }, { id: 2 }]);
    expect(() => [...iterateJsonl(text, { schema, strict: true })]).toThrow("does not match schema");
  });

  it.each([
    [{ type: ["string", "null"] }, '"text"\nnull\n0', ["text", null]],
    [{ type: "object", required: ["name"] }, '{}\n{"name":null}', [{ name: null }]],
    [{ type: "array", items: { enum: ["ok"] } }, '["ok"]\n["bad"]', [["ok"]]],
    [{ oneOf: [{ type: "number" }, { type: "integer" }] }, "1\n1.5", [1.5]],
    [{ anyOf: [{ type: "string" }, { type: "boolean" }] }, '"ok"\nfalse\n0', ["ok", false]],
    [{ type: "object", additionalProperties: false }, '{}\n{"__proto__":true}', [{}]],
  ])("supports the memory schema subset (%j)", (recordSchema, text, expected) => {
    expect([...iterateJsonl(text, { schema: recordSchema })]).toEqual(expected);
  });

  it("reports nested object and array paths", () => {
    const nested = { type: "object", properties: { items: { type: "array", items: { type: "object", required: ["name"], properties: { name: { type: "string" } } } } } };
    expect([...iterateJsonlEvents('{"items":[{}]}', { schema: nested })][0]).toMatchObject({ reason: "schema", validationError: { path: "items[0].name", message: "is required" } });
  });

  it("checks raw numeric enum tokens without constraining unrelated numbers", () => {
    const numeric = { type: "object", properties: { version: { enum: [1] }, measurement: { type: "number" } } };
    expect([...iterateJsonl('{"version":1e0,"measurement":1.0000000000000001}', { schema: numeric })]).toHaveLength(1);
    for (const text of ['{"version":1.0000000000000001}', String.raw`{"\u0076ersion":1.0000000000000001}`, `{"version":${"1" + "0".repeat(257)}e-257}`]) {
      expect([...iterateJsonlEvents(text, { schema: numeric })][0]).toMatchObject({ reason: "schema", validationError: { message: expect.stringContaining("cannot be represented exactly") } });
    }
    const alternatives = { anyOf: [{ type: "array", items: { enum: [1] } }, { type: "null" }] };
    expect([...iterateJsonlEvents("[1.0000000000000001]", { schema: alternatives })][0]).toMatchObject({ reason: "schema" });
  });

  it.each([{ format: "date" }, { type: "bogus" }, { enum: [9007199254740992] }, { oneOf: [{ type: "string" }], type: "string" }])("rejects unsupported schemas even for an empty input (%j)", invalid => {
    expect(() => [...iterateJsonl("", { schema: invalid })]).toThrow(TypeError);
    expect(() => new JsonlParser({ schema: invalid })).toThrow(TypeError);
    const open = vi.spyOn(fs, "openSync");
    try {
      expect(() => [...readJsonlFile("nonexistent-schema-input", { schema: invalid, fatalUTF8: true })]).toThrow(TypeError);
      expect(open).not.toHaveBeenCalled();
    } finally {
      open.mockRestore();
    }
  });

  it("validates transformed records after filtering and propagates callback errors", () => {
    const options = { schema, fieldFilter: { field: "id", value: 1 }, lineTransform: line => line.replace(/^log /, "") };
    expect([...iterateJsonl('log {"id":2,"extra":true}\nlog {"id":1}', options)]).toEqual([{ id: 1 }]);
    expect(() => [
      ...iterateJsonl('{"id":"bad"}', {
        schema,
        onError: () => {
          throw new Error("consumer failed");
        },
      }),
    ]).toThrow("consumer failed");
  });

  it("supports multi-megabyte records and lazy persistent and async validation", async () => {
    const text = `{"id":1,"text":"${"x".repeat(2 * 1024 * 1024)}"}\n{"id":"bad"}\n{"id":2}`;
    const parser = new JsonlParser({ schema });
    const events = [];
    for (let offset = 0; offset < text.length; offset += 4096) events.push(...parser.push(Buffer.from(text.slice(offset, offset + 4096))));
    events.push(...parser.finish());
    expect(events.map(event => event.type)).toEqual(["json", "invalid", "json"]);
    const asyncEvents = await collect(streamJsonlEvents([text], { schema }));
    expect(asyncEvents.map(event => event.type)).toEqual(["json", "invalid", "json"]);
    expect(await collect(streamJsonl(['{"id":1}\n{"id":"bad"}'], { schema }))).toEqual([{ id: 1 }]);
    await expect(collect(streamJsonl(['{"id":"bad"}'], { schema, strict: true }))).rejects.toThrow("does not match schema");
  });

  it("does not rescan raw tokens when a schema has no numeric enums", () => {
    const parse = vi.spyOn(JSON, "parse");
    try {
      expect([...iterateJsonl('{"id":1,"text":"ok"}', { schema })]).toHaveLength(1);
      expect(parse).toHaveBeenCalledTimes(1);
    } finally {
      parse.mockRestore();
    }
  });

  it("handles deeply nested unrelated data without recursive token scans or error serialization", () => {
    const deep = "[".repeat(40000) + "0" + "]".repeat(40000);
    const numeric = { type: "object", properties: { version: { enum: [1] } } };
    expect([...iterateJsonl(`{"version":1,"unrelated":${deep}}`, { schema: numeric })]).toHaveLength(1);
    expect([...iterateJsonlEvents(deep, { schema: { type: "string" } })][0]).toMatchObject({ reason: "schema", validationError: { path: "", message: "must be a string" } });
  });

  it("still checks numeric enum tokens at the maximum schema depth", () => {
    let numeric = { enum: [1] };
    let text = "1.0000000000000001";
    for (let index = 0; index < 32; index++) {
      numeric = { properties: { nested: numeric } };
      text = `{"nested":${text}}`;
    }
    expect([...iterateJsonlEvents(text, { schema: numeric })][0]).toMatchObject({ reason: "schema", validationError: { message: expect.stringContaining("cannot be represented exactly") } });
  });

  it("streams validated files and closes them when validation fails", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-jsonl-schema-"));
    const file = path.join(directory, "records.jsonl");
    fs.writeFileSync(file, '{"id":1}\n{"id":"bad"}\n'.repeat(10000));
    const readFile = vi.spyOn(fs, "readFileSync");
    const read = vi.spyOn(fs, "readSync");
    const close = vi.spyOn(fs, "closeSync");
    try {
      const iterator = readJsonlFile(file, { schema, strict: true, fatalUTF8: true });
      expect(read).not.toHaveBeenCalled();
      expect(iterator.next().value).toEqual({ id: 1 });
      expect(() => iterator.next()).toThrow("does not match schema");
      expect(readFile).not.toHaveBeenCalled();
      expect(read).toHaveBeenCalledTimes(1);
      expect(read.mock.calls[0][3]).toBe(64 * 1024);
      expect(close).toHaveBeenCalledTimes(1);
    } finally {
      vi.restoreAllMocks();
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });

  it.each(["\ufeff", "\u00a0", "\v", "\f"])("preserves strict JSON boundary whitespace when requested (%j)", whitespace => {
    expect([...iterateJsonl(`${whitespace}{"id":1}${whitespace}`, { schema })]).toHaveLength(1);
    expect([...iterateJsonlEvents(`${whitespace}{"id":1}${whitespace}`, { schema, jsonWhitespaceOnly: true })][0]).toMatchObject({ reason: "syntax" });
  });
});

describe("JSONL fatal UTF-8 validation", () => {
  it("preserves already-decoded strings and flushes incomplete byte prefixes before them", async () => {
    const chunks = [Buffer.from('{"text":"'), Buffer.from([0xc3]), '\u00e9"}\n{"text":"\ud800"}'];
    expect(await collect(streamJsonlEvents(chunks, { fatalUTF8: true }))).toMatchObject([
      { type: "invalid", reason: "encoding", lineNumber: 1 },
      { type: "json", value: { text: "\ud800" }, rawLine: '{"text":"\ud800"}', lineNumber: 2 },
    ]);
  });

  it("recovers after invalid bytes with exact physical record numbers", async () => {
    const bytes = Buffer.concat([Buffer.from('{"text":"\u00e9\ufffd"}\r\n{"text":"'), Buffer.from([0xff]), Buffer.from('"}\n{"text":"ok"}')]);
    const chunks = Array.from(bytes, byte => Buffer.from([byte]));
    const events = await collect(streamJsonlEvents(chunks, { fatalUTF8: true }));
    expect(events).toMatchObject([
      { type: "json", value: { text: "\u00e9\ufffd" }, lineNumber: 1 },
      { type: "invalid", reason: "encoding", lineNumber: 2 },
      { type: "json", value: { text: "ok" }, lineNumber: 3 },
    ]);
    expect(events[1].error.message).toBe("JSONL record contains invalid UTF-8");
    await expect(collect(streamJsonl(chunks, { fatalUTF8: true, strict: true }))).rejects.toThrow("invalid UTF-8");
  });

  it("rejects incomplete UTF-8 at EOF and honors pending-record bounds", async () => {
    const bytes = Buffer.concat([Buffer.from('{"text":"'), Buffer.from([0xc3])]);
    expect(await collect(streamJsonlEvents([bytes], { fatalUTF8: true }))).toMatchObject([{ reason: "encoding", lineNumber: 1 }]);
    await expect(collect(streamJsonlEvents([bytes], { fatalUTF8: true, maxPendingRecordLength: 4 }))).rejects.toThrow("maxPendingRecordLength");
  });

  it("closes upstream immediately when stopping before later invalid bytes", async () => {
    let closed = false;
    async function* chunks() {
      try {
        yield Buffer.concat([Buffer.from('{"id":1}\n'), Buffer.from([0xff])]);
        throw new Error("must not request another chunk");
      } finally {
        closed = true;
      }
    }
    for await (const event of streamJsonlEvents(chunks(), { fatalUTF8: true })) {
      expect(event).toMatchObject({ type: "json", value: { id: 1 } });
      break;
    }
    expect(closed).toBe(true);
  });

  it("rejects invalid UTF-8 in files without confusing a literal replacement character", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-jsonl-utf8-"));
    const file = path.join(directory, "records.jsonl");
    fs.writeFileSync(file, Buffer.concat([Buffer.from('"\ufffd"\n"'), Buffer.from([0xff]), Buffer.from('"\n')]));
    try {
      expect([...readJsonlFileEvents(file, { fatalUTF8: true })]).toMatchObject([
        { type: "json", value: "\ufffd" },
        { reason: "encoding", lineNumber: 2 },
      ]);
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });
});
