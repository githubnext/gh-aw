// @ts-check
import { describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { transformJSONStrings } = require("./json_string_scanner.cjs");

describe("JSON string scanner", () => {
  it("passes complete quoted tokens and plain spans to separate callbacks", () => {
    const strings = vi.fn(token => `[${token}]`);
    const plain = vi.fn(text => `{${text}}`);
    expect(transformJSONStrings('head "one" between "two" tail', strings, plain)).toBe('{head }["one"]{ between }["two"]{ tail}');
    expect(strings.mock.calls.map(([token]) => token)).toEqual(['"one"', '"two"']);
    expect(plain.mock.calls.map(([text]) => text)).toEqual(["head ", " between ", " tail"]);
  });

  it("visits adjacent and empty tokens without separators or trailing text", () => {
    const strings = vi.fn(token => token);
    const content = '"""one""two"""';
    expect(transformJSONStrings(content, strings)).toBe(content);
    expect(strings.mock.calls.map(([token]) => token)).toEqual(['""', '"one"', '"two"', '""']);
  });

  it.each(['"unterminated', '"ends\\', '"ends\\u', '"ends\\u123', 'plain "unterminated', '"safe" "ends\\'])("preserves truncated input and never transforms incomplete strings (%j)", content => {
    const strings = vi.fn(token => token);
    const plain = vi.fn(text => text);
    expect(transformJSONStrings(content, strings, plain)).toBe(content);
    expect(strings.mock.calls.map(([token]) => token)).toEqual(content.startsWith('"safe"') ? ['"safe"'] : []);
    expect(plain.mock.calls.map(([text]) => text).join("")).toBe(content.startsWith('"safe"') ? content.slice(6) : content);
  });

  it.each(['"bad\t ', '"bad\\q ', '"bad\\u12gg '])("recovers valid tokens after invalid unterminated prefixes without losing bytes (%j)", prefix => {
    const strings = vi.fn(token => token);
    const plain = vi.fn(text => text);
    expect(transformJSONStrings(prefix + '"\\u006fpaque" tail', strings, plain)).toBe(prefix + '"\\u006fpaque" tail');
    expect(strings.mock.calls.map(([token]) => token)).toEqual(['"\\u006fpaque"']);
    expect(plain.mock.calls.map(([text]) => text).join("")).toBe(prefix + " tail");
  });

  it("preserves bytes even when malformed terminated input makes quote pairing ambiguous", () => {
    const strings = vi.fn(token => token);
    const content = '"bad\t" "\\u006fpaque"';
    expect(transformJSONStrings(content, strings)).toBe(content);
    expect(strings.mock.calls.map(([token]) => token)).toEqual(['" "']);
  });

  it.each([1024, 2048])("bounds character inspections after %i malformed escaped quotes", count => {
    const prefix = '"' + '\\"'.repeat(count) + "\\q ";
    const content = prefix + '"\\u006fpaque" tail';
    const strings = vi.fn(token => token);
    const plain = vi.fn(text => text);
    const original = String.prototype.charCodeAt;
    let inspections = 0;
    const spy = vi.spyOn(String.prototype, "charCodeAt").mockImplementation(function (index) {
      inspections++;
      return original.call(this, index);
    });
    let result;
    try {
      result = transformJSONStrings(content, strings, plain);
    } finally {
      spy.mockRestore();
    }
    expect(inspections).toBeLessThanOrEqual(content.length);
    expect(result).toBe(content);
    expect(strings.mock.calls.map(([token]) => token)).toEqual(['"\\u006fpaque"']);
    expect(plain.mock.calls.map(([text]) => text).join("")).toBe(prefix + " tail");
  });

  it("continues scanning when a callback removes tokens and adjacent spans", () => {
    const content = 'before "" "remove" after "keep"';
    expect(transformJSONStrings(content, token => (token === '"remove"' ? "" : token))).toBe('before ""  after "keep"');
  });

  it.each(["string", "plain"])("propagates %s callback failures without adding source text", callback => {
    const error = new Error("synthetic callback failure");
    const fail = () => {
      throw error;
    };
    expect(() => transformJSONStrings('plain "value"', callback === "string" ? fail : token => token, callback === "plain" ? fail : text => text)).toThrow(error);
  });

  it.each(["", "ordinary", 'quote"slash\\', "\u0000\b\f\n\r\t\u001f", "\ud800\udfff", "\\u006fpaque", '\\"'.repeat(32)])("only transforms a complete token at every possible truncation boundary (%j)", value => {
    const encoded = JSON.stringify(value);
    for (let length = 0; length <= encoded.length; length++) {
      const strings = vi.fn(token => `[${token}]`);
      const content = `prefix ${encoded.slice(0, length)}`;
      expect(transformJSONStrings(content, strings)).toBe(length === encoded.length ? `prefix [${encoded}]` : content);
      expect(strings.mock.calls.map(([token]) => token)).toEqual(length === encoded.length ? [encoded] : []);
    }
  });
});
