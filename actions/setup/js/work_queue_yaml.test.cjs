import { describe, expect, it } from "vitest";
import { parseDocument } from "./work_queue_yaml.cjs";

const parse = (text, options = { version: "1.2", uniqueKeys: true }) => parseDocument(text, options);

describe("work_queue_yaml parseDocument", () => {
  it("parses scalars, maps and sequences to JS", () => {
    const doc = parse("name: test\ncount: 3\nenabled: true\nitems:\n  - a\n  - b\n");
    expect(doc.errors).toHaveLength(0);
    expect(doc.toJS()).toEqual({ name: "test", count: 3, enabled: true, items: ["a", "b"] });
  });

  it("parses flow collections", () => {
    expect(parse("a: [1, 2, 3]\nb: {x: 1, y: two}\n").toJS()).toEqual({ a: [1, 2, 3], b: { x: 1, y: "two" } });
  });

  it("parses null values and empty values", () => {
    expect(parse("a: null\nb: ~\nc:\n").toJS()).toEqual({ a: null, b: null, c: null });
  });

  it("parses block scalars", () => {
    const doc = parse("run: |\n  echo one\n  echo two\nfolded: >\n  hello\n  world\n");
    expect(doc.toJS()).toEqual({ run: "echo one\necho two\n", folded: "hello world\n" });
  });

  it("uses YAML 1.2 so 'on' and 'yes' stay strings", () => {
    expect(parse("on: push\nflag: yes\n").toJS()).toEqual({ on: "push", flag: "yes" });
  });

  it("reports duplicate keys when uniqueKeys is enabled", () => {
    const doc = parse("a: 1\na: 2\n");
    expect(doc.errors.length).toBeGreaterThan(0);
  });

  it("reports syntax errors", () => {
    expect(parse("a: [1, 2\n").errors.length).toBeGreaterThan(0);
  });

  it("resolves anchors and aliases", () => {
    expect(parse("base: &b {x: 1}\ncopy: *b\n").toJS()).toEqual({ base: { x: 1 }, copy: { x: 1 } });
  });

  it("preserves quoted strings", () => {
    expect(parse("a: '123'\nb: \"true\"\n").toJS()).toEqual({ a: "123", b: "true" });
  });

  it("round-trips a document via toString", () => {
    const doc = parse("# comment\na: 1\nb:\n  - x\n");
    expect(doc.toString()).toContain("# comment");
    expect(parse(doc.toString()).toJS()).toEqual({ a: 1, b: ["x"] });
  });
});
